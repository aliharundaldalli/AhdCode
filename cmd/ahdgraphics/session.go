package main

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"sync/atomic"
	"time"
)

// session serves one Canvas after its open request. In window mode the
// protocol runs on its own goroutine while the window owns the main thread;
// headless mode has no window and serves on the main goroutine.
type session struct {
	model    *model
	in       *bufio.Reader
	out      *bufio.Writer
	headless bool

	// closed is true once the Canvas can no longer be drawn on: the user
	// closed its window, wait returned, or close was requested.
	closed atomic.Bool
	// closeRequested asks the window to end itself.
	closeRequested atomic.Bool
	// windowGone is closed when the window no longer exists.
	windowGone chan struct{}
}

const closedMessage = "the Canvas is closed"

// serve answers requests until the runtime closes the pipe or sends close.
func (s *session) serve() {
	for {
		value, err := readRequest(s.in)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			_ = writeResponse(s.out, response{Error: err.Error()})
			if errors.Is(err, errRequestTooLarge) {
				// The rest of that line cannot be told apart from the next
				// request, so the stream is not trusted any further.
				return
			}
			continue
		}
		reply, stop := s.handle(value)
		if writeResponse(s.out, reply) != nil || stop {
			return
		}
	}
}

func (s *session) handle(value request) (response, bool) {
	switch value.Op {
	case "status":
		open := !s.closed.Load()
		return response{OK: true, Open: &open}, false
	case "close":
		s.requestClose()
		return response{OK: true}, true
	case "wait":
		if !s.headless {
			<-s.windowGone
		}
		s.closed.Store(true)
		return response{OK: true}, false
	case "save":
		if s.closed.Load() {
			return response{Error: closedMessage, Closed: true}, false
		}
		if err := s.save(value.Path); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "clear", "line", "circle", "rectangle":
		if s.closed.Load() {
			return response{Error: closedMessage, Closed: true}, false
		}
		if err := s.model.apply(value); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	}
	return response{Error: "unknown request " + value.Op}, false
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

func (s *session) save(path string) error {
	format, err := exportFormat(path)
	if err != nil {
		return err
	}
	snap := s.model.snapshot()
	var data []byte
	if format == "png" {
		if data, err = encodePNG(snap); err != nil {
			return errors.New("could not encode the PNG image")
		}
	} else {
		data = encodeSVG(snap)
	}
	if err := os.WriteFile(path, data, 0o666); err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			return errors.New("could not write the file: " + pathError.Err.Error())
		}
		return errors.New("could not write the file")
	}
	return nil
}
