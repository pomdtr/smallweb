package cmd

import (
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func completeApp(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("dir")))
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, app := range apps {
		completions = append(completions, app.Name)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
