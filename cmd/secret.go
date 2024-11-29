package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdSecret() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "secret",
		Aliases: []string{"secrets"},
		PreRunE: requireSops(),
	}

	cmd.AddCommand(NewCmdSecretDump())
	cmd.AddCommand(NewCmdSecretSet())
	cmd.AddCommand(NewCmdSecretGet())
	cmd.AddCommand(NewCmdSecretEdit())

	return cmd
}

func NewCmdSecretDump() *cobra.Command {
	var flags struct {
		format string
		app    string
		global bool
	}

	cmd := &cobra.Command{
		Use:     "dump",
		Args:    cobra.NoArgs,
		PreRunE: requireSops(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.FileExists("secrets.enc.json") {
				return fmt.Errorf("secrets.enc.json not found")
			}

			command := exec.Command("sops", "--decrypt", "--output-type", flags.format, "secrets.enc.json")
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			return command.Run()
		},
	}

	cmd.Flags().StringVar(&flags.format, "format", "json", "output format")
	cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "dotenv"}, cobra.ShellCompDirectiveDefault
	})
	cmd.Flags().StringVarP(&flags.app, "app", "a", "", "app name")
	cmd.Flags().BoolVarP(&flags.global, "global", "g", false, "global secrets")

	return cmd
}

func NewCmdSecretSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set <key> <value>",
		Args:    cobra.ExactArgs(2),
		PreRunE: requireSops(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.FileExists("secrets.enc.json") {
				return fmt.Errorf("secrets.enc.json not found")
			}

			key, err := json.Marshal(args[0])
			if err != nil {
				return fmt.Errorf("failed to marshal key: %w", err)
			}

			value, err := json.Marshal(args[1])
			if err != nil {
				return fmt.Errorf("failed to marshal value: %w", err)
			}

			command := exec.Command("sops", "set", "secrets.enc.json", fmt.Sprintf("[%s]", key), string(value))
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			return command.Run()
		},
	}

	return cmd
}

func NewCmdSecretGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <key>",
		Args:    cobra.ExactArgs(1),
		PreRunE: requireSops(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.FileExists("secrets.enc.json") {
				return fmt.Errorf("secrets.enc.json not found")
			}

			buf := bytes.Buffer{}

			command := exec.Command("sops", "--decrypt", "secrets.enc.json")
			command.Stdout = &buf
			command.Stderr = os.Stderr
			if err := command.Run(); err != nil {
				return err
			}

			var secrets map[string]string
			if err := json.Unmarshal(buf.Bytes(), &secrets); err != nil {
				return fmt.Errorf("failed to unmarshal secrets: %w", err)
			}

			value, ok := secrets[args[0]]
			if !ok {
				return fmt.Errorf("key not found: %s", args[0])
			}

			cmd.Println(value)
			return nil
		},
	}

	return cmd
}

func NewCmdSecretEdit() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "edit",
		Args:    cobra.NoArgs,
		PreRunE: requireSops(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.FileExists("secrets.enc.json") {
				return fmt.Errorf("secrets.enc.json not found")
			}

			command := exec.Command("sops", "secrets.enc.json")
			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			return command.Run()
		},
	}

	return cmd
}

func requireSops() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if _, err := exec.LookPath("sops"); err != nil {
			return fmt.Errorf("sops is required to run this command")
		}

		return nil
	}
}
