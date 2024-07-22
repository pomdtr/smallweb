package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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

func ListCronWithApp(domains map[string]string, appName string) ([]CronItem, error) {
	app, err := GetApp(domains, appName)
	if err != nil {
		return nil, err
	}

	w := worker.Worker{Dir: app.Dir}
	config, err := w.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load config")
	}

	var items []CronItem
	for _, job := range config.Crons {
		items = append(items, CronItem{
			Job: job,
			App: app,
		})
	}

	return items, nil
}

func ListCronWithDir(domains map[string]string, dir string) ([]CronItem, error) {
	apps, err := GetAppsFromDir(domains, dir)
	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("no apps found in dir: %s", dir)
	}

	w := worker.Worker{Dir: apps[0].Dir}
	config, err := w.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load config")
	}

	var items []CronItem
	for _, job := range config.Crons {
		items = append(items, CronItem{
			Job: job,
			App: apps[0],
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
		all  bool
		app  string
		dir  string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List cron jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			domains := expandDomains(k.StringMap("domains"))
			var crons []CronItem
			if flags.all {
				c, err := ListCronItems(domains)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}
				crons = c
			} else if flags.app != "" {
				c, err := ListCronWithApp(domains, flags.app)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}
				crons = c
			} else if flags.dir != "" {
				c, err := ListCronWithDir(domains, flags.dir)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}
				crons = c
			} else {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working dir: %w", err)
				}

				c, err := ListCronWithDir(domains, wd)
				if err != nil {
					return fmt.Errorf("current dir is not a smallweb app, please specify --app, --dir or use --all")
				}
				crons = c
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

			printer.AddHeader([]string{"Name", "App", "Schedule", "Command"})
			for _, item := range crons {
				printer.AddField(item.Job.Name)
				printer.AddField(item.App.Name)
				printer.AddField(item.Job.Schedule)

				cmd := exec.Command(item.Job.Command.Name, item.Job.Command.Args...)
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
	cmd.Flags().BoolVar(&flags.all, "all", false, "list all cron jobs")
	cmd.Flags().StringVar(&flags.dir, "dir", "", "filter by dir")
	cmd.Flags().StringVar(&flags.app, "app", "", "filter by app")
	cmd.RegisterFlagCompletionFunc("app", completeApp)

	cmd.MarkFlagsMutuallyExclusive("all", "app", "dir")

	return cmd
}

func NewCmdCronTrigger() *cobra.Command {
	var flags struct {
		app string
		dir string
	}

	cmd := &cobra.Command{
		Use:   "trigger <cron>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var items []CronItem
			if flags.app == "" {
				i, err := ListCronWithApp(expandDomains(k.StringMap("domains")), flags.app)
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
				items = i
			} else if flags.dir != "" {
				i, err := ListCronWithDir(expandDomains(k.StringMap("domains")), flags.dir)
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
				items = i
			} else {
				wd, err := os.Getwd()
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}

				i, err := ListCronWithDir(expandDomains(k.StringMap("domains")), wd)
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
				items = i
			}

			var completions []string
			for _, item := range items {
				completions = append(completions, item.Job.Name)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var crons []CronItem
			if flags.app != "" {
				c, err := ListCronWithApp(expandDomains(k.StringMap("domains")), flags.app)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				crons = c
			} else if flags.dir != "" {
				c, err := ListCronWithDir(expandDomains(k.StringMap("domains")), flags.dir)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}
				crons = c
			} else {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working dir: %w", err)
				}

				c, err := ListCronWithDir(expandDomains(k.StringMap("domains")), wd)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				crons = c
			}

			for _, item := range crons {
				if item.Job.Name == args[0] {
					command := item.Job.Command
					w := worker.Worker{Dir: item.App.Dir, Env: k.StringMap("env")}
					cmd.PrintErrln("Triggering cron job", item.Job.Name)
					if err := w.Run(command.Name, command.Args...); err != nil {
						return fmt.Errorf("failed to run cron job %s: %w", item.Job.Command, err)
					}

					return nil
				}
			}

			return fmt.Errorf("cron job not found: %s", args[0])
		},
	}

	cmd.Flags().StringVar(&flags.app, "app", "", "app cron job belongs to")
	cmd.RegisterFlagCompletionFunc("app", completeApp)
	cmd.Flags().StringVar(&flags.dir, "dir", "", "dir cron job belongs to")
	cmd.MarkFlagsMutuallyExclusive("app", "dir")

	return cmd
}
