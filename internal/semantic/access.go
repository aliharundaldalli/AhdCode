package semantic

import "ahdcode/internal/types"

// CanAccessConfidentialMember reports whether code executing in enclosingClass
// may access a Confidential member owned by ownerClass. enclosingClass is the
// Class symbol for the innermost enclosing Class callable (a method or
// structure body); nil means module scope or a non-class callable, where
// Confidential members are never accessible. This mirrors the analyzer's own
// canAccessConfidentialMember rule without duplicating class hierarchy logic.
func CanAccessConfidentialMember(enclosingClass *Symbol, ownerClass *types.ClassSymbol) bool {
	if ownerClass == nil || enclosingClass == nil || enclosingClass.Class == nil {
		return false
	}
	return classAssignableTo(enclosingClass.Class, ownerClass)
}
