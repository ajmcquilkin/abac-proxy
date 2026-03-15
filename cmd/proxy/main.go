package main

import (
	"log"

	"github.com/abac/proxy/cmd/proxy/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
