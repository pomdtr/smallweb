package esm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type DenoConfig struct {
	Imports map[string]string `json:"imports"`
}

// resolveImports replaces bare import specifiers and package subpaths in TypeScript/JavaScript code
// using the provided import map. It rewrites module specifiers in import/export statements.
func resolveImports(code string, imports map[string]string) string {
	for key, value := range imports {
		// bare import: from "key"
		code = strings.ReplaceAll(code, fmt.Sprintf(`from "%s"`, key), fmt.Sprintf(`from "%s"`, value))
		code = strings.ReplaceAll(code, fmt.Sprintf(`from '%s'`, key), fmt.Sprintf(`from '%s'`, value))
		code = strings.ReplaceAll(code, fmt.Sprintf(`import("%s")`, key), fmt.Sprintf(`import("%s")`, value))
		code = strings.ReplaceAll(code, fmt.Sprintf(`import('%s')`, key), fmt.Sprintf(`import('%s')`, value))

		// subpath import: from "key/..."
		code = strings.ReplaceAll(code, fmt.Sprintf(`from "%s/`, key), fmt.Sprintf(`from "%s/`, strings.TrimSuffix(value, "/")))
		code = strings.ReplaceAll(code, fmt.Sprintf(`from '%s/`, key), fmt.Sprintf(`from '%s/`, strings.TrimSuffix(value, "/")))
		code = strings.ReplaceAll(code, fmt.Sprintf(`import("%s/`, key), fmt.Sprintf(`import("%s/`, strings.TrimSuffix(value, "/")))
		code = strings.ReplaceAll(code, fmt.Sprintf(`import('%s/`, key), fmt.Sprintf(`import('%s/`, strings.TrimSuffix(value, "/")))
	}
	return code
}

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

		// If revision is not a short hash, redirect to the short hash URL
		shortHash := hash.String()[:7]
		if len(parts) == 1 || parts[1] != shortHash {
			// reconstruct the URL with the short hash
			http.Redirect(w, r, fmt.Sprintf("/%s@%s/%s", app, shortHash, fp), http.StatusFound)
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

		contents, err := f.Contents()
		if err != nil {
			http.Error(w, "Failed to read file content: "+err.Error(), http.StatusInternalServerError)
			return
		}

		configFile, err := tree.File("deno.json")
		if err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(contents))
			return

		}

		config, err := configFile.Contents()
		if err != nil {
			http.Error(w, "Failed to read Deno config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var denoConfig DenoConfig
		if err := json.Unmarshal([]byte(config), &denoConfig); err != nil {
			http.Error(w, "Failed to parse Deno config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		contents = resolveImports(contents, denoConfig.Imports)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(contents))
	})

	return mux
}
