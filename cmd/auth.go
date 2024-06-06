package cmd

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func signupForm() (*SignupParams, error) {
	var params SignupParams

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Email").Description("Enter your email").Value(&params.Email),
			huh.NewInput().Title("Username").Description("Choose a username").Value(&params.Username),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &params, nil
}

func loginForm() (*LoginParams, error) {
	var params LoginParams

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Email").Description("Enter your email").Value(&params.Email),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &params, nil
}

func promptVerificationCode() (string, error) {
	var code string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Code").Description("Enter the verification code").Value(&code),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return code, nil
}

func NewCmdAuth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}

	cmd.AddCommand(NewCmdAuthSignup())
	cmd.AddCommand(NewCmdAuthLogin())
	cmd.AddCommand(NewCmdAuthLogout())
	cmd.AddCommand(NewCmdAuthWhoAmI())

	return cmd
}

func NewCmdAuthSignup() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "signup",
		Short:        "Sign up for the smallweb server",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewClientWithDefaults()
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}

			addr := fmt.Sprintf("%s:%d", client.Config.Host, client.Config.SSHPort)
			conn, err := net.DialTimeout("tcp", addr, client.sshConfig.Timeout)
			if err != nil {
				return err
			}

			c, chans, reqs, err := ssh.NewClientConn(conn, addr, client.sshConfig)
			if err != nil {
				return err
			}

			go func() {
				for req := range reqs {
					if req.Type == "code" {
						code, err := promptVerificationCode()
						if err != nil {
							req.Reply(false, nil)
						}

						req.Reply(true, ssh.Marshal(VerifyEmailParams{Code: code}))
					}
					req.Reply(false, nil)
				}
			}()

			go func() {
				for newChan := range chans {
					newChan.Reject(ssh.Prohibited, "no channels available")
				}
			}()

			userExists, _, err := c.SendRequest("user", true, nil)
			if err != nil {
				return fmt.Errorf("could not send request: %v", err)
			}

			if userExists {
				fmt.Println("You are already authenticated.")
				return nil
			}

			params, err := signupForm()
			if err != nil {
				return fmt.Errorf("could not prompt for email: %v", err)
			}

			if ok, payload, err := c.SendRequest("signup", true, ssh.Marshal(params)); err != nil {
				return fmt.Errorf("could not sign up: %v", err)
			} else if !ok {
				var resp ErrorResponse
				if err := ssh.Unmarshal(payload, &resp); err != nil {
					log.Fatalf("could not unmarshal response: %v", err)
				}

				return fmt.Errorf("failed to sign up: %s", resp.Message)
			} else {
				fmt.Printf("Signed up as %s\n", params.Username)
				return nil
			}
		},
	}

	return cmd
}

func NewCmdAuthLogin() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "login",
		Short:        "Log in to the smallweb server",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewClientWithDefaults()
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}

			addr := fmt.Sprintf("%s:%d", client.Config.Host, client.Config.SSHPort)
			conn, err := net.DialTimeout("tcp", addr, client.sshConfig.Timeout)
			if err != nil {
				return err
			}

			c, chans, reqs, err := ssh.NewClientConn(conn, addr, client.sshConfig)
			if err != nil {
				return err
			}

			go func() {
				for req := range reqs {
					if req.Type == "code" {
						code, err := promptVerificationCode()
						if err != nil {
							req.Reply(false, nil)
						}

						req.Reply(true, ssh.Marshal(VerifyEmailParams{Code: code}))
					}
					req.Reply(false, nil)
				}
			}()

			go func() {
				for newChan := range chans {
					newChan.Reject(ssh.Prohibited, "no channels available")
				}
			}()

			ok, _, err := c.SendRequest("user", true, nil)
			if err != nil {
				return fmt.Errorf("could not send request: %v", err)
			} else if ok {
				fmt.Println("You are already logged in.")
				return nil
			}

			params, err := loginForm()
			if err != nil {
				return fmt.Errorf("could not prompt for username: %v", err)
			}

			if ok, payload, err := c.SendRequest("login", true, ssh.Marshal(params)); err != nil {
				return fmt.Errorf("could not log in: %v", err)
			} else if !ok {
				var resp ErrorResponse
				if err := ssh.Unmarshal(payload, &resp); err != nil {
					log.Fatalf("could not unmarshal response: %v", err)
				}

				return fmt.Errorf("failed to log out: %s", resp.Message)
			} else {
				fmt.Fprintf(os.Stderr, "You are now logged in!\n")
				return nil
			}
		},
	}

	return cmd
}

func NewCmdAuthLogout() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "logout",
		Short:        "Log out of the smallweb server",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewClientWithDefaults()
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}

			addr := fmt.Sprintf("%s:%d", client.Config.Host, client.Config.SSHPort)
			conn, err := ssh.Dial("tcp", addr, client.sshConfig)
			if err != nil {
				log.Fatalf("could not dial: %v", err)
			}

			ok, _, err := conn.SendRequest("user", true, nil)
			if err != nil {
				return fmt.Errorf("could not send request: %v", err)
			} else if !ok {
				fmt.Println("You are not logged in.")
				return nil
			}

			if ok, payload, err := conn.SendRequest("logout", true, nil); err != nil {
				return fmt.Errorf("could not log out: %v", err)
			} else if !ok {
				var resp ErrorResponse
				if err := ssh.Unmarshal(payload, &resp); err != nil {
					return fmt.Errorf("failed to log out: %v", err)
				}

				return fmt.Errorf("failed to log out: %s", resp.Message)
			} else {
				fmt.Println("You are now logged out.")
				return nil
			}
		},
	}

	return cmd
}

func NewCmdAuthWhoAmI() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "whoami",
		Short:        "Display the current user",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewClientWithDefaults()
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}

			addr := fmt.Sprintf("%s:%d", client.Config.Host, client.Config.SSHPort)
			conn, err := ssh.Dial("tcp", addr, client.sshConfig)
			if err != nil {
				log.Fatalf("could not dial: %v", err)
			}

			ok, payload, err := conn.SendRequest("user", true, nil)
			if err != nil {
				log.Fatalf("could not send request: %v", err)
			} else if !ok {
				return fmt.Errorf("not logged in")
			}

			var user UserResponse
			if err := ssh.Unmarshal(payload, &user); err != nil {
				log.Fatalf("could not unmarshal user: %v", err)
			}

			fmt.Printf("You are logged in as %s\n", user.Name)

			return nil
		},
	}

	return cmd
}
