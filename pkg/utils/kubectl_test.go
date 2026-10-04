package utils

import (
	"runtime"
	"strings"
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"
)

func TestIsKubectl(t *testing.T) {
	tests := []struct {
		owner   string
		repo    string
		release string
		want    bool
	}{
		{"kubernetes", "kubectl", "", true},
		{"", "kubectl", "", true},
		{"custom", "custom", "https://dl.k8s.io/release/v1.35.3/bin/linux/amd64/kubectl", true},
		{"siderolabs", "talos", "", false},
		{"helm", "helm", "v4.2.3", false},
	}

	for _, tt := range tests {
		got := IsKubectl(tt.owner, tt.repo, tt.release)
		require.Equal(t, tt.want, got, "IsKubectl(%q, %q, %q)", tt.owner, tt.repo, tt.release)
	}
}

func TestGetKubectlVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://dl.k8s.io/release/v1.35.3/bin/linux/amd64/kubectl", "v1.35.3"},
		{"https://dl.k8s.io/release/v1.37.1/bin/darwin/arm64/kubectl", "v1.37.1"},
		{"v1.35.3", "v1.35.3"},
		{"1.35.3", "v1.35.3"},
		{"latest", "latest"},
	}

	for _, tt := range tests {
		got := GetKubectlVersion(tt.input)
		require.Equal(t, tt.want, got, "GetKubectlVersion(%q)", tt.input)
	}
}

func TestGetKubectlDownloadURL(t *testing.T) {
	url := GetKubectlDownloadURL("v1.37.1")
	require.True(t, strings.HasPrefix(url, "https://dl.k8s.io/release/v1.37.1/bin/"))
	require.True(t, strings.Contains(url, runtime.GOOS))
	require.True(t, strings.Contains(url, runtime.GOARCH))
	require.True(t, strings.HasSuffix(url, "/kubectl"))

	urlWithoutV := GetKubectlDownloadURL("1.35.3")
	require.True(t, strings.HasPrefix(urlWithoutV, "https://dl.k8s.io/release/v1.35.3/bin/"))
}

func TestGetKubectlLatestRelease(t *testing.T) {
	defer gock.Off()

	gock.New("https://dl.k8s.io").
		Get("/release/stable.txt").
		Reply(200).
		BodyString("v1.37.1\n")

	tag, err := GetKubectlLatestRelease()
	require.NoError(t, err)
	require.Equal(t, "v1.37.1", tag)
}
