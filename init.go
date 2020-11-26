package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func initialize() {
	if dirExists(kelpDir) == false {
		fmt.Println("Creating Kelp dir...")
		err := os.Mkdir(kelpDir, 0777)
		if err != nil {
			fmt.Println(err)
		}
	}

	if dirExists(kelpCache) == false {
		fmt.Println("Creating Kelp cache...")
		os.Mkdir(kelpCache, 0777)

	}

	if dirExists(kelpBin) == false {
		fmt.Println("Creating Kelp bin...")
		os.Mkdir(kelpBin, 0777)
	}

	// create empty config
	if fileExists(kelpConf) == false {
		var kp kelpPackage
		kp.Owner = "crhuber"
		kp.Repo = "kelp"
		kp.Release = "latest"
		var kc kelpConfig
		kc2 := append(kc, kp)
		bs, err := json.MarshalIndent(kc2, "", " ")
		if err != nil {
			fmt.Println(bs)
		}
		ioutil.WriteFile(kelpConf, bs, 0644)
	}

	fmt.Println("🌱 Kelp Initialized!")
	fmt.Printf("🗒  Add Kelp to your path by running: \nexport PATH=%s:$PATH", kelpBin)
}
