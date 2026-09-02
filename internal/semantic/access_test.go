package semantic

import (
	"testing"

	"ahdcode/internal/types"
)

func TestCanAccessConfidentialMemberSameClass(t *testing.T) {
	owner := &types.ClassSymbol{Name: "Account"}
	enclosing := &Symbol{Name: "Account", Class: owner}
	if !CanAccessConfidentialMember(enclosing, owner) {
		t.Fatal("expected access from same Class")
	}
}

func TestCanAccessConfidentialMemberDeniedOutside(t *testing.T) {
	owner := &types.ClassSymbol{Name: "Account"}
	if CanAccessConfidentialMember(nil, owner) {
		t.Fatal("expected nil enclosing class to deny access")
	}
}

func TestCanAccessConfidentialMemberSubclass(t *testing.T) {
	parent := &types.ClassSymbol{Name: "Animal"}
	child := &types.ClassSymbol{Name: "Dog", Parent: parent}
	enclosing := &Symbol{Name: "Dog", Class: child}
	if !CanAccessConfidentialMember(enclosing, parent) {
		t.Fatal("expected subclass to access parent Confidential members")
	}
}
