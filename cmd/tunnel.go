package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
	"github.com/caarlos0/env/v6"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/keygen"
	"github.com/gorilla/handlers"
	"github.com/mitchellh/go-homedir"
	"github.com/pomdtr/smallweb/client"
	"github.com/pomdtr/smallweb/proxy"
	"github.com/spf13/cobra"

	"golang.org/x/crypto/ssh"
)

var (
	ErrMissingSSHAuth = errors.New("no SSH auth found")
)

func installDeno() error {
	// if we are on windows, we need to use the powershell script
	if runtime.GOOS == "windows" {
		return exec.Command("powershell", "-c", "irm https://deno.land/install.ps1 | iex").Run()
	}

	if _, err := exec.LookPath("curl"); err == nil {
		return exec.Command("sh", "-c", "curl -fsSL https://deno.land/x/install/install.sh | sh").Run()
	}

	if _, err := exec.LookPath("wget"); err == nil {
		return exec.Command("sh", "-c", "wget -qO- https://deno.land/x/install/install.sh | sh").Run()
	}

	return nil
}

func setupDenoIfRequired() error {
	if _, err := client.DenoExecutable(); err == nil {
		return nil
	}

	var confirm bool
	form := huh.NewForm(huh.NewGroup(huh.NewConfirm().Description("Deno is required to run smallweb. Do you want to install it now?").Value(&confirm)))
	if err := form.Run(); err != nil {
		return fmt.Errorf("could not get user input: %v", err)
	}

	if !confirm {
		return fmt.Errorf("deno is required to run smallweb")
	}

	fmt.Fprintln(os.Stderr, "Installing deno...")
	if err := installDeno(); err != nil {
		return fmt.Errorf("could not install deno: %v", err)
	}
	fmt.Fprintln(os.Stderr, "Deno installed successfully!")

	return nil
}

func NewCmdTunnel() *cobra.Command {
	return &cobra.Command{
		Use:          "tunnel",
		Short:        "Start a smallweb tunnel",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDenoIfRequired(); err != nil {
				return err
			}

			cc, err := NewClientWithDefaults()
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}

			addr := fmt.Sprintf("%s:%d", cc.Config.Host, cc.Config.SSHPort)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Fatalf("could not dial: %v", err)
			}

			sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cc.sshConfig)
			if err != nil {
				log.Fatalf("could not create client connection: %v", err)
			}
			defer sshConn.Close()

			go ssh.DiscardRequests(reqs)

			var user proxy.UserResponse
			if ok, payload, err := sshConn.SendRequest("user", true, nil); err != nil {
				log.Fatalf("could not send request: %v", err)
			} else if !ok {
				return fmt.Errorf("you are not logged in, please run 'smallweb auth login' or 'smallweb auth signup'")
			} else {
				if err := ssh.Unmarshal(payload, &user); err != nil {
					return fmt.Errorf("could not unmarshal user: %v", err)
				}
			}

			if ok, _, err := sshConn.SendRequest("smallweb-forward", true, nil); err != nil {
				return fmt.Errorf("could not forward: %v", err)
			} else if !ok {
				return fmt.Errorf("user not logged in, please run 'smallweb auth login' or 'smallweb auth signup'")
			}

			cmd.Printf("Smallweb tunnel is up and running, you can now access your apps at https://<app>-%s.smallweb.run\n", user.Name)

			freeport, err := proxy.GetFreePort()
			if err != nil {
				return err
			}

			server := http.Server{
				Addr: fmt.Sprintf(":%d", freeport),
				Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
					app := r.Header.Get("X-Smallweb-App")
					if app == "" {
						http.Error(rw, "X-Smallweb-App header not found", http.StatusBadRequest)
						return
					}

					worker, err := client.NewWorker(app)
					handler := handlers.LoggingHandler(os.Stderr, worker)
					if err != nil {
						rw.WriteHeader(http.StatusInternalServerError)
						return
					}

					handler.ServeHTTP(rw, r)
				}),
			}

			go server.ListenAndServe()

			for ch := range chans {
				if ch.ChannelType() != "forwarded-smallweb" {
					ch.Reject(ssh.UnknownChannelType, "unknown channel type")
				}

				ch, reqs, err := ch.Accept()
				if err != nil {
					log.Fatalf("could not accept channel: %v", err)
				}

				go ssh.DiscardRequests(reqs)
				c, err := net.Dial("tcp", fmt.Sprintf(":%d", freeport))
				if err != nil {
					log.Fatalf("could not dial: %v", err)
				}

				go func() {
					defer ch.Close()
					defer c.Close()
					io.Copy(ch, c)
				}()
				go func() {
					defer ch.Close()
					defer c.Close()
					io.Copy(c, ch)
				}()

			}

			return nil
		},
	}
}

