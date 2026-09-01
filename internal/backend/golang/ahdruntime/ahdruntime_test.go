package ahdruntime

import (
	"archive/zip"
	"bufio"
	"bytes"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"
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
		AhdClassIndexError, AhdClassIOError, AhdClassKeyError, AhdClassNullError, AhdClassOverflowError, AhdClassValueError,
		AhdClassLatexError, AhdClassFileError, AhdClassWordError, AhdClassJSONError, AhdClassXMLError, AhdClassEnvError,
		AhdClassListsError,
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
	for text, want := range map[string]int64{
		"0": 0, "  +42\t": 42, "-42": -42,
		"9223372036854775807": math.MaxInt64, "-9223372036854775808": math.MinInt64,
	} {
		if got := AhdStringToInt(text); got != want {
			t.Fatalf("AhdStringToInt(%q) = %d, want %d", text, got, want)
		}
	}
	for _, text := range []string{"", "+", "3.0", "1e3", "1_0", "0x10", "１２"} {
		t.Run("invalid Int "+text, func(t *testing.T) {
			expectRaise(t, AhdClassDomainError, func() { AhdStringToInt(text) })
		})
	}
	expectRaise(t, AhdClassOverflowError, func() { AhdStringToInt("9223372036854775808") })
	expectRaise(t, AhdClassOverflowError, func() { AhdStringToInt("-9223372036854775809") })

	for text, want := range map[string]float64{
		"3": 3.0, "  +3.14\n": 3.14, "1e3": 1000.0, "1E+3": 1000.0, "-2.5e-4": -0.00025,
	} {
		if got := AhdStringToReal(text); got != want {
			t.Fatalf("AhdStringToReal(%q) = %g, want %g", text, got, want)
		}
	}
	for _, text := range []string{"", "+", ".5", "3.", "1e", "1_0", "0x10", "NaN", "Infinity", "inf"} {
		t.Run("invalid Real "+text, func(t *testing.T) {
			expectRaise(t, AhdClassDomainError, func() { AhdStringToReal(text) })
		})
	}
	expectRaise(t, AhdClassOverflowError, func() { AhdStringToReal("1e309") })
	expectRaise(t, AhdClassOverflowError, func() { AhdStringToReal("1e-4000") })
	if AhdStrInt(-7) != "-7" || AhdStrBool(true) != "true" || AhdStrString("x") != "x" {
		t.Fatal("scalar text is wrong")
	}
	if AhdRealAdd(1, 2) != 3 || AhdRealSubtract(3, 1) != 2 {
		t.Fatal("Real arithmetic is wrong")
	}
}

func TestListAddAndEjectMutateInPlace(t *testing.T) {
	values := AhdNewList[*int64](AhdBox(int64(10)), AhdBox(int64(20)))
	alias := values
	values.Add(AhdBox(int64(30)))
	if alias.Len() != 3 || *alias.At(2) != 30 {
		t.Fatal("add did not mutate the shared List")
	}
	values.Eject(1)
	if alias.Len() != 2 || *alias.At(1) != 30 {
		t.Fatal("eject did not remove the indexed element in place")
	}
	values.Eject(-1)
	if alias.Len() != 1 || *alias.At(0) != 10 {
		t.Fatal("negative eject did not remove the final element")
	}
	expectRaise(t, AhdClassIndexError, func() { values.Eject(5) })
	expectRaise(t, AhdClassIndexError, func() { AhdNewList[*int64]().Eject(0) })
	expectRaise(t, AhdClassNullError, func() {
		var absent *AhdList[*int64]
		absent.Add(nil)
	})
}

func TestPairEjectMutatesInPlaceAndKeepsOrder(t *testing.T) {
	scores := AhdNewPair[string, *int64]()
	scores.Set("Ali", AhdBox(int64(85)))
	scores.Set("Ayse", AhdBox(int64(92)))
	alias := scores
	scores.Eject("Ali")
	if alias.Len() != 1 || alias.Has("Ali") {
		t.Fatal("eject did not remove the key from the shared Pair")
	}
	scores.Set("Ali", AhdBox(int64(100)))
	if keys := alias.Keys(); strings.Join(keys, ",") != "Ayse,Ali" {
		t.Fatalf("a re-added key was not appended: %v", keys)
	}
	expectRaise(t, AhdClassKeyError, func() { scores.Eject("missing") })
	expectRaise(t, AhdClassNullError, func() {
		var absent *AhdPair[string, *int64]
		absent.Eject("a")
	})
}

func TestFrozenCollectionsRejectAddAndEject(t *testing.T) {
	values := AhdNewList[*int64](AhdBox(int64(1)))
	AhdFreeze(values)
	expectRaise(t, AhdClassConstantError, func() { values.Add(AhdBox(int64(2))) })
	expectRaise(t, AhdClassConstantError, func() { values.Eject(0) })
	if values.Len() != 1 {
		t.Fatal("a rejected mutation must not take effect")
	}

	scores := AhdBuildPair([]string{"a"}, []*int64{AhdBox(int64(1))})
	AhdFreeze(scores)
	expectRaise(t, AhdClassConstantError, func() { scores.Eject("a") })
	if scores.Len() != 1 {
		t.Fatal("a rejected Pair mutation must not take effect")
	}
}

// collectRange drains a lazy range so its yielded values can be compared.
func collectRange(iteration *AhdRange) []int64 {
	var values []int64
	for {
		value, ok := iteration.Next()
		if !ok {
			return values
		}
		values = append(values, value)
	}
}

func TestBetweenYieldsPythonStyleRanges(t *testing.T) {
	cases := []struct {
		name              string
		start, stop, step int64
		expected          []int64
	}{
		{"stop only", 0, 5, 1, []int64{0, 1, 2, 3, 4}},
		{"start and stop", 1, 5, 1, []int64{1, 2, 3, 4}},
		{"positive step", 0, 10, 2, []int64{0, 2, 4, 6, 8}},
		{"negative step", 5, 0, -1, []int64{5, 4, 3, 2, 1}},
		{"unreachable stop counting down", 0, 5, -1, nil},
		{"unreachable stop counting up", 5, 0, 1, nil},
		{"equal start and stop", 3, 3, 1, nil},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			values := collectRange(AhdBetween(testCase.start, testCase.stop, testCase.step))
			if len(values) != len(testCase.expected) {
				t.Fatalf("yielded %v; want %v", values, testCase.expected)
			}
			for index := range values {
				if values[index] != testCase.expected[index] {
					t.Fatalf("yielded %v; want %v", values, testCase.expected)
				}
			}
		})
	}
}

