package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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

func NewCmdCron() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Manage cron jobs",
	}

	cmd.AddCommand(NewCmdCronList())
	cmd.AddCommand(NewCmdCronTrigger())
	cmd.AddCommand(NewCmdCronUp())

	return cmd
}

type CronItem struct {
	ID  string `json:"id"`
	App string `json:"app"`
	app.CronJob
}

func ListCronItems(app app.App) ([]CronItem, error) {
	var items []CronItem
	for _, job := range app.Config.Crons {
		items = append(items, CronItem{App: app.Name, ID: fmt.Sprintf("%s:%s", filepath.Base(app.Dir), job.Name), CronJob: job})
	}

	return items, nil
}

func NewCmdCronList() *cobra.Command {
	var flags struct {
		json bool
		app  string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List cron jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.RootDir
			apps, err := app.ListApps(rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			var crons []CronItem
			for _, name := range apps {
				if cmd.Flags().Changed("app") && flags.app != name {
					continue
				}

				app, err := app.NewApp(name, rootDir, k.String("domain"))
				if err != nil {
					return fmt.Errorf("failed to load app: %w", err)
				}

				items, err := ListCronItems(app)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
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

			printer.AddHeader([]string{"ID", "Schedule", "Args", "Description"})
			for _, item := range crons {
				printer.AddField(item.ID)
				printer.AddField(item.Schedule)

				args, err := json.Marshal(item.Args)
				if err != nil {
					return fmt.Errorf("failed to marshal args: %w", err)
				}
				printer.AddField(string(args))
				printer.AddField(item.Description)

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
	_ = cmd.RegisterFlagCompletionFunc("app", completeApp(utils.RootDir))

	return cmd
}

func NewCmdCronUp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the cron daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := CronRunner()
			fmt.Fprintln(os.Stderr, "Starting cron daemon...")
			runner.Start()

			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			ctx := runner.Stop()
			<-ctx.Done()
			return nil
		},
	}

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
			a, err := app.NewApp(name, rootDir, k.String("domain"))
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

				wk := worker.NewWorker(a.Name, rootDir, k.String("domain"), k.StringMap("env"))

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

func NewCmdCronTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <id>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := k.String("dir")

			var completions []string
			apps, err := app.ListApps(rootDir)
			if err != nil {
				return nil, cobra.ShellCompDirectiveDefault
			}

			for _, name := range apps {
				app, err := app.NewApp(name, rootDir, k.String("domain"))
				if err != nil {
					continue
				}

				jobs, err := ListCronItems(app)
				if err != nil {
					continue
				}

				for _, job := range jobs {
					completions = append(completions, fmt.Sprintf("%s\t%s", job.ID, job.Description))
				}
			}

			return completions, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := k.String("dir")
			parts := strings.Split(args[0], ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid job name")
			}

			appname, jobName := parts[0], parts[1]
			app, err := app.NewApp(appname, rootDir, k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}

			crons, err := ListCronItems(app)
			if err != nil {
				return fmt.Errorf("failed to list cron jobs: %w", err)
			}

			for _, cron := range crons {
				if cron.Name != jobName {
					continue
				}

				w := worker.NewWorker(app.Name, rootDir, k.String("domain"), k.StringMap("env"))
				command, err := w.Command(cron.Args...)
				if err != nil {
					return fmt.Errorf("failed to create command: %w", err)
				}

				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				if err := command.Run(); err != nil {
					return fmt.Errorf("failed to run command: %w", err)
				}
			}

			return fmt.Errorf("could not find job")

		},
	}

	return cmd
}
