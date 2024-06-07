package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net/mail"
	"regexp"

	"github.com/gliderlabs/ssh"
	"github.com/pomdtr/smallweb/server/storage"
	gossh "golang.org/x/crypto/ssh"
)

func NewSSHServer(SSHPort int, db *storage.DB, forwarder *Forwarder, emailer *ValTownEmail) *ssh.Server {
	return &ssh.Server{
		Addr: fmt.Sprintf(":%d", SSHPort),
		PublicKeyHandler: func(ctx ssh.Context, publicKey ssh.PublicKey) bool {
			log.Printf("attempted public key login: %s", publicKey.Type())
			key, err := keyText(publicKey)
			if err != nil {
				return false
			}
			ctx.SetValue("key", key)
			return true
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"user": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
				user, err := db.UserFromContext(ctx)
				if err != nil {
					slog.Info("no user found", slog.String("error", err.Error()))
					return false, gossh.Marshal(ErrorPayload{Message: "no user found"})
				}

				return true, gossh.Marshal(UserResponse{
					ID:    user.PublicID,
					Name:  user.Name,
					Email: user.Email,
				})

			},
			"signup": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
				// if the user already exists, it means they are already authenticated
				if _, err := db.UserFromContext(ctx); err == nil {
					return false, nil
				}

				var params SignupParams
				if err := gossh.Unmarshal(req.Payload, &params); err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "invalid payload"})
				}

				if err := isValidUsername(params.Username); err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "invalid username"})
				}

				if _, err := mail.ParseAddress(params.Email); err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "invalid email"})
				}

				if err := db.CheckUserInfo(params.Email, params.Username); err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "user already exists"})
				}

				code, err := generateVerificationCode()
				if err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "could not generate code"})
				}

				if err := sendVerificationCode(emailer, params.Email, code); err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "could not send code"})
				}

				conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
				codeOk, payload, err := conn.SendRequest("code", true, nil)
				if err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "could not send code"})
				}
				if !codeOk {
					return false, gossh.Marshal(ErrorPayload{Message: "could not send code"})
				}

				var verifyParams VerifyEmailParams
				if err := gossh.Unmarshal(payload, &verifyParams); err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "could not unmarshal code"})
				}

				if verifyParams.Code != code {
					return false, gossh.Marshal(ErrorPayload{Message: "invalid code"})
				}

				key, ok := ctx.Value("key").(string)
				if !ok {
					return false, gossh.Marshal(ErrorPayload{Message: "invalid key"})
				}

				if _, err := db.CreateUser(key, params.Email, params.Username); err != nil {
					return false, gossh.Marshal(ErrorPayload{Message: "could not create user"})
				}

				return true, nil
			},
			"login": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
				var params LoginParams
				if err := gossh.Unmarshal(req.Payload, &params); err != nil {
					return false, nil
				}

				user, err := db.GetUserWithEmail(params.Email)
				if err != nil {
					return false, nil
				}

				code, err := generateVerificationCode()
				if err != nil {
					return false, nil
				}

				if err := sendVerificationCode(emailer, params.Email, code); err != nil {
					return false, nil
				}

				conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
				codeOk, payload, err := conn.SendRequest("code", true, nil)
				if err != nil {
					return false, nil
				}
				if !codeOk {
					return false, nil
				}

				var verifyParams VerifyEmailParams
				if err := gossh.Unmarshal(payload, &verifyParams); err != nil {
					return false, nil
				}

				if verifyParams.Code != code {
					return false, nil
				}

				key, ok := ctx.Value("key").(string)
				if !ok {
					return false, nil
				}

				if err := db.AddUserPublicKey(user.ID, key); err != nil {
					return false, nil
				}

				return true, gossh.Marshal(LoginResponse{Username: user.Name})
			},
			"logout": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
				user, err := db.UserFromContext(ctx)
				if err != nil {
					return false, nil
				}

				key := ctx.Value("key").(string)
				if err := db.DeleteUserPublicKey(user.ID, key); err != nil {
					return false, nil
				}

				return true, nil
			},
			"smallweb-forward": forwarder.HandleSSHRequest,
		},
	}

}

// keyText is the base64 encoded public key for the glider.Session.
func keyText(publicKey gossh.PublicKey) (string, error) {
	kb := base64.StdEncoding.EncodeToString(publicKey.Marshal())
	return fmt.Sprintf("%s %s", publicKey.Type(), kb), nil
}

func isValidUsername(username string) error {
	// Check length
	if len(username) < 3 || len(username) > 15 {
		return fmt.Errorf("username must be between 3 and 15 characters")
	}

	// Check if it only contains alphanumeric characters
	alnumRegex := regexp.MustCompile(`^[a-z][a-z0-9]+$`)
	if !alnumRegex.MatchString(username) {
		return fmt.Errorf("username must only contain lowercase letters and numbers")
	}

	return nil
}

func generateVerificationCode() (string, error) {
	return generateRandomString(8, "0123456789")
}

func generateRandomString(length int, alphabet string) (string, error) {
	result := make([]byte, length)
	alphabetLength := big.NewInt(int64(len(alphabet)))

	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, alphabetLength)
		if err != nil {
			return "", err
		}
		result[i] = alphabet[index.Int64()]
	}

	return string(result), nil
}

func sendVerificationCode(client *ValTownEmail, email string, code string) error {
	err := client.SendEmail(Email{
		To:      email,
		Subject: "Smallweb signup code",
		Text:    fmt.Sprintf("Your Smallweb signup code is: %s", code),
	})

	return err
}
