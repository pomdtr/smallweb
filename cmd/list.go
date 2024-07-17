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
	Hostname string `json:"hostname"`
	Dir      string `json:"dir"`
}

func ListApps(domains map[string]string) ([]App, error) {
	apps := make([]App, 0)

	for domain, rootDir := range domains {
		if !utils.IsGlob(rootDir) {
			apps = append(apps, App{
				Hostname: domain,
				Dir:      rootDir,
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

			apps = append(apps, App{
				Hostname: strings.Replace(domain, "*", match, 1),
				Dir:      strings.Replace(rootDir, "*", match, 1),
			})
		}

	}

	// sort by hostname
	slices.SortFunc(apps, func(a, b App) int {
		return strings.Compare(a.Hostname, b.Hostname)
	})

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

			printer.AddHeader([]string{"Hostname", "Dir"})
			for _, app := range apps {
				printer.AddField(app.Hostname)
				printer.AddField(app.Dir)
				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
