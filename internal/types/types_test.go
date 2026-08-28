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
