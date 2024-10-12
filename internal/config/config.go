package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	// FileExtensions is a list of file extensions to include in the rollup
	FileExtensions []string `yaml:"file_extensions"`

	// IgnorePaths is a list of glob patterns for paths to ignore
	IgnorePaths []string `yaml:"ignore_paths"`

	// CodeGeneratedPaths is a list of glob patterns for code-generated files
	CodeGeneratedPaths []string `yaml:"code_generated_paths"`

	// Sites is a list of site configurations for web scraping
	Sites []SiteConfig `yaml:"sites"`

	// OutputType specifies how the output should be generated
	OutputType string `yaml:"output_type"`

	// RequestsPerSecond limits the rate of web requests
	RequestsPerSecond *float64 `yaml:"requests_per_second,omitempty"`

	// BurstLimit sets the maximum burst size for rate limiting
	BurstLimit *int `yaml:"burst_limit,omitempty"`
}

type SiteConfig struct {
	// BaseURL is the starting point for scraping this site
	BaseURL string `yaml:"base_url"`

	// CSSLocator is used to extract specific content
	CSSLocator string `yaml:"css_locator"`

	// ExcludeSelectors lists CSS selectors for content to exclude
	ExcludeSelectors []string `yaml:"exclude_selectors"`

	// MaxDepth sets the maximum depth for link traversal
	MaxDepth int `yaml:"max_depth"`

	// AllowedPaths lists paths that are allowed to be scraped
	AllowedPaths []string `yaml:"allowed_paths"`

	// ExcludePaths lists paths that should not be scraped
	ExcludePaths []string `yaml:"exclude_paths"`

	// OutputAlias provides an alternative name for output files
	OutputAlias string `yaml:"output_alias"`

	// PathOverrides allows for path-specific configurations
	PathOverrides []PathOverride `yaml:"path_overrides"`
}

type PathOverride struct {
	// Path is the URL path this override applies to
	Path string `yaml:"path"`

	// CSSLocator overrides the site-wide CSS locator for this path
	CSSLocator string `yaml:"css_locator"`

	// ExcludeSelectors overrides the site-wide exclude selectors for this path
	ExcludeSelectors []string `yaml:"exclude_selectors"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if len(c.FileExtensions) == 0 {
		return fmt.Errorf("at least one file extension must be specified")
	}

	if c.RequestsPerSecond != nil && *c.RequestsPerSecond <= 0 {
		return fmt.Errorf("requests_per_second must be positive")
	}

	if c.BurstLimit != nil && *c.BurstLimit <= 0 {
		return fmt.Errorf("burst_limit must be positive")
	}

	for _, site := range c.Sites {
		if site.BaseURL == "" {
			return fmt.Errorf("base_url must be specified for each site")
		}
		if site.MaxDepth < 0 {
			return fmt.Errorf("max_depth must be non-negative")
		}
	}

	return nil
}
