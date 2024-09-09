package main

import (
	_ "embed"
	"os"

	"github.com/pomdtr/smallweb/cmd"
)

//go:embed CHANGELOG.md
var changelog string

var version = "dev"

func main() {
	root := cmd.NewCmdRoot(version, changelog)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
