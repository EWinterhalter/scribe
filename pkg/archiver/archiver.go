package archiver

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CreateArchive(sourcePath, outputPath string) error {
	if sourcePath == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if outputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source directory info: %w", err)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", sourcePath)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := outputFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close output file: %w", closeErr)
		}
	}()

	zipWriter := zip.NewWriter(outputFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close zip writer: %w", closeErr)
		}
	}()

	basePath := filepath.Dir(sourcePath)

	err = filepath.Walk(sourcePath, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error walking to %s: %w", filePath, walkErr)
		}

		relPath, err := filepath.Rel(basePath, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header for %s: %w", filePath, err)
		}

		header.Name = relPath
		header.Method = zip.Deflate

		if info.IsDir() {
			header.Name += "/"
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create entry in archive for %s: %w", header.Name, err)
		}

		if !info.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open source file %s: %w", filePath, err)
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return fmt.Errorf("failed to copy file content for %s: %w", filePath, err)
			}
		}

		return nil
	})

	if err != nil {
		if removeErr := os.Remove(outputPath); removeErr != nil {
			return fmt.Errorf("archive creation failed: %w, and failed to remove partial archive: %v", err, removeErr)
		}
		return fmt.Errorf("failed to create archive: %w", err)
	}

	return nil
}
