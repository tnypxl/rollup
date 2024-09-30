package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
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
	MaxDepth         int
	AllowedPaths     []string
	ExcludePaths     []string
	OutputAlias      string
	PathOverrides    []PathOverride
	LinksContainerSelector string
}

// PathOverride holds path-specific overrides
type PathOverride struct {
	Path             string
	CSSLocator       string
	ExcludeSelectors []string
}

func ScrapeSites(config Config) (map[string]string, error) {
	logger.Println("Starting ScrapeSites function - Verbose mode is active")
	results := make(chan struct {
		url     string
		content string
		err     error
	})

	limiter := rate.NewLimiter(rate.Limit(config.Scrape.RequestsPerSecond), config.Scrape.BurstLimit)
	logger.Printf("Rate limiter configured with %f requests per second and burst limit of %d\n", config.Scrape.RequestsPerSecond, config.Scrape.BurstLimit)

	var wg sync.WaitGroup
	totalURLs := 0
	var mu sync.Mutex
	for _, site := range config.Sites {
		logger.Printf("Processing site: %s\n", site.BaseURL)
		wg.Add(1)
		go func(site SiteConfig) {
			defer wg.Done()
			visited := make(map[string]bool)
			for _, path := range site.AllowedPaths {
				fullURL := site.BaseURL + path
				mu.Lock()
				totalURLs++
				mu.Unlock()
				logger.Printf("Queueing URL for scraping: %s\n", fullURL)
				scrapeSingleURL(fullURL, site, results, limiter, visited, 0)
			}
		}(site)
	}

	go func() {
		wg.Wait()
		close(results)
		logger.Println("All goroutines completed, results channel closed")
	}()

	scrapedContent := make(map[string]string)
	for result := range results {
		if result.err != nil {
			logger.Printf("Error scraping %s: %v\n", result.url, result.err)
			continue
		}
		logger.Printf("Successfully scraped content from %s (length: %d)\n", result.url, len(result.content))
		scrapedContent[result.url] = result.content
	}

	logger.Printf("Total URLs processed: %d\n", totalURLs)
	logger.Printf("Successfully scraped content from %d URLs\n", len(scrapedContent))

	return scrapedContent, nil
}

func scrapeSingleURL(url string, site SiteConfig, results chan<- struct {
	url     string
	content string
	err     error
}, limiter *rate.Limiter, visited map[string]bool, currentDepth int) {
	if site.MaxDepth > 0 && currentDepth > site.MaxDepth {
		return
	}

	if visited[url] {
		return
	}
	visited[url] = true

	logger.Printf("Starting to scrape URL: %s\n", url)

	// Wait for rate limiter before making the request
	err := limiter.Wait(context.Background())
	if err != nil {
		logger.Printf("Rate limiter error for %s: %v\n", url, err)
		results <- struct {
			url     string
			content string
			err     error
		}{url, "", fmt.Errorf("rate limiter error: %v", err)}
		return
	}

	content, err := FetchWebpageContent(url)
	if err != nil {
		logger.Printf("Error fetching content for %s: %v\n", url, err)
		results <- struct {
			url     string
			content string
			err     error
		}{url, "", err}
		return
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		logger.Printf("Error parsing HTML for %s: %v\n", url, err)
		results <- struct {
			url     string
			content string
			err     error
		}{url, "", fmt.Errorf("error parsing HTML: %v", err)}
		return
	}

	if site.LinksContainerSelector != "" {
		logger.Printf("Processing links container for %s\n", url)
		linkContainers := doc.Find(site.LinksContainerSelector)
		linkContainers.Each(func(i int, container *goquery.Selection) {
			container.Find("a[href]").Each(func(j int, link *goquery.Selection) {
				href, exists := link.Attr("href")
				if exists {
					resolvedURL := resolveURL(href, url)
					if isAllowedURL(resolvedURL, site) && !visited[resolvedURL] {
						go scrapeSingleURL(resolvedURL, site, results, limiter, visited, currentDepth+1)
					}
				}
			})
		})
		return
	}

	cssLocator, excludeSelectors := getOverrides(url, site)
	logger.Printf("Using CSS locator for %s: %s\n", url, cssLocator)
	logger.Printf("Exclude selectors for %s: %v\n", url, excludeSelectors)

	extractedContent, err := ExtractContentWithCSS(content, cssLocator, excludeSelectors)
	if err != nil {
		logger.Printf("Error extracting content for %s: %v\n", url, err)
		results <- struct {
			url     string
			content string
			err     error
		}{url, "", err}
		return
	}

	if extractedContent == "" {
		logger.Printf("Warning: Empty content scraped from %s\n", url)
	} else {
		logger.Printf("Successfully scraped content from %s (length: %d)\n", url, len(extractedContent))
	}

	results <- struct {
		url     string
		content string
		err     error
	}{url, extractedContent, nil}
}

