package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mholt/archiver/v3"
)

func installAll() {
	fmt.Println("Installing all packages in config")
	kc := loadKelpConfig()
	for _, kp := range kc {
		// fmt.Printf("\n Installing: %s/%s:%s", kp.Owner, kp.Repo, kp.Release)
		install(kp.Owner, kp.Repo, kp.Release)
	}
}

func install(owner, repo, release string) int {
	// handle http packages
	if strings.HasPrefix(release, "http") {
		urlsplit := strings.SplitAfter(release, "/")
		filename := urlsplit[len(urlsplit)-1]
		downloadPath := filepath.Join(kelpCache, filename)
		tempdir, _ := ioutil.TempDir("", "kelp")
		downloadFile(downloadPath, release)
		extractPackage(downloadPath, tempdir)
		installBinary(tempdir)
		os.RemoveAll(tempdir)

	} else {
		assets, err := downloadGithubRelease(owner, repo, release)
		if err == nil {
			for _, asset := range assets {
				downloadPath := filepath.Join(kelpCache, asset.Name)
				tempdir, err := ioutil.TempDir("", "kelp")
				if err != nil {
					log.Panic("No temp dir")
				}
				extractPackage(downloadPath, tempdir)
				installBinary(tempdir)
				os.RemoveAll(tempdir)
			}
		} else {
			fmt.Printf("Error: %s", err)
		}

	}

	return 0
}

// downloadFile downloads files
func downloadFile(filepath string, url string) error {
	fmt.Printf("Downloading %s ... \n", url)
	fmt.Printf("Destination: %s \n", filepath)
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

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

func extractPackage(downloadPath, tempDir string) {
	fmt.Printf("Extracting %s \n", downloadPath)
	reader, err := os.Open(downloadPath)
	if err != nil {
		log.Fatal("Could not extract package")
	}
	defer reader.Close()
	if strings.HasSuffix(downloadPath, ".tar.gz") {
		err = archiver.Unarchive(downloadPath, tempDir)
		if err != nil {
			fmt.Println(err)
		}
	}

	if strings.HasSuffix(downloadPath, ".bz2") {
		err = archiver.Unarchive(downloadPath, tempDir)
		if err != nil {
			fmt.Println(err)
		}
	}
	if strings.HasSuffix(downloadPath, ".tgz") {
		Untar(tempDir, reader)
	}
	if strings.HasSuffix(downloadPath, ".xz") {
		err := Unxz(tempDir, reader)
		if err != nil {
			fmt.Println(err)
		}
	}
	if strings.HasSuffix(downloadPath, ".gz") {
		Untar(tempDir, reader)
	}
	if strings.HasSuffix(downloadPath, ".zip") {
		Unzip(downloadPath, tempDir)
	}
	if strings.HasSuffix(downloadPath, ".dmg") {
		fmt.Println("Skippping dmg..")
	}
	// sometimes there is no unzip file and its just the file
	fp := strings.SplitAfter(downloadPath, "/")
	fn := fp[len(fp)-1]
	if !strings.Contains(fn, ".") {
		fmt.Println("Found unextractable file. Installing instead")
		installBinary(downloadPath)
	}

}

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func copyFile(source, destination string) {
	from, err := os.Open(source)
	if err != nil {
		log.Fatal(err)
	}
	defer from.Close()

	to, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE, 0744)
	if err != nil {
		log.Fatal(err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		log.Fatal(err)
	}
}

func installBinary(tempDir string) {
	fmt.Println("Checking for binary files...")
	files, err := filePathWalkDir(tempDir)
	if err != nil {
		log.Panic("Could not walk directory")
	}
	for _, file := range files {
		mime, _ := mimetype.DetectFile(string(file))
		// only install binary files
		if mime.String() == "application/x-mach-binary" {
			fmt.Println("Binary file found in extract.")
			splits := strings.SplitAfter(file, "/")
			destination := filepath.Join(kelpBin, splits[len(splits)-1])
			fmt.Printf("Installing %v ... \n", splits[len(splits)-1])
			copyFile(file, destination)
			fmt.Printf("✅ Installed %v ! \n", splits[len(splits)-1])
		}
	}
}

// methods

func (a Asset) isDownloadableExtension() bool {
	downLoadableExtension := []string{".zip", ".tar", ".gz", ".xz", ".dmg", ".pkg", ".tgz", ".bz2"}
	for _, word := range downLoadableExtension {
		result := strings.HasSuffix(a.BrowserDownloadURL, word)
		if result == true {
			return result
		}
	}
	return false
}

func (a Asset) hasNoExtension() bool {
	bdu := strings.SplitAfter(a.BrowserDownloadURL, "/")
	filename := bdu[len(bdu)-1]
	return !strings.Contains(filename, ".")
}

func (a Asset) isMacAsset() bool {
	macIdentifiers := []string{"mac", "macOs", "macos", "darwin", "osx"}

	for _, word := range macIdentifiers {
		result := strings.Contains(a.BrowserDownloadURL, word)
		if result == true {
			return result
		}
	}
	return false
}

func findGithubReleaseMacAssets(assets []Asset) []Asset {
	fmt.Println("\nFinding mac assets to download...")
	var downloadableAssets []Asset
	for _, asset := range assets {
		if asset.isMacAsset() && asset.isDownloadableExtension() {
			downloadableAssets = append(downloadableAssets, asset)
		}
		// some files are not zipped and have no extension
		if asset.isMacAsset() && asset.hasNoExtension() {
			downloadableAssets = append(downloadableAssets, asset)
		}
	}
	return downloadableAssets
}

func dirExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func downloadGithubRelease(owner, repo, release string) ([]Asset, error) {
	fmt.Printf("\n===> Installing %s/%s:%s ...", owner, repo, release)
	fmt.Printf("\nFetching info about %s/%s:%s ...", owner, repo, release)
	var url string
	if release == "latest" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%s", owner, repo, release)

	} else {
		// try by tag
		fmt.Printf("\nGetting releases by tag %s...", release)
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, release)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ghr := GithubRelease{}

	if err := json.Unmarshal(body, &ghr); err != nil {
		panic(err)
	}
	downloadableAssets := findGithubReleaseMacAssets(ghr.Assets)

	for _, da := range downloadableAssets {
		downloadPath := filepath.Join(kelpCache, da.Name)
		if fileExists(downloadPath) {
			fmt.Printf("File %v already exists in cache, skipping download.\n", da.Name)
		} else {
			downloadFile(downloadPath, da.BrowserDownloadURL)
		}

	}
	if len(downloadableAssets) == 0 {
		err := errors.New("Could not find a github asset with mac binaries")
		return downloadableAssets, err
	}
	return downloadableAssets, nil
}
