package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var ghRepoRegexp = regexp.MustCompile(`^[a-z\d](?:[a-z\d-]{0,38}[a-z\d])?/[a-zA-Z0-9_.-]+$`)

func NewCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <repo> [app]",
		Args:  cobra.RangeArgs(1, 2),
		Short: "Install an app",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("git is not installed: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repoUrl := args[0]
			// check if repo is <user>/<repo>
			if ghRepoRegexp.MatchString(repoUrl) {
				repoUrl = fmt.Sprintf("https://github.com/%s.git", repoUrl)
			}

			var appName string
			if len(args) > 1 {
				appName = args[1]
			} else {
				parts := strings.Split(repoUrl, "/")
				if len(parts) < 2 {
					return fmt.Errorf("invalid repository URL: %s", repoUrl)
				}

				repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
				appName = strings.TrimPrefix(repoName, "smallweb-")
			}
			appDir := filepath.Join(k.String("dir"), appName)
			if _, err := exec.LookPath(appDir); err == nil {
				return fmt.Errorf("directory already exists: %s", appDir)
			}

			branches, err := listBranches(repoUrl)
			if err != nil {
				return fmt.Errorf("failed to list branches: %w", err)
			}

			if len(branches) == 0 {
				return fmt.Errorf("no branches found in the repository")
			}

			if _, err := os.Stat(filepath.Join(k.String("dir"), ".gitmodules")); err == nil {
				if slices.Contains(branches, "smallweb") {
					addCmd := exec.Command("git", "submodule", "add", "--branch", "smallweb", repoUrl, appDir)
					if err := addCmd.Run(); err != nil {
						return fmt.Errorf("failed to add submodule: %w", err)
					}

					cmd.PrintErrf("App %s available at https://%s.%s\n", appName, appName, k.String("domain"))
					return nil
				}

				addCmd := exec.Command("git", "submodule", "add", repoUrl, appDir)
				if err := addCmd.Run(); err != nil {
					return fmt.Errorf("failed to add submodule: %w", err)
				}

				cmd.PrintErrf("App %s available at https://%s.%s\n", appName, appName, k.String("domain"))
				return nil
			}

			if slices.Contains(branches, "smallweb") {
				cmd.PrintErrf("Cloning branch 'smallweb' from %s to %s...\n", repoUrl, appDir)
				cloneCmd := exec.Command("git", "clone", "--single-branch", "--branch", "smallweb", repoUrl, appDir)
				if err := cloneCmd.Run(); err != nil {
					return fmt.Errorf("failed to clone branch: %w", err)
				}

				cmd.PrintErrf("App %s available at https://%s.%s\n", appName, appName, k.String("domain"))
				return nil
			}

			cmd.PrintErrf("Cloning %s to %s...\n", repoUrl, appDir)
			cloneCmd := exec.Command("git", "clone", "--single-branch", appDir)
			if err := cloneCmd.Run(); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}

			cmd.Printf("App %s available at https://%s.%s\n", appName, appName, k.String("domain"))
			return nil
		},
	}

	cmd.Flags().StringP("url", "u", "", "URL of the app to install")
	cmd.Flags().StringP("dir", "d", "", "Directory to install the app in")

	return cmd
}

func listBranches(remote string) ([]string, error) {
	out, err := exec.Command("git", "ls-remote", "--heads", remote).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	branches := make([]string, 0, len(lines))

	for _, line := range lines {
		if !strings.Contains(line, "refs/heads/") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		branch := strings.TrimPrefix(parts[1], "refs/heads/")
		branches = append(branches, branch)
	}

	return branches, nil
}
