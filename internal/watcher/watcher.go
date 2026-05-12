package watcher

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Callback func(paths []string)

type Watcher struct {
	logDir   string
	callback Callback
	debounce time.Duration
	watcher  *fsnotify.Watcher
	done     chan struct{}
}

func New(logDir string, debounce time.Duration, cb Callback) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		logDir:   logDir,
		callback: cb,
		debounce: debounce,
		watcher:  fw,
		done:     make(chan struct{}),
	}

	// Walk the directory tree and add all directories
	if err := w.addDirs(logDir); err != nil {
		fw.Close()
		return nil, err
	}

	return w, nil
}

func (w *Watcher) addDirs(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				log.Printf("Warning: cannot watch %s: %v", path, err)
			}
		}
		return nil
	})
}

func (w *Watcher) Start() {
	go w.loop()
}

func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
}

func (w *Watcher) loop() {
	var mu sync.Mutex
	pending := make(map[string]bool)
	var timer *time.Timer

	flush := func() {
		mu.Lock()
		if len(pending) == 0 {
			mu.Unlock()
			return
		}
		paths := make([]string, 0, len(pending))
		for p := range pending {
			paths = append(paths, p)
		}
		pending = make(map[string]bool)
		mu.Unlock()

		w.callback(paths)
	}

	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about JSONL write/create events
			if !strings.HasSuffix(event.Name, ".jsonl") {
				// If a new directory is created, start watching it
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						w.watcher.Add(event.Name)
					}
				}
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			mu.Lock()
			pending[event.Name] = true
			mu.Unlock()

			// Reset debounce timer
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(w.debounce, flush)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
