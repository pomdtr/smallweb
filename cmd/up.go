package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/charmbracelet/keygen"
	"github.com/mitchellh/go-homedir"
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"

	"golang.org/x/crypto/ssh"
)

var (
	ErrMissingSSHAuth = errors.New("no SSH auth found")
)

func NewCmdUp() *cobra.Command {
	return &cobra.Command{
		Use:          "up",
		Short:        "Start a smallweb tunnel",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := NewClientWithDefaults()
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}

			addr := fmt.Sprintf("%s:%d", client.Config.Host, client.Config.SSHPort)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Fatalf("could not dial: %v", err)
			}

			sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, client.sshConfig)
			if err != nil {
				log.Fatalf("could not create client connection: %v", err)
			}
			defer sshConn.Close()

			go func() {
				for req := range reqs {
					if req.Type != "email" {
						req.Reply(false, nil)
						continue
					}
					var email Email
					if err := ssh.Unmarshal(req.Payload, &email); err != nil {
						req.Reply(false, nil)
						continue
					}

					app, err := email.App()
					if err != nil {
						req.Reply(false, nil)
						continue
					}

					worker, err := NewWorker(app)
					if err != nil {
						req.Reply(false, nil)
						continue
					}

					if _, err := worker.Email(&email); err != nil {
						req.Reply(false, nil)
						continue
					}

					req.Reply(true, nil)
				}
			}()

			ok, _, err := sshConn.SendRequest("smallweb-forward", true, nil)
			if err != nil {
				return fmt.Errorf("could not forward: %v", err)
			}
			if !ok {
				return fmt.Errorf("user not logged in, please run 'smallweb auth login' or 'smallweb auth signup'")
			}

			go func() {
				ticker := time.NewTicker(30 * time.Second)
				for {
					<-ticker.C
					_, _, err := sshConn.SendRequest("keepalive", true, nil)
					if err != nil {
						log.Printf("could not send keepalive: %v", err)
						break
					}
				}
			}()

			handleReq := func(req *Request) (*Response, error) {
				alias, err := req.App()
				if err != nil {
					return nil, fmt.Errorf("could not get alias: %v", err)
				}

				worker, err := NewWorker(alias)
				if err != nil {
					return nil, fmt.Errorf("could not create worker: %v", err)
				}
				return worker.Fetch(req)
			}

			exampleUrl := fmt.Sprintf("https://<app>-<user>.%s", client.Config.Host)
			fmt.Printf("Smallweb tunnel is up and running, you can now access your apps at: %s\n", exampleUrl)

			for newChannel := range chans {
				if newChannel.ChannelType() != "forwarded-smallweb" {
					newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				}

				ch, reqs, err := newChannel.Accept()
				if err != nil {
					log.Printf("could not accept channel: %v", err)
				}
				go ssh.DiscardRequests(reqs)

				go func(ch ssh.Channel) {
					decoder := json.NewDecoder(ch)
					var req *Request
					if err := decoder.Decode(&req); err != nil {
						log.Printf("could not decode request: %v", err)
						return
					}

					resp, err := handleReq(req)
					if err != nil {
						resp = &Response{
							Code: 500,
							Body: []byte("Internal server error"),
						}
					}

					encoder := json.NewEncoder(ch)
					encoder.SetEscapeHTML(false)
					if err := encoder.Encode(resp); err != nil {
						log.Printf("could not encode response: %v", err)
						return
					}
				}(ch)
			}

			return nil
		},
	}
}

type ClientConfig struct {
	Host    string `env:"SMALLWEB_HOST" envDefault:"smallweb.run"`
	SSHPort int    `env:"SMALLWEB_SSH_PORT" envDefault:"2222"`
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
	scope := gap.NewScope(gap.User, filepath.Join("smallweb", cc.Config.Host))
	dataPath, err := scope.DataPath("")
	if err != nil {
		return "", err
	}
	return dataPath, nil
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
