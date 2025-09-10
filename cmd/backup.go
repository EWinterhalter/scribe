package cmd

import (
	"fmt"
	"log"
	"time"

	"scrible/internal/cli"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create backup and upload to Yandex Cloud",
	Long:  `Create a ZIP archive of specified directory and upload it to Yandex Cloud Object Storage`,
	Run: func(cmd *cobra.Command, args []string) {
		sourceDir, _ := cmd.Flags().GetString("source")
		bucketName, _ := cmd.Flags().GetString("bucket")
		objectPrefix, _ := cmd.Flags().GetString("prefix")

		timestamp := time.Now().Format("2006-01-02-15-04-05")
		archiveName := fmt.Sprintf("backup-%s.zip", timestamp)

		backupHandler := cli.NewBackupHandler()

		err := backupHandler.RunBackup(sourceDir, archiveName, bucketName, objectPrefix)
		if err != nil {
			log.Fatalf("Backup failed: %v", err)
		}

		fmt.Println("✅ Backup completed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringP("source", "s", "", "Source directory to backup (required)")
	backupCmd.Flags().StringP("bucket", "b", "", "Yandex Cloud bucket name (required)")
	backupCmd.Flags().StringP("prefix", "p", "backups/", "Object key prefix in bucket")

	backupCmd.MarkFlagRequired("source")
	backupCmd.MarkFlagRequired("bucket")
}
