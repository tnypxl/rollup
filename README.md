# Rollup

Rollup is a powerful CLI tool designed to aggregate and process files based on specified criteria. It's particularly useful for developers and system administrators who need to collect and summarize information from multiple files across a project or system. It now includes advanced web scraping capabilities.

## Features

- File type filtering
- Ignore patterns for excluding files
- Support for code-generated file detection
- Advanced web scraping functionality
- Verbose logging option for detailed output
- Exclusionary CSS selectors for web scraping
- Support for multiple URLs in web scraping
- Configurable output format (single file or separate files)

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

- `rollup`: Run the main rollup functionality
- `rollup web`: Run the web scraping functionality

### Flags for main rollup command

- `--path, -p`: Path to the project directory (default: current directory)
- `--types, -t`: Comma-separated list of file extensions to include (default: .go,.md,.txt)
- `--codegen, -g`: Comma-separated list of glob patterns for code-generated files
- `--ignore, -i`: Comma-separated list of glob patterns for files to ignore
- `--config, -f`: Path to the configuration file (default: rollup.yml in the current directory)
- `--verbose, -v`: Enable verbose logging

### Flags for web scraping command

- `--urls, -u`: URLs of the webpages to scrape (comma-separated)
- `--output, -o`: Output type: 'single' for one file, 'separate' for multiple files (default: single)
- `--depth, -d`: Depth of link traversal (default: 0, only scrape the given URLs)
- `--css`: CSS selector to extract specific content
- `--exclude`: CSS selectors to exclude from the extracted content (comma-separated)

## Configuration

Rollup can be configured using a YAML file. By default, it looks for `rollup.yml` in the current directory. You can specify a different configuration file using the `--config` flag.

Example `rollup.yml`:

```yaml
file_types:
  - go
  - md
ignore:
  - vendor/**
  - **/test/**
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

1. Basic usage with default configuration:

   ```bash
   rollup
   ```

2. Use specific file types and enable verbose logging:

   ```bash
   rollup --types=go,js,py --verbose
   ```

3. Use a custom configuration file:

   ```bash
   rollup --config=my-config.yml
   ```

4. Web scraping with multiple URLs and content exclusion:

   ```bash
   rollup web --urls=https://example.com,https://another-example.com --css=.main-content --exclude=.ads,.sidebar
   ```

5. Web scraping with separate output files:
   ```bash
   rollup web --urls=https://example.com,https://another-example.com --output=separate
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
