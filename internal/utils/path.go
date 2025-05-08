package utils

import (
	"os"
	"path/filepath"
)

func FindConfigPath(rootDir string) string {
	for _, candidate := range []string{".smallweb/config.jsonc", ".smallweb/config.json"} {
		configPath := filepath.Join(rootDir, candidate)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return filepath.Join(rootDir, ".smallweb/config.json")
}
