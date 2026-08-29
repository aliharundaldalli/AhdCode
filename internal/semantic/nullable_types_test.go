package semantic

import "testing"

// TestNullableTypeVsPlainType locks in the core v0.1.7 default: a plain type
// never accepts null, and T? does.
func TestNullableTypeVsPlainType(t *testing.T) {
	_, rejected := analyzeText(t, "age: Int := null")
	requireSemanticCode(t, rejected, codeNullNotAllowed)

	_, accepted := analyzeText(t, "age: Int? := null")
	requireSemanticClean(t, accepted)
}

// TestInferredDeclarationTypeMatrix covers the type-inference decision:
// `name := value` infers a fixed, non-nullable type from the initializer,
// never widens to T? merely because null exists, and rejects a bare null
// initializer outright since it carries no type at all.
func TestInferredDeclarationTypeMatrix(t *testing.T) {
	_, clean := analyzeText(t, `age := 25
price := 12.5
name := "Ali"
active := true
numbers := [1, 2]
age = 30`)
	requireSemanticClean(t, clean)

	_, rejectedNullAssign := analyzeText(t, "age := 25\nage = null")
	requireSemanticCode(t, rejectedNullAssign, codeNullNotAllowed)

	_, cannotInfer := analyzeText(t, "value := null")
	requireSemanticCode(t, cannotInfer, codeCannotInferType)
}

// TestNonNullToNullableWidening covers the one safe implicit conversion: a
// proven-NonNull T is usable wherever T? is expected, but never the reverse.
func TestNonNullToNullableWidening(t *testing.T) {
	_, widened := analyzeText(t, `acceptNullable: Function := (
    x: Int?
) -> Nothing {
}
number: Int := 5
acceptNullable(number)`)
	requireSemanticClean(t, widened)

	_, rejected := analyzeText(t, `acceptStrict: Function := (
    x: Int
) -> Nothing {
}
relay: Function := (
    maybeNumber: Int?
) -> Nothing {
    acceptStrict(maybeNumber)
}`)
	requireSemanticCode(t, rejected, codeNullableUse)
}

// TestFlowNarrowingMatrix covers if/!=, if/==-with-else, and early-return
// narrowing, plus the sibling requirement that the null branch itself stays
// nullable/null-known.
func TestFlowNarrowingMatrix(t *testing.T) {
	_, notNullBranch := analyzeText(t, `value: Int? := 5
if value != null {
    write(value + 1)
}`)
	requireSemanticClean(t, notNullBranch)

	_, elseBranch := analyzeText(t, `value: Int? := 5
if value == null {
    write("missing")
} else {
    write(value + 1)
}`)
	requireSemanticClean(t, elseBranch)

	_, earlyReturn := analyzeText(t, `describe: Function := (
    value: Int?
) -> Nothing {
    if value == null {
        return
    }

    write(value + 1)
}`)
	requireSemanticClean(t, earlyReturn)

	// The null branch must still see the value as usable only as null/nullable,
	// never coerced to NonNull.
	_, nullBranchStillNullable := analyzeText(t, `value: Int? := 5
if value == null {
    write(value + 1)
}`)
	requireSemanticCode(t, nullBranchStillNullable, codeNullableUse)
}

// TestNarrowingInvalidatedByReassignment ensures a narrowed binding loses its
// NonNull status the moment it is assigned again, even back to a nullable
// value inside the same narrowed branch.
func TestNarrowingInvalidatedByReassignment(t *testing.T) {
	_, result := analyzeText(t, `value: Int? := 5
if value != null {
    value = null
    write(value + 1)
}`)
	requireSemanticCode(t, result, codeNullableUse)
}

// TestNullableMemberIndexAndArithmeticRejection covers 2.6: unsafe operations
// on T? are rejected until narrowed.
func TestNullableMemberIndexAndArithmeticRejection(t *testing.T) {
	_, memberAccess := analyzeText(t, `User: Class<> := {
    structure: Attributes := (name: String)
}
user: User? := null
write(user.name)`)
	requireSemanticCode(t, memberAccess, codeNullableUse)

	_, arithmetic := analyzeText(t, `useNumber: Function := (
    number: Int?
) -> Nothing {
    number + 1
}`)
	requireSemanticCode(t, arithmetic, codeNullableUse)

	_, indexing := analyzeText(t, "items: List<Int>? := null\nitems[0]")
	requireSemanticCode(t, indexing, codeNullableUse)
}

