package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ListApps(domain string, rootDir string) ([]worker.App, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %w", err)
	}

	var apps []worker.App
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		app := worker.App{
			Name:     entry.Name(),
			Hostname: fmt.Sprintf("%s.%s", entry.Name(), domain),
			Root:     path.Join(rootDir, entry.Name()),
		}

		if cname := path.Join(rootDir, entry.Name(), "CNAME"); utils.FileExists(cname) {
			b, err := os.ReadFile(cname)
			if err != nil {
				continue
			}

			app.Hostname = strings.TrimSpace(string(b))
		}

		apps = append(apps, app)
	}

	return apps, nil
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
			apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("root")))
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
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

			printer.AddHeader([]string{"Name", "Root", "Url"})
			for _, app := range apps {
				printer.AddField(app.Name)
				printer.AddField(app.Root)
				printer.AddField(fmt.Sprintf("https://%s/", app.Hostname))
				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
