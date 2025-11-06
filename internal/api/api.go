package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/hostrouter"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/utils"
)

type GetBlobsOutput struct {
	Body []string
}

type GetBlobOutput struct {
	ContentType string `header:"Content-Type"`

	Body []byte
}

type GetAppOutput struct {
	Body struct {
	}
}

type GetAppsOutput struct {
	Body struct {
		Apps []string `json:"apps"`
	}
}

func NewHandler(conf *utils.Config) http.Handler {
	// Create a new router & API.
	r := chi.NewRouter()
	hr := hostrouter.New()

	hr.Map("api.localhost", apiRouter(conf))
	hr.Map("git.localhost", gitRouter(conf))
	hr.Map("raw.localhost", rawRouter(conf))
	hr.Map("*", catchAllRouter())

	r.Mount("/", hr)

	return r
}

func apiRouter(conf *utils.Config) chi.Router {
	r := chi.NewRouter()

	api := humachi.New(r, huma.DefaultConfig("Smallweb API", "v1.0.0"))

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps",
	}, func(ctx context.Context, i *struct {
	}) (*GetAppsOutput, error) {
		appList, err := app.List(conf.String("dir"))
		if err != nil {
			return nil, err
		}

		resp := &GetAppsOutput{}
		resp.Body.Apps = appList

		return resp, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps",
	}, func(ctx context.Context, i *struct{}) (*struct{}, error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps/{app}",
	}, func(ctx context.Context, i *struct {
	}) (*GetAppOutput, error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPut,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps/{app}",
	}, func(ctx context.Context, i *struct {
		App string `path:"app" doc:"The app name"`
	}) (*struct{}, error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodDelete,
		Path:   "/v1/apps/{app}",
		Tags:   []string{"Apps"},
	}, func(ctx context.Context, i *struct {
	}) (*struct{}, error) {
		return nil, nil
	})

	return r
}

func gitRouter(conf *utils.Config) chi.Router {
	r := chi.NewRouter()

	repoRoot := filepath.Join(conf.String("dir"))

	gitPath, err := exec.LookPath("git")
	if err != nil {
		r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "git binary not found on server", http.StatusInternalServerError)
		})
		return r
	}

	gitHandler := &cgi.Handler{
		Path: gitPath,
		Args: []string{"http-backend"},
		Env: []string{
			"GIT_PROJECT_ROOT=" + repoRoot,
			"GIT_HTTP_EXPORT_ALL=", // allow read-only access
		},
		Stderr: os.Stderr,
		InheritEnv: []string{
			"PATH", "HOME", // inherit some safe env vars
		},
	}

	r.HandleFunc("/{app}/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/git-receive-pack") {
			http.Error(w, "pushes are disabled", http.StatusForbidden)
			return
		}

		fixChunked(r)

		gitHandler.ServeHTTP(w, r)
	})

	return r
}

func catchAllRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("unknown host: " + r.Host))
	})

	return r
}

func fixChunked(req *http.Request) {
	if len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == `chunked` {
		// hacking!
		req.TransferEncoding = nil
		req.Header.Set(`Transfer-Encoding`, `chunked`)

		// let cgi use Body.
		req.ContentLength = -1
	}
}

func serveRawFile(w http.ResponseWriter, conf *utils.Config, appname, ref, filePath string) {
	// Check app privacy
	if conf.String(fmt.Sprintf("apps.%s.privacy", appname)) != "public" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Open the git repository
	repoPath := filepath.Join(conf.String("dir"), appname)
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Resolve the reference (branch, tag, or commit hash)
	var hash plumbing.Hash

	if ref == "" {
		// Use HEAD if no ref is specified
		headRef, err := repo.Head()
		if err != nil {
			http.Error(w, "Failed to resolve HEAD", http.StatusInternalServerError)
			return
		}
		hash = headRef.Hash()
	} else {
		// Try as a branch first
		branchRef, err := repo.Reference(plumbing.NewBranchReferenceName(ref), true)
		if err == nil {
			hash = branchRef.Hash()
		} else {
			// Try as a tag
			tagRef, err := repo.Reference(plumbing.NewTagReferenceName(ref), true)
			if err == nil {
				hash = tagRef.Hash()
			} else {
				// Try as a commit hash
				h := plumbing.NewHash(ref)
				_, err := repo.CommitObject(h)
				if err == nil {
					hash = h
				} else {
					http.Error(w, "Reference not found", http.StatusNotFound)
					return
				}
			}
		}
	}

	// Get the commit
	commit, err := repo.CommitObject(hash)
	if err != nil {
		http.Error(w, "Commit not found", http.StatusNotFound)
		return
	}

	// Get the tree
	tree, err := commit.Tree()
	if err != nil {
		http.Error(w, "Failed to read tree", http.StatusInternalServerError)
		return
	}

	// Get the file from the tree
	file, err := tree.File(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Read file contents
	contents, err := file.Contents()
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Set headers and write response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(contents))
}

func rawRouter(conf *utils.Config) chi.Router {
	r := chi.NewRouter()

	// Handle pattern with ref: /{app}@{ref}/{filepath...}
	r.Get("/{app}@{ref}/*", func(w http.ResponseWriter, r *http.Request) {
		appname := chi.URLParam(r, "app")
		ref := chi.URLParam(r, "ref")
		filePath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")

		if filePath == "" {
			http.Error(w, "File path is required", http.StatusBadRequest)
			return
		}

		serveRawFile(w, conf, appname, ref, filePath)
	})

	// Handle pattern without ref (redirects to default branch): /{app}/{filepath...}
	r.Get("/{app}/*", func(w http.ResponseWriter, r *http.Request) {
		appname := chi.URLParam(r, "app")
		filePath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")

		if filePath == "" {
			http.Error(w, "File path is required", http.StatusBadRequest)
			return
		}

		// Check app privacy
		if conf.String(fmt.Sprintf("apps.%s.privacy", appname)) != "public" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Open the git repository
		repoPath := filepath.Join(conf.String("dir"), appname)
		repo, err := git.PlainOpen(repoPath)
		if err != nil {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}

		// Get HEAD reference
		headRef, err := repo.Head()
		if err != nil {
			http.Error(w, "Failed to resolve HEAD", http.StatusInternalServerError)
			return
		}

		// Get the branch name or use the commit hash
		refName := headRef.Name().Short()

		// Redirect to the URL with the explicit ref
		redirectURL := fmt.Sprintf("/%s@%s/%s", appname, refName, filePath)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	return r
}
