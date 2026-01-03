package utils

import (
	"os"
	"path/filepath"
	"testing"

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
