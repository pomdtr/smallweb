package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdLog() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "log",
		Short:   "Show logs",
		GroupID: CoreGroupID,
	}

	cmd.AddCommand(NewCmdLogHttp())
	cmd.AddCommand(NewCmdLogCron())
	cmd.AddCommand(NewCmdLogConsole())

	return cmd

}

func NewCmdLogHttp() *cobra.Command {
	var flags struct {
		host string
		json bool
	}

	cmd := &cobra.Command{
		Use:   "http",
		Short: "Show HTTP logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// use api unix socket if available
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("unix", api.SocketPath)
					},
				},
			}

			req, err := http.NewRequest("GET", "http://unix/v0/logs/http", nil)
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
				return fmt.Errorf("failed to get logs: %s", resp.Status)
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

				var log api.HttpLog
				if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
					return fmt.Errorf("failed to parse log: %w", err)
				}

				fmt.Printf("%s %s %s %s %d\n", log.Time.Format(time.RFC3339), log.Request.Host, log.Request.Method, log.Request.Path, log.Response.Status)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.host, "host", "", "filter logs by host")
	cmd.Flags().BoolVar(&flags.json, "json", false, "output logs in JSON format")

	return cmd
}

func NewCmdLogCron() *cobra.Command {
	var flags struct {
		host string
		json bool
	}
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Show cron logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// use api unix socket if available
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("unix", api.SocketPath)
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

func NewCmdLogConsole() *cobra.Command {
	var flags struct {
		json bool
		app  string
	}

	cmd := &cobra.Command{
		Use:   "console",
		Short: "Show console logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// use api unix socket if available
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("unix", api.SocketPath)
					},
				},
			}

			req, err := http.NewRequest("GET", "http://unix/v0/logs/console", nil)
			if err != nil {
				return err
			}

			q := req.URL.Query()
			if flags.app != "" {
				q.Add("app", flags.app)
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
				}

				if flags.json {
					fmt.Println(scanner.Text())
					continue
				}

				var log api.ConsoleLog
				if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
					return fmt.Errorf("failed to parse log: %w", err)
				}

				if log.Type == "stderr" {
					fmt.Fprintln(os.Stderr, log.Text)
					continue
				}

				fmt.Fprintln(os.Stdout, log.Text)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output logs in JSON format")
	cmd.Flags().StringVar(&flags.app, "app", "", "filter logs by app")
	cmd.RegisterFlagCompletionFunc("app", completeApp(utils.ExpandTilde(k.String("dir"))))

	return cmd
}
