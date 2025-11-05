package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

// Supported file extensions for scanning
var supportedExtensions = map[string]bool{
	".url":  true,
	".md":   true,
	".html": true,
	".htm":  true,
	".txt":  true,
}

// scanDirectoryWithBatches scans a directory and processes subdirectories in batches
func scanDirectoryWithBatches(rootDir string, recursive bool, jobs chan<- string, stats *Stats) {
	if recursive {
		// Get all subdirectories
		subdirs, err := getSubdirectories(rootDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get subdirectories of %s: %v\n", rootDir, err)
			// Fall back to scanning root directory only
			scanDirectory(rootDir, false, jobs, stats)
			return
		}

		// If no subdirectories found, just scan the root directory
		if len(subdirs) == 0 {
			fmt.Printf("No subdirectories found in %s, scanning root directory only\n", rootDir)
			scanDirectory(rootDir, false, jobs, stats)
			return
		}

		// Calculate batch size: ceil(N / 10)
		batchSize := int(math.Ceil(float64(len(subdirs)) / 10.0))
		if batchSize == 0 {
			batchSize = 1
		}

		fmt.Printf("Found %d subdirectories in %s\n", len(subdirs), rootDir)
		fmt.Printf("Processing in batches of %d directories\n\n", batchSize)

		// Create violet progress bar
		bar := progressbar.NewOptions(len(subdirs),
			progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription("Processing directories"),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[magenta]█[reset]",
				SaucerHead:    "[magenta]█[reset]",
				SaucerPadding: "░",
				BarStart:      "│",
				BarEnd:        "│",
			}),
		)

		// Process subdirectories in batches
		for i := 0; i < len(subdirs); i += batchSize {
			end := i + batchSize
			if end > len(subdirs) {
				end = len(subdirs)
			}

			batch := subdirs[i:end]

			// Process each directory in the batch
			for _, dir := range batch {
				scanDirectory(dir, true, jobs, stats) // Recurse into subdirectories
				bar.Add(1)
			}
		}

		bar.Finish()
		fmt.Println()

		// Also scan files in the root directory itself
		scanDirectory(rootDir, false, jobs, stats)

	} else {
		// Non-recursive: just scan the root directory
		scanDirectory(rootDir, false, jobs, stats)
	}
}

// getSubdirectories returns a list of immediate subdirectories in a directory
func getSubdirectories(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			subdirPath := filepath.Join(rootDir, entry.Name())
			subdirs = append(subdirs, subdirPath)
		}
	}

	return subdirs, nil
}

// scanDirectory scans a single directory for supported file types
func scanDirectory(dir string, recursive bool, jobs chan<- string, stats *Stats) {
	if recursive {
		// Recursive scan using filepath.Walk
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot access %s: %v\n", path, err)
				return nil // Continue walking
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Check if file has supported extension
			if isSupportedFile(path) {
				jobs <- path
			}

			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error walking directory %s: %v\n", dir, err)
		}
	} else {
		// Non-recursive: only scan immediate files in directory
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read directory %s: %v\n", dir, err)
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			filePath := filepath.Join(dir, entry.Name())
			if isSupportedFile(filePath) {
				jobs <- filePath
			}
		}
	}
}

// isSupportedFile checks if a file has a supported extension
func isSupportedFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return supportedExtensions[ext]
}