func TestBetweenRejectsAZeroStep(t *testing.T) {
	expectRaise(t, AhdClassDomainError, func() { AhdBetween(0, 10, 0) })
}

// TestBetweenTerminatesAtTheIntBoundaries checks that a step which would leave
// the signed 64-bit range ends the iteration instead of wrapping into an
// endless loop.
func TestBetweenTerminatesAtTheIntBoundaries(t *testing.T) {
	cases := []struct {
		name              string
		start, stop, step int64
		expected          []int64
	}{
		{"positive boundary", math.MaxInt64 - 2, math.MaxInt64, 1, []int64{math.MaxInt64 - 2, math.MaxInt64 - 1}},
		{"overflowing positive step", math.MaxInt64 - 1, math.MaxInt64, 2, []int64{math.MaxInt64 - 1}},
		{"negative boundary", math.MinInt64 + 2, math.MinInt64, -1, []int64{math.MinInt64 + 2, math.MinInt64 + 1}},
		{"underflowing negative step", math.MinInt64 + 1, math.MinInt64, -2, []int64{math.MinInt64 + 1}},
		{"start at the minimum", math.MinInt64, math.MinInt64 + 2, 1, []int64{math.MinInt64, math.MinInt64 + 1}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			values := collectRange(AhdBetween(testCase.start, testCase.stop, testCase.step))
			if len(values) != len(testCase.expected) {
				t.Fatalf("yielded %v; want %v", values, testCase.expected)
			}
			for index := range values {
				if values[index] != testCase.expected[index] {
					t.Fatalf("yielded %v; want %v", values, testCase.expected)
				}
			}
		})
	}
}

// TestBetweenIsLazy asserts the non-allocation contract directly: a range over
// the whole Int domain must be constructible and steppable, which is only
// possible because no values are materialized.
func TestBetweenIsLazy(t *testing.T) {
	iteration := AhdBetween(math.MinInt64, math.MaxInt64, 1)
	before := ahdHeapInUse()
	for index := 0; index < 1000; index++ {
		if _, ok := iteration.Next(); !ok {
			t.Fatal("a range over the whole Int domain ended early")
		}
	}
	if growth := ahdHeapInUse() - before; growth > 1<<20 {
		t.Fatalf("stepping a lazy range allocated %d bytes", growth)
	}
	// The state is only the current value, the stop, and the step.
	if unsafe.Sizeof(*iteration) > 32 {
		t.Fatalf("a lazy range carries %d bytes of state", unsafe.Sizeof(*iteration))
	}
}

func ahdHeapInUse() uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapInuse
}

// withTerminal redirects the runtime terminal streams for one read, so the two
// take forms can be exercised directly.
func withTerminal(t *testing.T, input string, body func()) string {
	t.Helper()
	previousIn, previousOut := ahdIn, ahdOut
	var captured strings.Builder
	ahdIn = bufio.NewReader(strings.NewReader(input))
	ahdOut = bufio.NewWriter(&captured)
	defer func() {
		ahdIn, ahdOut = previousIn, previousOut
	}()
	body()
	AhdFlush()
	return captured.String()
}

func TestTakeReadsOneLine(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"an LF terminator is removed", "Ali\n", "Ali"},
		{"a CRLF terminator is removed", "Ali\r\n", "Ali"},
		{"ordinary whitespace is preserved", "  Ali\t \n", "  Ali\t "},
		{"an empty line yields an empty String", "\n", ""},
		{"end of input yields an empty String", "", ""},
		{"a final line without a terminator is read", "Ali", "Ali"},
		{"only the first line is read", "one\ntwo\n", "one"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var value string
			written := withTerminal(t, testCase.input, func() { value = AhdTake() })
			if value != testCase.expected {
				t.Fatalf("take read %q; want %q", value, testCase.expected)
			}
			if written != "" {
				t.Fatalf("take without a prompt wrote %q", written)
			}
		})
	}
}

func TestTakePromptWritesWithoutANewline(t *testing.T) {
	var value string
	written := withTerminal(t, "Ali\n", func() { value = AhdTakePrompt("Name: ") })
	if written != "Name: " {
		t.Fatalf("prompt output was %q", written)
	}
	if value != "Ali" {
		t.Fatalf("prompted take read %q", value)
	}
	// The prompt is never part of the returned text.
	if strings.Contains(value, "Name") {
		t.Fatal("the prompt leaked into the returned String")
	}
}

func TestConsecutiveTakesReadConsecutiveLines(t *testing.T) {
	var first, second, third string
	withTerminal(t, "one\ntwo\n", func() {
		first, second, third = AhdTake(), AhdTake(), AhdTake()
	})
	if first != "one" || second != "two" || third != "" {
		t.Fatalf("consecutive reads were %q, %q, %q", first, second, third)
	}
}

func TestTakeFlushesPendingOutputBeforeReading(t *testing.T) {
	written := withTerminal(t, "x\n", func() {
		AhdWrite("before")
		_ = AhdTakePrompt("Prompt: ")
	})
	if written != "before\nPrompt: " {
		t.Fatalf("terminal output was %q", written)
	}
}

func TestAbsoluteValueKeepsCheckedArithmetic(t *testing.T) {
	if AhdAbsInt(5) != 5 || AhdAbsInt(-5) != 5 || AhdAbsInt(0) != 0 {
		t.Fatal("Int magnitude is wrong")
	}
	if AhdAbsInt(math.MinInt64+1) != math.MaxInt64 {
		t.Fatal("the magnitude near the Int minimum is wrong")
	}
	if AhdAbsReal(2.5) != 2.5 || AhdAbsReal(-2.5) != 2.5 {
		t.Fatal("Real magnitude is wrong")
	}
	if zero := AhdAbsReal(math.Copysign(0, -1)); math.Signbit(zero) {
		t.Fatal("the magnitude of -0.0 must be 0.0")
	}
	expectRaise(t, AhdClassOverflowError, func() { AhdAbsInt(math.MinInt64) })
}

