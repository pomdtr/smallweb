package main

import (
	_ "embed"
	"os"

	"github.com/carapace-sh/carapace"
	"github.com/pomdtr/smallweb/cmd"
)

//go:embed CHANGELOG.md
var changelog string

func main() {
	root := cmd.NewCmdRoot(changelog)
	carapace.Gen(root)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
