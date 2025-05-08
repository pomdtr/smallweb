package watcher

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/utils"
)

type Watcher struct {
	watcher      *fsnotify.Watcher
	mu           sync.Mutex
	reloadConfig func()
	mtimes       map[string]time.Time
	root         string
}

func NewWatcher(rootDir string, reloadConfig func()) (*Watcher, error) {
	me := &Watcher{
		mtimes:       make(map[string]time.Time),
		root:         rootDir,
		reloadConfig: reloadConfig,
	}

	return me, nil
}

func (me *Watcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	me.watcher = watcher
	if err := me.AddDir(me.root); err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed")
			}
			if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) && !event.Has(fsnotify.Remove) {
				continue
			}
			fileinfo, err := os.Stat(event.Name)
			if err != nil {
				continue
			}
			if fileinfo.IsDir() {
				if event.Has(fsnotify.Create) {
					_ = me.AddDir(event.Name)
				}
				continue
			}

			// if the event is originated from config file, reload the config and update all mtimes
			if event.Name == utils.FindConfigPath(me.root) {
				go me.reloadConfig()
				apps, err := app.ListApps(me.root)
				if err != nil {
					continue
				}

				me.mu.Lock()
				for _, app := range apps {
					me.mtimes[app] = fileinfo.ModTime()
				}
				me.mu.Unlock()
				continue
			}

			var base string
			parent := filepath.Dir(event.Name)
			if parent == me.root {
				continue
			}

			for parent != me.root {
				base = filepath.Base(parent)
				parent = filepath.Dir(parent)
			}

			if strings.HasPrefix(base, ".") {
				continue
			}

			me.mu.Lock()
			me.mtimes[base] = fileinfo.ModTime()
			me.mu.Unlock()
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher closed")
			}

			if err != nil {
				return err
			}
		}
	}

}

func (me *Watcher) Stop() {
	if me.watcher == nil {
		return
	}

	me.watcher.Close()
	me.watcher = nil
}

func (me *Watcher) GetAppMtime(app string) time.Time {
	me.mu.Lock()
	defer me.mu.Unlock()

	mtime, ok := me.mtimes[app]
	if !ok {
		return time.Time{}
	}

	return mtime
}

func (me *Watcher) AddDir(dir string) error {
	if err := me.watcher.Add(dir); err != nil {
		return err
	}

	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		name := filepath.Base(path)
		if name == ".git" {
			return filepath.SkipDir
		}

		parent := filepath.Dir(path)

		// data dirs should be ignored
		if parent != me.root && (name == "data" || name == "node_modules") {
			return filepath.SkipDir
		}

		// _ prefixed app should be ignored
		if parent == me.root && strings.HasPrefix(name, "_") {
			return filepath.SkipDir
		}

		if err := me.watcher.Add(path); err != nil {
			return err
		}

		return nil
	})

	return nil
}
