package uploader

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithyhttp "github.com/aws/smithy-go/transport/http"
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

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ru-central1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://storage.yandexcloud.net")
	})

	return &S3Uploader{client: client}, nil
}

func (u *S3Uploader) BucketExists(bucketName string) (bool, error) {
	if bucketName == "" {
		return false, fmt.Errorf("bucket name cannot be empty")
	}

	_, err := u.client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return true, nil
}