func TestNumericListReductions(t *testing.T) {
	ints := AhdNewList(AhdBox(int64(8)), AhdBox(int64(3)), AhdBox(int64(12)))
	if AhdSumInt(ints) != 23 || AhdMinInt(ints) != 3 || AhdMaxInt(ints) != 12 {
		t.Fatal("Int reductions are wrong")
	}
	if ints.Len() != 3 {
		t.Fatal("a reduction must not modify its List")
	}
	reals := AhdNewList(AhdBox(3.5), AhdBox(-2.0), AhdBox(8.25))
	if AhdSumReal(reals) != 9.75 || AhdMinReal(reals) != -2.0 || AhdMaxReal(reals) != 8.25 {
		t.Fatal("Real reductions are wrong")
	}
	if AhdSumInt(AhdNewList[*int64]()) != 0 || AhdSumReal(AhdNewList[*float64]()) != 0.0 {
		t.Fatal("an empty List must sum to the additive identity")
	}
	expectRaise(t, AhdClassDomainError, func() { AhdMinInt(AhdNewList[*int64]()) })
	expectRaise(t, AhdClassDomainError, func() { AhdMaxReal(AhdNewList[*float64]()) })
	expectRaise(t, AhdClassNullError, func() { AhdSumInt(AhdNewList(AhdBox(int64(1)), nil)) })
	expectRaise(t, AhdClassNullError, func() { AhdMinReal(AhdNewList[*float64](nil)) })
	expectRaise(t, AhdClassOverflowError, func() {
		AhdSumInt(AhdNewList(AhdBox(int64(math.MaxInt64)), AhdBox(int64(1))))
	})
	expectRaise(t, AhdClassOverflowError, func() {
		AhdSumReal(AhdNewList(AhdBox(1.0e308), AhdBox(1.0e308)))
	})
}

func TestStringOperationsAreUnicodeAndImmutable(t *testing.T) {
	text := "  Ali Harun  "
	if AhdStringTrim(text) != "Ali Harun" || text != "  Ali Harun  " {
		t.Fatal("trim must not modify its receiver")
	}
	if AhdStringTrim("\t x \n") != "x" || AhdStringTrim(" x　") != "x" || AhdStringTrim("") != "" {
		t.Fatal("trim must remove Unicode whitespace at both ends only")
	}
	if AhdStringLower("AhdCode") != "ahdcode" || AhdStringUpper("AhdCode") != "AHDCODE" {
		t.Fatal("case conversion is wrong")
	}
	if AhdStringCapitalize("ali HARUN") != "Ali HARUN" || AhdStringCapitalize("aHD") != "AHD" || AhdStringCapitalize("") != "" {
		t.Fatal("capitalize must uppercase only the first character")
	}
	if AhdStringCapitalize("ünlü") != "Ünlü" {
		t.Fatal("capitalize must work on a multi-byte first character")
	}
	if AhdStringReplace("banana", "na", "X") != "baXX" || AhdStringReplace("abc", "b", "") != "ac" {
		t.Fatal("replace is wrong")
	}
	if !AhdStringStartsWith("abc", "") || !AhdStringStartsWith("abc", "ab") || AhdStringStartsWith("abc", "b") {
		t.Fatal("startsWith is wrong")
	}
	if !AhdStringEndsWith("abc", "") || !AhdStringEndsWith("abc", "bc") || AhdStringEndsWith("abc", "ab") {
		t.Fatal("endsWith is wrong")
	}
	if AhdStringCount("banana", "a") != 3 || AhdStringCount("banana", "na") != 2 || AhdStringCount("banana", "x") != 0 {
		t.Fatal("count must count non-overlapping occurrences")
	}
	if AhdStringIndex("banana", "na") != 2 || AhdStringIndex("a✓b✓", "✓") != 1 || AhdStringIndex("a✓b✓", "b") != 2 {
		t.Fatal("index must report a character index, not a byte offset")
	}
	expectRaise(t, AhdClassDomainError, func() { AhdStringSplit("abc", "") })
	expectRaise(t, AhdClassDomainError, func() { AhdStringReplace("abc", "", "x") })
	expectRaise(t, AhdClassDomainError, func() { AhdStringCount("abc", "") })
	expectRaise(t, AhdClassDomainError, func() { AhdStringIndex("abc", "") })
	expectRaise(t, AhdClassDomainError, func() { AhdStringIndex("abc", "z") })
}

func TestStringSplitPreservesEmptyFields(t *testing.T) {
	parts := AhdStringSplit("a,,b,", ",")
	if parts.Len() != 4 {
		t.Fatalf("split length = %d, want 4", parts.Len())
	}
	if parts.At(0) != "a" || parts.At(1) != "" || parts.At(2) != "b" || parts.At(3) != "" {
		t.Fatal("split must preserve empty fields")
	}
	single := AhdStringSplit("", ",")
	if single.Len() != 1 || single.At(0) != "" {
		t.Fatal("splitting an empty String must yield one empty field")
	}
}

func TestListReverseAndSearchOperations(t *testing.T) {
	list := AhdNewList(AhdBox(int64(5)), AhdBox(int64(7)), AhdBox(int64(5)))
	if AhdListCount(list, AhdBox(int64(5)), AhdEqNull(AhdEqInt)) != 2 {
		t.Fatal("count is wrong")
	}
	if AhdListIndex(list, AhdBox(int64(7)), AhdEqNull(AhdEqInt)) != 1 {
		t.Fatal("index must report the first match")
	}
	if list.Len() != 3 {
		t.Fatal("count and index must not modify the List")
	}
	expectRaise(t, AhdClassDomainError, func() {
		AhdListIndex(list, AhdBox(int64(99)), AhdEqNull(AhdEqInt))
	})
	list.Reverse()
	if *list.At(0) != 5 || *list.At(1) != 7 || *list.At(2) != 5 {
		t.Fatal("reverse is wrong")
	}
	ordered := AhdNewList(AhdBox(int64(3)), AhdBox(int64(1)), AhdBox(int64(2)))
	ordered.Reverse()
	if *ordered.At(0) != 2 || *ordered.At(1) != 1 || *ordered.At(2) != 3 {
		t.Fatal("reverse must reverse in place")
	}
}

