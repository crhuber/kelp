package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializePermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-init-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origDir := KelpDir
	origCache := KelpCache
	origBin := KelpBin
	defer func() {
		KelpDir = origDir
		KelpCache = origCache
		KelpBin = origBin
	}()

	KelpDir = filepath.Join(tempDir, ".kelp")
	KelpCache = filepath.Join(tempDir, ".kelp", "cache")
	KelpBin = filepath.Join(tempDir, ".kelp", "bin")
	configPath := filepath.Join(KelpDir, "kelp.json")

	err = Initialize(configPath)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	for _, dir := range []string{KelpDir, KelpCache, KelpBin} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("failed to stat %s: %v", dir, err)
		}
		// Must not be world-writable (perm & 0002 == 0)
		if info.Mode().Perm()&0002 != 0 {
			t.Errorf("directory %s is world-writable: %v", dir, info.Mode().Perm())
		}
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/path/to/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error loading nonexistent config file, got nil")
	}
}

