package config

import (
	"crhuber/kelp/pkg/logging"
	"crhuber/kelp/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

var home, _ = os.UserHomeDir()

// xdgDir returns $env if it is an absolute path, else ~/fallback.
// Relative values are ignored per the XDG spec so paths never depend on the cwd.
func xdgDir(env, fallback string) string {
	if d := os.Getenv(env); filepath.IsAbs(d) {
		return d
	}
	return filepath.Join(home, fallback)
}

var KelpDir = filepath.Join(home, ".kelp")
var KelpBin = filepath.Join(KelpDir, "bin")
var KelpCache = filepath.Join(xdgDir("XDG_CACHE_HOME", ".cache"), "kelp")
var KelpConf = filepath.Join(xdgDir("XDG_CONFIG_HOME", ".config"), "kelp", "kelp.json")
var legacyConf = filepath.Join(KelpDir, "kelp.json")

// DefaultConf returns the XDG config path, falling back to the pre-XDG ~/.kelp/kelp.json
// when only that one exists so existing installs keep working.
func DefaultConf() string {
	if !utils.FileExists(KelpConf) && utils.FileExists(legacyConf) {
		logging.LogInfo("ℹ️  Using legacy config %s, move it to %s (and delete %s)", legacyConf, KelpConf, filepath.Join(KelpDir, "cache"))
		return legacyConf
	}
	return KelpConf
}

type KelpConfig struct {
	Path     string `json:"-"`
	Packages []KelpPackage
}
type KelpPackage struct {
	Owner       string    `json:"Owner"`
	Repo        string    `json:"Repo"`
	Release     string    `json:"Release"`
	UpdatedAt   time.Time `json:"UpdatedAt"`
	Description string    `json:"Description"`
	Binary      string    `json:"Binary"`
}

func (kc *KelpConfig) GetPackage(repo string) (*KelpPackage, error) {
	parts := strings.Split(repo, "/")
	// Check if there is an owner part since some projects have the same repo name
	// like cli
	if len(parts) > 1 {
		// If there is an owner, get the more specific project first
		for _, kp := range kc.Packages {
			if kp.Owner == parts[0] && kp.Repo == parts[1] {
				return &kp, nil
			}
		}
	} else {
		for _, kp := range kc.Packages {
			if kp.Repo == repo {
				return &kp, nil
			}
		}
	}
	return nil, errors.New("package not found in config, try adding it first")
}

func Load(path string) (*KelpConfig, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", path, err)
	}
	kc := KelpConfig{}
	err = json.Unmarshal(bs, &kc.Packages)
	if err != nil {
		return nil, err
	}
	kc.Path = path
	return &kc, nil
}

func (kc *KelpConfig) Save() error {
	bs, _ := json.MarshalIndent(kc.Packages, "", " ")
	err := os.WriteFile(kc.Path, bs, 0600)
	if err != nil {
		return err
	}
	logging.LogInfo("Config saved.")
	return nil
}

func (kc *KelpConfig) RemovePackage(repo string) error {
	for i, kp := range kc.Packages {
		if kp.Repo == repo {
			kc.Packages = slices.Delete(kc.Packages, i, i+1)
			logging.LogInfo("Package %s removed\n", repo)
			return nil
		}
	}
	return errors.New("package not found in config")
}

func (kc *KelpConfig) AddPackage(owner, repo, release string) error {
	if err := ValidateRepoName(owner, repo); err != nil {
		return err
	}

	for _, p := range kc.Packages {
		if p.Owner == owner && p.Repo == repo {
			return fmt.Errorf("package already exists in config")
		}
	}

	// append a new item
	kp := KelpPackage{
		Owner:     owner,
		Repo:      repo,
		Release:   release,
		UpdatedAt: time.Now(),
	}
	kc.Packages = append(kc.Packages, kp)
	logging.LogDebug("Config added for %s/%s", owner, repo)

	return nil
}

