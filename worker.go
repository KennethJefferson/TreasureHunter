package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Result represents the result of processing a single file
type Result struct {
	FilePath        string
	URLsFound       int
	DownloadResults []DownloadResult
}

// worker processes files from the jobs channel
func worker(id int, jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range jobs {
		result := processFile(id, filePath)
		results <- result
	}
}

// processFile processes a single file and downloads all URLs found in it
func processFile(workerID int, filePath string) Result {
	result := Result{
		FilePath: filePath,
	}

	// Extract URLs from file
	urls, err := extractURLsFromFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Worker %d] ✗ Error reading %s: %v\n", workerID, filepath.Base(filePath), err)
		return result
	}

	result.URLsFound = len(urls)

	if len(urls) == 0 {
		// No URLs found, skip silently or print if verbose
		return result
	}

	fmt.Printf("[Worker %d] Found %d URL(s) in %s\n", workerID, len(urls), filepath.Base(filePath))

	// Get the directory of the source file
	targetDir := filepath.Dir(filePath)

	// Download each URL
	for _, url := range urls {
		downloadResult := downloadURL(url, targetDir)
		result.DownloadResults = append(result.DownloadResults, downloadResult)

		// Print download result
		if downloadResult.Success {
			fmt.Printf("[Worker %d] ✓ Downloaded: %s (%s)\n", workerID, filepath.Base(downloadResult.FilePath), formatBytes(downloadResult.BytesWritten))
		} else if downloadResult.Skipped {
			fmt.Printf("[Worker %d] ⏭ Skipped: %s (already exists)\n", workerID, filepath.Base(downloadResult.FilePath))
		} else {
			fmt.Fprintf(os.Stderr, "[Worker %d] ✗ Failed: %s - %v\n", workerID, url, downloadResult.Error)
		}
	}

	return result
}

// collectResults collects results from workers and updates statistics
func collectResults(results <-chan Result, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	for result := range results {
		// Increment files scanned counter
		atomic.AddInt32(&stats.FilesScanned, 1)

		// Increment URLs found counter
		atomic.AddInt32(&stats.URLsFound, int32(result.URLsFound))

		// Process download results
		for _, downloadResult := range result.DownloadResults {
			if downloadResult.Success {
				atomic.AddInt32(&stats.DownloadSuccess, 1)
			} else if downloadResult.Skipped {
				atomic.AddInt32(&stats.DownloadSkipped, 1)
			} else {
				atomic.AddInt32(&stats.DownloadFailed, 1)
			}
		}
	}
}
