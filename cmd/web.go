package cmd

import (
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Web-related commands",
	Long:  `This command is for web-related operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This is left empty intentionally
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}
