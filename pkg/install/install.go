package install

import (
	"context"
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/types"
	"crhuber/kelp/pkg/utils"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mholt/archives"
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
		destinations := installBinary(tempdir)
		if types.IsDarwin() {
			for _, d := range destinations {
				unquarantineFile(d)
			}
		}
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
		destinations := installBinary(tempdir)
		if runtime.GOOS == "darwin" {
			for _, d := range destinations {
				unquarantineFile(d)
			}
		}
		os.RemoveAll(tempdir)

	}
	return nil
}

func unquarantineFile(filepath string) error {
	fmt.Printf("🛃 Unquarantining %s...\n", filepath)
	cmd := exec.Command("xattr", "-d", "com.apple.quarantine", filepath)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// downloadFile downloads files
func downloadFile(filepath string, url string) error {
	fmt.Printf("===> Downloading %s...\n", url)
	fmt.Printf("To: %s...\n", filepath)

	// Get the data
	req, _ := http.NewRequest("GET", url, nil)
	// set headers for github auth
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken != "" {
		fmt.Println("Using Github token in http request")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
	}
	req.Header.Set("Accept", "application/octet-stream")
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
	fmt.Printf("📂 Extracting %s\n", downloadPath)

	// Handle dmg files
	if strings.HasSuffix(downloadPath, ".dmg") {
		fmt.Println("Skipping dmg..")
		return errors.New("kelp does not support dmg files")
	}

	// Check if it's a file without extension (binary)
	fp := strings.SplitAfter(downloadPath, "/")
	fn := fp[len(fp)-1]
	if !strings.Contains(fn, ".") {
		fmt.Println("Found unextractable file. Installing instead")
		installBinary(downloadPath)
		return nil
	}

	// Open the file
	file, err := os.Open(downloadPath)
	if err != nil {
		return fmt.Errorf("could not open archive: %w", err)
	}
	defer file.Close()

	// Use the correct Identify signature with context
	ctx := context.Background()
	format, stream, err := archives.Identify(ctx, downloadPath, file)
	if err != nil {
		return fmt.Errorf("could not identify archive format: %w", err)
	}

	// Check if the format supports extraction
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return fmt.Errorf("archive format does not support extraction")
	}

	// Extract all files to destination directory
	err = extractor.Extract(ctx, stream, func(_ context.Context, f archives.FileInfo) error {
		return extractFile(f, tempDir)
	})

	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	return nil
}

// Helper function to extract a single file
func extractFile(f archives.FileInfo, destDir string) error {
	extractPath := filepath.Join(destDir, f.NameInArchive)

	if f.IsDir() {
		return os.MkdirAll(extractPath, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(extractPath), 0755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(extractPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

func installBinary(tempDir string) []string {
	fmt.Println("🧐 Checking for binary files in extract...")
	files, err := utils.FilePathWalkDir(tempDir)
	if err != nil {
		log.Panic("Could not walk directory")
	}
	destinations := []string{}
	osCap := types.GetCapabilities()
	for _, file := range files {
		mime, _ := mimetype.DetectFile(string(file))
		// only install binary files
		if mime.String() == osCap.ExecutableMime {
			splits := strings.SplitAfter(file, "/")
			fileName := splits[len(splits)-1]
			fmt.Printf("Binary file %s found in extract.\n", fileName)
			destination := filepath.Join(config.KelpBin, fileName)
			fmt.Printf("💾 Copying %v to kelp bin...\n", fileName)
			utils.CopyFile(file, destination)
			fmt.Printf("✅ Installed %v !\n", fileName)
			destinations = append(destinations, destination)
		} else {
			fmt.Printf("Skipping non executable file: %v - %v\n", file, mime.String())
		}
	}
	return destinations
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

func evaluateAssetSuitability(capabilities *types.Capabilities, asset types.Asset) int {
	assetScore := 0
	if asset.IsSameOS(capabilities) {
		assetScore += 4
	}
	if asset.IsSameArchitecture(capabilities) {
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

	fmt.Println("🍏 Finding assets to download...")
	assetScores := map[int]int{}
	for index, asset := range assets {
		filename := strings.Split(asset.BrowserDownloadURL, "/")
		assetScore := evaluateAssetSuitability(types.GetCapabilities(), asset)
		if assetScore >= 6 {
			fmt.Printf("Found suitable candidate %v for download. Score: %v\n", filename[len(filename)-1], assetScore)
			assetScores[index] = assetScore
		}

	}
	if len(assetScores) == 0 {
		return types.Asset{}, errors.New("could not find a github asset")
	}

	// sort the map by value of score.
	highest := getHighestScore(assetScores)
	bestAsset := assets[highest.Key]
	filename := strings.Split(bestAsset.BrowserDownloadURL, "/")
	fmt.Printf("Adding highest ranked asset %v to download queue.\n", filename[len(filename)-1])
	return bestAsset, nil
}

func downloadGithubRelease(owner, repo, release string) (types.Asset, error) {
	fmt.Printf("===> Installing %s/%s:%s...\n", owner, repo, release)
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
		fmt.Printf("File %v already exists in cache, skipping download.\n", downloadableAsset.Name)
	} else {
		err := downloadFile(downloadPath, downloadableAsset.URL)
		if err != nil {
			return types.Asset{}, err
		}
	}

	return downloadableAsset, nil
}
