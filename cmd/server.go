package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/caarlos0/env/v6"
	"github.com/pomdtr/smallweb/server"
	"github.com/pomdtr/smallweb/server/storage"
	"github.com/spf13/cobra"
)

type ServerConfig struct {
	Host             string `env:"SMALLWEB_HOST" envDefault:"smallweb.run"`
	SSHPort          int    `env:"SMALLWEB_SSH_PORT" envDefault:"2222"`
	HttpPort         int    `env:"SMALLWEB_HTTP_PORT" envDefault:"8000"`
	TursoDatabaseURL string `env:"TURSO_DATABASE_URL"`
	TursoAuthToken   string `env:"TURSO_AUTH_TOKEN"`
	ValTownToken     string `env:"VALTOWN_TOKEN"`
	Debug            bool   `env:"SMALLWEB_DEBUG" envDefault:"false"`
}

func ServerConfigFromEnv() (*ServerConfig, error) {
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func NewCmdServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "server",
		Short:  "Start a smallweb server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := ServerConfigFromEnv()
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			db, err := storage.NewTursoDB(fmt.Sprintf("%s?authToken=%s", config.TursoDatabaseURL, config.TursoAuthToken))
			if err != nil {
				log.Fatalf("failed to open database: %v", err)
			}

			emailer := server.NewValTownEmail(config.ValTownToken)
			forwarder := server.NewForwarder(db)
			subdomainHandler := server.NewSubdomainHandler(db, forwarder)
			rootHandler := server.NewRootHandler(db, forwarder)

			httpServer := http.Server{
				Addr: fmt.Sprintf(":%d", config.HttpPort),
				Handler: server.NewAuthMiddleware(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						parts := strings.Split(r.Host, ".")
						if len(parts) == 3 {
							subdomainHandler.ServeHTTP(w, r)
							return
						}

						rootHandler.ServeHTTP(w, r)
					}), db,
				),
			}

			sshServer := server.NewSSHServer(config.SSHPort, db, forwarder, emailer)

			slog.Info("starting ssh server", slog.Int("port", config.SSHPort))
			go sshServer.ListenAndServe()
			slog.Info("starting http server", slog.Int("port", config.HttpPort))
			go httpServer.ListenAndServe()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			slog.Info("shutting down servers")
			httpServer.Close()
			sshServer.Close()
			return nil
		},
	}

	return cmd
}
