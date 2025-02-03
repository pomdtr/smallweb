package main

import (
	_ "embed"
	"errors"
	"os"
	"os/exec"

	"github.com/carapace-sh/carapace"
	"github.com/pomdtr/smallweb/cmd"
)

func main() {
	root := cmd.NewCmdRoot()
	carapace.Gen(root)
	if err := root.Execute(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.ExitCode())
		}

		os.Exit(1)
	}
}
