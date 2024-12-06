package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	"golang.org/x/time/rate"
)

var logger *log.Logger

var (
	pw      *playwright.Playwright
	browser playwright.Browser
)

// Config holds the scraper configuration
type Config struct {
	Sites      []SiteConfig
	OutputType string
	Verbose    bool
	Scrape     ScrapeConfig
}

// ScrapeConfig holds the scraping-specific configuration
type ScrapeConfig struct {
	RequestsPerSecond float64
	BurstLimit        int
}

// SiteConfig holds configuration for a single site
type SiteConfig struct {
	BaseURL          string
	CSSLocator       string
	ExcludeSelectors []string
	AllowedPaths     []string
	ExcludePaths     []string
	FileNamePrefix   string
	PathOverrides    []PathOverride
}

// PathOverride holds path-specific overrides
type PathOverride struct {
	Path             string
	CSSLocator       string
	ExcludeSelectors []string
}

func ScrapeSites(config Config) error {
	logger.Println("Starting ScrapeSites function - Verbose mode is active")
	results := make(chan struct {
		url     string
		content string
		site    SiteConfig // Add site config to track which site the content came from
		err     error
	})

	limiter := rate.NewLimiter(rate.Limit(config.Scrape.RequestsPerSecond), config.Scrape.BurstLimit)
	logger.Printf("Rate limiter configured with %f requests per second and burst limit of %d\n",
		config.Scrape.RequestsPerSecond, config.Scrape.BurstLimit)

	var wg sync.WaitGroup
	totalURLs := 0
	for _, site := range config.Sites {
		logger.Printf("Processing site: %s\n", site.BaseURL)
		wg.Add(1)
		go func(site SiteConfig) {
			defer wg.Done()
			for _, path := range site.AllowedPaths {
				fullURL := site.BaseURL + path
				totalURLs++
				logger.Printf("Queueing URL for scraping: %s\n", fullURL)
				scrapeSingleURL(fullURL, site, results, limiter)
			}
		}(site)
	}

	go func() {
		wg.Wait()
		close(results)
		logger.Println("All goroutines completed, results channel closed")
	}()

	// Use a map that includes site configuration
	scrapedContent := make(map[string]struct {
		content string
		site    SiteConfig
	})

	for result := range results {
		if result.err != nil {
			logger.Printf("Error scraping %s: %v\n", result.url, result.err)
			continue
		}
		logger.Printf("Successfully scraped content from %s (length: %d)\n",
			result.url, len(result.content))
		scrapedContent[result.url] = struct {
			content string
			site    SiteConfig
		}{
			content: result.content,
			site:    result.site,
		}
	}

	logger.Printf("Total URLs processed: %d\n", totalURLs)
	logger.Printf("Successfully scraped content from %d URLs\n", len(scrapedContent))

	return SaveToFiles(scrapedContent, config)
}

func scrapeSingleURL(url string, site SiteConfig, results chan<- struct {
	url     string
	content string
	site    SiteConfig
	err     error
}, limiter *rate.Limiter) {
	logger.Printf("Starting to scrape URL: %s\n", url)

	err := limiter.Wait(context.Background())
	if err != nil {
		results <- struct {
			url     string
			content string
			site    SiteConfig
			err     error
		}{url, "", site, fmt.Errorf("rate limiter error: %v", err)}
		return
	}

	cssLocator, excludeSelectors := getOverrides(url, site)
	content, err := scrapeURL(url, cssLocator, excludeSelectors)
	if err != nil {
		results <- struct {
			url     string
			content string
			site    SiteConfig
			err     error
		}{url, "", site, err}
		return
	}

	results <- struct {
		url     string
		content string
		site    SiteConfig
		err     error
	}{url, content, site, nil}
}

func isAllowedURL(urlStr string, site SiteConfig) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	baseURL, _ := url.Parse(site.BaseURL)
	if parsedURL.Host != baseURL.Host {
		return false
	}

	path := parsedURL.Path
	for _, allowedPath := range site.AllowedPaths {
		if strings.HasPrefix(path, allowedPath) {
			for _, excludePath := range site.ExcludePaths {
				if strings.HasPrefix(path, excludePath) {
					return false
				}
			}
			return true
		}
	}

	return false
}

