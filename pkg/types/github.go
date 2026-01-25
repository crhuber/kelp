package types

import (
	"crhuber/kelp/pkg/logging"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Asset represents a downloadable asset from a Github release
type Asset struct {
	URL                string    `json:"url"`
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Label              string    `json:"label"`
	ContentType        string    `json:"content_type"`
	State              string    `json:"state"`
	Size               int       `json:"size"`
	DownloadCount      int       `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadURL string    `json:"browser_download_url"`
}

// GithubRelease represents a Github release
type GithubRelease struct {
	URL             string    `json:"url"`
	AssetsURL       string    `json:"assets_url"`
	UploadURL       string    `json:"upload_url"`
	HTMLURL         string    `json:"html_url"`
	ID              int       `json:"id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []Asset   `json:"assets"`
	Body            string    `json:"body"`
}

// methods

func (a *Asset) isDownloadableExtension() bool {
	downLoadableExtension := []string{".zip", ".tar", ".gz", ".xz", ".dmg", ".pkg", ".tgz", ".bz2"}
	for _, word := range downLoadableExtension {
		result := strings.HasSuffix(a.BrowserDownloadURL, word)
		if result {
			return result
		}
	}
	return false
}

func (a *Asset) isChecksumFile() bool {
	checksumExtension := []string{".asc", ".sha256.asc", ".sha512.asc", ".sha256sum.asc", ".sha512sum.asc", ".sha1.asc", ".md5.asc"}
	for _, word := range checksumExtension {
		if strings.HasSuffix(a.BrowserDownloadURL, word) {
			return true
		}
	}
	return false
}

func (a *Asset) hasNoExtension() bool {
	bdu := strings.SplitAfter(a.BrowserDownloadURL, "/")
	filename := bdu[len(bdu)-1]
	return !strings.Contains(filename, ".")
}

// IsMacAsset checks if the download url contains "mac", "macos", "darwin", "osx", "apple" and returns true if so
func (a *Asset) isMacAsset() bool {
	macIdentifiers := []string{"mac", "macos", "darwin", "osx", "apple"}

	for _, word := range macIdentifiers {
		result := strings.Contains(strings.ToLower(a.BrowserDownloadURL), word)
		if result {
			return result
		}
	}
	return false
}

func (a *Asset) isLinuxAsset() bool {
	macIdentifiers := []string{"linux"}

	for _, word := range macIdentifiers {
		result := strings.Contains(strings.ToLower(a.BrowserDownloadURL), word)
		if result {
			return result
		}
	}
	return false
}

func (a *Asset) isSameOS(capabilities *Capabilities) bool {
	switch capabilities.OS {
	case Darwin:
		return a.isMacAsset()
	case Linux:
		return a.isLinuxAsset()
	}
	return false
}

func (a *Asset) isSameArchitecture(capabilities *Capabilities) bool {
	lowerURL := strings.ToLower(a.BrowserDownloadURL)

	// First check if the URL contains the exact arch name
	if strings.Contains(lowerURL, strings.ToLower(capabilities.Arch)) {
		return true
	}

	// Then handle architecture aliases
	switch capabilities.Arch {
	case "amd64":
		return strings.Contains(lowerURL, "x86_64")
	case "arm64":
		return strings.Contains(lowerURL, "arm64") || strings.Contains(lowerURL, "aarch64")
	default:
		return false
	}
}

const (
	MIN_ASSET_SCORE = 6 // minimum score for an asset to be considered suitable for download
)

func (a *Asset) EvaluateSuitability(capabilities *Capabilities) int {
	assetScore := 0
	if a.isSameOS(capabilities) {
		assetScore += 4
	}
	if a.isSameArchitecture(capabilities) {
		assetScore += 3
	}
	if a.isDownloadableExtension() {
		assetScore += 2
	}
	if a.hasNoExtension() {
		assetScore += 1
	}
	if a.isChecksumFile() {
		assetScore -= 10
	}
	return assetScore
}

func (a *Asset) RealFilename() string {
	if a.Name != "" {
		return a.Name
	}
	url := a.URL
	if url == "" {
		url = a.BrowserDownloadURL
	}
	filename := strings.Split(url, "/")
	return filename[len(filename)-1]
}

// A data structure to hold key/value pairs
type Pair struct {
	Key   int
	Value int
}

// A slice of pairs that implements sort.Interface to sort by values
type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }

