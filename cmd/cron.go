package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronItem struct {
	App string
	worker.CronJob
}

func ListCronItems(app string) ([]CronItem, error) {
	wk, err := worker.NewWorker(filepath.Join(rootDir, app))
	if err != nil {
		return nil, fmt.Errorf("could not create worker: %w", err)
	}

	var items []CronItem
	for _, job := range wk.Config.Crons {
		items = append(items, CronItem{App: app, CronJob: job})
	}

	return items, nil
}

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

func NewCmdCronList() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "List cron jobs",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return ListApps(), cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			apps := ListApps()

			var crons []CronItem
			for _, app := range apps {
				if len(args) > 0 && app != args[0] {
					continue
				}

				items, err := ListCronItems(app)
				if err != nil {
					continue
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

			printer.AddHeader([]string{"Name", "Schedule", "Args", "App"})
			for _, item := range crons {
				printer.AddField(item.Name)
				printer.AddField(item.Schedule)

				args, err := json.Marshal(item.Args)
				if err != nil {
					return fmt.Errorf("failed to marshal args: %w", err)
				}
				printer.AddField(string(args))

				printer.AddField(item.App)

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.RegisterFlagCompletionFunc("app", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return ListApps(), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func NewCmdCronTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <app> <job>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return ListApps(), cobra.ShellCompDirectiveNoFileComp
			}

			if len(args) == 1 {
				apps := ListApps()

				for _, app := range apps {
					if app != args[0] {
						continue
					}

					crons, err := ListCronItems(app)
					if err != nil {
						return nil, cobra.ShellCompDirectiveError
					}

					names := make([]string, 0, len(crons))
					for _, cron := range crons {
						names = append(names, cron.Name)
					}

					return names, cobra.ShellCompDirectiveDefault
				}

				return nil, cobra.ShellCompDirectiveError
			}

			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, app := range ListApps() {
				if app != args[0] {
					continue
				}

				crons, err := ListCronItems(app)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				for _, cron := range crons {
					if cron.Name != args[1] {
						continue
					}

					w, err := worker.NewWorker(filepath.Join(rootDir, app))
					if err != nil {
						return fmt.Errorf("could not create worker")
					}

					return w.Run(cron.Args)
				}

				return fmt.Errorf("could not find job")
			}

			return fmt.Errorf("could not find app")
		},
	}

	return cmd

}
