package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
		Use:               "crons [domain]",
		Aliases:           []string{"cron"},
		Short:             "List cron jobs",
		ValidArgsFunction: completeApp,
		RunE: func(cmd *cobra.Command, args []string) error {
			var crons []CronItem
			apps, err := ListApps(k.String("dir"))
			if err != nil {
				cmd.PrintErrf("failed to list apps: %v\n", err)
				return ExitError{1}
			}

			for _, appname := range apps {
				if flags.app != "" && flags.app != appname {
					continue
				}

				a, err := app.LoadApp(k.String("dir"), appname)
				if err != nil {
					cmd.PrintErrf("failed to load app %s: %v\n", appname, err)
					return ExitError{1}
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

			printer.AddHeader([]string{"Schedule", "App", "Args"})
			for _, item := range crons {
				printer.AddField(item.Schedule)
				printer.AddField(item.App)
				args, err := json.Marshal(item.Args)
				if err != nil {
					cmd.PrintErrf("failed to marshal args for app %s: %v\n", item.App, err)
					return ExitError{1}
				}

				printer.AddField(string(args))

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

func CronRunner(rootDir string, logger *slog.Logger) *cron.Cron {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser))
	_, _ = c.AddFunc("* * * * *", func() {
		domains, err := ListApps(rootDir)
		if err != nil {
			logger.Error("failed to list apps", "error", err)
			return
		}

		for _, domain := range domains {
			a, err := app.LoadApp(rootDir, domain)
			if err != nil {
				logger.Error("failed to load app", "app", domain, "error", err)
				continue
			}

			current := time.Now().Truncate(time.Minute)
			for _, job := range a.Config.Crons {
				sched, err := parser.Parse(job.Schedule)
				if err != nil {
					logger.Error("failed to parse cron schedule", "app", domain, "schedule", job.Schedule, "error", err)
					continue
				}

				if sched.Next(current.Add(-1*time.Second)) != current {
					continue
				}

				a, err := app.LoadApp(k.String("dir"), domain)
				if err != nil {
					logger.Error("failed to load app", "app", domain, "error", err)
					continue
				}
				wk := worker.NewWorker(a, nil)

				command, err := wk.Command(context.Background(), job.Args)
				if err != nil {
					logger.Error("failed to create command", "app", domain, "args", job.Args, "error", err)
					continue
				}

				logger.Info("running cron job", "app", domain, "args", job.Args, "schedule", job.Schedule)
				go func() {
					if err := command.Run(); err != nil {
						logger.Error("failed to run command", "app", domain, "args", job.Args, "error", err)
					}
				}()
			}
		}
	})

	return c
}

func ListApps(rootDir string) ([]string, error) {
	dirs, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("could not read directory %s: %v", rootDir, err)
	}

	domains := make([]string, 0)
	for _, entry := range dirs {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if !entry.IsDir() {
			continue
		}

		subdirs, err := os.ReadDir(filepath.Join(rootDir, entry.Name()))
		if err != nil {
			continue
		}

		for _, subdir := range subdirs {
			if strings.HasPrefix(subdir.Name(), ".") {
				continue
			}

			if !subdir.IsDir() {
				continue
			}

			domains = append(domains, fmt.Sprintf("%s.%s", subdir.Name(), entry.Name()))
		}
	}

	return domains, nil
}
