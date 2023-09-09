package config

import (
	"crhuber/kelp/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

var home, _ = os.UserHomeDir()

var KelpDir = filepath.Join(home, "/.kelp/")
var KelpBin = filepath.Join(home, "/.kelp/bin/")
var KelpCache = filepath.Join(home, "/.kelp/cache/")

// var KelpConf = filepath.Join(home, "/.kelp/kelp.json")

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
	for _, kp := range kc.Packages {
		if kp.Repo == repo {
			return kp, nil
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
	err := os.WriteFile(kc.Path, bs, 0644)
	if err != nil {
		return err
	}
	fmt.Println("\nConfig saved.")
	return nil
}

func (kc *KelpConfig) RemovePackage(repo string) error {
	for i, kp := range kc.Packages {
		if kp.Repo == repo {
			kc.Packages = kc.Pop(i)
			fmt.Printf("\nPackage %s removed", repo)
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
	fmt.Println("\nConfig added!")

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
			fmt.Println("\nConfig set!")
			return nil
		}
	}
	return nil
}

func (kc *KelpConfig) List() {
	fmt.Println("\nConfig: ")
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

	// sort by date
	sort.Slice(kc.Packages, func(i, j int) bool {
		return kc.Packages[i].UpdatedAt.Before(kc.Packages[j].UpdatedAt)
	})

	for _, kp := range kc.Packages {
		fmt.Fprintf(w, "\n%s/%s\t%s", kp.Owner, kp.Repo, kp.Release)
	}
	w.Flush()
}

func Initialize(path string) (*KelpConfig, error) {
	if !utils.DirExists(KelpDir) {
		fmt.Println("\nCreating Kelp dir...")
		err := os.Mkdir(KelpDir, 0777)
		if err != nil {
			return nil, err
		}
	}

	if !utils.DirExists(KelpCache) {
		fmt.Println("\nCreating Kelp cache...")
		err := os.Mkdir(KelpCache, 0777)
		if err != nil {
			return nil, err
		}

	}

	if !utils.DirExists(KelpBin) {
		fmt.Println("\nCreating Kelp bin...")
		err := os.Mkdir(KelpBin, 0777)
		if err != nil {
			return nil, err
		}
	}

	// create empty config
	kc := KelpConfig{}
	if !utils.FileExists(path) {
		var kp KelpPackage
		kp.Owner = "crhuber"
		kp.Repo = "kelp"
		kp.Release = "latest"
		kp.UpdatedAt = time.Now()
		kp.Description = "Simple homebrew alternative"
		kc.Packages = append(kc.Packages, kp)
	}

	fmt.Println("\n🌱 Kelp Initialized!")
	fmt.Printf("\n🗒  Add Kelp to your path by running: \nexport PATH=%s:$PATH >> ~/.bash_profile", KelpBin)
	return &kc, nil
}

func Inspect() {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", KelpDir).Start()
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
	fmt.Printf("\nOpening %s", url)

	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func (kc *KelpConfig) Doctor() {
	tw := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	defer tw.Flush()
	for _, p := range kc.Packages {

		// check alias first
		var binary string
		if p.Binary != "" {
			binary = p.Binary
		} else {
			binary = p.Repo
		}

		path, err := utils.CommandExists(binary)
		if err != nil {
			fmt.Fprintf(tw, "\n%s\t❌ Binary not found", p.Repo)
		} else {
			if strings.HasPrefix(path, KelpBin) {
				fmt.Fprintf(tw, "\n%s\t✅", p.Repo)
			} else {
				fmt.Fprintf(tw, "\n%s\t⛔️ Installed outside kelp", p.Repo)
			}

		}
	}
}
