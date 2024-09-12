package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

//go:embed embed/template/*
var initTemplate embed.FS

func NewCmdCreate() *cobra.Command {
	var flags struct {
		template string
	}
	repoRegexp := regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9_.-]+$`)

	cmd := &cobra.Command{
		Use:     "create <app>",
		Short:   "Init a new smallweb app",
		GroupID: CoreGroupID,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			appDir := filepath.Join(rootDir, args[0])
			if _, err := os.Stat(appDir); !os.IsNotExist(err) {
				return fmt.Errorf("directory already exists: %s", appDir)
			}

			if flags.template == "" {
				subFs, err := fs.Sub(initTemplate, "embed/template")
				if err != nil {
					return fmt.Errorf("failed to get template sub fs: %w", err)
				}

				if err := os.CopyFS(appDir, subFs); err != nil {
					return fmt.Errorf("failed to copy template: %w", err)
				}

				cmd.Printf("App initialized, you can now access it at %s.%s\n", args[0], k.String("domain"))
				return nil
			}

			var repoUrl string
			if !repoRegexp.MatchString(flags.template) {
				return fmt.Errorf("invalid template: %s", flags.template)
			}

			repoUrl = fmt.Sprintf("https://github.com/%s.git", flags.template)
			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("git not found: %w", err)
			}

			cloneCmd := exec.Command("git", "clone", "--depth=1", "--single-branch", repoUrl, appDir)
			if err := cloneCmd.Run(); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}

			if err := os.RemoveAll(filepath.Join(appDir, ".git")); err != nil {
				return fmt.Errorf("failed to remove .git directory: %w", err)
			}

			cmd.Printf("App initialized, you can now access it at %s.%s\n", args[0], k.String("domain"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.template, "template", "t", "", "The template to use")

	return cmd
}
