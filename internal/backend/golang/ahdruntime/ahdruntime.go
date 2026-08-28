// Package ahdruntime is the AhdCode v0.1 Go backend runtime.
//
// This file is compiled twice: once as part of the compiler (so ordinary Go
// tooling checks it) and once as generated program source, where the package
// clause is rewritten to main. It must therefore depend only on the Go
// standard library and must not reference any other AhdCode package.
package ahdruntime

import (
	"bufio"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// ---------------------------------------------------------------------------
// Class identity
// ---------------------------------------------------------------------------

// AhdClass is the canonical runtime identity of one AhdCode Class. Descriptors
// are compared by pointer, never by name.
type AhdClass struct {
	Name   string
	Parent *AhdClass
}

// The language-supplied Class catalog. Generated code aliases these
// descriptors so a runtime check and an AhdCode except clause agree on
// identity.
var (
	AhdClassObject              = &AhdClass{Name: "Object"}
	AhdClassError               = &AhdClass{Name: "Error", Parent: AhdClassObject}
	AhdClassConstantError       = &AhdClass{Name: "ConstantError", Parent: AhdClassError}
	AhdClassDivisionByZeroError = &AhdClass{Name: "DivisionByZeroError", Parent: AhdClassError}
	AhdClassDomainError         = &AhdClass{Name: "DomainError", Parent: AhdClassError}
	AhdClassIndexError          = &AhdClass{Name: "IndexError", Parent: AhdClassError}
	AhdClassKeyError            = &AhdClass{Name: "KeyError", Parent: AhdClassError}
	AhdClassNullError           = &AhdClass{Name: "NullError", Parent: AhdClassError}
	AhdClassOverflowError       = &AhdClass{Name: "OverflowError", Parent: AhdClassError}
	AhdClassValueError          = &AhdClass{Name: "ValueError", Parent: AhdClassError}
)

// AhdInstance is every AhdCode Class instance. The generated interface of each
// Class embeds it, so identity, freezing, and Class membership work uniformly.
type AhdInstance interface {
	AhdClassOf() *AhdClass
	AhdFreezeGraph(visited map[AhdFreezable]bool)
}

// AhdBase is embedded in the root of every generated Class struct.
type AhdBase struct {
	ahdClass  *AhdClass
	ahdFrozen bool
}

// AhdClassOf returns the exact runtime Class of an instance.
func (base *AhdBase) AhdClassOf() *AhdClass { return base.ahdClass }

// AhdSetClass stamps the exact runtime Class during construction.
func (base *AhdBase) AhdSetClass(class *AhdClass) { base.ahdClass = class }

// AhdFrozen reports whether the instance is part of a deep-frozen graph.
func (base *AhdBase) AhdFrozen() bool { return base.ahdFrozen }

// AhdMarkFrozen freezes the instance itself and reports whether this call was
// the transition, so a graph walk visits each object once.
func (base *AhdBase) AhdMarkFrozen() bool {
	if base.ahdFrozen {
		return false
	}
	base.ahdFrozen = true
	return true
}

// AhdRequireMutable rejects mutation of a deep-frozen object.
func (base *AhdBase) AhdRequireMutable() {
	if base.ahdFrozen {
		AhdRaiseClass(AhdClassConstantError, "cannot mutate a Constant object")
	}
}

// AhdIsClass reports Class membership, including inheritance.
func AhdIsClass(value AhdInstance, target *AhdClass) bool {
	if value == nil || target == nil {
		return false
	}
	for current := value.AhdClassOf(); current != nil; current = current.Parent {
		if current == target {
			return true
		}
	}
	return false
}

// AhdEqInstance is Class reference identity.
func AhdEqInstance(left, right AhdInstance) bool { return left == right }

// AhdSameInstance is strict same: exact runtime Class and the same object.
func AhdSameInstance(left, right AhdInstance) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.AhdClassOf() == right.AhdClassOf() && left == right
}

// AhdStrInstance renders the canonical Class instance text.
func AhdStrInstance(value AhdInstance) string {
	if value == nil {
		return "null"
	}
	class := value.AhdClassOf()
	if class == nil {
		return "<Object>"
	}
	return "<" + class.Name + ">"
}

// AhdRequireInstance reads a nullable Class reference the frontend proved
// non-null.
func AhdRequireInstance[T any](value T, present bool) T {
	if !present {
		AhdRaiseClass(AhdClassNullError, "value is null")
	}
	return value
}

// ---------------------------------------------------------------------------
// Runtime errors
// ---------------------------------------------------------------------------

// AhdSignal is the isolated control value used to raise an AhdCode Error. An
// ordinary Go panic is never an AhdSignal, so a compiler or runtime defect can
// never be caught by an AhdCode except clause.
type AhdSignal struct {
	Instance AhdInstance
	Message  string
}

func (signal *AhdSignal) Error() string {
	return ahdSignalClassName(signal) + ": " + signal.Message
}

func ahdSignalClassName(signal *AhdSignal) string {
	if signal == nil || signal.Instance == nil || signal.Instance.AhdClassOf() == nil {
		return "Error"
	}
	return signal.Instance.AhdClassOf().Name
}

// ahdErrorConstructors lets a runtime check raise a real AhdCode Error
// instance. Generated code installs the built-in Error constructors before the
// program body runs; lookups are by descriptor pointer, never by iteration.
var ahdErrorConstructors = map[*AhdClass]func(string) AhdInstance{}

// AhdRegisterError installs the generated constructor of one built-in Error.
func AhdRegisterError(class *AhdClass, construct func(string) AhdInstance) {
	ahdErrorConstructors[class] = construct
}

// AhdRaiseClass raises a built-in AhdCode runtime Error.
func AhdRaiseClass(class *AhdClass, message string) {
	construct := ahdErrorConstructors[class]
	if construct == nil {
		panic("ahdcode: built-in Error class " + class.Name + " has no registered constructor")
	}
	panic(&AhdSignal{Instance: construct(message), Message: message})
}

// AhdErrorInstance is an AhdCode Error instance, which always carries a
// message attribute inherited from the built-in Error Class.
type AhdErrorInstance interface {
	AhdInstance
	AhdErrorMessage() string
}

// AhdToss raises an AhdCode Error instance written by user code.
func AhdToss(instance AhdInstance) {
	if instance == nil {
		AhdRaiseClass(AhdClassNullError, "cannot toss a null Error")
	}
	message := ""
	if carrier, ok := instance.(AhdErrorInstance); ok {
		message = carrier.AhdErrorMessage()
	}
	panic(&AhdSignal{Instance: instance, Message: message})
}

