package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/utils"
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
		if !utils.IsGlob(rootDir) {
			apps = append(apps, App{
				Name: domain,
				Dir:  rootDir,
				Url:  fmt.Sprintf("https://%s/", domain),
			})
			continue
		}

		entries, err := filepath.Glob(rootDir)
		if err != nil {
			return nil, fmt.Errorf("failed to list apps: %w", err)
		}

		for _, entry := range entries {
			match, err := utils.ExtractGlobPattern(entry, rootDir)
			if err != nil {
				continue
			}

			hostname := strings.Replace(domain, "*", match, 1)

			apps = append(apps, App{
				Name: hostname,
				Url:  fmt.Sprintf("https://%s/", hostname),
				Dir:  strings.Replace(rootDir, "*", match, 1),
			})
		}

	}

	// sort by hostname
	slices.SortFunc(apps, func(a, b App) int {
		return strings.Compare(a.Url, b.Url)
	})

	return apps, nil
}

func NewCmdList(v *viper.Viper) *cobra.Command {
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

			printer.AddHeader([]string{"Name", "Dir", "Url"})
			for _, app := range apps {
				printer.AddField(app.Name)
				printer.AddField(app.Dir)
				printer.AddField(app.Url)
				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
