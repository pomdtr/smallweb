package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/pomdtr/smallweb/internal/cmd"
)

func main() {
	root := cmd.NewCmdRoot()

	root.SetIn(os.Stdin)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := root.Execute(); err != nil {
		var exitErr cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