func scrapeSite(site SiteConfig, results chan<- struct {
	url     string
	content string
	err     error
}, limiter *rate.Limiter,
) {
	visited := make(map[string]bool)
	queue := []string{site.BaseURL}

	for len(queue) > 0 {
		url := queue[0]
		queue = queue[1:]

		if visited[url] {
			continue
		}
		visited[url] = true

		if !isAllowedURL(url, site) {
			continue
		}

		// Wait for rate limiter before making the request
		err := limiter.Wait(context.Background())
		if err != nil {
			results <- struct {
				url     string
				content string
				err     error
			}{url, "", fmt.Errorf("rate limiter error: %v", err)}
			continue
		}

		cssLocator, excludeSelectors := getOverrides(url, site)
		content, err := scrapeURL(url, cssLocator, excludeSelectors)
		results <- struct {
			url     string
			content string
			err     error
		}{url, content, err}

		if len(visited) < site.MaxDepth {
			links, _ := ExtractLinks(url)
			for _, link := range links {
				if !visited[link] && isAllowedURL(link, site) {
					queue = append(queue, link)
				}
			}
		}
	}
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
	
	// Check if the URL is within allowed paths
	if len(site.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range site.AllowedPaths {
			if strings.HasPrefix(path, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check if the URL is in excluded paths
	for _, excludePath := range site.ExcludePaths {
		if strings.HasPrefix(path, excludePath) {
			return false
		}
	}

	return true
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
	// Remove any character that isn't alphanumeric, dash, or underscore
	reg, _ := regexp.Compile("[^a-zA-Z0-9-_]+")
	return reg.ReplaceAllString(name, "_")
}

// URLConfig holds configuration for a single URL
type URLConfig struct {
	URL              string
	CSSLocator       string
	ExcludeSelectors []string
	OutputAlias      string
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

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

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

// ExtractLinks extracts all links from the given URL
func ExtractLinks(urlStr string) ([]string, error) {
	logger.Printf("Extracting links from URL: %s\n", urlStr)

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %v", err)
	}
	defer page.Close()

	if _, err = page.Goto(urlStr, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return nil, fmt.Errorf("could not go to page: %v", err)
	}

	links, err := page.Evaluate(`() => {
		const anchors = document.querySelectorAll('a');
		return Array.from(anchors).map(a => a.href);
	}`)
	if err != nil {
		return nil, fmt.Errorf("could not extract links: %v", err)
	}

	var result []string
	for _, link := range links.([]interface{}) {
		// Normalize URL by removing trailing slash
		normalizedLink := strings.TrimRight(link.(string), "/")
		result = append(result, normalizedLink)
	}

	logger.Printf("Extracted %d links\n", len(result))
	return result, nil
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
func resolveURL(href, base string) string {
    parsedBase, err := url.Parse(base)
    if err != nil {
        return href
    }
    parsedHref, err := url.Parse(href)
    if err != nil {
        return href
    }
    return parsedBase.ResolveReference(parsedHref).String()
}
