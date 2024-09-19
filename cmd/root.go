package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	config "github.com/tnypxl/rollup/internal/config"
)

var (
	configFile string
	cfg        *config.Config
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "rollup",
	Short: "Rollup is a tool for combining and processing files",
	Long: `Rollup is a versatile tool that can combine and process files in various ways.
Use subcommands to perform specific operations.`,
}

func Execute(conf *config.Config) error {
	cfg = conf
	if cfg == nil {
		cfg = &config.Config{} // Use an empty config if none is provided
	}
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "f", "", "Path to the config file (default: rollup.yml in the current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.AddCommand(filesCmd)
}
