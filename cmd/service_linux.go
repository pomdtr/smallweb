//go:build linux

package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"text/template"
)

//go:embed embed/smallweb.service
var serviceConfigBytes []byte
var serviceConfig = template.Must(template.New("service").Parse(string(serviceConfigBytes)))

func GetService() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}

	writer := &strings.Builder{}

	if err := serviceConfig.Execute(writer, map[string]any{
		"ExecPath":    execPath,
		"SmallwebDir": k.String("dir"),
	}); err != nil {
		return "", fmt.Errorf("failed to write service file: %v", err)
	}

	return writer.String(), nil
}
