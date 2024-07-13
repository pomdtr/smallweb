package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/pomdtr/smallweb/templates"
	"github.com/spf13/cobra"
)

type Inputs struct {
	Name     string
	Template string
}

func (me *Inputs) Fill() error {
	var fields []huh.Field

	if me.Name == "" {
		fields = append(fields, huh.NewInput().Title("Choose a name").Value(&me.Name))
	}

	if me.Template == "" {
		templates, err := templates.List()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		var options []huh.Option[string]
		for _, template := range templates {
			options = append(options, huh.NewOption(template, template))
		}

		fields = append(fields, huh.NewSelect[string]().Title("Choose a template").Options(options...).Value(&me.Template))
	}

	if len(fields) == 0 {
		return nil
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	return form.Run()
}

func NewCmdCreate() *cobra.Command {
	var flags struct {
		name     string
		template string
	}

	cmd := &cobra.Command{
		Use:     "create <app>",
		Short:   "Create a new smallweb app",
		GroupID: CoreGroupID,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputs := Inputs{
				Name:     flags.name,
				Template: flags.template,
			}

			if err := inputs.Fill(); err != nil {
				return fmt.Errorf("failed to fill inputs: %w", err)
			}

			return templates.Install(inputs.Template, inputs.Name)
		},
	}

	cmd.Flags().StringVarP(&flags.name, "name", "n", "", "The name of the app")
	cmd.Flags().StringVarP(&flags.template, "template", "t", "", "The template to use")
	cmd.RegisterFlagCompletionFunc("template", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		options, err := templates.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return options, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
