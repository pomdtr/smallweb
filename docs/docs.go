package docs

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate mdbook build

//go:embed book
var embedFS embed.FS

type Handler struct {
	staticFS http.FileSystem
}

func NewHandler() (*Handler, error) {
	subFS, err := fs.Sub(embedFS, "book")
	if err != nil {
		return nil, err
	}

	return &Handler{staticFS: http.FS(subFS)}, nil
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.FileServer(me.staticFS).ServeHTTP(w, r)
}
