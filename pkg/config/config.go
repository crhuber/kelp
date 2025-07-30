package config

import (
	"crhuber/kelp/pkg/types"
	"crhuber/kelp/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

var home, _ = os.UserHomeDir()

var KelpDir = filepath.Join(home, "/.kelp/")
var KelpBin = filepath.Join(home, "/.kelp/bin/")
var KelpCache = filepath.Join(home, "/.kelp/cache/")

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

func (kc *KelpConfig) Pop(index int) []KelpPackage {
	return append(kc.Packages[:index], kc.Packages[index+1:]...)
}

func (kc *KelpConfig) GetPackage(repo string) (KelpPackage, error) {
	parts := strings.Split(repo, "/")
	// Check if there is an owner part since some projects have the same repo name
	// like cli
	if len(parts) > 1 {
		// If there is an owner, get the more specific project first
		for _, kp := range kc.Packages {
			if kp.Owner == parts[0] && kp.Repo == parts[1] {
				return kp, nil
			}
		}
	} else {
		for _, kp := range kc.Packages {

			if kp.Repo == repo {
				return kp, nil
			}
		}
	}
	err := errors.New("package not found in config, try adding it first")
	kp := KelpPackage{}
	return kp, err
}

func Load(path string) (*KelpConfig, error) {
	bs, _ := os.ReadFile(path)
	kc := KelpConfig{}
	err := json.Unmarshal(bs, &kc.Packages)
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
	fmt.Println("Config saved.")
	return nil
}

func (kc *KelpConfig) RemovePackage(repo string) error {
	for i, kp := range kc.Packages {
		if kp.Repo == repo {
			kc.Packages = kc.Pop(i)
			fmt.Printf("Package %s removed\n", repo)
			return nil
		}
	}
	return errors.New("package not found in config")
}

func (kc *KelpConfig) AddPackage(owner, repo, release string) error {

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
	fmt.Println("Config added!")

	return nil
}

func (kc *KelpConfig) UpdatePackage(repo string) (string, error) {
	for _, p := range kc.Packages {
		if p.Repo == repo {
			ghr, err := utils.GetGithubRelease(p.Owner, p.Repo, "latest")
			if err != nil {
				return "", err
			}
			return ghr.TagName, nil
		}
	}
	return "", errors.New("package not found in config")
}

func (kc *KelpConfig) SetPackage(repo, release, description, binary string) error {
	for i, p := range kc.Packages {
		if p.Repo == repo {
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
			fmt.Println("Config set!")
			return nil
		}
	}
	return nil
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
		} else {
			release = pkg.Release
		}

		fmt.Fprintf(w, "\n%s/%s\t%s\t%s", pkg.Owner, pkg.Repo, release, humanFriendlyTimestamp)
	}
	w.Flush()
}

func Initialize(path string) error {
	if !utils.DirExists(KelpDir) {
		fmt.Println("Creating Kelp dir...")
		err := os.Mkdir(KelpDir, 0777)
		if err != nil {
			return err
		}
	}

	if !utils.DirExists(KelpCache) {
		fmt.Println("Creating Kelp cache...")
		err := os.Mkdir(KelpCache, 0777)
		if err != nil {
			return err
		}
	}

	if !utils.DirExists(KelpBin) {
		fmt.Println("Creating Kelp bin...")
		err := os.Mkdir(KelpBin, 0777)
		if err != nil {
			return err
		}
	}

	// create empty config
	kc := KelpConfig{}
	kc.Path = path

	var kp KelpPackage
	kp.Owner = "crhuber"
	kp.Repo = "kelp"
	kp.Release = "latest"
	kp.UpdatedAt = time.Now()
	kp.Description = "Simple homebrew alternative"
	kc.Packages = append(kc.Packages, kp)

	if !utils.FileExists(path) {
		fmt.Println("Creating Kelp config file...")
		err := kc.Save()
		if err != nil {
			return err
		}
	} else {
		fmt.Println("Skipping Kelp config file creation since one alredy exists...")
	}

	fmt.Println("🌱 Kelp Initialized!")
	fmt.Printf("🗒  Add Kelp to your path by running: \nexport PATH=%s:$PATH >> ~/.bash_profile\n", KelpBin)
	return nil
}

func Inspect() {
	var err error
	switch types.GetOS() {
	case types.Darwin:
		err = exec.Command("open", KelpDir).Start()
	case types.Linux:
		err = exec.Command("xdg-open", KelpDir).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func Browse(owner, repo string) {
	var err error
	url := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	fmt.Printf("Opening %s\n", url)

	switch types.GetOS() {
	case types.Darwin:
		err = exec.Command("open", url).Start()
	case types.Linux:
		err = exec.Command("xdg-open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
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
		path, err := utils.CommandExists(binary)
		if err != nil {
			status = "❌ Binary not found"
		} else {
			if strings.HasPrefix(path, KelpBin) {
				status = "✅ Installed"
			} else {
				status = "⛔️ Installed outside kelp"
			}
		}
		fmt.Fprintf(w, "\n%s\t%s", binary, status)
	}
	w.Flush()
}