func TestListSortIsStableAndAtomic(t *testing.T) {
	ints := AhdNewList(AhdBox(int64(8)), AhdBox(int64(3)), AhdBox(int64(12)))
	AhdListSortInt(ints)
	if *ints.At(0) != 3 || *ints.At(2) != 12 {
		t.Fatal("natural Int sort is wrong")
	}
	words := AhdNewList(AhdBox("pear"), AhdBox("apple"))
	AhdListSortString(words)
	if *words.At(0) != "apple" {
		t.Fatal("natural String sort is wrong")
	}
	reals := AhdNewList(AhdBox(3.5), AhdBox(-2.0))
	AhdListSortReal(reals)
	if *reals.At(0) != -2.0 {
		t.Fatal("natural Real sort is wrong")
	}
	withNull := AhdNewList(AhdBox(int64(3)), nil)
	expectRaise(t, AhdClassNullError, func() { AhdListSortInt(withNull) })
	if *withNull.At(0) != 3 || withNull.At(1) != nil {
		t.Fatal("a rejected sort must leave the List unchanged")
	}
	keys := 0
	values := AhdNewList(AhdBox(int64(3)), AhdBox(int64(1)), AhdBox(int64(2)))
	AhdListSortKeyInt(values, func(item *int64) *int64 {
		keys++
		negated := -*item
		return &negated
	})
	if keys != 3 {
		t.Fatalf("key evaluations = %d, want one per element", keys)
	}
	if *values.At(0) != 3 || *values.At(1) != 2 || *values.At(2) != 1 {
		t.Fatal("keyed sort is wrong")
	}
	expectRaise(t, AhdClassNullError, func() {
		AhdListSortKeyInt(values, func(item *int64) *int64 { return nil })
	})
	if *values.At(0) != 3 {
		t.Fatal("a raising key Function must leave the order unchanged")
	}
}

func TestListMapAndFilterUseASnapshot(t *testing.T) {
	source := AhdNewList(AhdBox(int64(1)), AhdBox(int64(2)), AhdBox(int64(3)))
	doubled := AhdListMap(source, func(item *int64) *int64 {
		source.Add(AhdBox(int64(99)))
		value := *item * 2
		return &value
	})
	if doubled.Len() != 3 || *doubled.At(0) != 2 || *doubled.At(2) != 6 {
		t.Fatal("map must iterate the snapshot taken at entry")
	}
	if source.Len() != 6 {
		t.Fatal("the callback's own mutations must still apply to the source")
	}
	texts := AhdListMap(AhdNewList(AhdBox(int64(1))), func(item *int64) *string {
		value := "n"
		return &value
	})
	if texts.Len() != 1 || *texts.At(0) != "n" {
		t.Fatal("map must produce the callback result type")
	}
	values := AhdNewList(AhdBox(int64(1)), AhdBox(int64(2)), AhdBox(int64(3)))
	evens := AhdListFilter(values, func(item *int64) *bool {
		keep := *item%2 == 0
		return &keep
	})
	if evens.Len() != 1 || *evens.At(0) != 2 || values.Len() != 3 {
		t.Fatal("filter must build a new List without mutating the source")
	}
	evens.Add(AhdBox(int64(9)))
	if evens.Len() != 2 {
		t.Fatal("a filtered List must be mutable")
	}
	expectRaise(t, AhdClassNullError, func() {
		AhdListFilter(values, func(item *int64) *bool { return nil })
	})
}

func TestHasMemberWalksTheRuntimeClassChain(t *testing.T) {
	person := &AhdClass{Name: "Person", Parent: AhdClassObject, Members: []string{"describe", "name"}}
	student := &AhdClass{Name: "Student", Parent: person, Members: []string{"number", "study"}}
	instance := &stubError{}
	instance.AhdSetClass(student)

	for _, name := range []string{"name", "describe", "number", "study"} {
		if !AhdHasMember(instance, name) {
			t.Fatalf("member %q must be reachable through the runtime Class chain", name)
		}
	}
	for _, name := range []string{"nickname", "Name", ""} {
		if AhdHasMember(instance, name) {
			t.Fatalf("member %q must not be reported", name)
		}
	}

	parent := &stubError{}
	parent.AhdSetClass(person)
	if !AhdHasMember(parent, "name") || AhdHasMember(parent, "number") {
		t.Fatal("a parent instance must not gain the members of a subclass")
	}

	// The built-in Error catalog publishes message through the Parent chain.
	failure := &stubError{}
	failure.AhdSetClass(AhdClassValueError)
	if !AhdHasMember(failure, "message") || AhdHasMember(failure, "code") {
		t.Fatal("a built-in Error must publish exactly message")
	}
	if AhdHasMember(nil, "name") {
		t.Fatal("a nil instance has no members")
	}
}

func TestLatexSourceHelpersAreDeterministic(t *testing.T) {
	plain := "Türkçe: ç ğ ı İ ö ş ü"
	if AhdLatexEscape(plain) != plain {
		t.Fatal("ordinary Unicode text must be preserved")
	}
	input := `\{}$&#%_^~`
	want := `\textbackslash{}\{\}\$\&\#\%\_\textasciicircum{}\textasciitilde{}`
	if result := AhdLatexEscape(input); result != want {
		t.Fatalf("Latex.escape = %q, want %q", result, want)
	}
	if result := AhdLatexSection("A&B"); result != "\\section{A\\&B}\n" {
		t.Fatalf("Latex.section = %q", result)
	}
	if result := AhdLatexSubsection("A_B"); result != "\\subsection{A\\_B}\n" {
		t.Fatalf("Latex.subsection = %q", result)
	}
	if result := AhdLatexEquation(`\sum_{k=1}^n k`); result != "\\begin{equation}\n\\sum_{k=1}^n k\n\\end{equation}\n" {
		t.Fatalf("Latex.equation = %q", result)
	}
	first := AhdLatexDocument("Body", "Türkçe & Math", "Ali")
	second := AhdLatexDocument("Body", "Türkçe & Math", "Ali")
	if first != second || !strings.Contains(first, "lmroman10-regular.otf") ||
		!strings.Contains(first, "\\title{Türkçe \\& Math}") || strings.Contains(first, "today") {
		t.Fatalf("Latex.document is not stable:\n%s", first)
	}
}

