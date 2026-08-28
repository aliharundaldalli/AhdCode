package ahdruntime

import (
	"math"
	"strings"
	"testing"
)

// stubError stands in for a generated Error Class so the runtime can be tested
// without a generated program.
type stubError struct {
	AhdBase
	message string
}

func (value *stubError) AhdErrorMessage() string { return value.message }

func (value *stubError) AhdFreezeGraph(visited map[AhdFreezable]bool) {
	if !AhdEnterFreeze(value, visited) {
		return
	}
	value.AhdMarkFrozen()
}

func init() {
	for _, class := range []*AhdClass{
		AhdClassError, AhdClassConstantError, AhdClassDivisionByZeroError, AhdClassDomainError,
		AhdClassIndexError, AhdClassKeyError, AhdClassNullError, AhdClassOverflowError, AhdClassValueError,
	} {
		target := class
		AhdRegisterError(target, func(message string) AhdInstance {
			instance := &stubError{message: message}
			instance.AhdSetClass(target)
			return instance
		})
	}
}

// cyclicNode builds a reference cycle so the freeze walk can be shown to
// terminate.
type cyclicNode struct {
	AhdBase
	peer AhdFreezable
}

func (node *cyclicNode) AhdFreezeGraph(visited map[AhdFreezable]bool) {
	if !AhdEnterFreeze(node, visited) {
		return
	}
	node.AhdMarkFrozen()
	AhdFreezeChild(node.peer, visited)
}

func expectRaise(t *testing.T, class *AhdClass, body func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected an AhdCode %s", class.Name)
		}
		signal, ok := recovered.(*AhdSignal)
		if !ok {
			t.Fatalf("expected an AhdSignal; received %v", recovered)
		}
		if signal.Instance.AhdClassOf() != class {
			t.Fatalf("expected %s; received %s", class.Name, signal.Instance.AhdClassOf().Name)
		}
	}()
	body()
}

func TestCheckedIntArithmeticRejectsOverflow(t *testing.T) {
	if AhdIntAdd(2, 3) != 5 || AhdIntSubtract(2, 3) != -1 || AhdIntMultiply(6, 7) != 42 {
		t.Fatal("ordinary Int arithmetic is wrong")
	}
	if AhdIntNegate(math.MinInt64+1) != math.MaxInt64 {
		t.Fatal("Int negation is wrong")
	}
	expectRaise(t, AhdClassOverflowError, func() { AhdIntAdd(math.MaxInt64, 1) })
	expectRaise(t, AhdClassOverflowError, func() { AhdIntSubtract(math.MinInt64, 1) })
	expectRaise(t, AhdClassOverflowError, func() { AhdIntMultiply(math.MaxInt64, 2) })
	expectRaise(t, AhdClassOverflowError, func() { AhdIntMultiply(math.MinInt64, -1) })
	expectRaise(t, AhdClassOverflowError, func() { AhdIntNegate(math.MinInt64) })
	expectRaise(t, AhdClassOverflowError, func() { AhdIntPower(10, 100) })
}

func TestIntModuloAndPowerEdges(t *testing.T) {
	if AhdIntModulo(7, 3) != 1 || AhdIntModulo(-7, 3) != -1 {
		t.Fatal("Int modulo does not follow truncated remainder semantics")
	}
	if AhdIntModulo(-5, 2) != -1 || AhdIntModulo(5, -2) != 1 || AhdIntModulo(-5, -2) != -1 {
		t.Fatal("Int modulo must keep the dividend sign")
	}
	if AhdIntModulo(math.MinInt64, -1) != 0 {
		t.Fatal("the signed minimum modulo -1 must be 0")
	}
	if AhdIntPower(2, 10) != 1024 || AhdIntPower(5, 0) != 1 {
		t.Fatal("Int power is wrong")
	}
	expectRaise(t, AhdClassDivisionByZeroError, func() { AhdIntModulo(1, 0) })
	expectRaise(t, AhdClassDomainError, func() { AhdIntPower(2, -1) })
}

func TestRealArithmeticRejectsNonFiniteResults(t *testing.T) {
	if AhdRealDivide(5, 2) != 2.5 {
		t.Fatal("Real division is wrong")
	}
	expectRaise(t, AhdClassDivisionByZeroError, func() { AhdRealDivide(1, 0) })
	expectRaise(t, AhdClassOverflowError, func() { AhdRealMultiply(math.MaxFloat64, 10) })
	expectRaise(t, AhdClassDomainError, func() { AhdRealPower(-1, 0.5) })
}

