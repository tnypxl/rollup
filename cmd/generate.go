package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
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

	config := config.Config{
		FileTypes: make([]string, 0, len(fileTypes)),
		Ignore:    []string{"node_modules/**", "vendor/**", ".git/**"},
	}

	for ext := range fileTypes {
		config.FileTypes = append(config.FileTypes, ext)
	}

	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	err = ioutil.WriteFile("rollup.yml", yamlData, 0644)
	if err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	fmt.Println("Generated rollup.yml file successfully.")
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
