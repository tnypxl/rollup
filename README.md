# Rollup

Rollup is a powerful CLI tool designed to aggregate and process files based on specified criteria. It's particularly useful for developers and system administrators who need to collect and summarize information from multiple files across a project or system. It now includes advanced web scraping capabilities.

## Features

- File type filtering
- Ignore patterns for excluding files
- Support for code-generated file detection
- Advanced web scraping functionality
- Verbose logging option for detailed output
- Exclusionary CSS paths for web scraping

## Installation

To install Rollup, make sure you have Go installed on your system, then run:

```bash
go get github.com/tnypxl/rollup
```

## Usage

Basic usage:

```bash
rollup [flags]
```

### Flags

- `--file-types`: Comma-separated list of file types to include (default: all files)
- `--ignore`: Comma-separated list of patterns to ignore
- `--code-generated`: Comma-separated list of patterns for code-generated files
- `--verbose, -v`: Enable verbose logging
- `--config`: Path to the configuration file (default: rollup.yml)
- `--url`: URL to scrape (for web scraping functionality)
- `--css`: CSS selector for content extraction (for web scraping)
- `--exclude-css`: CSS selector for content to exclude (for web scraping)

## Configuration

Rollup can be configured using a YAML file. By default, it looks for `rollup.yml` in the current directory. You can specify a different configuration file using the `--config` flag.

Example `rollup.yml`:

```yaml
file_types:
  - .go
  - .md
ignore:
  - vendor/**
  - **/test/**
code_generated:
  - **/generated/**
scrape:
  url: https://example.com
  css_locator: .content
  exclude_selectors:
    - .ads
    - .navigation
```

## Examples

1. Basic usage with default configuration:
   ```bash
   rollup
   ```

2. Use specific file types and enable verbose logging:
   ```bash
   rollup --file-types=.go,.js,.py --verbose
   ```

3. Use a custom configuration file:
   ```bash
   rollup --config=my-config.yml
   ```

4. Web scraping with content exclusion:
   ```bash
   rollup --url=https://example.com --css=.main-content --exclude-css=.ads,.sidebar
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
