package esm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/tailscale/hujson"
)

func NewHandler(gitdir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Usage: /{app}@{ref}/{filepath}"))
	})

	mux.HandleFunc("GET /{app}", func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		revision := plumbing.Revision("HEAD")

		if parts := strings.Split(app, "@"); len(parts) > 1 {
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

		config, err := getConfig(tree)
		if err != nil {
			http.Error(w, "Failed to get import map: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(config.Exports) == 0 {
			http.Error(w, "No exports defined in deno.json", http.StatusNotFound)
			return
		}

		defaultExport, ok := config.Exports["."]
		if !ok {
			http.Error(w, "No default export defined in deno.json", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, path.Join("/", fmt.Sprintf("%s@%s", app, hash.String()[:7]), defaultExport), http.StatusFound)
	})

	mux.HandleFunc("GET /{app}/{pathname...}", func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		pathname := r.PathValue("pathname")
		revision := plumbing.Revision("HEAD")

		if parts := strings.Split(app, "@"); len(parts) > 1 {
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

		config, err := getConfig(tree)
		if err != nil {
			http.Error(w, "Failed to get import map: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(config.Exports) > 0 {
			exportPath, ok := config.Exports["."+pathname]
			if ok {
				http.Redirect(w, r, path.Join("/", fmt.Sprintf("%s@%s", app, shortHash), exportPath), http.StatusFound)
				return
			}
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
		code = rewriteImports(code, config.Imports)

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(code))
	})

	return mux
}

type DenoConfig struct {
	Imports map[string]string `json:"imports"`
	Exports DenoConfigExports `json:"exports"`
}

type DenoConfigExports map[string]string

func (e *DenoConfigExports) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		// If it's a string, treat it as a single export
		*e = DenoConfigExports{".": raw}
		return nil
	}

	var rawMap map[string]string
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return fmt.Errorf("failed to unmarshal exports: %w", err)
	}

	*e = DenoConfigExports(rawMap)
	return nil
}

func getConfig(tree *object.Tree) (*DenoConfig, error) {
	for _, entry := range []string{"deno.jsonc", "deno.json"} {
		f, err := tree.File(entry)
		if err != nil {
			continue
		}

		contents, err := f.Contents()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", entry, err)
		}

		jsonBytes, err := hujson.Standardize([]byte(contents))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", entry, err)
		}

		var config DenoConfig
		if err := json.Unmarshal(jsonBytes, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", entry, err)
		}

		return &config, nil

	}

	return &DenoConfig{}, nil // No config found, return empty config
}

var importRewriteRegexp = regexp.MustCompile(`(?m)(?:import\s+(?:[^'"]+from\s+)?|export\s+[^'"]+from\s+|import\s*\()\s*(['"])([^'"]+)['"]`)

func rewriteImports(src string, importMap map[string]string) string {
	if len(importMap) == 0 {
		return src // No imports to rewrite
	}

	return importRewriteRegexp.ReplaceAllStringFunc(src, func(match string) string {
		submatches := importRewriteRegexp.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		quote := submatches[1]
		specifier := submatches[2]

		base := specifier
		subpath := ""
		if i := strings.Index(specifier, "/"); i != -1 {
			base = specifier[:i]
			subpath = specifier[i:]
		}

		mappedBase, ok := importMap[base]
		if !ok {
			return match
		}

		newSpecifier := mappedBase + subpath

		oldQuoted := quote + specifier + quote
		newQuoted := quote + newSpecifier + quote
		return strings.Replace(match, oldQuoted, newQuoted, 1)
	})
}
