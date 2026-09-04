package main

import (
	"os"
	"time"
)

// devWatcher polls one entry source file for changes and reports them,
// debounced, on a channel. v0.13 has no `require(...)` yet (see dev.go), so
// there is exactly one file to watch -- no recursive directory walk, no
// project-wide scan.
//
// Polling a single file's mtime/size every pollInterval is deliberately the
// whole implementation: it is cheap (one stat call), handles every save
// style an editor uses -- write-in-place, or the common atomic
// write-temp-then-rename, both of which leave a normal, statable file at
// the watched path afterward -- and needs no new dependency. A missing file
// (mid-atomic-rename, or briefly absent) is treated as "no change yet", not
// a failure: see section 39, the watcher simply keeps polling.
const (
	devWatchPollInterval = 100 * time.Millisecond
	devWatchDebounce     = 200 * time.Millisecond
)

type devWatcher struct {
	path   string
	events chan<- devControllerEvent
	stop   chan struct{}
	done   chan struct{}
}

func newDevWatcher(path string, events chan<- devControllerEvent) *devWatcher {
	return &devWatcher{path: path, events: events, stop: make(chan struct{}), done: make(chan struct{})}
}

// start begins polling. It establishes its baseline from the file's current
// state first, so it never reports the state the initial build already used
// as a spurious "change".
func (w *devWatcher) start() {
	go w.run()
}

func (w *devWatcher) close() {
	close(w.stop)
	<-w.done
}

func (w *devWatcher) run() {
	defer close(w.done)
	lastModTime, lastSize, lastMissing := w.snapshot()
	var changedAt time.Time

	ticker := time.NewTicker(devWatchPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			modTime, size, missing := w.snapshot()
			if missing != lastMissing || modTime != lastModTime || size != lastSize {
				lastModTime, lastSize, lastMissing = modTime, size, missing
				if !missing {
					changedAt = time.Now()
				}
			}
			if !changedAt.IsZero() && time.Since(changedAt) >= devWatchDebounce {
				changedAt = time.Time{}
				select {
				case w.events <- devControllerEvent{kind: eventSourceChanged}:
				case <-w.stop:
					return
				}
			}
		}
	}
}

func (w *devWatcher) snapshot() (modTime time.Time, size int64, missing bool) {
	info, err := os.Stat(w.path)
	if err != nil {
		return time.Time{}, 0, true
	}
	return info.ModTime(), info.Size(), false
}
