package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type App struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Path string `json:"path"`
}

func ListApps() ([]App, error) {
	entries, err := os.ReadDir(worker.SMALLWEB_ROOT)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	apps := make([]App, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		subdomains, err := os.ReadDir(filepath.Join(worker.SMALLWEB_ROOT, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, subdomain := range subdomains {
			if !subdomain.IsDir() {
				continue
			}

			// Skip hidden files
			if strings.HasPrefix(subdomain.Name(), ".") {
				continue
			}

			apps = append(apps, App{
				Name: fmt.Sprintf("%s.%s", subdomain.Name(), entry.Name()),
				Url:  fmt.Sprintf("https://%s.%s", subdomain.Name(), entry.Name()),
				Path: filepath.Join(worker.SMALLWEB_ROOT, entry.Name(), subdomain.Name()),
			})

		}

	}

	return apps, nil
}

func listAppsWithDomain(domain string) ([]App, error) {
	apps, err := ListApps()
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	var filtered []App
	for _, app := range apps {
		if !strings.HasSuffix(app.Name, "."+domain) {
			continue
		}

		filtered = append(filtered, app)
	}

	return filtered, nil
}

func NewCmdDump() *cobra.Command {
	var flags struct {
		json   bool
		domain string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		GroupID: CoreGroupID,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var apps []App
			if flags.domain != "" {
				var err error
				apps, err = listAppsWithDomain(flags.domain)
				if err != nil {
					return fmt.Errorf("failed to list apps: %w", err)
				}
			} else {
				var err error
				apps, err = ListApps()
				if err != nil {
					return fmt.Errorf("failed to list apps: %w", err)
				}
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

			printer.AddHeader([]string{"Name", "URL", "Path"})
			for _, app := range apps {
				printer.AddField(app.Name)
				printer.AddField(app.Url)
				printer.AddField(app.Path)
				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().StringVar(&flags.domain, "domain", "", "filter by domain")
	cmd.RegisterFlagCompletionFunc("domain", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		entries, err := os.ReadDir(worker.SMALLWEB_ROOT)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			completions = append(completions, entry.Name())
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
