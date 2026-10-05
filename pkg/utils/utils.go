package utils

import (
	"crhuber/kelp/pkg/logging"
	"crhuber/kelp/pkg/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func DirExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	return !os.IsNotExist(err) && !info.IsDir()
}

func FilePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func CopyFile(source, destination string) error {
	from, err := os.Open(source)
	if err != nil {
		return err
	}
	defer from.Close()

	fromInfo, err := from.Stat()
	if err != nil {
		return err
	}

	mode := fromInfo.Mode().Perm()
	if mode == 0 {
		mode = 0755
	}

	// Write to a temporary file in the destination's directory and atomically rename.
	// This avoids ETXTBSY if the destination binary is currently running,
	// and prevents partial writes if copying is interrupted.
	dir := filepath.Dir(destination)
	tmpFile, err := os.CreateTemp(dir, "kelp-copy-*")
if err != nil {
	return fmt.Errorf("could not create temporary file in %s: %w", dir, err)
}
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := io.Copy(tmpFile, from); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, destination)
}

func GetGithubRelease(owner, repo, release string) (*types.GithubRelease, error) {
	var url string
	if release == "latest" {
		logging.LogInfo("🌐 Getting releases for %s/%s:%s...", owner, repo, release)
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%s", owner, repo, release)

	} else {
		// try by tag
		logging.LogInfo("🌐 Getting releases by tag %s...", release)
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, release)
	}

	// create client
	client := &http.Client{
		Timeout: GetHTTPTimeout(),
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// set headers for github auth
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken != "" {
		logging.LogDebug("Using Github token in http request")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
	}

	// make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid HTTP status: %v", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ghr := types.GithubRelease{}
	if err := json.Unmarshal(body, &ghr); err != nil {
		return nil, err
	}
	return &ghr, nil
}

// IsGitHubURL checks whether rawURL points to a GitHub domain (github.com or a subdomain such as api.github.com).
func IsGitHubURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hostname := strings.ToLower(u.Hostname())
	return hostname == "github.com" || strings.HasSuffix(hostname, ".github.com")
}

// GetHTTPTimeout returns the configured HTTP timeout duration.
// It checks KELP_HTTP_TIMEOUT (e.g. "60s", "2m", "0" or "off" to disable).
// Defaults to 60s if unset.
func GetHTTPTimeout() time.Duration {
	if val := os.Getenv("KELP_HTTP_TIMEOUT"); val != "" {
		trimmed := strings.TrimSpace(val)
		if trimmed == "0" || strings.EqualFold(trimmed, "none") || strings.EqualFold(trimmed, "off") {
			return 0
		}
		if d, err := time.ParseDuration(trimmed); err == nil {
			return d
		}
		// If the user provided a raw number like "120", treat as seconds
		var seconds int
		if _, err := fmt.Sscanf(trimmed, "%d", &seconds); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return 60 * time.Second
}


