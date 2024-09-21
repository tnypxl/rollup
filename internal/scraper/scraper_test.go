package scraper

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"strings"
	"reflect"
	"log"
	"io/ioutil"
)

func TestIsAllowedURL(t *testing.T) {
	site := SiteConfig{
		BaseURL:      "https://example.com",
		AllowedPaths: []string{"/blog", "/products"},
		ExcludePaths: []string{"/admin", "/private"},
	}

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/blog/post1", true},
		{"https://example.com/products/item1", true},
		{"https://example.com/admin/dashboard", false},
		{"https://example.com/private/data", false},
		{"https://example.com/other/page", false},
		{"https://othersite.com/blog/post1", false},
	}

	for _, test := range tests {
		result := isAllowedURL(test.url, site)
		if result != test.expected {
			t.Errorf("isAllowedURL(%q) = %v, want %v", test.url, result, test.expected)
		}
	}
}

func TestGetOverrides(t *testing.T) {
	site := SiteConfig{
		CSSLocator:       "main",
		ExcludeSelectors: []string{".ads"},
		PathOverrides: []PathOverride{
			{
				Path:             "/special",
				CSSLocator:       ".special-content",
				ExcludeSelectors: []string{".sidebar"},
			},
		},
	}

	tests := []struct {
		url               string
		expectedLocator   string
		expectedExcludes  []string
	}{
		{"https://example.com/normal", "main", []string{".ads"}},
		{"https://example.com/special", ".special-content", []string{".sidebar"}},
		{"https://example.com/special/page", ".special-content", []string{".sidebar"}},
	}

	for _, test := range tests {
		locator, excludes := getOverrides(test.url, site)
		if locator != test.expectedLocator {
			t.Errorf("getOverrides(%q) locator = %q, want %q", test.url, locator, test.expectedLocator)
		}
		if !reflect.DeepEqual(excludes, test.expectedExcludes) {
			t.Errorf("getOverrides(%q) excludes = %v, want %v", test.url, excludes, test.expectedExcludes)
		}
	}
}

func TestExtractContentWithCSS(t *testing.T) {
	// Initialize logger for testing
	logger = log.New(ioutil.Discard, "", 0)

	html := `
		<html>
			<body>
				<main>
					<h1>Main Content</h1>
					<p>This is the main content.</p>
					<div class="ads">Advertisement</div>
				</main>
				<aside>Sidebar content</aside>
			</body>
		</html>
	`

	tests := []struct {
		includeSelector  string
		excludeSelectors []string
		expected         string
	}{
		{"main", nil, "<h1>Main Content</h1>\n<p>This is the main content.</p>\n<div class=\"ads\">Advertisement</div>"},
		{"main", []string{".ads"}, "<h1>Main Content</h1>\n<p>This is the main content.</p>"},
		{"aside", nil, "Sidebar content"},
	}

	for _, test := range tests {
		result, err := ExtractContentWithCSS(html, test.includeSelector, test.excludeSelectors)
		if err != nil {
			t.Errorf("ExtractContentWithCSS() returned error: %v", err)
			continue
		}
		if strings.TrimSpace(result) != strings.TrimSpace(test.expected) {
			t.Errorf("ExtractContentWithCSS() = %q, want %q", result, test.expected)
		}
	}
}

func TestProcessHTMLContent(t *testing.T) {
	html := `
		<html>
			<body>
				<h1>Test Heading</h1>
				<p>This is a <strong>test</strong> paragraph.</p>
				<ul>
					<li>Item 1</li>
					<li>Item 2</li>
				</ul>
			</body>
		</html>
	`

	expected := strings.TrimSpace(`
# Test Heading

This is a **test** paragraph.

- Item 1
- Item 2
	`)

	result, err := ProcessHTMLContent(html, Config{})
	if err != nil {
		t.Fatalf("ProcessHTMLContent() returned error: %v", err)
	}

	if strings.TrimSpace(result) != expected {
		t.Errorf("ProcessHTMLContent() = %q, want %q", result, expected)
	}
}

func TestExtractLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<body>
					<a href="https://example.com/page1">Page 1</a>
					<a href="https://example.com/page2">Page 2</a>
					<a href="https://othersite.com">Other Site</a>
				</body>
			</html>
		`))
	}))
	defer server.Close()

	links, err := ExtractLinks(server.URL)
	if err != nil {
		t.Fatalf("ExtractLinks() returned error: %v", err)
	}

	expectedLinks := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://othersite.com",
	}

	if !reflect.DeepEqual(links, expectedLinks) {
		t.Errorf("ExtractLinks() = %v, want %v", links, expectedLinks)
	}
}
