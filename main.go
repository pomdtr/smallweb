package main

import (
	"os"

	"github.com/pomdtr/smallweb/cmd"
)

func main() {
	root := cmd.NewCmdRoot()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
