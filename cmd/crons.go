package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
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
		app  string
	}

	cmd := &cobra.Command{
		Use:     "crons",
		Aliases: []string{"cron"},
		Args:    cobra.NoArgs,
		Short:   "List cron jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := rootDir
			apps, err := app.ListApps(rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			crons := make([]CronItem, 0)
			for _, name := range apps {
				if cmd.Flags().Changed("app") && flags.app != name {
					continue
				}

				app, err := app.NewApp(name, rootDir, k.String("domain"), slices.Contains(k.Strings("adminApps"), name))
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

	cmd.Flags().StringVar(&flags.app, "app", "", "filter by app")
	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	_ = cmd.RegisterFlagCompletionFunc("app", completeApp(rootDir))

	return cmd
}

func CronRunner() *cron.Cron {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser))
	_, _ = c.AddFunc("* * * * *", func() {
		rounded := time.Now().Truncate(time.Minute)
		rootDir := utils.ExpandTilde(k.String("dir"))
		apps, err := app.ListApps(rootDir)
		if err != nil {
			fmt.Println(err)
		}

		for _, name := range apps {
			a, err := app.NewApp(name, rootDir, k.String("domain"), slices.Contains(k.Strings("adminApps"), name))
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

				wk := worker.NewWorker(a, rootDir, k.String("domain"))

				command, err := wk.Command(job.Args...)
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
