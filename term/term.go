package term

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func StripAnsi(b []byte) []byte {
	return re.ReplaceAll(b, nil)
}

type Handler struct {
	Name string
	Args []string
	Dir  string
	Env  []string
}

type ResizePayload struct {
	ID   string `json:"id"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func NewHandler(name string, args ...string) *Handler {
	return &Handler{
		Name: name,
		Args: args,
	}
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command(me.Name, me.Args...)
	cmd.Args = append(cmd.Args, extractArgs(r.URL)...)
	cmd.Stdin = r.Body
	cmd.Dir = me.Dir

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, me.Env...)
	cmd.Env = append(cmd.Env, "NO_COLOR=1")
	cmd.Env = append(cmd.Env, "CI=1")

	if r.Method == http.MethodPost {
		output, err := cmd.Output()
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(StripAnsi(exitError.Stderr))
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(StripAnsi(output))
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Stream output
	done := make(chan bool)
	go func() {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// Combine stdout and stderr
		multiReader := io.MultiReader(stdout, stderr)

		buf := make([]byte, 1024)
		for {
			n, err := multiReader.Read(buf)
			if n > 0 {
				data := StripAnsi(buf[:n])
				_, err := w.Write(data)
				if err != nil {
					break
				}
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
		done <- true
	}()

	// Wait for the command to finish
	cmd.Wait()
	<-done
}

func extractArgs(url *url.URL) []string {
	var args []string
	if url.Path != "/" {
		args = strings.Split(url.Path[1:], "/")
	}
	for key, values := range url.Query() {
		if key == "_payload" {
			continue
		}

		value := values[0]

		if len(key) == 1 {
			if value == "" {
				args = append(args, fmt.Sprintf("-%s", key))
			} else {
				args = append(args, fmt.Sprintf("-%s=%s", key, value))
			}
		} else {
			if value == "" {
				args = append(args, fmt.Sprintf("--%s", key))
			} else {
				args = append(args, fmt.Sprintf("--%s=%s", key, value))
			}
		}
	}

	return args
}
