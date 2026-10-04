package install

import (
	"context"
	"crypto/sha256"
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
	tempdir, err := os.MkdirTemp("", "kelp-*")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempdir)

	var downloadPath string

	if utils.IsKubectl(owner, repo, release) && !strings.HasPrefix(release, "http") {
		release = utils.GetKubectlDownloadURL(release)
	}

	if strings.HasPrefix(release, "http") {
		urlsplit := strings.SplitAfter(release, "/")
		filename := urlsplit[len(urlsplit)-1]
		urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(release)))[:12]
		downloadDir := filepath.Join(config.KelpCache, owner, repo, urlHash)
		if err := os.MkdirAll(downloadDir, 0o750); err != nil {
			return err
		}
		downloadPath = filepath.Join(downloadDir, filename)
		if utils.FileExists(downloadPath) {
			logging.LogDebug("File %v already exists in cache, skipping download.\n", filename)
		} else {
			err = downloadFile(downloadPath, release)
			if err != nil {
				return err
			}
		}
		if err := verifyHTTPChecksum(release, filename, downloadPath); err != nil {
			_ = os.Remove(downloadPath)
			return err
		}
	} else {
		downloadPath, err = downloadGithubRelease(owner, repo, release)
		if err != nil {
			return err
		}
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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %s: %w", url, err)
	}
	// set headers for github auth only if the URL points to GitHub
	if utils.IsGitHubURL(url) {
		if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
			logging.LogDebug("Using Github token in http request")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
		}
		req.Header.Set("Accept", "application/octet-stream")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if timeout := utils.GetHTTPTimeout(); timeout > 0 {
		transport.ResponseHeaderTimeout = timeout
	}
	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Do(req)
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
		if chmodErr := os.Chmod(destPath, 0o755); chmodErr != nil {
			return fmt.Errorf("could not make binary executable: %w", chmodErr)
		}
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
// For example, "direnv.darwin-arm64" becomes "direnv",
// and "talosctl-linux-amd64" becomes "talosctl".
func cleanBinaryName(name string) string {
	ext := ""
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".exe") {
		ext = ".exe"
		name = name[:len(name)-len(".exe")]
		lower = lower[:len(lower)-len(".exe")]
	}

	osNames := []string{"darwin", "linux", "macos", "osx", "windows", "freebsd", "openbsd", "netbsd"}
	archNames := []string{
		"arm64", "aarch64", "amd64", "x86_64", "x64",
		"armv7", "armv6", "arm", "riscv64", "universal", "all",
		"386", "i386",
	}
	seps1 := []string{".", "-", "_"}
	seps2 := []string{"-", "_", "."}

	for _, osName := range osNames {
		for _, arch := range archNames {
			for _, sep1 := range seps1 {
				for _, sep2 := range seps2 {
					// os-arch: direnv.darwin-arm64, talosctl-linux-amd64
					suffix := sep1 + osName + sep2 + arch
					if strings.HasSuffix(lower, suffix) {
						return name[:len(name)-len(suffix)] + ext
					}
					// arch-os: direnv.arm64-darwin, tool-amd64-linux
					suffix = sep1 + arch + sep2 + osName
					if strings.HasSuffix(lower, suffix) {
						return name[:len(name)-len(suffix)] + ext
					}
				}
			}
		}
	}
	return name + ext
}

// maxExtractFileSize defines the maximum allowed size of an extracted file (default 1GB).
var maxExtractFileSize int64 = 1 << 30

