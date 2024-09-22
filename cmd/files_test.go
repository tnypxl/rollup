package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tnypxl/rollup/internal/config"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"*.go", "file.go", true},
		{"*.go", "file.txt", false},
		{"**/*.go", "dir/file.go", true},
		{"**/*.go", "dir/subdir/file.go", true},
		{"dir/*.go", "dir/file.go", true},
		{"dir/*.go", "otherdir/file.go", false},
		{"**/test_*.go", "internal/test_helper.go", true},
		{"docs/**/*.md", "docs/api/endpoints.md", true},
		{"docs/**/*.md", "src/docs/readme.md", false},
	}

	for _, test := range tests {
		result := matchGlob(test.pattern, test.path)
		if result != test.expected {
			t.Errorf("matchGlob(%q, %q) = %v; want %v", test.pattern, test.path, result, test.expected)
		}
	}
}

func TestIsCodeGenerated(t *testing.T) {
	patterns := []string{"generated_*.go", "**/auto_*.go", "**/*_gen.go"}
	tests := []struct {
		path     string
		expected bool
	}{
		{"generated_file.go", true},
		{"normal_file.go", false},
		{"subdir/auto_file.go", true},
		{"subdir/normal_file.go", false},
		{"pkg/models_gen.go", true},
		{"pkg/handler.go", false},
	}

	for _, test := range tests {
		result := isCodeGenerated(test.path, patterns)
		if result != test.expected {
			t.Errorf("isCodeGenerated(%q, %v) = %v; want %v", test.path, patterns, result, test.expected)
		}
	}
}

func TestIsIgnored(t *testing.T) {
	patterns := []string{"*.tmp", "**/*.log", ".git/**", "vendor/**"}
	tests := []struct {
		path     string
		expected bool
	}{
		{"file.tmp", true},
		{"file.go", false},
		{"subdir/file.log", true},
		{"subdir/file.txt", false},
		{".git/config", true},
		{"src/.git/config", true},
		{"vendor/package/file.go", true},
		{"internal/vendor/file.go", false},
	}

	for _, test := range tests {
		result := isIgnored(test.path, patterns)
		if result != test.expected {
			t.Errorf("isIgnored(%q, %v) = %v; want %v", test.path, patterns, result, test.expected)
		}
	}
}

func TestRunRollup(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "rollup_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test files
	files := map[string]string{
		"file1.go":             "package main\n\nfunc main() {}\n",
		"file2.txt":            "This is a text file.\n",
		"subdir/file3.go":      "package subdir\n\nfunc Func() {}\n",
		"subdir/file4.json":    "{\"key\": \"value\"}\n",
		"generated_model.go":   "// Code generated DO NOT EDIT.\n\npackage model\n",
		"docs/api/readme.md":   "# API Documentation\n",
		".git/config":          "[core]\n\trepositoryformatversion = 0\n",
		"vendor/lib/helper.go": "package lib\n\nfunc Helper() {}\n",
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Set up test configuration
	cfg = &config.Config{
		FileTypes:     []string{"go", "txt", "md"},
		Ignore:        []string{"*.json", ".git/**", "vendor/**"},
		CodeGenerated: []string{"generated_*.go"},
	}

	// Change working directory to the temp directory
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// Run the rollup
	if err := runRollup(); err != nil {
		t.Fatalf("runRollup() failed: %v", err)
	}

	// Check if the output file was created
	outputFiles, err := filepath.Glob("*.rollup.md")
	if err != nil {
		t.Fatalf("Error globbing for output file: %v", err)
	}
	if len(outputFiles) == 0 {
		allFiles, _ := filepath.Glob("*")
		t.Fatalf("No rollup.md file found. Files in directory: %v", allFiles)
	}
	outputFile := outputFiles[0]

	// Read the content of the output file
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check if the content includes the expected files
	expectedContent := []string{
		"# File: file1.go",
		"# File: file2.txt",
		"# File: subdir/file3.go",
		"# File: docs/api/readme.md",
		"# File: generated_model.go (Code-generated, Read-only)",
	}
	for _, expected := range expectedContent {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Output file does not contain expected content: %s", expected)
		}
	}

	// Check if the ignored files are not included
	ignoredContent := []string{
		"file4.json",
		".git/config",
		"vendor/lib/helper.go",
	}
	for _, ignored := range ignoredContent {
		if strings.Contains(string(content), ignored) {
			t.Errorf("Output file contains ignored file: %s", ignored)
		}
	}
}
