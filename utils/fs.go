package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func FileExists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

func ExpandTilde(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return strings.Replace(path, "~", home, 1), nil
	}
	return path, nil
}
