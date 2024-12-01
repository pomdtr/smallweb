package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdSecrets() *cobra.Command {
	var flags struct {
		global     bool
		updateKeys bool
	}

	cmd := &cobra.Command{
		Use:               "secrets [app]",
		Short:             "Manage app secrets",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeApp(utils.RootDir),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := checkSOPS(); err != nil {
				return err
			}

			if len(args) == 1 && flags.global {
				return fmt.Errorf("cannot set both --global and specify an app")
			}

			if len(args) == 1 && flags.updateKeys {
				return fmt.Errorf("cannot set both --update-keys and specify an app")
			}

			if flags.updateKeys && flags.global {
				return fmt.Errorf("cannot set both --update-keys and --global")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.RootDir
			if flags.updateKeys {
				globalSecretsPath := filepath.Join(rootDir, ".smallweb", "secrets.enc.env")
				if stat, err := os.Stat(globalSecretsPath); err == nil && !stat.IsDir() {
					c := exec.Command("sops", "updatekeys", globalSecretsPath)
					c.Dir = rootDir

					if err := c.Run(); err != nil {
						return fmt.Errorf("failed to update keys: %w", err)
					}
				}

				apps, err := app.ListApps(rootDir)
				if err != nil {
					return fmt.Errorf("failed to list apps: %w", err)
				}

				for _, a := range apps {
					secretsPath := filepath.Join(rootDir, a, "secrets.enc.env")
					if stat, err := os.Stat(secretsPath); err == nil && !stat.IsDir() {
						c := exec.Command("sops", "updatekeys", secretsPath)
						c.Dir = rootDir

						if err := c.Run(); err != nil {
							return fmt.Errorf("failed to update keys: %w", err)
						}
					}
				}

				fmt.Fprintln(os.Stderr, "✅ Keys updated!")

				return nil
			}

			if flags.global {
				globalSecretsPath := filepath.Join(rootDir, ".smallweb", "secrets.enc.env")

				c := exec.Command("sops", globalSecretsPath)
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr

				if err := c.Run(); err != nil {
					var exitErr *exec.ExitError
					if errors.As(err, &exitErr) && exitErr.ExitCode() == 200 {
						return nil
					}

					return fmt.Errorf("failed to update keys: %w", err)
				}

				return nil
			}

			if len(args) == 1 {
				secretsPath := filepath.Join(utils.RootDir, args[0], "secrets.enc.env")
				if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
					return fmt.Errorf("secrets not found: %s", secretsPath)
				}

				c := exec.Command("sops", secretsPath)
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr

				if err := c.Run(); err != nil {
					var exitErr *exec.ExitError
					if errors.As(err, &exitErr) && exitErr.ExitCode() == 200 {
						return nil
					}

					return fmt.Errorf("failed to update keys: %w", err)
				}

				return nil
			}

			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			if filepath.Dir(wd) != rootDir {
				return fmt.Errorf("no app specified and not in an app directory")
			}

			secretPath := filepath.Join(wd, "secrets.enc.env")
			c := exec.Command("sops", secretPath)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			if err := c.Run(); err != nil {
				return fmt.Errorf("failed to edit secrets: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flags.global, "global", "g", false, "Set global secrets")
	cmd.Flags().BoolVar(&flags.updateKeys, "update-keys", false, "Update all keys")

	return cmd
}

func checkSOPS() error {
	_, err := exec.LookPath("sops")
	if err != nil {
		return fmt.Errorf("could not find sops executable")
	}

	return nil
}
