package types

import "testing"

func TestAssignableAndInvariantCollections(t *testing.T) {
	if !Assignable(Real, Int) || Assignable(Int, Real) {
		t.Fatal("numeric widening direction is incorrect")
	}
	if Assignable(List{Element: Real}, List{Element: Int}) {
		t.Fatal("mutable List types must be invariant")
	}
}

func TestClassInheritanceAssignment(t *testing.T) {
	object := &ClassSymbol{Name: "Object"}
	student := &ClassSymbol{Name: "Student", Parent: object}
	if !Assignable(Class{Symbol: object}, Class{Symbol: student}) {
		t.Fatal("derived instance should assign to base instance type")
	}
	if Assignable(Class{Symbol: student}, Class{Symbol: object}) {
		t.Fatal("base instance should not assign to derived instance type")
	}
}

func TestFunctionTypeEqualityIgnoresParameterNamesOnly(t *testing.T) {
	signature := func(name string, parameter, result Type) Function {
		return Function{Signature: &Signature{
			Parameters: []Parameter{{Name: name, Type: parameter}},
			Return:     result,
		}}
	}

	named := signature("value", Int, Int)
	otherName := signature("x", Int, Int)
	unnamed := signature("", Int, Int)
	if !Equal(named, otherName) || !Equal(named, unnamed) {
		t.Fatal("parameter spelling must not participate in Function type equality")
	}
	if Equal(named, signature("value", Real, Int)) {
		t.Fatal("parameter type mismatch must remain incompatible")
	}
	if Equal(named, signature("value", Int, Real)) {
		t.Fatal("return type mismatch must remain incompatible")
	}
	withDefault := Function{Signature: &Signature{
		Parameters: []Parameter{{Name: "value", Type: Int, HasDefault: true}},
		Return:     Int,
	}}
	if Equal(named, withDefault) {
		t.Fatal("required/default behavior must remain part of Function type equality")
	}
}
