package cmd

import (
	_ "embed"

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
			cmd.Println(string(typesBytes))
			return nil
		},
	}

	return cmd
}
