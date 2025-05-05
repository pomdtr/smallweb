package main

import (
	_ "embed"
	"os"

	"github.com/pomdtr/smallweb/cmd"
)

func main() {
	root := cmd.NewCmdRoot()

	root.SetIn(os.Stdin)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
