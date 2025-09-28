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
	cnames       map[string]string
	root         string
}

func NewWatcher(rootDir string, reloadConfig func()) (*Watcher, error) {
	apps, err := app.LookupApps(rootDir)
	if err != nil {
		return nil, err
	}

	cnames := make(map[string]string)
	for _, app := range apps {
		for _, cnamePath := range []string{
			filepath.Join(rootDir, app, "CNAME"),
			filepath.Join(rootDir, app, ".smallweb", "CNAME"),
		} {
			cnameBytes, err := os.ReadFile(cnamePath)
			if err != nil {
				continue
			}

			domain := strings.TrimSpace(string(cnameBytes))
			if domain != "" {
				cnames[domain] = app
			}
		}
	}

	me := &Watcher{
		mtimes:       make(map[string]time.Time),
		cnames:       cnames,
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
				apps, err := app.LookupApps(me.root)
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

			dir := filepath.Dir(event.Name)
			if dir == me.root {
				continue
			}

			var app string
			for dir != me.root {
				app = filepath.Base(dir)
				dir = filepath.Dir(dir)
			}

			if strings.HasPrefix(app, ".") {
				continue
			}

			if event.Name == filepath.Join(me.root, app, "CNAME") || event.Name == filepath.Join(me.root, app, ".smallweb", "CNAME") {
				cnameBytes, err := os.ReadFile(event.Name)
				if err != nil {
					continue
				}

				domain := strings.TrimSpace(string(cnameBytes))
				if domain == "" {
					continue
				}

				me.mu.Lock()
				me.cnames[domain] = app
				me.mu.Unlock()
				continue
			}

			me.mu.Lock()
			me.mtimes[app] = fileinfo.ModTime()
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

func (me *Watcher) LookupDomain(domain string) (string, bool) {
	domain, ok := me.cnames[domain]
	return domain, ok
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
