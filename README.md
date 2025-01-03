# Rollup

Rollup aggregates the contents of text-based files and webpages into a markdown file.

## Features

- File type filtering for targeted content aggregation
- Ignore patterns for excluding specific files or directories
- Support for code-generated file detection and exclusion
- Advanced web scraping functionality with depth control
- Verbose logging option for detailed operation insights
- Exclusionary CSS selectors for precise web content extraction
- Support for multiple URLs in web scraping operations
- Configurable output format for web scraping (single file or separate files)
- Flexible configuration file support (YAML)
- Automatic generation of default configuration file
- Custom output file naming
- Rate limiting for web scraping to respect server resources

## Installation

To install Rollup, make sure you have Go installed on your system, then run:

```bash
go get github.com/tnypxl/rollup
```

## Usage

Basic usage:

```bash
rollup [command] [flags]
```

### Commands

- `rollup files`: Rollup files into a single Markdown file
- `rollup web`: Scrape main content from webpages and convert to Markdown
- `rollup generate`: Generate a rollup.yml config file

### Flags for `files` command

- `--path, -p`: Path to the project directory (default: current directory)
- `--types, -t`: Comma-separated list of file extensions to include (default: .go,.md,.txt)
- `--codegen, -g`: Comma-separated list of glob patterns for code-generated files
- `--ignore, -i`: Comma-separated list of glob patterns for files to ignore

### Flags for `web` command

- `--urls, -u`: URLs of the webpages to scrape (comma-separated)
- `--output, -o`: Output type: 'single' for one file, 'separate' for multiple files (default: single)
- `--depth, -d`: Depth of link traversal (default: 0, only scrape the given URLs)
- `--css`: CSS selector to extract specific content
- `--exclude`: CSS selectors to exclude from the extracted content (comma-separated)

### Global flags

- `--config, -f`: Path to the configuration file (default: rollup.yml in the current directory)
- `--verbose, -v`: Enable verbose logging

## Configuration

Rollup can be configured using a YAML file. By default, it looks for `rollup.yml` in the current directory. You can specify a different configuration file using the `--config` flag.

Example `rollup.yml`:

```yaml
file_extensions:
  - go
  - md
ignore_paths:
  - node_modules/**
  - vendor/**
  - .git/**
code_generated_paths:
  - **/generated/**
sites:
  - base_url: https://example.com
    css_locator: .content
    exclude_selectors:
      - .ads
      - .navigation
    max_depth: 2
    allowed_paths:
      - /blog
      - /docs
    exclude_paths:
      - /admin
    output_alias: example
    path_overrides:
      - path: /special-page
        css_locator: .special-content
        exclude_selectors:
          - .special-ads
output_type: single
requests_per_second: 1.0
burst_limit: 3
```

## Examples

1. Rollup files with default configuration:

   ```bash
   rollup files
   ```

2. Web scraping with multiple URLs:

   ```bash
   rollup web --urls=https://example.com,https://another-example.com
   ```

3. Generate a default configuration file:

   ```bash
   rollup generate
   ```

4. Use a custom configuration file:

   ```bash
   rollup files --config=my-config.yml
   ```

5. Web scraping with separate output files:

   ```bash
   rollup web --urls=https://example.com,https://another-example.com --output=separate
   ```

6. Rollup files with specific types and ignore patterns:

   ```bash
   rollup files --types=go,md --ignore=vendor/**,*_test.go
   ```

7. Web scraping with depth and CSS selector:
   ```bash
   rollup web --urls=https://example.com --depth=2 --css=.main-content
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
