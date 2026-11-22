package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReferencesAndRenameUseTheRequiredFilesOwnSpans(t *testing.T) {
	directory := t.TempDir()
	entry := filepath.Join(directory, "app.ahd")
	part := filepath.Join(directory, "Part.ahd")
	const entryText = "shared: Int := 42\r\nrequire(\"Part.ahd\")\r\n"
	const partText = "write(\"😀 {shared}\")\r\n"
	if err := os.WriteFile(entry, []byte(entryText), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part, []byte(partText), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	store.Open(entry, entryText)
	offset := offsetOf(t, entryText, "shared")
	partOffset := offsetOf(t, partText, "shared")
	locations := store.References(entry, offset, false)
	if len(locations) != 1 || locations[0].Path != canonicalPath(part) ||
		locations[0].SourceText != partText || locations[0].Span.Start.Offset != partOffset {
		t.Fatalf("reference must use the required file's path, snapshot and offset: %+v", locations)
	}
	edits, ok := store.Rename(entry, offset, "answer")
	if !ok || len(edits) != 2 {
		t.Fatalf("rename must edit the declaration and required file: %+v, %t", edits, ok)
	}
	for _, edit := range edits {
		switch edit.Path {
		case canonicalPath(entry):
			if edit.SourceText == entryText && edit.Span.Start.Offset == offset {
				continue
			}
		case canonicalPath(part):
			if edit.SourceText == partText && edit.Span.Start.Offset == partOffset {
				continue
			}
		}
		t.Fatalf("rename edit refers to the wrong source span: %+v", edit)
	}
}

func TestRenameUpdatesWorkspaceReferences(t *testing.T) {
	directory := t.TempDir()
	mainPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	if err := os.WriteFile(helperPath, []byte("answer: Int := 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mainText := "from Helper bring answer\nwrite(answer)\n"
	store := NewStore()
	store.Open(mainPath, mainText)
	edits, ok := store.Rename(mainPath, offsetOf(t, mainText, "write(answer)")+len("write("), "result")
	if !ok || len(edits) != 3 {
		t.Fatalf("workspace rename edits = %#v, ok=%v; want 3", edits, ok)
	}
	for _, edit := range edits {
		if edit.NewText != "result" {
			t.Fatalf("rename edit = %#v", edit)
		}
	}
}

func TestRenamePreservesCaptureSigil(t *testing.T) {
	text := "value: Local Int := 7\n" +
		"read: Function := () -> Int uses [#value] { return value }\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	edits, ok := store.Rename(path, offsetOf(t, text, "#value")+2, "amount")
	if !ok || len(edits) != 3 {
		t.Fatalf("capture rename edits = %#v, ok=%v; want declaration, capture, and body", edits, ok)
	}
	for _, edit := range edits {
		if edit.Span.Start.Offset == offsetOf(t, text, "#value")+1 && edit.NewText != "amount" {
			t.Fatalf("capture edit = %#v", edit)
		}
	}
}

func TestRenamePreservesGlobalCaptureSigil(t *testing.T) {
	text := "moduleValue: Int := 3\n" +
		"read: Function := () -> Int uses [@moduleValue] { return moduleValue }\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	edits, ok := store.Rename(path, offsetOf(t, text, "@moduleValue")+2, "answer")
	if !ok || len(edits) != 3 {
		t.Fatalf("global capture rename edits = %#v, ok=%v; want 3", edits, ok)
	}
	for _, edit := range edits {
		if edit.NewText != "answer" {
			t.Fatalf("global capture rename edit = %#v", edit)
		}
	}
}

func TestRenameMultiFileWorkspaceComprehensive(t *testing.T) {
	dir := t.TempDir()
	helperPath := filepath.Join(dir, "Helper.ahd")
	consumerPath := filepath.Join(dir, "Consumer.ahd")
	mainPath := filepath.Join(dir, "main.ahd")

	helperSrc := "sharedVal: Int := 100\ncalc: Function := (x: Int) -> Int { return x + 1 }\n"
	consumerSrc := "from Helper bring sharedVal, calc\ncallCalc: Function := () -> Int uses [@sharedVal] {\n    return calc(sharedVal)\n}\n"
	mainSrc := "from Helper bring sharedVal\nouter: Function := () -> Int {\n    sharedVal: Local Int := 5\n    l := lambda [#sharedVal] () -> sharedVal * 2\n    inner: Function := () -> Int uses [@sharedVal] {\n        return sharedVal + l()\n    }\n    return inner()\n}\n"

	if err := os.WriteFile(helperPath, []byte(helperSrc), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(consumerPath, []byte(consumerSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewStore()
	store.Open(mainPath, mainSrc)

	// 1. Rename module binding sharedVal -> globalNumber
	// Cursor at @sharedVal in main.ahd
	edits, ok := store.Rename(mainPath, offsetOf(t, mainSrc, "@sharedVal")+1, "globalNumber")
	if !ok {
		t.Fatal("rename failed for @sharedVal")
	}
	// Check that edits only touch global occurrences (Helper declaration, imports in consumer and main, uses [@...], references),
	// but DO NOT touch the local variable `sharedVal: Local Int` or lambda `#sharedVal`!
	for _, edit := range edits {
		if edit.Path == canonicalPath(mainPath) {
			// In mainPath, it should touch `from Helper bring sharedVal` and `@sharedVal`,
			// NOT `sharedVal: Local Int` (offset: offsetOf(t, mainSrc, "sharedVal: Local"))
			localDeclOffset := offsetOf(t, mainSrc, "sharedVal: Local")
			if edit.Span.Start.Offset == localDeclOffset {
				t.Fatalf("rename of module symbol wrongly touched local binding in main.ahd at %d", edit.Span.Start.Offset)
			}
			localCaptureOffset := offsetOf(t, mainSrc, "#sharedVal") + 1
			if edit.Span.Start.Offset == localCaptureOffset {
				t.Fatalf("rename of module symbol wrongly touched local lambda capture #sharedVal in main.ahd at %d", edit.Span.Start.Offset)
			}
		}
	}

	// 2. Rename local variable sharedVal in main.ahd -> myLocal
	localOffset := offsetOf(t, mainSrc, "sharedVal: Local")
	localEdits, ok := store.Rename(mainPath, localOffset, "myLocal")
	if !ok {
		t.Fatal("rename failed for local sharedVal")
	}
	// Must only touch mainPath: local declaration, lambda [#myLocal], uses [#myLocal], and body return
	for _, edit := range localEdits {
		if edit.Path != canonicalPath(mainPath) {
			t.Fatalf("rename of local symbol touched external file %s", edit.Path)
		}
		// Must not touch module import or @sharedVal
		importOffset := offsetOf(t, mainSrc, "from Helper bring sharedVal") + len("from Helper bring ")
		if edit.Span.Start.Offset == importOffset {
			t.Fatalf("rename of local symbol wrongly touched module import")
		}
		globalCaptureOffset := offsetOf(t, mainSrc, "@sharedVal") + 1
		if edit.Span.Start.Offset == globalCaptureOffset {
			t.Fatalf("rename of local symbol wrongly touched @sharedVal")
		}
	}
}
