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
	configPath := config.DefaultConfigPath()
	var err error
	cfg, err = config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize the scraper logger with default verbosity (false)
	scraper.SetupLogger(false)

	err = scraper.InitPlaywright()
	if err != nil {
		log.Fatalf("Failed to initialize Playwright: %v", err)
	}
	defer scraper.ClosePlaywright()

	scraperConfig := scraper.Config{
		URLs:       []scraper.URLConfig{{URL: cfg.Scrape.URLs[0].URL}},
		OutputType: cfg.Scrape.OutputType,
		Verbose:    false, // Set default verbosity
	}

	if err := cmd.Execute(cfg, scraperConfig); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
