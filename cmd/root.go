package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scribe",
	Short: "archiver to the cloud",
	Long:  `archiver to the cloud`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("use --help for usage.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
