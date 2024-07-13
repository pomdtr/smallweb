package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CronItem struct {
	Job worker.CronJob `json:"schedule"`
	App App            `json:"app"`
}

func ListCronItems() ([]CronItem, error) {
	apps, err := ListApps()
	if err != nil {
		return nil, err
	}

	items := make([]CronItem, 0)
	for _, app := range apps {
		w := worker.NewWorker(app.Path)
		config, err := w.LoadConfig()
		if err != nil {
			continue
		}

		for _, job := range config.Crons {
			items = append(items, CronItem{
				Job: job,
				App: app,
			})
		}
	}

	return items, nil
}

func ListCronItemsWithApp(app string) ([]CronItem, error) {
	jobs, err := ListCronItems()
	if err != nil {
		return nil, err
	}

	var items []CronItem
	for _, job := range jobs {
		if job.App.Name != app {
			continue
		}

		items = append(items, job)
	}

	return items, nil
}

func ListCronItemsWithDomain(domain string) ([]CronItem, error) {
	items, err := ListCronItems()
	if err != nil {
		return nil, err
	}

	var filtered []CronItem
	for _, item := range items {
		if strings.HasSuffix(item.App.Name, "."+domain) {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

func NewCmdCron() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cron",
		Short:   "Manage cron jobs",
		GroupID: CoreGroupID,
	}

	cmd.AddCommand(NewCmdCronList())
	cmd.AddCommand(NewCmdCronTrigger())
	return cmd
}

func NewCmdCronList() *cobra.Command {
	var flags struct {
		json   bool
		all    bool
		domain string
		app    string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List cron jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			var items []CronItem
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			if flags.all {
				i, err := ListCronItems()
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				items = i
			} else if flags.domain != "" {
				i, err := ListCronItemsWithDomain(flags.domain)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				items = i
			} else if flags.app != "" {
				i, err := ListCronItemsWithApp(flags.app)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				items = i
			} else if app, err := inferAppName(wd); err == nil {
				i, err := ListCronItemsWithApp(app)
				if err != nil {
					return fmt.Errorf("failed to list cron jobs: %w", err)
				}

				items = i
			} else {
				return fmt.Errorf("the --app flag or the --domain flag or the --all flag is required when not in a smallweb app directory")
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(items); err != nil {
					return fmt.Errorf("failed to encode cron jobs: %w", err)
				}
				return nil
			}

			if (len(items)) == 0 {
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

			printer.AddHeader([]string{"Name", "App", "Schedule", "Command"})
			for _, item := range items {
				printer.AddField(item.Job.Name)
				printer.AddField(item.App.Name)
				printer.AddField(item.Job.Schedule)

				cmd := exec.Command(item.Job.Command, item.Job.Args...)
				printer.AddField(cmd.String())

				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output as json")
	cmd.Flags().StringVar(&flags.app, "app", "", "filter by app")
	cmd.RegisterFlagCompletionFunc("app", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		apps, err := ListApps()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, app := range apps {
			completions = append(completions, app.Name)
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&flags.domain, "domain", "", "filter by domain")
	cmd.RegisterFlagCompletionFunc("domain", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		entries, err := os.ReadDir(worker.SMALLWEB_ROOT)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			completions = append(completions, entry.Name())
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&flags.all, "all", false, "list all cron jobs")
	cmd.MarkFlagsMutuallyExclusive("app", "all")

	return cmd
}

func inferAppName(dir string) (string, error) {
	parent := filepath.Dir(dir)
	if !(filepath.Dir(parent) == worker.SMALLWEB_ROOT) {
		return "", fmt.Errorf("not a smallweb app")
	}

	subdomain := filepath.Base(dir)
	domain := filepath.Base(parent)
	return fmt.Sprintf("%s.%s", subdomain, domain), nil
}

func NewCmdCronTrigger() *cobra.Command {
	var flags struct {
		app string
	}

	cmd := &cobra.Command{
		Use:   "trigger <cron>",
		Short: "Trigger a cron job",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			app := flags.app
			if app == "" {
				wd, err := os.Getwd()
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}

				a, err := inferAppName(wd)
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}

				app = a
			}

			items, err := ListCronItemsWithApp(app)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, item := range items {
				completions = append(completions, item.Job.Name)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app := flags.app
			if app == "" {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				a, err := inferAppName(wd)
				if err != nil {
					return fmt.Errorf("the --app flag is required when not in a smallweb app directory")
				}
				app = a
			}

			parts := strings.SplitN(app, ".", 2)
			subdomain, domain := parts[0], parts[1]

			appDir := filepath.Join(worker.SMALLWEB_ROOT, domain, subdomain)
			if !worker.Exists(appDir) {
				return fmt.Errorf("app not found")
			}

			w := worker.NewWorker(appDir)
			if err := w.Trigger(args[0]); err != nil {
				return fmt.Errorf("failed to run cron job: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.app, "app", "", "app cron job belongs to")
	cmd.RegisterFlagCompletionFunc("app", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		apps, err := ListApps()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, app := range apps {
			completions = append(completions, app.Name)
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}
