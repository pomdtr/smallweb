package utils

import (
	"os"
	"path/filepath"
)

func FileExists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}
