package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/gobwas/glob"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/auth"
	"github.com/pomdtr/smallweb/config"

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

			consoleWriter := utils.NewMultiWriter()
			httpWriter := utils.NewMultiWriter()
			httpLogger := utils.NewLogger(httpWriter)
			httpServer := http.Server{
				Handler: httpLogger.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ServeApps(w, r, false, httpWriter, consoleWriter)
				})),
			}

			httpLn, err := getListener(k.String("addr"), utils.ExpandTilde(k.String("cert")), utils.ExpandTilde(k.String("key")))
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			fmt.Fprintf(os.Stderr, "Serving *.%s from %s on %s\n", k.String("domain"), utils.RootDir(), k.String("addr"))
			go httpServer.Serve(httpLn)

			crons := CronRunner()
			if flags.cron {
				fmt.Fprintln(os.Stderr, "Starting cron jobs...")
				crons.Start()
			}

			// sigint handling
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			fmt.Fprintln(os.Stderr, "Shutting down server...")
			httpServer.Close()
			if flags.cron {
				fmt.Fprintln(os.Stderr, "Shutting down cron jobs...")
				ctx := crons.Stop()
				<-ctx.Done()
			}
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

func ServeApps(w http.ResponseWriter, r *http.Request, disableAuth bool, httpWriter *utils.MultiWriter, consoleWriter *utils.MultiWriter) {
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
	if r.URL.Path == "/_logs" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`TODO`))
		return
	} else if r.URL.Path == "/_logs/http" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if httpWriter == nil {
				http.Error(w, "logs not enabled", http.StatusInternalServerError)
				return
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/x-ndjson")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Create a new channel for this client to receive logs
			clientChan := make(chan []byte)
			httpWriter.AddClient(clientChan)
			defer httpWriter.RemoveClient(clientChan)

			// Listen to the client channel and send logs to the client
			for {
				select {
				case logMsg := <-clientChan:
					// Send the log message as SSE event

					var log map[string]any
					if err := json.Unmarshal(logMsg, &log); err != nil {
						fmt.Fprintln(os.Stderr, "failed to parse log:", err)
						continue
					}

					request, ok := log["request"].(map[string]any)
					if !ok {
						continue
					}

					host, ok := request["host"].(string)
					if !ok {
						continue
					}

					if host != r.Host {
						continue
					}

					w.Write(logMsg)
					flusher.Flush() // Push data to the client
				case <-r.Context().Done():
					// If the client disconnects, stop the loop
					return
				}
			}
		})
	} else if r.URL.Path == "/_logs/console" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if consoleWriter == nil {
				http.Error(w, "logs not enabled", http.StatusInternalServerError)
				return
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/x-ndjson")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Create a new channel for this client to receive logs
			clientChan := make(chan []byte)
			httpWriter.AddClient(clientChan)
			defer httpWriter.RemoveClient(clientChan)

			// Listen to the client channel and send logs to the client
			for {
				select {
				case logMsg := <-clientChan:
					// Send the log message as SSE event

					var log map[string]any
					if err := json.Unmarshal(logMsg, &log); err != nil {
						fmt.Fprintln(os.Stderr, "failed to parse log:", err)
						continue
					}

					name, ok := log["app"].(string)
					if !ok {
						continue
					}

					if name != a.Name {
						continue
					}

					w.Write(logMsg)
					flusher.Flush() // Push data to the client
				case <-r.Context().Done():
					// If the client disconnects, stop the loop
					return
				}
			}
		})
	} else if a.Entrypoint() == "smallweb:static" {
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
			Domain: k.String("domain"),
		})

		if consoleWriter != nil {
			handler := slog.NewJSONHandler(consoleWriter, nil)
			wk.Logger = slog.New(handler)
		}
		handler = wk
	}

	isPrivateRoute := a.Config.Private
	for _, publicRoute := range a.Config.PublicRoutes {
		glob := glob.MustCompile(publicRoute)
		if glob.Match(r.URL.Path) {
			isPrivateRoute = false
		}
	}

	for _, privateRoute := range a.Config.PrivateRoutes {
		glob := glob.MustCompile(privateRoute)
		if glob.Match(r.URL.Path) {
			isPrivateRoute = true
		}
	}

	if !disableAuth && (isPrivateRoute || strings.HasPrefix(r.URL.Path, "/_auth/") || strings.HasPrefix(r.URL.Path, "/_logs/")) {
		authMiddleware := auth.Middleware(k.String("email"))
		handler = authMiddleware(handler)
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
