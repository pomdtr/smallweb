package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdLog() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "log",
		Aliases: []string{"logs"},
		Short:   "Show logs",
		GroupID: CoreGroupID,
	}

	cmd.AddCommand(NewCmdLogHttp())
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
						return net.Dial("unix", api.SocketPath(k.String("domain")))
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

				var log map[string]any
				if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
					return fmt.Errorf("failed to parse log: %w", err)
				}

				request, ok := log["request"].(map[string]any)
				if !ok {
					return fmt.Errorf("failed to parse request")
				}

				response, ok := log["response"].(map[string]any)
				if !ok {
					return fmt.Errorf("failed to parse response")
				}

				time, ok := log["time"].(string)
				if !ok {
					return fmt.Errorf("failed to parse time")
				}

				host, ok := request["host"].(string)
				if !ok {
					return fmt.Errorf("failed to parse host")
				}

				method, ok := request["method"].(string)
				if !ok {
					return fmt.Errorf("failed to parse method")
				}

				path, ok := request["path"].(string)
				if !ok {
					return fmt.Errorf("failed to parse path")
				}

				status, ok := response["status"].(float64)
				if !ok {
					return fmt.Errorf("failed to parse status")
				}

				fmt.Printf("%s %s %s %s %d\n", time, host, method, path, int(status))
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
						return net.Dial("unix", api.SocketPath(k.String("domain")))
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

				var log map[string]any
				if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
					fmt.Fprintln(os.Stderr, "failed to parse log:", err)
					continue
				}

				logType, ok := log["type"].(string)
				if !ok {
					fmt.Fprintln(os.Stderr, "failed to parse type")
				}

				text, ok := log["text"].(string)
				if !ok {
					fmt.Fprintln(os.Stderr, "failed to parse text")
				}

				if logType == "stderr" {
					fmt.Fprintln(os.Stderr, text)
					continue
				}

				fmt.Println(text)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "output logs in JSON format")
	cmd.Flags().StringVar(&flags.app, "app", "", "filter logs by app")
	cmd.RegisterFlagCompletionFunc("app", completeApp(utils.RootDir()))

	return cmd
}
