package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

type (
	HttpLog struct {
		Time    time.Time `json:"time"`
		Level   string    `json:"level"`
		Msg     string    `json:"msg"`
		Request struct {
			Url     string            `json:"url"`
			Host    string            `json:"host"`
			Method  string            `json:"method"`
			Path    string            `json:"path"`
			Headers map[string]string `json:"headers"`
		} `json:"request"`
		Response struct {
			Status  int     `json:"status"`
			Bytes   int     `json:"bytes"`
			Elapsed float64 `json:"elapsed"`
		} `json:"response"`
	}
	ConsoleLog struct {
		Time  time.Time `json:"time"`
		Level string    `json:"level"`
		Msg   string    `json:"msg"`
		Type  string    `json:"type"`
		App   string    `json:"app"`
		Text  string    `json:"text"`
	}
)

func GetLogFilename(domain string, logType string) string {
	return filepath.Join(xdg.CacheHome, "smallweb", "logs", domain, fmt.Sprintf("%s.json", logType))
}

func NewCmdLogs() *cobra.Command {
	var flags struct {
		app      string
		template string
		console  bool
	}

	cmd := &cobra.Command{
		Use:               "logs",
		Aliases:           []string{"log"},
		ValidArgsFunction: cobra.FixedCompletions([]string{"http", "console"}, cobra.ShellCompDirectiveNoFileComp),
		Short:             "View app logs",
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var logFilename string
			if flags.console {
				logFilename = GetLogFilename(k.String("domain"), "console")
			} else {
				logFilename = GetLogFilename(k.String("domain"), "http")
			}
			if _, err := os.Stat(logFilename); err != nil {
				if err := os.MkdirAll(filepath.Dir(logFilename), 0755); err != nil {
					return fmt.Errorf("failed to create log directory: %v", err)
				}

				file, err := os.Create(logFilename)
				if err != nil {
					return fmt.Errorf("failed to create log file: %v", err)
				}

				if err := file.Close(); err != nil {
					return fmt.Errorf("failed to close log file: %v", err)
				}
			}

			// Open the log file
			f, err := os.Open(logFilename)
			if err != nil {
				return err
			}
			defer f.Close()
			_, _ = f.Seek(0, io.SeekEnd)

			if flags.console {

				reader := bufio.NewReader(f)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							time.Sleep(1 * time.Second)
							continue
						}
						return err
					}
					var log ConsoleLog
					if err := json.Unmarshal([]byte(line), &log); err != nil {
						return fmt.Errorf("failed to unmarshal log line: %w", err)
					}

					if flags.app != "" && log.App != flags.app {
						continue
					}

					if flags.template != "" {
						tmpl, err := template.New("").Funcs(template.FuncMap{
							"json": func(v interface{}) string {
								b, _ := json.Marshal(v)
								return string(b)
							},
						}).Parse(flags.template)
						if err != nil {
							return fmt.Errorf("failed to parse template: %w", err)
						}

						if err := tmpl.Execute(os.Stdout, log); err != nil {
							return fmt.Errorf("failed to execute template: %w", err)
						}
						fmt.Println()

						continue
					}

					fmt.Println(log.Text)
				}
			}

			hosts := make(map[string]struct{})
			if flags.app != "" {
				appName := flags.app
				hosts[fmt.Sprintf("%s.%s", appName, k.String("domain"))] = struct{}{}

				for domain, app := range k.StringMap("customDomains") {
					if app != appName {
						continue
					}

					hosts[domain] = struct{}{}
				}
			}

			// Stream new lines as they are added
			reader := bufio.NewReader(f)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						time.Sleep(1 * time.Second)
						continue
					}
					return err
				}

				var log HttpLog
				if err := json.Unmarshal([]byte(line), &log); err != nil {
					return fmt.Errorf("failed to unmarshal log line: %w", err)
				}

				if _, ok := hosts[log.Request.Host]; flags.app != "" && !ok {
					continue
				}

				if flags.template != "" {
					tmpl, err := template.New("").Funcs(template.FuncMap{
						"json": func(v interface{}) string {
							b, _ := json.Marshal(v)
							return string(b)
						},
					}).Parse(flags.template)
					if err != nil {
						return fmt.Errorf("failed to parse template: %w", err)
					}

					if err := tmpl.Execute(os.Stdout, log); err != nil {
						return fmt.Errorf("failed to execute template: %w", err)
					}
					fmt.Println()

					continue
				}

				fmt.Printf("%s %s %s %d %d\n", log.Time.Format(time.RFC3339), log.Request.Method, log.Request.Url, log.Response.Status, log.Response.Bytes)
			}
		},
	}

	cmd.Flags().StringVar(&flags.template, "template", "", "output logs using a Go template")
	cmd.Flags().StringVar(&flags.app, "app", "", "filter by app")
	_ = cmd.RegisterFlagCompletionFunc("app", completeApp(k.String("dir")))
	cmd.Flags().BoolVar(&flags.console, "console", false, "output console logs")

	return cmd
}
