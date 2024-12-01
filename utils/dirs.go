package utils

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

var (
	RootDir    string
	PluginDirs []string
	DenoDir    string
	ConfigPath string
)

func init() {
	if env, ok := os.LookupEnv("SMALLWEB_DIR"); ok {
		RootDir = env
		return
	}

	RootDir = filepath.Join(os.Getenv("HOME"), "smallweb")
	PluginDirs = []string{filepath.Join(RootDir, ".smallweb", "plugins"), filepath.Join(xdg.DataHome, "smallweb", "plugins")}
	DenoDir = filepath.Join(xdg.CacheHome, "smallweb", "deno")
}
