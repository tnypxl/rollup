package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	FileTypes     []string     `yaml:"file_types"`
	Ignore        []string     `yaml:"ignore"`
	CodeGenerated []string     `yaml:"code_generated"`
	Scrape        ScrapeConfig `yaml:"scrape"`
}

type ScrapeConfig struct {
    Sites             []SiteConfig `yaml:"sites"`
    OutputType        string       `yaml:"output_type"`
    RequestsPerSecond float64      `yaml:"requests_per_second"`
    BurstLimit        int          `yaml:"burst_limit"`
}

type SiteConfig struct {
    BaseURL          string            `yaml:"base_url"`
    CSSLocator       string            `yaml:"css_locator"`
    ExcludeSelectors []string          `yaml:"exclude_selectors"`
    MaxDepth         int               `yaml:"max_depth"`
    AllowedPaths     []string          `yaml:"allowed_paths"`
    ExcludePaths     []string          `yaml:"exclude_paths"`
    OutputAlias      string            `yaml:"output_alias"`
    PathOverrides    []PathOverride    `yaml:"path_overrides"`
}

type PathOverride struct {
    Path             string   `yaml:"path"`
    CSSLocator       string   `yaml:"css_locator"`
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

	return &config, nil
}

func DefaultConfigPath() string {
	return "rollup.yml"
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

