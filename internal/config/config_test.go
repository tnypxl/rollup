package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := []byte(`
file_types:
  - go
  - md
ignore:
  - "*.tmp"
  - "**/*.log"
code_generated:
  - "generated_*.go"
scrape:
  sites:
    - base_url: "https://example.com"
      css_locator: "main"
      exclude_selectors:
        - ".ads"
      max_depth: 2
      allowed_paths:
        - "/blog"
      exclude_paths:
        - "/admin"
      output_alias: "example"
      path_overrides:
        - path: "/special"
          css_locator: ".special-content"
          exclude_selectors:
            - ".sidebar"
  output_type: "single"
  requests_per_second: 1.0
  burst_limit: 5
`)

	tmpfile, err := os.CreateTemp("", "config*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	config, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check if the loaded config matches the expected values
	expectedConfig := &Config{
		FileTypes:     []string{"go", "md"},
		Ignore:        []string{"*.tmp", "**/*.log"},
		CodeGenerated: []string{"generated_*.go"},
		Scrape: ScrapeConfig{
			Sites: []SiteConfig{
				{
					BaseURL:          "https://example.com",
					CSSLocator:       "main",
					ExcludeSelectors: []string{".ads"},
					MaxDepth:         2,
					AllowedPaths:     []string{"/blog"},
					ExcludePaths:     []string{"/admin"},
					OutputAlias:      "example",
					PathOverrides: []PathOverride{
						{
							Path:             "/special",
							CSSLocator:       ".special-content",
							ExcludeSelectors: []string{".sidebar"},
						},
					},
				},
			},
			OutputType:        "single",
			RequestsPerSecond: 1.0,
			BurstLimit:        5,
		},
	}

	if !reflect.DeepEqual(config, expectedConfig) {
		t.Errorf("Loaded config does not match expected config.\nGot: %+v\nWant: %+v", config, expectedConfig)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	expected := "rollup.yml"
	result := DefaultConfigPath()
	if result != expected {
		t.Errorf("DefaultConfigPath() = %q, want %q", result, expected)
	}
}

func TestFileExists(t *testing.T) {
	// Test with an existing file
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if !FileExists(tmpfile.Name()) {
		t.Errorf("FileExists(%q) = false, want true", tmpfile.Name())
	}

	// Test with a non-existing file
	if FileExists("non_existing_file.txt") {
		t.Errorf("FileExists(\"non_existing_file.txt\") = true, want false")
	}
}
