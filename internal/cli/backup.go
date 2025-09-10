package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"scrible/pkg/archiver"
	"scrible/pkg/uploader"

	"github.com/schollz/progressbar/v3"
)

type BackupHandler struct {
	uploader *uploader.S3Uploader
}

func NewBackupHandler() *BackupHandler {
	uploader, err := uploader.NewS3Uploader()
	if err != nil {
		fmt.Printf("Warning: Cloud uploader not available: %v\n", err)
		fmt.Println("ℹ️  Only local backup will be available")
	}
	return &BackupHandler{uploader: uploader}
}

func (h *BackupHandler) RunBackup(sourceDir, archiveName, bucketName, objectPrefix string) error {
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	if _, err := os.Stat(absSource); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", absSource)
	}

	tempArchive := archiveName
	defer func() {
		if _, err := os.Stat(tempArchive); err == nil {
			os.Remove(tempArchive)
			fmt.Printf("🧹 Cleaned up temporary file: %s\n", tempArchive)
		}
	}()

	fmt.Printf("📦 Creating archive from: %s\n", absSource)

	err = h.createArchiveWithProgress(absSource, tempArchive)
	if err != nil {
		return fmt.Errorf("archive creation failed: %w", err)
	}

	if h.uploader != nil {
		exists, err := h.uploader.BucketExists(bucketName)
		if err != nil {
			return fmt.Errorf("failed to check bucket: %w", err)
		}
		if !exists {
			return fmt.Errorf("bucket %s does not exist or access denied", bucketName)
		}

		objectKey := filepath.Join(objectPrefix, archiveName)
		fmt.Printf("☁️  Uploading to: s3://%s/%s\n", bucketName, objectKey)

		err = h.uploadFileWithProgress(tempArchive, bucketName, objectKey)
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
	} else {
		fmt.Printf("💾 Archive saved locally: %s\n", tempArchive)
		fmt.Println("ℹ️  Cloud upload skipped (no credentials available)")
	}

	return nil
}

func (h *BackupHandler) createArchiveWithProgress(source, output string) error {
	fmt.Println("⏳ Archiving files...")

	err := archiver.CreateArchive(source, output)
	if err != nil {
		return err
	}

	if _, err := os.Stat(output); os.IsNotExist(err) {
		return fmt.Errorf("archive was not created: %s", output)
	}

	fileInfo, _ := os.Stat(output)
	fmt.Printf("✅ Archive created: %s (size: %d bytes)\n", output, fileInfo.Size())
	return nil
}

func (h *BackupHandler) uploadFileWithProgress(localFile, bucket, objectKey string) error {
	fileInfo, err := os.Stat(localFile)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions64(fileInfo.Size(),
		progressbar.OptionSetDescription("📤 Uploading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetRenderBlankState(true),
	)

	err = h.uploader.UploadFileWithProgress(localFile, bucket, objectKey, bar)
	if err != nil {
		return err
	}

	fmt.Println("✅ Upload completed successfully!")
	return nil
}
