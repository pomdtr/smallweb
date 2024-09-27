package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/pomdtr/smallweb/api"
	"github.com/spf13/cobra"
)

func NewCmdLogs() *cobra.Command {
	var flags struct {
		host string
		json bool
	}
	cmd := &cobra.Command{
		Use:     "logs",
		Short:   "Show logs",
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			// use api unix socket if available
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("unix", apiSocketPath)
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
