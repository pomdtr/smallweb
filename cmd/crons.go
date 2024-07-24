package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronItem struct {
	worker.CronJob
	App App `json:"app"`
}

func ListCronItems(domains map[string]string) ([]CronItem, error) {
	apps, err := ListApps(domains)
	if err != nil {
		return nil, err
	}

	items := make([]CronItem, 0)
	for _, app := range apps {
		config, err := worker.LoadConfig(app.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}

		for _, job := range config.Crons {
			items = append(items, CronItem{
				CronJob: job,
				App:     app,
			})
		}
	}

	return items, nil
}

func NewCmdCrons() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "crons",
		Aliases: []string{"cron"},
		Short:   "List cron jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			domains := expandDomains(k.StringMap("domains"))
			crons, err := ListCronItems(domains)
			if err != nil {
				return fmt.Errorf("failed to list cron jobs: %w", err)
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

			printer.AddHeader([]string{"Schedule", "Url", "Dir"})
			for _, item := range crons {
				printer.AddField(item.Schedule)
				printer.AddField(fmt.Sprintf("https://%s%s", item.App.Name, item.Path))
				printer.AddField(item.App.Dir)
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")

	return cmd
}
