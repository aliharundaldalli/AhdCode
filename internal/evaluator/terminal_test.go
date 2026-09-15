package evaluator

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func newTerminalTestSession(t *testing.T) (*Session, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var output, errorOutput bytes.Buffer
	session := New(nil, &output, t.TempDir())
	session.ErrorOutput = &errorOutput
	return session, &output, &errorOutput
}

func terminalParts(items ...string) *List {
	list := &List{}
	for _, item := range items {
		list.Items = append(list.Items, item)
	}
	return list
}

func TestTerminalEmitThroughEvaluatorIsExact(t *testing.T) {
	for _, testCase := range []struct {
		arguments []any
		want      string
	}{
		{[]any{terminalParts("Ali", "95", "Passed"), nil, nil}, "Ali 95 Passed\n"},
		{[]any{terminalParts("Ali", "95", "Passed"), " | ", nil}, "Ali | 95 | Passed\n"},
		{[]any{terminalParts("Loading"), nil, ""}, "Loading"},
		{[]any{terminalParts(), nil, nil}, "\n"},
		{[]any{terminalParts(), ", ", ""}, ""},
		{[]any{terminalParts("a", "b"), "", ""}, "ab"},
		{[]any{terminalParts("ç", "🙂"), " · ", "\r\n"}, "ç · 🙂\r\n"},
		{[]any{terminalParts("line\nbreak", ""), "|", "!"}, "line\nbreak|!"},
	} {
		session, output, errorOutput := newTerminalTestSession(t)
		session.terminalBuiltin("emit", testCase.arguments)
		if output.String() != testCase.want || errorOutput.Len() != 0 {
			t.Fatalf("emit %v wrote %q (stderr %q), want %q", testCase.arguments[1:], output.String(), errorOutput.String(), testCase.want)
		}
	}
}

func TestTerminalErrorThroughEvaluatorUsesErrorOutput(t *testing.T) {
	session, output, errorOutput := newTerminalTestSession(t)
	session.terminalBuiltin("error", []any{"Configuration not found", nil})
	session.terminalBuiltin("error", []any{"çğ ✓", ""})
	session.terminalBuiltin("error", []any{"", "!\n"})
	if output.Len() != 0 {
		t.Fatalf("Terminal.error contaminated standard output: %q", output.String())
	}
	if want := "Configuration not found\nçğ ✓!\n"; errorOutput.String() != want {
		t.Fatalf("stderr = %q, want %q", errorOutput.String(), want)
	}
	// An unset ErrorOutput discards the text, like an unset Output.
	quiet := New(nil, nil, t.TempDir())
	quiet.terminalBuiltin("error", []any{"dropped", nil})
}

type failingFlushWriter struct{ bytes.Buffer }

func (*failingFlushWriter) Flush() error { return errors.New("device full") }

func TestTerminalFlushThroughEvaluator(t *testing.T) {
	var underlying bytes.Buffer
	buffered := bufio.NewWriter(&underlying)
	session := New(nil, buffered, t.TempDir())
	session.ErrorOutput = &bytes.Buffer{}
	session.terminalBuiltin("emit", []any{terminalParts("pending"), nil, nil})
	if underlying.Len() != 0 {
		t.Fatalf("buffered output reached the writer before flush: %q", underlying.String())
	}
	session.terminalBuiltin("flush", nil)
	session.terminalBuiltin("flush", nil)
	if underlying.String() != "pending\n" {
		t.Fatalf("after flush the writer holds %q", underlying.String())
	}
	// Terminal.error flushes pending output first, keeping program order.
	session.terminalBuiltin("emit", []any{terminalParts("before"), nil, nil})
	session.terminalBuiltin("error", []any{"problem", nil})
	if underlying.String() != "pending\nbefore\n" {
		t.Fatalf("Terminal.error did not flush pending output first: %q", underlying.String())
	}
	unbuffered, _, _ := newTerminalTestSession(t)
	unbuffered.terminalBuiltin("flush", nil)

	failing := New(nil, &failingFlushWriter{}, t.TempDir())
	message := evaluatorRaisedMessage(t, "TerminalError", func() { failing.terminalBuiltin("flush", nil) })
	if message != "standard output could not be flushed" {
		t.Fatalf("flush failure message %q", message)
	}
}

