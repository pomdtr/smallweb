package git

import (
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
)

func NewHandler(gitdir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set required environment variables for git-http-backend
		env := os.Environ()
		env = append(env, "GIT_PROJECT_ROOT="+gitdir)
		env = append(env, "GIT_HTTP_EXPORT_ALL=") // allow all repos to be exported via HTTP
		env = append(env, "PATH_INFO="+r.URL.Path)

		git, err := exec.LookPath("git")
		if err != nil {
			http.Error(w, "Git executable not found", http.StatusInternalServerError)
			return
		}

		// Prepare the CGI handler
		cgiHandler := &cgi.Handler{
			Path: git, // Adjust this path on your system
			Args: []string{"http-backend"},
			Dir:  gitdir,
			Env:  env,
		}

		cgiHandler.ServeHTTP(w, r)
	})
}
