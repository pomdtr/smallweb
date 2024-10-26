package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdTunnel() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "tunnel",
		Short:             "Start a tunnel to a remote server (powered by localhost.run)",
		GroupID:           CoreGroupID,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp(utils.RootDir()),
		RunE: func(cmd *cobra.Command, args []string) error {
			server := http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					rootDir := utils.RootDir()
					app, err := app.LoadApp(filepath.Join(rootDir, args[0]), k.String("domain"))
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					apiUrl := fmt.Sprintf("http://127.0.0.1:%d", k.Int("apiPort"))
					ServeApp(w, r, app, apiUrl, nil)
				}),
			}

			ln, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			_, port, err := net.SplitHostPort(ln.Addr().String())
			if err != nil {
				return fmt.Errorf("failed to parse address: %v", err)
			}

			sshCommand := exec.Command("ssh", "-R", fmt.Sprintf("80:localhost:%s", port), "nokey@localhost.run")

			sshCommand.Stdin = os.Stdin
			sshCommand.Stdout = os.Stdout
			sshCommand.Stderr = os.Stderr

			go server.Serve(ln)

			if err := sshCommand.Run(); err != nil {
				return fmt.Errorf("failed to start ssh command: %v", err)
			}

			return nil
		},
	}

	return cmd
}
