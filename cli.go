package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func printDefaults() {
	fmt.Printf("available commands: 'install' 'update' 'init' 'list' 'browse' \n")
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
	installPackage := installCmd.String("package", "", "package")
	installRelease := installCmd.String("release", "latest", "release")

	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	updatePackage := updateCmd.String("package", "", "package")

	browseCmd := flag.NewFlagSet("browse", flag.ExitOnError)
	browsePackage := browseCmd.String("package", "", "package")

	if len(os.Args) < 2 {
		printDefaults()
	}

	switch os.Args[1] {

	case "install":
		installCmd.Parse(os.Args[2:])
		if installCmd.Parsed() {
			// Required Flags
			if *installPackage == "" {
				installCmd.PrintDefaults()
				os.Exit(1)
			}
			if *installRelease == "" {
				installCmd.PrintDefaults()
				os.Exit(1)
			}
		}

		if *installRelease == "" {
			*installRelease = "latest"
		}

		ownerRepo := strings.Split(*installPackage, "/")
		if len(ownerRepo) < 2 {
			fmt.Println("use owner/repo format")
			os.Exit(1)

		}

		install(ownerRepo[0], ownerRepo[1], *installRelease)

	case "update":
		updateCmd.Parse(os.Args[2:])
		if updateCmd.Parsed() {
			// Required Flags
			if *updatePackage == "" {
				updateCmd.PrintDefaults()
				os.Exit(1)
			}
		}
		update(*updatePackage)

	case "init":
		initialize()

	case "list":
		list()

	case "inspect":
		inspect()

	case "browse":
		browseCmd.Parse(os.Args[2:])
		if browseCmd.Parsed() {
			// Required Flags
			if *browsePackage == "" {
				browseCmd.PrintDefaults()
				os.Exit(1)
			}
		}
		browse(*browsePackage)

	case "--help":
		printDefaults()

	default:
		printDefaults()
	}

}
