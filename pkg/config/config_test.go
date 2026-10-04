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

	for _, dir := range []string{KelpDir, KelpCache} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("failed to stat %s: %v", dir, err)
		}
		// KelpDir and KelpCache must not give other users any permissions (perm & 0007 == 0)
		if info.Mode().Perm()&0007 != 0 {
			t.Errorf("directory %s has permissions for other users: %v", dir, info.Mode().Perm())
		}
	}

	binInfo, err := os.Stat(KelpBin)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", KelpBin, err)
	}
	// KelpBin must not be world-writable
	if binInfo.Mode().Perm()&0002 != 0 {
		t.Errorf("directory %s is world-writable: %v", KelpBin, binInfo.Mode().Perm())
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/path/to/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error loading nonexistent config file, got nil")
	}
}

func TestValidateRepoName(t *testing.T) {
	tests := []struct {
		owner   string
		repo    string
		wantErr bool
	}{
		{"helm", "helm", false},
		{"kubernetes", "kubectl", false},
		{"cilium", "cilium-cli", false},
		{"my-org", "my_repo.v2", false},
		{"123org", "456repo", false},
		// Invalid cases:
		{"", "helm", true},
		{"helm", "", true},
		{"-invalid", "helm", true},
		{"helm", "-invalid", true},
		{"..", "helm", true},
		{"helm", "..", true},
		{".", "helm", true},
		{"helm", ".", true},
		{"owner/extra", "repo", true},
		{"owner", "repo/extra", true},
		{"owner;rm", "repo", true},
		{"owner", "repo&calc", true},
		{"owner space", "repo", true},
		{"--flag", "repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.owner+"/"+tt.repo, func(t *testing.T) {
			err := ValidateRepoName(tt.owner, tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepoName(%q, %q) error = %v, wantErr %v", tt.owner, tt.repo, err, tt.wantErr)
			}
		})
	}
}

func TestBrowseValidation(t *testing.T) {
	err := Browse("-bad-flag", "repo")
	if err == nil {
		t.Error("expected error for invalid owner, got nil")
	}
	err = Browse("owner", "../traversal")
	if err == nil {
		t.Error("expected error for invalid repo, got nil")
	}
}

func TestAddPackageValidation(t *testing.T) {
	kc := KelpConfig{}
	err := kc.AddPackage("..", "evil", "v1.0")
	if err == nil {
		t.Error("expected error for invalid owner in AddPackage, got nil")
	}
	err = kc.AddPackage("valid", "../evil", "v1.0")
	if err == nil {
		t.Error("expected error for invalid repo in AddPackage, got nil")
	}
	err = kc.AddPackage("valid-owner", "valid-repo", "v1.0")
	if err != nil {
		t.Errorf("unexpected error for valid package: %v", err)
	}
}



