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
	KelpCache = filepath.Join(tempDir, ".cache", "kelp")
	KelpBin = filepath.Join(tempDir, ".kelp", "bin")
	configPath := filepath.Join(tempDir, ".config", "kelp", "kelp.json")

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

func TestXdgDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/abs/cache")
	if got := xdgDir("XDG_CACHE_HOME", ".cache"); got != "/abs/cache" {
		t.Errorf("absolute env: got %s", got)
	}
	t.Setenv("XDG_CACHE_HOME", "relative")
	if got := xdgDir("XDG_CACHE_HOME", ".cache"); got != filepath.Join(home, ".cache") {
		t.Errorf("relative env must be ignored: got %s", got)
	}
	t.Setenv("XDG_CACHE_HOME", "")
	if got := xdgDir("XDG_CACHE_HOME", ".cache"); got != filepath.Join(home, ".cache") {
		t.Errorf("unset env: got %s", got)
	}
}

func TestDefaultConfLegacyFallback(t *testing.T) {
	tempDir := t.TempDir()
	origConf, origLegacy := KelpConf, legacyConf
	defer func() { KelpConf, legacyConf = origConf, origLegacy }()

	KelpConf = filepath.Join(tempDir, "config", "kelp.json")
	legacyConf = filepath.Join(tempDir, "legacy.json")
	if got := DefaultConf(); got != KelpConf {
		t.Errorf("no configs: got %s, want %s", got, KelpConf)
	}

	if err := os.WriteFile(legacyConf, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := DefaultConf(); got != legacyConf {
		t.Errorf("only legacy: got %s, want %s", got, legacyConf)
	}

	if err := os.MkdirAll(filepath.Dir(KelpConf), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(KelpConf, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := DefaultConf(); got != KelpConf {
		t.Errorf("both exist: got %s, want %s", got, KelpConf)
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

func TestSetPackageValidation(t *testing.T) {
	kc := KelpConfig{
		Packages: []KelpPackage{
			{Owner: "testowner", Repo: "testrepo", Release: "v1.0"},
		},
	}

	// 1. Valid binary name
	err := kc.SetPackage("testrepo", "v2.0", "new desc", "mybinary")
	if err != nil {
		t.Fatalf("expected valid set, got: %v", err)
	}
	if kc.Packages[0].Binary != "mybinary" {
		t.Errorf("expected binary to be 'mybinary', got %q", kc.Packages[0].Binary)
	}

	// 2. Traversal or path in binary name
	invalidBinaries := []string{
		"../escape",
		"../../etc/shadow",
		"/usr/bin/tool",
		"sub/tool",
		"sub\\tool",
		".",
		"..",
	}
	for _, inv := range invalidBinaries {
		err := kc.SetPackage("testrepo", "", "", inv)
		if err == nil {
			t.Errorf("expected error for invalid binary %q, got nil", inv)
		}
	}

	// 3. Nonexistent package returns error
	err = kc.SetPackage("nonexistent", "v1.0", "", "tool")
	if err == nil {
		t.Error("expected error for nonexistent package, got nil")
	}
}
