package rm

import (
	"crhuber/kelp/pkg/config"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveBinary(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-rm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origBin := config.KelpBin
	defer func() { config.KelpBin = origBin }()
	config.KelpBin = filepath.Join(tempDir, "bin")
	err = os.MkdirAll(config.KelpBin, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test binary
	binPath := filepath.Join(config.KelpBin, "mytool")
	err = os.WriteFile(binPath, []byte("echo hello"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Safe removal of existing binary
	err = RemoveBinary("mytool")
	if err != nil {
		t.Fatalf("expected successful removal, got: %v", err)
	}
	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Errorf("expected binary %s to be removed", binPath)
	}

	// 2. Removal of non-existent binary returns nil
	err = RemoveBinary("nonexistent")
	if err != nil {
		t.Fatalf("expected nil for nonexistent binary, got: %v", err)
	}

	// 3. Traversal attempts should be rejected
	traversalCases := []string{
		"../outside",
		"../../etc/hosts",
		"sub/../../outside",
		".",
	}

	for _, mal := range traversalCases {
		t.Run(mal, func(t *testing.T) {
			err := RemoveBinary(mal)
			if err == nil {
				t.Errorf("expected error for traversal %q, got nil", mal)
			}
		})
	}
}
