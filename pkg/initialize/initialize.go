package initialize

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func Initialize() {
	if utils.DirExists(config.KelpDir) == false {
		fmt.Println("\nCreating Kelp dir...")
		err := os.Mkdir(config.KelpDir, 0777)
		if err != nil {
			fmt.Println(err)
		}
	}

	if utils.DirExists(config.KelpCache) == false {
		fmt.Println("\nCreating Kelp cache...")
		os.Mkdir(config.KelpCache, 0777)

	}

	if utils.DirExists(config.KelpBin) == false {
		fmt.Println("\nCreating Kelp bin...")
		os.Mkdir(config.KelpBin, 0777)
	}

	// create empty config
	if utils.FileExists(config.KelpConf) == false {
		var kp config.KelpPackage
		kp.Owner = "crhuber"
		kp.Repo = "kelp"
		kp.Release = "latest"
		var kc config.KelpConfig
		kc2 := append(kc, kp)
		bs, err := json.MarshalIndent(kc2, "", " ")
		if err != nil {
			fmt.Println(bs)
		}
		ioutil.WriteFile(config.KelpConf, bs, 0644)
	}

	fmt.Println("\n🌱 Kelp Initialized!")
	fmt.Printf("\n🗒  Add Kelp to your path by running: \nexport PATH=%s:$PATH >> ~/.bash_profile", config.KelpBin)
}
