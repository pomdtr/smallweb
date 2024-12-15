package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"github.com/adrg/xdg"
	"github.com/knadh/koanf/providers/posflag"
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
		template string
		remote   string
		logType  string
	}

	cmd := &cobra.Command{
		Use:               "logs [app]",
		Aliases:           []string{"log"},
		ValidArgsFunction: completeApp,
		Short:             "View app logs",
		Args:              cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			flagProvider := posflag.Provider(cmd.Flags(), ".", k)
			_ = k.Load(flagProvider, nil)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var appName string
			if len(args) == 0 {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				if filepath.Dir(cwd) != k.String("dir") {
					return fmt.Errorf("no app specified and not in an app directory")
				}

				appName = filepath.Base(cwd)
			} else {
				appName = args[0]
			}

			if remote := k.String("remote"); remote != "" {
				cmd := exec.Command("ssh", remote, "smallweb", "logs", appName)
				cmd.Args = append(cmd.Args, args...)

				if flags.logType != "" {
					cmd.Args = append(cmd.Args, "--type", flags.logType)
				}

				if flags.template != "" {
					cmd.Args = append(cmd.Args, "--template", flags.template)
				}

				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin

				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to run remote command: %w", err)
				}

				return nil
			}

			var logFilename string
			if flags.logType == "console" {
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
					var log ConsoleLog
					if err := json.Unmarshal([]byte(line), &log); err != nil {
						return fmt.Errorf("failed to unmarshal log line: %w", err)
					}

					if log.App != appName {
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
			hosts[fmt.Sprintf("%s.%s", appName, k.String("domain"))] = struct{}{}

			for domain, app := range k.StringMap("customDomains") {
				if app != appName {
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

				var log HttpLog
				if err := json.Unmarshal([]byte(line), &log); err != nil {
					return fmt.Errorf("failed to unmarshal log line: %w", err)
				}

				if _, ok := hosts[log.Request.Host]; !ok {
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
	cmd.Flags().StringVar(&flags.remote, "remote", "", "ssh remote")
	_ = cmd.RegisterFlagCompletionFunc("app", completeApp)
	cmd.Flags().StringVar(&flags.logType, "type", "http", "log type")
	_ = cmd.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"http", "console"}, cobra.ShellCompDirectiveNoFileComp))

	return cmd
}
