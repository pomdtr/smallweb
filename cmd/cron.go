package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronItem struct {
	Job worker.CronJob `json:"schedule"`
	App App            `json:"app"`
}

func ListCronItems(domains map[string]string) ([]CronItem, error) {
	apps, err := ListApps(domains)
	if err != nil {
		return nil, err
	}

	items := make([]CronItem, 0)
	for _, app := range apps {
		w := worker.Worker{Dir: app.Dir}
		config, err := w.LoadConfig()
		if err != nil {
			continue
		}

		for _, job := range config.Crons {
			items = append(items, CronItem{
				Job: job,
				App: app,
			})
		}
	}

	return items, nil
}

func ListCronWithApps(domains map[string]string, app string) ([]CronItem, error) {
	items, err := ListCronItems(domains)
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(items, func(item CronItem) bool {
		return item.App.Url != app
	}), nil
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
		all  bool
		app  string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List cron jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			domains := expandDomains(k.StringMap("domains"))
			items, err := ListCronItems(domains)
			if err != nil {
				return fmt.Errorf("failed to list cron jobs: %w", err)
			}

			if flags.app != "" {
				var filteredItems []CronItem
				for _, item := range items {
					if item.App.Url == flags.app {
						filteredItems = append(filteredItems, item)
					}
				}

				items = filteredItems
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(items); err != nil {
					return fmt.Errorf("failed to encode cron jobs: %w", err)
				}
				return nil
			}

			if (len(items)) == 0 {
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

			printer.AddHeader([]string{"Name", "App", "Schedule", "Command"})
			for _, item := range items {
				printer.AddField(item.Job.Name)
				printer.AddField(item.App.Name)
				printer.AddField(item.Job.Schedule)

				cmd := exec.Command(item.Job.Command, item.Job.Args...)
				printer.AddField(cmd.String())

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().StringVar(&flags.app, "app", "", "filter by app")
	cmd.RegisterFlagCompletionFunc("app", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		apps, err := ListApps(expandDomains(k.StringMap("domains")))
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, app := range apps {
			completions = append(completions, app.Name)
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&flags.all, "all", false, "list all cron jobs")
	cmd.MarkFlagsMutuallyExclusive("app", "all")

	return cmd
}

func NewCmdCronTrigger() *cobra.Command {
	var flags struct {
		app string
	}

	cmd := &cobra.Command{
		Use:   "trigger <cron>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if flags.app == "" {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			items, err := ListCronWithApps(expandDomains(k.StringMap("domains")), flags.app)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, item := range items {
				completions = append(completions, item.Job.Name)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			crons, err := ListCronWithApps(expandDomains(k.StringMap("domains")), flags.app)
			if err != nil {
				return err
			}

			for _, item := range crons {
				if item.Job.Name == args[0] {
					w := worker.Worker{Dir: item.App.Dir, Env: k.StringMap("env")}
					cmd.PrintErrln("Triggering cron job", item.Job.Name)
					if err := w.Trigger(item.Job.Name); err != nil {
						return fmt.Errorf("failed to run cron job %s: %w", item.Job.Command, err)
					}

					return nil
				}
			}

			return fmt.Errorf("cron job not found: %s", args[0])
		},
	}

	cmd.Flags().StringVar(&flags.app, "app", "", "app cron job belongs to")
	cmd.RegisterFlagCompletionFunc("app", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		apps, err := ListApps(expandDomains(k.StringMap("domains")))
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, app := range apps {
			completions = append(completions, app.Url)
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}
