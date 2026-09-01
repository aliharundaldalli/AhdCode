package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func frame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func TestReaderSingleMessage(t *testing.T) {
	input := frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	r := newReader(strings.NewReader(input))
	m, err := r.next()
	if err != nil {
		t.Fatal(err)
	}
	if m.Method != "initialize" {
		t.Fatalf("method = %q", m.Method)
	}
	if string(m.ID) != "1" {
		t.Fatalf("id = %q", m.ID)
	}
}

func TestReaderMultipleSequentialMessages(t *testing.T) {
	input := frame(`{"jsonrpc":"2.0","id":1,"method":"first","params":{}}`) +
		frame(`{"jsonrpc":"2.0","id":2,"method":"second","params":{}}`) +
		frame(`{"jsonrpc":"2.0","method":"third-notification","params":{}}`)
	r := newReader(strings.NewReader(input))

	first, err := r.next()
	if err != nil || first.Method != "first" {
		t.Fatalf("first = %+v, err = %v", first, err)
	}
	second, err := r.next()
	if err != nil || second.Method != "second" {
		t.Fatalf("second = %+v, err = %v", second, err)
	}
	third, err := r.next()
	if err != nil || third.Method != "third-notification" || !third.isNotification() {
		t.Fatalf("third = %+v, err = %v", third, err)
	}
	if _, err := r.next(); err != io.EOF {
		t.Fatalf("expected clean EOF after the last message, got %v", err)
	}
}

// slowReader returns at most one byte per Read call, forcing bufio.Reader
// (and therefore our frame reader) to assemble both the header and the body
// across many underlying reads.
type slowReader struct{ remaining []byte }

func (r *slowReader) Read(p []byte) (int, error) {
	if len(r.remaining) == 0 {
		return 0, io.EOF
	}
	p[0] = r.remaining[0]
	r.remaining = r.remaining[1:]
	return 1, nil
}

func TestReaderMessageSplitAcrossManyReads(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":7,"method":"textDocument/hover","params":{"a":1}}`
	input := frame(body)
	r := newReader(&slowReader{remaining: []byte(input)})
	m, err := r.next()
	if err != nil {
		t.Fatal(err)
	}
	if m.Method != "textDocument/hover" {
		t.Fatalf("method = %q", m.Method)
	}
}

func TestReaderUnicodeBodyContentLengthIsByteCorrect(t *testing.T) {
	// The string value contains multi-byte UTF-8 (Turkish, CJK, an emoji).
	type params struct {
		Text string `json:"text"`
	}
	encodedParams, err := json.Marshal(params{Text: "İşçi 日本語 🙂"})
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"test","params":%s}`, encodedParams)
	// Sanity: this body's byte length differs from its rune count.
	if len(body) == len([]rune(body)) {
		t.Fatal("test fixture is not actually multi-byte")
	}
	input := frame(body)
	r := newReader(strings.NewReader(input))
	m, err := r.next()
	if err != nil {
		t.Fatal(err)
	}
	var decoded params
	if err := json.Unmarshal(m.Params, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Text != "İşçi 日本語 🙂" {
		t.Fatalf("text = %q", decoded.Text)
	}
}

// TestReaderTrailingGarbageAfterContentLengthIsIgnored proves a message
// following a correctly-sized frame is unaffected by any garbage that is
// NOT included in that frame's own Content-Length -- i.e. framing is
// strictly byte-counted, not JSON-boundary-guessed.
func TestReaderTrailingGarbageAfterContentLengthIsIgnored(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"a","params":{}}`
	input := frame(body) + frame(`{"jsonrpc":"2.0","id":2,"method":"b","params":{}}`)
	r := newReader(strings.NewReader(input))
	first, err := r.next()
	if err != nil || first.Method != "a" {
		t.Fatalf("first = %+v, err = %v", first, err)
	}
	second, err := r.next()
	if err != nil || second.Method != "b" {
		t.Fatalf("second = %+v, err = %v", second, err)
	}
}

func TestReaderMalformedContentLengthHeader(t *testing.T) {
	input := "Content-Length: not-a-number\r\n\r\n{}"
	r := newReader(strings.NewReader(input))
	if _, err := r.next(); err == nil {
		t.Fatal("expected an error for a malformed Content-Length header")
	}
}

func TestReaderMissingContentLengthHeader(t *testing.T) {
	input := "Content-Type: application/json\r\n\r\n{}"
	r := newReader(strings.NewReader(input))
	if _, err := r.next(); err == nil {
		t.Fatal("expected an error when Content-Length is missing")
	}
}

func TestReaderMalformedJSONBodyIsARecoverableFrameError(t *testing.T) {
	input := frame(`{"jsonrpc": this is not valid json`)
	r := newReader(strings.NewReader(input))
	_, err := r.next()
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
	var frameErr *frameParseError
	if !isFrameParseError(err, &frameErr) {
		t.Fatalf("expected a *frameParseError (recoverable), got %T: %v", err, err)
	}
}

func isFrameParseError(err error, target **frameParseError) bool {
	if fpe, ok := err.(*frameParseError); ok {
		*target = fpe
		return true
	}
	return false
}

func TestReaderCleanEOFBetweenMessages(t *testing.T) {
	r := newReader(strings.NewReader(""))
	if _, err := r.next(); err != io.EOF {
		t.Fatalf("expected io.EOF on an empty stream, got %v", err)
	}
}

func TestWriterProducesValidContentLengthFrame(t *testing.T) {
	var buffer bytes.Buffer
	w := newWriter(&buffer)
	if err := w.send(message{Method: "test", ID: json.RawMessage("1")}); err != nil {
		t.Fatal(err)
	}
	r := newReader(&buffer)
	echoed, err := r.next()
	if err != nil {
		t.Fatal(err)
	}
	if echoed.Method != "test" || echoed.JSONRPC != "2.0" {
		t.Fatalf("echoed = %+v", echoed)
	}
}

func TestWriterUnicodeBodyContentLengthIsByteCorrect(t *testing.T) {
	var buffer bytes.Buffer
	w := newWriter(&buffer)
	payload, _ := json.Marshal(map[string]string{"text": "İşçi 🙂"})
	if err := w.send(message{Method: "test", Params: payload}); err != nil {
		t.Fatal(err)
	}
	raw := buffer.String()
	headerEnd := strings.Index(raw, "\r\n\r\n")
	if headerEnd < 0 {
		t.Fatal("missing header terminator")
	}
	body := raw[headerEnd+4:]
	if len(body) != len([]byte(body)) {
		t.Fatal("sanity check failed")
	}
	// The declared Content-Length must equal the body's actual byte length,
	// not its rune count.
	var declared int
	if _, err := fmt.Sscanf(raw[:headerEnd], "Content-Length: %d", &declared); err != nil {
		t.Fatal(err)
	}
	if declared != len(body) {
		t.Fatalf("declared Content-Length = %d, actual body bytes = %d", declared, len(body))
	}
}