func (ghr *GithubRelease) FindBestAsset(capabilities *Capabilities) (*Asset, error) {
	var bestAsset Asset

	assetScores := map[int]int{}
	for index, asset := range ghr.Assets {
		if assetScore := asset.EvaluateSuitability(capabilities); assetScore >= MIN_ASSET_SCORE {
			logging.LogDebug("Found suitable candidate %v for download. Score: %v", asset.RealFilename(), assetScore)
			assetScores[index] = assetScore
		}
	}
	if len(assetScores) == 0 {
		// inspect the release body for links to downloadable assets
		links := ghr.inspectLinksInReleaseBody()
		// create a list of assets from the links and evaluate them
		assetsFromBodyScores := map[int]int{}
		assetLinks := make([]Asset, len(links))
		for index, link := range links {
			filename := strings.Split(link, "/")
			realFilename := filename[len(filename)-1]
			a := Asset{
				BrowserDownloadURL: link,
				URL:                link,
				Name:               realFilename,
			}
			assetLinks[index] = a
			if assetScore := a.EvaluateSuitability(capabilities); assetScore >= MIN_ASSET_SCORE {
				logging.LogDebug("Found suitable candidate %v for download in release body. Score: %v", realFilename, assetScore)
				assetsFromBodyScores[index] = assetScore
			}
		}
		if len(assetsFromBodyScores) == 0 {
			return nil, errors.New("no suitable candidates found in release body")
		}
		// sort the map by value of score.
		highest := getHighestScore(assetsFromBodyScores)
		bestAsset = assetLinks[highest.Key]
	} else {
		// sort the map by value of score.
		highest := getHighestScore(assetScores)
		bestAsset = ghr.Assets[highest.Key]
	}

	logging.LogDebug("Adding highest ranked asset %v to download queue.", bestAsset.RealFilename())
	return &bestAsset, nil
}

func getHighestScore(assetScores map[int]int) Pair {
	// sort the map by value of score.
	assetsByScore := make(PairList, len(assetScores))
	i := 0
	for k, v := range assetScores {
		assetsByScore[i] = Pair{k, v}
		i++
	}
	sort.Sort(assetsByScore)
	// return highest
	return assetsByScore[len(assetsByScore)-1]
}

func (ghr *GithubRelease) inspectLinksInReleaseBody() []string {
	const (
		NAME_REGEXP      = `([a-z][a-z0-9_-]+?)`
		ARCH_REGEXP      = `[._-](amd64|x86_64|x64|arm64|aarch64)`
		OS_REGEXP        = `[._-]((unknown[._-])?(linux|linux-gnu|linux-musl))|((apple[._-])?(darwin|macos|osx))`
		VERSION_REGEXP   = `([_-]v?[0-9.]+)?`
		SUFFIX_REGEXP    = `([_-][a-z0-9_-]+)?`
		EXTENSION_REGEXP = `(\.zip|\.tar\.gz|\.gz|\.tgz|\.tar\.xz|\.txz|\.tar\.bz2|\.tbz)?`
		REGEXP           = NAME_REGEXP + VERSION_REGEXP + "(" + OS_REGEXP + ARCH_REGEXP + "|" + ARCH_REGEXP + OS_REGEXP + ")" + SUFFIX_REGEXP + EXTENSION_REGEXP
	)
	re := regexp.MustCompile(`https:\/\/[a-z0-9.\/]+\/` + REGEXP)
	matches := re.FindAllString(ghr.Body, -1)
	// remove duplicates
	sort.Strings(matches)
	// remove duplicates
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, item := range matches {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
