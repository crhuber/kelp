package install

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mholt/archiver/v3"
	"github.com/schollz/progressbar/v3"
)

func InstallAll() {
	fmt.Println("\nInstalling all packages in config")
	kc := config.LoadKelpConfig()
	for _, kp := range kc {
		// fmt.Printf("\n Installing: %s/%s:%s", kp.Owner, kp.Repo, kp.Release)
		Install(kp.Owner, kp.Repo, kp.Release)
	}
}

func Install(owner, repo, release string) int {
	// handle http packages
	if strings.HasPrefix(release, "http") {
		urlsplit := strings.SplitAfter(release, "/")
		filename := urlsplit[len(urlsplit)-1]
		downloadPath := filepath.Join(config.KelpCache, filename)
		tempdir, _ := ioutil.TempDir("", "kelp")
		downloadFile(downloadPath, release)
		extractPackage(downloadPath, tempdir)
		installBinary(tempdir)
		os.RemoveAll(tempdir)

	} else {
		assets, err := downloadGithubRelease(owner, repo, release)
		if err == nil {
			for _, asset := range assets {
				downloadPath := filepath.Join(config.KelpCache, asset.Name)
				tempdir, err := ioutil.TempDir("", "kelp")
				if err != nil {
					log.Panic("No temp dir")
				}
				extractPackage(downloadPath, tempdir)
				installBinary(tempdir)
				os.RemoveAll(tempdir)
				// only install first asset if there are multiple
				break
			}
		} else {
			fmt.Printf("\nError: %s", err)
		}

	}

	return 0
}

// downloadFile downloads files
func downloadFile(filepath string, url string) error {
	fmt.Printf("\nDownloading %s ...", url)
	fmt.Printf("\nTo: %s ... \n", filepath)

	// Get the data
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
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
	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading",
	)
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	return err
}

func extractPackage(downloadPath, tempDir string) {
	fmt.Printf("\nExtracting %s", downloadPath)
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
		utils.Untar(tempDir, reader)
	}
	if strings.HasSuffix(downloadPath, ".xz") {
		err := utils.Unxz(tempDir, reader)
		if err != nil {
			fmt.Println(err)
		}
	}
	if strings.HasSuffix(downloadPath, ".gz") {
		utils.Untar(tempDir, reader)
	}
	if strings.HasSuffix(downloadPath, ".zip") {
		utils.Unzip(downloadPath, tempDir)
	}
	if strings.HasSuffix(downloadPath, ".dmg") {
		fmt.Println("\nSkippping dmg..")
	}
	// sometimes there is no unzip file and its just the file
	fp := strings.SplitAfter(downloadPath, "/")
	fn := fp[len(fp)-1]
	if !strings.Contains(fn, ".") {
		fmt.Println("\nFound unextractable file. Installing instead")
		installBinary(downloadPath)
	}

}

func installBinary(tempDir string) {
	fmt.Println("\nChecking for binary files...")
	files, err := utils.FilePathWalkDir(tempDir)
	if err != nil {
		log.Panic("Could not walk directory")
	}
	for _, file := range files {
		mime, _ := mimetype.DetectFile(string(file))
		// only install binary files
		if mime.String() == "application/x-mach-binary" {
			fmt.Println("Binary file found in extract.")
			splits := strings.SplitAfter(file, "/")
			destination := filepath.Join(config.KelpBin, splits[len(splits)-1])
			fmt.Printf("\nCopying %v to kelp bin...", splits[len(splits)-1])
			utils.CopyFile(file, destination)
			fmt.Printf("\n✅ Installed %v !", splits[len(splits)-1])
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
	macIdentifiers := []string{"mac", "macos", "darwin", "osx", "apple"}

	for _, word := range macIdentifiers {
		result := strings.Contains(strings.ToLower(a.BrowserDownloadURL), word)
		if result == true {
			return result
		}
	}
	return false
}

func (a Asset) isSameArchitecture() bool {
	filename := strings.Split(a.BrowserDownloadURL, "/")
	fmt.Printf("\nComparing asset %s to system architecture (%s)...\n", filename[len(filename)-1], runtime.GOARCH)
	return strings.Contains(a.BrowserDownloadURL, runtime.GOARCH)
}

func findGithubReleaseMacAssets(assets []Asset) []Asset {
	fmt.Println("\nFinding mac assets to download...")
	var downloadableAssets []Asset
	for _, asset := range assets {

		// only download same architecture. this is an exact match
		if asset.isMacAsset() && asset.isDownloadableExtension() && asset.isSameArchitecture() {
			fmt.Println("Zipped mac asset with matching architecture, adding to downloads.")
			downloadableAssets = append(downloadableAssets, asset)
			break
		}

		if asset.isMacAsset() && asset.isDownloadableExtension() {
			fmt.Println("Zipped mac asset, adding to downloads.")
			downloadableAssets = append(downloadableAssets, asset)
		}
		// some files are not zipped and have no extension
		if asset.isMacAsset() && asset.hasNoExtension() {
			fmt.Println("Found a mac asset, adding to downloads.")
			downloadableAssets = append(downloadableAssets, asset)
		}

		// handles case like direnv.darwin-amd64
		if asset.isMacAsset() && asset.isSameArchitecture() {
			fmt.Println("Found a mac asset with matching architecture, adding to downloads.")
			downloadableAssets = append(downloadableAssets, asset)
		}
	}
	return downloadableAssets
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
	if resp.StatusCode != http.StatusOK {
		err := errors.New(string(body))
		return nil, err
	}
	ghr := GithubRelease{}

	if err := json.Unmarshal(body, &ghr); err != nil {
		panic(err)
	}
	downloadableAssets := findGithubReleaseMacAssets(ghr.Assets)

	for _, da := range downloadableAssets {
		downloadPath := filepath.Join(config.KelpCache, da.Name)
		if utils.FileExists(downloadPath) {
			fmt.Printf("\nFile %v already exists in cache, skipping download.", da.Name)
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