// TestNullableFunctionSignatures covers 2.7: nullable parameters/returns are
// part of the signature, a non-null-returning Function must never return
// null, and a nullable default must respect the declared type.
func TestNullableFunctionSignatures(t *testing.T) {
	_, validReturn := analyzeText(t, `findUser: Function := (
    id: Int
) -> String? {
    if id == 1 {
        return "found"
    }

    return null
}`)
	requireSemanticClean(t, validReturn)

	_, invalidReturn := analyzeText(t, `getUser: Function := (
    id: Int
) -> String {
    return null
}`)
	requireSemanticCode(t, invalidReturn, codeNullNotAllowed)

	_, validParam := analyzeText(t, `show: Function := (
    value: String?
) -> Nothing {
}`)
	requireSemanticClean(t, validParam)

	_, validDefault := analyzeText(t, `f: Function := (
    x: Int? := null
) -> Nothing {
}`)
	requireSemanticClean(t, validDefault)

	_, invalidDefault := analyzeText(t, `f: Function := (
    x: Int := null
) -> Nothing {
}`)
	requireSemanticCode(t, invalidDefault, codeNullNotAllowed)
}

// TestNullableOverloadResolution covers 2.8: a proven-non-null argument
// prefers the exact non-nullable overload, an un-narrowed nullable argument
// cannot select a non-nullable-only overload, and a bare null literal that
// matches more than one nullable overload is an ambiguity rather than a
// guess.
func TestNullableOverloadResolution(t *testing.T) {
	_, preferExact := analyzeText(t, `show: Function := (
    x: Int
) -> String {
    return "int"
}

show: Overload Function := (
    x: Int?
) -> String {
    return "maybe-int"
}

value: Int := 5
write(show(value))`)
	requireSemanticClean(t, preferExact)

	_, cannotSelectNonNullOnly := analyzeText(t, `show: Function := (
    x: Int
) -> String {
    return "int"
}

relay: Function := (
    maybe: Int?
) -> Nothing {
    write(show(maybe))
}`)
	requireSemanticCode(t, cannotSelectNonNullOnly, codeNullableUse)

	_, ambiguousNull := analyzeText(t, `show: Function := (
    x: Int?
) -> String {
    return "int"
}

show: Overload Function := (
    x: String?
) -> String {
    return "string"
}

show(null)`)
	requireSemanticCode(t, ambiguousNull, codeAmbiguousOverload)
}

// TestNullableGenericComposition covers 2.9: List<T?> and List<T>? are
// distinct, and this composes with Pair values (keys are never nullable).
func TestNullableGenericComposition(t *testing.T) {
	_, elementNullable := analyzeText(t, "numbers: List<Int?> := [1, null, 3]")
	requireSemanticClean(t, elementNullable)

	_, listNullable := analyzeText(t, "maybeNumbers: List<Int>? := null")
	requireSemanticClean(t, listNullable)

	_, distinctTypes := analyzeText(t, "numbers: List<Int?> := null")
	requireSemanticCode(t, distinctTypes, codeNullNotAllowed)

	_, pairValueNullable := analyzeText(t, `users: Pair<String, Int?> := {
    "first": null
}`)
	requireSemanticClean(t, pairValueNullable)

	_, mixedPairValues := analyzeText(t, `scores: Pair<String, Int?> := {
    "a": 1
    "b": null
}`)
	requireSemanticClean(t, mixedPairValues)

	_, nullableKeyRejected := analyzeText(t, "bad: Pair<String?, Int> := {}")
	requireSemanticCode(t, nullableKeyRejected, codeInvalidPairKey)
}

// TestNullableClassReferences covers 2.10: a Class reference may be
// declared nullable, and a non-nullable Class reference stays guaranteed
// non-null.
func TestNullableClassReferences(t *testing.T) {
	_, nullableRef := analyzeText(t, `User: Class<> := {}
parent: User? := null`)
	requireSemanticClean(t, nullableRef)

	_, nonNullableRejectsNull := analyzeText(t, `User: Class<> := {}
parent: User := null`)
	requireSemanticCode(t, nonNullableRejectsNull, codeNullNotAllowed)
}

