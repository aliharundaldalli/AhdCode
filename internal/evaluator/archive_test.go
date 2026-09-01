package evaluator

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newArchiveTestSession(t *testing.T) *Session {
	t.Helper()
	return New(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, t.TempDir())
}

func archiveTestPairValue(entries ...string) *Pair {
	pair := &Pair{Values: make(map[any]any)}
	for index := 0; index+1 < len(entries); index += 2 {
		key := entries[index]
		pair.Keys = append(pair.Keys, key)
		pair.Values[key] = entries[index+1]
	}
	return pair
}

// TestArchiveBuiltinSuccessPathWritesRealFile proves the evaluator bridge's
// refactor to call the shared, non-panicking ahdruntime core (instead of the
// panicking AhdArchive* wrappers) left the success path unchanged: a real
// archive is written for real, just as before.
func TestArchiveBuiltinSuccessPathWritesRealFile(t *testing.T) {
	session := newArchiveTestSession(t)
	directory := t.TempDir()
	source := filepath.Join(directory, "a.txt")
	if err := os.WriteFile(source, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "out.zip")
	result := session.archiveBuiltin("zip", []any{output, archiveTestPairValue("a.txt", source)})
	if result != Nothing {
		t.Fatalf("zip result = %#v, want Nothing", result)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("archive not written: %v", err)
	}
}

// TestArchiveBuiltinRaisesCatchableArchiveErrorOnFailure is the regression
// test for the bug: an Archive validation/I/O failure reaching the evaluator
// used to escape as an unrecoverable Go panic (the ahdruntime core called
// AhdRaiseClass, which requires REPL-side error-class registration that
// never happens outside generated native programs). It must now become an
// ordinary catchable AhdCode `raised` value carrying an ArchiveError, exactly
// like every other module's evaluator-raised error.
func TestArchiveBuiltinRaisesCatchableArchiveErrorOnFailure(t *testing.T) {
	session := newArchiveTestSession(t)
	directory := t.TempDir()
	missing := filepath.Join(directory, "does-not-exist.txt")
	output := filepath.Join(directory, "out.zip")

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		session.archiveBuiltin("zip", []any{output, archiveTestPairValue("missing.txt", missing)})
	}()

	failure, ok := recovered.(raised)
	if !ok {
		t.Fatalf("expected a catchable AhdCode error (raised), got %#v", recovered)
	}
	if failure.failure.Name != "ArchiveError" {
		t.Fatalf("error name = %q, want ArchiveError", failure.failure.Name)
	}
	if !strings.Contains(failure.failure.Message, missing) {
		t.Fatalf("error message = %q, missing source path %q", failure.failure.Message, missing)
	}
}

// TestArchiveBuiltinRaisesCatchableArchiveErrorForEveryOperation proves the
// fix applies uniformly to zip, tar, and tarGzip, not only zip.
func TestArchiveBuiltinRaisesCatchableArchiveErrorForEveryOperation(t *testing.T) {
	for _, name := range []string{"zip", "tar", "tarGzip"} {
		t.Run(name, func(t *testing.T) {
			session := newArchiveTestSession(t)
			directory := t.TempDir()
			missing := filepath.Join(directory, "does-not-exist.txt")
			output := filepath.Join(directory, name+"-output")

			var recovered any
			func() {
				defer func() { recovered = recover() }()
				session.archiveBuiltin(name, []any{output, archiveTestPairValue("missing.txt", missing)})
			}()

			failure, ok := recovered.(raised)
			if !ok {
				t.Fatalf("%s: expected a catchable AhdCode error (raised), got %#v", name, recovered)
			}
			if failure.failure.Name != "ArchiveError" {
				t.Fatalf("%s: error name = %q, want ArchiveError", name, failure.failure.Name)
			}
		})
	}
}

// TestArchiveBuiltinFailurePreservesExistingDestination proves the refactor
// to an error-returning shared core did not weaken atomic publication: a
// failed Archive operation reached through the evaluator (not just the
// native wrapper, which internal/backend/golang/ahdruntime's own test
// already covers) must never touch a valid existing destination archive.
func TestArchiveBuiltinFailurePreservesExistingDestination(t *testing.T) {
	session := newArchiveTestSession(t)
	directory := t.TempDir()
	output := filepath.Join(directory, "existing.zip")
	if err := os.WriteFile(output, []byte("existing valid content"), 0o600); err != nil {
		t.Fatal(err)
	}

	func() {
		defer func() { _ = recover() }()
		session.archiveBuiltin("zip", []any{output, archiveTestPairValue("a.txt", filepath.Join(directory, "missing.txt"))})
	}()

	content, err := os.ReadFile(output)
	if err != nil || string(content) != "existing valid content" {
		t.Fatalf("a failed evaluator Archive build changed the destination: %q, %v", content, err)
	}
}

// TestArchiveBuiltinDoesNotSwallowUnrelatedPanics proves the fix translates
// only the explicit error the Archive core returns, not arbitrary panics.
// args[0] here is not a String, so the type assertion inside archiveBuiltin
// itself panics with a genuine Go interface-conversion failure -- a host
// defect unrelated to Archive validation -- and that must propagate exactly
// as any other unexpected panic would, never disguised as a catchable
// ArchiveError.
func TestArchiveBuiltinDoesNotSwallowUnrelatedPanics(t *testing.T) {
	session := newArchiveTestSession(t)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		session.archiveBuiltin("zip", []any{int64(42), archiveTestPairValue("a.txt", "a.txt")})
	}()

	if recovered == nil {
		t.Fatal("expected an unrelated Go panic to propagate")
	}
	if _, isRaised := recovered.(raised); isRaised {
		t.Fatalf("an unrelated Go panic must not be translated into an AhdCode error: %#v", recovered)
	}
}
