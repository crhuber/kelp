package main

import (
	"crhuber/kelp/cmd"
	"log"
)

var version = "1.6.3"

func main() {
	rootCmd := cmd.NewRootCmd(version)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
