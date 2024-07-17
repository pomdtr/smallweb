package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

type App struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Dir  string `json:"dir"`
}

func ListApps(domains map[string]string) ([]App, error) {
	apps := make([]App, 0)
	for domain, rootDir := range domains {
		if !IsWildcard(domain) {
			apps = append(apps, App{
				Name: domain,
				Url:  fmt.Sprintf("https://%s", domain),
				Dir:  rootDir,
			})
		}

		rootDomain := strings.SplitN(domain, ".", 2)[1]
		entries, err := os.ReadDir(rootDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			apps = append(apps, App{
				Name: fmt.Sprintf("%s.%s", entry.Name(), rootDomain),
				Url:  fmt.Sprintf("https://%s.%s", entry.Name(), rootDomain),
				Dir:  filepath.Join(rootDir, entry.Name()),
			})
		}
	}

	return apps, nil
}

func NewCmdDump(v *viper.Viper) *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		GroupID: CoreGroupID,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			apps, err := ListApps(extractDomains(v))
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

			printer.AddHeader([]string{"Name", "Url", "Dir"})
			for _, app := range apps {
				printer.AddField(app.Name)
				printer.AddField(app.Url)
				printer.AddField(app.Dir)
				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
