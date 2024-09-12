package docs

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate mdbook build

//go:embed book
var embedFS embed.FS

type Handler struct{}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	subFS, err := fs.Sub(embedFS, "book")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.FileServer(http.FS(subFS)).ServeHTTP(w, r)
}
