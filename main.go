package main

import (
	"crhuber/kelp/cmd"
	"log"
)

var version = "1.11.0"

func main() {
	rootCmd := cmd.NewRootCmd(version)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
