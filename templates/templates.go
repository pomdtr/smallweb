package templates

import (
	"embed"
	"fmt"
	"path"

	"github.com/leaanthony/debme"
	"github.com/leaanthony/gosod"
)

//go:embed all:templates
var templateFS embed.FS

func List() ([]string, error) {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, err
	}
	var templates []string
	for _, entry := range entries {
		templates = append(templates, entry.Name())
	}

	return templates, nil
}

func Install(template string, dir string) error {
	root, err := debme.FS(templateFS, path.Join("templates", template))
	if err != nil {
		return fmt.Errorf("template %s not found", template)
	}

	return gosod.New(root).Extract(dir, nil)
}
