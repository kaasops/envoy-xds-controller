package filewatcher

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher    *fsnotify.Watcher
	files      map[string]fileEntry // originalPath -> fileEntry
	dirs       map[string]struct{}  // watched directories
	mu         sync.Mutex
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

type fileEntry struct {
	realPath string
	callback func(string)
}

// NewFileWatcher initializes the watcher
func NewFileWatcher() (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fw := &FileWatcher{
		watcher:    watcher,
		files:      make(map[string]fileEntry),
		dirs:       make(map[string]struct{}),
		cancelFunc: cancel,
	}

	fw.wg.Add(1)
	go fw.run(ctx)

	return fw, nil
}

// Add registers a file for watching with a specific callback
func (fw *FileWatcher) Add(file string, callback func(string)) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	absPath := filepath.Clean(file)
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		realPath = absPath // fallback if symlink resolution fails
	}

	fw.files[absPath] = fileEntry{
		realPath: realPath,
		callback: callback,
	}

	dir := filepath.Dir(absPath)
	if _, alreadyWatching := fw.dirs[dir]; !alreadyWatching {
		if err := fw.watcher.Add(dir); err != nil {
			return fmt.Errorf("failed to watch directory %s: %w", dir, err)
		}
		fw.dirs[dir] = struct{}{}
	}

	return nil
}

// Remove unregisters a file from watching
func (fw *FileWatcher) Remove(file string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	delete(fw.files, filepath.Clean(file))
}

// Cancel stops all watching and cleans up resources
func (fw *FileWatcher) Cancel() {
	fw.cancelFunc()
	fw.wg.Wait()
	_ = fw.watcher.Close()
}

func (fw *FileWatcher) run(ctx context.Context) {
	defer fw.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			fw.mu.Lock()
			for original, entry := range fw.files {
				cleanEvent := filepath.Clean(event.Name)

				// Trigger callback on direct file change
				if cleanEvent == original &&
					(event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0) {
					cb := entry.callback
					fw.mu.Unlock()
					cb(original)
					fw.mu.Lock()
					continue
				}

				// Trigger callback if symlink target changed
				currentReal, err := filepath.EvalSymlinks(original)
				if err == nil && currentReal != entry.realPath {
					entry.realPath = currentReal
					fw.files[original] = entry
					cb := entry.callback
					fw.mu.Unlock()
					cb(original)
					fw.mu.Lock()
				}
			}
			fw.mu.Unlock()

		case _, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			// Errors are silently ignored
		}
	}
}
