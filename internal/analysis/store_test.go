package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenValidDocumentHasNoDiagnostics(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	result := store.Open(path, "write(\"hello\")\n")
	if diagnostics, ok := result.Diagnostics[canonicalPath(path)]; !ok || len(diagnostics) != 0 {
		t.Fatalf("expected zero diagnostics for a valid document, got %#v", result.Diagnostics)
	}
}

func TestOpenLexerErrorIsReported(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	// An unterminated ordinary String literal is a lexer diagnostic (LEX007).
	result := store.Open(path, "write(\"unterminated)\n")
	diagnosticsFound := result.Diagnostics[canonicalPath(path)]
	if len(diagnosticsFound) == 0 {
		t.Fatal("expected a lexer diagnostic for an unterminated string")
	}
}

func TestOpenParserErrorIsReported(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	result := store.Open(path, "x: Int :=\n")
	diagnosticsFound := result.Diagnostics[canonicalPath(path)]
	if len(diagnosticsFound) == 0 {
		t.Fatal("expected a parser diagnostic for a missing RHS")
	}
	if diagnosticsFound[0].Code == "" {
		t.Fatal("expected a stable diagnostic code")
	}
}

func TestOpenSemanticTypeErrorIsReported(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	result := store.Open(path, "x: Int := \"not an int\"\n")
	diagnosticsFound := result.Diagnostics[canonicalPath(path)]
	if len(diagnosticsFound) == 0 {
		t.Fatal("expected a semantic type-mismatch diagnostic")
	}
}

func TestChangeAnalyzesUnsavedTextNotDiskContent(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(path, []byte("x: Int := \"still on disk, still wrong\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	// Open with the (also invalid) disk content, then fix it purely in memory.
	opened := store.Open(path, "x: Int := \"also wrong\"\n")
	if len(opened.Diagnostics[canonicalPath(path)]) == 0 {
		t.Fatal("expected the initial open to report the type error")
	}
	changed := store.Change(path, "write(\"now valid\")\n")
	if diagnosticsFound := changed.Diagnostics[canonicalPath(path)]; len(diagnosticsFound) != 0 {
		t.Fatalf("expected didChange to clear the diagnostic once the in-memory text is valid, got %#v", diagnosticsFound)
	}
	// The real file on disk must never have been touched.
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != "x: Int := \"still on disk, still wrong\"\n" {
		t.Fatalf("the analysis layer wrote to the real file: %q", onDisk)
	}
}

func TestImportedModuleErrorIsAttributedToItsOwnPath(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	if err := os.WriteFile(helperPath, []byte("x: Int := \"broken helper\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	result := store.Open(entryPath, "bring Helper\nwrite(\"ok\")\n")

	// The entry legitimately gets its own diagnostic too -- "module Helper
	// could not be analyzed", attributed to the `bring Helper` statement --
	// since importing a broken module really is an error at that call site.
	// What this test actually verifies is that Helper's own root-cause type
	// error is attributed to Helper.ahd's own path, not folded into the
	// entry's diagnostics or dropped.
	helperDiagnostics := result.Diagnostics[canonicalPath(helperPath)]
	if len(helperDiagnostics) == 0 {
		t.Fatal("expected the imported Helper.ahd's own type error to be attributed to its own path")
	}
	for _, item := range helperDiagnostics {
		if item.Code == "SEM033" {
			t.Fatalf("Helper.ahd's own diagnostics should be its real type error, not the importer's could-not-be-analyzed diagnostic: %#v", item)
		}
	}
}

func TestImportedModulePrefersOpenOverlayOverDiskContent(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	if err := os.WriteFile(helperPath, []byte("x: Int := \"broken on disk\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	// Open the sibling module directly with fixed, in-memory-only content.
	store.Open(helperPath, "x: Int := 5\n")

	result := store.Open(entryPath, "bring Helper\nwrite(\"ok\")\n")
	helperDiagnostics := result.Diagnostics[canonicalPath(helperPath)]
	if len(helperDiagnostics) != 0 {
		t.Fatalf("expected the entry's compile to use Helper's open in-memory text, not its broken disk content: %#v", helperDiagnostics)
	}
}

func TestCloseReportsPreviouslyOwnedPathsAndForgetsOverlay(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, "x: Int := \"wrong\"\n")

	owned := store.Close(path)
	found := false
	for _, item := range owned {
		if item == canonicalPath(path) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Close to report the entry's own path among previously-diagnosed paths, got %v", owned)
	}
	if _, ok := store.text(canonicalPath(path)); ok {
		t.Fatal("expected Close to forget the document's overlay text")
	}
}
