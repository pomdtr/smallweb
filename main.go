package main

import (
	_ "embed"
	"errors"
	"os"
	"os/exec"

	"github.com/pomdtr/smallweb/cmd"
)

func main() {
	root := cmd.NewCmdRoot()

	root.SetIn(os.Stdin)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := root.Execute(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.ExitCode())
		}

		os.Exit(1)
	}
}