func TestLatexTableEscapesAndValidatesCells(t *testing.T) {
	headers := AhdNewList("Name", "A&B")
	rows := AhdNewList(AhdNewList("Ali", "1_2"))
	result := AhdLatexTable(headers, rows, AhdNewList[int64]())
	for _, expected := range []string{"\\begin{tabular}{ll}", "A\\&B", "1\\_2", "\\toprule", "\\midrule", "\\bottomrule"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("Latex.table omitted %q:\n%s", expected, result)
		}
	}
	expectRaise(t, AhdClassValueError, func() {
		AhdLatexTable(AhdNewList[string](), AhdNewList[*AhdList[string]](), AhdNewList[int64]())
	})
	expectRaise(t, AhdClassValueError, func() {
		AhdLatexTable(headers, AhdNewList(AhdNewList("only one")), AhdNewList[int64]())
	})
	expectRaise(t, AhdClassNullError, func() {
		AhdLatexTable(headers, AhdNewList[*AhdList[string]](nil), AhdNewList[int64]())
	})
}

func TestLatexInvocationUsesBundledArgvAndPublishesAtomically(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the argv-capture fixture is a POSIX test executable")
	}
	root := t.TempDir()
	capture := filepath.Join(root, "captured arguments")
	engine := filepath.Join(root, "tectonic")
	script := `#!/bin/sh
out=""
input=""
for argument do
    printf '%s\n' "$argument" >> "$AHD_LATEX_CAPTURE"
    if [ "$previous" = "--outdir" ]; then out="$argument"; fi
    previous="$argument"
    case "$argument" in *.tex) input="$argument";; esac
done
base=${input##*/}
base=${base%.*}
printf '%%PDF-fake' > "$out/$base.pdf"
`
	if err := os.WriteFile(engine, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(root, "ahdcode-latex.ttb")
	if err := os.WriteFile(bundle, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	t.Setenv("AHD_LATEX_CAPTURE", capture)
	outputDirectory := filepath.Join(t.TempDir(), "space ü $; &")
	if err := os.Mkdir(outputDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(outputDirectory, "result (safe).pdf")
	AhdLatexPDF("\\documentclass{article}\\begin{document}ok\\end{document}", output)
	content, err := os.ReadFile(output)
	if err != nil || string(content) != "%PDF-fake" {
		t.Fatalf("published output = %q, %v", content, err)
	}
	arguments, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	text := string(arguments)
	for _, required := range []string{"--untrusted\n", "--only-cached\n", "--bundle\n", bundle + "\n"} {
		if !strings.Contains(text, required) {
			t.Fatalf("engine argv omitted %q:\n%s", required, text)
		}
	}
	if strings.Contains(text, "shell-escape") {
		t.Fatalf("shell escape leaked into engine argv:\n%s", text)
	}
}

func TestLatexFailuresRaiseLatexErrorAndPreserveDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the failing engine fixture is a POSIX test executable")
	}
	root := t.TempDir()
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	expectRaise(t, AhdClassLatexError, func() { AhdLatexPDF("bad", filepath.Join(t.TempDir(), "x.pdf")) })

	if err := os.WriteFile(filepath.Join(root, "ahdcode-latex.ttb"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tectonic"), []byte("#!/bin/sh\nprintf 'error: invalid control sequence' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "existing.pdf")
	if err := os.WriteFile(destination, []byte("%PDF-existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectRaise(t, AhdClassLatexError, func() { AhdLatexPDF("bad", destination) })
	content, err := os.ReadFile(destination)
	if err != nil || string(content) != "%PDF-existing" {
		t.Fatalf("a failed compile changed the destination: %q, %v", content, err)
	}
}

func TestLatexTimeoutIsBoundedAndRaisesLatexError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the sleeping engine fixture is a POSIX test executable")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ahdcode-latex.ttb"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tectonic"), []byte("#!/bin/sh\nwhile :; do :; done\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	previous := ahdLatexCompileTimeout
	ahdLatexCompileTimeout = 20 * time.Millisecond
	t.Cleanup(func() { ahdLatexCompileTimeout = previous })
	started := time.Now()
	expectRaise(t, AhdClassLatexError, func() { AhdLatexPDF("loop", filepath.Join(t.TempDir(), "x.pdf")) })
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("timeout took %s", elapsed)
	}
}

func TestLatexTableMathColumns(t *testing.T) {
	headers := AhdNewList("Fonksiyon", "Türev", "Yorum")
	rows := AhdNewList(AhdNewList(
		`g(x)=e^{a(\ln x)^m}`, `\frac{1}{2}\star e^a`, "İlk & son"))

	// Without math columns every cell is escaped, exactly as before.
	text := AhdLatexTable(headers, rows, AhdNewList[int64]())
	for _, escaped := range []string{`\textbackslash{}ln`, `\textasciicircum{}`, `İlk \& son`} {
		if !strings.Contains(text, escaped) {
			t.Fatalf("text columns are no longer escaped, expected %q in:\n%s", escaped, text)
		}
	}
	if strings.Contains(text, `\(`) {
		t.Fatalf("a table without math columns must not open math mode:\n%s", text)
	}

	// A listed column is raw math: commands, braces, and scripts survive.
	text = AhdLatexTable(headers, rows, AhdNewList(int64(0), int64(1)))
	for _, expected := range []string{
		`\(g(x)=e^{a(\ln x)^m}\)`,
		`\(\frac{1}{2}\star e^a\)`,
		`İlk \& son`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %q in:\n%s", expected, text)
		}
	}
	if strings.Contains(text, `$$`) {
		t.Fatalf("math cells must use inline delimiters:\n%s", text)
	}

	// A repeated index selects the column once rather than nesting delimiters.
	repeated := AhdLatexTable(headers, rows,
		AhdNewList(int64(0), int64(0), int64(0)))
	once := AhdLatexTable(headers, rows, AhdNewList(int64(0)))
	if repeated != once {
		t.Fatalf("duplicate math columns are not idempotent:\n%s\n%s", repeated, once)
	}

	expectRaise(t, AhdClassValueError, func() {
		AhdLatexTable(headers, rows, AhdNewList(int64(-1)))
	})
	expectRaise(t, AhdClassValueError, func() {
		AhdLatexTable(headers, rows, AhdNewList(int64(3)))
	})
	expectRaise(t, AhdClassValueError, func() {
		AhdLatexTable(headers, rows, AhdNewList(int64(0), int64(9)))
	})
	// Row width validation is unchanged.
	expectRaise(t, AhdClassValueError, func() {
		short := AhdNewList(AhdNewList("only"))
		AhdLatexTable(headers, short, AhdNewList[int64]())
	})
}

func TestLatexTableTwoArgumentOutputIsUnchanged(t *testing.T) {
	headers := AhdNewList("A", "B")
	rows := AhdNewList(AhdNewList("x_1", "ç & ğ"))
	expected := "\\begin{tabular}{ll}\n\\toprule\nA & B \\\\\n\\midrule\n" +
		"x\\_1 & ç \\& ğ \\\\\n\\bottomrule\n\\end{tabular}\n"
	if text := AhdLatexTable(headers, rows, AhdNewList[int64]()); text != expected {
		t.Fatalf("two-argument table output changed:\n%q\nwant\n%q", text, expected)
	}
}

func TestLatexBeamerThemesAreBoundedAndComposeWithColor(t *testing.T) {
	theorems := AhdNewPair[string, string]()
	for _, theme := range []string{"Madrid", "Warsaw"} {
		source := AhdLatexDocumentFull("Body", "Title", "Author", "", "Beamer", 2.5, "#8A1538", "", theorems, theme)
		themeLine := `\usetheme{` + theme + `}`
		colorLine := `\definecolor{ahdaccent}{HTML}{8A1538}`
		if !strings.Contains(source, themeLine) || !strings.Contains(source, colorLine) {
			t.Fatalf("%s source omitted theme or color:\n%s", theme, source)
		}
		if strings.Index(source, themeLine) > strings.Index(source, colorLine) {
			t.Fatalf("custom color must follow and override %s:\n%s", theme, source)
		}
	}
	defaultSource := AhdLatexDocumentFull("Body", "", "", "", "Beamer", 2.5, "", "", theorems, "Default")
	if strings.Contains(defaultSource, `\usetheme{`) {
		t.Fatalf("Default unexpectedly emitted a named Beamer theme:\n%s", defaultSource)
	}
	expectRaise(t, AhdClassValueError, func() {
		AhdLatexDocumentFull("Body", "", "", "", "Beamer", 2.5, "", "", theorems, "Metropolis")
	})
	expectRaise(t, AhdClassValueError, func() {
		AhdLatexDocumentFull("Body", "", "", "", "Article", 2.5, "", "", theorems, "Madrid")
	})
}

func wordTestZIPEntry(t *testing.T, path, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer opened.Close()
		var content bytes.Buffer
		if _, err := content.ReadFrom(opened); err != nil {
			t.Fatal(err)
		}
		return content.Bytes()
	}
	t.Fatalf("DOCX entry %s is missing", name)
	return nil
}

