package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// URL patterns for regex extraction
var (
	// Match http:// and https:// URLs
	urlPattern = regexp.MustCompile(`https?://[^\s<>"{}|\\^\[\]` + "`" + `()]+`)

	// Match markdown links: [text](url)
	markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	// Match HTML href and src attributes
	htmlHrefPattern = regexp.MustCompile(`(?i)href=["']([^"']+)["']`)
	htmlSrcPattern  = regexp.MustCompile(`(?i)src=["']([^"']+)["']`)
)

// extractURLsFromFile extracts URLs from a file based on its extension
func extractURLsFromFile(filePath string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".url":
		return extractURLsFromURLFile(filePath)
	case ".md":
		return extractURLsFromMarkdown(filePath)
	case ".html", ".htm":
		return extractURLsFromHTML(filePath)
	case ".txt":
		return extractURLsFromText(filePath)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// extractURLsFromURLFile parses Windows .url files (INI format)
func extractURLsFromURLFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for URL= or BaseURL= lines
		if strings.HasPrefix(line, "URL=") {
			url := strings.TrimPrefix(line, "URL=")
			url = strings.TrimSpace(url)
			if url != "" && isValidURL(url) {
				urls = append(urls, url)
			}
		} else if strings.HasPrefix(line, "BaseURL=") {
			url := strings.TrimPrefix(line, "BaseURL=")
			url = strings.TrimSpace(url)
			if url != "" && isValidURL(url) {
				urls = append(urls, url)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return urls, nil
}

// extractURLsFromMarkdown extracts URLs from markdown files
func extractURLsFromMarkdown(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)
	urlMap := make(map[string]bool) // Use map to avoid duplicates

	// Extract markdown links [text](url)
	markdownMatches := markdownLinkPattern.FindAllStringSubmatch(text, -1)
	for _, match := range markdownMatches {
		if len(match) >= 3 {
			url := strings.TrimSpace(match[2])
			if isValidURL(url) {
				urlMap[url] = true
			}
		}
	}

	// Extract plain URLs
	plainURLs := urlPattern.FindAllString(text, -1)
	for _, url := range plainURLs {
		url = strings.TrimSpace(url)
		if isValidURL(url) {
			urlMap[url] = true
		}
	}

	// Convert map to slice
	urls := make([]string, 0, len(urlMap))
	for url := range urlMap {
		urls = append(urls, url)
	}

	return urls, nil
}

// extractURLsFromHTML extracts URLs from HTML files
func extractURLsFromHTML(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)
	urlMap := make(map[string]bool) // Use map to avoid duplicates

	// Extract href attributes
	hrefMatches := htmlHrefPattern.FindAllStringSubmatch(text, -1)
	for _, match := range hrefMatches {
		if len(match) >= 2 {
			url := strings.TrimSpace(match[1])
			if isValidURL(url) {
				urlMap[url] = true
			}
		}
	}

	// Extract src attributes
	srcMatches := htmlSrcPattern.FindAllStringSubmatch(text, -1)
	for _, match := range srcMatches {
		if len(match) >= 2 {
			url := strings.TrimSpace(match[1])
			if isValidURL(url) {
				urlMap[url] = true
			}
		}
	}

	// Extract plain URLs
	plainURLs := urlPattern.FindAllString(text, -1)
	for _, url := range plainURLs {
		url = strings.TrimSpace(url)
		if isValidURL(url) {
			urlMap[url] = true
		}
	}

	// Convert map to slice
	urls := make([]string, 0, len(urlMap))
	for url := range urlMap {
		urls = append(urls, url)
	}

	return urls, nil
}

// extractURLsFromText extracts URLs from plain text files using regex
func extractURLsFromText(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)
	urlMap := make(map[string]bool) // Use map to avoid duplicates

	// Extract all URLs
	matches := urlPattern.FindAllString(text, -1)
	for _, url := range matches {
		url = strings.TrimSpace(url)
		if isValidURL(url) {
			urlMap[url] = true
		}
	}

	// Convert map to slice
	urls := make([]string, 0, len(urlMap))
	for url := range urlMap {
		urls = append(urls, url)
	}

	return urls, nil
}

// isValidURL performs basic validation on a URL string
func isValidURL(url string) bool {
	// Must start with http:// or https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}

	// Must have more than just the protocol
	if len(url) < 10 {
		return false
	}

	// Filter out common false positives
	lowerURL := strings.ToLower(url)
	if strings.Contains(lowerURL, "example.com") ||
		strings.Contains(lowerURL, "example.org") ||
		strings.Contains(lowerURL, "localhost") ||
		strings.Contains(lowerURL, "127.0.0.1") {
		return false
	}

	return true
}