func getOverrides(urlStr string, site SiteConfig) (string, []string) {
	parsedURL, _ := url.Parse(urlStr)
	path := parsedURL.Path

	for _, override := range site.PathOverrides {
		if strings.HasPrefix(path, override.Path) {
			if override.CSSLocator != "" {
				return override.CSSLocator, override.ExcludeSelectors
			}
			return site.CSSLocator, override.ExcludeSelectors
		}
	}

	return site.CSSLocator, site.ExcludeSelectors
}

func scrapeURL(url, cssLocator string, excludeSelectors []string) (string, error) {
	content, err := FetchWebpageContent(url)
	if err != nil {
		return "", err
	}

	if cssLocator != "" {
		content, err = ExtractContentWithCSS(content, cssLocator, excludeSelectors)
		if err != nil {
			return "", err
		}
	}

	return ProcessHTMLContent(content, Config{})
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
	// Replace all non-alphanumeric characters with dashes
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	name = reg.ReplaceAllString(name, "-")
	// Remove any leading or trailing dashes
	name = strings.Trim(name, "-")
	// Collapse multiple consecutive dashes into one
	reg = regexp.MustCompile("-+")
	return reg.ReplaceAllString(name, "-")
}

// URLConfig holds configuration for a single URL
type URLConfig struct {
	URL              string
	CSSLocator       string
	ExcludeSelectors []string
	FileNamePrefix   string
}

// SetupLogger initializes the logger based on the verbose flag
func SetupLogger(verbose bool) {
	if verbose {
		logger = log.New(os.Stdout, "SCRAPER: ", log.LstdFlags)
	} else {
		logger = log.New(io.Discard, "", 0)
	}
}

// InitPlaywright initializes Playwright and launches the browser
func InitPlaywright() error {
	logger.Println("Initializing Playwright")
	var err error

	// Install Playwright and Chromium browser
	err = playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}})
	if err != nil {
		return fmt.Errorf("could not install Playwright and Chromium: %v", err)
	}

	pw, err = playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start Playwright: %v", err)
	}

	userAgent := "Mozilla/5.0 (Linux; Android 15; Pixel 9 Build/AP3A.241105.008) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.6723.106 Mobile Safari/537.36 OPX/2.5"

	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Args: []string{fmt.Sprintf("--user-agent=%s", userAgent)},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %v", err)
	}

	logger.Println("Playwright initialized successfully")
	return nil
}

// ClosePlaywright closes the browser and stops Playwright
func ClosePlaywright() {
	if browser != nil {
		browser.Close()
	}
	if pw != nil {
		pw.Stop()
	}
}

// InitBrowser initializes the browser
func InitBrowser() error {
	return InitPlaywright()
}

// CloseBrowser closes the browser
func CloseBrowser() {
	ClosePlaywright()
}

