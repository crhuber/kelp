package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/logging"
	"crhuber/kelp/pkg/types"
	"crhuber/kelp/pkg/utils"

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
		return errors.New("kelp does not support dmg files")
	}

	// Open the file
	file, err := os.Open(downloadPath)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	// Try to identify archive format
	ctx := context.Background()
	format, stream, err := archives.Identify(ctx, downloadPath, file)
	if err != nil {
		// Not a recognized archive — treat as raw binary
		logging.LogDebug("File is not a recognized archive format. Treating as raw binary.")
		cleanName := cleanBinaryName(filepath.Base(downloadPath))
		destPath := filepath.Join(tempDir, cleanName)
		if copyErr := utils.CopyFile(downloadPath, destPath); copyErr != nil {
			return fmt.Errorf("could not copy binary to temp dir: %w", copyErr)
		}
		os.Chmod(destPath, 0o755)
		return nil
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

// cleanBinaryName strips OS/arch suffixes from binary filenames.
// For example, "direnv.darwin-arm64" becomes "direnv".
func cleanBinaryName(name string) string {
	osNames := []string{"darwin", "linux", "macos", "windows"}
	archNames := []string{"arm64", "aarch64", "amd64", "x86_64", "x64"}
	lower := strings.ToLower(name)
	for _, osName := range osNames {
		for _, arch := range archNames {
			for _, sep := range []string{"-", "_"} {
				// os-arch: direnv.darwin-arm64
				suffix := "." + osName + sep + arch
				if strings.HasSuffix(lower, suffix) {
					return name[:len(name)-len(suffix)]
				}
				// arch-os: direnv.arm64-darwin
				suffix = "." + arch + sep + osName
				if strings.HasSuffix(lower, suffix) {
					return name[:len(name)-len(suffix)]
				}
			}
		}
	}
	return name
}

// Helper function to extract a single file
func extractFile(f archives.FileInfo, destDir string) error {
	extractPath := filepath.Join(destDir, f.NameInArchive)

	if f.IsDir() {
		return os.MkdirAll(extractPath, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(extractPath), 0o755); err != nil {
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
	var foundLibs []string
	for _, file := range files {
		mime, _ := mimetype.DetectFile(string(file))
		// only install binary files
		switch mime.String() {
		case osCap.ExecutableMime:
			destinations = append(destinations, copyToKelpBin(file))
		case osCap.SharedLibrary:
			splits := strings.SplitAfter(file, "/")
			fileName := splits[len(splits)-1]
			logging.LogDebug("Shared/Static Library file %s found in extract.\n", fileName)
			foundLibs = append(foundLibs, file)
		default:
			logging.LogDebug("Skipping non executable file: %v - %v\n", file, mime.String())
		}
	}
	if len(destinations) == 0 { // if no binary was found in extract, then filter shared libararies
		if len(foundLibs) == 1 {
			destinations = append(destinations, copyToKelpBin(foundLibs[0]))
		} else {
			var filteredLibs []string
			for _, currentLib := range foundLibs {
				if !strings.HasPrefix(currentLib, "lib") && !strings.HasSuffix(currentLib, "dynlib") {
					filteredLibs = append(filteredLibs, currentLib)
				} else {
					mime, _ := mimetype.DetectFile(string(currentLib))
					logging.LogDebug("Skipping non executable file: %v - %v\n", currentLib, mime.String())
				}
			}
			if len(filteredLibs) == 1 {
				destinations = append(destinations, copyToKelpBin(filteredLibs[0]))
			} else {
				for _, currentUnrecognizedLib := range filteredLibs {
					mime, _ := mimetype.DetectFile(string(currentUnrecognizedLib))
					logging.LogDebug("Skipping non executable file: %v - %v\n", currentUnrecognizedLib, mime.String())
				}
			}
		}
	}
	return destinations
}

func copyToKelpBin(file string) string {
	splits := strings.SplitAfter(file, "/")
	fileName := splits[len(splits)-1]
	logging.LogDebug("Binary file %s found in extract.\n", fileName)
	destination := filepath.Join(config.KelpBin, fileName)
	logging.LogInfo("💾 Copying %v to kelp bin...\n", fileName)
	utils.CopyFile(file, destination)
	logging.LogInfo("✅ Installed %v !\n", fileName)
	return destination
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
