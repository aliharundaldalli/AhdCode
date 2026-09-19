// Command ahdgui is the bundled window helper for AhdCode's GUI standard
// module. Each open Window runs one ahdgui process, so several Windows are
// several independent processes and the window library never manages more
// than one window per process.
//
// The helper is a separate executable and a separate Go module on purpose:
// the window library stays out of the compiler, out of the runtime embedded in
// every compiled AhdCode program, and out of the main module's dependency
// graph. It speaks only the line-based JSON protocol in protocol.go over
// standard input and output. It runs no shell command, evaluates nothing,
// reads no file or environment file, makes no network connection, and logs
// nothing it is given: TextInput contents are the program's data. The same
// executable also shows GUI's file and message dialogs (dialog.go), one
// dialog per process.
package main

import (
	"bufio"
	"errors"
	"io"
	"os"
)

func main() {
	in := bufio.NewReaderSize(os.Stdin, maxRequestBytes+1)
	out := bufio.NewWriter(os.Stdout)

	first, err := readRequest(in)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			_ = writeLine(out, response{Error: err.Error()})
		}
		os.Exit(1)
	}
	if first.Op == "dialog" {
		reply := runDialog(first)
		reply.ID = first.ID
		_ = writeLine(out, reply)
		if !reply.OK {
			os.Exit(1)
		}
		return
	}
	if err := validateOpen(first); err != nil {
		_ = writeLine(out, response{ID: first.ID, Error: err.Error()})
		os.Exit(1)
	}
	s := newSession(first, in, out)
	if s.headless {
		// No window: widgets are still created, changed, and read, and the
		// test script stands in for the user.
		close(s.windowGone)
		if s.send(response{ID: first.ID, OK: true}) != nil {
			os.Exit(1)
		}
		s.serve()
		return
	}
	if err := runWindow(s); err != nil {
		os.Exit(1)
	}
}

func newSession(first request, in *bufio.Reader, out *bufio.Writer) *session {
	return &session{
		model:      newModel(first.Title, first.Width, first.Height, measureText),
		in:         in,
		out:        out,
		headless:   first.Headless,
		openID:     first.ID,
		windowGone: make(chan struct{}),
		clicks:     map[int64]bool{},
		changes:    map[int64]bool{},
		script:     first.Script,
	}
}
