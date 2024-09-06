package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown"
	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/config"
	"github.com/tnypxl/rollup/internal/scraper"
)

var (
	urls       []string
	outputFile string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Scrape main content from webpages and convert to Markdown",
	Long:  `Scrape the main content from one or more webpages, ignoring navigational elements, ads, and other UI aspects. Convert the content to a well-structured Markdown file.`,
	RunE:  runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.Flags().StringSliceVarP(&urls, "urls", "u", []string{}, "URLs of the webpages to scrape (comma-separated)")
	webCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output Markdown file (default: rollup-web-<timestamp>.md)")
}

func runWeb(cmd *cobra.Command, args []string) error {
	var err error
	cfg, err = config.Load("rollup.yml")
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("rollup.yml file not found. Please create a configuration file or provide command-line arguments")
		}
		return fmt.Errorf("error loading configuration: %v", err)
	}

	// Use config if available, otherwise use command-line flags
	if len(urls) == 0 && cfg.Scrape.URL != "" {
		urls = []string{cfg.Scrape.URL}
	}

	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided. Use --urls flag with comma-separated URLs or set 'scrape.url' in the rollup.yml file")
	}

	if outputFile == "" {
		outputFile = generateDefaultFilename(urls)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer file.Close()

	for i, u := range urls {
		extractedContent, err := extractAndConvertContent(u)
		if err != nil {
			return fmt.Errorf("error extracting and converting content from %s: %v", u, err)
		}

		if i > 0 {
			_, err = file.WriteString("\n\n---\n\n")
			if err != nil {
				return fmt.Errorf("error writing separator to file: %v", err)
			}
		}

		_, err = file.WriteString(extractedContent)
		if err != nil {
			return fmt.Errorf("error writing content to file: %v", err)
		}
	}

	fmt.Printf("Content has been extracted from %d URL(s) and saved to %s\n", len(urls), outputFile)
	return nil
}

func generateDefaultFilename(urls []string) string {
	var hostnames []string
	for _, u := range urls {
		parsedURL, err := url.Parse(u)
		if err == nil {
			hostnames = append(hostnames, parsedURL.Hostname())
		}
	}

	var baseFilename string
	if len(hostnames) == 1 {
		baseFilename = hostnames[0]
	} else if len(hostnames) == 2 {
		baseFilename = fmt.Sprintf("%s-and-%s", hostnames[0], hostnames[1])
	} else if len(hostnames) > 2 {
		baseFilename = fmt.Sprintf("%s-and-%d-others", hostnames[0], len(hostnames)-1)
	} else {
		baseFilename = "web-content"
	}

	baseFilename = strings.NewReplacer(
		".com", "",
		".org", "",
		".net", "",
		".edu", "",
		".", "-",
	).Replace(baseFilename)

	if len(baseFilename) > 50 {
		baseFilename = baseFilename[:50]
	}

	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s.md", baseFilename, timestamp)
}

func extractAndConvertContent(urlStr string) (string, error) {
	content, err := scraper.FetchWebpageContent(urlStr)
	if err != nil {
		return "", fmt.Errorf("error fetching webpage content: %v", err)
	}

	// Use the CSS locator from the config
	cssLocator := cfg.Scrape.CSSLocator
	if cssLocator != "" {
		content, err = scraper.ExtractContentWithCSS(content, cssLocator)
		if err != nil {
			return "", fmt.Errorf("error extracting content with CSS selector: %v", err)
		}
	}

	// Create a new converter
	converter := md.NewConverter("", true, nil)

	// Convert HTML to Markdown
	markdown, err := converter.ConvertString(content)
	if err != nil {
		return "", fmt.Errorf("error converting HTML to Markdown: %v", err)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %v", err)
	}
	header := fmt.Sprintf("# Content from %s\n\n", parsedURL.String())

	return header + markdown + "\n\n", nil
}
