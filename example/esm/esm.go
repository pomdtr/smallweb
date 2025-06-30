package esm

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func NewHandler(root string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/{app}/{filepath...}", func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		revision := plumbing.Revision("HEAD")
		fp := r.PathValue("filepath")

		parts := strings.Split(app, "@")
		if len(parts) > 1 {
			app = parts[0]
			revision = plumbing.Revision(parts[1])
		}

		repoDir := filepath.Join(root, app)
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}

		repo, err := git.PlainOpen(repoDir)
		if err != nil {
			http.Error(w, "Failed to open repository: "+err.Error(), http.StatusInternalServerError)
			return
		}

		hash, err := repo.ResolveRevision(revision)
		if err != nil {
			if err == plumbing.ErrReferenceNotFound {
				http.Error(w, fmt.Sprintf("Revision not found: %s", revision), http.StatusNotFound)
				return
			}

			http.Error(w, "Failed to resolve reference: "+err.Error(), http.StatusInternalServerError)
			return
		}

		commit, err := repo.CommitObject(*hash)
		if err != nil {
			http.Error(w, "Failed to get commit object: "+err.Error(), http.StatusInternalServerError)
			return
		}

		tree, err := commit.Tree()
		if err != nil {
			http.Error(w, "Failed to get commit tree: "+err.Error(), http.StatusInternalServerError)
			return
		}

		f, err := tree.File(fp)
		if err != nil {
			http.Error(w, "File not found: "+err.Error(), http.StatusNotFound)
			return
		}

		content, err := f.Contents()
		if err != nil {
			http.Error(w, "Failed to read file content: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(content)); err != nil {
			http.Error(w, "Failed to write response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return mux
}
