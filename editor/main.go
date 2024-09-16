package editor

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"text/template"

	"golang.org/x/net/webdav"
)

//go:generate deno task build

//go:embed dist/*
var embedFS embed.FS

//go:embed index.html.tmpl
var homepageBytes []byte
var homepageTemplate = template.Must(template.New("index.html").Parse(string(homepageBytes)))

type Handler struct {
	fileServer   http.Handler
	webdavServer http.Handler
}

func NewHandler(rootDir string) (*Handler, error) {
	subFS, err := fs.Sub(embedFS, "dist")
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(subFS))
	webdavServer := webdav.Handler{
		FileSystem: webdav.Dir(rootDir),
		LockSystem: webdav.NewMemLS(),
		Prefix:     "/webdav",
	}

	return &Handler{
		fileServer:   fileServer,
		webdavServer: &webdavServer,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/webdav") {
		h.webdavServer.ServeHTTP(w, r)
		return
	}

	if r.URL.Path == "/" {
		rootPath := "/"
		if appname := r.URL.Query().Get("app"); appname != "" {
			rootPath = "/" + appname + "/"
		}

		config := getProductConfig(rootPath)
		configJson, err := json.Marshal(config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		homepageTemplate.Execute(w, map[string]interface{}{
			"ProductConfig": string(configJson),
		})

		return
	}

	h.fileServer.ServeHTTP(w, r)
}
