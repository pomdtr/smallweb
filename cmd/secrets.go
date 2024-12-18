package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/getsops/sops/v3/decrypt"
	"github.com/joho/godotenv"
	"github.com/pomdtr/smallweb/app"
	"github.com/spf13/cobra"
)

func NewCmdSecrets() *cobra.Command {
	var flags struct {
		global     bool
		json       bool
		dotenv     bool
		updateKeys bool
	}

	cmd := &cobra.Command{
		Use:               "secrets [app]",
		Short:             "Manage app secrets",
		Aliases:           []string{"secret"},
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeApp,
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
			if flags.updateKeys {
				globalSecretsPath := filepath.Join(k.String("dir"), ".smallweb", "secrets.enc.env")
				if stat, err := os.Stat(globalSecretsPath); err == nil && !stat.IsDir() {
					c := exec.Command("sops", "updatekeys", globalSecretsPath)
					c.Dir = k.String("dir")

					if err := c.Run(); err != nil {
						return fmt.Errorf("failed to update keys: %w", err)
					}
				}

				apps, err := app.ListApps(k.String("dir"))
				if err != nil {
					return fmt.Errorf("failed to list apps: %w", err)
				}

				for _, a := range apps {
					secretsPath := filepath.Join(k.String("dir"), a, "secrets.enc.env")
					if stat, err := os.Stat(secretsPath); err == nil && !stat.IsDir() {
						c := exec.Command("sops", "updatekeys", secretsPath)
						c.Dir = k.String("dir")

						if err := c.Run(); err != nil {
							return fmt.Errorf("failed to update keys: %w", err)
						}
					}
				}

				fmt.Fprintln(os.Stderr, "✅ Keys updated!")

				return nil
			}

			if flags.global {
				globalSecretsPath := filepath.Join(k.String("dir"), ".smallweb", "secrets.enc.env")
				if flags.json {
					return dumpAsJSON(os.Stdout, globalSecretsPath)
				}

				if flags.dotenv {
					return dumpAsDotenv(os.Stdout, globalSecretsPath)
				}

				c := exec.Command("sops", globalSecretsPath)
				c.Dir = k.String("dir")
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
				secretsPath := filepath.Join(k.String("dir"), args[0], "secrets.enc.env")
				if flags.json {
					return dumpAsJSON(os.Stdout, secretsPath)
				}

				if flags.dotenv {
					return dumpAsDotenv(os.Stdout, secretsPath)
				}

				c := exec.Command("sops", secretsPath)
				c.Dir = k.String("dir")
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

			if filepath.Dir(wd) != k.String("dir") {
				return fmt.Errorf("no app specified and not in an app directory")
			}

			secretPath := filepath.Join(wd, "secrets.enc.env")

			if flags.json {
				return dumpAsJSON(os.Stdout, secretPath)
			}

			if flags.dotenv {
				return dumpAsDotenv(os.Stdout, secretPath)
			}

			c := exec.Command("sops", secretPath)
			c.Dir = k.String("dir")
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
	cmd.Flags().BoolVar(&flags.json, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&flags.dotenv, "dotenv", false, "Output as dotenv")

	cmd.MarkFlagsMutuallyExclusive("json", "update-keys")
	cmd.MarkFlagsMutuallyExclusive("dotenv", "update-keys")
	cmd.MarkFlagsMutuallyExclusive("json", "dotenv")

	return cmd
}

func dumpAsDotenv(w io.Writer, secretPath string) error {
	secretBytes, err := os.ReadFile(secretPath)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	dotenvBytes, err := decrypt.Data(secretBytes, "dotenv")
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	if _, err := w.Write(dotenvBytes); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}

func dumpAsJSON(w io.Writer, secretPath string) error {
	secretBytes, err := os.ReadFile(secretPath)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	dotenvBytes, err := decrypt.Data(secretBytes, "dotenv")
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	dotenv, err := godotenv.Parse(bytes.NewReader(dotenvBytes))
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dotenv); err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	return nil
}

func checkSOPS() error {
	_, err := exec.LookPath("sops")
	if err != nil {
		return fmt.Errorf("could not find sops executable")
	}

	return nil
}
