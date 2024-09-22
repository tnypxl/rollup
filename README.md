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
- Concurrent processing for improved performance

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
- `--output, -o`: Name of the output file (default: rollup_output.md)
- `--concurrent, -c`: Number of concurrent goroutines for processing (default: 4)

### Flags for `web` command

- `--urls, -u`: URLs of the webpages to scrape (comma-separated)
- `--output, -o`: Output type: 'single' for one file, 'separate' for multiple files (default: single)
- `--depth, -d`: Depth of link traversal (default: 0, only scrape the given URLs)
- `--css`: CSS selector to extract specific content
- `--exclude`: CSS selectors to exclude from the extracted content (comma-separated)
- `--concurrent, -c`: Number of concurrent goroutines for processing (default: 4)

### Global flags

- `--config, -f`: Path to the configuration file (default: rollup.yml in the current directory)
- `--verbose, -v`: Enable verbose logging
- `--timeout, -t`: Timeout for web requests in seconds (default: 30)

## Configuration

Rollup can be configured using a YAML file. By default, it looks for `rollup.yml` in the current directory. You can specify a different configuration file using the `--config` flag.

Example `rollup.yml`:

```yaml
file_types:
  - go
  - md
ignore:
  - node_modules/**
  - vendor/**
  - .git/**
code_generated:
  - **/generated/**
scrape:
  urls:
    - url: https://example.com
      css_locator: .content
      exclude_selectors:
        - .ads
        - .navigation
      output_alias: example
  output_type: single
```

## Examples

1. Rollup files with default configuration:

   ```bash
   rollup files
   ```

2. Web scraping with multiple URLs and increased concurrency:

   ```bash
   rollup web --urls=https://example.com,https://another-example.com --concurrent=8
   ```

3. Generate a default configuration file:

   ```bash
   rollup generate
   ```

4. Use a custom configuration file and specify output:

   ```bash
   rollup files --config=my-config.yml --output=project_summary.md
   ```

5. Web scraping with separate output files and custom timeout:
   ```bash
   rollup web --urls=https://example.com,https://another-example.com --output=separate --timeout=60
   ```

6. Rollup files with specific types and ignore patterns:
   ```bash
   rollup files --types=.go,.md --ignore=vendor/**,*_test.go
   ```

7. Web scraping with depth and CSS selector:
   ```bash
   rollup web --urls=https://example.com --depth=2 --css=.main-content
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
