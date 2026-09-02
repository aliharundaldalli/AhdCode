// Package lsp is the AhdCode language server's protocol layer: JSON-RPC
// transport, LSP lifecycle, document synchronization, and the conversion
// between LSP wire types and internal/analysis facts. It has no language
// semantics of its own -- every diagnostic and hover response is a
// translation of exactly what internal/analysis already computed from the
// real AhdCode compiler frontend.
package lsp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"ahdcode/internal/analysis"
	"ahdcode/internal/diagnostics"
)

// Server drives one LSP session over a single reader/writer pair. It is not
// safe for concurrent use: the LSP specification allows request/notification
// interleaving but does not require a server to process them concurrently,
// and v0.2.0 deliberately keeps the message loop single-threaded and
// synchronous (see the module doc comment).
type Server struct {
	store   *analysis.Store
	log     io.Writer
	out     *writer
	closing bool
	// published tracks, for each analyzed entry document, the set of URIs a
	// publishDiagnostics notification was most recently sent for -- so the
	// next analysis of that same entry can clear any URI that no longer has
	// diagnostics (e.g. a deleted `bring` or a fixed dependency), including
	// clearing all of them when the entry document itself closes.
	published map[string]map[string]bool
	// clientURI remembers, for every canonical path a client has ever named
	// in a didOpen/didChange, the exact URI string it used. internal/module
	// resolves a canonical path with symlinks followed (so a document keeps
	// one identity no matter how it's imported), but a client's own URI
	// almost never has symlinks resolved -- on macOS, for example, a path
	// under /var is silently canonicalized to /private/var. Publishing a
	// diagnostic under the canonicalized URI instead of the one the client
	// actually used would create a phantom document the editor never
	// recognizes as the one it has open. A path with no known client URI
	// (an imported module the client never opened directly) falls back to
	// PathToURI of its canonical path, which is the best available identity
	// for it.
	clientURI map[string]string
}

// NewServer creates a language server. log receives internal diagnostic
// text (never protocol frames); pass io.Discard for silence. It must never
// be the same writer used for the protocol stream.
func NewServer(log io.Writer) *Server {
	if log == nil {
		log = io.Discard
	}
	return &Server{
		store:     analysis.NewStore(),
		log:       log,
		published: make(map[string]map[string]bool),
		clientURI: make(map[string]string),
	}
}

// Run reads framed JSON-RPC messages from in and writes responses/
// notifications to out until the client sends `exit`, the input stream
// closes cleanly, or an unrecoverable transport error occurs. out receives
// LSP protocol frames only -- Run never writes a banner, a log line, or any
// other human-readable text to it.
func (server *Server) Run(in io.Reader, out io.Writer) error {
	server.out = newWriter(out)
	source := newReader(in)
	for {
		next, err := source.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			var frameErr *frameParseError
			if errors.As(err, &frameErr) {
				server.sendErrorResponse(nil, errCodeParseError, frameErr.Error())
				continue
			}
			return err
		}
		if exit := server.dispatch(next); exit {
			return nil
		}
	}
}

func (server *Server) dispatch(m message) (exit bool) {
	if server.closing && m.Method != "exit" {
		if !m.isNotification() {
			server.sendErrorResponse(m.ID, errCodeInvalidRequest, "the server has received shutdown; only exit is valid now")
		}
		return false
	}
	switch m.Method {
	case "initialize":
		server.handleInitialize(m)
	case "initialized":
		// No response required; nothing to do yet.
	case "shutdown":
		server.closing = true
		server.reply(m.ID, json.RawMessage("null"))
	case "exit":
		return true
	case "textDocument/didOpen":
		server.handleDidOpen(m)
	case "textDocument/didChange":
		server.handleDidChange(m)
	case "textDocument/didClose":
		server.handleDidClose(m)
	case "textDocument/hover":
		server.handleHover(m)
	case "textDocument/definition":
		server.handleDefinition(m)
	case "textDocument/documentSymbol":
		server.handleDocumentSymbol(m)
	case "textDocument/signatureHelp":
		server.handleSignatureHelp(m)
	case "textDocument/references":
		server.handleReferences(m)
	case "textDocument/completion":
		server.handleCompletion(m)
	default:
		if !m.isNotification() {
			server.sendErrorResponse(m.ID, errCodeMethodNotFound, fmt.Sprintf("method not found: %s", m.Method))
		}
		// An unrecognized notification is silently ignored, per the LSP
		// convention that servers must tolerate notifications they don't
		// implement (e.g. workspace/didChangeConfiguration).
	}
	return false
}

func (server *Server) logf(format string, args ...any) {
	fmt.Fprintf(server.log, format+"\n", args...)
}

func (server *Server) reply(id json.RawMessage, result json.RawMessage) {
	if len(id) == 0 {
		return // never respond to a notification
	}
	if err := server.out.send(message{ID: id, Result: result}); err != nil {
		server.logf("lsp: failed to send response: %v", err)
	}
}

func (server *Server) sendErrorResponse(id json.RawMessage, code int, text string) {
	if err := server.out.send(message{ID: id, Error: &responseError{Code: code, Message: text}}); err != nil {
		server.logf("lsp: failed to send error response: %v", err)
	}
}

func (server *Server) notify(method string, params any) {
	encoded, err := json.Marshal(params)
	if err != nil {
		server.logf("lsp: failed to encode %s params: %v", method, err)
		return
	}
	if err := server.out.send(message{Method: method, Params: encoded}); err != nil {
		server.logf("lsp: failed to send %s notification: %v", method, err)
	}
}

// replyResult marshals a handler's result value and sends it as the
// response for id, or sends an internal-error response if it cannot be
// encoded (a host defect, not a protocol or compiler failure).
func (server *Server) replyResult(id json.RawMessage, result any) {
	if result == nil {
		server.reply(id, json.RawMessage("null"))
		return
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		server.sendErrorResponse(id, errCodeInternalError, "failed to encode response: "+err.Error())
		return
	}
	server.reply(id, encoded)
}

// diagnosticsFor converts one document's compiler diagnostics into LSP
// diagnostics and publishes them, then clears any URI that previously had
// diagnostics attributed to this same entry's analysis but no longer does.
func (server *Server) publishDiagnostics(result analysis.Result) {
	previous := server.published[result.EntryPath]
	current := make(map[string]bool, len(result.Diagnostics))
	for path, items := range result.Diagnostics {
		current[path] = true
		server.publishOne(path, result.Text[path], items)
	}
	for path := range previous {
		if !current[path] {
			server.publishOne(path, "", nil)
		}
	}
	server.published[result.EntryPath] = current
}

func (server *Server) publishOne(path, text string, items []diagnostics.Diagnostic) {
	index := newLineIndex(text)
	converted := make([]lspDiagnostic, 0, len(items))
	for _, item := range items {
		converted = append(converted, convertDiagnostic(item, index))
	}
	server.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{
		URI:         server.uriFor(path),
		Diagnostics: converted,
	})
}

// uriFor returns the URI a client should see for path: the exact URI string
// the client itself used to open it, if it has one open, or else
// PathToURI's best-available synthesis of its canonical form (a sibling
// module the client never opened directly, for instance).
func (server *Server) uriFor(path string) string {
	if uri, ok := server.clientURI[path]; ok {
		return uri
	}
	return PathToURI(path)
}
