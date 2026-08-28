package ir

import (
	"strings"
	"testing"
)

func TestValidatorRejectsMalformedIRWithoutPanic(t *testing.T) {
	compilation := &Compilation{Entry: "main", Modules: []*Module{{ID: "main", Globals: []*Global{{ID: "x", Type: Type{Kind: IntType}, Initializer: &ConvertExpr{ExprBase: ExprBase{Type: Type{Kind: IntType}}, From: Type{Kind: BoolType}, Value: &LiteralExpr{ExprBase: ExprBase{Type: Type{Kind: BoolType}}, Kind: BoolLiteral, Value: "true"}}}}}}}
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

// TestInvalidPairKeyTypeIsRejected checks the defensive IR guard: the frontend
// enforces the Pair key rule, and a hand-built IR that violates it must not
// reach a backend.
func TestInvalidPairKeyTypeIsRejected(t *testing.T) {
	realType := Type{Kind: RealType}
	stringType := Type{Kind: StringType}
	intType := Type{Kind: IntType}
	pairOf := func(key, value Type) Type {
		return Type{Kind: PairType, Key: &key, Value: &value}
	}
	cases := map[string]Type{
		"Real key":            pairOf(realType, stringType),
		"nested Real key":     {Kind: ListType, Element: pointerTo(pairOf(realType, intType))},
		"Class key":           pairOf(Type{Kind: ClassType, Class: "m::class::C"}, intType),
		"Pair key":            pairOf(pairOf(stringType, intType), intType),
		"Real key in a value": pairOf(stringType, pairOf(realType, intType)),
	}
	for name, declared := range cases {
		t.Run(name, func(t *testing.T) {
			compilation := &Compilation{
				Entry: "m",
				Modules: []*Module{{
					ID: "m", Name: "Main",
					Globals: []*Global{{ID: "g", Name: "g", Type: declared, NullState: NonNull}},
				}},
			}
			found := false
			for _, diagnostic := range Validate(compilation) {
				if diagnostic.Code == CodeMalformedNode {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %s for an invalid Pair key type", CodeMalformedNode)
			}
		})
	}
}

func TestValidPairKeyTypesPassValidation(t *testing.T) {
	intType := Type{Kind: IntType}
	for name, key := range map[string]Type{
		"String": {Kind: StringType},
		"Int":    {Kind: IntType},
		"Bool":   {Kind: BoolType},
	} {
		t.Run(name, func(t *testing.T) {
			keyCopy := key
			declared := Type{Kind: PairType, Key: &keyCopy, Value: &intType}
			compilation := &Compilation{
				Entry: "m",
				Modules: []*Module{{
					ID: "m", Name: "Main",
					Globals: []*Global{{ID: "g", Name: "g", Type: declared, NullState: NonNull}},
				}},
			}
			if diagnostics := Validate(compilation); len(diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics for a %s key: %+v", name, diagnostics)
			}
		})
	}
}

func pointerTo(value Type) *Type { return &value }
