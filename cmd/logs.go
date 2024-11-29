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
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func GetLogFilename(domain string) string {
	return filepath.Join(xdg.CacheHome, "smallweb", "logs", domain, "http.json")
}

func NewCmdLogs() *cobra.Command {
	var flags struct {
		json     bool
		template string
		app      string
	}

	cmd := &cobra.Command{
		Use:     "logs [remote]",
		Aliases: []string{"log"},
		Short:   "View app logs",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				remote := args[0]
				args := []string{remote, "smallweb", "logs"}
				if flags.app != "" {
					args = append(args, "--app", flags.app)
				}

				if flags.json {
					args = append(args, "--json")
				}

				if flags.template != "" {
					args = append(args, "--template", flags.template)
				}

				command := exec.Command("ssh", args...)

				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				return command.Run()
			}

			logFilename := GetLogFilename(k.String("domain"))
			if _, err := os.Stat(logFilename); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("log file does not exist: %s", logFilename)
				}

				return err
			}

			hosts := make(map[string]struct{})
			if flags.app != "" {
				hosts[fmt.Sprintf("%s.%s", flags.app, k.String("domain"))] = struct{}{}

				for domain, app := range k.StringMap("customDomains") {
					if app != flags.app {
						continue
					}

					hosts[domain] = struct{}{}
				}
			}

			// Open the log file
			f, err := os.Open(logFilename)
			if err != nil {
				return err
			}
			defer f.Close()

			_, _ = f.Seek(0, io.SeekEnd)
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

				var log utils.HttpLog
				if err := json.Unmarshal([]byte(line), &log); err != nil {
					return fmt.Errorf("failed to unmarshal log line: %w", err)
				}

				if flags.json {
					fmt.Println(line)
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

				if len(hosts) > 0 {
					if _, ok := hosts[log.Request.Host]; !ok {
						continue
					}
				}

				msg, err := formatLog(log)
				if err != nil {
					return fmt.Errorf("failed to format log line: %w", err)
				}

				fmt.Println(msg)
			}
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output logs in JSON format")
	cmd.Flags().StringVarP(&flags.app, "app", "a", "", "app to view logs for")
	cmd.Flags().StringVar(&flags.template, "template", "", "output logs using a Go template")
	_ = cmd.RegisterFlagCompletionFunc("app", completeApp(utils.RootDir()))

	return cmd
}

func formatLog(log utils.HttpLog) (string, error) {
	return fmt.Sprintf("%s %s %s %d %d", log.Time.Format(time.RFC3339), log.Request.Method, log.Request.Url, log.Response.Status, log.Response.Bytes), nil
}
