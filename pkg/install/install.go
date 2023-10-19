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
		err := downloadFile(downloadPath, release)
		if err != nil {
			return err
		}
		err = extractPackage(downloadPath, tempdir)
		if err != nil {
			return err
		}
		installBinary(tempdir)
		os.RemoveAll(tempdir)

	} else {
		asset, err := downloadGithubRelease(owner, repo, release)
		if err != nil {
			return err
		}

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

	}
	return nil
}

// downloadFile downloads files
func downloadFile(filepath string, url string) error {
	fmt.Printf("\n===> Downloading %s ...", url)
	fmt.Printf("\nTo: %s ...", filepath)

	// Get the data
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("\ninvalid HTTP status: %v", resp.StatusCode)
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
	if err != nil {
		return err
	}
	return nil
}

func extractPackage(downloadPath, tempDir string) error {
	fmt.Printf("\n📂 Extracting %s", downloadPath)
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
	fmt.Println("\n🧐 Checking for binary files in extract...")
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
			fmt.Printf("\n💾 Copying %v to kelp bin...", fileName)
			utils.CopyFile(file, destination)
			fmt.Printf("\n✅ Installed %v !", fileName)
		}
	}
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

func evaluateAssetSuitability(asset types.Asset) int {
	assetScore := 0
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
	return assetScore

}

func findGithubReleaseMacAssets(assets []types.Asset) (types.Asset, error) {

	fmt.Println("\n🍏 Finding mac assets to download...")
	assetScores := map[int]int{}
	for index, asset := range assets {
		filename := strings.Split(asset.BrowserDownloadURL, "/")
		assetScore := evaluateAssetSuitability(asset)
		if assetScore >= 6 {
			fmt.Printf("\nFound suitable candiate %v for download. Score: %v", filename[len(filename)-1], assetScore)
			assetScores[index] = assetScore
		}

	}
	if len(assetScores) == 0 {
		return types.Asset{}, errors.New("could not find a github asset with mac binaries")
	}

	// sort the map by value of score.
	highest := getHighestScore(assetScores)
	bestAsset := assets[highest.Key]
	filename := strings.Split(bestAsset.BrowserDownloadURL, "/")
	fmt.Printf("\nAdding highest ranked asset %v to download queue.", filename[len(filename)-1])
	return bestAsset, nil
}

func downloadGithubRelease(owner, repo, release string) (types.Asset, error) {
	fmt.Printf("\n===> Installing %s/%s:%s ...", owner, repo, release)
	ghr, err := utils.GetGithubRelease(owner, repo, release)
	if err != nil {
		return types.Asset{}, err
	}
	downloadableAsset, err := findGithubReleaseMacAssets(ghr.Assets)
	if err != nil {
		return types.Asset{}, err
	}

	downloadPath := filepath.Join(config.KelpCache, downloadableAsset.Name)
	if utils.FileExists(downloadPath) {
		fmt.Printf("\nFile %v already exists in cache, skipping download.", downloadableAsset.Name)
	} else {
		err := downloadFile(downloadPath, downloadableAsset.BrowserDownloadURL)
		if err != nil {
			return types.Asset{}, err
		}
	}

	return downloadableAsset, nil
}
