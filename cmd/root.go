package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tnypxl/rollup/internal/config"
)

var (
	path            string
	fileTypes       string
	codeGenPatterns string
	ignorePatterns  string
	configFile      string
	cfg             *config.Config
	verbose         bool
)

var rootCmd = &cobra.Command{
	Use:   "rollup",
	Short: "Rollup files into a single Markdown file",
	Long: `Rollup is a tool that writes the contents of all files (with target custom file types provided)
in a given project, current path or a custom path, to a single timestamped markdown file
whose name is <project-directory-name>-rollup-<timestamp>.md.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if configFile == "" {
			defaultConfig := config.DefaultConfigPath()
			if config.FileExists(defaultConfig) {
				configFile = defaultConfig
			}
		}

		if configFile != "" {
			var err error
			cfg, err = config.Load(configFile)
			if err != nil {
				return fmt.Errorf("error loading config file: %v", err)
			}
		}
		return runRollup()
	},
}

func Execute(config *config.Config) error {
	cfg = config
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVarP(&path, "path", "p", ".", "Path to the project directory")
	rootCmd.Flags().StringVarP(&fileTypes, "types", "t", ".go,.md,.txt", "Comma-separated list of file extensions to include")
	rootCmd.Flags().StringVarP(&codeGenPatterns, "codegen", "g", "", "Comma-separated list of glob patterns for code-generated files")
	rootCmd.Flags().StringVarP(&ignorePatterns, "ignore", "i", "", "Comma-separated list of glob patterns for files to ignore")
	rootCmd.Flags().StringVarP(&configFile, "config", "f", "", "Path to the config file (default: rollup.yml in the current directory)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}

func matchGlob(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	return matchGlobRecursive(parts, path)
}

func matchGlobRecursive(patternParts []string, path string) bool {
	if len(patternParts) == 0 {
		return path == ""
	}

	if patternParts[0] == "**" {
		for i := 0; i <= len(path); i++ {
			if matchGlobRecursive(patternParts[1:], path[i:]) {
				return true
			}
		}
		return false
	}

	i := strings.IndexByte(path, '/')
	if i < 0 {
		matched, _ := filepath.Match(patternParts[0], path)
		return matched && len(patternParts) == 1
	}

	matched, _ := filepath.Match(patternParts[0], path[:i])
	return matched && matchGlobRecursive(patternParts[1:], path[i+1:])
}

func isCodeGenerated(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(pattern, "**") {
			if matchGlob(pattern, filePath) {
				return true
			}
		} else {
			matched, err := filepath.Match(pattern, filepath.Base(filePath))
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

func isIgnored(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(pattern, "**") {
			if matchGlob(pattern, filePath) {
				return true
			}
		} else {
			matched, err := filepath.Match(pattern, filepath.Base(filePath))
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

func runRollup() error {
	// Use config if available, otherwise use command-line flags
	var types, codeGenList, ignoreList []string
	if cfg != nil && len(cfg.FileTypes) > 0 {
		types = cfg.FileTypes
	} else {
		types = strings.Split(fileTypes, ",")
	}
	if cfg != nil && len(cfg.CodeGenerated) > 0 {
		codeGenList = cfg.CodeGenerated
	} else {
		codeGenList = strings.Split(codeGenPatterns, ",")
	}
	if cfg != nil && cfg.Ignore != nil && len(cfg.Ignore) > 0 {
		ignoreList = cfg.Ignore
	} else {
		ignoreList = strings.Split(ignorePatterns, ",")
	}

	// Get the absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %v", err)
	}

	// Get the project directory name
	projectName := filepath.Base(absPath)

	// Generate the output file name
	timestamp := time.Now().Format("20060102-150405")
	outputFileName := fmt.Sprintf("%s-rollup-%s.md", projectName, timestamp)

	// Open the output file
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFile.Close()

	// Walk through the directory
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, _ := filepath.Rel(absPath, path)

		// Check if the file should be ignored
		if isIgnored(relPath, ignoreList) {
			return nil
		}

		ext := filepath.Ext(path)
		for _, t := range types {
			if ext == "."+t {
				// Read file contents
				content, err := os.ReadFile(path)
				if err != nil {
					fmt.Printf("Error reading file %s: %v", path, err)
					return nil
				}

				// Check if the file is code-generated
				isCodeGen := isCodeGenerated(relPath, codeGenList)
				codeGenNote := ""
				if isCodeGen {
					codeGenNote = " (Code-generated, Read-only)"
				}

				// Write file name and contents to the output file
				fmt.Fprintf(outputFile, "# File: %s%s\n\n```%s\n%s```\n\n", relPath, codeGenNote, t, string(content))
				break
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking through directory: %v", err)
	}

	fmt.Printf("Rollup complete. Output file: %s", outputFileName)
	return nil
}
