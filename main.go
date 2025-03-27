package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/pomdtr/smallweb/cmd"
)

func main() {
	root := cmd.NewCmdRoot()

	root.SetIn(os.Stdin)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := root.Execute(); err != nil {
		if !errors.Is(err, cmd.ErrSilent) {
			fmt.Fprintln(os.Stderr, err)
		}

		os.Exit(1)
	}
}
