package cmd

import (
	"testing"

	"github.com/tnypxl/rollup/internal/config"
)

func TestConvertPathOverrides(t *testing.T) {
	configOverrides := []config.PathOverride{
		{
			Path:             "/blog",
			CSSLocator:       "article",
			ExcludeSelectors: []string{".ads", ".comments"},
		},
		{
			Path:             "/products",
			CSSLocator:       ".product-description",
			ExcludeSelectors: []string{".related-items"},
		},
	}

	scraperOverrides := convertPathOverrides(configOverrides)

	if len(scraperOverrides) != len(configOverrides) {
		t.Errorf("Expected %d overrides, got %d", len(configOverrides), len(scraperOverrides))
	}

	for i, override := range scraperOverrides {
		if override.Path != configOverrides[i].Path {
			t.Errorf("Expected Path %s, got %s", configOverrides[i].Path, override.Path)
		}
		if override.CSSLocator != configOverrides[i].CSSLocator {
			t.Errorf("Expected CSSLocator %s, got %s", configOverrides[i].CSSLocator, override.CSSLocator)
		}
		if len(override.ExcludeSelectors) != len(configOverrides[i].ExcludeSelectors) {
			t.Errorf("Expected %d ExcludeSelectors, got %d", len(configOverrides[i].ExcludeSelectors), len(override.ExcludeSelectors))
		}
		for j, selector := range override.ExcludeSelectors {
			if selector != configOverrides[i].ExcludeSelectors[j] {
				t.Errorf("Expected ExcludeSelector %s, got %s", configOverrides[i].ExcludeSelectors[j], selector)
			}
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "Hello_World"},
		{"file/with/path", "file_with_path"},
		{"file.with.dots", "file_with_dots"},
		{"___leading_underscores___", "leading_underscores"},
		{"", "untitled"},
		{"!@#$%^&*()", "untitled"},
	}

	for _, test := range tests {
		result := sanitizeFilename(test.input)
		if result != test.expected {
			t.Errorf("sanitizeFilename(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestGetFilenameFromContent(t *testing.T) {
	tests := []struct {
		content   string
		url       string
		expected  string
		expectErr bool
	}{
		{"<title>Test Page</title>", "http://example.com", "Test_Page.rollup.md", false},
		{"No title here", "http://example.com/page", "example_com_page.rollup.md", false},
		{"<title>  Trim  Me  </title>", "http://example.com", "Trim_Me.rollup.md", false},
		{"<title></title>", "http://example.com", "example_com.rollup.md", false},
		{"<title>   </title>", "http://example.com", "example_com.rollup.md", false},
		{"Invalid URL", "not a valid url", "", true},
		{"No host", "http://", "", true},
	}

	for _, test := range tests {
		result, err := getFilenameFromContent(test.content, test.url)
		if test.expectErr {
			if err == nil {
				t.Errorf("getFilenameFromContent(%q, %q) expected an error, but got none", test.content, test.url)
			}
		} else {
			if err != nil {
				t.Errorf("getFilenameFromContent(%q, %q) unexpected error: %v", test.content, test.url, err)
			}
			if result != test.expected {
				t.Errorf("getFilenameFromContent(%q, %q) = %q; want %q", test.content, test.url, result, test.expected)
			}
		}
	}
}

// Mock functions for testing
func mockExtractAndConvertContent(urlStr string) (string, error) {
	return "Mocked content for " + urlStr, nil
}
