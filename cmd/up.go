package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	var flags struct {
		addr string
		cert string
		key  string
	}

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := utils.ExpandTilde(k.String("dir"))

			addr := flags.addr
			if addr == "" {
				if flags.cert != "" || flags.key != "" {
					addr = ":443"
				} else {
					addr = ":7777"
				}
			}

			server := http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Host == k.String("domain") {
						target := r.URL
						target.Scheme = "https"
						target.Host = "www." + k.String("domain")
						http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
					}

					apps, err := ListApps(k.String("domain"), root)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for _, app := range apps {
						if r.Host != app.Hostname {
							continue
						}

						wk, err := worker.NewWorker(app, k.StringMap("env"))
						if err != nil {
							w.WriteHeader(http.StatusNotFound)
							return
						}

						if err := wk.Start(); err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						wk.ServeHTTP(w, r)
						if err := wk.Stop(); err != nil {
							log.Printf("failed to stop worker: %v", err)
							return
						}
						return
					}

					// no app was found
					w.WriteHeader(http.StatusNotFound)
				}),
			}

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			c.AddFunc("* * * * *", func() {
				rounded := time.Now().Truncate(time.Minute)

				apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("dir")))
				if err != nil {
					fmt.Println(err)
					return
				}

				for _, app := range apps {
					w, err := worker.NewWorker(app, k.StringMap("env"))
					if err != nil {
						fmt.Println(err)
						continue
					}

					for _, job := range w.Config.Crons {
						sched, err := parser.Parse(job.Schedule)
						if err != nil {
							fmt.Println(err)
							continue
						}

						if sched.Next(rounded.Add(-1*time.Second)) != rounded {
							continue
						}

						go w.Run(job.Args)
					}

				}
			})

			go c.Start()

			if flags.cert != "" || flags.key != "" {
				if flags.cert == "" {
					return fmt.Errorf("TLS certificate file is required")
				}

				if flags.key == "" {
					return fmt.Errorf("TLS key file is required")
				}

				cert, err := tls.LoadX509KeyPair(flags.cert, flags.key)
				if err != nil {
					return fmt.Errorf("failed to load TLS certificate and key: %w", err)
				}

				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS12,
				}

				server.TLSConfig = tlsConfig

				cmd.Printf("Evaluation server listening on %s\n", addr)
				return server.ListenAndServeTLS(flags.cert, flags.key)
			}

			cmd.Printf("Evaluation server listening on %s\n", addr)
			return server.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&flags.addr, "addr", "", "Address to listen on")
	cmd.Flags().StringVar(&flags.cert, "--cert", "", "TLS certificate file path")
	cmd.Flags().StringVar(&flags.key, "--key", "", "TLS key file path")

	if err := k.Load(posflag.Provider(cmd.Flags(), ".", k), nil); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	return cmd
}
