package utils

import (
	"crhuber/kelp/pkg/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func DirExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func FilePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func CopyFile(source, destination string) error {
	from, err := os.Open(source)
	if err != nil {
		log.Fatal(err)
	}
	defer from.Close()

	to, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE, 0744)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	return nil
}

func CommandExists(cmd string) (string, error) {
	path, err := exec.LookPath(cmd)
	return path, err
}

func GetGithubRelease(owner, repo, release string) (types.GithubRelease, error) {
	var url string
	if release == "latest" {
		fmt.Printf("🌐 Getting releases for %s/%s:%s...\n", owner, repo, release)
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%s", owner, repo, release)

	} else {
		// try by tag
		fmt.Printf("🌐 Getting releases by tag %s...\n", release)
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, release)
	}

	// create client
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return types.GithubRelease{}, err
	}

	// set headers for github auth
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken != "" {
		fmt.Println("Using Github token in http request")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
	}

	// make request
	resp, err := client.Do(req)
	if err != nil {
		return types.GithubRelease{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return types.GithubRelease{}, fmt.Errorf("invalid HTTP status: %v", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.GithubRelease{}, err
	}
	ghr := types.GithubRelease{}

	if err := json.Unmarshal(body, &ghr); err != nil {
		return types.GithubRelease{}, err
	}
	return ghr, nil
}
