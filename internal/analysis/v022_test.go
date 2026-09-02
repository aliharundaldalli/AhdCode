package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/lexer"
)

func TestCompletionConfidentialMemberInsideClass(t *testing.T) {
	text := "Account: Class<> := {\n" +
		"    structure: Attributes := (\n" +
		"        name: String\n" +
		"        password: Confidential String\n" +
		"    )\n\n" +
		"    describe: Function := () -> String {\n" +
		"        return attribute.pass\n" +
		"    }\n" +
		"}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	offset := offsetOf(t, text, "attribute.pass") + len("attribute.pass")
	items := store.Completion(path, offset)
	if !hasLabel(items, "password") {
		t.Fatalf("expected Confidential member password inside Class, got %#v", items)
	}
}

func TestCompletionConfidentialMemberOutsideClassStillHidden(t *testing.T) {
	text := "Account: Class<> := {\n" +
		"    structure: Attributes := (\n" +
		"        name: String\n" +
		"        password: Confidential String\n" +
		"    )\n" +
		"}\n" +
		"account: Account := Account(name: \"Ada\", password: \"x\")\n" +
		"write(account.pass)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	offset := offsetOf(t, text, "account.pass") + len("account.pass")
	items := store.Completion(path, offset)
	if hasLabel(items, "password") {
		t.Fatalf("expected password hidden outside Class, got %#v", items)
	}
}

