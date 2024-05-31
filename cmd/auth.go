package cmd

import (
	"fmt"
	"log"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func promptEmail() (string, error) {
	var email string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Email").Description("Enter your email").Value(&email),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return email, nil
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

	return cmd
}

func NewCmdAuthSignup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Sign up for the smallweb server",
		Args:  cobra.NoArgs,
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

			ok, payload, err := conn.SendRequest("get-user", true, nil)
			if err != nil {
				return fmt.Errorf("could not send request: %v", err)
			} else if ok {
				var user UserResponse
				if err := ssh.Unmarshal(payload, &user); err != nil {
					return fmt.Errorf("could not unmarshal user: %v", err)
				}

				if user.EmailVerified {
					fmt.Println("You are already logged in.")
					return nil
				}

				return verifyEmail(conn, user.Email)
			}

			email, err := promptEmail()
			if err != nil {
				return fmt.Errorf("could not prompt for email: %v", err)
			}

			ok, payload, err = conn.SendRequest("signup", true, ssh.Marshal(SignupBody{Email: email}))
			if err != nil {
				log.Fatalf("could not send request: %v", err)
			} else if !ok {
				return fmt.Errorf("failed to sign up")
			}

			var res UserResponse
			if err := ssh.Unmarshal(payload, &res); err != nil {
				return fmt.Errorf("could not unmarshal user: %v", err)
			}

			return verifyEmail(conn, res.Email)
		},
	}

	return cmd
}

func verifyEmail(conn ssh.Conn, email string) error {
	emailSent, _, err := conn.SendRequest("verify-email", true, nil)
	if err != nil {
		log.Fatalf("could not send request: %v", err)
	}
	if !emailSent {
		return fmt.Errorf("failed to send verification email")
	}

	code, err := promptVerificationCode()
	if err != nil {
		return fmt.Errorf("could not prompt for verification code: %v", err)
	}

	emailVerified, _, err := conn.SendRequest("verify-email", true, ssh.Marshal(VerifyEmailBody{Code: code}))
	if err != nil {
		log.Fatalf("could not send request: %v", err)
	}

	if !emailVerified {
		return fmt.Errorf("failed to verify email")
	}

	return nil
}

func NewCmdAuthLogin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to the smallweb server",
		Args:  cobra.NoArgs,
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

			ok, _, err := conn.SendRequest("/me", true, nil)
			if err != nil {
				return fmt.Errorf("could not send request: %v", err)
			} else if ok {
				fmt.Println("You are already logged in.")
				return nil
			}

			ok, _, err = conn.SendRequest("/auth/login", true, nil)
			if err != nil {
				log.Fatalf("could not send request: %v", err)
			} else if !ok {
				return fmt.Errorf("failed to log out")
			}

			return nil

		},
	}

	return cmd
}

func NewCmdAuthLogout() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of the smallweb server",
		Args:  cobra.NoArgs,
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

			ok, _, err := conn.SendRequest("get-user", true, nil)
			if err != nil {
				return fmt.Errorf("could not send request: %v", err)
			} else if !ok {
				fmt.Println("You are not logged in.")
				return nil
			}

			ok, _, err = conn.SendRequest("logout", true, nil)
			if err != nil {
				log.Fatalf("could not send request: %v", err)
			} else if !ok {
				return fmt.Errorf("failed to log out")
			}

			return nil

		},
	}

	return cmd
}