// Control outcomes of an attempt body. They transfer a pending return, break,
// or continue across the error-handling boundary without skipping ultimately.
const (
	AhdOutcomeNormal   = 0
	AhdOutcomeReturn   = 1
	AhdOutcomeBreak    = 2
	AhdOutcomeContinue = 3
)

// AhdSignalOf converts a recovered value into an AhdCode error signal. An
// ordinary Go panic is re-raised unchanged rather than being handled as an
// AhdCode Error.
func AhdSignalOf(recovered any) *AhdSignal {
	if recovered == nil {
		return nil
	}
	signal, ok := recovered.(*AhdSignal)
	if !ok {
		panic(recovered)
	}
	return signal
}

// AhdMatches reports whether an error signal is handled by one except clause.
func AhdMatches(signal *AhdSignal, class *AhdClass) bool {
	return signal != nil && AhdIsClass(signal.Instance, class)
}

var ahdOut = bufio.NewWriter(os.Stdout)
var ahdIn = bufio.NewReader(os.Stdin)

// AhdFlush writes buffered program output.
func AhdFlush() {
	_ = ahdOut.Flush()
}

// AhdMain runs a generated program body and turns an uncaught AhdCode error
// into a diagnostic exit instead of a Go panic trace.
func AhdMain(install func(), body func()) {
	install()
	failed := false
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			signal, ok := recovered.(*AhdSignal)
			if !ok {
				AhdFlush()
				panic(recovered)
			}
			AhdFlush()
			fmt.Fprintln(os.Stderr, ahdSignalClassName(signal)+": "+signal.Message)
			failed = true
		}()
		body()
	}()
	AhdFlush()
	if failed {
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Constant deep freeze
// ---------------------------------------------------------------------------

// AhdFreezable is every runtime object that participates in the Constant
// deep-freeze graph: List, Pair, and Class instances.
type AhdFreezable interface {
	AhdFreezeGraph(visited map[AhdFreezable]bool)
}

// AhdFreeze deep-freezes the object graph reachable from a Constant binding.
// The traversal is idempotent and terminates on cyclic graphs.
func AhdFreeze[T any](value T) T {
	if target, ok := any(value).(AhdFreezable); ok && target != nil {
		target.AhdFreezeGraph(make(map[AhdFreezable]bool))
	}
	return value
}

// AhdFreezeChild continues a freeze walk into one reachable value. Values with
// no reference identity, such as scalars, are skipped.
func AhdFreezeChild[T any](value T, visited map[AhdFreezable]bool) {
	if child, ok := any(value).(AhdFreezable); ok && child != nil {
		child.AhdFreezeGraph(visited)
	}
}

// AhdEnterFreeze reports whether a graph walk should descend into an object it
// has not visited yet.
func AhdEnterFreeze(target AhdFreezable, visited map[AhdFreezable]bool) bool {
	if target == nil || visited[target] {
		return false
	}
	visited[target] = true
	return true
}

func ahdRejectMutation() {
	AhdRaiseClass(AhdClassConstantError, "cannot mutate a Constant object")
}

// ---------------------------------------------------------------------------
// Null representation
// ---------------------------------------------------------------------------

// AhdBox stores a non-null scalar into a nullable slot.
func AhdBox[T any](value T) *T { return &value }

// AhdNonNull reads a nullable slot that the frontend proved non-null.
func AhdNonNull[T any](value *T) T {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, "value is null")
	}
	return *value
}

// ---------------------------------------------------------------------------
// Terminal I/O
// ---------------------------------------------------------------------------

// AhdWrite prints one canonical line of program output.
func AhdWrite(text string) {
	_, _ = ahdOut.WriteString(text)
	_ = ahdOut.WriteByte('\n')
}

// AhdTake reads one line of terminal input.
func AhdTake() string { return ahdReadLine() }

// AhdTakePrompt writes a prompt, without adding a newline, and then reads one
// line of terminal input. The prompt is never part of the returned String.
func AhdTakePrompt(prompt string) string {
	_, _ = ahdOut.WriteString(prompt)
	return ahdReadLine()
}

// ahdReadLine is the single terminal read behind both take forms. Pending
// output is flushed first so a prompt is visible before the process blocks.
// Only the line terminator is removed, in both the LF and CRLF forms; ordinary
// whitespace in the entered text is preserved. End of input yields an empty
// String.
func ahdReadLine() string {
	AhdFlush()
	line, _ := ahdIn.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	return strings.TrimSuffix(line, "\r")
}

// ---------------------------------------------------------------------------
// Checked Int arithmetic
// ---------------------------------------------------------------------------

func ahdOverflow() {
	AhdRaiseClass(AhdClassOverflowError, "Int arithmetic overflowed signed 64-bit range")
}

// AhdIntAdd adds two Int values without silent wrap-around.
func AhdIntAdd(left, right int64) int64 {
	sum := left + right
	if (sum > left) != (right > 0) {
		ahdOverflow()
	}
	return sum
}

// AhdIntSubtract subtracts two Int values without silent wrap-around.
func AhdIntSubtract(left, right int64) int64 {
	difference := left - right
	if (difference < left) != (right > 0) {
		ahdOverflow()
	}
	return difference
}

// AhdIntMultiply multiplies two Int values without silent wrap-around.
func AhdIntMultiply(left, right int64) int64 {
	if left == 0 || right == 0 {
		return 0
	}
	if (left == math.MinInt64 && right == -1) || (right == math.MinInt64 && left == -1) {
		ahdOverflow()
	}
	product := left * right
	if product/right != left {
		ahdOverflow()
	}
	return product
}

// AhdIntNegate negates an Int value without silent wrap-around.
func AhdIntNegate(value int64) int64 {
	if value == math.MinInt64 {
		ahdOverflow()
	}
	return -value
}

// AhdIntModulo is Int remainder with a zero-divisor error.
func AhdIntModulo(left, right int64) int64 {
	if right == 0 {
		AhdRaiseClass(AhdClassDivisionByZeroError, "Int modulo by zero")
	}
	if right == -1 {
		return 0
	}
	return left % right
}

