package analysis

import (
	"path/filepath"
	"testing"
)

// Hover and signature help show the ? of a nullable parameter or return, which
// the compiler keeps in the callable's null metadata rather than in its types.
func TestHoverAndSignatureHelpShowNullability(t *testing.T) {
	text := `bring Env

find: Function := (key: String?) -> String? {
    if key == null {
        return null
    }
    return key
}

square: Function := (value: Int) -> Int {
    return value * value
}

found := find("a")
area := square(3)
home := Env.get("HOME")
`
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	for _, testCase := range []struct{ target, want string }{
		{"find(\"a\")", "find: (key: String?) -> String?"},
		{"square(3)", "square: (value: Int) -> Int"},
		{"Env.get", "get: (name: String) -> String?"},
	} {
		offset := offsetOf(t, text, testCase.target) + 1
		if testCase.target == "Env.get" {
			offset = offsetOf(t, text, "Env.get") + len("Env.") + 1
		}
		hover, ok := store.Hover(path, offset)
		if !ok || hover.Text != testCase.want {
			t.Fatalf("hover on %s = %q (%v), want %q", testCase.target, hover.Text, ok, testCase.want)
		}
	}
	help, ok := store.SignatureHelp(path, offsetOf(t, text, "find(\"a\")")+len("find("))
	if !ok || help.Label != "(key: String?) -> String?" || len(help.Parameters) != 1 || help.Parameters[0] != "key: String?" {
		t.Fatalf("signature help for find = %#v (%v)", help, ok)
	}
}
