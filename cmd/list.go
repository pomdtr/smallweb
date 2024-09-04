package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ListApps(rootDir string) []string {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil
	}

	apps := make([]string, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if !entry.IsDir() {
			continue
		}

		apps = append(apps, entry.Name())
	}

	return apps
}

type AppItem struct {
	Name string   `json:"name"`
	Dir  string   `json:"dir"`
	Urls []string `json:"urls"`
}

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
			var items []AppItem
			for _, a := range ListApps(rootDir) {
				appDir := filepath.Join(rootDir, a)
				item := AppItem{
					Name: a,
					Dir:  strings.Replace(appDir, os.Getenv("HOME"), "~", 1),
					Urls: []string{fmt.Sprintf("https://%s.%s", a, k.String("domain"))},
				}

				if cnamePath := filepath.Join(appDir, "CNAME"); utils.FileExists(cnamePath) {
					cname, err := os.ReadFile(cnamePath)
					if err == nil {
						item.Urls = []string{fmt.Sprintf("https://%s", strings.TrimSpace(string(cname))), item.Urls[0]}
					}
				}

				items = append(items, item)
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(items); err != nil {
					return fmt.Errorf("failed to encode tree: %w", err)
				}

				return nil
			}

			if len(items) == 0 {
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
			for _, app := range items {
				printer.AddField(app.Name)
				printer.AddField(strings.Replace(app.Dir, os.Getenv("HOME"), "~", 1))
				printer.AddField(app.Urls[0])

				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
