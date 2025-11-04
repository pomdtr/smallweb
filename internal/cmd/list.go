package cmd

import (
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

func NewCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if conf.String("dir") == "" {
				cmd.PrintErrln("smallweb directory not set")
				return ExitError{1}
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

			apps, err := app.List(conf.String("dir"))
			if err != nil {
				cmd.PrintErrf("failed to list apps: %v\n", err)
				return ExitError{1}
			}

			if len(apps) == 0 {
				cmd.Println("No apps found")
				return nil
			}

			printer.AddHeader([]string{"Name", "Dir", "Url"})
			for _, appname := range apps {
				printer.AddField(appname)
				printer.AddField(strings.Replace(filepath.Join(conf.String("dir"), appname), os.Getenv("HOME"), "~", 1))
				printer.AddField(fmt.Sprintf("https://%s.%s", appname, conf.String("domain")))

				printer.EndRow()
			}

			return printer.Render()
		},
	}

	return cmd
}