// AhdIntPower raises an Int to a non-negative Int power without silent wrap.
func AhdIntPower(base, exponent int64) int64 {
	if exponent < 0 {
		AhdRaiseClass(AhdClassDomainError, "Int power requires a non-negative exponent")
	}
	result := int64(1)
	current := base
	remaining := exponent
	for remaining > 0 {
		if remaining&1 == 1 {
			result = AhdIntMultiply(result, current)
		}
		remaining >>= 1
		if remaining > 0 {
			current = AhdIntMultiply(current, current)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Checked Real arithmetic
// ---------------------------------------------------------------------------

func ahdRealCheck(result float64, operation string) float64 {
	if math.IsInf(result, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Real "+operation+" produced a non-finite result")
	}
	if math.IsNaN(result) {
		AhdRaiseClass(AhdClassDomainError, "Real "+operation+" is not defined for these operands")
	}
	return result
}

// AhdRealAdd adds two Real values and rejects non-finite results.
func AhdRealAdd(left, right float64) float64 { return ahdRealCheck(left+right, "addition") }

// AhdRealSubtract subtracts two Real values and rejects non-finite results.
func AhdRealSubtract(left, right float64) float64 {
	return ahdRealCheck(left-right, "subtraction")
}

// AhdRealMultiply multiplies two Real values and rejects non-finite results.
func AhdRealMultiply(left, right float64) float64 {
	return ahdRealCheck(left*right, "multiplication")
}

// AhdRealDivide divides two Real values; a zero divisor is an error.
func AhdRealDivide(left, right float64) float64 {
	if right == 0 {
		AhdRaiseClass(AhdClassDivisionByZeroError, "division by zero")
	}
	return ahdRealCheck(left/right, "division")
}

// AhdRealPower raises a Real to a Real power and rejects non-finite results.
func AhdRealPower(base, exponent float64) float64 {
	return ahdRealCheck(math.Pow(base, exponent), "power")
}

// AhdRealNegate negates a Real value.
func AhdRealNegate(value float64) float64 { return -value }

// AhdIntToReal is the explicit Int -> Real widening conversion.
func AhdIntToReal(value int64) float64 { return float64(value) }

// AhdRealToInt truncates toward zero and preserves the checked Int contract.
func AhdRealToInt(value float64) int64 {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "cannot convert a non-number Real to Int")
	}
	if math.IsInf(value, 0) || value < -9223372036854775808.0 || value >= 9223372036854775808.0 {
		AhdRaiseClass(AhdClassOverflowError, "Real value is outside signed 64-bit Int range")
	}
	return int64(math.Trunc(value))
}

// AhdStringToInt parses the exact v0.1 signed ASCII-decimal Int grammar.
func AhdStringToInt(value string) int64 {
	text := strings.TrimSpace(value)
	if !ahdValidIntText(text) {
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Int text")
	}
	result, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		if number, ok := err.(*strconv.NumError); ok && number.Err == strconv.ErrRange {
			AhdRaiseClass(AhdClassOverflowError, "decimal String is outside signed 64-bit Int range")
		}
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Int text")
	}
	return result
}

// AhdStringToReal parses the exact v0.1 finite ASCII-decimal Real grammar.
func AhdStringToReal(value string) float64 {
	text := strings.TrimSpace(value)
	if !ahdValidRealText(text) {
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Real text")
	}
	result, err := strconv.ParseFloat(text, 64)
	if err != nil {
		if number, ok := err.(*strconv.NumError); ok && number.Err == strconv.ErrRange {
			AhdRaiseClass(AhdClassOverflowError, "decimal String is outside finite Real range")
		}
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Real text")
	}
	if math.IsInf(result, 0) {
		AhdRaiseClass(AhdClassOverflowError, "decimal String is outside finite Real range")
	}
	if math.IsNaN(result) {
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Real text")
	}
	if result == 0 && ahdNonzeroRealSignificand(text) {
		AhdRaiseClass(AhdClassOverflowError, "decimal String is outside finite Real range")
	}
	return result
}

func ahdValidIntText(text string) bool {
	index := ahdAfterSign(text)
	end, digits := ahdDecimalDigits(text, index)
	return digits && end == len(text)
}

func ahdValidRealText(text string) bool {
	index := ahdAfterSign(text)
	var digits bool
	index, digits = ahdDecimalDigits(text, index)
	if !digits {
		return false
	}
	if index < len(text) && text[index] == '.' {
		index++
		var fraction bool
		index, fraction = ahdDecimalDigits(text, index)
		if !fraction {
			return false
		}
	}
	if index < len(text) && (text[index] == 'e' || text[index] == 'E') {
		index++
		if index < len(text) && (text[index] == '+' || text[index] == '-') {
			index++
		}
		var exponent bool
		index, exponent = ahdDecimalDigits(text, index)
		if !exponent {
			return false
		}
	}
	return index == len(text)
}

func ahdAfterSign(text string) int {
	if len(text) > 0 && (text[0] == '+' || text[0] == '-') {
		return 1
	}
	return 0
}

func ahdDecimalDigits(text string, index int) (int, bool) {
	start := index
	for index < len(text) && text[index] >= '0' && text[index] <= '9' {
		index++
	}
	return index, index > start
}