// SaveToFiles writes the scraped content to files based on output type
func SaveToFiles(content map[string]struct {
	content string
	site    SiteConfig
}, config Config) error {
	if config.OutputType == "" {
		config.OutputType = "separate" // default to separate files if not specified
	}

	switch config.OutputType {
	case "single":
		if err := os.MkdirAll("output", 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
		var combined strings.Builder
		for url, data := range content {
			combined.WriteString(fmt.Sprintf("## %s\n\n", url))
			combined.WriteString(data.content)
			combined.WriteString("\n\n")
		}
		return os.WriteFile(filepath.Join("output", "combined.md"), []byte(combined.String()), 0644)

	case "separate":
		if err := os.MkdirAll("output", 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Group content by site and path
		contentBySitePath := make(map[string]map[string]string)
		for urlStr, data := range content {
			parsedURL, err := url.Parse(urlStr)
			if err != nil {
				logger.Printf("Warning: Could not parse URL %s: %v", urlStr, err)
				continue
			}

			// Find matching allowed path for this URL
			var matchingPath string
			for _, path := range data.site.AllowedPaths {
				if strings.HasPrefix(parsedURL.Path, path) {
					matchingPath = path
					break
				}
			}
			if matchingPath == "" {
				logger.Printf("Warning: No matching allowed path for URL %s", urlStr)
				continue
			}

			siteKey := fmt.Sprintf("%s-%s", data.site.BaseURL, data.site.FileNamePrefix)
			if contentBySitePath[siteKey] == nil {
				contentBySitePath[siteKey] = make(map[string]string)
			}

			// Combine all content for the same path
			if existing, exists := contentBySitePath[siteKey][matchingPath]; exists {
				contentBySitePath[siteKey][matchingPath] = existing + "\n\n" + data.content
			} else {
				contentBySitePath[siteKey][matchingPath] = data.content
			}
		}

		// Write files for each site and path
		for siteKey, pathContent := range contentBySitePath {
			for path, content := range pathContent {
				parts := strings.SplitN(siteKey, "-", 2) // Split only on first hyphen
				prefix := parts[1]                       // Get the FileNamePrefix part
				if prefix == "" {
					prefix = "doc" // default prefix if none specified
				}

				normalizedPath := NormalizePathForFilename(path)
				if normalizedPath == "" {
					normalizedPath = "index"
				}

				filename := filepath.Join("output", fmt.Sprintf("%s-%s.md",
					prefix, normalizedPath))

				// Ensure we don't have empty files
				if strings.TrimSpace(content) == "" {
					logger.Printf("Skipping empty content for path %s", path)
					continue
				}

				if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
					return fmt.Errorf("failed to write file %s: %v", filename, err)
				}
				logger.Printf("Wrote content to %s", filename)
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported output type: %s", config.OutputType)
	}
}

// NormalizePathForFilename converts a URL path into a valid filename component
func NormalizePathForFilename(urlPath string) string {
	// Remove leading/trailing slashes
	path := strings.Trim(urlPath, "/")
	// Replace all non-alphanumeric characters with dashes
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	path = reg.ReplaceAllString(path, "-")
	// Remove any leading or trailing dashes
	path = strings.Trim(path, "-")
	// Collapse multiple consecutive dashes into one
	reg = regexp.MustCompile("-+")
	path = reg.ReplaceAllString(path, "-")
	return path
}

// FetchWebpageContent retrieves the content of a webpage using Playwright
func FetchWebpageContent(urlStr string) (string, error) {
	logger.Printf("Fetching webpage content for URL: %s\n", urlStr)

	page, err := browser.NewPage()
	if err != nil {
		logger.Printf("Error creating new page: %v\n", err)
		return "", fmt.Errorf("could not create page: %v", err)
	}
	defer page.Close()

	time.Sleep(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)

	logger.Printf("Navigating to URL: %s\n", urlStr)
	if _, err = page.Goto(urlStr, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		logger.Printf("Error navigating to page: %v\n", err)
		return "", fmt.Errorf("could not go to page: %v", err)
	}

	logger.Println("Waiting for page load state")
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		logger.Printf("Error waiting for page load: %v\n", err)
		return "", fmt.Errorf("error waiting for page load: %v", err)
	}

	logger.Println("Scrolling page")
	err = scrollPage(page)
	if err != nil {
		logger.Printf("Error scrolling page: %v\n", err)
		return "", fmt.Errorf("error scrolling page: %v", err)
	}

	logger.Println("Waiting for body element")

	bodyElement := page.Locator("body")
	err = bodyElement.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	if err != nil {
		logger.Printf("Error waiting for body: %v\n", err)
		return "", fmt.Errorf("error waiting for body: %v", err)
	}

	logger.Println("Getting page content")
	content, err := page.Content()
	if err != nil {
		logger.Printf("Error getting page content: %v\n", err)
		return "", fmt.Errorf("could not get page content: %v", err)
	}

	if content == "" {
		logger.Println(" content is empty, falling back to body content")
		content, err = bodyElement.InnerHTML()
		if err != nil {
			logger.Printf("Error getting body content: %v\n", err)
			return "", fmt.Errorf("could not get body content: %v", err)
		}
	}

	logger.Printf("Successfully fetched webpage content (length: %d)\n", len(content))
	return content, nil
}

// ProcessHTMLContent converts HTML content to Markdown
func ProcessHTMLContent(htmlContent string, config Config) (string, error) {
	logger.Printf("Processing HTML content (length: %d)\n", len(htmlContent))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		logger.Printf("Error parsing HTML: %v\n", err)
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	selection := doc.Find("body")
	logger.Println("Processing entire body")

	if selection.Length() == 0 {
		return "", fmt.Errorf("no content found in the document")
	}

	content, err := selection.Html()
	if err != nil {
		logger.Printf("Error extracting content: %v\n", err)
		return "", fmt.Errorf("error extracting content: %v", err)
	}

	// Create a new converter
	converter := md.NewConverter("", true, nil)

	// Convert HTML to Markdown
	markdown, err := converter.ConvertString(content)
	if err != nil {
		logger.Printf("Error converting HTML to Markdown: %v\n", err)
		return "", fmt.Errorf("error converting HTML to Markdown: %v", err)
	}

	logger.Printf("Converted HTML to Markdown (length: %d)\n", len(markdown))
	return markdown, nil
}

func scrollPage(page playwright.Page) error {
	logger.Println("Starting page scroll")
	script := `
		() => {
			window.scrollTo(0, document.body.scrollHeight);
			return document.body.scrollHeight;
			// wait for 500 ms
			new Promise(resolve => setTimeout(resolve, 500));
		}
	`

	previousHeight := 0
	for i := 0; i < 250; i++ {
		height, err := page.Evaluate(script)
		if err != nil {
			logger.Printf("Error scrolling (iteration %d): %v\n", i+1, err)
			return fmt.Errorf("error scrolling: %v", err)
		}

		var currentHeight int
		switch v := height.(type) {
		case int:
			currentHeight = v
		case float64:
			currentHeight = int(v)
		default:
			logger.Printf("Unexpected height type: %T\n", height)
			return fmt.Errorf("unexpected height type: %T", height)
		}

		logger.Printf("Scroll iteration %d: height = %d\n", i+1, currentHeight)

		if currentHeight == previousHeight {
			logger.Println("Reached bottom of the page")
			break
		}

		previousHeight = currentHeight

		// Wait for a while before scrolling again

	}

	logger.Println("Scrolling back to top")
	_, err := page.Evaluate(`() => { window.scrollTo(0, 0); }`)
	if err != nil {
		logger.Printf("Error scrolling back to top: %v\n", err)
		return fmt.Errorf("error scrolling back to top: %v", err)
	}

	logger.Println("Page scroll completed")
	return nil
}

// ExtractContentWithCSS extracts content from HTML using a CSS selector
func ExtractContentWithCSS(content, includeSelector string, excludeSelectors []string) (string, error) {
	logger.Printf("Extracting content with CSS selector: %s\n", includeSelector)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	selection := doc.Find(includeSelector)
	if selection.Length() == 0 {
		logger.Printf("Warning: No content found with CSS selector: %s. Falling back to body content.\n", includeSelector)
		selection = doc.Find("body")
		if selection.Length() == 0 {
			return "", fmt.Errorf("no content found in body")
		}
	}

	for _, excludeSelector := range excludeSelectors {
		selection.Find(excludeSelector).Remove()
	}

	selectedContent, err := selection.Html()
	if err != nil {
		return "", fmt.Errorf("error extracting content with CSS selector: %v", err)
	}

	// Trim leading and trailing whitespace
	selectedContent = strings.TrimSpace(selectedContent)

	// Normalize newlines
	selectedContent = strings.ReplaceAll(selectedContent, "\r\n", "\n")
	selectedContent = strings.ReplaceAll(selectedContent, "\r", "\n")

	// Remove indentation while preserving structure
	lines := strings.Split(selectedContent, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	selectedContent = strings.Join(lines, "\n")

	// Remove any leading or trailing newlines
	selectedContent = strings.Trim(selectedContent, "\n")

	logger.Printf("Extracted content length: %d\n", len(selectedContent))
	return selectedContent, nil
}
