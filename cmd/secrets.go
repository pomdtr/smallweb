package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type Secret struct {
	Env   string `json:"env"`
	Value string `json:"value"`
}

func NewCmdSecrets() *cobra.Command {
	var flags struct {
		json   bool
		app    string
		dotenv bool
	}

	cmd := &cobra.Command{
		Use:   "secrets [app]",
		Short: "Print app secrets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := flags.app
			if appName == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("could not get current working directory: %v", err)
				}

				if cwd == path.Clean(k.String("dir")) {
					return fmt.Errorf("not in an app directory")
				}

				if !strings.HasPrefix(cwd, k.String("dir")) {
					return fmt.Errorf("not in an app directory")
				}

				appDir := cwd
				for filepath.Dir(appDir) != k.String("dir") {
					appDir = filepath.Dir(appDir)
				}

				appName = filepath.Base(appDir)
			}

			secretPath := filepath.Join(k.String("dir"), appName, "secrets.enc.env")

			rawBytes, err := os.ReadFile(secretPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("secrets file %s does not exist", secretPath)
				}

				return fmt.Errorf("could not read %s: %v", secretPath, err)
			}

			dotenvBytes, err := decrypt.Data(rawBytes, "dotenv")
			if err != nil {
				return fmt.Errorf("could not decrypt %s: %v", secretPath, err)
			}

			dotenv, err := godotenv.Parse(bytes.NewReader(dotenvBytes))
			if err != nil {
				return fmt.Errorf("could not parse %s: %v", secretPath, err)
			}

			if flags.dotenv {
				dotenvBytes, err := godotenv.Marshal(dotenv)
				if err != nil {
					return fmt.Errorf("could not marshal %s: %v", secretPath, err)
				}

				fmt.Fprintln(cmd.OutOrStdout(), string(dotenvBytes))
				return nil
			}

			var secrets []Secret
			for key, value := range dotenv {
				secrets = append(secrets, Secret{Env: key, Value: value})
			}

			if flags.json {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(secrets); err != nil {
					return fmt.Errorf("could not encode %s: %v", secretPath, err)
				}

				return nil
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

			printer.AddHeader([]string{"Env", "Value"})
			for _, secret := range secrets {
				printer.AddField(secret.Env)
				printer.AddField(secret.Value)
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&flags.dotenv, "dotenv", false, "Output as dotenv")
	cmd.Flags().StringVar(&flags.app, "app", "", "App name")
	cmd.RegisterFlagCompletionFunc("app", completeApp)

	return cmd

}
