package api

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/knadh/koanf/v2"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"golang.org/x/net/webdav"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml ./openapi.json

var (
	SocketPath = filepath.Join(xdg.CacheHome, "smallweb", "api.sock")
)

//go:embed schemas
var schemas embed.FS

//go:embed index.html
var swaggerHomepage []byte

//go:embed dist
var swaggerDist embed.FS

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
		FileSystem: webdav.Dir(utils.ExpandTilde(k.String("dir"))),
		LockSystem: webdav.NewMemLS(),
		Prefix:     "/webdav",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/webdav") {
			webdavHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/schemas") {
			http.ServeFileFS(w, r, schemas, r.URL.Path)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/v0") {
			handler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "text/yaml")
			spec, err := GetSwagger()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			encoder.Encode(spec)
			return
		}

		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write(swaggerHomepage)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/dist") {
			server := http.FileServer(http.FS(swaggerDist))
			server.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}

// GetV0AppsAppEnv implements ServerInterface.
func (me *Server) GetV0AppsApp(w http.ResponseWriter, r *http.Request, appname string) {
	rootDir := utils.ExpandTilde(me.k.String("dir"))
	a, err := app.LoadApp(filepath.Join(rootDir, appname), me.k.String("domain"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	manifestPath := a.Manifest()
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

	var manifest Manifest
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

func (me *Server) GetV0Apps(w http.ResponseWriter, r *http.Request) {
	rootDir := utils.ExpandTilde(me.k.String("dir"))
	names, err := app.ListApps(rootDir)
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

		manifestPath := a.Manifest()
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

		var manifest Manifest
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

func (me *Server) PostV0RunApp(w http.ResponseWriter, r *http.Request, app string) {
	executable, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var body PostV0RunAppJSONRequestBody
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

func (me *Server) GetV0LogsHttp(w http.ResponseWriter, r *http.Request, params GetV0LogsHttpParams) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set the necessary headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
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

			var log HttpLog
			if err := json.Unmarshal(logMsg, &log); err != nil {
				fmt.Fprintln(os.Stderr, "failed to parse log:", err)
				continue
			}

			if log.Request.Host != *params.Host {
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

func (me *Server) GetV0LogsCron(w http.ResponseWriter, r *http.Request, params GetV0LogsCronParams) {
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
	w.Header().Set("Content-Type", "text/event-stream")
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

			var log CronLog
			if err := json.Unmarshal(logMsg, &log); err != nil {
				fmt.Fprintln(os.Stderr, "failed to parse log:", err)
				continue
			}

			if log.App != *params.App {
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

func (me *Server) GetV0LogsConsole(w http.ResponseWriter, r *http.Request, params GetV0LogsConsoleParams) {
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
	w.Header().Set("Content-Type", "text/event-stream")
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

			var log ConsoleLog
			if err := json.Unmarshal(logMsg, &log); err != nil {
				fmt.Fprintln(os.Stderr, "failed to parse log:", err)
				continue
			}

			if log.App != *params.App {
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
