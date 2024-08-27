package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronItem struct {
	worker.CronJob
	App worker.App `json:"app"`
}

func ListCronItems(app worker.App) ([]CronItem, error) {
	items := make([]CronItem, 0)
	config, err := worker.LoadConfig(app.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	for _, job := range config.Crons {
		items = append(items, CronItem{
			CronJob: job,
			App:     app,
		})
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
				return completeApp(cmd, args, toComplete)
			}

			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("root"))

			apps, err := ListApps(k.String("domain"), rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			var crons []CronItem
			for _, app := range apps {
				if len(args) > 0 && app.Name != args[0] {
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

				printer.AddField(item.App.Name)

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.RegisterFlagCompletionFunc("app", completeApp)

	return cmd
}

func NewCmdCronTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <app> <job>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeApp(cmd, args, toComplete)
			}

			if len(args) == 1 {
				apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("root")))
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}

				for _, app := range apps {
					if app.Name != args[0] {
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
			apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("root")))
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			for _, app := range apps {
				if app.Name != args[0] {
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

					w, err := worker.NewWorker(app, k.StringMap("env"))
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
