package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/cli/browser"
	"github.com/spf13/cobra"
)

func GetApp(name string) (*App, error) {
	apps, err := ListApps()
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %v", err)
	}

	for _, app := range apps {
		if app.Name == name {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("app not found: %s", name)
}

func GetAppsFromDir(dir string) ([]App, error) {
	apps, err := ListApps()
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

func NewCmdOpen() *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "Open the current smallweb app in the browser",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			apps, err := ListApps()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, app := range apps {
				completions = append(completions, app.Name)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if err := browser.OpenURL(fmt.Sprintf("https://%s", args[0])); err != nil {
					return fmt.Errorf("failed to open browser: %v", err)
				}

				return nil
			}

			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %v", err)
			}

			apps, err := GetAppsFromDir(wd)
			if err != nil {
				return fmt.Errorf("failed to get app from dir: %v", err)
			}

			if len(apps) == 0 {
				return fmt.Errorf("no app found for current directory")
			}

			if len(apps) == 1 {
				if err := browser.OpenURL(apps[0].Url); err != nil {
					return fmt.Errorf("failed to open browser: %v", err)
				}

				return nil
			}
			var options []huh.Option[string]
			for _, app := range apps {
				options = append(options, huh.Option[string]{
					Key:   app.Name,
					Value: app.Name,
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