func ahdNonzeroRealSignificand(text string) bool {
	for index := ahdAfterSign(text); index < len(text) && text[index] != 'e' && text[index] != 'E'; index++ {
		if text[index] >= '1' && text[index] <= '9' {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// String operations
// ---------------------------------------------------------------------------

// AhdStringRepeat implements String * Int.
func AhdStringRepeat(text string, count int64) string {
	if count < 0 {
		AhdRaiseClass(AhdClassValueError, "String repeat count must not be negative")
	}
	if count > math.MaxInt32 {
		AhdRaiseClass(AhdClassValueError, "String repeat count is too large")
	}
	return strings.Repeat(text, int(count))
}

// AhdStringChars splits a String into its one-character String elements.
func AhdStringChars(text string) []string {
	runes := []rune(text)
	result := make([]string, len(runes))
	for index, value := range runes {
		result[index] = string(value)
	}
	return result
}

// AhdStringTrim removes leading and trailing Unicode whitespace and preserves
// every interior character.
func AhdStringTrim(text string) string { return strings.TrimSpace(text) }

// AhdStringLower is deterministic, locale-independent Unicode lowercasing.
func AhdStringLower(text string) string { return strings.ToLower(text) }

// AhdStringUpper is deterministic, locale-independent Unicode uppercasing.
func AhdStringUpper(text string) string { return strings.ToUpper(text) }

// AhdStringCapitalize uppercases the first character and leaves the remainder
// exactly as written.
func AhdStringCapitalize(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	return string(unicode.ToUpper(runes[0])) + string(runes[1:])
}

// AhdStringSplit divides a String on every non-overlapping occurrence of a
// non-empty separator, preserving empty fields.
func AhdStringSplit(text, separator string) *AhdList[*string] {
	if separator == "" {
		AhdRaiseClass(AhdClassDomainError, "split requires a non-empty separator")
	}
	parts := strings.Split(text, separator)
	items := make([]*string, len(parts))
	for index := range parts {
		field := parts[index]
		items[index] = &field
	}
	return &AhdList[*string]{items: items}
}

// AhdStringReplace rewrites every non-overlapping occurrence of a non-empty
// search text. The replacement may be empty.
func AhdStringReplace(text, old, replacement string) string {
	if old == "" {
		AhdRaiseClass(AhdClassDomainError, "replace requires a non-empty search text")
	}
	return strings.ReplaceAll(text, old, replacement)
}

// AhdStringStartsWith reports a prefix match.
func AhdStringStartsWith(text, prefix string) bool { return strings.HasPrefix(text, prefix) }

// AhdStringEndsWith reports a suffix match.
func AhdStringEndsWith(text, suffix string) bool { return strings.HasSuffix(text, suffix) }

// AhdStringCount counts non-overlapping occurrences. An empty search text has
// no occurrence count in AhdCode, so it is rejected rather than defined as one
// match per position.
func AhdStringCount(text, needle string) int64 {
	if needle == "" {
		AhdRaiseClass(AhdClassDomainError, "count requires a non-empty search text")
	}
	return int64(strings.Count(text, needle))
}

// AhdStringIndex is the first character index of a search text. The result is
// an AhdCode character index, never a UTF-8 byte offset, and a missing text is
// an error rather than a sentinel index.
func AhdStringIndex(text, needle string) int64 {
	if needle == "" {
		AhdRaiseClass(AhdClassDomainError, "index requires a non-empty search text")
	}
	position := strings.Index(text, needle)
	if position < 0 {
		AhdRaiseClass(AhdClassDomainError, "index did not find the search text")
	}
	return int64(len([]rune(text[:position])))
}

// AhdStringLen counts characters, not bytes.
func AhdStringLen(text string) int64 { return int64(len([]rune(text))) }

// AhdStringAt returns the one-character String at a possibly negative index.
func AhdStringAt(text string, index int64) string {
	runes := []rune(text)
	position := ahdResolveIndex(index, int64(len(runes)))
	return string(runes[position])
}

// AhdStringSlice slices by character position with optional bounds.
func AhdStringSlice(text string, start int64, hasStart bool, end int64, hasEnd bool) string {
	runes := []rune(text)
	low, high := ahdResolveRange(start, hasStart, end, hasEnd, int64(len(runes)))
	return string(runes[low:high])
}

func ahdResolveIndex(index, length int64) int64 {
	position := index
	if position < 0 {
		position += length
	}
	if position < 0 || position >= length {
		AhdRaiseClass(AhdClassIndexError, "index "+strconv.FormatInt(index, 10)+" is out of range for length "+strconv.FormatInt(length, 10))
	}
	return position
}

func ahdResolveRange(start int64, hasStart bool, end int64, hasEnd bool, length int64) (int64, int64) {
	low := int64(0)
	if hasStart {
		low = start
		if low < 0 {
			low += length
		}
	}
	high := length
	if hasEnd {
		high = end
		if high < 0 {
			high += length
		}
	}
	if low < 0 {
		low = 0
	}
	if high > length {
		high = length
	}
	if low > length {
		low = length
	}
	if high < low {
		high = low
	}
	return low, high
}

// ---------------------------------------------------------------------------
// Lazy integer iteration
// ---------------------------------------------------------------------------

// AhdRange is the lazy integer iteration produced by between. Its whole state
// is the current value, the excluded stop, and the step, so iterating any
// range costs O(1) memory no matter how many values it yields. It never
// materializes a List.
type AhdRange struct {
	current  int64
	stop     int64
	step     int64
	finished bool
}

// AhdBetween builds a lazy integer iteration. A zero step cannot make progress
// and is a DomainError rather than a silently substituted step of one.
func AhdBetween(start, stop, step int64) *AhdRange {
	if step == 0 {
		AhdRaiseClass(AhdClassDomainError, "between requires a non-zero step")
	}
	return &AhdRange{current: start, stop: stop, step: step}
}

// Next yields the next Int of the iteration. It computes each value on demand
// and stops before any step that would leave the signed 64-bit range, so a
// range near the Int boundaries terminates instead of wrapping.
func (iteration *AhdRange) Next() (int64, bool) {
	if iteration == nil || iteration.finished {
		return 0, false
	}
	if iteration.step > 0 && iteration.current >= iteration.stop {
		iteration.finished = true
		return 0, false
	}
	if iteration.step < 0 && iteration.current <= iteration.stop {
		iteration.finished = true
		return 0, false
	}
	value := iteration.current
	// A step that would overflow can only move past a stop that already fits in
	// Int, so the iteration is complete rather than in error.
	if iteration.step > 0 && iteration.current > math.MaxInt64-iteration.step {
		iteration.finished = true
		return value, true
	}
	if iteration.step < 0 && iteration.current < math.MinInt64-iteration.step {
		iteration.finished = true
		return value, true
	}
	iteration.current += iteration.step
	return value, true
}

// ---------------------------------------------------------------------------
// List: reference semantics with stable object identity
// ---------------------------------------------------------------------------

// AhdList is the pointer-backed runtime representation of List<T>. Aliases
// share one AhdList value, so in-place mutation is observed by every alias.
type AhdList[T any] struct {
	items  []T
	frozen bool
}

// AhdFreezeGraph deep-freezes this List and every reachable element.
func (list *AhdList[T]) AhdFreezeGraph(visited map[AhdFreezable]bool) {
	if list == nil || !AhdEnterFreeze(list, visited) {
		return
	}
	list.frozen = true
	for _, item := range list.items {
		AhdFreezeChild(item, visited)
	}
}

func (list *AhdList[T]) requireMutable() {
	list.require()
	if list.frozen {
		ahdRejectMutation()
	}
}

// AhdNewList builds a List from its literal elements.
func AhdNewList[T any](items ...T) *AhdList[T] {
	return &AhdList[T]{items: items}
}

func (list *AhdList[T]) require() {
	if list == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
}

// Len reports the element count.
func (list *AhdList[T]) Len() int64 {
	list.require()
	return int64(len(list.items))
}

// At reads a possibly negative index.
func (list *AhdList[T]) At(index int64) T {
	list.require()
	return list.items[ahdResolveIndex(index, int64(len(list.items)))]
}

// Set writes a possibly negative index.
func (list *AhdList[T]) Set(index int64, value T) {
	list.requireMutable()
	list.items[ahdResolveIndex(index, int64(len(list.items)))] = value
}

// Add appends one element, mutating the List in place so every alias observes
// the new element.
func (list *AhdList[T]) Add(value T) {
	list.requireMutable()
	list.items = append(list.items, value)
}

// Eject removes the element at a possibly negative index, mutating the List in
// place. An out-of-range index is an IndexError.
func (list *AhdList[T]) Eject(index int64) {
	list.requireMutable()
	position := ahdResolveIndex(index, int64(len(list.items)))
	list.items = append(list.items[:position], list.items[position+1:]...)
}

// Clear empties the List in place, preserving object identity.
func (list *AhdList[T]) Clear() {
	list.requireMutable()
	list.items = nil
}

// Snapshot returns the shallow iteration snapshot taken at loop entry.
func (list *AhdList[T]) Snapshot() []T {
	list.require()
	result := make([]T, len(list.items))
	copy(result, list.items)
	return result
}

// Slice produces a new List over a character-free index range.
func (list *AhdList[T]) Slice(start int64, hasStart bool, end int64, hasEnd bool) *AhdList[T] {
	list.require()
	low, high := ahdResolveRange(start, hasStart, end, hasEnd, int64(len(list.items)))
	items := make([]T, high-low)
	copy(items, list.items[low:high])
	return &AhdList[T]{items: items}
}

// AhdListConcat implements List<T> + List<T>.
func AhdListConcat[T any](left, right *AhdList[T]) *AhdList[T] {
	left.require()
	right.require()
	items := make([]T, 0, len(left.items)+len(right.items))
	items = append(items, left.items...)
	items = append(items, right.items...)
	return &AhdList[T]{items: items}
}

// AhdListContains implements value membership for List.
func AhdListContains[T any](list *AhdList[T], value T, equal func(T, T) bool) bool {
	list.require()
	for _, item := range list.items {
		if equal(item, value) {
			return true
		}
	}
	return false
}

// AhdListEqual implements deep List value equality.
func AhdListEqual[T any](left, right *AhdList[T], equal func(T, T) bool) bool {
	if left == nil || right == nil {
		return left == right
	}
	if len(left.items) != len(right.items) {
		return false
	}
	for index := range left.items {
		if !equal(left.items[index], right.items[index]) {
			return false
		}
	}
	return true
}

// Reverse reverses the List in place, so every alias observes the new order.
func (list *AhdList[T]) Reverse() {
	list.requireMutable()
	for left, right := 0, len(list.items)-1; left < right; left, right = left+1, right-1 {
		list.items[left], list.items[right] = list.items[right], list.items[left]
	}
}

// Shuffle permutes the List in place with an unbiased descending
// Fisher-Yates pass. AhdMathRandomInt owns all generator advancement, so this
// operation shares the exact Math RNG sequence with random and randomInt.
func (list *AhdList[T]) Shuffle() {
	list.requireMutable()
	for index := len(list.items) - 1; index > 0; index-- {
		selected := int(AhdMathRandomInt(0, int64(index)))
		list.items[index], list.items[selected] = list.items[selected], list.items[index]
	}
}

// AhdListCount counts equal elements with the ordinary AhdCode == semantics.
func AhdListCount[T any](list *AhdList[T], value T, equal func(T, T) bool) int64 {
	list.require()
	total := int64(0)
	for _, item := range list.items {
		if equal(item, value) {
			total++
		}
	}
	return total
}

// AhdListIndex is the first index of an equal element. A value the List does
// not contain is an error rather than a sentinel index.
func AhdListIndex[T any](list *AhdList[T], value T, equal func(T, T) bool) int64 {
	list.require()
	for index, item := range list.items {
		if equal(item, value) {
			return int64(index)
		}
	}
	AhdRaiseClass(AhdClassDomainError, "index did not find the value in the List")
	return 0
}

// AhdListMap builds a new List from a Function applied left to right to a
// shallow snapshot, so a callback that mutates the source cannot change what
// is iterated. The source List is never modified.
func AhdListMap[T any, U any](list *AhdList[T], transform func(T) U) *AhdList[U] {
	items := list.Snapshot()
	result := make([]U, 0, len(items))
	for _, item := range items {
		result = append(result, transform(item))
	}
	return &AhdList[U]{items: result}
}

// AhdListFilter builds a new List of the snapshot elements a predicate keeps.
// The predicate must produce a real Bool: AhdCode has no truthiness, so a null
// result is an error.
func AhdListFilter[T any](list *AhdList[T], keep func(T) *bool) *AhdList[T] {
	items := list.Snapshot()
	result := make([]T, 0, len(items))
	for _, item := range items {
		if ahdPredicate(keep(item)) {
			result = append(result, item)
		}
	}
	return &AhdList[T]{items: result}
}

func ahdPredicate(value *bool) bool {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, "filter predicate returned null")
	}
	return *value
}

func ahdSortKey[K any](value *K) K {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, "sort key Function returned null")
	}
	return *value
}

