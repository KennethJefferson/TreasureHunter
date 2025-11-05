# ArchiveDownloader

A concurrent Go application that scans directories for files containing URLs (.url, .md, .html, .txt) and downloads the linked files to the same location as the source file.

## Architecture

### File Structure
```
ArchiveDownloader/
├── main.go          # CLI parsing & orchestration
├── scanner.go       # Directory traversal with batch processing
├── parser.go        # Format-aware URL extraction
├── downloader.go    # HTTP download with timeout & cleanup
├── worker.go        # Worker pool & result collection
├── config.json      # Completion chime configuration
├── go.mod           # Go module definition
└── CLAUDE.md        # This file
```

### Data Flow

```
Scanner → jobs channel → Workers (N) → results channel → Collector
         (buffered: 100)                (buffered: 100)
```

**Detailed Flow**:
1. Scanner finds `.url`, `.md`, `.html`, `.txt` files → sends to jobs channel
2. Workers consume jobs → extract URLs → download → send results
3. Collector aggregates statistics using atomic operations
4. Completion chime plays when all processing is done

## Command-Line Interface

### Usage
```bash
ArchiveDownloader.exe -workers <num> -scan <dir1> [-scan <dir2>...] [--recursive]
```

### Arguments
- `-workers <num>` (required): Number of concurrent download workers
- `-scan <directory>` (required, repeatable): Root directories to scan for files
- `--recursive` (optional): Scan subdirectories with unlimited depth

### Examples
```bash
# Scan single directory (root only) with 4 workers
ArchiveDownloader.exe -workers 4 -scan C:\Downloads

# Scan multiple directories recursively with 8 workers
ArchiveDownloader.exe -workers 8 -scan C:\Downloads -scan D:\Archives --recursive

# Scan with batch processing (80 subdirectories = 8 per batch)
ArchiveDownloader.exe -workers 6 -scan C:\Projects --recursive
```

## Key Features

### 1. Batch Processing with Progress Bar

When `--recursive` is specified, the application processes subdirectories in batches:

**Batch Size Calculation**: `ceil(totalSubdirectories / 10)`

**Example**: 80 subdirectories
- Batch size = ceil(80 / 10) = 8
- Process 10 batches of 8 directories each
- Violet progress bar shows batch progress

**Visual Output**:
```
Found 80 subdirectories in C:\Projects
Processing in batches of 8 directories

Processing directories: │████████████░░░░│ 40/80 (50%)
```

**Progress Bar Theme**: Violet/Magenta (matches user preference for batch operations)
```go
progressbar.Theme{
    Saucer:        "[magenta]█[reset]",
    SaucerHead:    "[magenta]█[reset]",
    SaucerPadding: "░",
    BarStart:      "│",
    BarEnd:        "│",
}
```

### 2. Format-Aware URL Extraction

**Supported File Types**:
- **`.url` files**: Parse Windows INI format for `URL=` and `BaseURL=` fields
- **`.md` files**: Extract markdown links `[text](url)` and plain URLs
- **`.html` files**: Parse `href` and `src` attributes, plus plain URLs
- **`.txt` files**: Regex scan for `http://` and `https://` URLs

**Deduplication**: Uses map internally to avoid duplicate URLs from same file

**URL Validation**:
- Must start with `http://` or `https://`
- Filters out common false positives (example.com, localhost, 127.0.0.1)
- Minimum length requirement

### 3. Smart HTTP Downloads

**Features**:
- 5-minute HTTP timeout to prevent hanging
- Filename generation from URL path or `Content-Disposition` header
- Skip existing files (no re-download)
- Clean up partial files on error
- Download location: same directory as source file

**Filename Sanitization**:
- Removes invalid Windows characters (`< > : " / \ | ? *`)
- Limits length to 200 characters
- Adds `.bin` extension if none present

**Error Handling**:
- Non-blocking errors (warnings continue processing)
- Partial file cleanup on download failure
- Clear error messages with HTTP status codes

### 4. Worker Pool Concurrency

**Pattern**: Producer-Consumer with Buffered Channels

**Components**:
```go
jobs := make(chan string, 100)      // File paths to process
results := make(chan Result, 100)   // Download results
```

**Worker Function**:
- Consumes file paths from jobs channel
- Extracts URLs using format-aware parser
- Downloads each URL to source file's directory
- Sends result for each download to results channel

**Result Collection**:
- Separate goroutine for collecting results
- Atomic counters for thread-safe statistics
- Real-time progress output per worker

**Shutdown Sequence**:
```go
close(jobs)           // No more files to process
workerWg.Wait()       // Wait for all workers to drain channel
close(results)        // No more results to collect
collectorWg.Wait()    // Wait for collector to finish
```

### 5. Statistics Tracking

**Atomic Counters** (thread-safe):
- Files scanned
- URLs found
- Downloads succeeded
- Downloads skipped (already exists)
- Downloads failed

**Output Format**:
```
Summary
=======
Files scanned: 120
URLs found: 45
Downloads succeeded: 38
Downloads skipped: 5
Downloads failed: 2
```

### 6. Completion Notification

