package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := []byte(`
file_extensions:
  - .go
  - .md
ignore_paths:
  - "*.tmp"
  - "**/*.log"
code_generated_paths:
  - "generated_*.go"
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

	if _, err = tmpfile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	config, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check if the loaded config matches the expected values
	rps := 1.0
	bl := 5
	expectedConfig := &Config{
		FileExtensions:     []string{".go", ".md"},
		IgnorePaths:        []string{"*.tmp", "**/*.log"},
		CodeGeneratedPaths: []string{"generated_*.go"},
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
		RequestsPerSecond: &rps,
		BurstLimit:        &bl,
	}

	if !reflect.DeepEqual(config, expectedConfig) {
		t.Errorf("Loaded config does not match expected config.\nGot: %+v\nWant: %+v", config, expectedConfig)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: Config{
				FileExtensions: []string{".go"},
				Sites: []SiteConfig{
					{BaseURL: "https://example.com", MaxDepth: 2},
				},
			},
			wantErr: false,
		},
		{
			name:    "No file extensions",
			config:  Config{},
			wantErr: true,
		},
		{
			name: "Invalid requests per second",
			config: Config{
				FileExtensions:    []string{".go"},
				RequestsPerSecond: func() *float64 { f := -1.0; return &f }(),
			},
			wantErr: true,
		},
		{
			name: "Invalid burst limit",
			config: Config{
				FileExtensions: []string{".go"},
				BurstLimit:     func() *int { i := -1; return &i }(),
			},
			wantErr: true,
		},
		{
			name: "Site without base URL",
			config: Config{
				FileExtensions: []string{".go"},
				Sites:          []SiteConfig{{}},
			},
			wantErr: true,
		},
		{
			name: "Negative max depth",
			config: Config{
				FileExtensions: []string{".go"},
				Sites:          []SiteConfig{{BaseURL: "https://example.com", MaxDepth: -1}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
