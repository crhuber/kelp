package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"runtime"
)

type kelpConfig []kelpPackage

type kelpPackage struct {
	Owner   string `json:"Owner"`
	Repo    string `json:"Repo"`
	Release string `json:"Release"`
}

func findKelpConfig(repo string) (kelpPackage, error) {
	kc := loadKelpConfig()
	for _, kp := range kc {
		if kp.Repo == repo {
			return kp, nil
		}
	}
	err := errors.New("package not found in config, try adding it first")
	kp := kelpPackage{}
	return kp, err
}

func configAdd(owner, repo, release string) {
	kp := kelpPackage{
		Owner:   owner,
		Repo:    repo,
		Release: release,
	}
	kp.saveToConfig()
}
func loadKelpConfig() kelpConfig {
	bs, _ := ioutil.ReadFile(kelpConf)
	var kc kelpConfig
	err := json.Unmarshal(bs, &kc)
	if err != nil {
		fmt.Println(err)
	}
	return kc
}

func (kp kelpPackage) saveToConfig() error {
	//kc := loadKelpConfig()
	bs, _ := ioutil.ReadFile(kelpConf)
	var kc kelpConfig
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
		fmt.Println("Config exists!")
	}

	var configUpdated bool = false
	if !matchFound {
		for i := range kc {
			c := &kc[i]
			if kp.Repo == c.Repo {
				c.Release = kp.Release
				bs, _ := json.MarshalIndent(kc, "", " ")
				ioutil.WriteFile(kelpConf, bs, 0644)
				fmt.Println("Config updated!")
				configUpdated = true
				break
			}
		}
	}
	if !matchFound && !configUpdated {
		kc = append(kc, kp)
		bs, _ := json.MarshalIndent(kc, "", " ")
		ioutil.WriteFile(kelpConf, bs, 0644)
		fmt.Println("Config added!")
	}

	return err
}

func list() {
	fmt.Println("Install Config: ")
	kc := loadKelpConfig()
	for _, kp := range kc {
		fmt.Printf("\n%s/%s: %s", kp.Owner, kp.Repo, kp.Release)
	}
}

func inspect() {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", kelpDir).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func browse(owner, repo string) {
	var err error
	var url string
	url = fmt.Sprintf("https://github.com/%s/%s/releases", owner, repo)
	fmt.Printf("Opening %s", url)

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