// TestConstantContractUnchangedByNullableSyntax covers 2.11: Constant must
// remain deeply frozen and never actually null, whether or not its declared
// type carries a `?`.
func TestConstantContractUnchangedByNullableSyntax(t *testing.T) {
	_, plainConstantNull := analyzeText(t, "value: Constant Int := null")
	requireSemanticCode(t, plainConstantNull, codeConstantInitializer)

	_, nullableConstantNull := analyzeText(t, "value: Constant Int? := null")
	requireSemanticCode(t, nullableConstantNull, codeConstantInitializer)

	_, nullableConstantValue := analyzeText(t, "value: Constant Int? := 5")
	requireSemanticClean(t, nullableConstantValue)
}

// TestFormatterIdempotenceForNestedNullableGenerics is a light formatter
// smoke test (the exhaustive formatter suite lives in internal/formatter);
// this just confirms the semantic layer accepts arbitrarily nested `?`
// compositions the formatter must also round-trip.
func TestNullableTypesAcceptArbitraryNesting(t *testing.T) {
	_, result := analyzeText(t, `x: List<List<Int?>?>? := null
y: Pair<String, List<Int?>?>? := null`)
	requireSemanticClean(t, result)
}

// TestInferredNullabilityMatchesInitializer is the exact matrix requested
// after the inference-forces-non-null bug was found: an inferred
// declaration's type, including nullability, must be exactly the
// initializer's own type -- never forced non-null, never widened to
// nullable merely because null exists (only a bare `null` literal, which
// carries no type at all, is rejected).
func TestInferredNullabilityMatchesInitializer(t *testing.T) {
	_, plain := analyzeText(t, "x := 5\nwrite(x)")
	requireSemanticClean(t, plain)

	_, nullableInferred := analyzeText(t, `User: Class<> := {}
fetchUser: Function := () -> User? {
    return null
}
user := fetchUser()
write(user == null)`)
	requireSemanticClean(t, nullableInferred)

	_, explicitNullableStillValid := analyzeText(t, `User: Class<> := {}
fetchUser: Function := () -> User? {
    return null
}
user: User? := fetchUser()`)
	requireSemanticClean(t, explicitNullableStillValid)

	_, explicitNonNullStillRejected := analyzeText(t, `User: Class<> := {}
fetchUser: Function := () -> User? {
    return null
}
user: User := fetchUser()`)
	requireSemanticCode(t, explicitNonNullStillRejected, codeNullNotAllowed)

	_, bareNullStillRejected := analyzeText(t, "value := null")
	requireSemanticCode(t, bareNullStillRejected, codeCannotInferType)
}

// TestInferredDeclarationScopeStaysExplicit is the v0.1.7 fix for the
// Local/Global inconsistency: inference must never silently bypass the
// existing "nested declaration requires Local" rule, and an explicit scope
// modifier before `:=` (no type) still infers only the value type.
func TestInferredDeclarationScopeStaysExplicit(t *testing.T) {
	_, moduleRootBareOK := analyzeText(t, "x := 5\nwrite(x)")
	requireSemanticClean(t, moduleRootBareOK)

	_, nestedBareRejected := analyzeText(t, `greet: Function := () -> Nothing {
    x := 5
    write(x)
}`)
	requireSemanticCode(t, nestedBareRejected, codeMissingLocal)

	_, nestedLocalInfersInt := analyzeText(t, `greet: Function := () -> Nothing {
    x: Local := 5
    write(x)
}`)
	requireSemanticClean(t, nestedLocalInfersInt)

	_, nestedLocalInfersNullable := analyzeText(t, `User: Class<> := {}
fetchUser: Function := () -> User? {
    return null
}
greet: Function := () -> Nothing {
    user: Local := fetchUser()
    write(user == null)
}`)
	requireSemanticClean(t, nestedLocalInfersNullable)

	_, localBareNullRejected := analyzeText(t, `greet: Function := () -> Nothing {
    user: Local := null
}`)
	requireSemanticCode(t, localBareNullRejected, codeCannotInferType)

	_, localExplicitNullableStillValid := analyzeText(t, `User: Class<> := {}
greet: Function := () -> Nothing {
    user: Local User? := null
    write(user == null)
}`)
	requireSemanticClean(t, localExplicitNullableStillValid)

	_, moduleRootLocalStillRejected := analyzeText(t, "x: Local := 5")
	requireSemanticCode(t, moduleRootLocalStillRejected, codeScopeModifier)

	_, inferredGlobalAdoptsModuleType := analyzeText(t, `counter: Int := 0
increase: Function := () -> Nothing {
    counter: Global
    counter = counter + 1
}
increase()`)
	requireSemanticClean(t, inferredGlobalAdoptsModuleType)
}
