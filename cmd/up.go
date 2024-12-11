package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	_ "embed"

	"github.com/caddyserver/certmagic"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/watcher"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	var flags struct {
		cron        bool
		addr        string
		onDemandTLS bool
		cert        string
		key         string
	}

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if k.String("domain") == "" {
				return fmt.Errorf("domain cannot be empty")
			}

			logFilename := GetLogFilename(k.String("domain"), "http")
			if err := os.MkdirAll(filepath.Dir(logFilename), 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %v", err)
			}

			httpLogger := utils.NewLogger(&lumberjack.Logger{
				Filename:   logFilename,
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     28,
			})

			watcher, err := watcher.NewWatcher(utils.RootDir)
			if err != nil {
				return fmt.Errorf("failed to create watcher: %v", err)
			}

			//nolint:errcheck
			go watcher.Start()
			defer watcher.Stop()

			consoleLogger := slog.New(slog.NewJSONHandler(&lumberjack.Logger{
				Filename:   GetLogFilename(k.String("domain"), "console"),
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     28,
			}, nil))

			handler := httpLogger.Middleware(&Handler{
				logger:  consoleLogger,
				watcher: watcher,
				workers: make(map[string]*worker.Worker),
			})

			if flags.onDemandTLS {
				certmagic.Default.OnDemand = &certmagic.OnDemandConfig{
					DecisionFunc: func(ctx context.Context, name string) error {
						appname, _, ok := lookupApp(name, k.String("domain"), k.StringMap("customDomains"))
						if !ok {
							return fmt.Errorf("failed to lookup app: %v", err)
						}

						if _, err := app.NewApp(appname, utils.RootDir, k.String("domain"), slices.Contains(k.Strings("adminApps"), appname)); err != nil {
							return fmt.Errorf("failed to load app: %v", err)
						}

						return nil
					},
				}

				fmt.Fprintf(os.Stderr, "Serving *.%s using on-demand TLS...\n", k.String("domain"))
				//nolint:errcheck
				go certmagic.HTTPS(nil, handler)
			} else {
				addr := flags.addr
				if addr == "" {
					if flags.cert != "" || flags.key != "" {
						addr = "0.0.0.0:443"
					} else {
						addr = "localhost:7777"
					}
				}

				listener, err := getListener(addr, flags.cert, flags.key)
				if err != nil {
					return fmt.Errorf("failed to get listener: %v", err)
				}

				fmt.Fprintf(os.Stderr, "Serving *.%s on %s...\n", k.String("domain"), addr)
				//nolint:errcheck
				go http.Serve(listener, handler)
			}

			if flags.cron {
				fmt.Fprintln(os.Stderr, "Starting cron jobs...")
				crons := CronRunner()
				crons.Start()
				defer crons.Stop()
			}

			// sigint handling
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			fmt.Fprintln(os.Stderr, "Shutting down server...")
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.addr, "addr", "", "address to listen on")
	cmd.Flags().BoolVar(&flags.onDemandTLS, "on-demand-tls", false, "enable on-demand TLS")
	cmd.Flags().StringVar(&flags.cert, "cert", "", "tls certificate file")
	cmd.Flags().StringVar(&flags.key, "key", "", "key file")
	cmd.Flags().BoolVar(&flags.cron, "cron", false, "enable cron jobs")

	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "cert")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "key")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "addr")

	return cmd
}

func getListener(addr, cert, key string) (net.Listener, error) {
	var config *tls.Config
	if cert != "" && key != "" {
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, fmt.Errorf("failed to load cert: %v", err)
		}

		config = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	if strings.HasPrefix(addr, "unix/") {
		socketPath := strings.TrimPrefix(addr, "unix/")

		if utils.FileExists(socketPath) {
			if err := os.Remove(socketPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing socket: %v", err)
			}
		}

		if config != nil {
			return tls.Listen("unix", utils.ExpandTilde(socketPath), config)
		}

		return net.Listen("unix", utils.ExpandTilde(socketPath))
	}

	addr = strings.TrimPrefix(addr, "tcp/")
	if config != nil {
		return tls.Listen("tcp", addr, config)
	}

	return net.Listen("tcp", addr)
}

type Handler struct {
	watcher *watcher.Watcher
	logger  *slog.Logger
	mu      sync.Mutex
	workers map[string]*worker.Worker
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rootDir := utils.RootDir

	appname, redirect, ok := lookupApp(r.Host, k.String("domain"), k.StringMap("customDomains"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if redirect {
		target := r.URL
		target.Scheme = r.Header.Get("X-Forwarded-Proto")
		if target.Scheme == "" {
			target.Scheme = "http"
		}

		target.Host = fmt.Sprintf("%s.%s", appname, r.Host)
		http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
		return
	}

	wk, err := me.GetWorker(appname, rootDir, k.String("domain"))
	if err != nil {
		if errors.Is(err, app.ErrAppNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to get worker: %v", err)
		return
	}

	wk.ServeHTTP(w, r)
}

func lookupApp(host string, domain string, customDomains map[string]string) (app string, redirect bool, found bool) {
	// check exact matches first
	for key, value := range customDomains {
		if value == "*" {
			continue
		}

		if key == host {
			return value, false, true
		}
	}

	if host == domain {
		return "www", true, true
	}

	// check for subdomains
	for key, value := range customDomains {
		if value != "*" {
			continue
		}

		if key == host {
			return "www", true, true
		}

		if strings.HasSuffix(host, "."+key) {
			return strings.TrimSuffix(host, "."+key), false, true
		}
	}

	if strings.HasSuffix(host, "."+domain) {
		return strings.TrimSuffix(host, "."+domain), false, true
	}

	return "", false, false
}

func (me *Handler) GetWorker(appname, rootDir, domain string) (*worker.Worker, error) {
	if wk, ok := me.workers[appname]; ok && wk.IsRunning() && me.watcher.GetAppMtime(appname).Before(wk.StartedAt) {
		return wk, nil
	}

	me.mu.Lock()
	defer me.mu.Unlock()

	a, err := app.NewApp(appname, rootDir, domain, slices.Contains(k.Strings("adminApps"), appname))
	if err != nil {
		return nil, fmt.Errorf("failed to load app: %w", err)
	}

	wk := worker.NewWorker(a, rootDir, domain)

	wk.Logger = me.logger
	if err := wk.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	me.workers[appname] = wk
	return wk, nil
}
