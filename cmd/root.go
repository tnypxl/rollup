package cmd

import (
	"github.com/spf13/cobra"
	config "github.com/tnypxl/rollup/internal/config"
)

var (
	configFile string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "rollup",
	Short: "Rollup is a tool for combining and processing files",
	Long: `Rollup is a versatile tool that can combine and process files in various ways.
Use subcommands to perform specific operations.`,
}

func Execute(conf *config.Config) error {
	if conf == nil {
		conf = &config.Config{} // Use an empty config if none is provided
	}
	cfg = conf // Set the cfg variable in cmd/files.go
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "f", "", "Path to the config file (default: rollup.yml in the current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.AddCommand(filesCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(generateCmd)
}
