package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/pomdtr/smallweb/client"
	"github.com/spf13/cobra"
)

type AppKind int

const (
	AppKindUnknown AppKind = iota
	AppKindHTTP
	AppKindCLI
)

func listApps(kind ...AppKind) ([]string, error) {
	if len(kind) == 0 {
		kind = []AppKind{AppKindHTTP, AppKindCLI}
	}

	apps := make(map[string]struct{})
	entries, err := os.ReadDir(path.Join(client.SMALLWEB_ROOT))
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		for _, extension := range client.EXTENSIONS {
			for _, k := range kind {
				switch k {
				case AppKindHTTP:
					if exists(path.Join(client.SMALLWEB_ROOT, entry.Name(), "http"+extension)) {
						apps[entry.Name()] = struct{}{}
					}
					if exists(path.Join(client.SMALLWEB_ROOT, entry.Name(), "index.html")) {
						apps[entry.Name()] = struct{}{}
					}
				case AppKindCLI:
					if exists(path.Join(client.SMALLWEB_ROOT, "cli", extension)) {
						apps[entry.Name()] = struct{}{}
					}
				}
			}
		}
	}

	var appList []string
	for app := range apps {
		appList = append(appList, app)
	}

	return appList, nil
}

func NewCmdList() *cobra.Command {
	var flags struct {
		kind string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			apps, err := func() ([]string, error) {
				switch flags.kind {
				case "http":
					return listApps(AppKindHTTP)
				case "cli":
					return listApps(AppKindCLI)
				default:
					return listApps()
				}
			}()

			if err != nil {
				return fmt.Errorf("failed to list apps: %v", err)
			}

			for _, app := range apps {
				fmt.Println(app)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.kind, "kind", "k", "", "filter by kind")

	return cmd
}
