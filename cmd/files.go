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

var cfg *config.Config

var (
	path            string
	fileTypes       string
	codeGenPatterns string
	ignorePatterns  string
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Rollup files into a single Markdown file",
	Long: `The files subcommand writes the contents of all files (with target custom file types provided)
in a given project, current path or a custom path, to a single timestamped markdown file
whose name is <project-directory-name>-rollup-<timestamp>.md.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load("rollup.yml") // Assuming the config file is named rollup.yml
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRollup(cfg)
	},
}

func init() {
	filesCmd.Flags().StringVarP(&path, "path", "p", ".", "Path to the project directory")
	filesCmd.Flags().StringVarP(&fileTypes, "types", "t", ".go,.md,.txt", "Comma-separated list of file extensions to include")
	filesCmd.Flags().StringVarP(&codeGenPatterns, "codegen", "g", "", "Comma-separated list of glob patterns for code-generated files")
	filesCmd.Flags().StringVarP(&ignorePatterns, "ignore", "i", "", "Comma-separated list of glob patterns for files to ignore")
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
			// Check if the pattern matches the full path or any part of it
			if matched, _ := filepath.Match(pattern, filePath); matched {
				return true
			}
			pathParts := strings.Split(filePath, string(os.PathSeparator))
			for i := range pathParts {
				partialPath := filepath.Join(pathParts[:i+1]...)
				if matched, _ := filepath.Match(pattern, partialPath); matched {
					return true
				}
			}
		}
	}
	return false
}

func runRollup(cfg *config.Config) error {
	// Use config if available, otherwise use command-line flags
	var types []string
	var codeGenList, ignoreList []string
	if cfg != nil && len(cfg.FileExtensions) > 0 {
		types = cfg.FileExtensions
	} else {
		types = strings.Split(fileTypes, ",")
	}
	if cfg != nil && len(cfg.CodeGeneratedPaths) > 0 {
		codeGenList = cfg.CodeGeneratedPaths
	} else {
		codeGenList = strings.Split(codeGenPatterns, ",")
	}
	if cfg != nil && len(cfg.IgnorePaths) > 0 {
		ignoreList = cfg.IgnorePaths
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
	outputFileName := fmt.Sprintf("%s-%s.rollup.md", projectName, timestamp)

	// Open the output file
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFile.Close()

	startTime := time.Now()
	showProgress := false
	progressTicker := time.NewTicker(500 * time.Millisecond)
	defer progressTicker.Stop()

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
			if verbose {
				fmt.Printf("Ignoring file: %s\n", relPath)
			}
			return nil
		}

		ext := filepath.Ext(path)
		for _, t := range types {
			if ext == "."+t {
				// Verbose logging for processed file
				if verbose {
					size := humanReadableSize(info.Size())
					fmt.Printf("Processing file: %s (%s)\n", relPath, size)
				}

				// Read file contents
				content, err := os.ReadFile(path)
				if err != nil {
					fmt.Printf("Error reading file %s: %v\n", path, err)
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

		if !showProgress && time.Since(startTime) > 5*time.Second {
			showProgress = true
			fmt.Print("This is taking a while (hold tight) ")
		}

		select {
		case <-progressTicker.C:
			if showProgress {
				fmt.Print(".")
			}
		default:
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking through directory: %v", err)
	}

	if showProgress {
		fmt.Println() // Print a newline after the progress dots
	}

	fmt.Printf("Rollup complete. Output file: %s\n", outputFileName)
	return nil
}

func humanReadableSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
