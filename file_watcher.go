package parser

import (
	"context"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches for file system changes
type FileWatcher interface {
	// Watch starts watching the specified directory for changes
	Watch(ctx context.Context, dir, extension string, recursive bool, callback func(name string)) error

	// Close stops watching and cleans up resources
	Close() error
}

// fsnotifyWatcher implements FileWatcher using fsnotify
type fsnotifyWatcher struct {
	watcher *fsnotify.Watcher
	mu      sync.Mutex
	closed  bool
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher() (FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &fsnotifyWatcher{
		watcher: watcher,
	}, nil
}

// Watch implements FileWatcher
func (f *fsnotifyWatcher) Watch(ctx context.Context, dir, extension string, recursive bool, callback func(name string)) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return ErrWatcherClosed
	}

	// Add the directory to watch
	err := f.watcher.Add(dir)
	if err != nil {
		return err
	}

	// If recursive, add subdirectories
	if recursive {
		err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() && path != dir {
				return f.watcher.Add(path)
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	// Start watching in a goroutine
	go f.watchLoop(ctx, dir, extension, callback)

	return nil
}

// watchLoop handles file system events
func (f *fsnotifyWatcher) watchLoop(ctx context.Context, rootDir, extension string, callback func(name string)) {
	// Debounce file changes to avoid multiple events for the same file
	debounce := make(map[string]*time.Timer)
	debounceMu := sync.Mutex{}

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}

			// Only handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Check if file matches our extension
				if extension != "" && filepath.Ext(event.Name) != extension {
					continue
				}

				// Get relative path from root directory
				relPath, err := filepath.Rel(rootDir, event.Name)
				if err != nil {
					continue
				}

				// Remove extension for template name
				if extension != "" {
					relPath = relPath[:len(relPath)-len(extension)]
				}

				// Debounce the event
				debounceMu.Lock()
				if timer, exists := debounce[relPath]; exists {
					timer.Stop()
				}

				debounce[relPath] = time.AfterFunc(100*time.Millisecond, func() {
					callback(relPath)
					debounceMu.Lock()
					delete(debounce, relPath)
					debounceMu.Unlock()
				})
				debounceMu.Unlock()
			}

		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			_ = err
		}
	}
}

// Close implements FileWatcher
func (f *fsnotifyWatcher) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return nil
	}

	f.closed = true
	return f.watcher.Close()
}
