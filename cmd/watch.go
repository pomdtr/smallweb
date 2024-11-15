package cmd

import (
	"fmt"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/watcher"
	"github.com/spf13/cobra"
)

func NewCmdWatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "watch",
		Hidden: true,
		Short:  "Watch for changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			watcher, err := watcher.NewWatcher(utils.RootDir())
			if err != nil {
				return fmt.Errorf("failed to create watcher: %v", err)
			}

			return watcher.Start()
		},
	}

	return cmd

}
