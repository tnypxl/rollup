package cmd

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/config"
	"github.com/tnypxl/rollup/internal/scraper"
)

var (
	urls             []string
	outputType       string
	includeSelector  string
	excludeSelectors []string
)

var scraperConfig scraper.Config

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Scrape main content from webpages and convert to Markdown",
	Long:  `Scrape the main content from one or more webpages, ignoring navigational elements, ads, and other UI aspects. Convert the content to a well-structured Markdown file.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Playwright for web scraping
		if err := scraper.InitPlaywright(); err != nil {
			return fmt.Errorf("failed to initialize Playwright: %w", err)
		}
		return nil
	},
	RunE: runWeb,
	PostRunE: func(cmd *cobra.Command, args []string) error {
		// Clean up Playwright resources
		scraper.ClosePlaywright()
		return nil
	},
}

func init() {
	webCmd.Flags().StringSliceVarP(&urls, "urls", "u", []string{}, "URLs of the webpages to scrape (comma-separated)")
	webCmd.Flags().StringVarP(&outputType, "output", "o", "", "Output type: 'single' for one file, 'separate' for multiple files")
	webCmd.Flags().StringVar(&includeSelector, "css", "", "CSS selector to extract specific content")
	webCmd.Flags().StringSliceVar(&excludeSelectors, "exclude", []string{}, "CSS selectors to exclude from the extracted content (comma-separated)")
}

func runWeb(cmd *cobra.Command, args []string) error {
	scraper.SetupLogger(verbose)
	logger := log.New(os.Stdout, "WEB: ", log.LstdFlags)
	if !verbose {
		logger.SetOutput(io.Discard)
	}
	logger.Printf("Starting web scraping process with verbose mode: %v", verbose)
	scraperConfig.Verbose = verbose

	var siteConfigs []scraper.SiteConfig
	if len(cfg.Sites) > 0 {
		logger.Printf("Using configuration from rollup.yml for %d sites", len(cfg.Sites))
		siteConfigs = make([]scraper.SiteConfig, len(cfg.Sites))
		for i, site := range cfg.Sites {
			siteConfigs[i] = scraper.SiteConfig{
				BaseURL:          site.BaseURL,
				CSSLocator:       site.CSSLocator,
				ExcludeSelectors: site.ExcludeSelectors,
				AllowedPaths:     site.AllowedPaths,
				ExcludePaths:     site.ExcludePaths,
				PathOverrides:    convertPathOverrides(site.PathOverrides),
			}
			logger.Printf("Site %d configuration: BaseURL=%s, CSSLocator=%s, AllowedPaths=%v",
				i+1, site.BaseURL, site.CSSLocator, site.AllowedPaths)
		}
	} else {
		logger.Printf("No sites defined in rollup.yml, falling back to URL-based configuration")
		siteConfigs = make([]scraper.SiteConfig, len(urls))
		for i, u := range urls {
			siteConfigs[i] = scraper.SiteConfig{
				BaseURL:          u,
				CSSLocator:       includeSelector,
				ExcludeSelectors: excludeSelectors,
			}
			logger.Printf("URL %d configuration: BaseURL=%s, CSSLocator=%s",
				i+1, u, includeSelector)
		}
	}

	if len(siteConfigs) == 0 {
		logger.Println("Error: No sites or URLs provided")
		return fmt.Errorf("no sites or URLs provided. Use --urls flag with comma-separated URLs or set 'scrape.sites' in the rollup.yml file")
	}

	// Set default values for rate limiting
	defaultRequestsPerSecond := 1.0
	defaultBurstLimit := 3

	// Use default values if not set in the configuration
	requestsPerSecond := defaultRequestsPerSecond
	if cfg.RequestsPerSecond != nil {
		requestsPerSecond = *cfg.RequestsPerSecond
	}
	burstLimit := defaultBurstLimit
	if cfg.BurstLimit != nil {
		burstLimit = *cfg.BurstLimit
	}

	scraperConfig := scraper.Config{
		Sites:      siteConfigs,
		OutputType: outputType,
		Verbose:    verbose,
		Scrape: scraper.ScrapeConfig{
			RequestsPerSecond: requestsPerSecond,
			BurstLimit:        burstLimit,
		},
	}
	logger.Printf("Scraper configuration: OutputType=%s, RequestsPerSecond=%f, BurstLimit=%d",
		outputType, requestsPerSecond, burstLimit)

	logger.Println("Starting scraping process")
	startTime := time.Now()
	progressTicker := time.NewTicker(time.Second)
	defer progressTicker.Stop()

	done := make(chan bool)
	messagePrinted := false
	go func() {
		for {
			select {
			case <-progressTicker.C:
				if time.Since(startTime) > 5*time.Second && !messagePrinted {
					fmt.Print("This is taking a while (hold tight) ")
					messagePrinted = true
				} else if messagePrinted {
					fmt.Print(".")
				}
			case <-done:
				return
			}
		}
	}()

	err := scraper.ScrapeSites(scraperConfig)
	done <- true
	fmt.Println() // New line after progress indicator

	if err != nil {
		logger.Printf("Error occurred during scraping: %v", err)
		return fmt.Errorf("error scraping content: %v", err)
	}
	logger.Println("Scraping completed")

	return nil
}

func getFilenameFromContent(content, urlStr string) (string, error) {
	// Try to extract title from content
	titleStart := strings.Index(content, "<title>")
	titleEnd := strings.Index(content, "</title>")
	if titleStart != -1 && titleEnd != -1 && titleEnd > titleStart {
		title := strings.TrimSpace(content[titleStart+7 : titleEnd])
		if title != "" {
			return sanitizeFilename(title) + ".rollup.md", nil
		}
	}

	// If no title found or title is empty, use the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}

	filename := parsedURL.Host
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		filename += strings.TrimSuffix(parsedURL.Path, "/")
	}
	return sanitizeFilename(filename) + ".rollup.md", nil
}

func sanitizeFilename(name string) string {
	// Remove any character that isn't alphanumeric, dash, or underscore
	reg := regexp.MustCompile("[^a-zA-Z0-9-_]+")
	name = reg.ReplaceAllString(name, "_")

	// Trim any leading or trailing underscores
	name = strings.Trim(name, "_")

	// If the name is empty after sanitization, use a default name
	if name == "" {
		name = "untitled"
	}

	return name
}

func convertPathOverrides(configOverrides []config.PathOverride) []scraper.PathOverride {
	scraperOverrides := make([]scraper.PathOverride, len(configOverrides))
	for i, override := range configOverrides {
		scraperOverrides[i] = scraper.PathOverride{
			Path:             override.Path,
			CSSLocator:       override.CSSLocator,
			ExcludeSelectors: override.ExcludeSelectors,
		}
	}
	return scraperOverrides
}
