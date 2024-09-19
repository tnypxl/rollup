package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/yourproject/internal/config"
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
	}

	for _, test := range tests {
		result := matchGlob(test.pattern, test.path)
		if result != test.expected {
			t.Errorf("matchGlob(%q, %q) = %v; want %v", test.pattern, test.path, result, test.expected)
		}
	}
}

func TestIsCodeGenerated(t *testing.T) {
	patterns := []string{"generated_*.go", "**/auto_*.go"}
	tests := []struct {
		path     string
		expected bool
	}{
		{"generated_file.go", true},
		{"normal_file.go", false},
		{"subdir/auto_file.go", true},
		{"subdir/normal_file.go", false},
	}

	for _, test := range tests {
		result := isCodeGenerated(test.path, patterns)
		if result != test.expected {
			t.Errorf("isCodeGenerated(%q, %v) = %v; want %v", test.path, patterns, result, test.expected)
		}
	}
}

func TestIsIgnored(t *testing.T) {
	patterns := []string{"*.tmp", "**/*.log"}
	tests := []struct {
		path     string
		expected bool
	}{
		{"file.tmp", true},
		{"file.go", false},
		{"subdir/file.log", true},
		{"subdir/file.txt", false},
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
		"file1.go":          "package main\n\nfunc main() {}\n",
		"file2.txt":         "This is a text file.\n",
		"subdir/file3.go":   "package subdir\n\nfunc Func() {}\n",
		"subdir/file4.json": "{\"key\": \"value\"}\n",
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Set up test configuration
	cfg = &Config{
		FileTypes: []string{"go", "txt"},
		Ignore:    []string{"*.json"},
	}
	path = tempDir

	// Run the rollup
	err = runRollup()
	if err != nil {
		t.Fatalf("runRollup() failed: %v", err)
	}

	// Check if the output file was created
	outputFiles, err := filepath.Glob(filepath.Join(tempDir, "*.rollup.md"))
	if err != nil {
		t.Fatalf("Failed to glob output files: %v", err)
	}
	if len(outputFiles) != 1 {
		t.Fatalf("Expected 1 output file, got %d", len(outputFiles))
	}

	// Read the content of the output file
	content, err := os.ReadFile(outputFiles[0])
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check if the content includes the expected files
	expectedContent := []string{
		"# File: file1.go",
		"# File: file2.txt",
		"# File: subdir/file3.go",
	}
	for _, expected := range expectedContent {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Output file does not contain expected content: %s", expected)
		}
	}

	// Check if the ignored file is not included
	if strings.Contains(string(content), "file4.json") {
		t.Errorf("Output file contains ignored file: file4.json")
	}
}
