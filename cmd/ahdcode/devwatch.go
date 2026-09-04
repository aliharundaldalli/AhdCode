package main

import (
	"os"
	"time"
)

// devWatcher polls a bounded set of local source files for changes and
// reports them, debounced, on a channel. The set starts as just the entry
// file and is replaced wholesale after every build attempt (see dev.go) with
// the entry plus the compiler's resolved require(...) graph plus any
// require(...) target the latest attempt named but could not find yet --
// never a recursive directory walk or project-wide scan (section 26): only
// files the compiler itself named are ever watched.
//
// Polling each file's mtime/size every pollInterval is deliberately the
// whole implementation: it is cheap (one stat call per file), handles every
// save style an editor uses -- write-in-place, or the common atomic
// write-temp-then-rename, both of which leave a normal, statable file at the
// watched path afterward -- and needs no new dependency. A missing file
// (mid-atomic-rename, briefly absent, or a require(...) target not created
// yet) is treated as "no change yet", not a failure: the watcher simply
// keeps polling until it appears.
const (
	devWatchPollInterval = 100 * time.Millisecond
	devWatchDebounce     = 200 * time.Millisecond
)

type fileSnapshot struct {
	modTime time.Time
	size    int64
	missing bool
}

type devWatcher struct {
	events    chan<- devControllerEvent
	stop      chan struct{}
	done      chan struct{}
	setPaths  chan []string
	snapshots map[string]fileSnapshot
}

func newDevWatcher(initialPaths []string, events chan<- devControllerEvent) *devWatcher {
	w := &devWatcher{
		events: events, stop: make(chan struct{}), done: make(chan struct{}),
		setPaths: make(chan []string, 1), snapshots: make(map[string]fileSnapshot),
	}
	for _, path := range initialPaths {
		w.snapshots[path] = snapshotFile(path)
	}
	return w
}

// start begins polling. Every path's baseline is taken before the loop
// starts, so the state the initial build already used is never reported as
// a spurious first change.
func (w *devWatcher) start() {
	go w.run()
}

func (w *devWatcher) close() {
	close(w.stop)
	<-w.done
}

// setPathsAsync replaces the watched set. Safe to call from any goroutine;
// the watcher's own loop applies it on its next tick. Only the most recent
// call matters -- an in-flight update that has not been picked up yet is
// dropped in favor of the newer one, never queued, since a full replacement
// set makes every earlier one obsolete.
func (w *devWatcher) setPathsAsync(paths []string) {
	for {
		select {
		case w.setPaths <- paths:
			return
		case <-w.stop:
			return
		default:
			select {
			case <-w.setPaths:
			default:
			}
		}
	}
}

func (w *devWatcher) run() {
	defer close(w.done)
	var changedAt time.Time

	ticker := time.NewTicker(devWatchPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case paths := <-w.setPaths:
			w.applyPathSet(paths)
		case <-ticker.C:
			if w.pollChanges() {
				changedAt = time.Now()
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

// applyPathSet replaces the tracked set. A path that continues to be
// tracked keeps its existing snapshot, so it is not spuriously reported as
// changed just for surviving a graph update; a newly tracked path gets a
// fresh baseline for the same reason a brand new watcher does.
func (w *devWatcher) applyPathSet(paths []string) {
	next := make(map[string]fileSnapshot, len(paths))
	for _, path := range paths {
		if existing, tracked := w.snapshots[path]; tracked {
			next[path] = existing
			continue
		}
		next[path] = snapshotFile(path)
	}
	w.snapshots = next
}

// pollChanges re-snapshots every tracked path and reports whether any of
// them differ from its last known state. One changed file is enough to
// trigger the same single debounced rebuild as any other: dev.go's queue is
// already source-graph-wide, not per-file (section 27).
func (w *devWatcher) pollChanges() bool {
	changed := false
	for path, previous := range w.snapshots {
		current := snapshotFile(path)
		if current == previous {
			continue
		}
		w.snapshots[path] = current
		if !current.missing {
			changed = true
		}
	}
	return changed
}

func snapshotFile(path string) fileSnapshot {
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{missing: true}
	}
	return fileSnapshot{modTime: info.ModTime(), size: info.Size()}
}
