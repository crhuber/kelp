package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalAssetSuitabilityDarwin(t *testing.T) {
	t.Parallel()
	// pluto_4.2.0_darwin_amd64.tar.gz = 9
	// ruplacer-osx = 6
	// croc_9.2.0_macOS-64bit.tar.gz = 7
	// conftest_0.28.1_Darwin_x86_64.tar.gz = 7
	// conftest_0.28.1_Darwin_arm64.tar.gz = 6
	// pandoc-2.14.2-macOS.pkg = 6
	// direnv.darwin-amd64 =8
	osCap := &Capabilities{
		OS:             Darwin,
		ExecutableMime: "application/x-mach-binary",
		Arch:           "arm64",
	}
	asset := Asset{
		BrowserDownloadURL: "https://github.com/foo/bar/releases/download/v1.0/direnv.darwin-arm64",
	}
	require.Equal(t, 7, asset.EvaluateSuitability(osCap))
	// pluto_4.2.0_darwin_amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pluto_4.2.0_darwin_arm64.tar.gz"
	require.Equal(t, 9, asset.EvaluateSuitability(osCap))
	// ruplacer-osx
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/ruplacer-osx"
	require.Equal(t, 5, asset.EvaluateSuitability(osCap))
	// croc_9.2.0_macOS-64bit.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/croc_9.2.0_macOS-64bit.tar.gz"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// conftest_0.28.1_Darwin_x86_64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Darwin_x86_64.tar.gz"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// conftest_0.28.1_Darwin_arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Darwin_arm64.tar.gz"
	require.Equal(t, 9, asset.EvaluateSuitability(osCap))
	// pandoc-2.14.2-macOS.pkg
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pandoc-2.14.2-macOS.pkg"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// gopass-1.15.11-darwin-amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-darwin-amd64.tar.gz"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// gopass-1.15.11-darwin-arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-darwin-arm64.tar.gz"
	require.Equal(t, 9, asset.EvaluateSuitability(osCap))
}

func TestEvalAssetSuitabilityLinux(t *testing.T) {
	t.Parallel()
	// pluto_4.2.0_darwin_amd64.tar.gz = 9
	// ruplacer-osx = 6
	// croc_9.2.0_macOS-64bit.tar.gz = 7
	// conftest_0.28.1_Darwin_x86_64.tar.gz = 7
	// conftest_0.28.1_Darwin_arm64.tar.gz = 6
	// pandoc-2.14.2-macOS.pkg = 6
	// direnv.darwin-amd64 =8
	// helm-v4.0.4-linux-arm64.tar.gz.asc <0
	osCap := &Capabilities{
		OS:             Linux,
		ExecutableMime: "asdf",
		Arch:           "amd64",
	}
	asset := Asset{
		BrowserDownloadURL: "https://github.com/foo/bar/releases/download/v1.0/direnv.linux-amd64",
	}
	require.Equal(t, 7, asset.EvaluateSuitability(osCap))
	// pluto_4.2.0_darwin_amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pluto_4.2.0_linux_amd64.tar.gz"
	require.Equal(t, 9, asset.EvaluateSuitability(osCap))
	// ruplacer-osx
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/ruplacer-linux"
	require.Equal(t, 5, asset.EvaluateSuitability(osCap))
	// croc_9.2.0_macOS-64bit.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/croc_9.2.0_linuX-64bit.tar.gz"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// conftest_0.28.1_Darwin_x86_64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Linux_x86_64.tar.gz"
	require.Equal(t, 9, asset.EvaluateSuitability(osCap))
	// conftest_0.28.1_Darwin_arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/conftest_0.28.1_Linux_arm64.tar.gz"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// pandoc-2.14.2-macOS.pkg
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/pandoc-2.14.2-linux.pkg"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// gopass-1.15.11-darwin-amd64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-linux-arm64.tar.gz"
	require.Equal(t, 6, asset.EvaluateSuitability(osCap))
	// gopass-1.15.11-darwin-arm64.tar.gz
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/gopass-1.15.11-linux-amd64.tar.gz"
	require.Equal(t, 9, asset.EvaluateSuitability(osCap))
	// helm-v4.0.4-linux-arm64.tar.gz.asc
	asset.BrowserDownloadURL = "https://github.com/foo/bar/releases/download/v1.0/helm-v4.0.4-linux-arm64.tar.gz.asc"
	require.Less(t, asset.EvaluateSuitability(osCap), 0)
}

func TestFindGithubReleaseMacAssets(t *testing.T) {
	t.Parallel()
	var assets []Asset
	asset1 := Asset{
		BrowserDownloadURL: "https://github.com/trufflesecurity/trufflehog/releases/download/v3.60.1/trufflehog_3.60.1_linux_amd64.tar.gz",
	}
	asset2 := Asset{
		BrowserDownloadURL: "https://github.com/trufflesecurity/trufflehog/releases/download/v3.60.1/trufflehog_3.60.1_linux_arm64.tar.gz",
	}
	assets = append(assets, asset1, asset2)

	ghr := GithubRelease{
		Assets: assets,
	}

	capAMD64 := &Capabilities{
		OS:   Linux,
		Arch: "amd64",
	}
	capARM64 := &Capabilities{
		OS:   Linux,
		Arch: "arm64",
	}
	downloadableAsset, _ := ghr.FindBestAsset(capAMD64)
	require.Equal(t, asset1, *downloadableAsset)

	downloadableAsset, _ = ghr.FindBestAsset(capARM64)
	require.Equal(t, asset2, *downloadableAsset)
}

func TestGetHighestScore(t *testing.T) {
	t.Parallel()
	assetScores := map[int]int{}
	assetScores[0] = 6
	assetScores[1] = 8
	assetScores[2] = 1
	assetScores[3] = 9
	assetScores[4] = 3
	assetsByScore := getHighestScore(assetScores)
	require.Equal(t, assetsByScore.Value, assetScores[3])
}

func TestInspectLinksInReleaseBody(t *testing.T) {
	t.Parallel()
	filename := filepath.Join("..", "..", "testdata", "helm-latest.json")
	jsonBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal("error reading testdata: ", err)
	}
	ghr := GithubRelease{}
	if err := json.Unmarshal(jsonBytes, &ghr); err != nil {
		t.Fatal("error unmarshalling testdata: ", err)
	}
	asset, err := ghr.FindBestAsset(&Capabilities{
		OS:   Linux,
		Arch: "amd64",
	})
	require.NoError(t, err)
	require.Equal(t, "helm-v4.0.4-linux-amd64.tar.gz", asset.Name)
	require.Contains(t, asset.BrowserDownloadURL, "get.helm.sh")
}
