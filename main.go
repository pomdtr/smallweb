package main

import (
	"context"
	_ "embed"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/pomdtr/smallweb/internal/cmd"
)

func main() {
	root := cmd.NewCmdRoot()

	root.SetIn(os.Stdin)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := fang.Execute(context.Background(), root); err != nil {
		os.Exit(1)
	}
}
