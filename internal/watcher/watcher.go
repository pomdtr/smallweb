package watcher

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher *fsnotify.Watcher
	mu      sync.Mutex
	mtimes  map[string]time.Time
	cnames  map[string]string
	root    string
}

func NewWatcher(rootDir string) (*Watcher, error) {
	me := &Watcher{
		mtimes: make(map[string]time.Time),
		cnames: make(map[string]string),
		root:   rootDir,
	}

	me.updateCnames()

	return me, nil
}

func (me *Watcher) updateCnames() error {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.cnames = make(map[string]string)

	// track chosen mtime per mtime so we can prefer the oldest one
	mtimes := make(map[string]time.Time)

	domainEntries, err := os.ReadDir(me.root)
	if err != nil {
		return err
	}

	for _, domainEntry := range domainEntries {
		appEntries, err := os.ReadDir(filepath.Join(me.root, domainEntry.Name()))
		if err != nil {
			continue
		}

		for _, appEntry := range appEntries {
			cnamePath := filepath.Join(me.root, domainEntry.Name(), appEntry.Name(), "CNAME")
			stat, err := os.Stat(cnamePath)
			if err != nil || stat.IsDir() {
				continue
			}

			data, err := os.ReadFile(cnamePath)
			if err != nil {
				continue
			}

			cname := strings.TrimSpace(string(data))
			if cname == "" {
				continue
			}
			domain := fmt.Sprintf("%s.%s", appEntry.Name(), domainEntry.Name())

			mtime := stat.ModTime()

			if prevMTime, ok := mtimes[cname]; !ok {
				// first seen for this cname
				me.cnames[cname] = domain
				mtimes[cname] = mtime
			} else {
				// prefer the oldest ctime
				if mtime.Before(prevMTime) {
					me.cnames[cname] = domain
					mtimes[cname] = mtime
				}
			}
		}
	}

	return nil
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

			if filepath.Dir(event.Name) == me.root {
				// ignore changes in the root dir
				continue
			}

			relPath, err := filepath.Rel(me.root, event.Name)
			if err != nil {
				continue
			}

			parts := strings.Split(relPath, string(os.PathSeparator))
			if len(parts) < 2 {
				// ignore changes outside of app dirs
				continue
			}

			domain := fmt.Sprintf("%s.%s", parts[1], parts[0])
			if filepath.Base(event.Name) == "CNAME" {
				_ = me.updateCnames()
			}

			me.mu.Lock()
			me.mtimes[domain] = fileinfo.ModTime()
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

func (me *Watcher) GetAppMtime(domain string) time.Time {
	me.mu.Lock()
	defer me.mu.Unlock()

	mtime, ok := me.mtimes[domain]
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

func ExtractScheme(r *http.Request) string {
	if scheme := r.URL.Query().Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}

func (me *Watcher) ResolveHostname(hostname string) (string, string, bool) {
	if domain, ok := me.cnames[hostname]; ok {
		hostname = domain
	}

	if _, err := os.Stat(filepath.Join(me.root, hostname, "_")); err == nil {
		return "_", hostname, true
	}

	parts := strings.SplitN(hostname, ".", 2)
	if _, err := os.Stat(filepath.Join(me.root, parts[1], parts[0])); err == nil {
		return parts[0], parts[1], true
	}

	return "", "", false
}
