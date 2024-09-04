package cmd

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"
)

//go:embed embed/types.d.ts
var typesBytes []byte

func NewCmdTypes() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "types",
		Short:   "Print smallweb types",
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stdout.Write(typesBytes); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
