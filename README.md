# Alldebrid CLI

A command-line interface for interacting with the Alldebrid API to unrestrict links, download magnets, and process torrent files.

## Features

- Unrestrict HTTP/HTTPS links
- Process magnet links
- Upload and process torrent files
- Filter files by size
- Multiple output formats (plain text, HTML, JSON)
- Configurable timeout for download operations
- Debug mode for troubleshooting

## Installation

```bash
go build -o alldebrid-cli
```

## Usage

```bash
./alldebrid-cli [options] <input>
```

Where `<input>` can be:
- An HTTP/HTTPS URL to unrestrict
- A magnet link (starting with `magnet:`)
- A path to a `.torrent` file

## Configuration

### API Token

You need an Alldebrid API token to use this tool. You can provide it in two ways:

1. Set the `ALLDEBRID_TOKEN` environment variable:
   ```bash
   export ALLDEBRID_TOKEN="your_token_here"
   ```

2. Use the `-token` flag:
   ```bash
   ./alldebrid-cli -token "your_token_here" <input>
   ```

## Command Line Options

| Flag                            | Type     | Default             | Description                              |
|---------------------------------|----------|---------------------|------------------------------------------|
| `-token`                        | string   | `$ALLDEBRID_TOKEN`  | Alldebrid API Token                      |
| `-timeout`                      | duration | `10m`               | Timeout for waiting for download links   |
| `-ignore-files-smaller-than-mb` | float    | `5.0`               | Ignore files smaller than this size in MB |
| `-html`                         | bool     | `false`             | Print links as HTML                      |
| `-json`                         | bool     | `false`             | Print links as JSON                      |
| `-debug`                        | bool     | `false`             | Enable debug mode                        |
| `-version`                      | bool     | `false`             | Print version information and exit       |

### Timeout Flag

The `-timeout` flag controls how long the CLI will wait for download links to become available when processing magnet links or torrent files. This is particularly important for large torrents that may take time to be processed by Alldebrid's servers.

**Format**: Go duration format (e.g., `30s`, `5m`, `1h30m`)

**Default**: 10 minutes

## Examples

### Unrestrict a URL
```bash
./alldebrid-cli "https://example.com/file.zip"
```

### Process a magnet link with custom timeout
```bash
./alldebrid-cli -timeout 30m "magnet:?xt=urn:btih:..."
```

### Upload a torrent file and filter small files
```bash
./alldebrid-cli -ignore-files-smaller-than-mb 100 "/path/to/file.torrent"
```

### Output as JSON with debug information
```bash
./alldebrid-cli -json -debug "magnet:?xt=urn:btih:..."
```

### Output as HTML
```bash
./alldebrid-cli -html "https://example.com/file.zip"
```

## Environment Variables

- `ALLDEBRID_TOKEN` - Your Alldebrid API token

## Output Formats

### Plain Text (Default)
Simple list of download URLs, one per line.

### HTML (`-html`)
Formatted HTML with a table and clickable links.

### JSON (`-json`)
Structured JSON output with file information including names, sizes, and download URLs.
