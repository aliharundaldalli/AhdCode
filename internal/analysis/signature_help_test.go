package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSignatureHelpInsideSingleParameterCall(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\nresult := square(5)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	help, ok := store.SignatureHelp(path, offsetOf(t, text, "square(5)")+len("square("))
	if !ok {
		t.Fatal("expected signature help inside square(...)")
	}
	if help.Label != "(value: Int) -> Int" {
		t.Fatalf("help.Label = %q", help.Label)
	}
	if len(help.Parameters) != 1 || help.Parameters[0] != "value: Int" {
		t.Fatalf("help.Parameters = %#v", help.Parameters)
	}
	if help.ActiveParameter != 0 {
		t.Fatalf("help.ActiveParameter = %d", help.ActiveParameter)
	}
}

// TestSignatureHelpZeroParameterCallableReportsSafeActiveParameter is a
// regression test: a zero-parameter callable's active-parameter index
// previously computed len(parameters)-1 = -1, an invalid negative index no
// real LSP client expects. It must clamp to 0 instead.
func TestSignatureHelpZeroParameterCallableReportsSafeActiveParameter(t *testing.T) {
	text := "greet: Function := (\n) -> String {\n    return \"hi\"\n}\nresult := greet()\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	help, ok := store.SignatureHelp(path, offsetOf(t, text, "greet()")+len("greet("))
	if !ok {
		t.Fatal("expected signature help inside greet()")
	}
	if len(help.Parameters) != 0 {
		t.Fatalf("help.Parameters = %#v, want empty", help.Parameters)
	}
	if help.ActiveParameter != 0 {
		t.Fatalf("help.ActiveParameter = %d, want 0 (never negative)", help.ActiveParameter)
	}
}

func TestSignatureHelpTracksActiveParameterAcrossMultipleArguments(t *testing.T) {
	text := "add: Function := (\n    x: Int,\n    y: Int,\n    z: Int\n) -> Int {\n    return x + y + z\n}\nresult := add(1, 2, 3)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	callOpen := offsetOf(t, text, "add(1, 2, 3)") + len("add(")

	first, ok := store.SignatureHelp(path, callOpen)
	if !ok || first.ActiveParameter != 0 {
		t.Fatalf("first argument: help = %#v, ok = %v", first, ok)
	}

	second, ok := store.SignatureHelp(path, offsetOf(t, text, "2, 3)"))
	if !ok || second.ActiveParameter != 1 {
		t.Fatalf("second argument: help = %#v, ok = %v", second, ok)
	}

	third, ok := store.SignatureHelp(path, offsetOf(t, text, "3)"))
	if !ok || third.ActiveParameter != 2 {
		t.Fatalf("third argument: help = %#v, ok = %v", third, ok)
	}
}

func TestSignatureHelpWorksInsideAnUnclosedCall(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\nresult := square(\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	help, ok := store.SignatureHelp(path, len(text)-1)
	if !ok {
		t.Fatal("expected signature help while the call is still unclosed")
	}
	if help.Label != "(value: Int) -> Int" || help.ActiveParameter != 0 {
		t.Fatalf("help = %#v", help)
	}
}

func TestSignatureHelpAfterATrailingCommaTargetsTheNextParameter(t *testing.T) {
	text := "add: Function := (\n    x: Int,\n    y: Int\n) -> Int {\n    return x + y\n}\nresult := add(1, \n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	help, ok := store.SignatureHelp(path, len(text)-1)
	if !ok {
		t.Fatal("expected signature help after the trailing comma")
	}
	if help.ActiveParameter != 1 {
		t.Fatalf("help.ActiveParameter = %d, want 1", help.ActiveParameter)
	}
}

func TestSignatureHelpHTTPCookieAndSessions(t *testing.T) {
	text := "bring HTTP\nvalue := HTTP.cookie(\"a\", \"1\")\nstore := HTTP.sessions(\nclient := HTTP.client(\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	cookieHelp, ok := store.SignatureHelp(path, offsetOf(t, text, "HTTP.cookie(\"a\"")+len("HTTP.cookie("))
	if !ok || !strings.Contains(cookieHelp.Label, "Cookie") {
		t.Fatalf("HTTP.cookie signature = %#v, ok = %v", cookieHelp, ok)
	}
	sessionsHelp, ok := store.SignatureHelp(path, offsetOf(t, text, "HTTP.sessions(")+len("HTTP.sessions("))
	if !ok || !strings.Contains(sessionsHelp.Label, "SessionStore") {
		t.Fatalf("HTTP.sessions signature = %#v, ok = %v", sessionsHelp, ok)
	}
	clientHelp, ok := store.SignatureHelp(path, len(text)-1)
	if !ok || !strings.Contains(clientHelp.Label, "Client") {
		t.Fatalf("HTTP.client signature = %#v, ok = %v", clientHelp, ok)
	}
}

func TestSignatureHelpOutsideAnyCallReportsNoResult(t *testing.T) {
	text := "x: Int := 5\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	if _, ok := store.SignatureHelp(path, offsetOf(t, text, "5")); ok {
		t.Fatal("expected no signature help outside any call")
	}
}

func TestSignatureHelpOnUnknownCallReportsNoResult(t *testing.T) {
	text := "result := unknownFunction(1, 2)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	if _, ok := store.SignatureHelp(path, offsetOf(t, text, "1, 2)")); ok {
		t.Fatal("expected no signature help for a call that never resolved a callable")
	}
}
