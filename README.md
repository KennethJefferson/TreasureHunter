# TreasureHunter - ArchiveDownloader

A fast, concurrent Go application that scans directories for files containing URLs (.url, .md, .html, .txt) and automatically downloads the linked files.

## Features

- **Format-Aware URL Extraction**: Intelligently parses multiple file formats
  - `.url` files: Windows Internet Shortcut format
  - `.md` files: Markdown links and plain URLs
  - `.html` files: href and src attributes
  - `.txt` files: Plain HTTP/HTTPS URLs

- **Concurrent Downloads**: Worker pool pattern for parallel downloading
  - Configurable worker count for optimal performance
  - Thread-safe statistics tracking
  - Real-time progress output

- **Batch Processing**: Efficient handling of large directory trees
  - Processes subdirectories in batches of `ceil(N/10)`
  - Violet progress bar for visual feedback
  - Recursive scanning with unlimited depth

- **Smart Download Management**:
  - Skip existing files (no re-download)
  - Automatic filename generation from URLs or Content-Disposition headers
  - 5-minute HTTP timeout to prevent hanging
  - Cleanup of partial files on errors

- **Completion Notification**: Optional audio chime when processing finishes

## Installation

### From Source

Requires Go 1.19 or later:

```bash
cd ArchiveDownloader
go build -o ArchiveDownloader.exe
```

### Pre-built Binary

The compiled `ArchiveDownloader.exe` is ready to use.

## Quick Start

### Basic Usage

Scan a single directory (root only) with 4 workers:

```bash
ArchiveDownloader.exe -workers 4 -scan C:\Downloads
```

### Recursive Scanning

Scan directory and all subdirectories with 8 workers:

```bash
ArchiveDownloader.exe -workers 8 -scan C:\Projects --recursive
```

### Multiple Directories

Scan multiple directories at once:

```bash
ArchiveDownloader.exe -workers 6 -scan C:\Downloads -scan D:\Archives --recursive
```

## Usage

### Command-Line Options

```
ArchiveDownloader.exe -workers <num> -scan <dir1> [-scan <dir2>...] [--recursive]
```

**Required Arguments**:
- `-workers <num>`: Number of concurrent download workers
- `-scan <directory>`: Directory to scan (can be repeated for multiple directories)

**Optional Arguments**:
- `--recursive`: Scan subdirectories with unlimited depth (default: root only)

### Examples

**Scan downloads folder (root only)**:
```bash
ArchiveDownloader.exe -workers 4 -scan C:\Downloads
```

**Scan project directory recursively**:
```bash
ArchiveDownloader.exe -workers 8 -scan C:\MyProjects --recursive
```

**Scan multiple locations with batch processing**:
```bash
ArchiveDownloader.exe -workers 10 -scan D:\Archives -scan E:\Backups --recursive
```

## Configuration

### config.json

Optional configuration file for completion notification:

```json
{
  "completion_chime": "C:\\Windows\\Media\\Windows Notify System Generic.wav"
}
```

Place `config.json` in the same directory as `ArchiveDownloader.exe`.

**Supported Audio Formats**:
- Windows: `.wav` files (via PowerShell Media.SoundPlayer)
- macOS: Any format supported by `afplay`
- Linux: `.wav` files (via `paplay` or `aplay`)

## How It Works

### File Processing

1. **Scanner** finds supported files (.url, .md, .html, .txt)
2. **Parser** extracts URLs using format-specific logic
3. **Workers** download files concurrently
4. **Collector** aggregates statistics

### Batch Processing

When scanning recursively, subdirectories are processed in batches:

**Example**: Directory with 80 subdirectories
- Batch size: `ceil(80 / 10)` = 8
- Processes 10 batches of 8 directories each
- Violet progress bar shows batch progress

```
Found 80 subdirectories in C:\Projects
Processing in batches of 8 directories

Processing directories │████████████░░░░│ 40/80 (50%)
```

### Download Location

Files are downloaded to the **same directory** as the source file containing the URL.

**Example**:
- URL found in: `C:\Downloads\links.md`
- File downloaded to: `C:\Downloads\archive.zip`

### URL Validation

The application filters out:
- Non-HTTP/HTTPS URLs
- Example/placeholder URLs (example.com, example.org)
- Localhost URLs (localhost, 127.0.0.1)

## Output

### Real-time Progress

```
[Worker 1] Found 3 URL(s) in test.md
[Worker 1] ✓ Downloaded: go1.21.0.windows-amd64.zip (70.2 MB)
[Worker 2] ⏭ Skipped: existing-file.zip (already exists)
[Worker 1] ✗ Failed: https://invalid.url/file.zip - HTTP 404: 404 Not Found
```

### Summary Statistics

```
Summary
=======
Files scanned: 120
URLs found: 45
Downloads succeeded: 38
Downloads skipped: 5
Downloads failed: 2
```

## Performance Tips

### Optimal Worker Count

- **Local/fast network**: 8-16 workers
- **Remote/slow network**: 4-8 workers
- **Very slow network**: 2-4 workers

Higher worker counts may not improve performance due to bandwidth limitations.

### Large Directory Trees

When scanning directories with hundreds of subdirectories:
- Use `--recursive` flag for automatic batch processing
- Monitor network bandwidth to avoid saturation
- Consider splitting into multiple runs for better control

## Troubleshooting

### "Error: at least one -scan directory must be specified"

You must provide at least one directory to scan:

```bash
ArchiveDownloader.exe -workers 4 -scan C:\Downloads
```

### "Error: -workers flag is required"

You must specify the number of workers:

```bash
ArchiveDownloader.exe -workers 4 -scan C:\Downloads
```

### Downloads timing out

Increase the timeout in `downloader.go` (default: 5 minutes):

```go
HTTPClient = &http.Client{
    Timeout: 10 * time.Minute,  // Increase to 10 minutes
}
```

### Completion chime not playing

1. Check that `config.json` exists in the same directory as the executable
2. Verify the audio file path is correct
3. Ensure the audio file exists and is a valid format

### Progress bar not showing colors

Windows 10+ required for ANSI color support. On older systems, colors may appear as escape codes.

## File Format Support

### .url Files (Windows Internet Shortcuts)

```ini
[InternetShortcut]
URL=https://example.com/archive.zip
```

### .md Files (Markdown)

```markdown
Download: [Archive](https://example.com/archive.zip)

Or plain URL: https://example.com/another-file.zip
```

### .html Files

```html
<a href="https://example.com/archive.zip">Download</a>
<img src="https://example.com/image.jpg">
```

### .txt Files

```
Check out this file: https://example.com/archive.zip

More text here...
```

## Technical Details

- **Language**: Go 1.19+
- **Concurrency**: Worker pool pattern with buffered channels
- **Statistics**: Atomic operations for thread-safety
- **HTTP Client**: 5-minute timeout per request
- **Dependencies**:
  - `github.com/schollz/progressbar/v3` - Progress bar
  - `github.com/k0kubun/go-ansi` - ANSI color support

## License

This is a standalone utility tool. See repository root for license information.

## Related Tools

Part of the TreasureHunter toolkit:
- **GoGithubRepoFetch**: Download GitHub repositories from .url/.md files
- **LemonFoxWhisperTranscriber**: Concurrent audio transcription
- **WhisperBatchTextScrubber**: Process Whisper API batch output

## Support

For issues, questions, or contributions, please see the main TreasureHunter repository.
