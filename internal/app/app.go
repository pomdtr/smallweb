package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsops/sops/v3/decrypt"
	"github.com/joho/godotenv"
	"github.com/pomdtr/smallweb/internal/utils"
	"github.com/tailscale/hujson"
)

var (
	ErrAppNotFound = errors.New("app not found")
)

type AppConfig struct {
	Entrypoint string    `json:"entrypoint,omitempty"`
	Root       string    `json:"root,omitempty"`
	AtUri      string    `json:"atUri,omitempty"`
	Crons      []CronJob `json:"crons,omitempty"`
}

type DenoConfig struct {
	Smallweb AppConfig `json:"smallweb"`
}

type CronJob struct {
	Description string   `json:"description"`
	Schedule    string   `json:"schedule"`
	Args        []string `json:"args"`
}

type App struct {
	Name   string
	Domain string
	Dir    string
	Config AppConfig
	env    map[string]string
}

func (me *App) Root() string {
	dir := me.Dir
	if fi, err := os.Lstat(dir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if root, err := os.Readlink(dir); err == nil {
			dir = filepath.Join(filepath.Dir(dir), root)
		}
	}

	if me.Config.Root != "" {
		return filepath.Join(dir, me.Config.Root)
	}

	if me.Config.Entrypoint != "" {
		return dir
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		if utils.FileExists(filepath.Join(dir, candidate)) {
			return dir
		}
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		if utils.FileExists(filepath.Join(dir, "dist", candidate)) {
			return filepath.Join(dir, "dist")
		}
	}

	if utils.FileExists(filepath.Join(dir, "dist", "index.html")) {
		return filepath.Join(dir, "dist")
	}

	return dir
}

func (me *App) DataDir() string {
	dir := filepath.Join(me.Root(), "data")
	if fi, err := os.Lstat(dir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if root, err := os.Readlink(dir); err == nil {
			dir = filepath.Join(filepath.Dir(dir), root)
		}
	}

	return dir
}

func LoadApp(baseDir string, baseDomain string, name string) (App, error) {
	appDir := filepath.Join(baseDir, baseDomain, name)
	app := App{
		Dir:  appDir,
		Name: name,
		env:  make(map[string]string),
	}

	if name == "_" {
		app.Domain = baseDomain
	} else {
		app.Domain = fmt.Sprintf("%s.%s", name, baseDomain)
	}

	if dotenvPath := filepath.Join(appDir, ".env"); utils.FileExists(dotenvPath) {
		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			app.env[key] = value
		}
	}

	for _, secretPath := range []string{
		filepath.Join(appDir, "secrets.enc.env"),
		filepath.Join(appDir, "secrets.env"),
	} {
		if !utils.FileExists(secretPath) {
			continue
		}

		rawBytes, err := os.ReadFile(secretPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read file: %v", err)
		}

		dotenvBytes, err := decrypt.Data(rawBytes, "dotenv")
		if err != nil {
			return App{}, fmt.Errorf("could not decrypt %s: %v", secretPath, err)
		}

		dotenv, err := godotenv.Parse(bytes.NewReader(dotenvBytes))
		if err != nil {
			return App{}, fmt.Errorf("could not parse %s: %v", secretPath, err)
		}

		for key, value := range dotenv {
			app.env[key] = value
		}

		break
	}

	for _, configPath := range []string{
		filepath.Join(appDir, "smallweb.json"),
		filepath.Join(appDir, "smallweb.jsonc"),
	} {
		if !utils.FileExists(configPath) {
			continue
		}

		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read %s: %v", configPath, err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize %s: %v", configPath, err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal %s: %v", configPath, err)
		}

		return app, nil
	}

	return app, nil
}

func (me App) URL() string {
	return fmt.Sprintf("https://%s/", me.Domain)
}

func (me App) Entrypoint() (string, error) {
	if strings.HasPrefix(me.Config.Entrypoint, "jsr:") || strings.HasPrefix(me.Config.Entrypoint, "npm:") {
		return me.Config.Entrypoint, nil
	}

	if strings.HasPrefix(me.Config.Entrypoint, "https://") || strings.HasPrefix(me.Config.Entrypoint, "http://") {
		return me.Config.Entrypoint, nil
	}

	if me.Config.Entrypoint != "" {
		return filepath.Join(me.Root(), me.Config.Entrypoint), nil
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		path := filepath.Join(me.Root(), candidate)
		if utils.FileExists(path) {
			return fmt.Sprintf("file://%s", path), nil
		}
	}

	return "jsr:@smallweb/file-server@0.8.4", nil
}

func (me App) Env() []string {
	env := []string{}

	for k, v := range me.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	env = append(env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
	env = append(env, "DENO_NO_UPDATE_CHECK=1")

	// open telemetry
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "OTEL_") {
			env = append(env, value)
		}
	}
	env = append(env, fmt.Sprintf("OTEL_SERVICE_NAME=%s", me.Domain))

	return env
}