func TestTerminalCapabilitiesWithoutATerminal(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	buffer, _, _ := newTerminalTestSession(t)
	pipe := New(nil, writer, t.TempDir())
	for name, session := range map[string]*Session{"buffer": buffer, "pipe": pipe} {
		if interactive := session.terminalBuiltin("isInteractive", nil); interactive != false {
			t.Fatalf("%s: isInteractive = %#v", name, interactive)
		}
		if width := session.terminalBuiltin("width", nil); width != nil {
			t.Fatalf("%s: width = %#v, want null", name, width)
		}
		if height := session.terminalBuiltin("height", nil); height != nil {
			t.Fatalf("%s: height = %#v, want null", name, height)
		}
		if color := session.terminalBuiltin("supportsColor", nil); color != false {
			t.Fatalf("%s: supportsColor = %#v", name, color)
		}
		if styled := session.terminalBuiltin("style", []any{"Başarılı", "green", nil, true, nil}); styled != "Başarılı" {
			t.Fatalf("%s: style without color = %#v", name, styled)
		}
	}
}

func TestTerminalStyleRejectsUnknownNamesThroughEvaluator(t *testing.T) {
	session, _, _ := newTerminalTestSession(t)
	message := evaluatorRaisedMessage(t, "TerminalError", func() {
		session.terminalBuiltin("style", []any{"x", "purple", nil, nil, nil})
	})
	if !strings.HasPrefix(message, `unsupported foreground color "purple"; use default`) {
		t.Fatalf("foreground message %q", message)
	}
	message = evaluatorRaisedMessage(t, "TerminalError", func() {
		session.terminalBuiltin("style", []any{"x", nil, "Blue", nil, nil})
	})
	if !strings.HasPrefix(message, `unsupported background color "Blue"; use default`) {
		t.Fatalf("background message %q", message)
	}
}

func TestTerminalPrettyRenderThroughEvaluator(t *testing.T) {
	session, _, _ := newTerminalTestSession(t)
	grades := &Pair{Keys: []any{"Ali", "Ayşe"}, Values: map[any]any{
		"Ali":  &List{Items: []any{int64(90), int64(95), int64(100)}},
		"Ayşe": &List{Items: []any{int64(85), int64(91), int64(97)}},
	}}
	want := "{\n    \"Ali\": [\n        90,\n        95,\n        100\n    ],\n    \"Ayşe\": [\n        85,\n        91,\n        97\n    ]\n}"
	if got := session.prettyRender(grades, "", map[visit]bool{}); got != want {
		t.Fatalf("grades =\n%s\nwant\n%s", got, want)
	}
	for _, testCase := range []struct {
		value any
		want  string
	}{
		{&List{}, "[]"},
		{&Pair{Values: map[any]any{}}, "{}"},
		{(*List)(nil), "null"},
		{&List{Items: []any{"a\"b", "line\nbreak"}}, "[\n    \"a\\\"b\",\n    \"line\\nbreak\"\n]"},
		{&List{Items: []any{1.5, nil}}, "[\n    1.5,\n    null\n]"},
		{&List{Items: []any{&List{Items: []any{"x"}}, &List{}}}, "[\n    [\n        \"x\"\n    ],\n    []\n]"},
		{&Pair{Keys: []any{int64(1), int64(2)}, Values: map[any]any{int64(1): true, int64(2): false}}, "{\n    1: true,\n    2: false\n}"},
	} {
		if got := session.prettyRender(testCase.value, "", map[visit]bool{}); got != testCase.want {
			t.Fatalf("pretty %#v = %q, want %q", testCase.value, got, testCase.want)
		}
	}
	cyclic := &List{}
	cyclic.Items = append(cyclic.Items, cyclic)
	if got := session.prettyRender(cyclic, "", map[visit]bool{}); got != "[\n    [...]\n]" {
		t.Fatalf("a List containing itself = %q", got)
	}
}