type ClientConfig struct {
	Host    string `env:"SMALLWEB_HOST" envDefault:"smallweb.run"`
	SSHPort int    `env:"SMALLWEB_SSH_PORT" envDefault:"22"`
	Debug   bool   `env:"SMALLWEB_DEBUG" envDefault:"false"`
	KeyType string `env:"SMALLWEB_KEY_TYPE" envDefault:"ed25519"`
	DataDir string `env:"SMALLWEB_DATA_DIR" envDefault:""`
}

// KeygenType returns the keygen key type.
func (cfg *ClientConfig) KeygenType() keygen.KeyType {
	kt := strings.ToLower(cfg.KeyType)
	switch kt {
	case "ed25519":
		return keygen.Ed25519
	case "rsa":
		return keygen.RSA
	default:
		return keygen.Ed25519
	}
}

type Client struct {
	Config       *ClientConfig
	sshConfig    *ssh.ClientConfig
	authKeyPaths []string
}

func NewClient(cfg *ClientConfig) (*Client, error) {
	cc := &Client{
		Config: cfg,
	}

	sshKeys, err := cc.findAuthKeys(cfg.KeyType)
	if err != nil {
		return nil, err
	}
	if len(sshKeys) == 0 {
		dp, err := cc.DataPath()
		if err != nil {
			return nil, err
		}

		_, err = keygen.New(filepath.Join(dp, "smallweb_"+cfg.KeygenType().String()), keygen.WithKeyType(cfg.KeygenType()), keygen.WithWrite())
		if err != nil {
			return nil, err
		}
		sshKeys, err = cc.findAuthKeys(cfg.KeyType)
		if err != nil {
			return nil, err
		}
	}

	var pkam ssh.AuthMethod
	for i := 0; i < len(sshKeys); i++ {
		signer, err := parseKey(sshKeys[i])
		if err != nil && i == len(sshKeys)-1 {
			return nil, ErrMissingSSHAuth
		}
		if err := checkKeyAlgo(signer); err != nil && i == len(sshKeys)-1 {
			return nil, err
		}
		pkam = ssh.PublicKeys(signer)
	}
	cc.authKeyPaths = sshKeys

	cc.sshConfig = &ssh.ClientConfig{
		User:            "smallweb",
		Auth:            []ssh.AuthMethod{pkam},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // nolint
	}
	return cc, nil
}

// ConfigFromEnv loads the configuration from the environment.
func ConfigFromEnv() (*ClientConfig, error) {
	var cfg ClientConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewClientWithDefaults creates a new Charm client with default values.
func NewClientWithDefaults() (*Client, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	cc, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return cc, nil
}

func (cc *Client) DataPath() (string, error) {
	if cc.Config.DataDir != "" {
		return filepath.Join(cc.Config.DataDir, cc.Config.Host), nil
	}

	return filepath.Join(xdg.DataHome, "smallweb", cc.Config.Host), nil
}

// FindAuthKeys looks in a user's XDG smallweb-dir for possible auth keys.
// If no keys are found we return an empty slice.
func (cc *Client) findAuthKeys(keyType string) (pathsToKeys []string, err error) {
	keyPath, err := cc.DataPath()
	if err != nil {
		return nil, err
	}
	m, err := filepath.Glob(filepath.Join(keyPath, "smallweb_*"))
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, nil
	}

	var found []string
	for _, f := range m {
		if filepath.Base(f) == fmt.Sprintf("smallweb_%s", keyType) {
			found = append(found, f)
		}
	}

	return found, nil
}

func algo(keyType string) string {
	if idx := strings.Index(keyType, "@"); idx > 0 {
		return algo(keyType[0:idx])
	}
	parts := strings.Split(keyType, "-")
	if len(parts) == 2 {
		return parts[1]
	}
	if parts[0] == "sk" {
		return algo(strings.TrimPrefix(keyType, "sk-"))
	}
	return parts[0]
}

func checkKeyAlgo(signer ssh.Signer) error {
	ka := signer.PublicKey().Type()
	for _, a := range []string{"ssh-rsa", "ssh-ed25519"} {
		if a == ka {
			return nil
		}
	}
	return fmt.Errorf("sorry, we don't support %s keys yet. Supported types are rsa and ed25519", algo(ka))
}

func parseKey(kp string) (ssh.Signer, error) {
	keyPath, err := homedir.Expand(kp)
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return signer, nil
}