// Helper function to extract a single file safely
func extractFile(f archives.FileInfo, destDir string) error {
	cleanDest := filepath.Clean(destDir)
	extractPath := filepath.Join(cleanDest, f.NameInArchive)
	cleanExtract := filepath.Clean(extractPath)

	rel, err := filepath.Rel(cleanDest, cleanExtract)
	if err != nil || strings.HasPrefix(rel, "..") || (rel == "." && !f.IsDir()) {
		return fmt.Errorf("illegal file path in archive: %s escapes destination directory", f.NameInArchive)
	}

	if f.IsDir() {
		return os.MkdirAll(cleanExtract, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(cleanExtract), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Remove any existing file/link to prevent symlink traversal
	_ = os.Remove(cleanExtract)

	mode := f.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	// Strip setuid and setgid bits
	mode = mode & 0o777

	outFile, err := os.OpenFile(cleanExtract, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Limit reader to prevent decompression bombs from exhausting disk space
	written, err := io.Copy(outFile, io.LimitReader(rc, maxExtractFileSize+1))
	if err != nil {
		return err
	}
	if written > maxExtractFileSize {
		_ = outFile.Close()
		_ = os.Remove(cleanExtract)
		return fmt.Errorf("file %s exceeds maximum allowed extraction size (%d bytes)", f.NameInArchive, maxExtractFileSize)
	}
	return nil
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
			if dest := copyToKelpBin(file); dest != "" {
				destinations = append(destinations, dest)
			}
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
			if dest := copyToKelpBin(foundLibs[0]); dest != "" {
				destinations = append(destinations, dest)
			}
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
				if dest := copyToKelpBin(filteredLibs[0]); dest != "" {
					destinations = append(destinations, dest)
				}
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
	fileName := cleanBinaryName(splits[len(splits)-1])
	logging.LogDebug("Binary file %s found in extract.\n", fileName)
	destination := filepath.Join(config.KelpBin, fileName)
	logging.LogInfo("💾 Copying %v to kelp bin...\n", fileName)
	if err := utils.CopyFile(file, destination); err != nil {
		logging.LogInfo("❌ Failed to copy %v to %v: %v\n", fileName, destination, err)
		return ""
	}
	logging.LogInfo("✅ Installed %v !\n", fileName)
	return destination
}

func downloadGithubRelease(owner, repo, release string) (string, error) {
	logging.LogInfo("===> Installing %s/%s:%s...\n", owner, repo, release)
	ghr, err := utils.GetGithubRelease(owner, repo, release)
	if err != nil {
		return "", err
	}

	logging.LogInfo("🍏 Finding assets to download...")
	downloadableAsset, err := ghr.FindBestAsset(types.GetCapabilities())
	if err != nil {
		return "", err
	}

	safeRelease := strings.ReplaceAll(release, "/", "_")
	downloadDir := filepath.Join(config.KelpCache, owner, repo, safeRelease)
	if err := os.MkdirAll(downloadDir, 0o750); err != nil {
		return "", err
	}
	downloadPath := filepath.Join(downloadDir, downloadableAsset.Name)
	if utils.FileExists(downloadPath) {
		logging.LogDebug("File %v already exists in cache, skipping download.\n", downloadableAsset.Name)
	} else {
		err := downloadFile(downloadPath, downloadableAsset.URL)
		if err != nil {
			return "", err
		}
	}

	if err := verifyGithubReleaseChecksum(ghr, downloadableAsset.Name, downloadPath); err != nil {
		_ = os.Remove(downloadPath)
		return "", err
	}

	return downloadPath, nil
}

func verifyGithubReleaseChecksum(ghr *types.GithubRelease, targetAsset, downloadPath string) error {
	var checksumContent string

	// 1. Look for asset-specific checksum file (e.g. targetAsset + ".sha256", targetAsset + ".sha256sum")
	for _, asset := range ghr.Assets {
		if asset.Name == targetAsset+".sha256" ||
			asset.Name == targetAsset+".sha256sum" ||
			asset.Name == targetAsset+".sha256.txt" {
			url := asset.URL
			if url == "" {
				url = asset.BrowserDownloadURL
			}
			content, err := utils.FetchURLText(url)
			if err == nil && content != "" {
				checksumContent = content
				break
			}
		}
	}

	// 2. Look for bundle checksum files (e.g. checksums.txt, SHA256SUMS, etc.)
	if checksumContent == "" {
		for _, asset := range ghr.Assets {
			lowerName := strings.ToLower(asset.Name)
			if lowerName == "checksums.txt" ||
				lowerName == "sha256sums" ||
				lowerName == "sha256sums.txt" ||
				strings.HasSuffix(lowerName, "checksums.txt") ||
				strings.HasSuffix(lowerName, "sha256sums.txt") {
				url := asset.URL
				if url == "" {
					url = asset.BrowserDownloadURL
				}
				content, err := utils.FetchURLText(url)
				if err == nil && content != "" {
					checksumContent = content
					break
				}
			}
		}
	}

	// 3. Fallback: check release body
	if checksumContent == "" && ghr.Body != "" {
		checksumContent = ghr.Body
	}

	if checksumContent == "" {
		logging.LogDebug("No SHA-256 checksum found for %s, skipping verification.", targetAsset)
		return nil
	}

	expectedHash, found := utils.ParseSHA256FromManifest(checksumContent, targetAsset)
	if !found {
		logging.LogDebug("No matching SHA-256 hash found for %s in release checksums, skipping verification.", targetAsset)
		return nil
	}

	logging.LogInfo("🔍 Verifying SHA-256 checksum for %s...\n", targetAsset)
	if err := utils.VerifyFileSHA256(downloadPath, expectedHash); err != nil {
		return err
	}
	logging.LogInfo("🔒 Verified SHA-256 checksum: %s\n", expectedHash)
	return nil
}

func verifyHTTPChecksum(releaseURL, filename, downloadPath string) error {
	// For dl.k8s.io or other HTTP releases that publish .sha256
	checksumURL := releaseURL + ".sha256"
	content, err := utils.FetchURLText(checksumURL)
	if err != nil {
		content, err = utils.FetchURLText(releaseURL + ".sha256sum")
	}
	if err != nil || content == "" {
		logging.LogDebug("No HTTP checksum found for %s, skipping verification.", filename)
		return nil
	}

	expectedHash, found := utils.ParseSHA256FromManifest(content, filename)
	if !found {
		logging.LogDebug("No matching SHA-256 hash found for %s at %s.", filename, checksumURL)
		return nil
	}

	logging.LogInfo("🔍 Verifying SHA-256 checksum for %s...\n", filename)
	if err := utils.VerifyFileSHA256(downloadPath, expectedHash); err != nil {
		return err
	}
	logging.LogInfo("🔒 Verified SHA-256 checksum: %s\n", expectedHash)
	return nil
}