func (kc *KelpConfig) SetPackage(repo, release, description, binary string) error {
	if binary != "" {
		if filepath.Base(binary) != binary || strings.ContainsAny(binary, `/\`) || binary == "." || binary == ".." {
			return fmt.Errorf("invalid binary name %q: must be a plain filename without path components", binary)
		}
	}
	parts := strings.Split(repo, "/")
	for i, p := range kc.Packages {
		match := false
		if len(parts) > 1 {
			match = (p.Owner == parts[0] && p.Repo == parts[1])
		} else {
			match = (p.Repo == repo)
		}
		if match {
			if release != "" {
				kc.Packages[i].Release = release
				kc.Packages[i].UpdatedAt = time.Now()
			}
			if description != "" {
				kc.Packages[i].Description = description
			}
			if binary != "" {
				kc.Packages[i].Binary = binary
			}
			logging.LogDebug("Config set for %s", repo)
			return nil
		}
	}
	return errors.New("package not found in config")
}

func (kc *KelpConfig) List() {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

	// sort by date
	sort.Slice(kc.Packages, func(i, j int) bool {
		return kc.Packages[i].UpdatedAt.Before(kc.Packages[j].UpdatedAt)
	})

	for _, pkg := range kc.Packages {

		// Format the timestamp in a more human-friendly way
		humanFriendlyTimestamp := pkg.UpdatedAt.Format("Jan 2 2006")
		if humanFriendlyTimestamp == "Jan 1 0001" {
			humanFriendlyTimestamp = ""
		}
		release := ""
		if strings.HasPrefix(pkg.Release, "http") {
			if utils.IsKubectl(pkg.Owner, pkg.Repo, pkg.Release) {
				release = fmt.Sprintf("%s (https)", utils.GetKubectlVersion(pkg.Release))
			} else {
				// Define the regex pattern to extract version numbers
				pattern := `[/v-]([\d.]+)`
				// Compile the regex pattern
				re := regexp.MustCompile(pattern)
				match := re.FindStringSubmatch(pkg.Release)
				if len(match) > 1 {
					release = fmt.Sprintf("%s (https)", match[1])
				} else {
					release = "unknown (https)"
				}
			}
		} else {
			release = pkg.Release
		}

		fmt.Fprintf(w, "%s/%s\t%s\t%s\n", pkg.Owner, pkg.Repo, release, humanFriendlyTimestamp)
	}
	_ = w.Flush()
}

func Initialize(path string) error {
	for _, dir := range []string{KelpDir, filepath.Dir(path), KelpCache} {
		if !utils.DirExists(dir) {
			logging.LogDebug("Creating %s...", dir)
			err := os.MkdirAll(dir, 0750)
			if err != nil {
				return err
			}
		}
	}

	if !utils.DirExists(KelpBin) {
		logging.LogDebug("Creating Kelp bin...")
		err := os.MkdirAll(KelpBin, 0755)
		if err != nil {
			return err
		}
	}

	// create empty config
	kp := KelpPackage{
		Owner:       "crhuber",
		Repo:        "kelp",
		Release:     "latest",
		UpdatedAt:   time.Now(),
		Description: "Simple homebrew alternative",
	}
	kc := KelpConfig{
		Path:     path,
		Packages: []KelpPackage{kp},
	}

	if !utils.FileExists(path) {
		logging.LogDebug("Creating Kelp config file...")
		err := kc.Save()
		if err != nil {
			return err
		}
	} else {
		logging.LogDebug("Skipping Kelp config file creation since one alredy exists...")
	}

	logging.LogInfo("🌱 Kelp Initialized!")
	logging.LogInfo("🗒  Add Kelp to your path by running: \nexport PATH=%s:$PATH >> ~/.bash_profile\n", KelpBin)
	return nil
}

var validRepoIdentifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// ValidateRepoName checks that owner and repo start with alphanumeric characters
// and contain only safe characters ([a-zA-Z0-9_.-]).
func ValidateRepoName(owner, repo string) error {
	if !validRepoIdentifier.MatchString(owner) {
		return fmt.Errorf("invalid repository owner %q: must start with alphanumeric and contain only [a-zA-Z0-9_.-]", owner)
	}
	if !validRepoIdentifier.MatchString(repo) {
		return fmt.Errorf("invalid repository name %q: must start with alphanumeric and contain only [a-zA-Z0-9_.-]", repo)
	}
	return nil
}

// openPath opens a file, directory or URL with the platform's default handler.
func openPath(target string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target).Start()
	case "linux":
		return exec.Command("xdg-open", target).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func Inspect() error {
	return openPath(KelpDir)
}

func Browse(owner, repo string) error {
	if err := ValidateRepoName(owner, repo); err != nil {
		return err
	}

	url := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	logging.LogDebug("Opening %s\n", url)
	return openPath(url)
}

func (kc *KelpConfig) Doctor() {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	for _, p := range kc.Packages {
		// check alias first
		var binary string
		if p.Binary != "" {
			binary = p.Binary
		} else {
			binary = p.Repo
		}

		status := ""
		path, err := commandExists(binary)
		if err != nil {
			status = "❌ Binary not found"
		} else {
			if strings.HasPrefix(path, KelpBin) {
				status = "✅ Installed"
			} else {
				status = "⛔️ Installed outside kelp"
			}
		}
		logging.LogInfo("%s\t%s\n", binary, status)
	}
	_ = w.Flush()
}

func commandExists(cmd string) (string, error) {
	return exec.LookPath(cmd)
}
