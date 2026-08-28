package semantic

import (
	"reflect"
	"testing"

	"ahdcode/internal/types"
)

func TestMathStandardModuleHasTheExactV01Surface(t *testing.T) {
	mathModule := StandardModuleInterfaces()["Math"]
	if mathModule == nil || mathModule.ModuleID != "builtin:Math" {
		t.Fatalf("Math module = %#v", mathModule)
	}
	want := []string{"E", "PI", "ceil", "cos", "exp", "floor", "log", "log10", "random", "randomInt", "round", "seed", "sin", "sqrt", "tan"}
	if !reflect.DeepEqual(mathModule.ExportNames, want) {
		t.Fatalf("Math exports = %v, want %v", mathModule.ExportNames, want)
	}
	for _, name := range want {
		if mathModule.Exports[name] == nil || mathModule.Symbols[name] != mathModule.Exports[name] {
			t.Fatalf("Math export %q is incomplete", name)
		}
	}
	for _, name := range []string{"abs", "sum", "min", "max", "pow"} {
		if mathModule.Symbols[name] != nil {
			t.Fatalf("Math must not export %q", name)
		}
	}
}

func TestMathStandardModuleSignaturesAndConstants(t *testing.T) {
	mathModule := StandardModuleInterfaces()["Math"]
	for _, name := range []string{"PI", "E"} {
		symbol := mathModule.Exports[name]
		if symbol == nil || !symbol.Constant || symbol.Type.Kind() != types.RealKind || symbol.ConstValue == nil || symbol.BuiltinLiteral == "" {
			t.Fatalf("Math.%s = %#v", name, symbol)
		}
	}
	round := mathModule.Exports["round"]
	if round.OverloadSet == nil || len(round.OverloadSet.Candidates) != 2 {
		t.Fatalf("Math.round overloads = %#v", round.OverloadSet)
	}
	assertMathSignature(t, round.OverloadSet.Candidates[0], types.Real, types.Real)
	assertMathSignature(t, round.OverloadSet.Candidates[1], types.Real, types.Real, types.Int)
	assertMathSignature(t, mathModule.Exports["floor"].Callable, types.Int, types.Real)
	assertMathSignature(t, mathModule.Exports["ceil"].Callable, types.Int, types.Real)
	assertMathSignature(t, mathModule.Exports["seed"].Callable, types.Nothing, types.Int)
	assertMathSignature(t, mathModule.Exports["random"].Callable, types.Real)
	assertMathSignature(t, mathModule.Exports["randomInt"].Callable, types.Int, types.Int, types.Int)
}

func assertMathSignature(t *testing.T, callable *Callable, result types.Type, parameters ...types.Type) {
	t.Helper()
	if callable == nil || callable.Signature == nil || !types.Equal(callable.Signature.Return, result) || len(callable.Signature.Parameters) != len(parameters) {
		t.Fatalf("Math signature = %#v, want (%v) -> %s", callable, parameters, types.Display(result))
	}
	for index, parameter := range parameters {
		if !types.Equal(callable.Signature.Parameters[index].Type, parameter) || callable.ParameterNull[index] != NonNull {
			t.Fatalf("Math parameter %d = %#v, want NonNull %s", index, callable.Signature.Parameters[index], types.Display(parameter))
		}
	}
	if callable.ReturnNull != NonNull {
		t.Fatalf("Math return null-state = %s", callable.ReturnNull)
	}
}
