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
)

type Watcher struct {
	watcher     *fsnotify.Watcher
	mu          sync.Mutex
	mtimes      map[string]time.Time
	baseDomains []string
	appDomains  map[string]string
	root        string
}

func NewWatcher(rootDir string, domain string, additionalDomains ...string) (*Watcher, error) {
	me := &Watcher{
		mtimes:     make(map[string]time.Time),
		appDomains: make(map[string]string),
		root:       rootDir,
	}

	me.baseDomains = append(me.baseDomains, domain)
	me.baseDomains = append(me.baseDomains, additionalDomains...)
	me.RefreshAppDomains()

	return me, nil
}

func (me *Watcher) RefreshAppDomains() {
	appList, err := app.ListApps(me.root)
	if err != nil {
		return
	}

	me.mu.Lock()
	defer me.mu.Unlock()

	me.appDomains = make(map[string]string)
	for _, appName := range appList {
		app, err := app.LoadApp(filepath.Join(me.root, appName))
		if err != nil {
			continue
		}

		for _, domain := range app.Config.AdditionalDomains {
			if existingApp, ok := me.appDomains[domain]; ok {
				existingMtime := me.mtimes[existingApp]
				currentMtime := me.mtimes[appName]
				if currentMtime.Before(existingMtime) {
					me.appDomains[domain] = appName
				}
			} else {
				me.appDomains[domain] = appName
			}
		}
	}
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

			if event.Op == fsnotify.Create {
				fileinfo, err := os.Stat(event.Name)
				if err != nil {
					continue
				}

				if fileinfo.IsDir() {
					_ = me.AddDir(event.Name)
				}
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
			me.mtimes[base] = time.Now()
			me.mu.Unlock()

			if filepath.Dir(event.Name) == me.root || filepath.Base(event.Name) == "smallweb.json" || filepath.Base(event.Name) == "smallweb.jsonc" {
				me.RefreshAppDomains()
			}
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

func (me *Watcher) ResolveDomain(domain string) (string, bool) {
	me.mu.Lock()
	defer me.mu.Unlock()

	if appname, ok := me.appDomains[domain]; ok {
		return appname, true
	}

	parts := strings.SplitN(domain, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	for _, baseDomain := range me.baseDomains {
		if parts[1] == baseDomain {
			return parts[0], true
		}
	}

	return "", false
}
