package editor

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"golang.org/x/net/webdav"
)

//go:generate deno task build

//go:embed dist/*
var embedFS embed.FS

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

	h.fileServer.ServeHTTP(w, r)
}
