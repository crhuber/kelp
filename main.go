package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mholt/archiver/v3"
)

var githubToken string
var githubUsername string
var home, err = os.UserHomeDir()

var kelpDir = filepath.Join(home, "/.kelp/")
var kelpBin = filepath.Join(home, "/.kelp/bin/")
var kelpCache = filepath.Join(home, "/.kelp/cache/")
var kelpConf = filepath.Join(home, "/.kelp/kelp.json")

type kelpConfig []kelpPackage

type kelpPackage struct {
	Owner   string `json:"Owner"`
	Repo    string `json:"Repo"`
	Release string `json:"Release"`
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
	fmt.Printf("\nExtracting %s ", downloadPath)
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
		fmt.Println("\nFound unextractable file. Installing instead")
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
	fmt.Println("\nChecking for binary files...")
	files, err := filePathWalkDir(tempDir)
	if err != nil {
		log.Panic("Could not walk directory")
	}
	for _, file := range files {
		mime, _ := mimetype.DetectFile(string(file))
		// only install binary files
		if mime.String() == "application/x-mach-binary" {
			splits := strings.SplitAfter(file, "/")
			destination := filepath.Join(home, kelpBin, splits[len(splits)-1])
			fmt.Printf("\nInstalling %v ...", splits[len(splits)-1])
			copyFile(file, destination)
			fmt.Printf("\n✅ Installed %v !", splits[len(splits)-1])
		}
	}
}

// downloadFile downloads files
func downloadFile(filepath string, url string) error {
	fmt.Printf("\nDownloading %s ...", url)

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

func (kp kelpPackage) saveToConfig() {
	kc := loadKelpConfig()
	var repoExists bool
	for _, c := range kc {
		if kp.Repo == c.Repo {
			repoExists = true
		}
	}
	if repoExists == false {
		kc = append(kc, kp)
		bs, err := json.MarshalIndent(kc, "", " ")
		if err != nil {
			fmt.Println(bs)
		}
		ioutil.WriteFile(kelpConf, bs, 0644)
	}
}

func loadKelpConfig() kelpConfig {
	bs, _ := ioutil.ReadFile(kelpConf)
	var kc kelpConfig
	err := json.Unmarshal(bs, &kc)
	if err != nil {
		fmt.Println(err)
	}
	return kc
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
		fmt.Println("\nGetting releases by tag ...")
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
		home, _ := os.UserHomeDir()
		downloadPath := filepath.Join(home, kelpCache, da.Name)
		if fileExists(downloadPath) {
			fmt.Printf("\nFile %v already exists in cache, skipping download.", da.Name)
		} else {
			downloadFile(downloadPath, da.BrowserDownloadURL)
		}

	}
	return downloadableAssets, nil
}

func install(owner, repo, release string) int {
	// handle http packages
	if strings.HasPrefix(release, "http") {
		urlsplit := strings.SplitAfter(release, "/")
		filename := urlsplit[len(urlsplit)-1]
		downloadPath := filepath.Join(home, kelpCache, filename)
		tempdir, _ := ioutil.TempDir("", "kelp")
		downloadFile(downloadPath, release)
		extractPackage(downloadPath, tempdir)
		installBinary(tempdir)
		os.RemoveAll(tempdir)
		kp := kelpPackage{
			Owner:   owner,
			Repo:    repo,
			Release: release,
		}
		kp.saveToConfig()

	} else {
		assets, err := downloadGithubRelease(owner, repo, release)
		if err == nil {
			for _, asset := range assets {
				downloadPath := filepath.Join(home, kelpCache, asset.Name)
				tempdir, err := ioutil.TempDir("", "kelp")
				if err != nil {
					log.Panic("No temp dir")
				}
				extractPackage(downloadPath, tempdir)
				installBinary(tempdir)
				os.RemoveAll(tempdir)
				kp := kelpPackage{
					Owner:   owner,
					Repo:    repo,
					Release: release,
				}
				kp.saveToConfig()
			}
		} else {
			fmt.Printf("Error: %s", err)
		}

	}

	return 0
}

func update(repo string) {
	kc := loadKelpConfig()
	for _, kp := range kc {
		if repo == kp.Repo {
			// Just install the latest
			install(kp.Owner, kp.Repo, "latest")
		}
	}

}

func list() {
	fmt.Println("Installed Packages: ")
	kc := loadKelpConfig()
	for _, kp := range kc {
		fmt.Printf("\n%s/%s:%s", kp.Owner, kp.Repo, kp.Release)
	}
}

func initialize() {
	if dirExists(kelpDir) == false {
		fmt.Println("Creating Kelp dir...")
		err := os.Mkdir(kelpDir, 0777)
		if err != nil {
			fmt.Println(err)
		}
	}

	if dirExists(kelpCache) == false {
		fmt.Println("Creating Kelp cache...")
		os.Mkdir(kelpCache, 0777)

	}

	if dirExists(kelpBin) == false {
		fmt.Println("Creating Kelp bin...")
		os.Mkdir(kelpBin, 0777)
	}

	// create empty config
	if fileExists(kelpConf) == false {
		var kp kelpPackage
		kp.Owner = "crhuber"
		kp.Repo = "kelp"
		kp.Release = "latest"
		var kc kelpConfig
		kc2 := append(kc, kp)
		bs, err := json.MarshalIndent(kc2, "", " ")
		if err != nil {
			fmt.Println(bs)
		}
		ioutil.WriteFile(kelpConf, bs, 0644)
	}

	fmt.Println("🌱 Kelp Initialized!")
	fmt.Printf("🗒  Add Kelp to your path by running: \nexport PATH=%s:$PATH", kelpBin)
}

func inspect() {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", kelpDir).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func browse(pkg string) {
	var err error
	kc := loadKelpConfig()
	var url string
	for _, kp := range kc {
		if pkg == kp.Repo {
			url = fmt.Sprintf("https://github.com/%s/%s/releases", kp.Owner, kp.Repo)
		}
	}
	if url == "" {
		fmt.Println("Could not find package installed")
		os.Exit(1)
	}
	fmt.Printf("Opening %s", url)

	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	Cli()
}
