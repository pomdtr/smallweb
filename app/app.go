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
	"github.com/pomdtr/smallweb/utils"
	"github.com/tailscale/hujson"
)

var (
	ErrAppNotFound = errors.New("app not found")
)

type AppConfig struct {
	Entrypoint string    `json:"entrypoint,omitempty"`
	Root       string    `json:"root,omitempty"`
	Crons      []CronJob `json:"crons,omitempty"`
}

type CronJob struct {
	Schedule string   `json:"schedule"`
	Args     []string `json:"args"`
}

type App struct {
	Admin  bool              `json:"admin"`
	Name   string            `json:"name"`
	Dir    string            `json:"dir,omitempty"`
	Domain string            `json:"-"`
	URL    string            `json:"url"`
	Env    map[string]string `json:"-"`
	Config AppConfig         `json:"-"`
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
		path := filepath.Join(dir, candidate)
		if utils.FileExists(path) {
			return dir
		}
	}

	if utils.FileExists(filepath.Join(dir, "dist", "index.html")) {
		return filepath.Join(dir, "dist")
	}

	return dir
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

func NewApp(appname string, rootDir string, domain string, isAdmin bool) (App, error) {
	appDir := filepath.Join(rootDir, appname)
	if !utils.FileExists(filepath.Join(rootDir, appname)) {
		return App{}, ErrAppNotFound
	}

	app := App{
		Name:   appname,
		Admin:  isAdmin,
		Dir:    filepath.Join(rootDir, appname),
		Domain: fmt.Sprintf("%s.%s", appname, domain),
		URL:    fmt.Sprintf("https://%s.%s/", appname, domain),
		Env:    make(map[string]string),
	}

	for _, dotenvPath := range []string{
		filepath.Join(rootDir, ".env"),
		filepath.Join(appDir, ".env"),
	} {
		if !utils.FileExists(dotenvPath) {
			continue
		}

		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			app.Env[key] = value
		}
	}

	for _, secretPath := range []string{
		filepath.Join(rootDir, ".smallweb", "secrets.enc.env"),
		filepath.Join(appDir, "secrets.enc.env"),
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
			app.Env[key] = value
		}
	}

	if configPath := filepath.Join(appDir, "smallweb.json"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize smallweb.jsonc: %v", err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal smallweb.json: %v", err)
		}

		return app, nil
	}

	if configPath := filepath.Join(appDir, "smallweb.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize smallweb.jsonc: %v", err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal smallweb.json: %v", err)
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
			return path
		}
	}

	return "jsr:@smallweb/file-server@0.6.0"
}
