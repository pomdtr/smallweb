package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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

			forwarder := server.NewForwarder(db)
			httpServer := http.Server{
				Addr:    fmt.Sprintf(":%d", config.HttpPort),
				Handler: server.NewHandler(db, forwarder),
			}

			emailer := server.NewValTownEmail(config.ValTownToken)
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
