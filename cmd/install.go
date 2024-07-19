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
	Name          string `json:"name"`
	Description   string `json:"description"`
	RepositoryURL string `json:"repository_url"`
}

func NewCmdInstall() *cobra.Command {
	var flags struct {
		branch string
	}

	cmd := &cobra.Command{
		Use:  "install [app] [dir]",
		Args: cobra.RangeArgs(1, 2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveFilterDirs
			}

			resp, err := http.Get("https://apps.smallweb.run")
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

			url := fmt.Sprintf("https://github.com/%s.git", args[0])

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

			cloneArgs := []string{"clone", url, dir}
			cloneCmd := exec.Command("git", cloneArgs...)
			cmd.PrintErrln("Running command", cloneCmd.String())
			if err := cloneCmd.Run(); err != nil {
				return fmt.Errorf("failed to clone repo: %w", err)
			}

			branchCmd := exec.Command("git", "branch", `--format="%(refname:short)"`)
			branchCmd.Dir = dir
			out, err := branchCmd.Output()
			if err != nil {
				return fmt.Errorf("failed to get branch: %w", err)
			}

			branches := strings.Split(strings.Trim(string(out), "\n"), "\n")
			for _, branch := range branches {
				if branch != flags.branch {
					continue
				}

				cmd.PrintErrln("Checking out smallweb branch")
				checkoutCmd := exec.Command("git", "checkout", "smallweb")
				branchCmd.Dir = dir
				if err := checkoutCmd.Run(); err != nil {
					return fmt.Errorf("failed to checkout smallweb branch: %w", err)
				}

				return nil
			}

			if cmd.Flags().Changed("branch") {
				// clean up
				if err := os.RemoveAll(dir); err != nil {
					return fmt.Errorf("failed to remove dir: %w", err)
				}

				return fmt.Errorf("branch not found: %s", flags.branch)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.branch, "branch", "smallweb", "branch to checkout")
	return cmd

}
