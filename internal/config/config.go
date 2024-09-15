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
	URLs       []URLConfig `yaml:"urls"`
	OutputType string      `yaml:"output_type"`
}

type URLConfig struct {
	URL         string `yaml:"url"`
	CSSLocator  string `yaml:"css_locator"`
	OutputAlias string `yaml:"output_alias"`
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

