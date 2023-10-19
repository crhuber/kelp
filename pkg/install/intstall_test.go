package install

import (
	"crhuber/kelp/pkg/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindGithubReleaseMacAssets(t *testing.T) {

	var assets []types.Asset
	asset1 := types.Asset{
		BrowserDownloadURL: "https://github.com/trufflesecurity/trufflehog/releases/download/v3.60.1/trufflehog_3.60.1_darwin_amd64.tar.gz",
	}
	asset2 := types.Asset{
		BrowserDownloadURL: "https://github.com/trufflesecurity/trufflehog/releases/download/v3.60.1/trufflehog_3.60.1_darwin_arm64.tar.gz",
	}
	assets = append(assets, asset1, asset2)

	downloadableAssets, _ := findGithubReleaseMacAssets(assets)
	require.Equal(t, asset2, downloadableAssets)
}

func TestGetHighestScore(t *testing.T) {

	assetScores := map[int]int{}
	assetScores[0] = 6
	assetScores[1] = 8
	assetScores[2] = 1
	assetScores[3] = 9
	assetScores[4] = 3
	assetsByScore := getHighestScore(assetScores)
	require.Equal(t, assetsByScore.Value, assetScores[3])
}

func TestEvalAssetSuitability(t *testing.T) {
	// pluto_4.2.0_darwin_amd64.tar.gz = 9
	// ruplacer-osx = 6
	// croc_9.2.0_macOS-64bit.tar.gz = 7
	// conftest_0.28.1_Darwin_x86_64.tar.gz = 7
	// conftest_0.28.1_Darwin_arm64.tar.gz = 6
	// pandoc-2.14.2-macOS.pkg = 6
	// direnv.darwin-amd64 =8
	asset := types.Asset{
		BrowserDownloadURL: "https://github.com/foo/bar/releases/download/v1.0/direnv.darwin-arm64",
	}
	require.Equal(t, 7, evaluateAssetSuitability(asset))
	// pluto_4.2.0_darwin_amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pluto_4.2.0_darwin_arm64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(asset))
	// ruplacer-osx
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/ruplacer-osx"
	require.Equal(t, 5, evaluateAssetSuitability(asset))
	// croc_9.2.0_macOS-64bit.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/croc_9.2.0_macOS-64bit.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(asset))
	// conftest_0.28.1_Darwin_x86_64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Darwin_x86_64.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(asset))
	// conftest_0.28.1_Darwin_arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Darwin_arm64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(asset))
	// pandoc-2.14.2-macOS.pkg
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pandoc-2.14.2-macOS.pkg"
	require.Equal(t, 6, evaluateAssetSuitability(asset))

}
