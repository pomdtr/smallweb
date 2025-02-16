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

	"github.com/knadh/koanf/providers/posflag"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

type (
	Log struct {
		Time  time.Time `json:"time"`
		Level string    `json:"level"`
		Msg   string    `json:"msg"`
		Type  string    `json:"type"`
	}
	HttpLog struct {
		Time    time.Time `json:"time"`
		Level   string    `json:"level"`
		Msg     string    `json:"msg"`
		Type    string    `json:"type"`
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
		Json  string    `json:"json"`
		App   string    `json:"app"`
		Text  string    `json:"text"`
	}
)

func NewCmdLogs() *cobra.Command {
	var flags struct {
		template string
		all      bool
		logType  string
	}

	cmd := &cobra.Command{
		Use:               "logs [app]",
		Aliases:           []string{"log"},
		ValidArgsFunction: completeApp,
		Short:             "View app logs",
		Args:              cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && flags.all {
				return fmt.Errorf("cannot set both --all and specify an app")
			}

			flagProvider := posflag.Provider(cmd.Flags(), ".", k)
			_ = k.Load(flagProvider, nil)
			return nil
		},
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

			logFilename := utils.GetLogFilename(k.String("domain"))

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

			if flags.logType == "console" {
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
					var l Log
					if err := json.Unmarshal([]byte(line), &l); err != nil {
						return fmt.Errorf("failed to unmarshal log line: %w", err)
					}

					if l.Type != "console" {
						continue
					}

					var log ConsoleLog
					if err := json.Unmarshal([]byte(line), &log); err != nil {
						return fmt.Errorf("failed to unmarshal log line: %w", err)
					}

					if appName != "" && log.App != appName {
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

						if err := tmpl.Execute(cmd.OutOrStdout(), log); err != nil {
							return fmt.Errorf("failed to execute template: %w", err)
						}
						fmt.Println()

						continue
					}

					fmt.Println(log.Text)
				}
			}

			hosts := make(map[string]struct{})
			hosts[fmt.Sprintf("%s.%s", appName, k.String("domain"))] = struct{}{}

			for domain, app := range k.StringMap("additionalDomains") {
				if appName != "" && app != appName {
					continue
				}

				hosts[domain] = struct{}{}
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

				var l Log
				if err := json.Unmarshal([]byte(line), &l); err != nil {
					return fmt.Errorf("failed to unmarshal log line: %w", err)
				}

				if l.Type != "http" {
					continue
				}

				var log HttpLog
				if err := json.Unmarshal([]byte(line), &log); err != nil {
					return fmt.Errorf("failed to unmarshal log line: %w", err)
				}

				if _, ok := hosts[log.Request.Host]; !flags.all && !ok {
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

					if err := tmpl.Execute(cmd.OutOrStdout(), log); err != nil {
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
	_ = cmd.RegisterFlagCompletionFunc("app", completeApp)
	cmd.Flags().StringVar(&flags.logType, "type", "http", "log type (http, console)")
	_ = cmd.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"http", "console"}, cobra.ShellCompDirectiveNoFileComp))
	cmd.Flags().BoolVar(&flags.all, "all", false, "show logs for all apps")

	return cmd
}
