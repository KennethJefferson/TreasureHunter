package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// HTTPClient is the shared HTTP client with timeout
var HTTPClient = &http.Client{
	Timeout: 5 * time.Minute,
}

// GitHub URL patterns for matching
var (
	githubRepoPattern = regexp.MustCompile(`^https?://github\.com/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_.-]+)`)
)

// DownloadResult represents the result of a download attempt
type DownloadResult struct {
	URL         string
	FilePath    string
	Success     bool
	Skipped     bool
	Error       error
	BytesWritten int64
}

// downloadURL downloads a file from a URL to a target directory
func downloadURL(downloadURL, targetDir string) DownloadResult {
	result := DownloadResult{
		URL: downloadURL,
	}

	// Check if this is a GitHub URL and handle it specially
	if isGitHubRepoURL(downloadURL) {
		// Extract owner and repo from URL
		owner, repo := extractGitHubInfo(downloadURL)
		if owner != "" && repo != "" {
			// Generate filename for GitHub repo
			filename := fmt.Sprintf("%s-%s.zip", owner, repo)
			filePath := filepath.Join(targetDir, filename)
			result.FilePath = filePath

			// Check if file already exists - skip if it does
			if _, err := os.Stat(filePath); err == nil {
				result.Skipped = true
				result.Error = fmt.Errorf("file already exists")
				return result
			}

			// Try to download from main, master, and HEAD branches
			branches := []string{"main", "master", "HEAD"}
			var lastErr error

			for _, branch := range branches {
				var archiveURL string
				if branch == "HEAD" {
					archiveURL = fmt.Sprintf("https://github.com/%s/%s/archive/HEAD.zip", owner, repo)
				} else {
					archiveURL = fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, branch)
				}

				// Try to download from this branch
				resp, err := HTTPClient.Get(archiveURL)
				if err != nil {
					lastErr = err
					continue
				}
				defer resp.Body.Close()

				// Check if successful
				if resp.StatusCode == http.StatusOK {
					// Create output file
					outFile, err := os.Create(filePath)
					if err != nil {
						result.Error = fmt.Errorf("failed to create file: %w", err)
						return result
					}

					// Copy response body to file
					bytesWritten, err := io.Copy(outFile, resp.Body)
					outFile.Close()

					if err != nil {
						// Clean up partial file on error
						os.Remove(filePath)
						result.Error = fmt.Errorf("failed to write file: %w", err)
						return result
					}

					// Success
					result.Success = true
					result.BytesWritten = bytesWritten
					return result
				}
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			}

			// All branches failed
			result.Error = fmt.Errorf("failed to download GitHub repo from all branches: %w", lastErr)
			return result
		}
	}

	// Not a GitHub repo URL or failed to parse - proceed with normal download
	// Generate filename from URL
	filename, err := getFilenameFromURL(downloadURL)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse URL: %w", err)
		return result
	}

	// Create full file path
	filePath := filepath.Join(targetDir, filename)
	result.FilePath = filePath

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		result.Skipped = true
		result.Error = fmt.Errorf("file already exists")
		return result
	}

	// Make HTTP request
	resp, err := HTTPClient.Get(downloadURL)
	if err != nil {
		result.Error = fmt.Errorf("HTTP request failed: %w", err)
		return result
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		return result
	}

	// Try to get filename from Content-Disposition header
	if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
		if cdFilename := parseContentDisposition(contentDisposition); cdFilename != "" {
			filename = cdFilename
			filePath = filepath.Join(targetDir, filename)
			result.FilePath = filePath

			// Check again if file exists with new filename
			if _, err := os.Stat(filePath); err == nil {
				result.Skipped = true
				result.Error = fmt.Errorf("file already exists")
				return result
			}
		}
	}

	// Create output file
	outFile, err := os.Create(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create file: %w", err)
		return result
	}

	// Copy response body to file
	bytesWritten, err := io.Copy(outFile, resp.Body)
	outFile.Close()

	if err != nil {
		// Clean up partial file on error
		os.Remove(filePath)
		result.Error = fmt.Errorf("failed to write file: %w", err)
		return result
	}

	// Success
	result.Success = true
	result.BytesWritten = bytesWritten
	return result
}

// isGitHubRepoURL checks if a URL points to a GitHub repository
func isGitHubRepoURL(urlStr string) bool {
	// Skip if already an archive URL or raw content
	if strings.Contains(urlStr, "/archive/") ||
		strings.Contains(urlStr, "/releases/") ||
		strings.Contains(urlStr, "raw.githubusercontent.com") {
		return false
	}

	// Check if it matches GitHub repo pattern
	return githubRepoPattern.MatchString(urlStr)
}

// extractGitHubInfo extracts owner and repo name from a GitHub URL
func extractGitHubInfo(urlStr string) (owner, repo string) {
	matches := githubRepoPattern.FindStringSubmatch(urlStr)
	if len(matches) >= 3 {
		owner = matches[1]
		repo = matches[2]

		// Clean up repo name
		repo = strings.TrimSuffix(repo, ".git")

		// Remove any path segments after repo name
		if idx := strings.IndexAny(repo, "/#?"); idx != -1 {
			repo = repo[:idx]
		}
	}
	return owner, repo
}

// getFilenameFromURL extracts a filename from a URL
func getFilenameFromURL(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Get the last segment of the path
	filename := path.Base(parsedURL.Path)

	// If filename is empty, root, or a directory, generate a default name
	if filename == "" || filename == "/" || filename == "." {
		// Use domain name as base
		filename = fmt.Sprintf("download_%s", parsedURL.Host)
		// Sanitize the filename
		filename = sanitizeFilename(filename)
	}

	// If still no extension, add .bin
	if !strings.Contains(filename, ".") {
		filename += ".bin"
	}

	return filename, nil
}

// parseContentDisposition extracts filename from Content-Disposition header
func parseContentDisposition(header string) string {
	// Look for filename= or filename*= in the header
	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Handle filename*=UTF-8''filename format
		if strings.HasPrefix(part, "filename*=") {
			value := strings.TrimPrefix(part, "filename*=")
			// Remove UTF-8'' prefix if present
			if idx := strings.Index(value, "''"); idx != -1 {
				value = value[idx+2:]
			}
			// Remove quotes
			value = strings.Trim(value, "\"'")
			if value != "" {
				return sanitizeFilename(value)
			}
		}

		// Handle filename="value" or filename=value format
		if strings.HasPrefix(part, "filename=") {
			value := strings.TrimPrefix(part, "filename=")
			value = strings.Trim(value, "\"'")
			if value != "" {
				return sanitizeFilename(value)
			}
		}
	}

	return ""
}

// sanitizeFilename removes or replaces invalid characters in filenames
func sanitizeFilename(filename string) string {
	// Replace invalid Windows filename characters
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalid {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	// Remove leading/trailing spaces and dots
	filename = strings.TrimSpace(filename)
	filename = strings.Trim(filename, ".")

	// Limit filename length (Windows has 260 char path limit)
	if len(filename) > 200 {
		ext := filepath.Ext(filename)
		base := filename[:200-len(ext)]
		filename = base + ext
	}

	return filename
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}