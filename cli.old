package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func printDefaults(command string) {

	switch command {
	case "root":
		fmt.Printf("Available commands: 'add' 'browse' 'install' 'init' 'list' \n")
	case "add":
		fmt.Printf("Usage: kelp add <owner>/<repo> <release>\n")
	case "browse":
		fmt.Printf("Usage: kelp browse <project> \n")
	case "install":
		fmt.Printf("Usage: kelp install <project> <release>\n")
	}

	os.Exit(1)
}

// Cli does command line interface
func Cli() {

	// if os.Getenv("GITHUB_USERNAME") == "" {
	// 	fmt.Println("GITHUB_USERNAME not set")
	// 	os.Exit(1)
	// }

	// if os.Getenv("GITHUB_TOKEN") == "" {
	// 	fmt.Println("GITHUB_TOKEN not set")
	// 	os.Exit(1)
	// }
	// githubToken = os.Getenv("GITHUB_TOKEN")
	// githubUsername = os.Getenv("GITHUB_USERNAME")

	var err error
	home, err = os.UserHomeDir()
	if err != nil {
		log.Panic()
	}

	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	installCmdAll := installCmd.String("all", "false", "to install all packages or not")

	if len(os.Args) < 2 {
		printDefaults("root")
	}

	switch os.Args[1] {

	case "browse":
		if len(os.Args) < 3 {
			printDefaults("browse")
		}

		repo := os.Args[2]
		kc, err := findKelpConfig(repo)
		if err != nil {
			fmt.Printf("%s \n", err)
			os.Exit(1)
		}
		browse(kc.Owner, kc.Repo)

	case "add":
		if len(os.Args) < 3 {
			printDefaults("add")
		}

		project := os.Args[2]
		var release string
		if len(os.Args) == 3 {
			release = "latest"
		} else {
			release = os.Args[3]
		}

		ownerRepo := strings.Split(project, "/")
		if len(ownerRepo) < 2 {
			fmt.Println("use owner/repo format")
			os.Exit(1)

		}
		configAdd(ownerRepo[0], ownerRepo[1], release)

	case "init":
		initialize()

	case "inspect":
		inspect()

	case "install":
		installCmd.Parse(os.Args[2:])
		if installCmd.Parsed() {
			if *installCmdAll == "true" {
				installAll()
				os.Exit(0)
			} else {
				if len(os.Args) < 3 {
					printDefaults("install")
				}
				repo := os.Args[2]

				kc, err := findKelpConfig(repo)
				if err != nil {
					fmt.Printf("%s \n", err)
					os.Exit(1)
				}
				install(kc.Owner, kc.Repo, kc.Release)
			}
		}

	case "list":
		list()

	case "--help":
		printDefaults("root")

	default:
		printDefaults("root")
	}

}
