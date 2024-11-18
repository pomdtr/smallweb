package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/browser"
	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/hashicorp/go-getter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/config"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

//go:embed embed/template/*
var initTemplate embed.FS

func NewCmdCreate() *cobra.Command {
	var flags struct {
		template string
	}

	cmd := &cobra.Command{
		Use:     "create <app>",
		Aliases: []string{"new"},
		Short:   "Create a new smallweb app",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.RootDir()
			appDir := filepath.Join(rootDir, args[0])
			if _, err := os.Stat(appDir); !os.IsNotExist(err) {
				return fmt.Errorf("directory already exists: %s", appDir)
			}

			if flags.template != "" {
				if err := getter.Get(appDir, flags.template); err != nil {
					return fmt.Errorf("failed to get template: %w", err)
				}

				cmd.Printf("App initialized, you can now access it at https://%s.%s\n", args[0], k.String("domain"))
				return nil
			}

			subFs, err := fs.Sub(initTemplate, "embed/template")
			if err != nil {
				return fmt.Errorf("failed to get template sub fs: %w", err)
			}

			if err := os.CopyFS(appDir, subFs); err != nil {
				return fmt.Errorf("failed to copy template: %w", err)
			}

			cmd.Printf("App initialized, you can now access it at https://%s.%s\n", args[0], k.String("domain"))
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.template, "template", "", "template to use")
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

			if len(args) == 0 {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				if filepath.Dir(cwd) != rootDir {
					return fmt.Errorf("no app specified and not in an app directory")
				}

				appname := filepath.Base(cwd)
				a, err := app.LoadApp(appname, config.Config{
					Dir:    rootDir,
					Domain: k.String("domain"),
				})
				if err != nil {
					return fmt.Errorf("failed to load app: %w", err)
				}

				if err := browser.OpenURL(a.URL); err != nil {
					return fmt.Errorf("failed to open browser: %w", err)
				}

				return nil
			}

			a, err := app.LoadApp(args[0], config.Config{
				Dir:    rootDir,
				Domain: k.String("domain"),
			})
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			if err := browser.OpenURL(a.URL); err != nil {
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
			rootDir := utils.RootDir()
			names, err := app.ListApps(rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			apps := make([]app.App, 0)
			for _, name := range names {
				a, err := app.LoadApp(name, config.Config{
					Dir:    rootDir,
					Domain: k.String("domain"),
				})
				if err != nil {
					return fmt.Errorf("failed to load app: %w", err)
				}

				apps = append(apps, a)
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
			for _, a := range apps {
				printer.AddField(a.Name)
				printer.AddField(strings.Replace(a.Dir, os.Getenv("HOME"), "~", 1))
				printer.AddField(a.URL)

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
			rootDir := utils.RootDir()
			src := filepath.Join(rootDir, args[0])
			dst := filepath.Join(rootDir, args[1])

			if _, err := os.Stat(src); os.IsNotExist(err) {
				return fmt.Errorf("app not found: %s", args[0])
			}

			if _, err := os.Stat(dst); !os.IsNotExist(err) {
				return fmt.Errorf("app already exists: %s", args[1])
			}

			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("failed to rename app: %w", err)
			}

			cmd.Printf("App %s renamed to %s\n", args[0], args[1])
			return nil
		},
	}

	return cmd
}

func NewCmdClone() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "clone [app] [new-name]",
		Short:             "Clone an app",
		Aliases:           []string{"cp", "copy", "fork"},
		ValidArgsFunction: completeApp(),
		Args:              cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.RootDir()
			src := filepath.Join(rootDir, args[0])
			dst := filepath.Join(rootDir, args[1])

			if _, err := os.Stat(src); os.IsNotExist(err) {
				return fmt.Errorf("app not found: %s", args[0])
			}

			if _, err := os.Stat(dst); !os.IsNotExist(err) {
				return fmt.Errorf("app already exists: %s", args[1])
			}

			fs := os.DirFS(src)
			if err := os.CopyFS(dst, fs); err != nil {
				return fmt.Errorf("failed to copy app: %w", err)
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
			rootDir := utils.RootDir()
			p := filepath.Join(rootDir, args[0])
			if _, err := os.Stat(p); os.IsNotExist(err) {
				return fmt.Errorf("app not found: %s", args[0])
			}

			if err := os.RemoveAll(p); err != nil {
				return fmt.Errorf("failed to delete app: %w", err)
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

		apps, err := app.ListApps(utils.RootDir())
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}

		return apps, cobra.ShellCompDirectiveDefault
	}
}
