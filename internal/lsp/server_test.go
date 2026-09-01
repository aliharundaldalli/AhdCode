package lsp

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

// testClient builds a request/notification stream and lets a test replay
// the server's response/notification stream after Server.Run returns.
type testClient struct {
	input bytes.Buffer
}

func (c *testClient) request(id int, method string, params any) {
	encodedParams, _ := json.Marshal(params)
	encodedID, _ := json.Marshal(id)
	body, _ := json.Marshal(message{ID: encodedID, Method: method, Params: encodedParams})
	c.input.WriteString(frame(string(body)))
}

func (c *testClient) notify(method string, params any) {
	encodedParams, _ := json.Marshal(params)
	body, _ := json.Marshal(message{Method: method, Params: encodedParams})
	c.input.WriteString(frame(string(body)))
}

// runServer runs the server to completion against the client's queued input
// and returns every message it sent back, in order.
func runServer(t *testing.T, client *testClient) []message {
	t.Helper()
	server := NewServer(io.Discard)
	var output bytes.Buffer
	if err := server.Run(&client.input, &output); err != nil {
		t.Fatalf("Server.Run: %v", err)
	}
	r := newReader(&output)
	var messages []message
	for {
		m, err := r.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decoding server output: %v", err)
		}
		messages = append(messages, m)
	}
	return messages
}

func findResponse(t *testing.T, messages []message, id int) message {
	t.Helper()
	encodedID, _ := json.Marshal(id)
	for _, m := range messages {
		if string(m.ID) == string(encodedID) && m.Method == "" {
			return m
		}
	}
	t.Fatalf("no response found for id %d among %d messages", id, len(messages))
	return message{}
}

func findNotifications(messages []message, method string) []message {
	var found []message
	for _, m := range messages {
		if m.Method == method {
			found = append(found, m)
		}
	}
	return found
}

func TestInitializeAdvertisesOnlyImplementedCapabilities(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{"processId": nil, "rootUri": nil, "capabilities": map[string]any{}})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 1)
	if response.Error != nil {
		t.Fatalf("initialize returned an error: %+v", response.Error)
	}
	var result initializeResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.Capabilities.TextDocumentSync != textDocumentSyncFull {
		t.Fatalf("textDocumentSync = %d, want %d (Full)", result.Capabilities.TextDocumentSync, textDocumentSyncFull)
	}
	if !result.Capabilities.HoverProvider {
		t.Fatal("expected hoverProvider = true")
	}
	// Confirm the raw JSON never mentions any out-of-scope capability.
	raw := string(response.Result)
	for _, forbidden := range []string{
		"completionProvider", "definitionProvider", "referencesProvider", "renameProvider",
		"documentSymbolProvider", "signatureHelpProvider", "semanticTokensProvider",
		"codeActionProvider", "documentFormattingProvider", "foldingRangeProvider",
		"callHierarchyProvider", "inlayHintProvider",
	} {
		if bytes.Contains([]byte(raw), []byte(forbidden)) {
			t.Fatalf("initialize result advertises out-of-scope capability %q: %s", forbidden, raw)
		}
	}
}

func TestInitializedNotificationIsAccepted(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("initialized", map[string]any{})
	client.notify("exit", nil)
	messages := runServer(t, client)
	// initialized must not produce any response (it's a notification).
	for _, m := range messages {
		if string(m.ID) != "" {
			var idNumber int
			_ = json.Unmarshal(m.ID, &idNumber)
			if idNumber != 1 {
				t.Fatalf("unexpected response with id %s for a notification-only stream", m.ID)
			}
		}
	}
}

func TestShutdownThenExitTerminatesCleanly(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.request(2, "shutdown", nil)
	client.notify("exit", nil)
	messages := runServer(t, client)

	shutdownResponse := findResponse(t, messages, 2)
	if shutdownResponse.Error != nil {
		t.Fatalf("shutdown returned an error: %+v", shutdownResponse.Error)
	}
}

func TestRequestAfterShutdownIsRejected(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.request(2, "shutdown", nil)
	client.request(3, "textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": "file:///nope.ahd"},
		"position":     map[string]any{"line": 0, "character": 0},
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	afterShutdown := findResponse(t, messages, 3)
	if afterShutdown.Error == nil {
		t.Fatal("expected an error response for a request sent after shutdown")
	}
	if afterShutdown.Error.Code != errCodeInvalidRequest {
		t.Fatalf("error code = %d, want %d", afterShutdown.Error.Code, errCodeInvalidRequest)
	}
}

func TestNotificationReceivesNoResponse(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{
		TextDocument: textDocumentItem{URI: "file:///main.ahd", Text: "write(\"hi\")\n"},
	})
	client.notify("exit", nil)
	messages := runServer(t, client)
	for _, m := range messages {
		if len(m.ID) != 0 {
			var idNumber int
			_ = json.Unmarshal(m.ID, &idNumber)
			if idNumber == 0 {
				t.Fatalf("a notification produced a response-shaped message: %+v", m)
			}
		}
	}
}

func TestUnsupportedMethodGetsMethodNotFound(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.request(2, "textDocument/completion", map[string]any{})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error == nil {
		t.Fatal("expected an error response for an unimplemented method")
	}
	if response.Error.Code != errCodeMethodNotFound {
		t.Fatalf("error code = %d, want %d (method not found)", response.Error.Code, errCodeMethodNotFound)
	}
}

func TestUnsupportedNotificationIsSilentlyIgnored(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("workspace/didChangeConfiguration", map[string]any{})
	client.notify("exit", nil)
	// Must not error, panic, or hang.
	runServer(t, client)
}

func TestMultipleFramedMessagesInOneStreamAreAllProcessed(t *testing.T) {
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: "file:///a.ahd", Text: "write(1)\n"}})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: "file:///b.ahd", Text: "write(2)\n"}})
	client.notify("exit", nil)
	messages := runServer(t, client)
	published := findNotifications(messages, "textDocument/publishDiagnostics")
	if len(published) < 2 {
		t.Fatalf("expected at least 2 publishDiagnostics notifications, got %d", len(published))
	}
}

func TestMalformedContentLengthDoesNotCrashTheServer(t *testing.T) {
	var input bytes.Buffer
	input.WriteString("Content-Length: not-a-number\r\n\r\n{}")
	server := NewServer(io.Discard)
	var output bytes.Buffer
	err := server.Run(&input, &output)
	if err == nil {
		t.Fatal("expected Server.Run to report a transport error for a malformed frame")
	}
}

func TestMalformedJSONBodyGetsParseErrorAndSessionContinues(t *testing.T) {
	var input bytes.Buffer
	input.WriteString(frame(`{ this is not valid json`))
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("exit", nil)
	input.Write(client.input.Bytes())

	server := NewServer(io.Discard)
	var output bytes.Buffer
	if err := server.Run(&input, &output); err != nil {
		t.Fatalf("Server.Run: %v", err)
	}
	r := newReader(&output)
	first, err := r.next()
	if err != nil {
		t.Fatal(err)
	}
	if first.Error == nil || first.Error.Code != errCodeParseError {
		t.Fatalf("first message = %+v, want a parse-error response", first)
	}
	second, err := r.next()
	if err != nil {
		t.Fatal(err)
	}
	if second.Error != nil {
		t.Fatalf("initialize after the malformed message failed: %+v", second.Error)
	}
}
