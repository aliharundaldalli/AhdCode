package ahdruntime

import (
	"strings"
	"testing"
	"unicode"
)

func TestIdentityIDFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 64; i++ {
		id := AhdIdentityID(AhdClassIdentityError)
		if len(id) != 22 {
			t.Fatalf("Identity.id length = %d; want 22", len(id))
		}
		for _, r := range id {
			if unicode.IsSpace(r) || r == '/' || r == '\\' || r == '+' || r == '=' {
				t.Fatalf("Identity.id %q contains a forbidden character %q", id, r)
			}
			if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
				t.Fatalf("Identity.id %q is not URL-safe: %q", id, r)
			}
		}
		if strings.ContainsAny(id, " \t\n") {
			t.Fatalf("Identity.id contains whitespace: %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate Identity.id in the sample: %q", id)
		}
		seen[id] = true
	}
}
