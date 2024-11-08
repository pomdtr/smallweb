package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdLogs() *cobra.Command {
	var flags struct {
		json bool
		app  string
	}

	cmd := &cobra.Command{
		Use:     "logs",
		Aliases: []string{"log"},
		Short:   "View app logs",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			logPath := filepath.Join(xdg.CacheHome, "smallweb", "http.log")
			if _, err := os.Stat(logPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("log file does not exist: %s", logPath)
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
			f, err := os.Open(logPath)
			if err != nil {
				return err
			}
			defer f.Close()

			f.Seek(0, io.SeekEnd)
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
	cmd.Flags().StringVar(&flags.app, "app", "", "app to view logs for")
	cmd.RegisterFlagCompletionFunc("app", completeApp())

	return cmd
}

func formatLog(log utils.HttpLog) (string, error) {
	return fmt.Sprintf("%s %s %s %d %d", log.Time.Format(time.RFC3339), log.Request.Method, log.Request.Url, log.Response.Status, log.Response.Bytes), nil
}