func wordTestWriteZIP(t *testing.T, path string, entries [][2][]byte) {
	t.Helper()
	var content bytes.Buffer
	writer := zip.NewWriter(&content)
	for _, entry := range entries {
		part, err := writer.Create(string(entry[0]))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(entry[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWordDocumentCreationFormattingPageBreakAndDeterminism(t *testing.T) {
	base := AhdWordNew()
	left := AhdWordParagraph(base, "One", "left", false, false, false)
	right := AhdWordParagraph(base, "Two", "right", false, false, false)
	if AhdWordText(base) != "" || AhdWordText(left) != "One" || AhdWordText(right) != "Two" {
		t.Fatalf("Document derivation mutated a sibling: base=%q left=%q right=%q",
			AhdWordText(base), AhdWordText(left), AhdWordText(right))
	}

	document := AhdWordHeading(base, "A&B <Report>", 2)
	document = AhdWordParagraph(document, " formatted text ", "justify", true, true, true)
	document = AhdWordPageBreak(document)
	first := filepath.Join(t.TempDir(), "first.docx")
	second := filepath.Join(t.TempDir(), "second.docx")
	AhdWordSave(document, first)
	AhdWordSave(document, second)
	firstBytes, _ := os.ReadFile(first)
	secondBytes, _ := os.ReadFile(second)
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("saving the same Document twice produced different package bytes")
	}
	if err := ahdWordValidatePackage(firstBytes); err != nil {
		t.Fatalf("generated package is invalid: %v", err)
	}
	xmlText := string(wordTestZIPEntry(t, first, "word/document.xml"))
	for _, want := range []string{
		`<w:pStyle w:val="Heading2"/>`, `A&amp;B &lt;Report&gt;`,
		`<w:jc w:val="both"/>`, `<w:b/>`, `<w:i/>`, `<w:u w:val="single"/>`,
		`<w:br w:type="page"/>`,
	} {
		if !strings.Contains(xmlText, want) {
			t.Fatalf("document.xml omitted %q:\n%s", want, xmlText)
		}
	}
	if got := AhdWordHeadings(document).Snapshot(); len(got) != 1 || got[0] != "A&B <Report>" {
		t.Fatalf("headings = %v", got)
	}
	if got := AhdWordParagraphs(document).Snapshot(); len(got) != 1 || got[0] != " formatted text " {
		t.Fatalf("paragraphs = %v", got)
	}
	expectRaise(t, AhdClassWordError, func() { AhdWordHeading(base, "bad", 0) })
	expectRaise(t, AhdClassWordError, func() { AhdWordHeading(base, "bad", 7) })
	expectRaise(t, AhdClassWordError, func() { AhdWordParagraph(base, "bad", "middle", false, false, false) })
	expectRaise(t, AhdClassWordError, func() { AhdWordSave(base, "report.txt") })
}

func TestWordTablesValidateMergesAndCopyCallerLists(t *testing.T) {
	headers := AhdNewList("A", "B", "C")
	firstRow := AhdNewList("1", "2", "3")
	secondRow := AhdNewList("4", "5", "6")
	rows := AhdNewList(firstRow, secondRow)
	horizontal := AhdNewList(int64(0), int64(0), int64(1), int64(2))
	vertical := AhdNewList(int64(1), int64(2), int64(2), int64(1))
	merges := AhdNewList(horizontal, vertical)
	document := AhdWordTable(AhdWordNew(), headers, rows, merges, "center")

	if got := headers.Snapshot(); strings.Join(got, ",") != "A,B,C" {
		t.Fatalf("table mutated headers: %v", got)
	}
	if got := firstRow.Snapshot(); strings.Join(got, ",") != "1,2,3" {
		t.Fatalf("table mutated a row: %v", got)
	}
	if got := horizontal.Snapshot(); len(got) != 4 || got[3] != 2 {
		t.Fatalf("table mutated a merge descriptor: %v", got)
	}
	headers.Set(0, "changed")
	firstRow.Set(0, "changed")
	horizontal.Set(3, 3)
	tables := AhdWordTables(document).Snapshot()
	grid := tables[0].Snapshot()
	if grid[0].Snapshot()[0] != "A" || grid[1].Snapshot()[0] != "1" {
		t.Fatalf("Document retained caller List aliases: %v", tables)
	}

	path := filepath.Join(t.TempDir(), "merges.docx")
	AhdWordSave(document, path)
	xmlText := string(wordTestZIPEntry(t, path, "word/document.xml"))
	for _, want := range []string{
		`<w:jc w:val="center"/>`, `<w:gridSpan w:val="2"/>`,
		`<w:vMerge w:val="restart"/>`, `<w:vMerge/>`,
	} {
		if !strings.Contains(xmlText, want) {
			t.Fatalf("merged table omitted %q:\n%s", want, xmlText)
		}
	}
	dataMerge := AhdWordTable(AhdWordNew(), AhdNewList("A", "B", "C"),
		AhdNewList(AhdNewList("1", "2", "3")),
		AhdNewList(AhdNewList(int64(1), int64(0), int64(1), int64(2))), "left")
	dataMergePath := filepath.Join(t.TempDir(), "data-merge.docx")
	AhdWordSave(dataMerge, dataMergePath)
	// A horizontal merge in a data row reads back as fewer physical cells than
	// the header. Re-saving that bounded semantic reading must stay safe.
	AhdWordSave(AhdWordRead(dataMergePath), filepath.Join(t.TempDir(), "resaved.docx"))

	validHeaders := AhdNewList("A", "B")
	validRows := AhdNewList(AhdNewList("1", "2"))
	invalid := []*AhdList[*AhdList[int64]]{
		AhdNewList(AhdNewList(int64(0), 0, 1)),
		AhdNewList(AhdNewList(int64(-1), 0, 1, 2)),
		AhdNewList(AhdNewList(int64(0), 0, 0, 2)),
		AhdNewList(AhdNewList(int64(0), 0, 1, 1)),
		AhdNewList(AhdNewList(int64(0), 1, 1, 2)),
		AhdNewList(AhdNewList(int64(0), 0, 1, 2), AhdNewList(int64(0), 1, 2, 1)),
	}
	for _, descriptors := range invalid {
		descriptors := descriptors
		expectRaise(t, AhdClassWordError, func() {
			AhdWordTable(AhdWordNew(), validHeaders, validRows, descriptors, "left")
		})
	}
	expectRaise(t, AhdClassWordError, func() {
		AhdWordTable(AhdWordNew(), AhdNewList[string](), AhdNewList[*AhdList[string]](), AhdNewList[*AhdList[int64]](), "left")
	})
	expectRaise(t, AhdClassWordError, func() {
		AhdWordTable(AhdWordNew(), validHeaders, AhdNewList(AhdNewList("short")), AhdNewList[*AhdList[int64]](), "left")
	})
	expectRaise(t, AhdClassWordError, func() {
		AhdWordTable(AhdWordNew(), validHeaders, validRows, AhdNewList[*AhdList[int64]](), "justify")
	})
}

func TestWordImagesEmbedBytesAndResolveSizing(t *testing.T) {
	directory := t.TempDir()
	imagePath := filepath.Join(directory, "source.png")
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 200, 100))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	size := AhdNewPair[string, float64]()
	size.Set("width", 10)
	document := AhdWordImage(AhdWordNew(), imagePath, size)
	if err := os.Remove(imagePath); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "image.docx")
	AhdWordSave(document, path)
	secondPath := filepath.Join(directory, "image-second.docx")
	AhdWordSave(document, secondPath)
	firstPackage, _ := os.ReadFile(path)
	secondPackage, _ := os.ReadFile(secondPath)
	if !bytes.Equal(firstPackage, secondPackage) {
		t.Fatal("image relationship IDs or media names are nondeterministic")
	}
	if got := wordTestZIPEntry(t, path, "word/media/image1.png"); !bytes.Equal(got, encoded.Bytes()) {
		t.Fatal("embedded image bytes differ from the source snapshot")
	}
	relationships := string(wordTestZIPEntry(t, path, "word/_rels/document.xml.rels"))
	if !strings.Contains(relationships, `Id="rId2"`) || !strings.Contains(relationships, `Target="media/image1.png"`) {
		t.Fatalf("image relationship is not deterministic:\n%s", relationships)
	}
	xmlText := string(wordTestZIPEntry(t, path, "word/document.xml"))
	if !strings.Contains(xmlText, `cx="3600000" cy="1800000"`) {
		t.Fatalf("width-only sizing did not preserve the 2:1 aspect ratio:\n%s", xmlText)
	}

	both := AhdNewPair[string, float64]()
	both.Set("width", 4)
	both.Set("height", 3)
	if width, height := ahdWordImageExtent(both, 200, 100); width != 1440000 || height != 1080000 {
		t.Fatalf("explicit image extent = %d x %d", width, height)
	}
	heightOnly := AhdNewPair[string, float64]()
	heightOnly.Set("height", 2)
	if width, height := ahdWordImageExtent(heightOnly, 200, 100); width != 1440000 || height != 720000 {
		t.Fatalf("height-only image extent = %d x %d", width, height)
	}
	natural := AhdNewPair[string, float64]()
	if width, height := ahdWordImageExtent(natural, 200, 100); width != 1905000 || height != 952500 {
		t.Fatalf("natural image extent = %d x %d", width, height)
	}
	badKey := AhdNewPair[string, float64]()
	badKey.Set("depth", 1)
	expectRaise(t, AhdClassWordError, func() { ahdWordImageExtent(badKey, 1, 1) })
	badWidth := AhdNewPair[string, float64]()
	badWidth.Set("width", 0)
	expectRaise(t, AhdClassWordError, func() { ahdWordImageExtent(badWidth, 1, 1) })
}

