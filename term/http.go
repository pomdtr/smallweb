package term

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
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

var Handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	executable, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	args := strings.Split(r.URL.Path[1:], "/")
	for key, values := range r.URL.Query() {
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

	cmd := exec.Command(executable, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NO_COLOR=1")
	cmd.Env = append(cmd.Env, "CI=1")

	if r.Method == http.MethodPost {
		cmd.Stdin = r.Body
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		w.Write(StripAnsi(scanner.Bytes()))
		w.Write([]byte("\n"))
		w.(http.Flusher).Flush()
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("Command finished with error: %v", err)
	}
})
