package editor

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	dir string
}

//go:embed index.html
var indexTemplateRaw string
var indexTemplate = template.Must(template.New("index").Parse(indexTemplateRaw))

func NewHandler(dir string) http.Handler {
	return &Handler{
		dir: dir,
	}
}

func getExtension(basename string) string {
	parts := strings.Split(basename, ".")
	return parts[len(parts)-1]
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	parts := strings.Split(r.URL.Path[1:], "/")
	if len(parts) < 2 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	path := filepath.Join(h.dir, r.URL.Path)
	if r.Method == http.MethodPost {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		if _, err := io.Copy(f, r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("ok"))
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	appname := parts[0]
	basename := parts[len(parts)-1]

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTemplate.Execute(w, map[string]interface{}{
		"Title":    fmt.Sprintf("%s - %s", basename, appname),
		"Language": getExtension(basename),
		"Code":     string(content),
	})
}
