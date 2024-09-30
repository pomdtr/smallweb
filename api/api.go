package api

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	xj "github.com/basgys/goxml2json"

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
			if r.Header.Get("Accept") == "application/json" {
				// use api unix socket if available
				client := &http.Client{
					Transport: &http.Transport{
						DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
							return net.Dial("unix", SocketPath)
						},
					},
				}

				req, err := http.NewRequest(r.Method, "http://unix"+r.URL.Path, r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				for k, v := range r.Header {
					if k == "Accept" {
						continue
					}

					req.Header[k] = v
				}

				resp, err := client.Do(req)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer resp.Body.Close()

				contentType := resp.Header.Get("Content-Type")
				if !strings.HasPrefix(contentType, "text/xml") {
					http.Error(w, "invalid content type", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				body, err := xj.Convert(resp.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Write(body.Bytes())
			}

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

		http.NotFound(w, r)
	})
}

// GetV0AppsAppConfig implements ServerInterface.
func (me *Server) GetV0AppsAppConfig(w http.ResponseWriter, r *http.Request, appname string) {
	rootDir := utils.ExpandTilde(me.k.String("dir"))
	a, err := app.LoadApp(filepath.Join(rootDir, appname), me.k.String("domain"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(a.Config)
}

// GetV0AppsAppEnv implements ServerInterface.
func (me *Server) GetV0AppsAppEnv(w http.ResponseWriter, r *http.Request, appname string) {
	rootDir := utils.ExpandTilde(me.k.String("dir"))
	a, err := app.LoadApp(filepath.Join(rootDir, appname), me.k.String("domain"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(a.Env)
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
		a, err := app.LoadApp(name, me.k.String("domain"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		apps = append(apps, App{
			Name: a.Name,
			Url:  a.Url,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (me *Server) GetV0Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	var cert *string
	if value := me.k.String("cert"); value != "" {
		cert = &value
	}

	var key *string
	if value := me.k.String("key"); value != "" {
		key = &value
	}

	var port *int
	if value := me.k.Int("port"); value != 0 {
		port = &value
	}

	var email *string
	if value := me.k.String("email"); value != "" {
		email = &value
	}

	var dir *string
	if value := me.k.String("dir"); value != "" {
		dir = &value
	}

	var domain *string
	if value := me.k.String("domain"); value != "" {
		domain = &value
	}

	var customDomains *map[string]string
	if value := me.k.StringMap("customDomains"); len(value) > 0 {
		customDomains = &value
	}

	var env *map[string]string
	if value := me.k.StringMap("env"); len(value) > 0 {
		env = &value
	}

	var host *string
	if value := me.k.String("host"); value != "" {
		host = &value
	}

	encoder.Encode(Config{
		Dir:           dir,
		Domain:        domain,
		CustomDomains: customDomains,
		Cert:          cert,
		Key:           key,
		Email:         email,
		Env:           env,
		Host:          host,
		Port:          port,
	})

}

var ansiRegexp = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

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

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(executable, "run", app)
	cmd.Args = append(cmd.Args, body.Args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NO_COLOR=1")
	cmd.Env = append(cmd.Env, "CI=1")

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			w.Header().Set("X-Exit-Code", strconv.Itoa(exitError.ExitCode()))
			w.Header().Set("Content-Type", "text/plain")
			w.Write(ansiRegexp.ReplaceAll(stderr.Bytes(), nil))
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Exit-Code", "0")
	w.Header().Set("Content-Type", "text/plain")
	w.Write(ansiRegexp.ReplaceAll(stdout.Bytes(), nil))
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
				w.Write([]byte(err.Error()))
				return
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
				w.Write([]byte(err.Error()))
				return
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
				w.Write([]byte(err.Error()))
				return
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
