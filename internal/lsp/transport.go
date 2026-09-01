package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// message is one JSON-RPC 2.0 envelope, covering requests, responses, and
// notifications alike. ID is nil for a notification. json.RawMessage is
// used for ID, Params, and Result/Error so a message can be decoded without
// knowing its method's parameter shape up front, and so a request ID can be
// either a JSON string or a JSON number, exactly as the spec allows.
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard JSON-RPC error codes used by this server.
const (
	errCodeParseError     = -32700
	errCodeInvalidRequest = -32600
	errCodeMethodNotFound = -32601
	errCodeInvalidParams  = -32602
	errCodeInternalError  = -32603
)

// isNotification reports whether a message carries no ID, meaning the
// sender expects no response.
func (m message) isNotification() bool { return len(m.ID) == 0 }

// reader reads a stream of Content-Length-framed JSON-RPC messages from an
// io.Reader, exactly as the LSP specification defines: one or more
// "Header: value\r\n" lines, a blank "\r\n" line, then exactly
// Content-Length bytes of UTF-8 JSON. Content-Length counts bytes, not
// runes. bufio.Reader already handles a message body arriving split across
// multiple underlying Read calls.
type reader struct {
	source *bufio.Reader
}

func newReader(source io.Reader) *reader {
	return &reader{source: bufio.NewReader(source)}
}

// next reads one framed message. It returns io.EOF (unwrapped) once the
// stream ends cleanly between messages.
func (r *reader) next() (message, error) {
	contentLength := -1
	for {
		line, err := r.source.ReadString('\n')
		if err != nil {
			if err == io.EOF && line == "" {
				return message{}, io.EOF
			}
			return message{}, fmt.Errorf("reading header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return message{}, fmt.Errorf("malformed header line %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, parseErr := strconv.Atoi(strings.TrimSpace(value))
			if parseErr != nil || parsed < 0 {
				return message{}, fmt.Errorf("malformed Content-Length header %q", value)
			}
			contentLength = parsed
		}
		// Any other header (e.g. Content-Type) is accepted and ignored.
	}
	if contentLength < 0 {
		return message{}, fmt.Errorf("message frame is missing Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r.source, body); err != nil {
		return message{}, fmt.Errorf("reading message body: %w", err)
	}
	var decoded message
	if err := json.Unmarshal(body, &decoded); err != nil {
		return message{}, &frameParseError{cause: err}
	}
	return decoded, nil
}

// frameParseError marks a message whose Content-Length framing was read
// successfully but whose body was not valid JSON -- distinguishing "this one
// message is malformed" (recoverable: reply with a parse-error response and
// keep the connection open) from a genuine transport/IO failure.
type frameParseError struct{ cause error }

func (e *frameParseError) Error() string { return "invalid JSON-RPC message body: " + e.cause.Error() }
func (e *frameParseError) Unwrap() error { return e.cause }

// writer sends Content-Length-framed JSON-RPC messages to an io.Writer.
// Every write is one atomic frame; writer is not safe for concurrent use
// from multiple goroutines without external synchronization (the server's
// message loop is single-threaded by design, so none is needed here).
type writer struct {
	sink io.Writer
}

func newWriter(sink io.Writer) *writer {
	return &writer{sink: sink}
}

func (w *writer) send(m message) error {
	m.JSONRPC = "2.0"
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("encoding message: %w", err)
	}
	if _, err := fmt.Fprintf(w.sink, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.sink.Write(body)
	return err
}
