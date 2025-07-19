package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdList() *cobra.Command {
	var flags struct {
		template     string
		templateFile string
		json         bool
		admin        bool
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if k.String("dir") == "" {
				return fmt.Errorf("smallweb directory not set")
			}

			names, err := app.LookupApps(k.String("dir"))
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			apps := make([]app.App, 0)
			for _, name := range names {
				apps = append(apps, app.App{
					Name:    name,
					BaseDir: filepath.Join(k.String("dir"), name),
					Domain:  fmt.Sprintf("%s.%s", name, k.String("domain")),
				})
			}

			if flags.json {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(apps); err != nil {
					return fmt.Errorf("failed to encode tree: %w", err)
				}

				return nil
			}

			if flags.template != "" {
				tmpl, err := template.New("").Funcs(
					template.FuncMap{
						"json": func(v interface{}) string {
							b, _ := json.Marshal(v)
							return string(b)
						},
					},
				).Parse(flags.template)
				if err != nil {
					return fmt.Errorf("failed to parse template: %w", err)
				}

				if err := tmpl.Execute(cmd.OutOrStdout(), apps); err != nil {
					return fmt.Errorf("failed to execute template: %w", err)
				}

				return nil
			}

			if flags.templateFile != "" {
				tmpl, err := template.New("").Funcs(template.FuncMap{
					"json": func(v interface{}) string {
						b, _ := json.Marshal(v)
						return string(b)
					},
				}).ParseFiles(flags.templateFile)
				if err != nil {
					return fmt.Errorf("failed to parse template file: %w", err)
				}

				if err := tmpl.Execute(cmd.OutOrStdout(), apps); err != nil {
					return fmt.Errorf("failed to execute template: %w", err)
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

				printer = tableprinter.New(cmd.OutOrStdout(), true, width)
			} else {
				printer = tableprinter.New(cmd.OutOrStdout(), false, 0)
			}

			printer.AddHeader([]string{"Name", "Dir", "Domain", "Admin"})
			for _, a := range apps {
				printer.AddField(a.Name)
				printer.AddField(strings.Replace(a.BaseDir, os.Getenv("HOME"), "~", 1))
				printer.AddField(a.Domain)

				if k.Bool(fmt.Sprintf("apps.%s.admin", a.Name)) {
					printer.AddField("Yes")
				} else {
					printer.AddField("No")
				}

				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().StringVar(&flags.template, "template", "", "template to use")
	cmd.Flags().StringVar(&flags.templateFile, "template-file", "", "template file to use")
	cmd.Flags().BoolVar(&flags.admin, "admin", false, "filter by admin")

	return cmd
}
