package main

import (
	_ "embed"
	"os"

	"github.com/pomdtr/smallweb/cmd"
)

//go:embed CHANGELOG.md
var changelog string

func main() {
	root := cmd.NewCmdRoot(changelog)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
