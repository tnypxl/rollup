# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Build the binary
go build -o rollup .

# Run directly
go run main.go [command]

# Run tests
go test ./...

# Run a single test
go test -run TestName ./path/to/package
```

## Project Overview

Rollup is a Go CLI tool that aggregates text-based files and webpages into markdown files. It has three main commands:
- `files` - Rolls up local files into a single markdown file
- `web` - Scrapes webpages and converts to markdown using Playwright
- `generate` - Creates a default rollup.yml config file

## Architecture

**Entry Point**: `main.go` initializes Playwright browser and loads config before executing commands via Cobra.

**Command Layer** (`cmd/`):
- `root.go` - Cobra root command with global flags (--config, --verbose)
- `files.go` - File aggregation with glob pattern matching for ignore/codegen detection
- `web.go` - Web scraping orchestration, converts config site definitions to scraper configs
- `generate.go` - Scans directory for text file types and generates rollup.yml

**Internal Packages**:
- `internal/config` - YAML config loading and validation. Defines `Config`, `SiteConfig`, `PathOverride` structs
- `internal/scraper` - Playwright-based web scraping with rate limiting, HTML-to-markdown conversion via goquery and html-to-markdown library

**Key Dependencies**:
- `spf13/cobra` - CLI framework
- `playwright-go` - Browser automation for web scraping
- `PuerkitoBio/goquery` - HTML parsing and CSS selector extraction
- `JohannesKaufmann/html-to-markdown` - HTML to markdown conversion

## Configuration

The tool reads from `rollup.yml` by default. Key config fields:
- `file_extensions` - File types to include in rollup
- `ignore_paths` / `code_generated_paths` - Glob patterns for exclusion
- `sites` - Web scraping targets with CSS selectors, path filtering, rate limiting