func TestWordReadRoundTripForeignDocumentAndSecurityBounds(t *testing.T) {
	directory := t.TempDir()
	generated := filepath.Join(directory, "roundtrip.docx")
	document := AhdWordHeading(AhdWordNew(), "Başlık", 1)
	document = AhdWordParagraph(document, "Türkçe içerik", "left", false, false, false)
	document = AhdWordTable(document, AhdNewList("Ad", "Puan"), AhdNewList(AhdNewList("Ali", "91")), AhdNewList[*AhdList[int64]](), "left")
	AhdWordSave(document, generated)
	loaded := AhdWordRead(generated)
	if got := AhdWordText(loaded); got != "Başlık\nTürkçe içerik" {
		t.Fatalf("round-trip text = %q", got)
	}
	if tables := AhdWordTables(loaded).Snapshot(); len(tables) != 1 || tables[0].Snapshot()[1].Snapshot()[1] != "91" {
		t.Fatalf("round-trip tables = %v", tables)
	}

	foreign := filepath.Join(directory, "foreign.docx")
	foreignXML := []byte(`<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:pPr><w:pStyle w:val="Heading 3"/></w:pPr><w:r><w:t>Independent</w:t></w:r></w:p><w:p><w:r><w:t>one</w:t></w:r><w:r><w:tab/><w:t>two</w:t><w:br/><w:t>three</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`)
	wordTestWriteZIP(t, foreign, [][2][]byte{{[]byte("word/document.xml"), foreignXML}})
	foreignDocument := AhdWordRead(foreign)
	if got := AhdWordHeadings(foreignDocument).Snapshot(); len(got) != 1 || got[0] != "Independent" {
		t.Fatalf("foreign headings = %v", got)
	}
	if got := AhdWordText(foreignDocument); got != "Independent\none\ttwo\nthree" {
		t.Fatalf("foreign text = %q", got)
	}

	notZIP := filepath.Join(directory, "not.docx")
	if err := os.WriteFile(notZIP, []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(directory, "missing.docx")
	wordTestWriteZIP(t, missing, [][2][]byte{{[]byte("other.xml"), []byte("<x/>")}})
	unsafe := filepath.Join(directory, "unsafe.docx")
	wordTestWriteZIP(t, unsafe, [][2][]byte{
		{[]byte("../../../etc/evil.xml"), []byte("bad")},
		{[]byte("word/document.xml"), foreignXML},
	})
	duplicate := filepath.Join(directory, "duplicate.docx")
	wordTestWriteZIP(t, duplicate, [][2][]byte{
		{[]byte("word/document.xml"), foreignXML},
		{[]byte("word/document.xml"), foreignXML},
	})
	bomb := filepath.Join(directory, "bomb.docx")
	wordTestWriteZIP(t, bomb, [][2][]byte{{[]byte("word/document.xml"), bytes.Repeat([]byte("0"), 1024*1024)}})
	for _, path := range []string{"", filepath.Join(directory, "absent.docx"), notZIP, missing, unsafe, duplicate, bomb} {
		path := path
		expectRaise(t, AhdClassWordError, func() { AhdWordRead(path) })
	}
}

// TestSubMinuteLocalOffsetKeepsTheInstantExact pins the historical-offset rule.
// A few host zones sit at a UTC offset that is not a whole number of minutes
// (Europe/Istanbul is +01:55:52 before 1880). AhdCode publishes offsetMinutes
// as whole minutes, so the leftover seconds are carried as runtime
// representation: truncating them would silently move the instant, and
// rejecting them would refuse ordinary historical dates. This test is
// host-timezone independent because it builds the zone explicitly.
func TestSubMinuteLocalOffsetKeepsTheInstantExact(t *testing.T) {
	for _, testCase := range []struct {
		name                     string
		offsetSeconds            int
		wantMinutes, wantSeconds int64
	}{
		{"istanbul LMT", 6952, 115, 52},
		{"west of UTC", -6952, -115, -52},
		{"whole minute", 10800, 180, 0},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			zone := time.FixedZone("", testCase.offsetSeconds)
			original := time.Date(999, 12, 31, 23, 59, 59, 250*1e6, zone)
			civil := ahdCivilFrom(original)
			if civil.OffsetMinutes != testCase.wantMinutes || civil.OffsetSeconds != testCase.wantSeconds {
				t.Fatalf("offset = %dm %ds; want %dm %ds",
					civil.OffsetMinutes, civil.OffsetSeconds, testCase.wantMinutes, testCase.wantSeconds)
			}
			// The published civil fields are never shifted to absorb the remainder.
			if civil.Hour != 23 || civil.Minute != 59 || civil.Second != 59 || civil.Millisecond != 250 {
				t.Fatalf("civil clock fields changed: %+v", civil)
			}
			// minutes*60+seconds must reproduce the original offset exactly, so
			// the rebuilt instant equals the one we started from.
			if rebuilt := AhdTimeInstantCivil(civil); !rebuilt.Equal(original) {
				t.Fatalf("instant shifted by %v", rebuilt.Sub(original))
			}
		})
	}
}
