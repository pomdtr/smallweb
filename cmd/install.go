package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
)

var script = `#!/bin/sh

exec smallweb --dir=%s run %s -- "$@"
`

func NewCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "install [app]",
		Short:             "Install an app to your shell",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp(k.String("dir")),
		RunE: func(cmd *cobra.Command, args []string) error {
			binDir := path.Join(os.Getenv("HOME"), ".local", "bin")
			if err := os.MkdirAll(binDir, 0755); err != nil {
				return fmt.Errorf("failed to create bin directory: %w", err)
			}

			binPath := path.Join(binDir, fmt.Sprintf("%s.%s", args[0], k.String("domain")))
			if err := os.WriteFile(binPath, []byte(fmt.Sprintf(script, k.String("dir"), args[0])), 0755); err != nil {
				return fmt.Errorf("failed to write script: %w", err)
			}

			// make script executable
			if err := os.Chmod(binPath, 0755); err != nil {
				return fmt.Errorf("failed to make script executable: %w", err)
			}

			return nil
		},
	}

	return cmd

}
