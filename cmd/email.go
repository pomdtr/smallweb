package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

type Message struct {
	Subject string
	From    string
	To      string
	Text    string
}

func (me *Message) String() string {
	builder := strings.Builder{}
	builder.WriteString("Subject: ")
	builder.WriteString(me.Subject)
	builder.WriteString("\n")

	builder.WriteString("From: ")
	builder.WriteString(me.From)
	builder.WriteString("\n")

	builder.WriteString("To: ")
	builder.WriteString(me.To)
	builder.WriteString("\n")

	builder.WriteString("Content-Type: text/plain; charset=utf-8")

	builder.WriteString("\n")
	builder.WriteString("\n")

	builder.WriteString(me.Text)

	return builder.String()
}

func NewCmdEmail() *cobra.Command {
	var flags struct {
		subject string
		from    string
		body    string
	}

	cmd := &cobra.Command{
		Use:               "email <app>",
		Short:             "Send an email",
		Long:              "Send an email to an app",
		ValidArgsFunction: completeApp,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var text string
			if cmd.Flags().Changed("body") {
				text = flags.body
			} else if !isatty.IsTerminal(os.Stdin.Fd()) {
				bytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read from stdin: %w", err)
				}

				text = string(bytes)
			} else {
				return fmt.Errorf("body is required")
			}

			message := &Message{
				Subject: flags.subject,
				From:    flags.from,
				To:      fmt.Sprintf("%s@%s", args[0], k.String("domain")),
				Text:    text,
			}

			a, err := app.LoadApp(args[0], k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), args[0]))
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			wk := worker.NewWorker(a, k.String("email"))

			if err := wk.SendEmail(cmd.Context(), strings.NewReader(message.String())); err != nil {
				return fmt.Errorf("failed to send email: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.subject, "subject", "s", "", "Email's subject")
	cmd.Flags().StringVarP(&flags.from, "from", "f", "", "Email's sender")
	cmd.Flags().StringVarP(&flags.body, "body", "b", "", "Email's body")

	return cmd
}
