package cmd

import (
	"fmt"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "config [key]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "dir" {
				fmt.Println(utils.RootDir())
				return nil
			}

			v := k.Get(args[0])
			if v == nil {
				return fmt.Errorf("key %q not found", args[0])
			}

			fmt.Println(v)
			return nil
		},
	}

	return cmd
}
