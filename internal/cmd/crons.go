package cmd

import (
	"context"
	"encoding/json"
	"fmt"
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
	App      string   `json:"app"`
	Args     []string `json:"args"`
	Schedule string   `json:"schedule"`
}

func NewCmdCrons() *cobra.Command {
	var flags struct {
		json bool
		app  string
	}

	cmd := &cobra.Command{
		Use:               "crons [app]",
		Aliases:           []string{"cron"},
		Short:             "List cron jobs",
		ValidArgsFunction: completeApp,
		RunE: func(cmd *cobra.Command, args []string) error {
			var crons []CronItem
			apps, err := app.LookupApps(k.String("dir"))
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			for _, appname := range apps {
				if flags.app != "" && flags.app != appname {
					continue
				}

				a, err := app.LoadApp(appname, k.String("dir"), k.String("domain"))
				if err != nil {
					return fmt.Errorf("failed to load app %s: %w", appname, err)
				}

				for _, job := range a.Config.Crons {
					crons = append(crons, CronItem{
						App:      appname,
						Args:     job.Args,
						Schedule: job.Schedule,
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

				printer = tableprinter.New(cmd.OutOrStdout(), true, width)
			} else {
				printer = tableprinter.New(cmd.OutOrStdout(), false, 0)
			}

			printer.AddHeader([]string{"Schedule", "App", "Args"})
			for _, item := range crons {
				printer.AddField(item.Schedule)
				printer.AddField(item.App)
				args, err := json.Marshal(item.Args)
				if err != nil {
					return fmt.Errorf("failed to marshal args: %w", err)
				}

				printer.AddField(string(args))

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().StringVar(&flags.app, "app", "", "filter by app name")
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
				wk := worker.NewWorker(a, k.Bool(fmt.Sprintf("apps.%s.admin", a.Name)), nil)

				command, err := wk.Command(context.Background(), job.Args)
				if err != nil {
					logger.Error("failed to create command", "app", appname, "args", job.Args, "error", err)
					continue
				}

				logger.Info("running cron job", "app", appname, "args", job.Args, "schedule", job.Schedule)
				go func() {
					if err := command.Run(); err != nil {
						logger.Error("failed to run command", "app", appname, "args", job.Args, "error", err)
					}
				}()
			}
		}
	})

	return c
}
