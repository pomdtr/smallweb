package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "embed"

	"github.com/adrg/xdg"
	"github.com/pomdtr/smallweb/watcher"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	var flags struct {
		cron bool
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

			logFilename := filepath.Join(xdg.CacheHome, "smallweb", k.String("domain"), "logs", "http.json")
			if err := os.MkdirAll(filepath.Dir(logFilename), 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %v", err)
			}

			httpLogger := utils.NewLogger(&lumberjack.Logger{
				Filename:   logFilename,
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     28,
			})

			watcher, err := watcher.NewWatcher(utils.RootDir())
			if err != nil {
				return fmt.Errorf("failed to create watcher: %v", err)
			}

			go watcher.Start()
			defer watcher.Stop()

			httpServer := http.Server{
				ReadHeaderTimeout: 5 * time.Second,
				Handler: httpLogger.Middleware(&Handler{
					watcher: watcher,
					workers: make(map[string]*worker.Worker),
				}),
			}

			httpLn, err := getListener(k.String("addr"), utils.ExpandTilde(k.String("cert")), utils.ExpandTilde(k.String("key")))
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			fmt.Fprintf(os.Stderr, "Serving *.%s from %s on %s\n", k.String("domain"), utils.RootDir(), k.String("addr"))
			go httpServer.Serve(httpLn)

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
			httpServer.Close()
			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.cron, "cron", false, "run cron jobs")
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
	mu      sync.Mutex
	workers map[string]*worker.Worker
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rootDir := utils.RootDir()

	if r.Host == k.String("domain") {
		// if we are on the apex domain and www exists, redirect to www
		if _, err := os.Stat(filepath.Join(rootDir, "www")); err == nil {
			target := r.URL
			target.Scheme = r.Header.Get("X-Forwarded-Proto")
			if target.Scheme == "" {
				target.Scheme = "http"
			}

			target.Host = "www." + k.String("domain")
			http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !strings.HasSuffix(r.Host, "."+k.String("domain")) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var appname string
	if name, ok := k.StringMap("customDomains")[r.Host]; ok {
		appname = name
	} else {
		appname = strings.TrimSuffix(r.Host, "."+k.String("domain"))
	}

	wk, err := me.GetWorker(appname, rootDir, k.String("domain"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to get worker: %v", err)
		return
	}

	wk.ServeHTTP(w, r)
}

func (me *Handler) GetWorker(appname, rootDir, domain string) (*worker.Worker, error) {
	if wk, ok := me.workers[appname]; ok && wk.IsRunning() && me.watcher.GetAppMtime(appname).Before(wk.StartedAt) {
		return wk, nil
	}

	me.mu.Lock()
	defer me.mu.Unlock()

	wk := worker.NewWorker(appname, rootDir, domain)
	if err := wk.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %v", err)
	}

	me.workers[appname] = wk
	return wk, nil
}