func TestAutoImportCompletionFromDiscoveredModule(t *testing.T) {
	directory := t.TempDir()
	researchPath := filepath.Join(directory, "ResearchTools.ahd")
	researchText := "specialFunction: Function := (value: Int) -> Int {\n    return value\n}\n"
	if err := os.WriteFile(researchPath, []byte(researchText), 0o600); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(directory, "main.ahd")
	mainText := "spe\n"
	store := NewStore()
	store.Open(mainPath, mainText)

	items := store.Completion(mainPath, len("spe"))
	if !hasLabel(items, "specialFunction") {
		t.Fatalf("expected auto-import completion for specialFunction, got %#v", items)
	}
	found := false
	for _, item := range items {
		if item.Label == "specialFunction" {
			if item.Import == nil || item.Import.ModuleName != "ResearchTools" {
				t.Fatalf("expected import from ResearchTools, got %#v", item.Import)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("specialFunction item missing import metadata")
	}
}

func TestRenameLocalDoesNotTouchShadowedRoot(t *testing.T) {
	text := "value: Int := 1\n\n" +
		"f: Function := () -> Nothing {\n" +
		"    value: Local Int := 2\n" +
		"    write(value)\n" +
		"}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	offset := offsetOf(t, text, "write(value)") + len("write(")
	edits, ok := store.Rename(path, offset, "total")
	if !ok {
		t.Fatal("rename failed")
	}
	if len(edits) != 2 {
		t.Fatalf("expected 2 edits (local declaration + use), got %d: %#v", len(edits), edits)
	}
}

func TestPrepareRenameRejectsBuiltin(t *testing.T) {
	text := "write(1)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	offset := offsetOf(t, text, "write")
	if _, ok := store.PrepareRename(path, offset); ok {
		t.Fatal("expected prepareRename to reject builtin write")
	}
}

func TestRenameRejectsInvalidIdentifier(t *testing.T) {
	text := "count: Int := 1\nwrite(count)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	offset := offsetOf(t, text, "count")
	if _, ok := store.Rename(path, offset, "1bad"); ok {
		t.Fatal("expected rename to reject invalid identifier")
	}
	if _, ok := store.Rename(path, offset, "if"); ok {
		t.Fatal("expected rename to reject keyword")
	}
}

func TestValidIdentifierMatchesLexer(t *testing.T) {
	if !lexer.ValidIdentifier("score") || !lexer.ValidIdentifier("café") || !lexer.ValidIdentifier("_hidden") {
		t.Fatal("expected valid identifiers")
	}
	if lexer.ValidIdentifier("1bad") || lexer.ValidIdentifier("") {
		t.Fatal("expected invalid identifiers rejected")
	}
}

func TestSemanticTokensUTF16Length(t *testing.T) {
	text := "score: Real := 85.0\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	tokens := store.SemanticTokens(path)
	encoded := EncodeSemanticTokens(text, tokens)
	if len(encoded) == 0 {
		t.Fatal("expected semantic tokens")
	}
	if len(encoded)%5 != 0 {
		t.Fatalf("encoded length %d is not a multiple of 5", len(encoded))
	}
}

func TestSemanticTokensUnicodeLine(t *testing.T) {
	text := "/* 🙂 */ score: Int := 1\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	tokens := store.SemanticTokens(path)
	encoded := EncodeSemanticTokens(text, tokens)
	if len(encoded) < 5 {
		t.Fatalf("expected tokens after emoji, got %v", encoded)
	}
}

func TestInlayHintInferredType(t *testing.T) {
	text := "score := 85.0\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	hints := store.InlayHints(path)
	found := false
	for _, hint := range hints {
		if hint.Kind == InlayHintType && hint.Label == ": Real" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected inferred Real type hint, got %#v", hints)
	}
}

func TestInlayHintNoDuplicateExplicitType(t *testing.T) {
	text := "score: Real := 85.0\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	for _, hint := range store.InlayHints(path) {
		if hint.Kind == InlayHintType {
			t.Fatalf("unexpected type hint on explicit declaration: %#v", hint)
		}
	}
}

func TestCodeActionMissingLocal(t *testing.T) {
	text := "f: Function := () -> Nothing {\n    x: Int := 1\n}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	result := store.Open(path, text)
	var diagOffset int
	for _, items := range result.Diagnostics {
		for _, item := range items {
			if item.Code == codeMissingLocal {
				diagOffset = item.Span.Start.Offset
			}
		}
	}
	if diagOffset == 0 {
		t.Fatal("expected missing Local diagnostic")
	}
	actions := store.CodeActions(path, diagOffset)
	if len(actions) == 0 {
		t.Fatal("expected quick fix for missing Local")
	}
}

func TestAutoImportReflectsUnsavedModuleEdit(t *testing.T) {
	directory := t.TempDir()
	researchPath := filepath.Join(directory, "ResearchTools.ahd")
	if err := os.WriteFile(researchPath, []byte("specialFunction: Function := (value: Int) -> Int { return value }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(mainPath, "ren\n")
	store.Open(researchPath, "renamedFunction: Function := (value: Int) -> Int { return value }\n")

	items := store.Completion(mainPath, len("ren"))
	if hasLabel(items, "specialFunction") {
		t.Fatalf("stale specialFunction still offered after unsaved rename, got %#v", items)
	}
	if !hasLabel(items, "renamedFunction") {
		t.Fatalf("expected renamedFunction from unsaved module, got %#v", items)
	}
}

func TestFormatDocumentMatchesFormatter(t *testing.T) {
	text := "write( 1 )\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	formatted, ok := store.FormatDocument(path)
	if !ok {
		t.Fatal("format failed")
	}
	if formatted == text {
		t.Fatalf("expected formatting to change source, got %q", formatted)
	}
}

func TestWorkspaceSymbolsFindsUserModule(t *testing.T) {
	directory := t.TempDir()
	researchPath := filepath.Join(directory, "ResearchTools.ahd")
	if err := os.WriteFile(researchPath, []byte("specialFunction: Function := (value: Int) -> Int { return value }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(mainPath, "write(1)\n")
	symbols := store.WorkspaceSymbols(mainPath, "special")
	found := false
	for _, symbol := range symbols {
		if symbol.Name == "specialFunction" && symbol.ModuleName == "ResearchTools" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected specialFunction from ResearchTools, got %#v", symbols)
	}
}

func TestFoldingRangesFunctionBody(t *testing.T) {
	text := "f: Function := () -> Nothing {\n    write(1)\n}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	ranges := store.FoldingRanges(path)
	if len(ranges) == 0 {
		t.Fatal("expected folding ranges")
	}
}

func TestSelectionRangesExpand(t *testing.T) {
	text := "f: Function := () -> Nothing {\n    write(1)\n}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)
	offset := offsetOf(t, text, "write")
	head := store.SelectionRanges(path, offset)
	if head == nil || head.Parent == nil {
		t.Fatal("expected nested selection ranges")
	}
}
