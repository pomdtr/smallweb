package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronItem struct {
	App string `json:"app"`
	app.CronJob
}

func NewCmdCrons() *cobra.Command {
	var flags struct {
		json bool
		all  bool
	}

	cmd := &cobra.Command{
		Use:               "crons [app]",
		Aliases:           []string{"cron"},
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeApp,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && flags.all {
				return fmt.Errorf("cannot set both --all and specify an app")
			}

			return nil
		},
		Short: "List cron jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			var appName string
			if len(args) > 0 {
				appName = args[0]
			} else if !flags.all {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				if filepath.Dir(cwd) != k.String("dir") {
					return fmt.Errorf("no app specified and not in an app directory")
				}

				appName = filepath.Base(cwd)
			}

			apps, err := app.ListApps(k.String("dir"))
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			crons := make([]CronItem, 0)
			for _, name := range apps {
				if appName != "" && name != appName {
					continue
				}

				app, err := app.LoadApp(name, k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), name))
				if err != nil {
					return fmt.Errorf("failed to load app: %w", err)
				}

				for _, job := range app.Config.Crons {
					crons = append(crons, CronItem{
						App:     name,
						CronJob: job,
					})
				}
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

			printer.AddHeader([]string{"Schedule", "App", "Args"})
			for _, item := range crons {
				printer.AddField(item.Schedule)
				printer.AddField(item.App)
				printer.AddField(strings.Join(item.Args, " "))

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().BoolVar(&flags.all, "all", false, "show all cron jobs")

	return cmd
}

func CronRunner() *cron.Cron {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser))
	_, _ = c.AddFunc("* * * * *", func() {
		rounded := time.Now().Truncate(time.Minute)
		apps, err := app.ListApps(k.String("dir"))
		if err != nil {
			fmt.Println(err)
		}

		for _, name := range apps {
			a, err := app.LoadApp(name, k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), name))
			if err != nil {
				fmt.Println(err)
				continue
			}

			for _, job := range a.Config.Crons {
				sched, err := parser.Parse(job.Schedule)
				if err != nil {
					fmt.Println(err)
					continue
				}

				if sched.Next(rounded.Add(-1*time.Second)) != rounded {
					continue
				}

				wk := worker.NewWorker(a, k.String("dir"), k.String("domain"))

				command, err := wk.Command(context.Background(), job.Args...)
				if err != nil {
					fmt.Println(err)
					continue
				}
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				if err := command.Run(); err != nil {
					log.Printf("failed to run command: %v", err)
				}
			}
		}
	})

	return c
}
