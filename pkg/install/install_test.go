package install

import (
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func (d dummyFile) Read(p []byte) (int, error) { return d.Reader.Read(p) }
func (d dummyFile) Close() error               { return nil }
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
	})
}

