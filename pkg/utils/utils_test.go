package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"
)

func TestGetGithubRelease(t *testing.T) {
	defer gock.Off()

	filename := filepath.Join("..", "..", "testdata", "helm-latest.json")
	jsonBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal("error reading testdata: ", err)
	}
	gock.New("https://api.github.com").
		Get("/repos/helm/helm/releases/latest").
		Reply(200).
		JSON(jsonBytes)
	ghr, err := GetGithubRelease("helm", "helm", "latest")
	require.NoError(t, err)
	require.Equal(t, "v4.0.4", ghr.TagName)
}

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://api.github.com/repos/helm/helm/releases", true},
		{"https://github.com/cli/cli/releases/download/v2.0.0/cli.tar.gz", true},
		{"https://codeload.github.com/foo/bar", true},
		{"http://github.com/test", true},
		{"https://dl.k8s.io/release/v1.35.3/bin/linux/amd64/kubectl", false},
		{"https://example.com/foo.tar.gz", false},
		{"https://evil-github.com/download", false},
		{"https://github.com.attacker.com/download", false},
		{"https://objects.githubusercontent.com/foo", false},
		{"not-a-valid-url:://%", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsGitHubURL(tt.url)
			require.Equal(t, tt.want, got, "IsGitHubURL(%q)", tt.url)
		})
	}
}

func TestGetHTTPTimeout(t *testing.T) {
	origVal, exists := os.LookupEnv("KELP_HTTP_TIMEOUT")
	defer func() {
		if exists {
			os.Setenv("KELP_HTTP_TIMEOUT", origVal)
		} else {
			os.Unsetenv("KELP_HTTP_TIMEOUT")
		}
	}()

	tests := []struct {
		envVal   string
		setEnv   bool
		expected time.Duration
	}{
		{"", false, 60 * time.Second},
		{"", true, 60 * time.Second},
		{"120s", true, 120 * time.Second},
		{"2m", true, 2 * time.Minute},
		{"90", true, 90 * time.Second},
		{"0", true, 0},
		{"off", true, 0},
		{"OFF", true, 0},
		{"none", true, 0},
		{"invalid", true, 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.envVal, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv("KELP_HTTP_TIMEOUT", tt.envVal)
			} else {
				os.Unsetenv("KELP_HTTP_TIMEOUT")
			}
			got := GetHTTPTimeout()
			require.Equal(t, tt.expected, got)
		})
	}
}