**Config File** (`config.json`):
```json
{
  "completion_chime": "C:\\Windows\\Media\\Windows Notify System Generic.wav"
}
```

**Audio Playback**:
- Plays asynchronously (non-blocking)
- Cross-platform support (Windows, macOS, Linux)
- Windows: Uses PowerShell `Media.SoundPlayer`
- macOS: Uses `afplay`
- Linux: Uses `paplay` or `aplay`

## Implementation Details

### Concurrency Model

**Two WaitGroups**:
1. `workerWg`: Tracks worker goroutines
2. `collectorWg`: Tracks result collector goroutine

**Buffered Channels** (size 100):
- Prevents blocking on send/receive
- Balances memory usage with throughput

**Atomic Operations**:
- `atomic.AddInt32()` for thread-safe counter updates
- `atomic.LoadInt32()` for reading final statistics
- No mutex locks needed (better performance)

### Error Handling Philosophy

**Non-Fatal Errors** (continue processing):
- File read errors (print warning, continue)
- URL parsing errors (skip URL, continue)
- Download failures (print error, continue)

**Fatal Errors** (stop execution):
- Invalid command-line arguments
- Non-existent scan directories
- Worker count ≤ 0

**Graceful Degradation**:
- Missing config.json → skip completion chime
- No subdirectories → scan root only
- Download failure → clean up and continue

### Dependencies

```go
require (
    github.com/schollz/progressbar/v3 v3.18.0
    github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213
)
```

**Why These Dependencies**:
- `progressbar/v3`: Rich progress bar with color support
- `go-ansi`: ANSI color code support on Windows

## Testing

### Manual Testing Checklist

1. **CLI Argument Parsing**
   - [ ] Test with no arguments (should fail)
   - [ ] Test with -workers only (should fail)
   - [ ] Test with -scan only (should fail)
   - [ ] Test with invalid worker count (should fail)
   - [ ] Test with non-existent directory (should fail)
   - [ ] Test with valid arguments (should succeed)

2. **File Format Parsing**
   - [ ] Create sample .url file with URL= field
   - [ ] Create sample .md file with markdown links
   - [ ] Create sample .html file with href/src attributes
   - [ ] Create sample .txt file with plain URLs
   - [ ] Verify URL extraction for each format

3. **Batch Processing**
   - [ ] Test with directory containing 25 subdirectories (batch size = 3)
   - [ ] Verify progress bar displays correctly
   - [ ] Test with directory containing 0 subdirectories
   - [ ] Test --recursive vs non-recursive modes

4. **Download Functionality**
   - [ ] Test downloading various file types
   - [ ] Test Content-Disposition filename extraction
   - [ ] Test existing file skip behavior
   - [ ] Test invalid/broken URLs (should fail gracefully)
   - [ ] Verify files download to correct directory

5. **Worker Pool**
   - [ ] Test with different worker counts (1, 4, 8, 16)
   - [ ] Verify concurrent downloads work correctly
   - [ ] Check statistics are accurate

6. **Completion Chime**
   - [ ] Test with valid audio file path
   - [ ] Test with invalid path (should warn)
   - [ ] Test with missing config.json (should skip)

## Performance Characteristics

**Expected Performance**:
- File scanning: ~1000+ files/second (limited by disk I/O)
- URL extraction: ~100+ files/second (limited by regex parsing)
- Downloads: Limited by network bandwidth and worker count

**Memory Usage**:
- Base: ~10-20 MB
- Channel buffers: 200 file paths in memory max
- Workers: ~1 MB per worker (HTTP client + buffers)

**Scalability**:
- Worker count: Recommended 4-16 (balance CPU/network)
- Channel buffer size: 100 (can increase for large workloads)
- No limits on file count or download sizes

## Known Limitations

1. **URL Validation**: Basic validation only (no DNS lookup)
2. **Content-Type**: Downloads any file type (not just archives)
3. **Authentication**: No support for authenticated downloads
4. **Redirects**: Follows redirects automatically via http.Client
5. **HTTPS**: Certificate validation enabled (may fail with self-signed certs)

## Future Enhancements

- [ ] Add support for authenticated downloads (HTTP Basic Auth)
- [ ] Add retry logic for failed downloads
- [ ] Add support for rate limiting
- [ ] Add support for filtering by file extension
- [ ] Add support for dry-run mode (scan without downloading)
- [ ] Add support for resuming interrupted downloads
- [ ] Add support for parallel chunk downloading (for large files)

## Troubleshooting

**Problem**: "Error: at least one -scan directory must be specified"
- **Solution**: Add `-scan <directory>` flag

**Problem**: Downloads fail with timeout
- **Solution**: Increase timeout in downloader.go (currently 5 minutes)

**Problem**: Progress bar not showing colors on Windows
- **Solution**: Ensure Windows 10+ with ANSI support enabled

**Problem**: Completion chime not playing
- **Solution**: Check config.json path and audio file existence

## Code References

- `main.go:62`: CLI argument validation
- `scanner.go:23`: Batch size calculation
- `parser.go:17`: URL regex patterns
- `downloader.go:35`: HTTP download implementation
- `worker.go:17`: Worker pool pattern
- `main.go:138`: Completion chime playback
