package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type AppEntry struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
	Url  string `json:"url"`
}

func NewCmdList() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if k.String("dir") == "" {
				cmd.PrintErrln("smallweb directory not set")
				return ExitError{1}
			}

			domain := k.String("domain")
			if domain == "" {
				cmd.PrintErrln("domain not set")
				return ExitError{1}
			}

			baseDir := filepath.Join(k.String("dir"), domain)
			entries, err := os.ReadDir(baseDir)
			if err != nil {
				cmd.PrintErrf("failed to read directory %q: %v\n", baseDir, err)
				return ExitError{1}
			}

			apps := make([]app.App, 0)
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				a, err := app.LoadApp(baseDir, k.String("domain"), entry.Name())
				if err != nil {
					continue
				}

				apps = append(apps, a)
			}

			if flags.json {
				var entries []AppEntry

				for _, a := range apps {
					entries = append(entries, AppEntry{
						Url:  a.URL(),
						Name: a.Name,
						Dir:  strings.Replace(a.Dir, os.Getenv("HOME"), "~", 1),
					})
				}

				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(entries); err != nil {
					cmd.PrintErrf("failed to encode apps as json: %v\n", err)
					return ExitError{1}
				}

				return nil
			}

			if len(apps) == 0 {
				cmd.Println("No apps found")
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

			printer.AddHeader([]string{"Name", "Dir", "Url"})
			for _, a := range apps {
				printer.AddField(a.Name)
				printer.AddField(strings.Replace(a.Dir, os.Getenv("HOME"), "~", 1))
				printer.AddField(a.URL())

				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
