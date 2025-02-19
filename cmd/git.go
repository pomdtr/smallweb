package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

func NewCmdGit(baseDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "Git server",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("git"); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.AddCommand(NewCmdGitReceivePack(baseDir))
	cmd.AddCommand(NewCmdGitUploadPack(baseDir))

	return cmd
}

func NewCmdGitReceivePack(baseDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git-receive-pack <git-dir>",
		Short: "Git receive-pack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gitCacheDir := filepath.Join(xdg.CacheHome, "smallweb", "git")
			if err := os.MkdirAll(gitCacheDir, 0755); err != nil {
				return err
			}

			appDir := filepath.Join(baseDir, args[0])
			repoDir := filepath.Join(gitCacheDir, filepath.Base(appDir))
			if _, err := os.Stat(repoDir); os.IsNotExist(err) {
				initCmd := exec.Command("git", "init", repoDir, "--bare")
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

			return nil
		},
	}

	return cmd
}

func NewCmdGitUploadPack(baseDir string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "git-upload-pack <git-dir>",
		Short: "Git upload-pack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(baseDir, args[0])
			gitCacheDir := filepath.Join(xdg.CacheHome, "smallweb", "git")

			repoDir := filepath.Join(gitCacheDir, filepath.Base(appDir))
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
