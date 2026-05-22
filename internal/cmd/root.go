package cmd

import (
	"os"

	"github.com/spf13/cobra"

	upload_contract "github.com/contracttesting/cli/internal/features/upload_contract"
)

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "contracttests CLI",
}

func init() {
	upload_contract.Register(rootCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
