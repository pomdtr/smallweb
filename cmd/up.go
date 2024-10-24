package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
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
	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/auth"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

func NewCmdUp() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if k.String("domain") == "" {
				return fmt.Errorf("domain cannot be empty")
			}

			httpWriter := utils.NewMultiWriter()
			consoleWriter := utils.NewMultiWriter()

			consoleLogger := slog.New(slog.NewJSONHandler(consoleWriter, nil))

			apiHandler := api.NewHandler(k.String("domain"), httpWriter, consoleWriter)
			appHandler := &AppHandler{apiServer: apiHandler, logger: consoleLogger}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

				appHandler.ServeApp(w, r, a)
			})

			fmt.Fprintf(os.Stderr, "Serving *.%s from %s on %s\n", k.String("domain"), utils.RootDir(), k.String("addr"))
			httpLogger := utils.NewLogger(httpWriter)
			server := http.Server{
				Handler: httpLogger.Middleware(handler),
			}

			ln, err := getListener(k.String("addr"), utils.ExpandTilde(k.String("cert")), utils.ExpandTilde(k.String("key")))
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			go server.Serve(ln)

			// sigint handling
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			log.Println("Shutting down server...")
			server.Close()
			return nil
		},
	}

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

type AppHandler struct {
	apiServer http.Handler
	logger    *slog.Logger
}

func (me *AppHandler) ServeApp(w http.ResponseWriter, r *http.Request, a app.App) {
	var handler http.Handler
	if a.Entrypoint() == "smallweb:api" {
		handler = me.apiServer
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
				if utils.FileExists(p) {
					http.ServeFile(w, r, p)
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
		handler = worker.NewWorker(a, me.logger)
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

	if isPrivateRoute || strings.HasPrefix(r.URL.Path, "/_auth") {
		authMiddleware := auth.Middleware(k.String("auth"), k.String("email"), a.Name)
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
