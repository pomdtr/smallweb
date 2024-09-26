package docs

import (
	"embed"
	_ "embed"
	"io/fs"
	"net/http"
)

//go:generate mdbook build

//go:embed book
var embedFS embed.FS

var Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	subFS, err := fs.Sub(embedFS, "book")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	server := http.FileServer(http.FS(subFS))
	server.ServeHTTP(w, r)
})
