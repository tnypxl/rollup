# Rollup

Rollup aggregates the contents of text-based files and webpages into markdown files.

## Features

- **File aggregation**: Combine multiple source files into a single markdown document
- **File type filtering**: Include only specific file extensions
- **Ignore patterns**: Exclude files/directories using glob patterns
- **Code-generated file detection**: Mark auto-generated files as read-only in output
- **Web scraping**: Scrape webpage content using Playwright browser automation
- **HTML to Markdown conversion**: Automatically converts scraped HTML to clean markdown
- **CSS selectors**: Extract specific content sections or exclude unwanted elements
- **Path-based overrides**: Configure different selectors for specific URL paths
- **Rate limiting**: Configurable requests per second and burst limits for web scraping
- **Output modes**: Single combined file or separate files per source
- **Verbose logging**: Detailed operation insights for debugging
- **YAML configuration**: Flexible configuration file support

## Installation

Ensure you have Go 1.21+ installed, then run:

```bash
go install github.com/tnypxl/rollup@latest
```

Or build from source:

```bash
git clone https://github.com/tnypxl/rollup.git
cd rollup
go build -o rollup .
```

## Usage

```bash
rollup [command] [flags]
```

### Commands

| Command | Description |
|---------|-------------|
| `files` | Aggregate local files into a single markdown file |
| `web` | Scrape webpages and convert to markdown |
| `generate` | Generate a default rollup.yml config file |

### Flags for `files` command

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--path` | `-p` | `.` | Path to the project directory |
| `--types` | `-t` | `go,md,txt` | Comma-separated list of file extensions (without dots) |
| `--codegen` | `-g` | | Glob patterns for code-generated files |
| `--ignore` | `-i` | | Glob patterns for files to ignore |

### Flags for `web` command

| Flag | Short | Description |
|------|-------|-------------|
| `--urls` | `-u` | URLs of webpages to scrape (comma-separated) |
| `--output` | `-o` | Output type: `single` or `separate` |
| `--css` | | CSS selector to extract specific content |
| `--exclude` | | CSS selectors to exclude (comma-separated) |

### Global flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-f` | Path to config file (default: `rollup.yml`) |
| `--verbose` | `-v` | Enable verbose logging |

## Configuration

Rollup reads from `rollup.yml` by default. Use `--config` to specify a different file.

### Configuration Options

```yaml
# File extensions to include (without leading dots)
file_extensions:
  - go
  - md
  - js

# Glob patterns for paths to ignore
ignore_paths:
  - node_modules/**
  - vendor/**
  - .git/**

# Glob patterns for code-generated files (marked as read-only in output)
code_generated_paths:
  - "**/*.pb.go"
  - "**/generated/**"

# Web scraping site configurations
sites:
  - base_url: https://example.com
    css_locator: .main-content
    exclude_selectors:
      - .ads
      - .navigation
      - footer
    allowed_paths:
      - /docs
      - /blog
    exclude_paths:
      - /admin
    file_name_prefix: example-docs
    path_overrides:
      - path: /special-page
        css_locator: .special-content
        exclude_selectors:
          - .special-ads

# Output type for web scraping: 'single' or 'separate'
output_type: single

# Rate limiting for web requests
requests_per_second: 1.0
burst_limit: 3
```

### Configuration Reference

| Field | Type | Description |
|-------|------|-------------|
| `file_extensions` | list | File extensions to include in file rollup |
| `ignore_paths` | list | Glob patterns for files/directories to skip |
| `code_generated_paths` | list | Glob patterns for auto-generated files |
| `sites` | list | Web scraping target configurations |
| `output_type` | string | `single` (one file) or `separate` (multiple files) |
| `requests_per_second` | float | Rate limit for web requests (default: 1.0) |
| `burst_limit` | int | Maximum burst size for rate limiting (default: 3) |

#### Site Configuration

| Field | Type | Description |
|-------|------|-------------|
| `base_url` | string | Starting URL for scraping (required) |
| `css_locator` | string | CSS selector for content extraction |
| `exclude_selectors` | list | CSS selectors for content to exclude |
| `allowed_paths` | list | URL paths allowed for scraping |
| `exclude_paths` | list | URL paths to skip |
| `file_name_prefix` | string | Prefix for output file names |
| `path_overrides` | list | Path-specific selector overrides |

## Examples

### File Aggregation

```bash
# Rollup files using config file
rollup files

# Specify file types and ignore patterns
rollup files --types=go,js,ts --ignore="vendor/**,*_test.go"

# Rollup a specific directory
rollup files --path=/path/to/project
```

### Web Scraping

```bash
# Scrape URLs from command line
rollup web --urls=https://example.com/docs

# Scrape multiple URLs
rollup web --urls=https://example.com,https://another.com

# Extract specific content with CSS selector
rollup web --urls=https://example.com --css=".article-content"

# Exclude elements from scraped content
rollup web --urls=https://example.com --css=".content" --exclude=".ads,.sidebar"

# Output to separate files
rollup web --urls=https://example.com --output=separate
```

### Configuration Generation

```bash
# Generate rollup.yml based on files in current directory
rollup generate
```

### Using Custom Config

```bash
rollup files --config=my-config.yml
rollup web --config=my-config.yml
```

## Output

### File Rollup Output

The `files` command generates a markdown file named `<project-name>-<timestamp>.rollup.md` containing all matched files:

```markdown
# File: src/main.go

​```go
package main
// ... file contents
​```

# File: docs/README.md (Code-generated, Read-only)

​```md
// ... file contents
​```
```

### Web Rollup Output

The `web` command generates markdown files from scraped content, with filenames based on the page title or URL.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
