package golang

import (
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// The UUID runtime is standard library only, so every generated program
// carries it without requiring a vendored dependency tree.
func TestUUIDRuntimeIsStandardLibraryOnly(t *testing.T) {
	allowed := map[string]bool{"bytes": true, "crypto/rand": true, "sync": true, "time": true}
	for _, source := range []string{
		"write(\"hi\")\n",
		"bring UUID\nfrom UUID bring UUIDValue\nvalue: UUIDValue := UUID.v7()\nwrite(value.string() + str(value.compare(UUID.parse(value.string()))))\n",
	} {
		program := generate(t, source)
		if program.RequiresMySQL || program.RequiresCodes {
			t.Fatalf("a UUID program requires a vendored tree:\n%s", source)
		}
		seen := 0
		for _, file := range program.Files {
			if file.Name != uuidRuntimeFileName {
				continue
			}
			seen++
			parsed, err := parser.ParseFile(token.NewFileSet(), file.Name, file.Content, 0)
			if err != nil {
				t.Fatalf("UUID runtime is not valid Go: %v", err)
			}
			if parsed.Name.Name != "main" {
				t.Fatalf("UUID runtime package clause not rewritten: %s", parsed.Name.Name)
			}
			for _, spec := range parsed.Imports {
				path, _ := strconv.Unquote(spec.Path.Value)
				if !allowed[path] {
					t.Fatalf("UUID runtime imports %s", path)
				}
			}
		}
		if seen != 1 {
			t.Fatalf("UUID runtime emitted %d times, want 1", seen)
		}
	}
}
