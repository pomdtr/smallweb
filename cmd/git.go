package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewCmdGitReceivePack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-receive-pack <git-dir>",
		Short:  "Git receive-pack",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(k.String("dir"), args[0])
			if baseDir := filepath.Dir(appDir); baseDir != k.String("dir") {
				return fmt.Errorf("not in an app directory")
			}

			reposDir := filepath.Join(k.String("dir"), ".smallweb", "repos")
			if err := os.MkdirAll(reposDir, 0755); err != nil {
				return err
			}

			repoDir := filepath.Join(reposDir, filepath.Base(appDir))
			if _, err := os.Stat(repoDir); os.IsNotExist(err) {
				initCmd := exec.Command("git", "init", repoDir, "--bare", "--initial-branch=main")
				if err := initCmd.Run(); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			gitReceiveCmd := exec.Command("git-receive-pack", repoDir)

			gitReceiveCmd.Stdin = cmd.InOrStdin()
			gitReceiveCmd.Stdout = cmd.OutOrStdout()
			gitReceiveCmd.Stderr = cmd.ErrOrStderr()

			if err := gitReceiveCmd.Run(); err != nil {
				return err
			}

			if _, err := os.Stat(appDir); os.IsNotExist(err) {
				if err := os.MkdirAll(appDir, 0755); err != nil {
					return err
				}
			}

			gitCheckOutCmd := exec.Command("git", "--git-dir", repoDir, "--work-tree", appDir, "checkout", "HEAD", "--", ".")
			gitCheckOutCmd.Stderr = cmd.ErrOrStderr()
			if err := gitCheckOutCmd.Run(); err != nil {
				return err
			}

			cmd.PrintErrf("\nYour app is available at https://%s.%s\n\n", filepath.Base(appDir), k.String("domain"))

			return nil
		},
	}

	return cmd
}

func NewCmdGitUploadPack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-upload-pack <git-dir>",
		Short:  "Git upload-pack",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(k.String("dir"), args[0])
			if baseDir := filepath.Dir(appDir); baseDir != k.String("dir") {
				return fmt.Errorf("not in an app directory")
			}

			reposDir := filepath.Join(k.String("dir"), ".smallweb", "repos")
			repoDir := filepath.Join(reposDir, filepath.Base(appDir))
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
