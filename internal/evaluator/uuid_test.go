package evaluator

import (
	"strings"
	"testing"
)

// The evaluator calls the same UUID runtime a compiled program does, so these
// expectations match internal/build's native test.

func TestUUIDThroughEvaluator(t *testing.T) {
	session := newSecurityTestSession()
	parsed := session.uuidBuiltin("parse", []any{"017F22E2-79B0-7CC3-98C4-DC0C0C07398F"})
	if text := session.uuidOperation("UUIDValue.string", parsed, nil); text != "017f22e2-79b0-7cc3-98c4-dc0c0c07398f" {
		t.Fatalf("string = %#v", text)
	}
	if version := session.uuidOperation("UUIDValue.version", parsed, nil); version != int64(7) {
		t.Fatalf("version = %#v", version)
	}
	if zero := session.uuidOperation("UUIDValue.isZero", parsed, nil); zero != false {
		t.Fatalf("isZero = %#v", zero)
	}
	again := session.uuidBuiltin("parse", []any{"017f22e2-79b0-7cc3-98c4-dc0c0c07398f"})
	if same := session.uuidOperation("UUIDValue.equals", parsed, []any{again}); same != true {
		t.Fatalf("equals = %#v", same)
	}
	zero := session.uuidBuiltin("zero", nil)
	if empty := session.uuidOperation("UUIDValue.isZero", zero, nil); empty != true {
		t.Fatalf("zero isZero = %#v", empty)
	}
	if order := session.uuidOperation("UUIDValue.compare", zero, []any{parsed}); order != int64(-1) {
		t.Fatalf("compare = %#v", order)
	}
	random := session.uuidBuiltin("v4", nil)
	if version := session.uuidOperation("UUIDValue.version", random, nil); version != int64(4) {
		t.Fatalf("v4 version = %#v", version)
	}
	first := session.uuidBuiltin("v7", nil)
	second := session.uuidBuiltin("v7", nil)
	if order := session.uuidOperation("UUIDValue.compare", first, []any{second}); order != int64(-1) {
		t.Fatalf("successive v7 values compare %#v", order)
	}
	for text, want := range map[string]bool{
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398f":          true,
		"{017f22e2-79b0-7cc3-98c4-dc0c0c07398f}":        false,
		"urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f": false,
		"017f22e279b07cc398c4dc0c0c07398f":              false,
	} {
		if got := session.uuidBuiltin("isValid", []any{text}); got != want {
			t.Fatalf("isValid(%q) = %#v", text, got)
		}
	}
	message := evaluatorRaisedMessage(t, "UUIDError", func() {
		session.uuidBuiltin("parse", []any{"017f22e2-79b0-7cc3-98c4-dc0c0c07398"})
	})
	if message != "UUID text must be 36 characters in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" {
		t.Fatalf("parse message %q", message)
	}
	if strings.Contains(message, "017f22e2") {
		t.Fatalf("parse message echoes the rejected text: %q", message)
	}
}