// ahdOrdered is the set of key types the v0.1 ordering supports.
type ahdOrdered interface{ ~int64 | ~float64 | ~string }

// ahdSortNatural orders a scalar List ascending and stably. Every element is
// read before the receiver is rewritten, so a null element leaves the original
// order untouched.
func ahdSortNatural[T ahdOrdered](list *AhdList[*T]) {
	list.requireMutable()
	items := list.items
	sorted := make([]*T, len(items))
	for index, item := range items {
		sorted[index] = ahdElementPointer(item, "sort")
	}
	sort.SliceStable(sorted, func(left, right int) bool { return *sorted[left] < *sorted[right] })
	list.items = sorted
}

func ahdElementPointer[T any](value *T, name string) *T {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, name+" does not accept a null List element")
	}
	return value
}

// ahdSortByKey orders a List ascending and stably by a key Function. Every key
// is computed, left to right and exactly once per element, before the receiver
// is rewritten, so a key Function that raises leaves the original order
// unchanged.
func ahdSortByKey[T any, K ahdOrdered](list *AhdList[T], key func(T) *K) {
	list.requireMutable()
	items := list.items
	keys := make([]K, len(items))
	for index, item := range items {
		keys[index] = ahdSortKey(key(item))
	}
	order := make([]int, len(items))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(left, right int) bool { return keys[order[left]] < keys[order[right]] })
	sorted := make([]T, len(items))
	for position, index := range order {
		sorted[position] = items[index]
	}
	list.items = sorted
}

// AhdListSortInt orders a List<Int> ascending in place.
func AhdListSortInt(list *AhdList[*int64]) { ahdSortNatural(list) }

// AhdListSortReal orders a List<Real> ascending in place.
func AhdListSortReal(list *AhdList[*float64]) { ahdSortNatural(list) }

// AhdListSortString orders a List<String> ascending in place.
func AhdListSortString(list *AhdList[*string]) { ahdSortNatural(list) }

// AhdListSortKeyInt orders a List by an Int key.
func AhdListSortKeyInt[T any](list *AhdList[T], key func(T) *int64) { ahdSortByKey(list, key) }

// AhdListSortKeyReal orders a List by a Real key.
func AhdListSortKeyReal[T any](list *AhdList[T], key func(T) *float64) { ahdSortByKey(list, key) }

