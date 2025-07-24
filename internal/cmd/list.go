package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdList() *cobra.Command {
	var flags struct {
		json  bool
		admin bool
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

			names, err := app.LookupApps(k.String("dir"))
			if err != nil {
				cmd.PrintErrln("failed to list apps:", err)
				return ExitError{1}
			}

			apps := make([]app.App, 0)
			for _, name := range names {
				a, err := app.LoadApp(name, k.String("dir"), k.String("domain"))
				if err != nil {
					continue
				}
				if cmd.Flags().Changed("admin") && a.Config.Admin != flags.admin {
					continue
				}

				apps = append(apps, a)
			}

			if flags.json {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(apps); err != nil {
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

			printer.AddHeader([]string{"Name", "Dir", "Domain", "Admin"})
			for _, a := range apps {
				printer.AddField(a.Name)
				printer.AddField(strings.Replace(a.BaseDir, os.Getenv("HOME"), "~", 1))
				printer.AddField(a.Domain)

				if a.Config.Admin {
					printer.AddField("Yes")
				} else {
					printer.AddField("No")
				}

				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().BoolVar(&flags.admin, "admin", false, "filter by admin")

	return cmd
}
