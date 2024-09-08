package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdCron() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cron",
		Short:   "Manage cron jobs",
		GroupID: CoreGroupID,
	}

	cmd.AddCommand(NewCmdCronList())
	cmd.AddCommand(NewCmdCronTrigger())
	return cmd
}

type CronItem struct {
	ID  string `json:"id"`
	App string `json:"app"`
	app.CronJob
}

func ListCronItems(appname string) ([]CronItem, error) {
	rootDir := utils.ExpandTilde(k.String("dir"))
	appDir := filepath.Join(rootDir, appname)
	wk, err := app.NewApp(appDir, fmt.Sprintf("https://%s.%s/", appname, k.String("domain")), k.StringMap("env"))
	if err != nil {
		return nil, fmt.Errorf("could not create worker: %w", err)
	}

	var items []CronItem
	for _, job := range wk.Config.Crons {
		items = append(items, CronItem{CronJob: job, App: appname, ID: fmt.Sprintf("%s:%s", filepath.Base(appDir), job.Name)})
	}

	return items, nil
}

func NewCmdCronList() *cobra.Command {
	var flags struct {
		json bool
		app  string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List cron jobs",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			if len(args) == 0 {
				return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))

			var crons []CronItem
			for _, app := range ListApps(rootDir) {
				if cmd.Flags().Changed("app") && flags.app != app {
					continue
				}

				items, err := ListCronItems(app)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				crons = append(crons, items...)
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(crons); err != nil {
					return fmt.Errorf("failed to encode cron jobs: %w", err)
				}
				return nil
			}

			if (len(crons)) == 0 {
				cmd.Println("No cron jobs found")
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

			printer.AddHeader([]string{"ID", "Schedule", "Args", "Description"})
			for _, item := range crons {
				printer.AddField(item.ID)
				printer.AddField(item.Schedule)

				args, err := json.Marshal(item.Args)
				if err != nil {
					return fmt.Errorf("failed to marshal args: %w", err)
				}
				printer.AddField(string(args))
				printer.AddField(item.Description)

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.app, "app", "", "filter by app")
	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.RegisterFlagCompletionFunc("app", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		rootDir := utils.ExpandTilde(k.String("dir"))
		return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func NewCmdCronTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <id>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))

			var completions []string
			for _, app := range ListApps(rootDir) {
				jobs, err := ListCronItems(app)
				if err != nil {
					continue
				}

				for _, job := range jobs {
					completions = append(completions, fmt.Sprintf("%s\t%s", job.ID, job.Description))
				}
			}

			return completions, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			parts := strings.Split(args[0], ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid job name")
			}

			for _, a := range ListApps(rootDir) {
				if a != parts[0] {
					continue
				}

				crons, err := ListCronItems(a)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				for _, cron := range crons {
					if cron.Name != parts[1] {
						continue
					}

					w, err := app.NewApp(filepath.Join(rootDir, a), fmt.Sprintf("https://%s.%s/", cron.App, k.String("domain")), k.StringMap("env"))
					if err != nil {
						return fmt.Errorf("could not create worker")
					}

					return w.Run(cron.Args...)
				}

				return fmt.Errorf("could not find job")
			}

			return fmt.Errorf("could not find app")
		},
	}

	return cmd

}
