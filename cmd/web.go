package cmd

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/scraper"
)

var (
	urls             []string
	outputType       string
	depth            int
	includeSelector  string
	excludeSelectors []string
)

var scraperConfig scraper.Config

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Scrape main content from webpages and convert to Markdown",
	Long:  `Scrape the main content from one or more webpages, ignoring navigational elements, ads, and other UI aspects. Convert the content to a well-structured Markdown file.`,
	RunE:  runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.Flags().StringSliceVarP(&urls, "urls", "u", []string{}, "URLs of the webpages to scrape (comma-separated)")
	webCmd.Flags().StringVarP(&outputType, "output", "o", "single", "Output type: 'single' for one file, 'separate' for multiple files")
	webCmd.Flags().IntVarP(&depth, "depth", "d", 0, "Depth of link traversal (default: 0, only scrape the given URLs)")
	webCmd.Flags().StringVar(&includeSelector, "css", "", "CSS selector to extract specific content")
	webCmd.Flags().StringSliceVar(&excludeSelectors, "exclude", []string{}, "CSS selectors to exclude from the extracted content (comma-separated)")
}

func runWeb(cmd *cobra.Command, args []string) error {
	scraperConfig.Verbose = verbose

	// Use config if available, otherwise use command-line flags
	var urlConfigs []scraper.URLConfig
	if len(urls) == 0 && len(cfg.Scrape.URLs) > 0 {
		urlConfigs = make([]scraper.URLConfig, len(cfg.Scrape.URLs))
		for i, u := range cfg.Scrape.URLs {
			urlConfigs[i] = scraper.URLConfig{
				URL:         u.URL,
				CSSLocator:  u.CSSLocator,
				OutputAlias: u.OutputAlias,
			}
		}
	} else {
		urlConfigs = make([]scraper.URLConfig, len(urls))
		for i, u := range urls {
			urlConfigs[i] = scraper.URLConfig{URL: u, CSSLocator: includeSelector}
		}
	}

	if len(urlConfigs) == 0 {
		return fmt.Errorf("no URLs provided. Use --urls flag with comma-separated URLs or set 'scrape.urls' in the rollup.yml file")
	}

	scraperConfig := scraper.Config{
		URLs:       urlConfigs,
		OutputType: outputType,
		Verbose:    verbose,
	}

	scrapedContent, err := scraper.ScrapeMultipleURLs(scraperConfig)
	if err != nil {
		return fmt.Errorf("error scraping content: %v", err)
	}

	if outputType == "single" {
		return writeSingleFile(scrapedContent)
	} else {
		return writeMultipleFiles(scrapedContent)
	}
}

func writeSingleFile(content map[string]string) error {
	outputFile := generateDefaultFilename(urls)
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer file.Close()

	for url, c := range content {
		_, err = file.WriteString(fmt.Sprintf("# Content from %s\n\n%s\n\n---\n\n", url, c))
		if err != nil {
			return fmt.Errorf("error writing content to file: %v", err)
		}
	}

	fmt.Printf("Content has been extracted from %d URL(s) and saved to %s\n", len(content), outputFile)
	return nil
}

func writeMultipleFiles(content map[string]string) error {
	for url, c := range content {
		filename := getFilenameFromContent(c, url)
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("error creating output file %s: %v", filename, err)
		}

		_, err = file.WriteString(fmt.Sprintf("# Content from %s\n\n%s", url, c))
		file.Close()
		if err != nil {
			return fmt.Errorf("error writing content to file %s: %v", filename, err)
		}

		fmt.Printf("Content from %s has been saved to %s\n", url, filename)
	}
	return nil
}

func generateDefaultFilename(urls []string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("rollup-web-%s.md", timestamp)
}

func scrapeRecursively(urlStr string, depth int) (string, error) {
	visited := make(map[string]bool)
	return scrapeURL(urlStr, depth, visited)
}

func scrapeURL(urlStr string, depth int, visited map[string]bool) (string, error) {
	if depth < 0 || visited[urlStr] {
		return "", nil
	}

	visited[urlStr] = true

	content, err := extractAndConvertContent(urlStr)
	if err != nil {
		return "", err
	}

	if depth > 0 {
		links, err := scraper.ExtractLinks(urlStr)
		if err != nil {
			return content, fmt.Errorf("error extracting links: %v", err)
		}

		for _, link := range links {
			subContent, err := scrapeURL(link, depth-1, visited)
			if err != nil {
				fmt.Printf("Warning: Error scraping %s: %v\n", link, err)
				continue
			}
			content += "\n\n---\n\n" + subContent
		}
	}

	return content, nil
}

func extractAndConvertContent(urlStr string) (string, error) {
	content, err := scraper.FetchWebpageContent(urlStr)
	if err != nil {
		return "", fmt.Errorf("error fetching webpage content: %v", err)
	}

	if includeSelector != "" {
		content, err = scraper.ExtractContentWithCSS(content, includeSelector, excludeSelectors)
		if err != nil {
			return "", fmt.Errorf("error extracting content with CSS: %v", err)
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

func getFilenameFromContent(content, url string) string {
	// Try to extract title from content
	titleStart := strings.Index(content, "<title>")
	titleEnd := strings.Index(content, "</title>")
	if titleStart != -1 && titleEnd != -1 && titleEnd > titleStart {
		title := content[titleStart+7 : titleEnd]
		return sanitizeFilename(title) + ".md"
	}

	// If no title found, use the URL
	return sanitizeFilename(url) + ".md"
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
