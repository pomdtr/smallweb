package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"golang.org/x/net/webdav"
)

type Auth struct {
	Type     AuthType `json:"type"`
	User     string   `json:"user"`
	Password string   `json:"password"`
}

type AuthType string

const (
	AuthTypeNone  AuthType = ""
	AuthTypeBasic AuthType = "basic"
)

func (a Auth) Wrap(next http.HandlerFunc) http.HandlerFunc {
	switch a.Type {
	case AuthTypeNone:
		return next
	case AuthTypeBasic:
		return func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != a.User || pass != a.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	default:
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
}

func NewCmdUp() *cobra.Command {
	var flags struct {
		tls           bool
		tlsCert       string
		tlsKey        string
		tlsCaCert     string
		tlsClientAuth string
	}

	var auth Auth
	k.Unmarshal("auth", &auth)

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			port := k.Int("port")
			if port == 0 && flags.tls {
				port = 443
			} else if port == 0 {
				port = 7777
			}

			host := k.String("host")
			addr := fmt.Sprintf("%s:%d", host, port)
			root := utils.ExpandTilde(k.String("root"))

			server := http.Server{
				Addr: addr,
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

						handler := wk.ServeHTTP
						if wk.Config.Private {
							handler = auth.Wrap(wk.ServeHTTP)
						}

						handler(w, r)
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

			if flags.tls {
				if flags.tlsCert == "" || flags.tlsKey == "" {
					return fmt.Errorf("TLS enabled, but no certificate or key provided")
				}

				cert, err := tls.LoadX509KeyPair(flags.tlsCert, flags.tlsKey)
				if err != nil {
					return fmt.Errorf("failed to load TLS certificate and key: %w", err)
				}

				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS12,
				}

				if flags.tlsCaCert != "" {
					caCert, err := os.ReadFile(flags.tlsCaCert)
					if err != nil {
						return fmt.Errorf("failed to read CA certificate: %w", err)
					}
					caCertPool := x509.NewCertPool()
					if !caCertPool.AppendCertsFromPEM(caCert) {
						return fmt.Errorf("failed to parse CA certificate")
					}
					tlsConfig.ClientCAs = caCertPool

					switch flags.tlsClientAuth {
					case "require":
						tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
					case "request":
						tlsConfig.ClientAuth = tls.RequestClientCert
					case "verify":
						tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
					default:
						return fmt.Errorf("invalid client auth option: %s", flags.tlsClientAuth)
					}
				}

				server.TLSConfig = tlsConfig

				return server.ListenAndServeTLS(flags.tlsCert, flags.tlsKey)
			}

			cmd.Printf("Evaluation server listening on http://%s\n", addr)
			go server.ListenAndServe()

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			c.AddFunc("* * * * *", func() {
				rounded := time.Now().Truncate(time.Minute)

				apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("root")))
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

			webdavPort := k.Int("webdav-port")
			if webdavPort == 0 && flags.tls {
				webdavPort = 443
			} else if webdavPort == 0 {
				webdavPort = 7778
			}

			var webdavHandler http.Handler = &webdav.Handler{
				FileSystem: webdav.Dir(root),
				LockSystem: webdav.NewMemLS(),
			}

			if auth.Type != AuthTypeNone {
				webdavHandler = auth.Wrap(webdavHandler.ServeHTTP)
			}

			webdavAddr := fmt.Sprintf("%s:%d", host, webdavPort)
			cmd.Printf("WebDav server listening on http://%s\n", webdavAddr)
			go http.ListenAndServe(webdavAddr, webdavHandler)

			// signal handling
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs

			return nil
		},
	}

	cmd.Flags().String("host", "localhost", "Host to listen on")
	cmd.Flags().IntP("port", "p", 0, "Port to listen on")
	cmd.Flags().Int("webdav-port", 0, "Port to listen on for webdav")
	cmd.Flags().BoolVar(&flags.tls, "tls", false, "Enable TLS")
	cmd.Flags().StringVar(&flags.tlsCert, "tls-cert", "", "TLS certificate file path")
	cmd.Flags().StringVar(&flags.tlsKey, "tls-key", "", "TLS key file path")
	cmd.Flags().StringVar(&flags.tlsCaCert, "tls-ca-cert", "", "TLS CA certificate file path")
	cmd.Flags().StringVar(&flags.tlsClientAuth, "tls-client-auth", "require", "TLS client auth mode (require, request, verify)")

	if err := k.Load(posflag.Provider(cmd.Flags(), ".", k), nil); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	return cmd
}
