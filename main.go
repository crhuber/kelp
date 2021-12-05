package main

import (
	"crhuber/kelp/cmd"
	"log"
)

var version = "1.6.7"

func main() {
	rootCmd := cmd.NewRootCmd(version)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
