package cmd

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/database"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func NewCmdTunnel() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "tunnel",
		Short:             "Start a tunnel to a remote server (powered by localhost.run)",
		GroupID:           CoreGroupID,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ssh.Dial("tcp", "localhost.run:22", &ssh.ClientConfig{
				User:            "nokey",
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         5 * time.Second,
			})
			if err != nil {
				return fmt.Errorf("failed to dial: %v", err)
			}

			db, err := database.OpenDB(filepath.Join(DataDir(), "smallweb.db"))
			if err != nil {
				return fmt.Errorf("failed to open database: %v", err)
			}

			appHandler := AppHandler{
				db:        db,
				apiServer: api.NewHandler(k, nil, nil, nil),
				logger:    slog.New(slog.NewJSONHandler(os.Stderr, nil)),
			}

			server := http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					rootDir := utils.ExpandTilde(k.String("dir"))
					app, err := app.LoadApp(filepath.Join(rootDir, args[0]), k.String("domain"))
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					appHandler.ServeApp(w, r, app)
				}),
			}

			socketPath := filepath.Join(os.TempDir(), "smallweb.sock")

			go func() {
				_, _, err := client.SendRequest("tcpip-forward", true, ssh.Marshal(remoteForwardRequest{
					BindPort: 80,
				}))
				if err != nil {
					log.Printf("failed to request forward: %v", err)
					return
				}

				channels := client.HandleChannelOpen("forwarded-tcpip")

				for channel := range channels {
					ch, reqs, err := channel.Accept()
					if err != nil {
						log.Printf("failed to accept: %v", err)
						continue
					}

					go ssh.DiscardRequests(reqs)

					c, err := net.Dial("unix", socketPath)
					if err != nil {
						ch.Close()
						log.Printf("failed to dial: %v", err)
						continue
					}

					go func() {
						defer ch.Close()
						defer c.Close()

						io.Copy(c, ch)
					}()

					go func() {
						defer c.Close()
						defer ch.Close()

						io.Copy(ch, c)
					}()
				}
			}()

			session, err := client.NewSession()
			if err != nil {
				return fmt.Errorf("failed to create session: %v", err)
			}

			session.Stdin = os.Stdin
			session.Stdout = os.Stdout
			session.Stderr = os.Stderr

			if err := session.Shell(); err != nil {
				return fmt.Errorf("failed to start shell: %v", err)
			}

			ln, err := net.Listen("unix", socketPath)
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			go server.Serve(ln)

			if err := session.Wait(); err != nil {
				return fmt.Errorf("failed to wait: %v", err)
			}

			server.Close()
			if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove socket: %v", err)
			}

			return nil
		},
	}

	return cmd
}

type remoteForwardRequest struct {
	BindAddr string
	BindPort uint32
}
