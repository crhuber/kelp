package main

import (
	"crhuber/kelp/cmd"
	"log"
)

var version = "0.0.1"

func main() {
	rootCmd := cmd.NewRootCmd(version)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
