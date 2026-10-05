package utils

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
)

const (
	KubectlStableURL = "https://dl.k8s.io/release/stable.txt"
)

// IsKubectl checks if the given owner, repo, or release corresponds to kubectl.
func IsKubectl(owner, repo, release string) bool {
	if strings.EqualFold(repo, "kubectl") {
		return true
	}
	if strings.EqualFold(owner, "kubernetes") && strings.EqualFold(repo, "kubectl") {
		return true
	}
	if strings.Contains(release, "dl.k8s.io") {
		return true
	}
	return false
}

// GetKubectlVersion extracts the semantic version (e.g. "v1.35.3") from a kubectl release string or URL.
func GetKubectlVersion(release string) string {
	release = strings.TrimSpace(release)
	// If it's a dl.k8s.io URL: https://dl.k8s.io/release/v1.35.3/bin/linux/amd64/kubectl
	reURL := regexp.MustCompile(`dl\.k8s\.io/release/([^/]+)/`)
	if match := reURL.FindStringSubmatch(release); len(match) > 1 {
		return NormalizeKubectlVersion(match[1])
	}

	// Fallback regex for generic version extraction in URLs
	if strings.HasPrefix(release, "http") {
		reVer := regexp.MustCompile(`[/v-]([\d.]+)`)
		if match := reVer.FindStringSubmatch(release); len(match) > 1 {
			return NormalizeKubectlVersion(match[1])
		}
	}

	return NormalizeKubectlVersion(release)
}

// NormalizeKubectlVersion ensures the version has a "v" prefix (e.g. "1.35.3" -> "v1.35.3").
func NormalizeKubectlVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "latest" {
		return v
	}
	if !strings.HasPrefix(v, "v") && len(v) > 0 && (v[0] >= '0' && v[0] <= '9') {
		return "v" + v
	}
	return v
}

// GetKubectlDownloadURL returns the dl.k8s.io binary download URL for the given version and current OS/architecture.
func GetKubectlDownloadURL(version string) string {
	normVer := NormalizeKubectlVersion(version)
	osName := runtime.GOOS
	archName := runtime.GOARCH

	binaryName := "kubectl"
	if osName == "windows" {
		binaryName = "kubectl.exe"
	}

	return fmt.Sprintf("https://dl.k8s.io/release/%s/bin/%s/%s/%s", normVer, osName, archName, binaryName)
}

// GetKubectlLatestRelease fetches the latest stable release tag from dl.k8s.io.
func GetKubectlLatestRelease() (string, error) {
	client := &http.Client{
		Timeout: GetHTTPTimeout(),
	}
	resp, err := client.Get(KubectlStableURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid HTTP status fetching kubectl stable release: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	tag := strings.TrimSpace(string(body))
	return NormalizeKubectlVersion(tag), nil
}
