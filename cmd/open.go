package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/cli/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetApp(domains map[string]string, name string) (*App, error) {
	apps, err := ListApps(domains)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %v", err)
	}

	for _, app := range apps {
		if app.Hostname == name {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("app not found: %s", name)
}

func GetAppsFromDir(domains map[string]string, dir string) ([]App, error) {
	apps, err := ListApps(domains)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %v", err)
	}

	var foundApps []App
	for _, app := range apps {
		if app.Dir != dir {
			continue
		}

		foundApps = append(foundApps, app)
	}

	return foundApps, nil
}

func NewCmdOpen(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "open [dir]",
		Short: "Open the smallweb app specified by dir in the browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var dir string
			if len(args) == 0 {
				d, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current dir: %v", err)
				}
				dir = d
			} else {
				d, err := filepath.Abs(args[0])
				if err != nil {
					return fmt.Errorf("failed to get abs path: %v", err)
				}
				dir = d
			}

			apps, err := GetAppsFromDir(extractDomains(v), dir)
			if err != nil {
				return fmt.Errorf("failed to get app from dir: %v", err)
			}

			if len(apps) == 0 {
				return fmt.Errorf("no app found for provided dir")
			}

			if len(apps) == 1 {
				app := apps[0]
				if err := browser.OpenURL(fmt.Sprintf("https://%s", app.Hostname)); err != nil {
					return fmt.Errorf("failed to open browser: %v", err)
				}

				return nil
			}

			var options []huh.Option[string]
			for _, app := range apps {
				options = append(options, huh.Option[string]{
					Key:   app.Hostname,
					Value: app.Hostname,
				})
			}

			var app string
			input := huh.NewSelect[string]().Title("Select an app").Options(options...).Value(&app)

			if err := input.Run(); err != nil {
				return fmt.Errorf("failed to get app from user: %v", err)
			}

			if err := browser.OpenURL(fmt.Sprintf("https://%s", app)); err != nil {
				return fmt.Errorf("failed to open browser: %v", err)
			}

			return nil
		},
	}
}
