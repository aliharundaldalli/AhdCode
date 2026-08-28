package ir

import "testing"

func TestFunctionTypeEqualityIgnoresParameterNames(t *testing.T) {
	function := func(name string, parameter, result Type) Type {
		return Type{Kind: FunctionType, Signature: &Signature{
			Parameters: []ParameterType{{Name: name, Type: parameter}},
			Return:     result,
		}}
	}
	intType := Type{Kind: IntType}
	realType := Type{Kind: RealType}

	if !EqualType(function("value", intType, intType), function("", intType, intType)) ||
		!EqualType(function("x", intType, intType), function("y", intType, intType)) {
		t.Fatal("lowered Function type equality depends on parameter spelling")
	}
	if EqualType(function("x", intType, intType), function("y", realType, intType)) {
		t.Fatal("lowered Function parameter type mismatch was accepted")
	}
	if EqualType(function("x", intType, intType), function("y", intType, realType)) {
		t.Fatal("lowered Function return type mismatch was accepted")
	}
}
