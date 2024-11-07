package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/adrg/xdg"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/config"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"

	esbuild "github.com/evanw/esbuild/pkg/api"
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

			httpLogger := utils.NewLogger(&lumberjack.Logger{
				Filename:   filepath.Join(xdg.CacheHome, "smallweb", "http.log"),
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     28,
			})

			httpServer := http.Server{
				Handler: httpLogger.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ServeApps(w, r)
				})),
			}

			logSocketPath := filepath.Join(xdg.CacheHome, "smallweb", "smallweb.sock")
			if err := os.MkdirAll(filepath.Dir(logSocketPath), 0755); err != nil {
				return fmt.Errorf("failed to create log socket directory: %v", err)
			}
			defer os.Remove(logSocketPath)

			if _, err := os.Stat(logSocketPath); err == nil {
				if err := os.Remove(logSocketPath); err != nil {
					return fmt.Errorf("failed to remove existing log socket: %v", err)
				}
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

func ServeApps(w http.ResponseWriter, r *http.Request) {
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

	a, err := app.LoadApp(filepath.Join(rootDir, appname), k.String("domain"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var handler http.Handler
	if a.Entrypoint() == "smallweb:static" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if strings.HasPrefix(filepath.Base(r.URL.Path), ".") {
				http.Error(w, "file not found", http.StatusNotFound)
				return
			}

			p := filepath.Join(a.Root(), r.URL.Path)
			transformOptions := esbuild.TransformOptions{
				Target:       esbuild.ESNext,
				Format:       esbuild.FormatESModule,
				MinifySyntax: false,
				Sourcemap:    esbuild.SourceMapNone,
			}

			switch path.Ext(r.URL.Path) {
			case ".ts":
				transformOptions.Loader = esbuild.LoaderTS
				code, err := transpile(p, transformOptions)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				w.Header().Set("Content-Type", "application/javascript")
				w.Write(code)
				return
			case ".jsx":
				transformOptions.Loader = esbuild.LoaderJSX
				transformOptions.JSX = esbuild.JSXAutomatic
				code, err := transpile(p, transformOptions)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				w.Header().Set("Content-Type", "application/javascript")
				w.Write(code)
				return
			case ".tsx":
				transformOptions.Loader = esbuild.LoaderTSX
				transformOptions.JSX = esbuild.JSXAutomatic
				code, err := transpile(p, transformOptions)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				w.Header().Set("Content-Type", "application/javascript")
				w.Write(code)
				return
			default:
				if entry, err := os.Stat(p); err == nil {
					if !entry.IsDir() {
						http.ServeFile(w, r, p)
						return
					}

					if utils.FileExists(filepath.Join(p, "index.html")) {
						http.ServeFile(w, r, p+"/index.html")
						return
					}

					http.Error(w, "file not found", http.StatusNotFound)
					return
				}

				if utils.FileExists(p + ".html") {
					http.ServeFile(w, r, p+".html")
					return
				}

				http.Error(w, "file not found", http.StatusNotFound)
				return
			}
		})
	} else if strings.HasPrefix(a.Entrypoint(), "smallweb:") {
		http.Error(w, "invalid entrypoint", http.StatusInternalServerError)
		return
	} else {
		wk := worker.NewWorker(a, config.Config{
			Dir:    utils.RootDir(),
			Domain: k.String("domain"),
		})

		handler = wk
	}

	handler.ServeHTTP(w, r)
}

func transpile(p string, options esbuild.TransformOptions) ([]byte, error) {
	content, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	result := esbuild.Transform(string(content), options)

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf(result.Errors[0].Text)
	}

	return result.Code, nil
}
