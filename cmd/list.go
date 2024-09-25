package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdList() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		GroupID: CoreGroupID,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			names, err := app.ListApps(rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			apps := make([]app.App, 0)
			for _, name := range names {
				a, err := app.LoadApp(filepath.Join(rootDir, name), k.String("domain"))
				if err != nil {
					return fmt.Errorf("failed to load app: %w", err)
				}

				apps = append(apps, a)
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(apps); err != nil {
					return fmt.Errorf("failed to encode tree: %w", err)
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

				printer = tableprinter.New(os.Stdout, true, width)
			} else {
				printer = tableprinter.New(os.Stdout, false, 0)
			}

			printer.AddHeader([]string{"Name", "Dir", "Url"})
			for _, a := range apps {
				printer.AddField(a.Name)
				printer.AddField(strings.Replace(a.Dir, os.Getenv("HOME"), "~", 1))
				printer.AddField(a.Url)

				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
