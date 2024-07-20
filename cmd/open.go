package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func GetApp(domains map[string]string, name string) (App, error) {
	apps, err := ListApps(domains)
	if err != nil {
		return App{}, fmt.Errorf("failed to list apps: %v", err)
	}

	for _, app := range apps {
		if app.Name == name {
			return app, nil
		}
	}

	return App{}, fmt.Errorf("app not found: %s", name)
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

func NewCmdOpen() *cobra.Command {
	var flags struct {
		app string
		dir string
	}

	cmd := &cobra.Command{
		Use:     "open <dir>",
		Short:   "Open the smallweb app specified by dir in the browser",
		Args:    cobra.NoArgs,
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.app != "" {
				app, err := GetApp(expandDomains(k.StringMap("domains")), flags.app)
				if err != nil {
					return fmt.Errorf("failed to get app: %v", err)
				}

				if err := browser.OpenURL(app.Url); err != nil {
					return fmt.Errorf("failed to open browser: %v", err)
				}
				return nil
			}

			var dir = flags.dir
			if dir == "" {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working dir: %v", err)
				}
				dir = wd
			}

			apps, err := GetAppsFromDir(expandDomains(k.StringMap("domains")), dir)
			if err != nil {
				return fmt.Errorf("current dir is not a smallweb app, please provide the --app or --dir flag")
			}

			if len(apps) == 0 {
				return fmt.Errorf("no app found for provided dir")
			}

			if len(apps) == 1 {
				app := apps[0]
				if utils.IsGlob(app.Url) {
					return fmt.Errorf("cannot guess URL for app: %s", app.Name)
				}

				if err := browser.OpenURL(app.Url); err != nil {
					return fmt.Errorf("failed to open browser: %v", err)
				}

				return nil
			}

			var options []huh.Option[string]
			for _, app := range apps {
				options = append(options, huh.Option[string]{
					Key:   app.Name,
					Value: app.Url,
				})
			}

			var url string
			input := huh.NewSelect[string]().Title("Select an app").Options(options...).Value(&url)

			if err := input.Run(); err != nil {
				return fmt.Errorf("failed to get app from user: %v", err)
			}

			if err := browser.OpenURL(url); err != nil {
				return fmt.Errorf("failed to open browser: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.app, "app", "", "app to open")
	cmd.RegisterFlagCompletionFunc("app", completeApp)
	cmd.Flags().StringVar(&flags.dir, "dir", "", "dir to open")
	cmd.MarkFlagsMutuallyExclusive("app", "dir")

	return cmd
}
