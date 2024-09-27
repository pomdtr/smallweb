package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdCron() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cron",
		Short:   "Manage cron jobs",
		GroupID: CoreGroupID,
	}

	cmd.AddCommand(NewCmdCronList())
	cmd.AddCommand(NewCmdCronTrigger())
	cmd.AddCommand(NewCmdCronLogs())
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

func completeApp(rootDir string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveDefault
		}

		apps, err := app.ListApps(rootDir)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}

		return apps, cobra.ShellCompDirectiveDefault
	}
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
			rootDir := utils.ExpandTilde(k.String("dir"))

			apps, err := app.ListApps(rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			var crons []CronItem
			for _, name := range apps {
				if cmd.Flags().Changed("app") && flags.app != name {
					continue
				}

				app, err := app.LoadApp(filepath.Join(rootDir, name), k.String("domain"))
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
	cmd.RegisterFlagCompletionFunc("app", completeApp(utils.ExpandTilde(k.String("dir"))))

	return cmd
}

func NewCmdCronTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <id>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))

			var completions []string
			apps, err := app.ListApps(rootDir)
			if err != nil {
				return nil, cobra.ShellCompDirectiveDefault
			}

			for _, name := range apps {
				app, err := app.LoadApp(filepath.Join(rootDir, name), k.String("domain"))
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
			rootDir := utils.ExpandTilde(k.String("dir"))
			parts := strings.Split(args[0], ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid job name")
			}

			appname, jobName := parts[0], parts[1]
			app, err := app.LoadApp(filepath.Join(rootDir, appname), k.String("domain"))
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

				w := worker.NewWorker(app, k.StringMap("env"))
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

func NewCmdCronLogs() *cobra.Command {
	var flags struct {
		host string
		json bool
	}
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show cron logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// use api unix socket if available
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("unix", apiSocketPath)
					},
				},
			}

			req, err := http.NewRequest("GET", "http://unix/v0/logs/cron", nil)
			if err != nil {
				return err
			}

			q := req.URL.Query()
			if flags.host != "" {
				q.Add("host", flags.host)
			}
			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to get logs: %s", resp.Body)
			}

			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				if scanner.Err() != nil {
					if scanner.Err().Error() == "EOF" {
						break
					}

					return fmt.Errorf("failed to read logs: %w", scanner.Err())
				}

				if flags.json {
					fmt.Println(scanner.Text())
					continue
				}

				var log api.CronLog
				if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
					return fmt.Errorf("failed to parse log: %w", err)
				}

				fmt.Printf("%s %s %s %d\n", log.Time.Format(time.RFC3339), log.Id, log.Schedule, log.ExitCode)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.host, "host", "", "filter logs by host")
	cmd.Flags().BoolVar(&flags.json, "json", false, "output logs in JSON format")

	return cmd
}
