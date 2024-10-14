package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/config"
	"gopkg.in/yaml.v2"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a rollup.yml config file",
	Long:  `Scan the current directory for text and code files and generate a rollup.yml config file based on the found file extensions.`,
	RunE:  runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	fileTypes := make(map[string]bool)
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if isTextFile(ext) {
				fileTypes[ext] = true
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking the path: %v", err)
	}

	cfg := config.Config{
		FileExtensions: make([]string, 0, len(fileTypes)),
		IgnorePaths:    []string{"node_modules/**", "vendor/**", ".git/**"},
	}

	for ext := range fileTypes {
		cfg.FileExtensions = append(cfg.FileExtensions, ext)
	}

	// Sort file types for consistency
	sort.Strings(cfg.FileExtensions)

	yamlData, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	outputPath := "rollup.yml"
	err = os.WriteFile(outputPath, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	fmt.Printf("Generated %s file successfully.\n", outputPath)
	return nil
}

func isTextFile(ext string) bool {
	textExtensions := map[string]bool{
		"txt": true, "md": true, "go": true, "py": true, "js": true, "html": true, "css": true,
		"json": true, "xml": true, "yaml": true, "yml": true, "toml": true, "ini": true,
		"sh": true, "bash": true, "zsh": true, "fish": true,
		"c": true, "cpp": true, "h": true, "hpp": true, "java": true, "kt": true, "scala": true,
		"rs": true, "rb": true, "php": true, "ts": true, "swift": true,
	}
	return textExtensions[ext]
}

func init() {
	// Add any flags for the generate command here if needed
}