func TestCanonicalRealText(t *testing.T) {
	cases := map[float64]string{
		5:                    "5.0",
		2.5:                  "2.5",
		math.Copysign(0, -1): "-0.0",
		1000000:              "1000000.0",
		1e21:                 "1e+21",
		1e-7:                 "1e-07",
		1.0 / 3.0:            "0.3333333333333333",
	}
	for value, expected := range cases {
		if AhdStrReal(value) != expected {
			t.Fatalf("AhdStrReal(%v) = %q; want %q", value, AhdStrReal(value), expected)
		}
	}
}

func TestCanonicalCollectionText(t *testing.T) {
	list := AhdNewList[*int64](AhdBox(int64(1)), nil, AhdBox(int64(3)))
	if got := AhdStrList[*int64](AhdStrNull[int64](AhdStrInt))(list); got != "[1, null, 3]" {
		t.Fatalf("List text is %q", got)
	}
	strings_ := AhdNewList[*string](AhdBox("Ali"), AhdBox("A\"y\\e"))
	if got := AhdStrList[*string](AhdStrNull[string](AhdStrQuoted))(strings_); got != `["Ali", "A\"y\\e"]` {
		t.Fatalf("nested String text is %q", got)
	}
	pair := AhdBuildPair([]string{"Ali", "Ayşe"}, []*int64{AhdBox(int64(90)), AhdBox(int64(95))})
	if got := AhdStrPair[string, *int64](AhdStrQuoted, AhdStrNull[int64](AhdStrInt))(pair); got != `{"Ali": 90, "Ayşe": 95}` {
		t.Fatalf("Pair text is %q", got)
	}
	if AhdStrFunction("double") != "<Function double>" {
		t.Fatal("Function text is wrong")
	}
	student := &stubError{}
	student.AhdSetClass(&AhdClass{Name: "Student"})
	if AhdStrRefInstance[AhdInstance](student) != "<Student>" || AhdStrRefInstance[AhdInstance](nil) != "null" {
		t.Fatal("Class instance text is wrong")
	}
}

func TestListKeepsIdentityAcrossAliases(t *testing.T) {
	list := AhdNewList[*int64](AhdBox(int64(1)), AhdBox(int64(2)))
	alias := list
	list.Clear()
	if alias.Len() != 0 || list != alias {
		t.Fatal("clear did not empty the List in place")
	}
	list = AhdNewList[*int64](AhdBox(int64(1)), AhdBox(int64(2)), AhdBox(int64(3)))
	if *list.At(-1) != 3 || *list.At(0) != 1 {
		t.Fatal("negative indexing is wrong")
	}
	list.Set(-1, AhdBox(int64(9)))
	if *list.At(2) != 9 {
		t.Fatal("indexed assignment is wrong")
	}
	if list.Slice(1, true, 0, false).Len() != 2 || list.Slice(0, false, 2, true).Len() != 2 {
		t.Fatal("slicing is wrong")
	}
	snapshot := list.Snapshot()
	list.Clear()
	if len(snapshot) != 3 {
		t.Fatal("the iteration snapshot is not shallow-copied")
	}
	expectRaise(t, AhdClassIndexError, func() { AhdNewList[*int64]().At(0) })
	expectRaise(t, AhdClassNullError, func() { var absent *AhdList[*int64]; absent.Len() })
}

func TestPairPreservesInsertionOrder(t *testing.T) {
	pair := AhdNewPair[string, *int64]()
	pair.Set("b", AhdBox(int64(1)))
	pair.Set("a", AhdBox(int64(2)))
	pair.Set("b", AhdBox(int64(3)))
	if keys := pair.Keys(); strings.Join(keys, ",") != "b,a" {
		t.Fatalf("updating a key moved it: %v", keys)
	}
	pair.Remove("b")
	pair.Set("b", AhdBox(int64(4)))
	if keys := pair.Keys(); strings.Join(keys, ",") != "a,b" {
		t.Fatalf("a re-added key was not appended: %v", keys)
	}
	if !pair.Has("a") || pair.Has("z") {
		t.Fatal("key membership is wrong")
	}
	alias := pair
	pair.Clear()
	if alias.Len() != 0 {
		t.Fatal("clear did not empty the Pair in place")
	}
	expectRaise(t, AhdClassKeyError, func() { pair.Get("missing") })
	expectRaise(t, AhdClassNullError, func() { var absent *AhdPair[string, *int64]; absent.Len() })
}

