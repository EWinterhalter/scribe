package uploader

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Uploader struct {
	client *s3.Client
}

func NewS3Uploader() (*S3Uploader, error) {
	accessKeyID := os.Getenv("YC_STORAGE_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("YC_STORAGE_SECRET_ACCESS_KEY")

	if accessKeyID == "" {
		return nil, fmt.Errorf("YC_STORAGE_ACCESS_KEY_ID environment variable is not set")
	}
	if secretAccessKey == "" {
		return nil, fmt.Errorf("YC_STORAGE_SECRET_ACCESS_KEY environment variable is not set")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:           "https://storage.yandexcloud.net",
				SigningRegion: "ru-central1",
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Uploader{client: client}, nil
}

func (u *S3Uploader) UploadFile(localFilePath, bucketName, objectKey string) error {
	if localFilePath == "" {
		return fmt.Errorf("local file path cannot be empty")
	}
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if objectKey == "" {
		return fmt.Errorf("object key cannot be empty")
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localFilePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", localFilePath, err)
	}

	fmt.Printf("Uploading file: %s (size: %d bytes)\n", localFilePath, fileInfo.Size())
	fmt.Printf("Target: s3://%s/%s\n", bucketName, objectKey)

	_, err = u.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to Yandex Cloud: %w", err)
	}

	fmt.Printf("Successfully uploaded to: s3://%s/%s\n", bucketName, objectKey)
	return nil
}

func (u *S3Uploader) BucketExists(bucketName string) (bool, error) {
	if bucketName == "" {
		return false, fmt.Errorf("bucket name cannot be empty")
	}

	_, err := u.client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		var notFound *types.NotFound
		if errorIs(err, notFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return true, nil
}

func errorIs(err, target error) bool {
	return err != nil && target != nil && err.Error() == target.Error()
}
