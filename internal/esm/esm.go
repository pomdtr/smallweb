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

func NewHandler(gitdir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{app}/{pathname...}", func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		pathname := r.PathValue("pathname")
		revision := plumbing.Revision("HEAD")

		parts := strings.Split(app, "@")
		if len(parts) > 1 {
			app = parts[0]
			revision = plumbing.Revision(parts[1])
		}

		repoDir := filepath.Join(gitdir, app)
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

		// If revision is not a short hash, redirect to the short hash URL
		shortHash := hash.String()[:7]
		if len(parts) == 1 || parts[1] != shortHash {
			// reconstruct the URL with the short hash
			http.Redirect(w, r, fmt.Sprintf("/%s@%s/%s", app, shortHash, pathname), http.StatusFound)
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

		f, err := tree.File(pathname)
		if err != nil {
			http.Error(w, "File not found: "+err.Error(), http.StatusNotFound)
			return
		}

		code, err := f.Contents()
		if err != nil {
			http.Error(w, "Failed to read file content: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(code))
	})

	return mux
}
