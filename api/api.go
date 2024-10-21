package api

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/knadh/koanf/v2"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"golang.org/x/net/webdav"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml ./openapi.json

func SocketPath(domain string) string {
	return filepath.Join(xdg.CacheHome, "smallweb", "api", domain, "api.sock")
}

//go:embed schemas
var schemas embed.FS

//go:embed swagger
var swagger embed.FS

type Server struct {
	k             *koanf.Koanf
	httpWriter    *utils.MultiWriter
	cronWriter    *utils.MultiWriter
	consoleWriter *utils.MultiWriter
}

func NewHandler(k *koanf.Koanf, httpWriter *utils.MultiWriter, cronWriter *utils.MultiWriter, consoleWriter *utils.MultiWriter) http.Handler {
	server := &Server{k: k, httpWriter: httpWriter, cronWriter: cronWriter, consoleWriter: consoleWriter}
	handler := Handler(server)
	webdavHandler := webdav.Handler{
		FileSystem: webdav.Dir(k.String("dir")),
		LockSystem: webdav.NewMemLS(),
		Prefix:     "/webdav",
	}

	caddyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		if domain == k.String("domain") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		if !strings.HasSuffix(domain, "."+k.String("domain")) {
			http.Error(w, "invalid domain", http.StatusBadRequest)
			return
		}

		appname := strings.TrimSuffix(domain, "."+k.String("domain"))
		appDir := filepath.Join(k.String("dir"), appname)
		if _, err := app.LoadApp(appDir, k.String("domain")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/webdav") {
			webdavHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/schemas") {
			server := http.FileServer(http.FS(schemas))
			server.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/v0") {
			handler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/openapi.json" {
			spec, err := GetSwagger()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			scheme := "http"
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				scheme = "https"
			}

			spec.Servers = openapi3.Servers{
				{URL: scheme + "://" + r.Host},
			}

			w.Header().Set("Content-Type", "text/json")
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			encoder.Encode(spec)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/caddy/check") {
			http.StripPrefix("/caddy/check", caddyHandler).ServeHTTP(w, r)
			return
		}

		subfs, err := fs.Sub(swagger, "swagger")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server := http.FileServer(http.FS(subfs))
		server.ServeHTTP(w, r)
	})
}

// GetV0AppsAppEnv implements ServerInterface.
func (me *Server) GetApp(w http.ResponseWriter, r *http.Request, appname string) {
	a, err := app.LoadApp(filepath.Join(me.k.String("dir"), appname), me.k.String("domain"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	manifestPath := filepath.Join(a.Root(), "manifest.json")
	if !utils.FileExists(manifestPath) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(App{
			Name: a.Name,
			Url:  a.Url,
		})
		return
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(App{
			Name: a.Name,
			Url:  a.Url,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(App{
		Name:     a.Name,
		Url:      a.Url,
		Manifest: &manifest,
	})
}

func (me *Server) GetApps(w http.ResponseWriter, r *http.Request) {
	rootDir := me.k.String("dir")
	names, err := app.ListApps(me.k.String("dir"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apps []App
	for _, name := range names {
		a, err := app.LoadApp(filepath.Join(rootDir, name), me.k.String("domain"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		manifestPath := filepath.Join(a.Root(), "manifest.json")
		if !utils.FileExists(manifestPath) {
			apps = append(apps, App{
				Name: a.Name,
				Url:  a.Url,
			})
			continue
		}

		manifestBytes, err := os.ReadFile(manifestPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var manifest map[string]interface{}
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			continue
		}

		apps = append(apps, App{
			Name:     a.Name,
			Url:      a.Url,
			Manifest: &manifest,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (me *Server) RunApp(w http.ResponseWriter, r *http.Request, app string) {
	executable, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var body RunAppJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := exec.Command(executable, "run", app)
	cmd.Args = append(cmd.Args, body.Args...)
	cmd.Env = os.Environ()

	if strings.Contains(r.Header.Get("Accept"), "text/plain") {
		output, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, string(output), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write(output)
		return
	}

	var res CommandOutput
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.Code = exitErr.ExitCode()
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	res.Success = res.Code == 0
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (me *Server) GetHttpLogs(w http.ResponseWriter, r *http.Request, params GetHttpLogsParams) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set the necessary headers for SSE
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a new channel for this client to receive logs
	clientChan := make(chan []byte)
	me.httpWriter.AddClient(clientChan)
	defer me.httpWriter.RemoveClient(clientChan)

	// Listen to the client channel and send logs to the client
	for {
		select {
		case logMsg := <-clientChan:
			// Send the log message as SSE event
			if params.Host == nil {
				w.Write(logMsg)
				flusher.Flush() // Push data to the client
				continue
			}

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

			if host != *params.Host {
				continue
			}

			w.Write(logMsg)
			flusher.Flush() // Push data to the client
		case <-r.Context().Done():
			// If the client disconnects, stop the loop
			return
		}
	}
}

func (me *Server) GetCronLogs(w http.ResponseWriter, r *http.Request, params GetCronLogsParams) {
	if me.cronWriter == nil {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set the necessary headers for SSE
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a new channel for this client to receive logs
	clientChan := make(chan []byte)
	me.httpWriter.AddClient(clientChan)
	defer me.httpWriter.RemoveClient(clientChan)

	// Listen to the client channel and send logs to the client
	for {
		select {
		case logMsg := <-clientChan:
			// Send the log message as SSE event
			if params.App == nil {
				w.Write(logMsg)
				flusher.Flush() // Push data to the client
				continue
			}

			var log map[string]any
			if err := json.Unmarshal(logMsg, &log); err != nil {
				fmt.Fprintln(os.Stderr, "failed to parse log:", err)
				continue
			}

			app, ok := log["app"].(string)
			if !ok {
				continue
			}

			if app != *params.App {
				continue
			}

			w.Write(logMsg)
			flusher.Flush() // Push data to the client
		case <-r.Context().Done():
			// If the client disconnects, stop the loop
			return
		}
	}
}

func (me *Server) GetConsoleLogs(w http.ResponseWriter, r *http.Request, params GetConsoleLogsParams) {
	if me.consoleWriter == nil {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set the necessary headers for SSE
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a new channel for this client to receive logs
	clientChan := make(chan []byte)
	me.consoleWriter.AddClient(clientChan)
	defer me.consoleWriter.RemoveClient(clientChan)

	// Listen to the client channel and send logs to the client
	for {
		select {
		case logMsg := <-clientChan:
			// Send the log message as SSE event
			if params.App == nil {
				w.Write(logMsg)
				flusher.Flush() // Push data to the client
				continue
			}

			var log map[string]any
			if err := json.Unmarshal(logMsg, &log); err != nil {
				fmt.Fprintln(os.Stderr, "failed to parse log:", err)
				continue
			}

			app, ok := log["app"].(string)
			if !ok {
				continue
			}

			if app != *params.App {
				continue
			}

			w.Write(logMsg)
			flusher.Flush() // Push data to the client
		case <-r.Context().Done():
			// If the client disconnects, stop the loop
			return
		}
	}
}
