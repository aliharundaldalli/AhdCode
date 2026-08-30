package evaluator

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/ir"
	"ahdcode/internal/source"
)

func evaluatorSpan(offset int) source.Span {
	return source.Span{Start: source.Position{Offset: offset}, End: source.Position{Offset: offset + 1}}
}

func TestExecutePreservesStorageAndRunsOnlyNewSpans(t *testing.T) {
	session := New(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, t.TempDir())
	symbol := ir.SymbolID("x")
	intType := ir.Type{Kind: ir.IntType}
	literal := func(value string, offset int) ir.Expr {
		return &ir.LiteralExpr{ExprBase: ir.ExprBase{Span: evaluatorSpan(offset), Type: intType, NullState: ir.NonNull}, Kind: ir.IntLiteral, Value: value}
	}
	first := &ir.BindingStmt{StmtBase: ir.StmtBase{Span: evaluatorSpan(0)}, Symbol: symbol, Name: "x", Type: intType, Storage: ir.ModuleStorage, Initializer: literal("5", 0)}
	module := &ir.Module{ID: "entry", Init: ir.Block{Statements: []ir.Statement{first}}}
	if result := session.Execute(&ir.Compilation{Entry: "entry", Modules: []*ir.Module{module}}, 0); result.Failure != nil {
		t.Fatal(result.Failure)
	}
	assignment := &ir.AssignStmt{StmtBase: ir.StmtBase{Span: evaluatorSpan(10)}, Target: ir.Target{Kind: ir.SymbolTarget, Symbol: symbol, Type: intType}, Value: literal("7", 10)}
	load := &ir.LoadExpr{ExprBase: ir.ExprBase{Span: evaluatorSpan(11), Type: intType, NullState: ir.NonNull}, Symbol: symbol}
	module = &ir.Module{ID: "entry", Init: ir.Block{Statements: []ir.Statement{
		first,
		assignment,
		&ir.ExprStmt{StmtBase: ir.StmtBase{Span: evaluatorSpan(11)}, Value: load},
	}}}
	result := session.Execute(&ir.Compilation{Entry: "entry", Modules: []*ir.Module{module}}, 10)
	if result.Failure != nil || !result.HasValue || result.Value != int64(7) {
		t.Fatalf("second execution = %#v", result)
	}
}

func TestEvaluatorAliasesRNGInputAndFilesystem(t *testing.T) {
	var output bytes.Buffer
	session := New(bufio.NewReader(strings.NewReader("Ali\n")), &output, t.TempDir())
	if got := session.core("take", nil, []any{"Name: "}); got != "Ali" || output.String() != "Name: " {
		t.Fatalf("take = %q, output = %q", got, output.String())
	}
	list := &List{Items: []any{int64(1)}}
	alias := list
	session.core("List.add", list, []any{int64(2)})
	if alias != list || session.Render(alias) != "[1, 2]" {
		t.Fatal("List alias identity was not preserved")
	}
	session.math("seed", []any{int64(42)})
	first := session.math("random", nil)
	second := session.math("random", nil)
	if first == second {
		t.Fatal("Math RNG did not advance")
	}
	path := filepath.Join("nested", "note.txt")
	session.fileBuiltin("createDir", []any{"nested"})
	session.fileBuiltin("writeText", []any{path, "hello"})
	if got := session.fileBuiltin("readText", []any{path}); got != "hello" {
		t.Fatalf("File.readText = %q", got)
	}
	if got := session.pathBuiltin("base", []any{path}); got != "note.txt" {
		t.Fatalf("Path.base = %q", got)
	}
}
