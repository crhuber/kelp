package install

import (
	"fmt"
	"io"
	iofs "io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/types"
	"crhuber/kelp/pkg/utils"

	"github.com/mholt/archives"
)

func TestCleanBinaryName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"direnv.darwin-arm64", "direnv"},
		{"direnv.darwin-amd64", "direnv"},
		{"direnv.linux-arm64", "direnv"},
		{"direnv.linux-amd64", "direnv"},
		{"tool.darwin_arm64", "tool"},
		{"tool.linux_amd64", "tool"},
		{"tool.arm64-darwin", "tool"},
		{"tool.amd64-linux", "tool"},
		{"tool.macos-arm64", "tool"},
		{"tool.windows-x86_64", "tool"},
		{"tool.linux-aarch64", "tool"},
		{"tool.darwin-x64", "tool"},
		{"talosctl-linux-amd64", "talosctl"},
		{"talosctl-linux-arm64", "talosctl"},
		{"talosctl-darwin-arm64", "talosctl"},
		{"talosctl-darwin-amd64", "talosctl"},
		{"talosctl-freebsd-amd64", "talosctl"},
		{"talosctl-linux-armv7", "talosctl"},
		{"talosctl-linux-riscv64", "talosctl"},
		{"talosctl-windows-amd64.exe", "talosctl.exe"},
		{"talosctl_linux_amd64", "talosctl"},
		{"hadolint-linux-x86_64", "hadolint"},
		{"hadolint-Linux-x86_64", "hadolint"},
		{"tool-darwin-universal", "tool"},
		{"tool_amd64_linux", "tool"},
		{"tool.linux.amd64", "tool"},
		// No matching suffix — returned as-is
		{"mybinary", "mybinary"},
		{"archive.tar.gz", "archive.tar.gz"},
		{"tool.v1.2.3", "tool.v1.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanBinaryName(tt.input)
			if got != tt.want {
				t.Errorf("cleanBinaryName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

type dummyFileInfo struct {
	name  string
	isDir bool
	mode  os.FileMode
}

func (d dummyFileInfo) Name() string       { return d.name }
func (d dummyFileInfo) Size() int64        { return 4 }
func (d dummyFileInfo) Mode() os.FileMode  { return d.mode }
func (d dummyFileInfo) ModTime() time.Time { return time.Now() }
func (d dummyFileInfo) IsDir() bool        { return d.isDir }
func (d dummyFileInfo) Sys() any           { return nil }

type dummyFile struct {
	io.Reader
	info dummyFileInfo
}

func (d dummyFile) Read(p []byte) (int, error)   { return d.Reader.Read(p) }
func (d dummyFile) Close() error                 { return nil }
func (d dummyFile) Stat() (iofs.FileInfo, error) { return d.info, nil }

func TestExtractFilePathTraversal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-test-extract-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	maliciousPaths := []string{
		"../evil.sh",
		"../../evil.sh",
		"../../../etc/passwd",
		"sub/../../evil.sh",
		"sub/../../../evil.sh",
	}

	for _, malPath := range maliciousPaths {
		t.Run(malPath, func(t *testing.T) {
			info := dummyFileInfo{name: "evil", mode: 0o755}
			af := archives.FileInfo{
				FileInfo:      info,
				NameInArchive: malPath,
				Open: func() (iofs.File, error) {
					return dummyFile{Reader: strings.NewReader("evil"), info: info}, nil
				},
			}
			err := extractFile(af, tempDir)
			if err == nil {
				t.Errorf("expected error for path traversal %q, got nil", malPath)
			}
			if !strings.Contains(err.Error(), "escapes destination directory") {
				t.Errorf("expected 'escapes destination directory' error, got %v", err)
			}
		})
	}

	// Verify safe file extraction succeeds
	t.Run("safe file", func(t *testing.T) {
		info := dummyFileInfo{name: "safe.txt", mode: 0o644}
		af := archives.FileInfo{
			FileInfo:      info,
			NameInArchive: "sub/safe.txt",
			Open: func() (iofs.File, error) {
				return dummyFile{Reader: strings.NewReader("hello"), info: info}, nil
			},
		}
		err := extractFile(af, tempDir)
		if err != nil {
			t.Fatalf("unexpected error for safe file: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tempDir, "sub", "safe.txt"))
		if err != nil {
			t.Fatalf("could not read extracted safe file: %v", err)
		}
		if string(content) != "hello" {
			t.Errorf("content = %q, want 'hello'", string(content))
		}

		subDirInfo, err := os.Stat(filepath.Join(tempDir, "sub"))
		if err != nil {
			t.Fatalf("could not stat extracted dir: %v", err)
		}
		if subDirInfo.Mode().Perm()&0o007 != 0 {
			t.Errorf("extracted directory has world permissions: %o", subDirInfo.Mode().Perm())
		}
	})

	// Verify directory entries extracted with 0o750
	t.Run("directory entry in archive", func(t *testing.T) {
		dirInfo := dummyFileInfo{name: "mydir", isDir: true, mode: 0o777}
		af := archives.FileInfo{
			FileInfo:      dirInfo,
			NameInArchive: "mydir",
		}
		err := extractFile(af, tempDir)
		if err != nil {
			t.Fatalf("unexpected error for dir entry: %v", err)
		}
		extractedDir, err := os.Stat(filepath.Join(tempDir, "mydir"))
		if err != nil {
			t.Fatalf("could not stat extracted dir: %v", err)
		}
		if extractedDir.Mode().Perm()&0o007 != 0 {
			t.Errorf("extracted directory has world permissions: %o", extractedDir.Mode().Perm())
		}
	})
}

func TestExtractFileDecompressionBomb(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-test-bomb-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origLimit := maxExtractFileSize
	defer func() { maxExtractFileSize = origLimit }()
	maxExtractFileSize = 50 // Limit to 50 bytes for test

	// Create a stream that produces 100 bytes (exceeds limit of 50)
	payload := strings.Repeat("A", 100)
	info := dummyFileInfo{name: "bomb.bin", mode: 0o644}
	af := archives.FileInfo{
		FileInfo:      info,
		NameInArchive: "bomb.bin",
		Open: func() (iofs.File, error) {
			return dummyFile{Reader: strings.NewReader(payload), info: info}, nil
		},
	}

	err = extractFile(af, tempDir)
	if err == nil {
		t.Fatal("expected error for file exceeding size limit, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum allowed extraction size") {
		t.Errorf("expected size limit error, got: %v", err)
	}

	// Verify the file was cleaned up and does not remain on disk
	extractedPath := filepath.Join(tempDir, "bomb.bin")
	if _, err := os.Stat(extractedPath); !os.IsNotExist(err) {
		t.Errorf("expected file %s to be removed after exceeding size limit", extractedPath)
	}
}

func TestVerifyGithubReleaseChecksumFromBody(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-checksum-body-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "helm-v4.0.4-linux-amd64.tar.gz")
	err = os.WriteFile(testFilePath, []byte("fake-helm-binary-content\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	actualHash, err := utils.ComputeSHA256(testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	ghr := &types.GithubRelease{
		Body: fmt.Sprintf("- [Linux amd64](...) ([checksum](...) / %s)", actualHash),
	}

	// 1. Success matching body
	err = verifyGithubReleaseChecksum(ghr, "helm-v4.0.4-linux-amd64.tar.gz", testFilePath)
	if err != nil {
		t.Fatalf("expected checksum verification to pass, got: %v", err)
	}

	// 2. Mismatch in body
	ghrMismatch := &types.GithubRelease{
		Body: "- [Linux amd64](...) ([checksum](...) / 0000000000000000000000000000000000000000000000000000000000000000)",
	}
	err = verifyGithubReleaseChecksum(ghrMismatch, "helm-v4.0.4-linux-amd64.tar.gz", testFilePath)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}

	// 3. No checksum present in release (graceful fallback)
	ghrEmpty := &types.GithubRelease{}
	err = verifyGithubReleaseChecksum(ghrEmpty, "helm-v4.0.4-linux-amd64.tar.gz", testFilePath)
	if err != nil {
		t.Fatalf("expected nil when no checksum available, got: %v", err)
	}
}

func TestVerifyChecksumFetchFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/missing/"):
			http.NotFound(w, r)
		default:
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	testFilePath := filepath.Join(t.TempDir(), "tool.tar.gz")
	if err := os.WriteFile(testFilePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Advertised asset-specific checksum that fails to download must error
	ghr := &types.GithubRelease{Assets: []types.Asset{
		{Name: "tool.tar.gz.sha256", BrowserDownloadURL: srv.URL + "/error/tool.tar.gz.sha256"},
	}}
	if err := verifyGithubReleaseChecksum(ghr, "tool.tar.gz", testFilePath); err == nil {
		t.Error("expected error when asset-specific checksum fetch fails")
	}

	// Advertised bundle checksum that fails to download must error
	ghr = &types.GithubRelease{Assets: []types.Asset{
		{Name: "checksums.txt", BrowserDownloadURL: srv.URL + "/error/checksums.txt"},
	}}
	if err := verifyGithubReleaseChecksum(ghr, "tool.tar.gz", testFilePath); err == nil {
		t.Error("expected error when bundle checksum fetch fails")
	}

	// HTTP release: server error must abort
	if err := verifyHTTPChecksum(srv.URL+"/error/tool.tar.gz", "tool.tar.gz", testFilePath); err == nil {
		t.Error("expected error when HTTP checksum fetch fails")
	}

	// HTTP release: 404 means no checksum published, skip gracefully
	if err := verifyHTTPChecksum(srv.URL+"/missing/tool.tar.gz", "tool.tar.gz", testFilePath); err != nil {
		t.Errorf("expected nil when no HTTP checksum is published, got: %v", err)
	}
}

func TestCopyToKelpBin(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-copy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origBin := config.KelpBin
	defer func() { config.KelpBin = origBin }()
	config.KelpBin = filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(config.KelpBin, 0o755); err != nil {
		t.Fatal(err)
	}

	sourceDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Valid binary with os-arch suffix gets sanitized and copied
	srcFile := filepath.Join(sourceDir, "kubectl-linux-amd64")
	if err := os.WriteFile(srcFile, []byte("echo kubectl"), 0o755); err != nil {
		t.Fatal(err)
	}

	dest := copyToKelpBin(srcFile)
	expectedDest := filepath.Join(config.KelpBin, "kubectl")
	if dest != expectedDest {
		t.Fatalf("expected dest %q, got %q", expectedDest, dest)
	}
	if _, err := os.Stat(expectedDest); os.IsNotExist(err) {
		t.Fatalf("expected binary at %s to exist", expectedDest)
	}

	// 2. Directory should return empty string
	gotDir := copyToKelpBin(sourceDir)
	if gotDir != "" {
		t.Errorf("expected empty string for directory, got %q", gotDir)
	}

	// 3. Nonexistent file should return empty string
	gotNonexistent := copyToKelpBin(filepath.Join(sourceDir, "nonexistent"))
	if gotNonexistent != "" {
		t.Errorf("expected empty string for nonexistent file, got %q", gotNonexistent)
	}
}

func TestInstallInvalidURLAndCacheEscape(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-cache-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origCache := config.KelpCache
	defer func() { config.KelpCache = origCache }()
	config.KelpCache = filepath.Join(tempDir, "cache")
	if err := os.MkdirAll(config.KelpCache, 0o750); err != nil {
		t.Fatal(err)
	}

	// 1. URL without filename should fail
	err = Install("owner", "repo", "http://example.com/")
	if err == nil {
		t.Error("expected error for URL without filename, got nil")
	}

	// 2. Traversal in owner should fail
	err = Install("../../../escape", "repo", "http://example.com/tool.tar.gz")
	if err == nil {
		t.Error("expected error for escaping cache directory, got nil")
	}
}
