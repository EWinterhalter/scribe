package uploader

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	accessKeyID := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
	endpoint := os.Getenv("ENDPOINT")
	region := os.Getenv("REGION")

	if accessKeyID == "" {
		return nil, fmt.Errorf("ACCESS_KEY_ID environment variable is not set")
	}
	if secretAccessKey == "" {
		return nil, fmt.Errorf("SECRET_ACCESS_KEY environment variable is not set")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("ENDPOINT environment variable is not set")
	}
	if region == "" {
		return nil, fmt.Errorf("REGION environment variable is not set")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &S3Uploader{client: client}, nil
}

func (u *S3Uploader) UploadFile(localFilePath, bucketName, objectKey string) error {
	if localFilePath == "" || bucketName == "" || objectKey == "" {
		return fmt.Errorf("file path, bucket, and object key cannot be empty")
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fmt.Printf("Uploading file: %s (size: %d bytes) to s3://%s/%s\n", localFilePath, info.Size(), bucketName, objectKey)

	_, err = u.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	fmt.Printf("✅ Successfully uploaded to s3://%s/%s\n", bucketName, objectKey)
	return nil
}

func (u *S3Uploader) UploadFileWithProgress(localFilePath, bucketName, objectKey string, progressWriter io.Writer) error {
	if localFilePath == "" || bucketName == "" || objectKey == "" {
		return fmt.Errorf("file path, bucket, and object key cannot be empty")
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	progressReader := &ProgressReader{
		Reader:         file,
		Size:           info.Size(),
		ProgressWriter: progressWriter,
	}

	_, err = u.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   progressReader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file with progress: %w", err)
	}

	fmt.Fprintln(progressWriter, "\n✅ Upload completed")
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
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return true, nil
}

type ProgressReader struct {
	Reader         io.ReadSeeker
	Size           int64
	ProgressWriter io.Writer
	bytesRead      int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.bytesRead += int64(n)
		if pr.ProgressWriter != nil {
			progress := float64(pr.bytesRead) / float64(pr.Size) * 100
			fmt.Fprintf(pr.ProgressWriter, "\rUpload progress: %.1f%% (%d/%d bytes)",
				progress, pr.bytesRead, pr.Size)
		}
	}
	return n, err
}

func (pr *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	pos, err := pr.Reader.Seek(offset, whence)
	if err == nil && whence == io.SeekStart {
		pr.bytesRead = pos
	}
	return pos, err
}
