package ir

import (
	"strings"
	"testing"
)

func TestValidatorRejectsMalformedIRWithoutPanic(t *testing.T) {
	compilation := &Compilation{Entry: "main", Modules: []*Module{{ID: "main", Globals: []*Global{{ID: "x", Type: Type{Kind: RealType}, Initializer: &ConvertExpr{ExprBase: ExprBase{Type: Type{Kind: IntType}}, From: Type{Kind: RealType}}}}}}}
	diagnostics := Validate(compilation)
	if len(diagnostics) == 0 {
		t.Fatal("malformed IR was accepted")
	}
}

func TestValidatorChecksCallsReturnsAndClassParents(t *testing.T) {
	intType := Type{Kind: IntType}
	callable := CallableID("main::f")
	compilation := &Compilation{Entry: "main", Modules: []*Module{{
		ID: "main",
		Functions: []*Function{{
			ID: callable, Symbol: "main::f-symbol", Signature: Signature{Parameters: []ParameterType{{Name: "x", Type: intType}}, Return: intType},
			Body: Block{Statements: []Statement{&ReturnStmt{ReturnType: intType}}},
		}},
		Classes: []*Class{{ID: "main::Child", Symbol: "main::Child-symbol", Parent: "main::Missing"}},
		Init:    Block{Statements: []Statement{&ExprStmt{Value: &CallExpr{ExprBase: ExprBase{Type: intType, NullState: NonNull}, Callable: callable}}}},
	}}}
	diagnostics := Validate(compilation)
	for _, code := range []string{CodeInvalidCall, CodeInvalidReturn, CodeInvalidClass} {
		found := false
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == code {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s in %+v", code, diagnostics)
		}
	}
}

func TestValidatorChecksArgumentTypesDefaultsReturnValuesAndClassCallables(t *testing.T) {
	intType := Type{Kind: IntType}
	realType := Type{Kind: RealType}
	callable := CallableID("main::f")
	compilation := &Compilation{Entry: "main", Modules: []*Module{{
		ID: "main",
		Functions: []*Function{{
			ID: callable, Symbol: "main::f-symbol",
			Signature: Signature{Parameters: []ParameterType{{Name: "x", Type: intType}}, Return: intType},
			Body: Block{Statements: []Statement{&ReturnStmt{
				ReturnType: intType,
				Value:      &LiteralExpr{ExprBase: ExprBase{Type: realType}, Kind: RealLiteral, Value: "1.0"},
			}}},
		}},
		Classes: []*Class{{ID: "main::C", Symbol: "main::C-symbol", Constructor: "main::missing-constructor", Methods: []CallableID{"main::missing-method"}}},
		Init: Block{Statements: []Statement{&ExprStmt{Value: &CallExpr{
			ExprBase: ExprBase{Type: intType}, Callable: callable,
			Arguments: []Argument{{ParameterIndex: 0, ParameterName: "x", UsesDefault: true, Value: &LiteralExpr{ExprBase: ExprBase{Type: realType}, Kind: RealLiteral, Value: "2.0"}}},
		}}}},
	}}}
	diagnostics := Validate(compilation)
	counts := make(map[string]int)
	for _, diagnostic := range diagnostics {
		counts[diagnostic.Code]++
	}
	if counts[CodeInvalidCall] < 2 {
		t.Fatalf("call signature diagnostics = %+v", diagnostics)
	}
	if counts[CodeInvalidReturn] == 0 {
		t.Fatalf("return type diagnostic missing: %+v", diagnostics)
	}
	if counts[CodeUnresolvedIdentity] < 2 {
		t.Fatalf("Class callable diagnostics = %+v", diagnostics)
	}
}

func TestDeterministicDebugDump(t *testing.T) {
	compilation := &Compilation{Entry: "main", Modules: []*Module{{ID: "main", Globals: []*Global{{ID: "main::x", Name: "x", Type: Type{Kind: IntType}, Initializer: &LiteralExpr{ExprBase: ExprBase{Type: Type{Kind: IntType}, NullState: NonNull}, Kind: IntLiteral, Value: "5"}}}}}}
	want := "entry main\nmodule main deps=[]\n  global main::x : Int = int(\"5\")\n"
	if got := Dump(compilation); got != want {
		t.Fatalf("dump:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(Dump(compilation), "%!") {
		t.Fatal("dump contains formatting failure")
	}
}
