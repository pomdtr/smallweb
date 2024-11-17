package cmd

import (
	"fmt"
	"os"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [key]",
		Short: "Get config values",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				k.Set("dir", utils.RootDir())
				output, err := k.Marshal(&utils.HuJSON{})
				if err != nil {
					return fmt.Errorf("could not marshal config: %w", err)
				}

				os.Stdout.Write(output)
				return nil
			}

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
