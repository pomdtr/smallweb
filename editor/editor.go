package editor

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"golang.org/x/net/webdav"
)

//go:generate deno task build

//go:embed dist
var dist embed.FS

type Handler struct {
	webdavServer http.Handler
	fileServer   http.Handler
}

func NewHandler(rootDir string) (*Handler, error) {
	subFS, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, err
	}

	return &Handler{
		webdavServer: &webdav.Handler{
			Prefix:     "/webdav",
			FileSystem: webdav.Dir(rootDir),
			LockSystem: webdav.NewMemLS(),
		},
		fileServer: http.FileServer(http.FS(subFS)),
	}, nil
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/webdav") {
		me.webdavServer.ServeHTTP(w, r)
		return
	}

	me.fileServer.ServeHTTP(w, r)
}