func TestStringOperationsAreCharacterBased(t *testing.T) {
	if AhdStringLen("añb") != 3 {
		t.Fatal("len counts bytes instead of characters")
	}
	if AhdStringAt("añb", 1) != "ñ" || AhdStringAt("añb", -1) != "b" {
		t.Fatal("String indexing is wrong")
	}
	if AhdStringSlice("añbc", 1, true, 3, true) != "ñb" {
		t.Fatal("String slicing is wrong")
	}
	if strings.Join(AhdStringChars("añb"), "|") != "a|ñ|b" {
		t.Fatal("String iteration is wrong")
	}
	if AhdStringRepeat("ab", 3) != "ababab" {
		t.Fatal("String repeat is wrong")
	}
	if !AhdStringContains("cat", "a") || AhdStringContains("cat", "z") {
		t.Fatal("String membership is wrong")
	}
	expectRaise(t, AhdClassValueError, func() { AhdStringRepeat("a", -1) })
	expectRaise(t, AhdClassIndexError, func() { AhdStringAt("a", 5) })
}

func TestNullBoxing(t *testing.T) {
	if *AhdBox(int64(5)) != 5 || AhdNonNull(AhdBox("x")) != "x" {
		t.Fatal("boxing is wrong")
	}
	expectRaise(t, AhdClassNullError, func() { AhdNonNull[int64](nil) })
}

func TestEqualityHelpers(t *testing.T) {
	left := AhdNewList[*int64](AhdBox(int64(1)), nil)
	right := AhdNewList[*int64](AhdBox(int64(1)), nil)
	if !AhdEqList[*int64](AhdEqNull[int64](AhdEqInt))(left, right) {
		t.Fatal("deep List equality is wrong")
	}
	right.Set(1, AhdBox(int64(2)))
	if AhdEqList[*int64](AhdEqNull[int64](AhdEqInt))(left, right) {
		t.Fatal("deep List equality ignored a difference")
	}
	first := AhdBuildPair([]string{"a", "b"}, []*int64{AhdBox(int64(1)), AhdBox(int64(2))})
	second := AhdBuildPair([]string{"b", "a"}, []*int64{AhdBox(int64(2)), AhdBox(int64(1))})
	if !AhdEqPair[string, *int64](AhdEqNull[int64](AhdEqInt))(first, second) {
		t.Fatal("deep Pair equality must ignore order")
	}
	if AhdSameDifferent(int64(5), 5.0) {
		t.Fatal("statically distinct types can never be the same value")
	}
	person := &AhdClass{Name: "Person"}
	value := &stubError{}
	value.AhdSetClass(person)
	other := &stubError{}
	other.AhdSetClass(person)
	if !AhdEqRef[AhdInstance]()(value, value) || AhdEqRef[AhdInstance]()(value, other) {
		t.Fatal("reference identity is wrong")
	}
	if !AhdIsClass(value, person) || AhdIsClass(nil, person) {
		t.Fatal("Class membership is wrong")
	}
	if !AhdConstBool(1, true) || AhdConstBool(1, false) {
		t.Fatal("statically resolved Bool results are wrong")
	}
	if !AhdListContains(left, AhdBox(int64(1)), AhdEqNull[int64](AhdEqInt)) {
		t.Fatal("List membership is wrong")
	}
	if AhdListConcat(left, right).Len() != 4 {
		t.Fatal("List concatenation is wrong")
	}
}

func TestClassMembershipFollowsInheritance(t *testing.T) {
	animal := &AhdClass{Name: "Animal"}
	dog := &AhdClass{Name: "Dog", Parent: animal}
	instance := &stubError{}
	instance.AhdSetClass(dog)
	if !AhdIsClass(instance, dog) || !AhdIsClass(instance, animal) {
		t.Fatal("inheritance-aware Class membership is wrong")
	}
	if AhdIsClass(instance, &AhdClass{Name: "Cat"}) {
		t.Fatal("unrelated Class membership must be false")
	}
	other := &stubError{}
	other.AhdSetClass(animal)
	if AhdSameInstance(instance, other) || !AhdSameInstance(instance, instance) {
		t.Fatal("same must compare exact runtime Class and object identity")
	}
	if !AhdEqInstance(instance, instance) || AhdEqInstance(instance, other) {
		t.Fatal("Class equality must be reference identity")
	}
}

