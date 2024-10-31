package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cli/browser"
	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <app>",
		Aliases: []string{"new"},
		Short:   "Create a new smallweb app",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewApiClient(k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}

			res, err := client.CreateAppWithResponse(cmd.Context(), api.CreateAppJSONRequestBody{
				Name: args[0],
			})

			if err != nil {
				return fmt.Errorf("failed to create app: %w", err)
			}

			if res.StatusCode() != http.StatusCreated {
				return fmt.Errorf("failed to create app: %s", res.Status())
			}

			cmd.Printf("App %s created\n", args[0])
			return nil

		},
	}

	return cmd
}

func NewCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "open [app]",
		Short:             "Open an app in the browser",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeApp(),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.RootDir()
			client, err := NewApiClient(k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}

			if len(args) == 0 {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				if filepath.Dir(cwd) != rootDir {
					return fmt.Errorf("no app specified and not in an app directory")
				}

				appname := filepath.Base(cwd)
				res, err := client.GetAppWithResponse(cmd.Context(), appname)
				if err != nil {
					return fmt.Errorf("failed to get app: %w", err)
				}
				if res.StatusCode() != http.StatusOK {
					return fmt.Errorf("app not found: %s", appname)
				}

				app := *res.JSON200
				if err := browser.OpenURL(app.Url); err != nil {
					return fmt.Errorf("failed to open browser: %w", err)
				}

				return nil
			}

			res, err := client.GetAppWithResponse(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}
			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("app not found: %s", args[0])
			}

			app := *res.JSON200
			if err := browser.OpenURL(app.Url); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			if err := browser.OpenURL(app.Url); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func NewCmdList() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewApiClient(k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}

			res, err := client.GetAppsWithResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get apps: %w", err)
			}

			if res.JSON200 == nil {
				return fmt.Errorf("failed to get apps: %s", res.Status())
			}

			apps := *res.JSON200
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
				printer.AddField(filepath.Join(utils.RootDir(), a.Name))
				printer.AddField(a.Url)

				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}

func NewCmdRename() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "rename [app] [new-name]",
		Short:             "Rename an app",
		Aliases:           []string{"move", "mv"},
		ValidArgsFunction: completeApp(),
		Args:              cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewApiClient(k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}

			res, err := client.UpdateAppWithResponse(cmd.Context(), args[0], api.UpdateAppJSONRequestBody{
				Name: args[1],
			})
			if err != nil {
				return fmt.Errorf("failed to rename app: %w", err)
			}

			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("failed to rename app: %s", res.Status())
			}

			cmd.Printf("App %s renamed to %s\n", args[0], args[1])
			return nil
		},
	}

	return cmd
}

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete an app",
		Aliases:           []string{"remove", "rm"},
		ValidArgsFunction: completeApp(),
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewApiClient(k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}

			res, err := client.DeleteAppWithResponse(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to delete app: %w", err)
			}

			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("failed to delete app: %s", res.Status())
			}

			cmd.Printf("App %s deleted\n", args[0])
			return nil
		},
	}

	return cmd
}

func completeApp() func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveDefault
		}

		client, err := NewApiClient(k.String("domain"))
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		res, err := client.GetAppsWithResponse(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		if res.StatusCode() != http.StatusOK {
			return nil, cobra.ShellCompDirectiveError
		}

		var apps []string
		for _, app := range *res.JSON200 {
			apps = append(apps, app.Name)
		}

		return apps, cobra.ShellCompDirectiveDefault
	}
}
