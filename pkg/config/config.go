package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/tabwriter"
)

var githubToken string
var githubUsername string
var home, err = os.UserHomeDir()

var KelpDir = filepath.Join(home, "/.kelp/")
var KelpBin = filepath.Join(home, "/.kelp/bin/")
var KelpCache = filepath.Join(home, "/.kelp/cache/")
var KelpConf = filepath.Join(home, "/.kelp/kelp.json")

type KelpConfig []KelpPackage

type KelpPackage struct {
	Owner   string `json:"Owner"`
	Repo    string `json:"Repo"`
	Release string `json:"Release"`
}

func Pop(kp []KelpPackage, index int) []KelpPackage {
	return append(kp[:index], kp[index+1:]...)
}

func FindKelpConfig(repo string) (KelpPackage, error) {
	kc := LoadKelpConfig()
	for _, kp := range kc {
		if kp.Repo == repo {
			return kp, nil
		}
	}
	err := errors.New("package not found in config, try adding it first")
	kp := KelpPackage{}
	return kp, err
}

func ConfigAdd(owner, repo, release string) {
	kp := KelpPackage{
		Owner:   owner,
		Repo:    repo,
		Release: release,
	}
	kp.saveToConfig()
}

func LoadKelpConfig() KelpConfig {
	bs, _ := ioutil.ReadFile(KelpConf)
	var kc KelpConfig
	err := json.Unmarshal(bs, &kc)
	if err != nil {
		fmt.Println(err)
	}
	return kc
}

func (kc KelpConfig) RemovePackage(repo string) error {
	for i, kp := range kc {
		if kp.Repo == repo {
			var kcNew KelpConfig
			kcNew = Pop(kc, i)
			fmt.Printf("\nPackage %s removed", repo)
			kcNew.save()
			return nil
		}
	}
	err := errors.New("package not found in config")
	return err
}

func (kc KelpConfig) save() error {
	bs, _ := json.MarshalIndent(kc, "", " ")
	ioutil.WriteFile(KelpConf, bs, 0644)
	fmt.Println("\nConfig updated!")
	return nil
}

func (kp KelpPackage) saveToConfig() error {
	//kc := loadKelpConfig()
	bs, _ := ioutil.ReadFile(KelpConf)
	var kc KelpConfig
	err := json.Unmarshal(bs, &kc)

	if err != nil {
		return err
	}

	var matchFound bool = false
	// find exact match
	for _, c := range kc {
		if kp.Repo == c.Repo && kp.Release == c.Release {
			matchFound = true
		}
	}
	// if no match is found check first for a partial match then append
	if matchFound {
		fmt.Println("\nConfig exists!")
	}

	var configUpdated bool = false
	if !matchFound {
		for i := range kc {
			c := &kc[i]
			if kp.Repo == c.Repo {
				c.Release = kp.Release
				bs, _ := json.MarshalIndent(kc, "", " ")
				ioutil.WriteFile(KelpConf, bs, 0644)
				fmt.Println("\nConfig updated!")
				configUpdated = true
				break
			}
		}
	}
	if !matchFound && !configUpdated {
		kc = append(kc, kp)
		bs, _ := json.MarshalIndent(kc, "", " ")
		ioutil.WriteFile(KelpConf, bs, 0644)
		fmt.Println("\nConfig added!")
	}

	return err
}

func List() {
	fmt.Println("\nConfig: ")
	kc := LoadKelpConfig()
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

	for _, kp := range kc {
		fmt.Fprintf(w, "\n%s/%s\t%s", kp.Owner, kp.Repo, kp.Release)
		//fmt.Printf("\n%s/%s\t \t => %s", kp.Owner, kp.Repo, kp.Release)
	}
	w.Flush()
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
	var url string
	url = fmt.Sprintf("https://github.com/%s/%s/releases", owner, repo)
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
