package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewCmdGitReceivePack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-receive-pack <git-dir>",
		Hidden: true,
		Short:  "Git receive-pack",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := strings.TrimSuffix(args[0], ".git")
			repoDir := filepath.Join(k.String("dir"), ".smallweb", "repos", repo)
			gitReceiveCmd := exec.Command("git-receive-pack", repoDir)
			gitReceiveCmd.Stdin = cmd.InOrStdin()
			gitReceiveCmd.Stdout = cmd.OutOrStdout()
			gitReceiveCmd.Stderr = cmd.ErrOrStderr()

			if err := gitReceiveCmd.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func NewCmdGitUploadPack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-upload-pack <git-dir>",
		Hidden: true,
		Short:  "Git upload-pack",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := strings.TrimSuffix(args[0], ".git")
			repoDir := filepath.Join(k.String("dir"), ".smallweb", "repos", repo)
			uploadCmd := exec.Command("git-upload-pack", repoDir)

			uploadCmd.Stdin = cmd.InOrStdin()
			uploadCmd.Stdout = cmd.OutOrStdout()
			uploadCmd.Stderr = cmd.ErrOrStderr()

			if err := uploadCmd.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
