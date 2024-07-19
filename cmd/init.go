package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/pomdtr/smallweb/templates"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

type Inputs struct {
	Dir      string
	Template string
}

func (me *Inputs) Fill() error {
	var fields []huh.Field

	if me.Dir == "" {
		fields = append(
			fields,
			huh.NewInput().Title("Where should we create your project?").Value(&me.Dir).Placeholder("."),
		)
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

		fields = append(
			fields,
			huh.NewSelect[string]().Title("Which template would you like to use?").Options(options...).Value(&me.Template),
		)
	}

	if len(fields) == 0 {
		return nil
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to run form: %w", err)
	}

	if me.Dir != "" {
		dir, err := utils.ExpandTilde(me.Dir)
		if err != nil {
			return fmt.Errorf("failed to expand dir: %w", err)
		}

		me.Dir = dir
	} else {
		me.Dir = "."
	}

	return nil
}

func NewCmdInit() *cobra.Command {
	var flags struct {
		template string
	}

	cmd := &cobra.Command{
		Use:     "init [dir]",
		Short:   "Init a new smallweb app",
		GroupID: CoreGroupID,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var dir string
			if len(args) > 0 {
				dir = args[0]
			}

			inputs := Inputs{
				Dir:      dir,
				Template: flags.template,
			}

			if err := inputs.Fill(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					return nil
				}
				return fmt.Errorf("failed to fill inputs: %w", err)
			}

			if err := templates.Install(inputs.Template, inputs.Dir); err != nil {
				return fmt.Errorf("failed to install template: %w", err)
			}

			cmd.PrintErrf("Template %s installed in %s\n", inputs.Template, inputs.Dir)
			return nil
		},
	}

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
