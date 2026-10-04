package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var sha256Pattern = regexp.MustCompile(`(?i)\b([a-f0-9]{64})\b`)

// ComputeSHA256 calculates the SHA-256 checksum of the file at filePath.
func ComputeSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ParseSHA256FromManifest searches manifest content for a SHA-256 hash matching targetFilename.
func ParseSHA256FromManifest(content, targetFilename string) (string, bool) {
	trimmed := strings.TrimSpace(content)
	// Case 1: The entire content is just a 64-character hex hash
	if len(trimmed) == 64 && sha256Pattern.MatchString(trimmed) {
		return strings.ToLower(trimmed), true
	}

	baseName := filepath.Base(targetFilename)
	lines := strings.Split(content, "\n")

	// Case 2: Look for a line containing the target filename
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "#") {
			continue // Skip comments
		}
		if strings.Contains(line, targetFilename) || strings.Contains(line, baseName) {
			if match := sha256Pattern.FindString(line); match != "" {
				return strings.ToLower(match), true
			}
		}
	}

	// Case 3: If there is only one non-empty, non-comment line, check if it contains a hash
	var nonEmptyLines []string
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			nonEmptyLines = append(nonEmptyLines, trimmedLine)
		}
	}
	if len(nonEmptyLines) == 1 {
		if match := sha256Pattern.FindString(nonEmptyLines[0]); match != "" {
			return strings.ToLower(match), true
		}
	}

	return "", false
}

// VerifyFileSHA256 calculates the SHA-256 of filePath and compares it with expectedHash.
func VerifyFileSHA256(filePath, expectedHash string) error {
	actualHash, err := ComputeSHA256(filePath)
	if err != nil {
		return fmt.Errorf("failed to compute SHA-256: %w", err)
	}
	if !strings.EqualFold(actualHash, strings.TrimSpace(expectedHash)) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

// FetchURLText fetches remote text content from rawURL, applying timeouts and authentication.
func FetchURLText(rawURL string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	if IsGitHubURL(rawURL) {
		if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
		}
		req.Header.Set("Accept", "application/octet-stream")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if timeout := GetHTTPTimeout(); timeout > 0 {
		transport.ResponseHeaderTimeout = timeout
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   GetHTTPTimeout(),
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching checksum from %s", resp.StatusCode, rawURL)
	}

	// Limit checksum manifests to 2 MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
