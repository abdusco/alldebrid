# Alldebrid CLI

A command-line tool for downloading files through the Alldebrid service. Supports magnet links, HTTP/HTTPS URLs, and torrent files.

## Features

- Download files from magnet links, direct URLs, and torrent files
- Filter files by minimum size
- Output download links as plain text or HTML
- Automatic retry and timeout handling
- Progress monitoring for torrent/magnet processing

## Installation

```bash
go build -o alldebrid-cli
```

## Usage

```bash
# Set your API token (or use -token flag)
export ALLDEBRID_TOKEN=your_api_token_here

# Download from magnet link
./alldebrid-cli "magnet:?xt=urn:btih:..."

# Download from HTTP/HTTPS URL
./alldebrid-cli "https://file.host/file.zip"

# Download from torrent file
./alldebrid-cli "/path/to/file.torrent"
```

## Options

- `-token`: Alldebrid API token (can also use ALLDEBRID_TOKEN environment variable)
- `-html`: Output download links as HTML table instead of plain text
- `-ignore-files-smaller-than-mb`: Filter out files smaller than specified size in MB (default: 5.0)

## Examples

```bash
# Download large files only (10MB+) and output as HTML
./alldebrid-cli -ignore-files-smaller-than-mb 10 -html "magnet:?xt=urn:btih:..."

# Use custom API token
./alldebrid-cli -token "your_token" "https://example.com/file.zip"
```

## Requirements

- Go 1.24+
- Valid Alldebrid API token

## License

This project is provided as-is for personal use.
