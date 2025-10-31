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
	Entrypoint        string    `json:"entrypoint,omitempty"`
	Root              string    `json:"root,omitempty"`
	Crons             []CronJob `json:"crons,omitempty"`
	AuthorizedKeys    []string  `json:"authorizedKeys,omitempty"`
	AdditionalDomains []string  `json:"additionalDomains,omitempty"`
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
	Name   string            `json:"name"`
	Dir    string            `json:"dir,omitempty"`
	Config AppConfig         `json:"-"`
	env    map[string]string `json:"-"`
}

func (me *App) Root() string {
	root := me.Dir
	if fi, err := os.Lstat(root); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(root); err == nil {
			root = filepath.Join(filepath.Dir(root), target)
		}
	}

	if me.Config.Root != "" {
		return filepath.Join(root, me.Config.Root)
	}

	if me.Config.Entrypoint != "" {
		return root
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		if utils.FileExists(filepath.Join(root, candidate)) {
			return root
		}
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		if utils.FileExists(filepath.Join(root, "dist", candidate)) {
			return filepath.Join(root, "dist")
		}
	}

	if utils.FileExists(filepath.Join(root, "dist", "index.html")) {
		return filepath.Join(root, "dist")
	}

	return root
}

func (me *App) DataDir() string {
	dataDir := filepath.Join(me.Root(), "data")
	if fi, err := os.Lstat(dataDir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if root, err := os.Readlink(dataDir); err == nil {
			dataDir = filepath.Join(filepath.Dir(dataDir), root)
		}
	}

	return dataDir
}

func ListApps(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("could not read directory %s: %v", rootDir, err)
	}

	apps := make([]string, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if !entry.IsDir() {
			continue
		}

		apps = append(apps, entry.Name())
	}

	return apps, nil
}

func LoadApp(appDir string) (App, error) {
	app := App{
		Name: filepath.Base(appDir),
		Dir:  appDir,
		env:  make(map[string]string),
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

	for _, configName := range []string{"smallweb.json", "smallweb.jsonc"} {
		configPath := filepath.Join(appDir, configName)
		if !utils.FileExists(configPath) {
			continue
		}

		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read %s: %v", configName, err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize %s: %v", configName, err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal %s: %v", configName, err)
		}

		return app, nil
	}

	return app, nil
}

func (me App) Entrypoint() string {
	if strings.HasPrefix(me.Config.Entrypoint, "jsr:") || strings.HasPrefix(me.Config.Entrypoint, "npm:") {
		return me.Config.Entrypoint
	}

	if strings.HasPrefix(me.Config.Entrypoint, "https://") || strings.HasPrefix(me.Config.Entrypoint, "http://") {
		return me.Config.Entrypoint
	}

	if me.Config.Entrypoint != "" {
		return filepath.Join(me.Root(), me.Config.Entrypoint)
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		path := filepath.Join(me.Root(), candidate)
		if utils.FileExists(path) {
			return fmt.Sprintf("file://%s", path)
		}
	}

	return "jsr:@smallweb/file-server@0.8.2"
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
	env = append(env, fmt.Sprintf("OTEL_SERVICE_NAME=%s", me.Name))

	return env
}