func TestErrorSignalsAreIsolatedFromGoPanics(t *testing.T) {
	instance := &stubError{message: "boom"}
	instance.AhdSetClass(AhdClassError)
	expectRaise(t, AhdClassError, func() { AhdToss(instance) })
	expectRaise(t, AhdClassNullError, func() { AhdToss(nil) })

	signal := &AhdSignal{Instance: instance, Message: "boom"}
	if !AhdMatches(signal, AhdClassError) || AhdMatches(signal, AhdClassKeyError) {
		t.Fatal("except matching is wrong")
	}
	if AhdSignalOf(nil) != nil {
		t.Fatal("no recovered value must yield no signal")
	}
	if got := AhdSignalOf(signal); got != signal {
		t.Fatal("an AhdCode signal must be recovered as itself")
	}
	// An ordinary Go panic is never handled as an AhdCode Error.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("an ordinary Go panic must not be converted into an AhdCode error")
			}
		}()
		_ = AhdSignalOf("ordinary go panic")
	}()
	if signal.Error() != "Error: boom" {
		t.Fatalf("signal text is %q", signal.Error())
	}
}

func TestDeepFreezeIsIdempotentAndTerminatesOnCycles(t *testing.T) {
	inner := AhdNewList[*int64](AhdBox(int64(1)))
	outer := AhdNewList[*AhdList[*int64]](inner)
	AhdFreeze(outer)
	expectRaise(t, AhdClassConstantError, func() { inner.Clear() })
	expectRaise(t, AhdClassConstantError, func() { outer.Clear() })
	// Freezing again must not fail or loop.
	AhdFreeze(outer)

	first, second := &cyclicNode{}, &cyclicNode{}
	first.AhdSetClass(AhdClassObject)
	second.AhdSetClass(AhdClassObject)
	first.peer, second.peer = second, first
	AhdFreeze(first)
	AhdFreeze(first)
	if !first.AhdFrozen() || !second.AhdFrozen() {
		t.Fatal("a cyclic graph was not fully frozen")
	}

	pair := AhdBuildPair([]string{"a"}, []*AhdList[*int64]{AhdNewList[*int64]()})
	AhdFreeze(pair)
	expectRaise(t, AhdClassConstantError, func() { pair.Set("b", nil) })
	expectRaise(t, AhdClassConstantError, func() { pair.Remove("a") })
	expectRaise(t, AhdClassConstantError, func() { pair.Clear() })

	instance := &stubError{}
	instance.AhdSetClass(AhdClassError)
	AhdFreeze(instance)
	if !instance.AhdFrozen() {
		t.Fatal("a Class instance was not frozen")
	}
	expectRaise(t, AhdClassConstantError, func() { instance.AhdRequireMutable() })
	if instance.AhdMarkFrozen() {
		t.Fatal("freezing is not idempotent")
	}
}

func TestFrozenListRejectsIndexedWrites(t *testing.T) {
	values := AhdNewList[*int64](AhdBox(int64(1)))
	AhdFreeze(values)
	expectRaise(t, AhdClassConstantError, func() { values.Set(0, AhdBox(int64(2))) })
	if *values.At(0) != 1 {
		t.Fatal("a rejected mutation must not take effect")
	}
}

func TestNumericConversions(t *testing.T) {
	if AhdIntToReal(5) != 5.0 || AhdRealToInt(3.7) != 3 || AhdRealToInt(-3.7) != -3 || AhdRealNegate(2.5) != -2.5 {
		t.Fatal("Real conversion helpers are wrong")
	}
	expectRaise(t, AhdClassOverflowError, func() { AhdRealToInt(math.Inf(1)) })
	expectRaise(t, AhdClassDomainError, func() { AhdRealToInt(math.NaN()) })
	if AhdStrInt(-7) != "-7" || AhdStrBool(true) != "true" || AhdStrString("x") != "x" {
		t.Fatal("scalar text is wrong")
	}
	if AhdRealAdd(1, 2) != 3 || AhdRealSubtract(3, 1) != 2 {
		t.Fatal("Real arithmetic is wrong")
	}
}
