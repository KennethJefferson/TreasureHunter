package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
)

// Config holds the application configuration
type Config struct {
	CompletionChime string `json:"completion_chime"`
}

// Stats holds atomic counters for processing statistics
type Stats struct {
	FilesScanned    int32
	URLsFound       int32
	DownloadSuccess int32
	DownloadSkipped int32
	DownloadFailed  int32
}

// scanDirsFlag is a custom flag type for repeatable -scan arguments
type scanDirsFlag []string

func (s *scanDirsFlag) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *scanDirsFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	// Define command-line flags
	var scanDirs scanDirsFlag
	var workers int
	var recursive bool

	flag.Var(&scanDirs, "scan", "Directory to scan (can be specified multiple times)")
	flag.IntVar(&workers, "workers", 0, "Number of concurrent download workers (required)")
	flag.BoolVar(&recursive, "recursive", false, "Scan subdirectories recursively")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -workers <num> -scan <dir1> [-scan <dir2>...] [--recursive]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -workers 4 -scan C:\\Downloads\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -workers 8 -scan C:\\Downloads -scan D:\\Archives --recursive\n", os.Args[0])
	}

	flag.Parse()

	// Validate arguments
	if workers <= 0 {
		log.Fatalf("Error: -workers flag is required and must be greater than 0")
	}

	if len(scanDirs) == 0 {
		log.Fatalf("Error: at least one -scan directory must be specified")
	}

	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config.json: %v\n", err)
		config = &Config{} // Use empty config
	}

	// Convert scan directories to absolute paths and verify existence
	for i, dir := range scanDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			log.Fatalf("Error: failed to resolve path '%s': %v", dir, err)
		}

		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			log.Fatalf("Error: directory does not exist: %s", absDir)
		}

		scanDirs[i] = absDir
	}

	// Print configuration
	fmt.Printf("Archive Downloader\n")
	fmt.Printf("==================\n")
	fmt.Printf("Workers: %d\n", workers)
	fmt.Printf("Recursive: %v\n", recursive)
	fmt.Printf("Scan directories:\n")
	for _, dir := range scanDirs {
		fmt.Printf("  - %s\n", dir)
	}
	fmt.Println()

	// Create channels for job distribution and result collection
	jobs := make(chan string, 100)
	results := make(chan Result, 100)

	// Initialize statistics
	stats := &Stats{}

	// Start worker pool
	var workerWg sync.WaitGroup
	for i := 0; i < workers; i++ {
		workerWg.Add(1)
		go worker(i+1, jobs, results, &workerWg)
	}

	// Start result collector
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go collectResults(results, stats, &collectorWg)

	// Scan directories and send files to workers
	for _, scanDir := range scanDirs {
		scanDirectoryWithBatches(scanDir, recursive, jobs, stats)
	}

	// Shutdown sequence
	close(jobs)         // No more files to process
	workerWg.Wait()     // Wait for all workers to finish
	close(results)      // No more results to collect
	collectorWg.Wait()  // Wait for collector to finish

	// Print summary statistics
	fmt.Printf("\n")
	fmt.Printf("Summary\n")
	fmt.Printf("=======\n")
	fmt.Printf("Files scanned: %d\n", atomic.LoadInt32(&stats.FilesScanned))
	fmt.Printf("URLs found: %d\n", atomic.LoadInt32(&stats.URLsFound))
	fmt.Printf("Downloads succeeded: %d\n", atomic.LoadInt32(&stats.DownloadSuccess))
	fmt.Printf("Downloads skipped: %d\n", atomic.LoadInt32(&stats.DownloadSkipped))
	fmt.Printf("Downloads failed: %d\n", atomic.LoadInt32(&stats.DownloadFailed))

	// Play completion chime if configured
	if config.CompletionChime != "" {
		playCompletionChime(config.CompletionChime)
	}
}

// loadConfig loads the configuration from a JSON file
func loadConfig(path string) (*Config, error) {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, path)

	// Try executable directory first, then current directory
	file, err := os.Open(configPath)
	if err != nil {
		file, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// playCompletionChime plays an audio file as a completion notification
func playCompletionChime(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: completion chime file not found: %s\n", path)
		return
	}

	// Play audio asynchronously (don't wait for completion)
	go func() {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "windows":
			// Use PowerShell to play audio on Windows
			cmd = exec.Command("powershell", "-c", fmt.Sprintf("(New-Object Media.SoundPlayer '%s').PlaySync();", path))
		case "darwin":
			// Use afplay on macOS
			cmd = exec.Command("afplay", path)
		case "linux":
			// Try common Linux audio players
			if _, err := exec.LookPath("paplay"); err == nil {
				cmd = exec.Command("paplay", path)
			} else if _, err := exec.LookPath("aplay"); err == nil {
				cmd = exec.Command("aplay", path)
			}
		}

		if cmd != nil {
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to play completion chime: %v\n", err)
			}
		}
	}()
}
