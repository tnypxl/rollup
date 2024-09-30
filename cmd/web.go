package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/config"
	"github.com/tnypxl/rollup/internal/scraper"
)

var (
	urls             []string
	outputType       string
	depth            int
	includeSelector  string
	excludeSelectors []string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Scrape main content from webpages and convert to Markdown",
	Long:  `Scrape the main content from one or more webpages, ignoring navigational elements, ads, and other UI aspects. Convert the content to a well-structured Markdown file.`,
	RunE:  runWeb,
}

func init() {
	webCmd.Flags().StringSliceVarP(&urls, "urls", "u", []string{}, "URLs of the webpages to scrape (comma-separated)")
	webCmd.Flags().StringVarP(&outputType, "output", "o", "single", "Output type: 'single' for one file, 'separate' for multiple files")
	webCmd.Flags().IntVarP(&depth, "depth", "d", 0, "Depth of link traversal (default: 0, only scrape the given URLs)")
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

    // Prepare site configurations
    var siteConfigs []scraper.SiteConfig
    if len(cfg.Scrape.Sites) > 0 {
        // Use configurations from rollup.yml
        logger.Printf("Using configuration from rollup.yml for %d sites", len(cfg.Scrape.Sites))
        siteConfigs = make([]scraper.SiteConfig, len(cfg.Scrape.Sites))
        for i, site := range cfg.Scrape.Sites {
            siteConfigs[i] = scraper.SiteConfig{
                BaseURL:          site.BaseURL,
                CSSLocator:       site.CSSLocator,
                ExcludeSelectors: site.ExcludeSelectors,
                MaxDepth:         site.MaxDepth,
                AllowedPaths:     site.AllowedPaths,
                ExcludePaths:     site.ExcludePaths,
                OutputAlias:      site.OutputAlias,
                PathOverrides:    convertPathOverrides(site.PathOverrides),
            }
            logger.Printf("Site %d configuration: BaseURL=%s, CSSLocator=%s, MaxDepth=%d, AllowedPaths=%v",
                i+1, site.BaseURL, site.CSSLocator, site.MaxDepth, site.AllowedPaths)
        }
    } else {
        // Use command-line URLs
        if len(urls) == 0 {
            logger.Println("Error: No URLs provided via --urls flag")
            return fmt.Errorf("no URLs provided. Use --urls flag with comma-separated URLs or set 'scrape.sites' in the rollup.yml file")
        }
        siteConfigs = make([]scraper.SiteConfig, len(urls))
        for i, u := range urls {
            siteConfigs[i] = scraper.SiteConfig{
                BaseURL:          u,
                CSSLocator:       includeSelector,
                ExcludeSelectors: excludeSelectors,
                MaxDepth:         depth,
                AllowedPaths:     []string{"/"}, // Allow all paths by default
            }
            logger.Printf("URL %d configuration: BaseURL=%s, CSSLocator=%s, MaxDepth=%d",
                i+1, u, includeSelector, depth)
        }
    }

    // Set up scraper configuration
    scraperConfig := scraper.Config{
        Sites:      siteConfigs,
        OutputType: outputType,
        Verbose:    verbose,
        Scrape: scraper.ScrapeConfig{
            RequestsPerSecond: cfg.Scrape.RequestsPerSecond,
            BurstLimit:        cfg.Scrape.BurstLimit,
        },
    }
    logger.Printf("Scraper configuration: OutputType=%s, RequestsPerSecond=%f, BurstLimit=%d",
        outputType, scraperConfig.Scrape.RequestsPerSecond, scraperConfig.Scrape.BurstLimit)

    // Start scraping using scraper.ScrapeSites
    logger.Println("Starting scraping process")
    scrapedContent, err := scraper.ScrapeSites(scraperConfig)
    if err != nil {
        logger.Printf("Error occurred during scraping: %v", err)
        return fmt.Errorf("error scraping content: %v", err)
    }
    logger.Printf("Scraping completed. Total content scraped: %d", len(scrapedContent))

    // Write output to files
    if outputType == "single" {
        logger.Println("Writing content to a single file")
        return writeSingleFile(scrapedContent)
    } else {
        logger.Println("Writing content to multiple files")
        return writeMultipleFiles(scrapedContent)
    }
}

func writeSingleFile(content map[string]string) error {
	outputFile := generateDefaultFilename()
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer file.Close()

	for url, c := range content {
		_, err = fmt.Fprintf(file, "# ::: Content from %s\n\n%s\n\n---\n\n", url, c)
		if err != nil {
			return fmt.Errorf("error writing content to file: %v", err)
		}
	}

	fmt.Printf("Content has been extracted from %d URL(s) and saved to %s\n", len(content), outputFile)
	return nil
}

func writeMultipleFiles(content map[string]string) error {
	for url, c := range content {
		filename := sanitizeFilename(url) + ".rollup.md"
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("error creating output file %s: %v", filename, err)
		}

		_, err = file.WriteString(fmt.Sprintf("# ::: Content from %s\n\n%s\n", url, c))
		if err != nil {
			file.Close()
			return fmt.Errorf("error writing content to file %s: %v", filename, err)
		}

		file.Close()
		fmt.Printf("Content from %s has been saved to %s\n", url, filename)
	}

	return nil
}

func generateDefaultFilename() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("web-%s.rollup.md", timestamp)
}

func sanitizeFilename(name string) string {
	// Remove any character that isn't alphanumeric, dash, or underscore
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)

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