// AhdListSortKeyString orders a List by a String key.
func AhdListSortKeyString[T any](list *AhdList[T], key func(T) *string) { ahdSortByKey(list, key) }

// ---------------------------------------------------------------------------
// Math standard module
// ---------------------------------------------------------------------------

// AhdMathRound rounds to an integral Real. math.Round is explicitly the
// half-away-from-zero operation required by AhdCode, not bankers rounding.
func AhdMathRound(value float64) float64 {
	ahdMathRequireFinite("round", value)
	return math.Round(value)
}

// AhdMathRoundDigits rounds to 0..15 decimal places. When scaling a very large
// finite value would overflow, its float64 spacing is already much wider than
// the requested decimal place, so the value is returned unchanged.
func AhdMathRoundDigits(value float64, digits int64) float64 {
	ahdMathRequireFinite("round", value)
	if digits < 0 || digits > 15 {
		AhdRaiseClass(AhdClassDomainError, "Math.round digits must be in 0..15")
	}
	factor := math.Pow10(int(digits))
	if math.Abs(value) > math.MaxFloat64/factor {
		return value
	}
	return math.Round(value*factor) / factor
}

// AhdMathFloor returns the greatest Int not greater than value.
func AhdMathFloor(value float64) int64 { return ahdMathIntegral("floor", math.Floor(value)) }

// AhdMathCeil returns the least Int not less than value.
func AhdMathCeil(value float64) int64 { return ahdMathIntegral("ceil", math.Ceil(value)) }

func ahdMathIntegral(name string, value float64) int64 {
	ahdMathRequireFinite(name, value)
	limit := math.Ldexp(1, 63)
	if value < -limit || value >= limit {
		AhdRaiseClass(AhdClassOverflowError, "Math."+name+" result does not fit Int")
	}
	return int64(value)
}

// AhdMathSqrt computes the principal square root.
func AhdMathSqrt(value float64) float64 {
	ahdMathRequireFinite("sqrt", value)
	if value < 0 {
		AhdRaiseClass(AhdClassDomainError, "Math.sqrt requires a non-negative value")
	}
	return ahdMathFiniteResult("sqrt", math.Sqrt(value))
}

// AhdMathSin, AhdMathCos, and AhdMathTan use radians.
func AhdMathSin(value float64) float64 {
	ahdMathRequireFinite("sin", value)
	return ahdMathFiniteResult("sin", math.Sin(value))
}

func AhdMathCos(value float64) float64 {
	ahdMathRequireFinite("cos", value)
	return ahdMathFiniteResult("cos", math.Cos(value))
}

func AhdMathTan(value float64) float64 {
	ahdMathRequireFinite("tan", value)
	return ahdMathFiniteResult("tan", math.Tan(value))
}

// AhdMathLog is the natural logarithm; AhdMathLog10 is base ten.
func AhdMathLog(value float64) float64 {
	ahdMathRequireFinite("log", value)
	if value <= 0 {
		AhdRaiseClass(AhdClassDomainError, "Math.log requires a value greater than zero")
	}
	return ahdMathFiniteResult("log", math.Log(value))
}

func AhdMathLog10(value float64) float64 {
	ahdMathRequireFinite("log10", value)
	if value <= 0 {
		AhdRaiseClass(AhdClassDomainError, "Math.log10 requires a value greater than zero")
	}
	return ahdMathFiniteResult("log10", math.Log10(value))
}

// AhdMathExp computes e^value and preserves AhdCode's finite-Real contract.
func AhdMathExp(value float64) float64 {
	ahdMathRequireFinite("exp", value)
	return ahdMathFiniteResult("exp", math.Exp(value))
}

func ahdMathRequireFinite(name string, value float64) {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "Math."+name+" received NaN")
	}
	if math.IsInf(value, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Math."+name+" received a non-finite value")
	}
}

func ahdMathFiniteResult(name string, value float64) float64 {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "Math."+name+" produced an undefined result")
	}
	if math.IsInf(value, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Math."+name+" result exceeds finite Real range")
	}
	return value
}

const (
	ahdSplitMixIncrement uint64 = 0x9e3779b97f4a7c15
	ahdSplitMixFactor1   uint64 = 0xbf58476d1ce4e5b9
	ahdSplitMixFactor2   uint64 = 0x94d049bb133111eb
)

// ahdMathStateFromEntropy reads exactly one 64-bit seed in little-endian byte
// order. Accepting the reader here keeps startup entropy handling isolated and
// deterministic under test without exposing a public seed API.
func ahdMathStateFromEntropy(reader io.Reader) (uint64, error) {
	var seed [8]byte
	if _, err := io.ReadFull(reader, seed[:]); err != nil {
		return 0, fmt.Errorf("read 8 bytes of OS entropy: %w", err)
	}
	return binary.LittleEndian.Uint64(seed[:]), nil
}

func ahdMathStartupState(reader io.Reader, errorOutput io.Writer, exit func(int)) uint64 {
	state, err := ahdMathStateFromEntropy(reader)
	if err != nil {
		fmt.Fprintln(errorOutput, "AhdCode runtime: Math RNG initialization failed:", err)
		exit(1)
		return 0
	}
	return state
}

var ahdMathGenerator = struct {
	sync.Mutex
	state uint64
}{state: ahdMathStartupState(cryptorand.Reader, os.Stderr, os.Exit)}

// AhdMathSeed resets the one process-wide Math sequence. Converting the signed
// seed to uint64 is the specified two's-complement state mapping.
func AhdMathSeed(seed int64) {
	ahdMathGenerator.Lock()
	ahdMathGenerator.state = uint64(seed)
	ahdMathGenerator.Unlock()
}

// ahdSplitMix64 advances and mixes the generator state using the pinned v0.1
// SplitMix64 transition. The caller holds ahdMathGenerator's lock.
func ahdSplitMix64() uint64 {
	ahdMathGenerator.state += ahdSplitMixIncrement
	value := ahdMathGenerator.state
	value = (value ^ (value >> 30)) * ahdSplitMixFactor1
	value = (value ^ (value >> 27)) * ahdSplitMixFactor2
	return value ^ (value >> 31)
}

// AhdMathRandom constructs [0,1) from the high 53 bits of one raw output.
func AhdMathRandom() float64 {
	ahdMathGenerator.Lock()
	raw := ahdSplitMix64()
	ahdMathGenerator.Unlock()
	return float64(raw>>11) * (1.0 / (1 << 53))
}

