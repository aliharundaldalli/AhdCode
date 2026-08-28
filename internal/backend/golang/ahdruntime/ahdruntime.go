// Package ahdruntime is the AhdCode v0.1 Go backend runtime.
//
// This file is compiled twice: once as part of the compiler (so ordinary Go
// tooling checks it) and once as generated program source, where the package
// clause is rewritten to main. It must therefore depend only on the Go
// standard library and must not reference any other AhdCode package.
package ahdruntime

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Runtime errors
// ---------------------------------------------------------------------------

// AhdError is the value raised by every AhdCode runtime failure.
type AhdError struct {
	Class   string
	Message string
}

func (failure *AhdError) Error() string { return failure.Class + ": " + failure.Message }

// AhdRaise raises an AhdCode runtime error.
func AhdRaise(class, message string) {
	panic(&AhdError{Class: class, Message: message})
}

var ahdOut = bufio.NewWriter(os.Stdout)
var ahdIn = bufio.NewReader(os.Stdin)

// AhdFlush writes buffered program output.
func AhdFlush() {
	_ = ahdOut.Flush()
}

// AhdMain runs a generated program body and turns an uncaught AhdCode runtime
// error into a diagnostic exit instead of a Go panic trace.
func AhdMain(body func()) {
	failed := false
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			failure, ok := recovered.(*AhdError)
			if !ok {
				AhdFlush()
				panic(recovered)
			}
			AhdFlush()
			fmt.Fprintln(os.Stderr, "AhdCode runtime error: "+failure.Class+": "+failure.Message)
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
// Null representation
// ---------------------------------------------------------------------------

// AhdBox stores a non-null scalar into a nullable slot.
func AhdBox[T any](value T) *T { return &value }

// AhdNonNull reads a nullable slot that the frontend proved non-null.
func AhdNonNull[T any](value *T) T {
	if value == nil {
		AhdRaise("NullError", "value is null")
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

// AhdTake writes an optional prompt and reads one line of terminal input. The
// line terminator is removed. End of input yields an empty String.
func AhdTake(prompt string, hasPrompt bool) string {
	if hasPrompt {
		_, _ = ahdOut.WriteString(prompt)
	}
	AhdFlush()
	line, err := ahdIn.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if err != nil && line == "" {
		return ""
	}
	return line
}

// ---------------------------------------------------------------------------
// Checked Int arithmetic
// ---------------------------------------------------------------------------

func ahdOverflow() {
	AhdRaise("OverflowError", "Int arithmetic overflowed signed 64-bit range")
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
		AhdRaise("DivisionByZeroError", "Int modulo by zero")
	}
	if right == -1 {
		return 0
	}
	return left % right
}

// AhdIntPower raises an Int to a non-negative Int power without silent wrap.
func AhdIntPower(base, exponent int64) int64 {
	if exponent < 0 {
		AhdRaise("DomainError", "Int power requires a non-negative exponent")
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
		AhdRaise("OverflowError", "Real "+operation+" produced a non-finite result")
	}
	if math.IsNaN(result) {
		AhdRaise("DomainError", "Real "+operation+" is not defined for these operands")
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
		AhdRaise("DivisionByZeroError", "division by zero")
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

// ---------------------------------------------------------------------------
// String operations
// ---------------------------------------------------------------------------

// AhdStringRepeat implements String * Int.
func AhdStringRepeat(text string, count int64) string {
	if count < 0 {
		AhdRaise("ValueError", "String repeat count must not be negative")
	}
	if count > math.MaxInt32 {
		AhdRaise("ValueError", "String repeat count is too large")
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
		AhdRaise("IndexError", "index "+strconv.FormatInt(index, 10)+" is out of range for length "+strconv.FormatInt(length, 10))
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
// List: reference semantics with stable object identity
// ---------------------------------------------------------------------------

// AhdList is the pointer-backed runtime representation of List<T>. Aliases
// share one AhdList value, so in-place mutation is observed by every alias.
type AhdList[T any] struct {
	items []T
}

// AhdNewList builds a List from its literal elements.
func AhdNewList[T any](items ...T) *AhdList[T] {
	return &AhdList[T]{items: items}
}

func (list *AhdList[T]) require() {
	if list == nil {
		AhdRaise("NullError", "List value is null")
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
	list.require()
	list.items[ahdResolveIndex(index, int64(len(list.items)))] = value
}

// Clear empties the List in place, preserving object identity.
func (list *AhdList[T]) Clear() {
	list.require()
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

// ---------------------------------------------------------------------------
// Pair: reference semantics with insertion order
// ---------------------------------------------------------------------------

// AhdPair is the pointer-backed runtime representation of Pair<K, V>. Key
// order is insertion order; updating an existing key keeps its position.
type AhdPair[K comparable, V any] struct {
	keys   []K
	values map[K]V
}

// AhdNewPair builds an empty Pair.
func AhdNewPair[K comparable, V any]() *AhdPair[K, V] {
	return &AhdPair[K, V]{values: make(map[K]V)}
}

func (pair *AhdPair[K, V]) require() {
	if pair == nil {
		AhdRaise("NullError", "Pair value is null")
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
	pair.require()
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
		AhdRaise("KeyError", "Pair has no key "+ahdKeyText(key))
	}
	return value
}

// Has reports key membership.
func (pair *AhdPair[K, V]) Has(key K) bool {
	pair.require()
	_, exists := pair.values[key]
	return exists
}

// Remove deletes a key, keeping the order of the remaining keys.
func (pair *AhdPair[K, V]) Remove(key K) bool {
	pair.require()
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
	pair.require()
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
		AhdRaise("DomainError", "Real value is not a number")
	}
	if math.IsInf(value, 0) {
		AhdRaise("OverflowError", "Real value is not finite")
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

// AhdStrRef renders a reference-typed value as its canonical Class text.
func AhdStrRef[T any](name string) func(*T) string {
	return func(value *T) string {
		if value == nil {
			return "null"
		}
		return "<" + name + ">"
	}
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

// AhdEqRef compares reference identity.
func AhdEqRef[T any]() func(*T, *T) bool {
	return func(left, right *T) bool { return left == right }
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

// AhdIsType reports Class membership for a possibly null instance.
func AhdIsType[T any](value *T, matches bool) bool { return value != nil && matches }

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

// AhdUnreachable reports a Function that ended without returning a value.
func AhdUnreachable[T any]() T {
	AhdRaise("ReturnError", "Function ended without returning a value")
	var zero T
	return zero
}
