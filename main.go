package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tnypxl/rollup/cmd"
	"github.com/tnypxl/rollup/internal/config"
	"github.com/tnypxl/rollup/internal/scraper"
)

var cfg *config.Config

func main() {
	// Check if the command is "help"
	isHelpCommand := len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h")

	var cfg *config.Config
	var err error

	if !isHelpCommand {
		configPath := config.DefaultConfigPath()
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Printf("Warning: Failed to load configuration: %v", err)
			// Continue execution without a config file
		}

		// Initialize the scraper logger with default verbosity (false)
		scraper.SetupLogger(false)

		err = scraper.InitPlaywright()
		if err != nil {
			log.Fatalf("Failed to initialize Playwright: %v", err)
		}
		defer scraper.ClosePlaywright()
	}

	if err := cmd.Execute(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
