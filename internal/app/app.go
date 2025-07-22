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
	Entrypoint    string    `json:"entrypoint,omitempty"`
	Root          string    `json:"root,omitempty"`
	Crons         []CronJob `json:"crons,omitempty"`
	Private       bool      `json:"private"`
	PrivateRoutes []string  `json:"privateRoutes"`
	PublicRoutes  []string  `json:"publicRoutes"`
}

type CronJob struct {
	Description string   `json:"description"`
	Schedule    string   `json:"schedule"`
	Args        []string `json:"args"`
}

type App struct {
	Name       string            `json:"name"`
	RootDir    string            `json:"-"`
	RootDomain string            `json:"-"`
	BaseDir    string            `json:"dir,omitempty"`
	Domain     string            `json:"-"`
	Env        map[string]string `json:"domain,omitempty"`
	Config     AppConfig         `json:"-"`
}

func (me *App) Dir() string {
	dir := me.BaseDir
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
	dir := filepath.Join(me.Dir(), "data")
	if fi, err := os.Lstat(dir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if root, err := os.Readlink(dir); err == nil {
			dir = filepath.Join(filepath.Dir(dir), root)
		}
	}

	return dir
}

func LookupApps(rootDir string) ([]string, error) {
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

func LoadApp(appname string, rootDir string, domain string) (App, error) {
	appDir := filepath.Join(rootDir, appname)
	if !utils.FileExists(filepath.Join(rootDir, appname)) {
		return App{}, ErrAppNotFound
	}

	app := App{
		Name:       appname,
		RootDir:    rootDir,
		RootDomain: domain,
		BaseDir:    filepath.Join(rootDir, appname),
		Domain:     fmt.Sprintf("%s.%s", appname, domain),
		Env:        make(map[string]string),
	}

	if dotenvPath := filepath.Join(appDir, ".env"); utils.FileExists(dotenvPath) {
		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			app.Env[key] = value
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
			app.Env[key] = value
		}

		break
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
		return filepath.Join(me.Dir(), me.Config.Entrypoint)
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		path := filepath.Join(me.Dir(), candidate)
		if utils.FileExists(path) {
			return fmt.Sprintf("file://%s", path)
		}
	}

	return "jsr:@smallweb/file-server@0.8.2"
}