// AhdMathRandomInt returns an unbiased value from the inclusive interval. A
// singleton interval consumes no generator output. Rejection sampling avoids
// modulo bias, and a zero uint64 span denotes all 2^64 signed Int values.
func AhdMathRandomInt(minimum, maximum int64) int64 {
	if minimum > maximum {
		AhdRaiseClass(AhdClassDomainError, "Math.randomInt minimum must not exceed maximum")
	}
	if minimum == maximum {
		return minimum
	}
	span := uint64(maximum) - uint64(minimum) + 1
	ahdMathGenerator.Lock()
	defer ahdMathGenerator.Unlock()
	if span == 0 {
		return int64(uint64(minimum) + ahdSplitMix64())
	}
	threshold := -span % span
	for {
		raw := ahdSplitMix64()
		if raw >= threshold {
			return int64(uint64(minimum) + raw%span)
		}
	}
}

// ---------------------------------------------------------------------------
// Numeric Fundamentals: abs and the List reductions
// ---------------------------------------------------------------------------

// AhdAbsInt is the Int magnitude. The minimum Int has no Int magnitude, so it
// reports the ordinary checked-arithmetic overflow rather than wrapping.
func AhdAbsInt(value int64) int64 {
	if value < 0 {
		return AhdIntNegate(value)
	}
	return value
}

// AhdAbsReal is the Real magnitude. It preserves the finite-Real contract and
// turns -0.0 into 0.0.
func AhdAbsReal(value float64) float64 { return math.Abs(value) }

// ahdElement reads one element of a numeric List. A null element is an error
// rather than a zero, so a reduction never invents a value.
func ahdElement[T any](value *T, name string) T {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, name+" does not accept a null List element")
	}
	return *value
}

func ahdRequireElements[T any](list *AhdList[*T], name string) {
	list.require()
	if len(list.items) == 0 {
		AhdRaiseClass(AhdClassDomainError, name+" requires a non-empty List")
	}
}

// AhdSumInt adds every element of a List<Int> with checked Int arithmetic. An
// empty List sums to the additive identity 0, and the List is not modified.
func AhdSumInt(list *AhdList[*int64]) int64 {
	list.require()
	total := int64(0)
	for _, item := range list.items {
		total = AhdIntAdd(total, ahdElement(item, "sum"))
	}
	return total
}

// AhdSumReal adds every element of a List<Real> with checked Real arithmetic,
// so a non-finite total is an error rather than Inf or NaN. An empty List sums
// to 0.0, and the List is not modified.
func AhdSumReal(list *AhdList[*float64]) float64 {
	list.require()
	total := 0.0
	for _, item := range list.items {
		total = AhdRealAdd(total, ahdElement(item, "sum"))
	}
	return total
}

// AhdMinInt is the least element of a non-empty List<Int>.
func AhdMinInt(list *AhdList[*int64]) int64 {
	ahdRequireElements(list, "min")
	result := ahdElement(list.items[0], "min")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "min"); value < result {
			result = value
		}
	}
	return result
}

// AhdMaxInt is the greatest element of a non-empty List<Int>.
func AhdMaxInt(list *AhdList[*int64]) int64 {
	ahdRequireElements(list, "max")
	result := ahdElement(list.items[0], "max")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "max"); value > result {
			result = value
		}
	}
	return result
}

// AhdMinReal is the least element of a non-empty List<Real>.
func AhdMinReal(list *AhdList[*float64]) float64 {
	ahdRequireElements(list, "min")
	result := ahdElement(list.items[0], "min")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "min"); value < result {
			result = value
		}
	}
	return result
}

// AhdMaxReal is the greatest element of a non-empty List<Real>.
func AhdMaxReal(list *AhdList[*float64]) float64 {
	ahdRequireElements(list, "max")
	result := ahdElement(list.items[0], "max")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "max"); value > result {
			result = value
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Pair: reference semantics with insertion order
// ---------------------------------------------------------------------------

// AhdPair is the pointer-backed runtime representation of Pair<K, V>. Key
// order is insertion order; updating an existing key keeps its position.
type AhdPair[K comparable, V any] struct {
	keys   []K
	values map[K]V
	frozen bool
}

// AhdFreezeGraph deep-freezes this Pair and every reachable key and value.
func (pair *AhdPair[K, V]) AhdFreezeGraph(visited map[AhdFreezable]bool) {
	if pair == nil || !AhdEnterFreeze(pair, visited) {
		return
	}
	pair.require()
	pair.frozen = true
	for _, key := range pair.keys {
		AhdFreezeChild(key, visited)
		AhdFreezeChild(pair.values[key], visited)
	}
}

func (pair *AhdPair[K, V]) requireMutable() {
	pair.require()
	if pair.frozen {
		ahdRejectMutation()
	}
}

// AhdNewPair builds an empty Pair.
func AhdNewPair[K comparable, V any]() *AhdPair[K, V] {
	return &AhdPair[K, V]{values: make(map[K]V)}
}

func (pair *AhdPair[K, V]) require() {
	if pair == nil {
		AhdRaiseClass(AhdClassNullError, "Pair value is null")
	}
	if pair.values == nil {
		pair.values = make(map[K]V)
	}
}

// Len reports the entry count.
func (pair *AhdPair[K, V]) Len() int64 {
	pair.require()
	return int64(len(pair.keys))
}

// Set inserts or updates a key without moving an existing key.
func (pair *AhdPair[K, V]) Set(key K, value V) {
	pair.requireMutable()
	if _, exists := pair.values[key]; !exists {
		pair.keys = append(pair.keys, key)
	}
	pair.values[key] = value
}

// Get reads a key; a missing key is a KeyError.
func (pair *AhdPair[K, V]) Get(key K) V {
	pair.require()
	value, exists := pair.values[key]
	if !exists {
		AhdRaiseClass(AhdClassKeyError, "Pair has no key "+ahdKeyText(key))
	}
	return value
}

// Has reports key membership.
func (pair *AhdPair[K, V]) Has(key K) bool {
	pair.require()
	_, exists := pair.values[key]
	return exists
}

// Eject removes one key and its value, mutating the Pair in place. A missing
// key is a KeyError. Re-adding an ejected key appends it as a new final entry.
func (pair *AhdPair[K, V]) Eject(key K) {
	pair.requireMutable()
	if _, exists := pair.values[key]; !exists {
		AhdRaiseClass(AhdClassKeyError, "Pair has no key "+ahdKeyText(key))
	}
	pair.Remove(key)
}

// Remove deletes a key, keeping the order of the remaining keys.
func (pair *AhdPair[K, V]) Remove(key K) bool {
	pair.requireMutable()
	if _, exists := pair.values[key]; !exists {
		return false
	}
	delete(pair.values, key)
	for index, existing := range pair.keys {
		if existing == key {
			pair.keys = append(pair.keys[:index], pair.keys[index+1:]...)
			break
		}
	}
	return true
}

// Clear empties the Pair in place, preserving object identity.
func (pair *AhdPair[K, V]) Clear() {
	pair.requireMutable()
	pair.keys = nil
	pair.values = make(map[K]V)
}

// Keys returns the insertion-order iteration snapshot taken at loop entry.
func (pair *AhdPair[K, V]) Keys() []K {
	pair.require()
	result := make([]K, len(pair.keys))
	copy(result, pair.keys)
	return result
}

// AhdPairEqual implements deep Pair value equality; order is not significant.
func AhdPairEqual[K comparable, V any](left, right *AhdPair[K, V], equal func(V, V) bool) bool {
	if left == nil || right == nil {
		return left == right
	}
	left.require()
	right.require()
	if len(left.keys) != len(right.keys) {
		return false
	}
	for key, value := range left.values {
		other, exists := right.values[key]
		if !exists || !equal(value, other) {
			return false
		}
	}
	return true
}

func ahdKeyText(key any) string {
	switch typed := key.(type) {
	case string:
		return strconv.Quote(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(key)
	}
}

// ---------------------------------------------------------------------------
// Canonical str rendering
// ---------------------------------------------------------------------------

// AhdStrInt renders an Int in base-10 decimal text.
func AhdStrInt(value int64) string { return strconv.FormatInt(value, 10) }

// AhdStrBool renders a Bool.
func AhdStrBool(value bool) string { return strconv.FormatBool(value) }

// AhdStrString renders a String as itself.
func AhdStrString(value string) string { return value }

// AhdStrQuoted renders a String nested inside a collection.
func AhdStrQuoted(value string) string { return ahdQuote(value) }

// AhdStrReal renders a Real with locale-independent, shortest round-trip text.
// An integral Real keeps a trailing .0 and negative zero is preserved.
func AhdStrReal(value float64) string {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "Real value is not a number")
	}
	if math.IsInf(value, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Real value is not finite")
	}
	return ahdFormatReal(value)
}

