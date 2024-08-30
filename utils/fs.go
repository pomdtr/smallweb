package utils

import (
	"os"
	"strings"
)

func FileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func ExpandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		return strings.Replace(p, "~", os.Getenv("HOME"), 1)
	}
	return p
}
