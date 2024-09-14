package scraper

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	"github.com/russross/blackfriday/v2"
)

var (
	pw      *playwright.Playwright
	browser playwright.Browser
)

// Config holds the scraper configuration
type Config struct {
	CSSLocator string
}

// InitPlaywright initializes Playwright and launches the browser
func InitPlaywright() error {
	log.Println("Initializing Playwright")
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

	log.Println("Playwright initialized successfully")
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

	log.Println("Getting main content")
	content, err := page.InnerHTML("main")
	if err != nil {
		log.Printf("Error getting main content: %v\n", err)
		return "", fmt.Errorf("could not get main content: %v", err)
	}

	if content == "" {
		log.Println("Main content is empty, falling back to body content")
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

	var content string
	if config.CSSLocator != "" {
		log.Printf("Using CSS locator: %s\n", config.CSSLocator)
		content, err = doc.Find(config.CSSLocator).Html()
		if err != nil {
			log.Printf("Error extracting content with CSS locator: %v\n", err)
			return "", fmt.Errorf("error extracting content with CSS locator: %v", err)
		}
	} else {
		log.Println("No CSS locator provided, processing entire body")
		content, err = doc.Find("body").Html()
		if err != nil {
			log.Printf("Error extracting body content: %v\n", err)
			return "", fmt.Errorf("error extracting body content: %v", err)
		}
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
func ExtractContentWithCSS(content, selector string) (string, error) {
	log.Printf("Extracting content with CSS selector: %s\n", selector)
	
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	selection := doc.Find(selector)
	if selection.Length() == 0 {
		return "", fmt.Errorf("no content found with CSS selector: %s", selector)
	}

	selectedContent, err := selection.Html()
	if err != nil {
		return "", fmt.Errorf("error extracting content with CSS selector: %v", err)
	}

	log.Printf("Extracted content length: %d\n", len(selectedContent))
	return selectedContent, nil
}

// ExtractContentWithXPath extracts content from HTML using an XPath selector
func ExtractContentWithXPath(content, xpath string) (string, error) {
	log.Printf("Extracting content with XPath selector: %s\n", xpath)
	
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	var selectedContent string
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		if content, err := s.Html(); err == nil {
			selectedContent = content
		}
	})

	if selectedContent == "" {
		return "", fmt.Errorf("no content found with XPath selector: %s", xpath)
	}

	log.Printf("Extracted content length: %d\n", len(selectedContent))
	return selectedContent, nil
}
