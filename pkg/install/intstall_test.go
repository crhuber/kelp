package install

import (
	"crhuber/kelp/pkg/types"
	"testing"

	"github.com/stretchr/testify/require"
	"runtime"
)

func TestFindGithubReleaseMacAssets(t *testing.T) {

	var assets []types.Asset
	asset1 := types.Asset{
		BrowserDownloadURL: "https://github.com/trufflesecurity/trufflehog/releases/download/v3.60.1/trufflehog_3.60.1_" + runtime.GOOS + "_amd64.tar.gz",
	}
	asset2 := types.Asset{
		BrowserDownloadURL: "https://github.com/trufflesecurity/trufflehog/releases/download/v3.60.1/trufflehog_3.60.1_" + runtime.GOOS + "_arm64.tar.gz",
	}
	assets = append(assets, asset1, asset2)

	downloadableAssets, _ := findGithubReleaseMacAssets(assets)
	if runtime.GOOS == "arm64" {
		require.Equal(t, asset2, downloadableAssets)
	} else {
		require.Equal(t, asset1, downloadableAssets)
	}
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

func TestEvalAssetSuitabilityDarwin(t *testing.T) {
	// pluto_4.2.0_darwin_amd64.tar.gz = 9
	// ruplacer-osx = 6
	// croc_9.2.0_macOS-64bit.tar.gz = 7
	// conftest_0.28.1_Darwin_x86_64.tar.gz = 7
	// conftest_0.28.1_Darwin_arm64.tar.gz = 6
	// pandoc-2.14.2-macOS.pkg = 6
	// direnv.darwin-amd64 =8
	osCap := &types.Capabilities{
		OS:             types.Darwin,
		ExecutableMime: "application/x-mach-binary",
		Arch:           "arm64",
	}
	asset := types.Asset{
		BrowserDownloadURL: "https://github.com/foo/bar/releases/download/v1.0/direnv.darwin-arm64",
	}
	require.Equal(t, 7, evaluateAssetSuitability(osCap, asset))
	// pluto_4.2.0_darwin_amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pluto_4.2.0_darwin_arm64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(osCap, asset))
	// ruplacer-osx
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/ruplacer-osx"
	require.Equal(t, 5, evaluateAssetSuitability(osCap, asset))
	// croc_9.2.0_macOS-64bit.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/croc_9.2.0_macOS-64bit.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// conftest_0.28.1_Darwin_x86_64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Darwin_x86_64.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// conftest_0.28.1_Darwin_arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Darwin_arm64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(osCap, asset))
	// pandoc-2.14.2-macOS.pkg
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pandoc-2.14.2-macOS.pkg"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// gopass-1.15.11-darwin-amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-darwin-amd64.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// gopass-1.15.11-darwin-arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-darwin-arm64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(osCap, asset))
}

func TestEvalAssetSuitabilityLinux(t *testing.T) {
	// pluto_4.2.0_darwin_amd64.tar.gz = 9
	// ruplacer-osx = 6
	// croc_9.2.0_macOS-64bit.tar.gz = 7
	// conftest_0.28.1_Darwin_x86_64.tar.gz = 7
	// conftest_0.28.1_Darwin_arm64.tar.gz = 6
	// pandoc-2.14.2-macOS.pkg = 6
	// direnv.darwin-amd64 =8
	osCap := &types.Capabilities{
		OS:             types.Linux,
		ExecutableMime: "asdf",
		Arch:           "amd64",
	}
	asset := types.Asset{
		BrowserDownloadURL: "https://github.com/foo/bar/releases/download/v1.0/direnv.linux-amd64",
	}
	require.Equal(t, 7, evaluateAssetSuitability(osCap, asset))
	// pluto_4.2.0_darwin_amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pluto_4.2.0_linux_amd64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(osCap, asset))
	// ruplacer-osx
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/ruplacer-linux"
	require.Equal(t, 5, evaluateAssetSuitability(osCap, asset))
	// croc_9.2.0_macOS-64bit.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/croc_9.2.0_linuX-64bit.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// conftest_0.28.1_Darwin_x86_64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Linux_x86_64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(osCap, asset))
	// conftest_0.28.1_Darwin_arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Linux_arm64.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// pandoc-2.14.2-macOS.pkg
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pandoc-2.14.2-linux.pkg"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// gopass-1.15.11-darwin-amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-linux-arm64.tar.gz"
	require.Equal(t, 6, evaluateAssetSuitability(osCap, asset))
	// gopass-1.15.11-darwin-arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-linux-amd64.tar.gz"
	require.Equal(t, 9, evaluateAssetSuitability(osCap, asset))
}
