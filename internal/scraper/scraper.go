package scraper

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	"github.com/russross/blackfriday/v2"
)

var logger *log.Logger

var (
	pw      *playwright.Playwright
	browser playwright.Browser
)

// Config holds the scraper configuration
type Config struct {
	URLs       []URLConfig
	OutputType string
	Verbose    bool
}

// ScrapeMultipleURLs scrapes multiple URLs concurrently
func ScrapeMultipleURLs(config Config) (map[string]string, error) {
	results := make(chan struct {
		url     string
		content string
		err     error
	}, len(config.URLs))

	for _, urlConfig := range config.URLs {
		go func(cfg URLConfig) {
			content, err := scrapeURL(cfg)
			results <- struct {
				url     string
				content string
				err     error
			}{cfg.URL, content, err}
		}(urlConfig)
	}

	scrapedContent := make(map[string]string)
	for i := 0; i < len(config.URLs); i++ {
		result := <-results
		if result.err != nil {
			logger.Printf("Error scraping %s: %v\n", result.url, result.err)
			continue
		}
		scrapedContent[result.url] = result.content
	}

	return scrapedContent, nil
}

func scrapeURL(config URLConfig) (string, error) {
	content, err := FetchWebpageContent(config.URL)
	if err != nil {
		return "", err
	}

	if config.CSSLocator != "" {
		content, err = ExtractContentWithCSS(content, config.CSSLocator, nil)
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
	URL         string
	CSSLocator  string
	OutputAlias string
}

// SetupLogger initializes the logger based on the verbose flag
func SetupLogger(verbose bool) {
	if verbose {
		logger = log.New(log.Writer(), "SCRAPER: ", log.LstdFlags)
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}
}

// InitPlaywright initializes Playwright and launches the browser
func InitPlaywright() error {
	logger.Println("Initializing Playwright")
	var err error
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

// FetchWebpageContent retrieves the content of a webpage using Playwright
func FetchWebpageContent(urlStr string) (string, error) {
	log.Printf("Fetching webpage content for URL: %s\n", urlStr)

	page, err := browser.NewPage()
	if err != nil {
		log.Printf("Error creating new page: %v\n", err)
		return "", fmt.Errorf("could not create page: %v", err)
	}
	defer page.Close()

	time.Sleep(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)

	log.Printf("Navigating to URL: %s\n", urlStr)
	if _, err = page.Goto(urlStr, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		log.Printf("Error navigating to page: %v\n", err)
		return "", fmt.Errorf("could not go to page: %v", err)
	}

	log.Println("Waiting for page load state")
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		log.Printf("Error waiting for page load: %v\n", err)
		return "", fmt.Errorf("error waiting for page load: %v", err)
	}

	log.Println("Scrolling page")
	err = scrollPage(page)
	if err != nil {
		log.Printf("Error scrolling page: %v\n", err)
		return "", fmt.Errorf("error scrolling page: %v", err)
	}

	log.Println("Waiting for body element")
	_, err = page.WaitForSelector("body", playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	if err != nil {
		log.Printf("Error waiting for body: %v\n", err)
		return "", fmt.Errorf("error waiting for body: %v", err)
	}

	log.Println("Getting page content")
	content, err := page.Content()
	if err != nil {
		log.Printf("Error getting page content: %v\n", err)
		return "", fmt.Errorf("could not get page content: %v", err)
	}

	if content == "" {
		log.Println(" content is empty, falling back to body content")
		content, err = page.InnerHTML("body")
		if err != nil {
			log.Printf("Error getting body content: %v\n", err)
			return "", fmt.Errorf("could not get body content: %v", err)
		}
	}

	log.Printf("Successfully fetched webpage content (length: %d)\n", len(content))
	return content, nil
}

// ProcessHTMLContent converts HTML content to Markdown
func ProcessHTMLContent(htmlContent string, config Config) (string, error) {
	log.Printf("Processing HTML content (length: %d)\n", len(htmlContent))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML: %v\n", err)
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	selection := doc.Find("body")
	log.Println("Processing entire body")

	if selection.Length() == 0 {
		return "", fmt.Errorf("no content found in the document")
	}

	content, err := selection.Html()
	if err != nil {
		log.Printf("Error extracting content: %v\n", err)
		return "", fmt.Errorf("error extracting content: %v", err)
	}

	markdown := convertToMarkdown(content)
	log.Printf("Converted HTML to Markdown (length: %d)\n", len(markdown))
	return markdown, nil
}

func convertToMarkdown(html string) string {
	// Use a simple HTML-to-Markdown conversion
	markdown := blackfriday.Run([]byte(html),
		blackfriday.WithExtensions(blackfriday.CommonExtensions|blackfriday.HardLineBreak))
	return string(markdown)
}

func scrollPage(page playwright.Page) error {
	log.Println("Starting page scroll")
	script := `
		() => {
			window.scrollTo(0, document.body.scrollHeight);
			return document.body.scrollHeight;
		}
	`

	previousHeight := 0
	for i := 0; i < 250; i++ {
		height, err := page.Evaluate(script)
		if err != nil {
			log.Printf("Error scrolling (iteration %d): %v\n", i+1, err)
			return fmt.Errorf("error scrolling: %v", err)
		}

		var currentHeight int
		switch v := height.(type) {
		case int:
			currentHeight = v
		case float64:
			currentHeight = int(v)
		default:
			log.Printf("Unexpected height type: %T\n", height)
			return fmt.Errorf("unexpected height type: %T", height)
		}

		log.Printf("Scroll iteration %d: height = %d\n", i+1, currentHeight)

		if currentHeight == previousHeight {
			log.Println("Reached bottom of the page")
			break
		}

		previousHeight = currentHeight

		page.WaitForTimeout(500)
	}

	log.Println("Scrolling back to top")
	_, err := page.Evaluate(`() => { window.scrollTo(0, 0); }`)
	if err != nil {
		log.Printf("Error scrolling back to top: %v\n", err)
		return fmt.Errorf("error scrolling back to top: %v", err)
	}

	log.Println("Page scroll completed")
	return nil
}

// ExtractLinks extracts all links from the given URL
func ExtractLinks(urlStr string) ([]string, error) {
	log.Printf("Extracting links from URL: %s\n", urlStr)

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
		result = append(result, link.(string))
	}

	log.Printf("Extracted %d links\n", len(result))
	return result, nil
}

// ExtractContentWithCSS extracts content from HTML using a CSS selector
func ExtractContentWithCSS(content, includeSelector string, excludeSelectors []string) (string, error) {
	log.Printf("Extracting content with CSS selector: %s\n", includeSelector)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	selection := doc.Find(includeSelector)
	if selection.Length() == 0 {
		log.Printf("Warning: No content found with CSS selector: %s. Falling back to body content.\n", includeSelector)
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

	log.Printf("Extracted content length: %d\n", len(selectedContent))
	return selectedContent, nil
}
