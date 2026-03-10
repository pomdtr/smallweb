package cmd

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronItem struct {
	App string `json:"app"`
	app.CronJob
}

func NewCmdCron() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cron",
	}

	cmd.AddCommand(NewCmdCronList())
	cmd.AddCommand(NewCmdCronTrigger())

	return cmd
}

func NewCmdCronList() *cobra.Command {
	var flags struct {
		json bool
		app  string
	}

	cmd := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"cron"},
		Short:             "List cron jobs",
		ValidArgsFunction: completeApp,
		RunE: func(cmd *cobra.Command, args []string) error {
			var crons []CronItem
			apps, err := app.LookupApps(k.String("dir"))
			if err != nil {
				cmd.PrintErrf("failed to list apps: %v\n", err)
				return ExitError{1}
			}

			for _, appname := range apps {
				if flags.app != "" && flags.app != appname {
					continue
				}

				a, err := app.LoadApp(appname, k.String("dir"), k.String("domain"))
				if err != nil {
					cmd.PrintErrf("failed to load app %s: %v\n", appname, err)
					return ExitError{1}
				}

				for _, job := range a.Config.Crons {
					crons = append(crons, CronItem{
						App:     appname,
						CronJob: job,
					})
				}
			}

			if flags.json {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(crons); err != nil {
					cmd.PrintErrf("failed to encode cron jobs: %v\n", err)
					return ExitError{1}
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
					cmd.PrintErrf("failed to get terminal size: %v\n", err)
					return ExitError{1}
				}

				printer = tableprinter.New(cmd.OutOrStdout(), true, width)
			} else {
				printer = tableprinter.New(cmd.OutOrStdout(), false, 0)
			}

			printer.AddHeader([]string{"Schedule", "App", "Name"})
			for _, item := range crons {
				printer.AddField(item.Schedule)
				printer.AddField(item.App)
				printer.AddField(item.Name)

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				cmd.PrintErrf("failed to render table: %v\n", err)
				return ExitError{1}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().StringVarP(&flags.app, "app", "a", "", "filter by app name")
	cmd.RegisterFlagCompletionFunc("app", completeApp)

	return cmd
}

func CronRunner(logger *slog.Logger) *cron.Cron {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser))
	_, _ = c.AddFunc("* * * * *", func() {
		apps, err := app.LookupApps(k.String("dir"))
		if err != nil {
			logger.Error("failed to list apps", "error", err)
			return
		}

		for _, appname := range apps {
			a, err := app.LoadApp(appname, k.String("dir"), k.String("domain"))
			if err != nil {
				logger.Error("failed to load app", "app", appname, "error", err)
				continue
			}

			current := time.Now().Truncate(time.Minute)
			for _, job := range a.Config.Crons {
				sched, err := parser.Parse(job.Schedule)
				if err != nil {
					logger.Error("failed to parse cron schedule", "app", appname, "schedule", job.Schedule, "error", err)
					continue
				}

				if sched.Next(current.Add(-1*time.Second)) != current {
					continue
				}

				a, err := app.LoadApp(appname, k.String("dir"), k.String("domain"))
				if err != nil {
					logger.Error("failed to load app", "app", appname, "error", err)
					continue
				}
				wk := worker.NewWorker(a, nil)

				logger.Info("running cron job", "app", appname, "name", job.Name, "schedule", job.Schedule)
				go func() {
					if err := wk.TriggerCron(context.Background(), job); err != nil {
						logger.Error("failed to run command", "app", appname, "name", job.Name, "schedule", job.Schedule, "error", err)
					}
				}()
			}
		}
	})

	return c
}

func NewCmdCronTrigger() *cobra.Command {
	var flags struct {
		app string
	}

	cmd := &cobra.Command{
		Use:  "trigger <name>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.LoadApp(flags.app, k.String("dir"), k.String("domain"))
			if err != nil {
				return ExitError{1}
			}

			var job *app.CronJob
			for _, j := range a.Config.Crons {
				if j.Name == args[0] {
					job = &j
					break
				}
			}

			if job == nil {
				return ExitError{1}
			}

			wk := worker.NewWorker(a, nil)
			if err := wk.TriggerCron(cmd.Context(), *job); err != nil {
				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}
