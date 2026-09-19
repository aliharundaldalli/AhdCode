package main

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// session serves one Window after its open request. In window mode the
// protocol runs on its own goroutine while the window owns the main thread;
// headless mode has no window and serves on the main goroutine.
type session struct {
	model    *model
	in       *bufio.Reader
	out      *bufio.Writer
	headless bool
	openID   int64

	// closed is true once the Window is gone: the user closed it, or the
	// program asked to close it. A closed Window takes no further changes.
	closed         atomic.Bool
	closeRequested atomic.Bool
	windowGone     chan struct{}

	// writeMu serializes the output stream shared by responses and events.
	writeMu    sync.Mutex
	closedSent atomic.Bool

	// The events the program listens for: keys, and clicks on some Buttons.
	listenKey atomic.Bool
	clicksMu  sync.Mutex
	clicks    map[int64]bool

	// script is a headless test Window's input, replayed one event at a
	// time: wait starts it and each next request continues it.
	script []event
	cursor int
}

const closedMessage = "the Window is closed"

func (s *session) send(value any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return writeLine(s.out, value)
}

func (s *session) emit(value event) {
	if s.send(value) != nil {
		s.closeRequested.Store(true)
	}
}

// emitClosed tells the program, once, that the Window is gone, with the
// final TextInput and Checkbox readings.
func (s *session) emitClosed() {
	s.closed.Store(true)
	if s.closedSent.CompareAndSwap(false, true) {
		s.emit(event{Event: "closed", Values: s.model.values()})
	}
}

func (s *session) listensToClick(id int64) bool {
	s.clicksMu.Lock()
	defer s.clicksMu.Unlock()
	return s.clicks[id]
}

// clicked reports a Button click the program listens to.
func (s *session) clicked(id int64) {
	if id != 0 && s.listensToClick(id) {
		s.emit(event{Event: "click", Widget: id})
	}
}

// pressedKey reports a key press when the program listens for keys.
func (s *session) pressedKey(name string) {
	if s.listenKey.Load() {
		s.emit(event{Event: "key", Key: name})
	}
}

// serve answers requests until the runtime closes the pipe or sends close.
func (s *session) serve() {
	for {
		parsed, err := readRequest(s.in)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			_ = s.send(response{Error: err.Error()})
			if errors.Is(err, errRequestTooLarge) {
				return
			}
			continue
		}
		reply, stop := s.handle(parsed)
		reply.ID = parsed.ID
		if s.send(reply) != nil || stop {
			return
		}
		if s.headless && (parsed.Op == "wait" || parsed.Op == "next") {
			s.advance()
		}
	}
}

// advance replays the headless script up to and including its next event.
// Typing and toggling change the model silently; a click on a Button nobody
// listens to, or a key when nobody listens for keys, produces nothing. When
// the script is exhausted the Window reports itself closed.
func (s *session) advance() {
	for s.cursor < len(s.script) {
		step := s.script[s.cursor]
		s.cursor++
		switch step.Event {
		case "type":
			_ = s.model.typeInto(step.Widget, step.Text)
		case "toggle":
			_ = s.model.toggle(step.Widget)
		case "click":
			if s.listensToClick(step.Widget) {
				s.clicked(step.Widget)
				return
			}
		case "key":
			if s.listenKey.Load() {
				s.pressedKey(step.Key)
				return
			}
		}
	}
	s.emitClosed()
}

func (s *session) handle(r request) (response, bool) {
	switch r.Op {
	case "status":
		open := !s.closed.Load()
		return response{OK: true, Open: &open}, false
	case "close":
		s.requestClose()
		return response{OK: true, Values: s.model.values()}, true
	case "wait", "next":
		// wait never blocks the protocol: the program keeps sending requests
		// from its callbacks and learns that the Window is gone from the
		// closed event.
		return response{OK: true}, false
	case "get":
		text, checked, err := s.model.read(r.Widget)
		if err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true, Text: &text, Checked: &checked}, false
	}
	if s.closed.Load() {
		return response{Error: closedMessage, Closed: true}, false
	}
	switch r.Op {
	case "add":
		checked := r.Checked != nil && *r.Checked
		id, err := s.model.add(r.Parent, r.Kind, r.Text, r.Placeholder, checked, r.Spacing, r.Padding)
		if err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true, Widget: id}, false
	case "set":
		var err error
		switch {
		case r.Kind == "text":
			err = s.model.setText(r.Widget, r.Text)
		case r.Kind == "checked" && r.Checked != nil:
			err = s.model.setChecked(r.Widget, *r.Checked)
		default:
			err = errors.New("set needs text or checked")
		}
		if err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "title":
		if !validText(r.Text, maxTitleRunes) {
			return response{Error: "invalid window title"}, false
		}
		s.model.setTitle(r.Text)
		return response{OK: true}, false
	case "listen":
		switch r.Kind {
		case "key":
			s.listenKey.Store(true)
		case "click":
			if _, _, err := s.model.read(r.Widget); err != nil {
				return response{Error: err.Error()}, false
			}
			s.clicksMu.Lock()
			s.clicks[r.Widget] = true
			s.clicksMu.Unlock()
		default:
			return response{Error: "unknown event kind " + r.Kind}, false
		}
		return response{OK: true}, false
	}
	return response{Error: "unknown request " + r.Op}, false
}

// requestClose ends the window, if any, and waits briefly for it to go.
func (s *session) requestClose() {
	s.closed.Store(true)
	if s.headless {
		return
	}
	s.closeRequested.Store(true)
	select {
	case <-s.windowGone:
	case <-time.After(5 * time.Second):
	}
}
