package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/config"
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for generate and help commands
		if cmd.Name() == "generate" || cmd.Name() == "help" {
			return nil
		}

		// Determine config path
		configPath := configFile
		if configPath == "" {
			configPath = "rollup.yml"
		}

		// Load configuration
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Printf("Warning: Failed to load configuration from %s: %v", configPath, err)
			cfg = &config.Config{} // Use empty config if loading fails
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "f", "", "Path to the config file (default: rollup.yml in the current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.AddCommand(filesCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(generateCmd)
}