// ahdFormatReal renders the shortest round-trip decimal text for a finite
// Real. Fixed notation is used for ordinary magnitudes; scientific notation
// with a lowercase e is used only when a fixed rendering would be unwieldy.
func ahdFormatReal(value float64) string {
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	exponent := 0
	if marker := strings.IndexByte(scientific, 'e'); marker >= 0 {
		parsed, err := strconv.Atoi(scientific[marker+1:])
		if err == nil {
			exponent = parsed
		}
	}
	if exponent < -4 || exponent >= 21 {
		return scientific
	}
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text
}

// AhdStrNull lifts a renderer over a nullable slot.
func AhdStrNull[T any](render func(T) string) func(*T) string {
	return func(value *T) string {
		if value == nil {
			return "null"
		}
		return render(*value)
	}
}

// AhdStrRefInstance renders a nullable Class reference in canonical text.
func AhdStrRefInstance[T AhdInstance](value T) string {
	if any(value) == nil {
		return "null"
	}
	return AhdStrInstance(value)
}

// AhdStrFunction renders a named Function value.
func AhdStrFunction(name string) string { return "<Function " + name + ">" }

// AhdStrList renders a List with a canonical literal-like representation.
func AhdStrList[T any](render func(T) string) func(*AhdList[T]) string {
	return func(list *AhdList[T]) string {
		if list == nil {
			return "null"
		}
		parts := make([]string, len(list.items))
		for index, item := range list.items {
			parts[index] = render(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
}

// AhdStrPair renders a Pair in insertion order.
func AhdStrPair[K comparable, V any](renderKey func(K) string, renderValue func(V) string) func(*AhdPair[K, V]) string {
	return func(pair *AhdPair[K, V]) string {
		if pair == nil {
			return "null"
		}
		pair.require()
		parts := make([]string, 0, len(pair.keys))
		for _, key := range pair.keys {
			parts = append(parts, renderKey(key)+": "+renderValue(pair.values[key]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
}

func ahdQuote(value string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, item := range value {
		switch item {
		case '"':
			out.WriteString("\\\"")
		case '\\':
			out.WriteString("\\\\")
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		case '\t':
			out.WriteString("\\t")
		default:
			out.WriteRune(item)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// ---------------------------------------------------------------------------
// Equality helpers
// ---------------------------------------------------------------------------

// AhdEqInt compares two Int values.
func AhdEqInt(left, right int64) bool { return left == right }

// AhdEqReal compares two Real values.
func AhdEqReal(left, right float64) bool { return left == right }

// AhdEqString compares two String values.
func AhdEqString(left, right string) bool { return left == right }

// AhdEqBool compares two Bool values.
func AhdEqBool(left, right bool) bool { return left == right }

// AhdEqNull lifts an equality over a nullable slot.
func AhdEqNull[T any](equal func(T, T) bool) func(*T, *T) bool {
	return func(left, right *T) bool {
		if left == nil || right == nil {
			return left == nil && right == nil
		}
		return equal(*left, *right)
	}
}

// AhdEqRef compares Class reference identity.
func AhdEqRef[T comparable]() func(T, T) bool {
	return func(left, right T) bool { return left == right }
}

// AhdEqList lifts an element equality to deep List equality.
func AhdEqList[T any](equal func(T, T) bool) func(*AhdList[T], *AhdList[T]) bool {
	return func(left, right *AhdList[T]) bool { return AhdListEqual(left, right, equal) }
}

// AhdEqPair lifts a value equality to deep Pair equality.
func AhdEqPair[K comparable, V any](equal func(V, V) bool) func(*AhdPair[K, V], *AhdPair[K, V]) bool {
	return func(left, right *AhdPair[K, V]) bool { return AhdPairEqual(left, right, equal) }
}

// AhdSameDifferent evaluates both operands of a statically type-distinct same.
func AhdSameDifferent[A any, B any](left A, right B) bool { return false }

// AhdConstBool evaluates a value for its effects and yields a statically
// resolved Bool result, such as Class member existence.
func AhdConstBool[T any](value T, result bool) bool { return result }

// AhdStringContains implements String substring membership.
func AhdStringContains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

// AhdBuildPair builds a Pair literal in written key order.
func AhdBuildPair[K comparable, V any](keys []K, values []V) *AhdPair[K, V] {
	result := AhdNewPair[K, V]()
	for index, key := range keys {
		result.Set(key, values[index])
	}
	return result
}

// AhdUnreachable reports a Function that ended without returning a value. The
// frontend proves definite return, so reaching it indicates a compiler defect
// rather than a user program error.
func AhdUnreachable[T any]() T {
	panic("ahdcode: Function ended without returning a value")
}
