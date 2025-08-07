package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewCmdCreate() *cobra.Command {
	var flags struct {
		template string
	}

	cmd := &cobra.Command{
		Use:   "create [app]",
		Args:  cobra.ExactArgs(1),
		Short: "Create a new Smallweb app",
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(k.String("dir"), args[0])
			if _, err := os.Stat(appDir); !os.IsNotExist(err) {
				cmd.PrintErrf("App directory %s already exists.\n", appDir)
				return ExitError{1}
			}

			gitCloneCmd := exec.Command("git", "clone", "--single-branch", "--depth=1", flags.template, appDir)
			gitCloneCmd.Stderr = cmd.ErrOrStderr()
			gitCloneCmd.Stdout = cmd.OutOrStdout()

			if err := gitCloneCmd.Run(); err != nil {
				cmd.PrintErrf("Failed to clone template repository: %v\n", err)
				return ExitError{1}
			}

			if err := os.RemoveAll(filepath.Join(appDir, ".git")); err != nil {
				cmd.PrintErrf("Failed to remove .git directory: %v\n", err)
				return ExitError{1}
			}

			cmd.Printf("Created new Smallweb app in %s\n", appDir)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.template, "template", "t", "https://github.com/pomdtr/smallweb-app-template", "git url of the template repository to use for creating a new project")

	return cmd
}
