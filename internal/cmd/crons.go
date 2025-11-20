package cmd

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/internal/api"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronJob struct {
	App      string   `json:"app" mapstructure:"app"`
	Args     []string `json:"args" mapstructure:"args"`
	Schedule string   `json:"schedule" mapstructure:"schedule"`
}

func NewCmdCrons() *cobra.Command {
	var flags struct {
		json bool
		app  string
	}

	cmd := &cobra.Command{
		Use:               "crons",
		Aliases:           []string{"cron"},
		Short:             "List cron jobs",
		ValidArgsFunction: completeApp,
		RunE: func(cmd *cobra.Command, args []string) error {
			var crons []CronJob
			for _, jobConf := range conf.Slices("crons") {
				var cronItem CronJob
				if err := jobConf.Unmarshal("", &cronItem); err != nil {
					cmd.PrintErrf("failed to unmarshal cron job: %v\n", err)
					return ExitError{1}
				}

				if flags.app != "" && cronItem.App != flags.app {
					continue
				}

				crons = append(crons, cronItem)
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

			printer.AddHeader([]string{"App", "Args", "Schedule"})
			for _, item := range crons {
				printer.AddField(item.App)
				args, err := json.Marshal(item.Args)
				if err != nil {
					cmd.PrintErrf("failed to marshal args for app %s: %v\n", item.App, err)
					return ExitError{1}
				}
				printer.AddField(string(args))
				printer.AddField(item.Schedule)
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
		for _, jobConf := range conf.Slices("crons") {
			var job CronJob
			if err := jobConf.Unmarshal("", &job); err != nil {
				logger.Error("failed to unmarshal cron job", "error", err)
				continue
			}

			current := time.Now().Truncate(time.Minute)
			sched, err := parser.Parse(job.Schedule)
			if err != nil {
				logger.Error("failed to parse cron schedule", "app", job.App, "schedule", job.Schedule, "error", err)
				continue
			}

			if sched.Next(current.Add(-1*time.Second)) != current {
				continue
			}

			go func() {
				a, err := app.LoadApp(filepath.Join(conf.String("dir"), job.App))
				if err != nil {
					logger.Error("failed to load app", "app", job.App, "error", err)
					return
				}

				wk := worker.NewWorker(a, api.NewHandler("http://api.localhost", conf))
				logger.Info("running cron job", "app", job.App, "args", job.Args, "schedule", job.Schedule)

				if err := wk.Run(context.Background(), worker.RunParams{
					Args: job.Args,
				}); err != nil {
					logger.Error("failed to run command", "app", job.App, "args", job.Args, "error", err)
				}
			}()
		}
	})

	return c
}
