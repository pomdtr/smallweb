package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
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
	App      string   `json:"app"`
	Args     []string `json:"args"`
	Schedule string   `json:"schedule"`
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
			var crons []CronItem
			for _, appname := range k.MapKeys("apps") {
				if len(args) > 0 && appname != args[0] {
					continue
				}

				for _, job := range k.Slices(fmt.Sprintf("apps.%s.crons", appname)) {
					crons = append(crons, CronItem{
						App:      appname,
						Args:     job.Strings("args"),
						Schedule: job.String("schedule"),
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

func CronRunner(stdout, stderr io.Writer) *cron.Cron {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser))
	_, _ = c.AddFunc("* * * * *", func() {
		rounded := time.Now().Truncate(time.Minute)
		for _, appname := range k.MapKeys("apps") {
			for _, job := range k.Slices(fmt.Sprintf("apps.%s.crons", appname)) {
				sched, err := parser.Parse(job.String("schedule"))
				if err != nil {
					fmt.Println(err)
					continue
				}

				if sched.Next(rounded.Add(-1*time.Second)) != rounded {
					continue
				}

				a, err := app.LoadApp(appname, k.String("rootDir"), k.String("domain"), k.Bool(fmt.Sprintf("apps.%s.admin", appname)))
				if err != nil {
					fmt.Println(err)
					continue
				}
				wk := worker.NewWorker(a)

				command, err := wk.Command(context.Background(), job.Strings("args")...)
				if err != nil {
					fmt.Println(err)
					continue
				}
				command.Stdout = stdout
				command.Stderr = stderr

				if err := command.Run(); err != nil {
					log.Printf("failed to run command: %v", err)
				}
			}
		}
	})

	return c
}
