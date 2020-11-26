package main

import (
	"os"
	"path/filepath"
)

var githubToken string
var githubUsername string
var home, err = os.UserHomeDir()

var kelpDir = filepath.Join(home, "/.kelp/")
var kelpBin = filepath.Join(home, "/.kelp/bin/")
var kelpCache = filepath.Join(home, "/.kelp/cache/")
var kelpConf = filepath.Join(home, "/.kelp/kelp.json")

func main() {
	Cli()
}
