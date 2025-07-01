package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdRepo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
		Long:  "Commands to manage repositories in smallweb.",
	}

	cmd.AddCommand(
		NewCmdRepoCreate(),
		NewCmdRepoList(),
	)

	return cmd
}

func NewCmdRepoCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new repository",
		Long:  "Create a new repository with the given name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !strings.HasSuffix(name, ".git") {
				name += ".git"
			}

			repoDir := filepath.Join(k.String("dir"), ".smallweb", "repos", name)
			if _, err := os.Stat(repoDir); err == nil {
				return fmt.Errorf("repository %s already exists", name)
			}

			if err := os.MkdirAll(repoDir, 0755); err != nil {
				return fmt.Errorf("failed to create repository directory: %w", err)
			}

			gitInitCmd := exec.Command("git", "init", "--bare", repoDir)
			gitInitCmd.Dir = repoDir

			gitInitCmd.Stdout = cmd.OutOrStdout()
			gitInitCmd.Stderr = cmd.ErrOrStderr()

			if err := gitInitCmd.Run(); err != nil {
				return fmt.Errorf("failed to initialize git repository: %w", err)
			}

			// Logic to create a repository
			return nil
		},
	}

	return cmd
}

func NewCmdRepoList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all repositories",
		Long:    "List all repositories in the smallweb workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reposDir := filepath.Join(k.String("dir"), ".smallweb", "repos")
			files, err := os.ReadDir(reposDir)
			if err != nil {
				return fmt.Errorf("failed to read repositories directory: %w", err)
			}

			var names []string
			for _, file := range files {
				if !file.IsDir() {
					continue
				}

				names = append(names, strings.TrimSuffix(file.Name(), ".git"))
			}

			var printer tableprinter.TablePrinter
			if isatty.IsTerminal(os.Stdout.Fd()) {
				width, _, err := term.GetSize(int(os.Stdout.Fd()))
				if err != nil {
					return fmt.Errorf("failed to get terminal size: %w", err)
				}

				printer = tableprinter.New(cmd.OutOrStdout(), true, width)
			} else {
				printer = tableprinter.New(cmd.OutOrStdout(), false, 0)
			}

			printer.AddHeader([]string{"Name", "Remote URL", "Module URL"})
			for _, name := range names {
				printer.AddField(name)
				printer.AddField(fmt.Sprintf("git@%s:%s.git", k.String("domain"), name))
				printer.AddField(fmt.Sprintf("https://esm.%s/%s", k.String("domain"), name))
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	return cmd
}
