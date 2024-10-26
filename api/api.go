package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"golang.org/x/net/webdav"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml ./openapi.json

//go:embed swagger
var swagger embed.FS

type Server struct {
	domain        string
	httpWriter    *utils.MultiWriter
	consoleWriter *utils.MultiWriter
}

func NewHandler(domain string, httpWriter *utils.MultiWriter, consoleWriter *utils.MultiWriter) http.Handler {
	server := &Server{domain: domain, httpWriter: httpWriter, consoleWriter: consoleWriter}
	handler := Handler(server)
	webdavHandler := webdav.Handler{
		FileSystem: webdav.Dir(utils.RootDir()),
		LockSystem: webdav.NewMemLS(),
		Prefix:     "/webdav",
	}

	caddyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := r.URL.Query().Get("domain")
		if d == domain {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		if !strings.HasSuffix(d, "."+domain) {
			http.Error(w, "invalid domain", http.StatusBadRequest)
			return
		}

		appname := strings.TrimSuffix(d, "."+domain)
		appDir := filepath.Join(utils.RootDir(), appname)
		if _, err := app.LoadApp(appDir, domain); err != nil {
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

		if strings.HasPrefix(r.URL.Path, "/v0") {
			handler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/openapi.json" {
			spec, err := GetSwagger()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
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
	a, err := app.LoadApp(filepath.Join(utils.RootDir(), appname), me.domain)
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
	names, err := app.ListApps(utils.RootDir())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apps []App
	for _, name := range names {
		a, err := app.LoadApp(filepath.Join(utils.RootDir(), name), me.domain)
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

func (me *Server) GetHttpLogs(w http.ResponseWriter, r *http.Request, params GetHttpLogsParams) {
	if me.httpWriter == nil {
		http.Error(w, "Logging unsupported", http.StatusInternalServerError)
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

func (me *Server) GetConsoleLogs(w http.ResponseWriter, r *http.Request, params GetConsoleLogsParams) {
	if me.consoleWriter == nil {
		http.Error(w, "Logging unsupported", http.StatusInternalServerError)
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
