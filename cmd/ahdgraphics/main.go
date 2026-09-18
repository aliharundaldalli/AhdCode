// Command ahdgraphics is the bundled window helper for AhdCode's Graphics
// standard module. Each open Canvas runs one ahdgraphics process, so several
// Canvases are several independent windows and the rendering library never
// has to manage more than one window per process.
//
// The helper is a separate executable and a separate Go module on purpose:
// the rendering and window dependencies stay out of the compiler, out of the
// runtime embedded in every compiled AhdCode program, and out of the main
// module's dependency graph. It speaks only the line-based JSON protocol in
// protocol.go over standard input and output. It runs no shell command,
// evaluates nothing, reads no environment file, and makes no network
// connection; the only file it writes is the path a save request names.
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
			_ = writeResponse(out, response{Error: err.Error()})
		}
		os.Exit(1)
	}
	if err := validateOpen(first); err != nil {
		_ = writeResponse(out, response{Error: err.Error()})
		os.Exit(1)
	}
	s := &session{
		model:      newModel(first.Title, first.Width, first.Height, *first.Background),
		in:         in,
		out:        out,
		headless:   first.Headless,
		windowGone: make(chan struct{}),
	}
	if s.headless {
		// No window: the Canvas still records, clears, and saves drawings,
		// and wait returns at once because there is no window to close.
		close(s.windowGone)
		if writeResponse(out, response{OK: true}) != nil {
			os.Exit(1)
		}
		s.serve()
		return
	}
	if err := runWindow(s); err != nil {
		os.Exit(1)
	}
}
