package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pomdtr/smallweb/internal/utils"
	"github.com/spf13/cobra"
	"github.com/tailscale/hujson"
)

type JsonPatchOperation struct {
	Op    string      `json:"op"`
	From  string      `json:"from,omitempty"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type JsonPath []JsonPatchOperation

func PatchFile(fp string, patch JsonPath) error {
	b, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", fp, err)
	}

	parsed, err := hujson.Parse(b)
	if err != nil {
		return fmt.Errorf("parsing HuJSON file %s: %w", fp, err)
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshaling JSON patch for file %s: %w", fp, err)
	}

	if err := parsed.Patch(patchBytes); err != nil {
		return fmt.Errorf("applying JSON patch to file %s: %w", fp, err)
	}

	parsed.Format()
	packed := parsed.Pack()

	if err := os.WriteFile(fp, packed, 0o644); err != nil {
		return fmt.Errorf("writing patched HuJSON file %s: %w", fp, err)
	}

	return nil
}

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete <path>",
		Aliases:           []string{"del", "rm"},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp,
		Short:             "Delete a file or directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(k.String("dir"), args[0])
			if _, err := os.Stat(appDir); os.IsNotExist(err) {
				cmd.PrintErrf("path %q does not exist\n", args[0])
				return ExitError{1}
			}

			if err := os.RemoveAll(appDir); err != nil {
				return fmt.Errorf("failed to delete %q: %w", args[0], err)
			}

			if !slices.Contains(k.MapKeys("apps"), args[0]) {
				fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s\n", args[0])
				return nil
			}

			jsonPath := fmt.Sprintf("/%s/%s", "apps", args[0])
			patch := JsonPath{
				{
					Op:   "remove",
					Path: jsonPath,
				},
			}

			configPath := utils.FindConfigPath(k.String("dir"))
			if err := PatchFile(configPath, patch); err != nil {
				return fmt.Errorf("updating config file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Deleted", args[0])
			return nil
		},
	}

	return cmd
}
