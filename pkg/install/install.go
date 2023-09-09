package install

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/types"
	"crhuber/kelp/pkg/utils"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mholt/archiver/v3"
	"github.com/schollz/progressbar/v3"
)

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

func Install(owner, repo, release string) error {
	// handle http packages
	if strings.HasPrefix(release, "http") {
		urlsplit := strings.SplitAfter(release, "/")
		filename := urlsplit[len(urlsplit)-1]
		downloadPath := filepath.Join(config.KelpCache, filename)
		tempdir, _ := os.MkdirTemp("", "kelp")
		downloadFile(downloadPath, release)
		err := extractPackage(downloadPath, tempdir)
		if err != nil {
			return err
		}
		installBinary(tempdir)
		os.RemoveAll(tempdir)

	} else {
		assets, err := downloadGithubRelease(owner, repo, release)
		if err != nil {
			return err
		}
		for _, asset := range assets {
			downloadPath := filepath.Join(config.KelpCache, asset.Name)
			tempdir, err := os.MkdirTemp("", "kelp")
			if err != nil {
				return err
			}
			err = extractPackage(downloadPath, tempdir)
			if err != nil {
				return err
			}
			installBinary(tempdir)
			os.RemoveAll(tempdir)
			// only install first asset if there are multiple
			break
		}
	}
	return nil
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

func extractPackage(downloadPath, tempDir string) error {
	fmt.Printf("\nExtracting %s", downloadPath)
	reader, err := os.Open(downloadPath)
	if err != nil {
		return errors.New("could read archive")
	}
	defer reader.Close()
	if strings.HasSuffix(downloadPath, ".tar.gz") {
		err = archiver.Unarchive(downloadPath, tempDir)
		if err != nil {
			return err
		}
		return nil
	}

	if strings.HasSuffix(downloadPath, ".bz2") {
		err = archiver.Unarchive(downloadPath, tempDir)
		if err != nil {
			return err
		}
		return nil
	}
	if strings.HasSuffix(downloadPath, ".tgz") {
		err := utils.Untar(tempDir, reader)
		if err != nil {
			return err
		}
		return nil
	}

	if strings.HasSuffix(downloadPath, ".xz") {
		err := utils.Unxz(tempDir, reader)
		if err != nil {
			return err
		}
		return nil
	}
	if strings.HasSuffix(downloadPath, ".gz") {
		err := utils.Untar(tempDir, reader)
		if err != nil {
			return err
		}
		return nil
	}
	if strings.HasSuffix(downloadPath, ".zip") {
		_, err := utils.Unzip(downloadPath, tempDir)
		if err != nil {
			return err
		}
		return nil
	}
	if strings.HasSuffix(downloadPath, ".dmg") {
		fmt.Println("\nSkippping dmg..")
		return errors.New("kelp does not support dmg files")
	}
	// sometimes there is no unzip file and its just the file
	fp := strings.SplitAfter(downloadPath, "/")
	fn := fp[len(fp)-1]
	if !strings.Contains(fn, ".") {
		fmt.Println("\nFound unextractable file. Installing instead")
		installBinary(downloadPath)
		return nil
	}
	return errors.New("archive file format not known")
}

func installBinary(tempDir string) {
	fmt.Println("\nChecking for binary files in extract...")
	files, err := utils.FilePathWalkDir(tempDir)
	if err != nil {
		log.Panic("Could not walk directory")
	}
	for _, file := range files {
		mime, _ := mimetype.DetectFile(string(file))
		// only install binary files
		if mime.String() == "application/x-mach-binary" {
			splits := strings.SplitAfter(file, "/")
			fileName := splits[len(splits)-1]
			fmt.Printf("\nBinary file %s found in extract.", fileName)
			destination := filepath.Join(config.KelpBin, fileName)
			fmt.Printf("\nCopying %v to kelp bin...", fileName)
			utils.CopyFile(file, destination)
			fmt.Printf("\n✅ Installed %v !", fileName)
		}
	}
}

func findGithubReleaseMacAssets(assets []types.Asset) []types.Asset {

	fmt.Println("\nFinding mac assets to download...")
	var downloadableAssets []types.Asset
	assetScores := map[int]int{}
	for index, asset := range assets {
		filename := strings.Split(asset.BrowserDownloadURL, "/")
		assetScore := 0
		// scoring //
		// direnv.darwin-amd64 = 7
		// pluto_4.2.0_darwin_amd64.tar.gz = 9
		// ruplacer-osx = 6
		// croc_9.2.0_macOS-64bit.tar.gz = 7
		// conftest_0.28.1_Darwin_x86_64.tar.gz = 7
		// conftest_0.28.1_Darwin_arm64.tar.gz = 6
		// pandoc-2.14.2-macOS.pkg = 6
		if asset.IsMacAsset() {
			assetScore += 4
		}
		if asset.IsSameArchitecture() {
			assetScore += 3
		}
		if asset.IsDownloadableExtension() {
			assetScore += 2
		}
		if asset.HasNoExtension() {
			assetScore += 1
		}

		if assetScore >= 6 {
			fmt.Printf("\nFound suitable candiate %v for download. Score: %v", filename[len(filename)-1], assetScore)
			assetScores[index] = assetScore
		}

	}
	if len(assetScores) > 0 {
		// sort the map by value of score.
		assetsByScore := make(PairList, len(assetScores))
		i := 0
		for k, v := range assetScores {
			assetsByScore[i] = Pair{k, v}
			i++
		}
		sort.Sort(assetsByScore)
		highest := assetsByScore[len(assetsByScore)-1]
		bestAsset := assets[highest.Key]
		filename := strings.Split(bestAsset.BrowserDownloadURL, "/")
		fmt.Printf("\nAdding highest ranked asset %v to download queue.", filename[len(filename)-1])
		downloadableAssets = append(downloadableAssets, bestAsset)
	}
	return downloadableAssets
}

func downloadGithubRelease(owner, repo, release string) ([]types.Asset, error) {
	fmt.Printf("\n===> Installing %s/%s:%s ...", owner, repo, release)
	ghr, err := utils.GetGithubRelease(owner, repo, release)
	if err != nil {
		return nil, err
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
		err := errors.New("could not find a github asset with mac binaries")
		return downloadableAssets, err
	}
	return downloadableAssets, nil
}
