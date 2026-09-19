package ahdruntime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// ahdHelperLink is the program's side of one bundled window helper process
// (ahdgraphics for a Canvas, ahdgui for a Window). The helper speaks bounded
// JSON lines over its standard input and output: every request carries an id
// and gets exactly one response echoing it, and the helper may also write
// event lines ({"event": ...}) at any time.
//
// One goroutine reads the helper's output and routes each line: responses to
// the request that is waiting for one, events to a bounded, ordered queue.
// Event callbacks run on the program's own goroutine, which pops the queue
// while it waits; a callback that sends a request gets its response through
// the same reader, so a callback can never deadlock the protocol. The reader
// never blocks on the queue, and it reaps the process when the output ends,
// so no helper is left as a zombie.
type ahdHelperLink struct {
	process *exec.Cmd
	input   io.WriteCloser

	responses chan []byte
	events    *ahdEventQueue
	stop      chan struct{}
	exited    chan struct{}
	nextID    int64
	maxLine   int
	stopOnce  sync.Once
}

// ahdEventQueueLimit bounds the events waiting for a program's callbacks. A
// window cannot produce more than a few events per frame, so reaching this
// means the program stopped handling events; the queue then reports an
// overflow instead of growing without bound.
const ahdEventQueueLimit = 1024

type ahdEventQueue struct {
	mu       sync.Mutex
	items    []ahdQueuedEvent
	closed   bool
	overflow bool
	notify   chan struct{}
}

type ahdQueuedEvent struct {
	line   []byte
	closed bool
}

func newAhdEventQueue() *ahdEventQueue {
	return &ahdEventQueue{notify: make(chan struct{}, 1)}
}

// push appends one event line. The closed notice is always recorded, in its
// place in the order; other events past the limit set the overflow flag.
func (q *ahdEventQueue) push(line []byte, closed bool) {
	q.mu.Lock()
	switch {
	case closed:
		q.closed = true
		q.items = append(q.items, ahdQueuedEvent{line: line, closed: true})
	case len(q.items) >= ahdEventQueueLimit:
		q.overflow = true
	default:
		q.items = append(q.items, ahdQueuedEvent{line: line})
	}
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

// ahdEventKind is what next found.
type ahdEventKind int

const (
	ahdEventLine     ahdEventKind = iota // an event line
	ahdEventClosed                       // the window is gone
	ahdEventOverflow                     // events were dropped
	ahdEventExited                       // the helper process is gone
)

// next blocks until an event, the closed notice, an overflow, or the end of
// the helper. It never holds the lock while blocking. The closed notice comes
// with its own line, which may carry final readings.
func (q *ahdEventQueue) next(exited <-chan struct{}) ([]byte, ahdEventKind) {
	for {
		q.mu.Lock()
		if q.overflow {
			q.mu.Unlock()
			return nil, ahdEventOverflow
		}
		if len(q.items) > 0 {
			item := q.items[0]
			q.items = q.items[1:]
			q.mu.Unlock()
			if item.closed {
				return item.line, ahdEventClosed
			}
			return item.line, ahdEventLine
		}
		q.mu.Unlock()
		select {
		case <-q.notify:
		case <-exited:
			q.mu.Lock()
			pending := len(q.items) > 0 || q.overflow
			q.mu.Unlock()
			if !pending {
				return nil, ahdEventExited
			}
		}
	}
}

// closedPending reports that the window is gone even if earlier events are
// still queued.
func (q *ahdEventQueue) closedPending() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.closed
}

// ahdStartHelper starts one helper process and its output reader.
func ahdStartHelper(path string, maxLine int) (*ahdHelperLink, error) {
	// exec.Command searches PATH for a bare name such as "ahdgui", even when a
	// file of that name was found in the working directory; an absolute path
	// is started exactly as found.
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	process := exec.Command(path)
	input, err := process.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := process.StdoutPipe()
	if err != nil {
		return nil, err
	}
	// The helper's standard error is never part of the protocol, and
	// diagnostics from the window library are not program output.
	process.Stderr = nil
	if err := process.Start(); err != nil {
		return nil, err
	}
	link := &ahdHelperLink{process: process, input: input, responses: make(chan []byte),
		events: newAhdEventQueue(), stop: make(chan struct{}), exited: make(chan struct{}), maxLine: maxLine}
	go link.read(output)
	return link, nil
}

var (
	ahdEventPrefix  = []byte(`{"event":`)
	ahdClosedPrefix = []byte(`{"event":"closed"`)
)

func (l *ahdHelperLink) read(output io.Reader) {
	reader := bufio.NewReaderSize(output, 4096)
	for {
		var line []byte
		var err error
		for {
			var chunk []byte
			var isPrefix bool
			chunk, isPrefix, err = reader.ReadLine()
			line = append(line, chunk...)
			if err != nil || !isPrefix || len(line) > l.maxLine {
				break
			}
		}
		if err == nil && len(line) > l.maxLine {
			err = errors.New("line too large")
		}
		if err != nil {
			// Draining the rest lets the helper exit instead of blocking on a
			// full pipe, and then the process is reaped.
			_, _ = io.Copy(io.Discard, reader)
			_ = l.process.Wait()
			close(l.exited)
			return
		}
		if bytes.HasPrefix(line, ahdEventPrefix) {
			l.events.push(line, bytes.HasPrefix(line, ahdClosedPrefix))
			continue
		}
		// A response that arrives after its request gave up is dropped once
		// the link stops, so this goroutine always reaches the end of the
		// stream and reaps the process.
		select {
		case l.responses <- line:
		case <-l.stop:
		}
	}
}

// call sends one request and returns its response line. The caller
// serializes calls on one link. The request is any JSON object value; call
// adds the id.
func (l *ahdHelperLink) call(value any, timeout time.Duration) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) < 2 || encoded[0] != '{' {
		return nil, errors.New("could not encode a helper request")
	}
	l.nextID++
	id := l.nextID
	separator := ","
	if len(encoded) == 2 {
		separator = ""
	}
	line := append([]byte(`{"id":`+strconv.FormatInt(id, 10)+separator), encoded[1:]...)
	if _, err := l.input.Write(append(line, '\n')); err != nil {
		return nil, errAhdHelperStopped
	}
	var reply []byte
	received := true
	select {
	case reply, received = <-l.responses:
	case <-l.exited:
		received = false
	case <-ahdHelperTimer(timeout):
		return nil, errAhdHelperSilent
	}
	if !received {
		return nil, errAhdHelperStopped
	}
	var envelope struct {
		ID int64 `json:"id"`
	}
	if json.Unmarshal(reply, &envelope) != nil || envelope.ID != id {
		return nil, errAhdHelperInvalid
	}
	return reply, nil
}

var (
	errAhdHelperStopped = errors.New("stopped")
	errAhdHelperSilent  = errors.New("silent")
	errAhdHelperInvalid = errors.New("invalid")
)

// ahdHelperTimer is a timeout channel; a zero timeout never fires.
func ahdHelperTimer(timeout time.Duration) <-chan time.Time {
	if timeout <= 0 {
		return nil
	}
	return time.After(timeout)
}

// shutdown ends the helper: its input is closed, which a helper treats as the
// program ending, and it is killed if it does not exit in time.
func (l *ahdHelperLink) shutdown(grace time.Duration) {
	l.stopOnce.Do(func() {
		close(l.stop)
		_ = l.input.Close()
		select {
		case <-l.exited:
		case <-time.After(grace):
			_ = l.process.Process.Kill()
			<-l.exited
		}
	})
}
