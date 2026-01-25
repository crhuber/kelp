package install

import (
	"context"
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/logging"
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
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mholt/archives"
	"github.com/schollz/progressbar/v3"
)

func Install(owner, repo, release string) error {
	// handle http packages
	tempdir, _ := os.MkdirTemp("", "kelp")
	defer os.RemoveAll(tempdir)

	var downloadPath string

	if strings.HasPrefix(release, "http") {
		urlsplit := strings.SplitAfter(release, "/")
		filename := urlsplit[len(urlsplit)-1]
		downloadPath = filepath.Join(config.KelpCache, filename)
		err := downloadFile(downloadPath, release)
		if err != nil {
			return err
		}
	} else {
		asset, err := downloadGithubRelease(owner, repo, release)
		if err != nil {
			return err
		}
		downloadPath = filepath.Join(config.KelpCache, asset.Name)
	}
	err := extractPackage(downloadPath, tempdir)
	if err != nil {
		return err
	}
	destinations := installBinary(tempdir)
	if types.IsDarwin() {
		for _, d := range destinations {
			unquarantineFile(d)
		}
	}
	return nil
}

func unquarantineFile(filepath string) error {
	logging.LogInfo("🛃 Unquarantining %s...\n", filepath)
	cmd := exec.Command("xattr", "-d", "com.apple.quarantine", filepath)
	return cmd.Run()
}

// downloadFile downloads files
func downloadFile(filepath string, url string) error {
	logging.LogInfo("===> Downloading %s...\n", url)
	logging.LogDebug("To: %s...\n", filepath)

	// Get the data
	req, _ := http.NewRequest("GET", url, nil)
	// set headers for github auth
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		logging.LogDebug("Using Github token in http request")
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
	return err
}

func extractPackage(downloadPath, tempDir string) error {
	logging.LogInfo("📂 Extracting %s\n", downloadPath)

	// Handle dmg files
	if strings.HasSuffix(downloadPath, ".dmg") {
		logging.LogDebug("Skipping dmg..")
		return errors.New("kelp does not support dmg files")
	}

	// Check if it's a file without extension (binary)
	fp := strings.SplitAfter(downloadPath, "/")
	fn := fp[len(fp)-1]
	if !strings.Contains(fn, ".") {
		logging.LogDebug("Found unextractable file. Installing instead")
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
	logging.LogInfo("🧐 Checking for binary files in extract...")
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
			logging.LogDebug("Binary file %s found in extract.\n", fileName)
			destination := filepath.Join(config.KelpBin, fileName)
			logging.LogInfo("💾 Copying %v to kelp bin...\n", fileName)
			utils.CopyFile(file, destination)
			logging.LogInfo("✅ Installed %v !\n", fileName)
			destinations = append(destinations, destination)
		} else {
			logging.LogDebug("Skipping non executable file: %v - %v\n", file, mime.String())
		}
	}
	return destinations
}

func downloadGithubRelease(owner, repo, release string) (*types.Asset, error) {
	logging.LogInfo("===> Installing %s/%s:%s...\n", owner, repo, release)
	ghr, err := utils.GetGithubRelease(owner, repo, release)
	if err != nil {
		return nil, err
	}

	logging.LogInfo("🍏 Finding assets to download...")
	downloadableAsset, err := ghr.FindBestAsset(types.GetCapabilities())
	if err != nil {
		return nil, err
	}

	downloadPath := filepath.Join(config.KelpCache, downloadableAsset.Name)
	if utils.FileExists(downloadPath) {
		logging.LogDebug("File %v already exists in cache, skipping download.\n", downloadableAsset.Name)
	} else {
		err := downloadFile(downloadPath, downloadableAsset.URL)
		if err != nil {
			return nil, err
		}
	}

	return downloadableAsset, nil
}
