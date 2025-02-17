package sftp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/pkg/sftp"
)

// Based on https://github.com/pkg/sftp/blob/master/request-example.go

type handler struct {
	session ssh.Session
	root    *os.Root
}

func (h *handler) Filecmd(r *sftp.Request) error {
	switch r.Method {
	case "Rename":
		if _, err := h.root.Stat(strings.TrimPrefix(r.Filepath, "/")); err != nil {
			return err
		}

		if _, err := h.root.Stat(strings.TrimPrefix(r.Target, "/")); err == nil {
			return fmt.Errorf("target file exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		src := filepath.Join(h.root.Name(), strings.TrimPrefix(r.Filepath, "/"))
		dst := filepath.Join(h.root.Name(), strings.TrimPrefix(r.Target, "/"))
		return os.Rename(src, dst)
	case "Link":
		if _, err := h.root.Stat(strings.TrimPrefix(r.Filepath, "/")); err != nil {
			return err
		}

		if _, err := h.root.Stat(strings.TrimPrefix(r.Target, "/")); err == nil {
			return fmt.Errorf("target file exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		src := filepath.Join(h.root.Name(), strings.TrimPrefix(r.Filepath, "/"))
		dst := filepath.Join(h.root.Name(), strings.TrimPrefix(r.Target, "/"))
		return os.Link(src, dst)
	case "Symlink":
		if _, err := h.root.Stat(strings.TrimPrefix(r.Filepath, "/")); err != nil {
			return fmt.Errorf("file does not exist")
		}

		if _, err := h.root.Stat(strings.TrimPrefix(r.Target, "/")); err == nil {
			return fmt.Errorf("target file exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		src := filepath.Join(h.root.Name(), strings.TrimPrefix(r.Filepath, "/"))
		dst := filepath.Join(h.root.Name(), strings.TrimPrefix(r.Target, "/"))
		return os.Symlink(src, dst)
	case "Rmdir":
		return h.root.Remove(strings.TrimPrefix(r.Filepath, "/"))
	case "Remove":
		return h.root.Remove(strings.TrimPrefix(r.Filepath, "/"))
	case "Mkdir":
		return h.root.Mkdir(strings.TrimPrefix(r.Filepath, "/"), 0777)
	case "Setstat":
		if _, err := h.root.Stat(strings.TrimPrefix(r.Filepath, "/")); err != nil {
			return fmt.Errorf("file does not exist")
		}

		fp := filepath.Join(h.root.Name(), strings.TrimPrefix(r.Filepath, "/"))
		attrs := r.Attributes()

		if attrs.Size != 0 {
			if err := os.Truncate(fp, int64(attrs.Size)); err != nil {
				return err
			}
		}

		if attrs.Mode != 0 {
			if err := os.Chmod(fp, os.FileMode(attrs.Mode)); err != nil {
				return err
			}
		}

		if attrs.GID != 0 || attrs.UID != 0 {
			if err := os.Chown(fp, int(attrs.UID), int(attrs.GID)); err != nil {
				return err
			}
		}

		if attrs.Atime != 0 && attrs.Mtime != 0 {
			atime := time.Unix(int64(attrs.Atime), 0)
			mtime := time.Unix(int64(attrs.Mtime), 0)
			if err := os.Chtimes(fp, atime, mtime); err != nil {
				return err
			}
		}

		return nil
	}
	return errors.New("unsupported")
}

func (h *handler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "List":
		if r.Filepath == "/" {
			entries, err := os.ReadDir(h.root.Name())
			if err != nil {
				return nil, err
			}

			var fileInfos []os.FileInfo
			for _, entry := range entries {
				fileInfo, err := entry.Info()
				if err != nil {
					return nil, err
				}
				fileInfos = append(fileInfos, fileInfo)
			}

			return listerat(fileInfos), nil
		}

		f, err := h.root.Open(strings.TrimPrefix(r.Filepath, "/"))
		if err != nil {
			return nil, err
		}

		entries, err := f.ReadDir(0)
		if err != nil {
			return nil, err
		}

		var fileInfos []os.FileInfo
		for _, entry := range entries {
			fileInfo, err := entry.Info()
			if err != nil {
				return nil, err
			}
			fileInfos = append(fileInfos, fileInfo)
		}

		return listerat(fileInfos), nil

	case "Stat":
		if r.Filepath == "/" {
			stat, err := os.Stat(h.root.Name())
			if err != nil {
				return nil, err
			}

			return listerat([]os.FileInfo{stat}), nil
		}

		info, err := h.root.Stat(strings.TrimPrefix(r.Filepath, "/"))
		if err != nil {
			return nil, err
		}

		return listerat([]os.FileInfo{info}), nil
	}

	return nil, errors.New("unsupported")
}

func (h *handler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	f, err := h.root.OpenFile(strings.TrimPrefix(r.Filepath, "/"), flag(r.Pflags()), 0644)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func flag(pflags sftp.FileOpenFlags) int {
	var flag int
	if pflags.Creat {
		flag |= os.O_CREATE
	}
	if pflags.Trunc {
		flag |= os.O_TRUNC
	}
	if pflags.Excl {
		flag |= os.O_EXCL
	}
	if pflags.Write {
		flag |= os.O_WRONLY
	}
	if pflags.Read {
		flag |= os.O_RDONLY
	}
	if pflags.Append {
		flag |= os.O_APPEND
	}

	return flag
}

func (h *handler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	if r.Filepath == "/" {
		return nil, os.ErrInvalid
	}

	return h.root.Open(strings.TrimPrefix(r.Filepath, "/"))
}

type handlererr struct {
	Handler *handler
}

func (f *handlererr) Filecmd(r *sftp.Request) error {
	err := f.Handler.Filecmd(r)
	if err != nil {
		wish.Errorln(f.Handler.session, err)
	}
	return err
}
func (f *handlererr) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	result, err := f.Handler.Filelist(r)
	if err != nil {
		wish.Errorln(f.Handler.session, err)
	}
	return result, err
}
func (f *handlererr) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	result, err := f.Handler.Filewrite(r)
	if err != nil {
		wish.Errorln(f.Handler.session, err)
	}
	return result, err
}
func (f *handlererr) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	result, err := f.Handler.Fileread(r)
	if err != nil {
		wish.Errorln(f.Handler.session, err)
	}
	return result, err
}

type listerat []os.FileInfo

func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}
