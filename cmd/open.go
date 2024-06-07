package cmd

import (
	"fmt"
	"log"
	"net"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/server"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func NewCmdOpen() *cobra.Command {
	return &cobra.Command{
		Use:   "open <app>",
		Short: "Open a smallweb app in the browser",
		Args:  cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			apps, err := listApps()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			return apps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewClientWithDefaults()
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}

			addr := fmt.Sprintf("%s:%d", client.Config.Host, client.Config.SSHPort)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Fatalf("could not dial: %v", err)
			}

			sshConn, _, reqs, err := ssh.NewClientConn(conn, addr, client.sshConfig)
			if err != nil {
				log.Fatalf("could not create client connection: %v", err)
			}

			go ssh.DiscardRequests(reqs)

			ok, payload, err := sshConn.SendRequest("user", true, nil)
			if err != nil {
				log.Fatalf("could not get user: %v", err)
			}
			if !ok {
				return fmt.Errorf("credentials not found, please run 'smallweb auth signup' or 'smallweb auth login'")
			}

			var user server.UserResponse
			if err := ssh.Unmarshal(payload, &user); err != nil {
				log.Fatalf("could not unmarshal user: %v", err)
			}

			url := fmt.Sprintf("https://%s-%s.%s", args[0], user.Name, client.Config.Host)
			return browser.OpenURL(url)
		},
	}
}
