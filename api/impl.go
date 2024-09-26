package api

import (
	"bytes"
	"embed"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/knadh/koanf/v2"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"golang.org/x/net/webdav"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml openapi.json

//go:embed openapi.json
var specs []byte

//go:generate npm install

//go:embed node_modules/swagger-ui-dist
var swaggerUiDist embed.FS

//go:embed index.html
var swaggerHomepage []byte

var doc = MustLoadSpecs(specs)

func MustLoadSpecs(data []byte) *openapi3.T {
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData(data)
	if err != nil {
		panic(err)
	}
	return spec
}

type Server struct {
	k *koanf.Koanf
}

func NewHandler(k *koanf.Koanf) http.Handler {
	server := &Server{k: k}
	handler := Handler(server)
	webdavHandler := webdav.Handler{
		FileSystem: webdav.Dir(utils.ExpandTilde(k.String("dir"))),
		LockSystem: webdav.NewMemLS(),
		Prefix:     "/v0/webdav",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v0/webdav") {
			webdavHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/v0") {
			handler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "text/yaml")
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			encoder.Encode(doc)
			return
		}

		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write(swaggerHomepage)
			return
		}

		subfs, err := fs.Sub(swaggerUiDist, "node_modules/swagger-ui-dist")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.FileServer(http.FS(subfs)).ServeHTTP(w, r)
	})
}

func (me *Server) GetV0Apps(w http.ResponseWriter, r *http.Request) {
	rootDir := utils.ExpandTilde(me.k.String("dir"))
	names, err := app.ListApps(rootDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apps []App
	for _, name := range names {
		a, err := app.LoadApp(name, me.k.String("domain"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		apps = append(apps, App{
			Name: a.Name,
			Url:  a.Url,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (me *Server) GetV0Config(w http.ResponseWriter, r *http.Request) {
	b, err := me.k.Marshal(utils.ConfigParser())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

var ansiRegexp = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func (me *Server) PostV0RunApp(w http.ResponseWriter, r *http.Request, app string) {
	executable, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var body PostV0RunAppJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(executable, "run", app)
	cmd.Args = append(cmd.Args, body.Args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NO_COLOR=1")
	cmd.Env = append(cmd.Env, "CI=1")

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			w.Header().Set("X-Exit-Code", strconv.Itoa(exitError.ExitCode()))
			w.Header().Set("Content-Type", "text/plain")
			w.Write(ansiRegexp.ReplaceAll(stderr.Bytes(), nil))
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Exit-Code", "0")
	w.Header().Set("Content-Type", "text/plain")
	w.Write(ansiRegexp.ReplaceAll(stdout.Bytes(), nil))
}
