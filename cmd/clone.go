package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var SMALLWEB_ROOT string

func init() {
	SMALLWEB_ROOT = os.Getenv("SMALLWEB_ROOT")
	if SMALLWEB_ROOT == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}

		SMALLWEB_ROOT = path.Join(homedir, "www")
	}
}

func NewCmdClone() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone <url> [name]",
		Short: "Clone a smallweb app",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := url.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid URL: %v", err)
			}

			switch url.Host {
			case "val.town", "www.val.town":
				parts := strings.Split(url.Path, "/")
				if len(parts) != 4 {
					return fmt.Errorf("invalid val URL")
				}

				targetDir := filepath.Join(SMALLWEB_ROOT, parts[3])

				if exists(targetDir) {
					return fmt.Errorf("app already exists")
				}

				resp, err := http.Get(fmt.Sprintf("https://esm.town%s", url.Path))
				if err != nil {
					return fmt.Errorf("failed to fetch app: %v", err)
				}
				defer resp.Body.Close()

				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return fmt.Errorf("failed to create app directory: %v", err)
				}

				entrypoint, err := os.Create(filepath.Join(targetDir, "http.ts"))
				if err != nil {
					return fmt.Errorf("failed to create entrypoint: %v", err)
				}

				if _, err := io.Copy(entrypoint, resp.Body); err != nil {
					return fmt.Errorf("failed to write entrypoint: %v", err)
				}

				fmt.Fprintf(os.Stderr, "Cloned val to %s\n", targetDir)
				return nil
			case "github.com":
				targetDir := filepath.Join(SMALLWEB_ROOT, args[1])
				if exists(targetDir) {
					return fmt.Errorf("app %s already exists", args[1])
				}

				if err := gitClone(args[0], targetDir); err != nil {
					return fmt.Errorf("failed to clone: %v", err)
				}

				fmt.Fprintf(os.Stderr, "Cloned %s to %s\n", args[0], targetDir)
				return nil
			case "gitlab.com":
				targetDir := filepath.Join(SMALLWEB_ROOT, args[1])
				if exists(targetDir) {
					return fmt.Errorf("app %s already exists", args[1])
				}

				if err := gitClone(args[0], targetDir); err != nil {
					return fmt.Errorf("failed to clone: %v", err)
				}

				fmt.Fprintf(os.Stderr, "Cloned %s to %s\n", args[0], targetDir)
				return nil
			default:
				return fmt.Errorf("unsupported host: %s", url.Host)
			}
		},
	}

	return cmd
}

func gitClone(src, dst string) error {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found: %v", err)
	}

	if err := exec.Command(gitPath, "clone", src, dst).Run(); err != nil {
		return fmt.Errorf("failed to clone: %v", err)
	}

	return nil
}
