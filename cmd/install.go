package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

// owner/repo
var repoRegexp = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9]{0,38}\/[a-zA-Z0-9_.-]{1,100}$`)

type Repository struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

func NewCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "install [app] [dir]",
		Args: cobra.RangeArgs(1, 2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveFilterDirs
			}

			resp, err := http.Get("https://api.smallweb.run/v1/apps")
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
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("git not found: %w", err)
			}

			if !repoRegexp.MatchString(args[0]) {
				return fmt.Errorf("invalid app: %s", args[0])
			}

			var dir string
			if len(args) > 1 {
				dir = utils.ExpandTilde(args[1])
				if _, err := os.Stat(dir); err == nil {
					subdir := strings.Split(args[0], "/")[1]
					subdir = strings.TrimPrefix(subdir, "smallweb-")
					dir = filepath.Join(dir, subdir)
				}
			} else {
				dir = strings.Split(args[0], "/")[1]
				dir = strings.TrimPrefix(dir, "smallweb-")
			}

			url := fmt.Sprintf("https://github.com/%s.git", args[0])
			lsRemoteCmd := exec.Command("git", "ls-remote", "--heads", url)
			out, err := lsRemoteCmd.Output()
			if err != nil {
				return fmt.Errorf("failed to get branches: %w", err)
			}

			heads := strings.Fields(string(out))
			for i := range len(heads) {
				// Skip the hash
				if i%2 == 0 {
					continue
				}
				head := heads[i]

				if head != "refs/heads/smallweb" {
					continue
				}

				cloneCmd := exec.Command("git", "clone", "--branch", "smallweb", "--single-branch", url, dir)
				cloneCmd.Stdout = os.Stdout
				cloneCmd.Stderr = os.Stderr

				if err := cloneCmd.Run(); err != nil {
					return fmt.Errorf("failed to clone repository: %w", err)
				}

				return nil
			}

			cloneCmd := exec.Command("git", "clone", "--single-branch", url, dir)
			if err := cloneCmd.Run(); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}

			return nil
		},
	}

	return cmd
}
