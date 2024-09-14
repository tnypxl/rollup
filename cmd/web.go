package cmd

import (
	"fmt"
	"net/url"
	"os"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/config"
	"github.com/tnypxl/rollup/internal/scraper"
)

var (
	urls          []string
	outputFile    string
	depth         int
	cssSelector   string
	xpathSelector string
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
	webCmd.Flags().IntVarP(&depth, "depth", "d", 0, "Depth of link traversal (default: 0, only scrape the given URLs)")
	webCmd.Flags().StringVar(&cssSelector, "css", "", "CSS selector to extract specific content")
	webCmd.Flags().StringVar(&xpathSelector, "xpath", "", "XPath selector to extract specific content")
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
		extractedContent, err := scrapeRecursively(u, depth)
		if err != nil {
			return fmt.Errorf("error scraping content from %s: %v", u, err)
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
	// Simple implementation for now
	return "rollup-web-content.md"
}

func scrapeRecursively(url string, depth int) (string, error) {
	// Simple implementation for now
	return extractAndConvertContent(url)
}

func extractAndConvertContent(urlStr string) (string, error) {
	content, err := scraper.FetchWebpageContent(urlStr)
	if err != nil {
		return "", fmt.Errorf("error fetching webpage content: %v", err)
	}

	if cssSelector != "" || xpathSelector != "" {
		// TODO: Implement content extraction with CSS or XPath selector
		// For now, we'll just use the full content
		fmt.Println("Warning: CSS and XPath selectors are not yet implemented")
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
