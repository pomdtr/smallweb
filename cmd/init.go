package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

func NewCmdInit() *cobra.Command {
	var flags struct {
		template string
	}
	repoRegexp := regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9_.-]+$`)

	cmd := &cobra.Command{
		Use:     "init [dir]",
		Short:   "Init a new smallweb app",
		GroupID: CoreGroupID,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("git not found: %w", err)
			}

			var repoUrl string
			if flags.template != "" {
				if !repoRegexp.MatchString(flags.template) {
					return fmt.Errorf("invalid template: %s", flags.template)
				}

				repoUrl = fmt.Sprintf("https://github.com/%s.git", flags.template)
			} else {
				repoUrl = "https://github.com/pomdtr/smallweb-http-template.git"
			}

			var dir string
			if len(args) > 0 {
				dir = args[0]
			} else {
				dir = "."
			}

			cloneCmd := exec.Command("git", "clone", "--depth=1", "--single-branch", repoUrl, dir)
			cloneCmd.Stdout = os.Stdout
			cloneCmd.Stderr = os.Stderr
			if err := cloneCmd.Run(); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}

			if err := os.RemoveAll(filepath.Join(dir, ".git")); err != nil {
				return fmt.Errorf("failed to remove .git directory: %w", err)
			}

			return nil

		},
	}

	cmd.Flags().StringVarP(&flags.template, "template", "t", "", "The template to use")
	cmd.MarkFlagRequired("template")

	return cmd
}
