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
	RootDir = os.Getenv("SMALLWEB_DIR")
	if RootDir == "" {
		RootDir = filepath.Join(os.Getenv("HOME"), "smallweb")
	}
	PluginDirs = []string{filepath.Join(RootDir, ".smallweb", "plugins"), filepath.Join(xdg.DataHome, "smallweb", "plugins")}
	DenoDir = filepath.Join(xdg.CacheHome, "smallweb", "deno")
}
