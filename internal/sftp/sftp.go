package sftp

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/pkg/sftp"
)

func SSHOption(rootPath string, logger *slog.Logger) ssh.Option {
	return func(server *ssh.Server) error {
		if server.SubsystemHandlers == nil {
			server.SubsystemHandlers = map[string]ssh.SubsystemHandler{}
		}

		server.SubsystemHandlers["sftp"] = SubsystemHandler(rootPath, logger)
		return nil
	}
}

func SubsystemHandler(rootPath string, logger *slog.Logger) ssh.SubsystemHandler {
	return func(session ssh.Session) {
		defer func() {
			if r := recover(); r != nil {
				if logger != nil {
					logger.Error("error running sftp middleware", "err", r)
				}
				wish.Println(session, "error running sftp middleware, check the flags you are using")
			}
		}()

		var workingDir string
		if session.User() != "_" {
			workingDir = filepath.Join(rootPath, session.User())
		} else {
			workingDir = rootPath
		}

		root, err := os.OpenRoot(workingDir)
		if err != nil {
			if logger != nil {
				logger.Error("Error opening root", "err", err)
			}

			wish.Errorln(session, err)
		}

		handler := &handlererr{
			Handler: &handler{
				session: session,
				root:    root,
			},
		}

		handlers := sftp.Handlers{
			FilePut:  handler,
			FileList: handler,
			FileGet:  handler,
			FileCmd:  handler,
		}

		requestServer := sftp.NewRequestServer(session, handlers)

		if err := requestServer.Serve(); err != nil && !errors.Is(err, io.EOF) {
			if logger != nil {
				logger.Error("Error serving sftp subsystem", "err", err)
			}
			wish.Errorln(session, err)
		}
	}
}
