package types

import (
	"strings"
	"time"
)

// Asset does stuff
type Asset struct {
	URL                string    `json:"url"`
	ID                 int       `json:"id"`
	NodeID             string    `json:"node_id"`
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

// GithubRelease does stuff
type GithubRelease struct {
	URL             string `json:"url"`
	AssetsURL       string `json:"assets_url"`
	UploadURL       string `json:"upload_url"`
	HTMLURL         string `json:"html_url"`
	ID              int    `json:"id"`
	NodeID          string `json:"node_id"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Name            string `json:"name"`
	Draft           bool   `json:"draft"`
	Author          struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
	TarballURL  string    `json:"tarball_url"`
	ZipballURL  string    `json:"zipball_url"`
	Body        string    `json:"body"`
}

// methods

func (a Asset) IsDownloadableExtension() bool {
	downLoadableExtension := []string{".zip", ".tar", ".gz", ".xz", ".dmg", ".pkg", ".tgz", ".bz2"}
	for _, word := range downLoadableExtension {
		result := strings.HasSuffix(a.BrowserDownloadURL, word)
		if result {
			return result
		}
	}
	return false
}

func (a Asset) HasNoExtension() bool {
	bdu := strings.SplitAfter(a.BrowserDownloadURL, "/")
	filename := bdu[len(bdu)-1]
	return !strings.Contains(filename, ".")
}

// IsMacAsset checks if the download url contains "mac", "macos", "darwin", "osx", "apple" and returns true if so
func (a Asset) IsMacAsset() bool {
	macIdentifiers := []string{"mac", "macos", "darwin", "osx", "apple"}

	for _, word := range macIdentifiers {
		result := strings.Contains(strings.ToLower(a.BrowserDownloadURL), word)
		if result {
			return result
		}
	}
	return false
}

func (a Asset) IsLinuxAsset() bool {
	macIdentifiers := []string{"linux"}

	for _, word := range macIdentifiers {
		result := strings.Contains(strings.ToLower(a.BrowserDownloadURL), word)
		if result {
			return result
		}
	}
	return false
}

func (a Asset) IsSameOS(capabilities *Capabilities) bool {
	switch capabilities.OS {
	case Darwin:
		return a.IsMacAsset()
	case Linux:
		return a.IsLinuxAsset()
	}
	return false
}

func (a Asset) IsSameArchitecture(capabilities *Capabilities) bool {
	if strings.Contains(strings.ToLower(a.BrowserDownloadURL), strings.ToLower(capabilities.Arch)) {
		return true
	} else if capabilities.Arch == "amd64" && strings.Contains(strings.ToLower(a.BrowserDownloadURL), "x86_64") {
		return true
	} else if capabilities.Arch == "arm64" && strings.Contains(strings.ToLower(a.BrowserDownloadURL), "arm64") {
		return true
	} else if capabilities.Arch == "arm64" && strings.Contains(strings.ToLower(a.BrowserDownloadURL), "aarch64") {
		return true
	} else {
		return false
	}
}
