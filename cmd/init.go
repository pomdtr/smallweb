package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewCmdInit() *cobra.Command {
	var flags struct {
		template string
	}

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
	cmd.RegisterFlagCompletionFunc("template", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		resp, err := http.Get("https://api.smallweb.run/v1/templates")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		var repos []Repository
		if err := decoder.Decode(&repos); err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var completions []string
		for _, repo := range repos {
			completions = append(completions, fmt.Sprintf("%s\t%s", repo.Name, repo.Description))
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
