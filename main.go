package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/pomdtr/smallweb/cmd"
)

//go:embed CHANGELOG.md
var changelog string

var version = "dev"

func main() {
	root := cmd.NewCmdRoot(version, changelog)
	if err := root.Execute(); err != nil {
		if exitErr, ok := err.(*cmd.ExitError); ok {
			fmt.Printf("Error: %s\n", exitErr.Message)
			os.Exit(exitErr.Code)
		}

		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
