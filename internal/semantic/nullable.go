package semantic

import (
	"fmt"

	"ahdcode/internal/source"
)

// nullStateFor converts a static declared-nullable bit into the flow seed
// state a fresh binding/parameter starts from: a nullable declaration begins
// MaybeNull (its value is not yet known), and a non-nullable one begins
// NonNull, since v0.1.7 statically forbids ever storing null there.
func nullStateFor(nullable bool) NullState {
	if nullable {
		return MaybeNull
	}
	return NonNull
}

// requireCompatibleNull enforces the one rule that survives every other kind
// of null-safety flexibility: a value flowing into a non-nullable-declared
// slot (a variable initializer, an assignment, a parameter default, a
// return, a Class attribute initializer, a Pair/List literal element) must
// be statically proven NonNull. It reports codeNullNotAllowed and returns
// false on violation; an already-invalid value never cascades a second
// diagnostic.
func (a *analyzer) requireCompatibleNull(nullable bool, info expressionInfo, span source.Span, subject string) bool {
	if nullable || info.invalid() || info.nullState == NonNull {
		return true
	}
	a.error(codeNullNotAllowed, fmt.Sprintf("%s is not nullable; received a value that may be null", subject), span, "declare the type with a trailing ? or prove the value non-null before this point")
	return false
}

// targetNullable reports whether null may legally be written to an already
// analyzed assignment/mutation target. An identifier or member target's
// declared nullability never changes with narrowing, so it is read from the
// resolved symbol directly. An index target (list[i] = ...) has no backing
// symbol; analyzeIndex already reports the element/value slot's structural
// nullability as its nullState, unaffected by flow narrowing (indices are
// never narrowed), so that value doubles as the answer here.
func targetNullable(target expressionInfo) bool {
	if target.symbol != nil {
		return target.symbol.DeclaredNullable
	}
	return target.nullState != NonNull
}

// declaredNullState is the null state a declaration's own type asserts,
// independent of any initializer. It is the fallback when an initializer is
// already invalid and therefore says nothing about nullability.
func declaredNullState(nullable bool) NullState {
	if nullable {
		return MaybeNull
	}
	return NonNull
}
