// Package types defines AhdCode semantic types independently from syntax and
// code generation.
package types

import (
	"fmt"
	"strings"
)

// Kind identifies the semantic shape of a Type.
type Kind uint8

const (
	InvalidKind Kind = iota
	IntKind
	RealKind
	StringKind
	BoolKind
	NothingKind
	ListKind
	PairKind
	FunctionKind
	ClassKind
	// RangeKind is the compiler-internal type of a lazy integer iteration
	// produced by between. It has no AhdCode syntax in v0.1.
	RangeKind
)

func (kind Kind) String() string {
	switch kind {
	case IntKind:
		return "Int"
	case RealKind:
		return "Real"
	case StringKind:
		return "String"
	case BoolKind:
		return "Bool"
	case NothingKind:
		return "Nothing"
	case ListKind:
		return "List"
	case PairKind:
		return "Pair"
	case FunctionKind:
		return "Function"
	case ClassKind:
		return "Class"
	case RangeKind:
		return "Int range"
	default:
		return "<invalid>"
	}
}

// Type is implemented by every semantic type representation.
type Type interface {
	Kind() Kind
	String() string
}

type Basic struct{ TypeKind Kind }

func (basic Basic) Kind() Kind     { return basic.TypeKind }
func (basic Basic) String() string { return basic.TypeKind.String() }

var (
	Invalid Type = Basic{InvalidKind}
	Int     Type = Basic{IntKind}
	Real    Type = Basic{RealKind}
	String  Type = Basic{StringKind}
	Bool    Type = Basic{BoolKind}
	Nothing Type = Basic{NothingKind}
)

// Range is the lazy integer iteration produced by between. It is a
// compiler-internal type: no AhdCode type syntax denotes it, and its only
// public contract is that iterating it yields Int.
type Range struct{}

func (Range) Kind() Kind     { return RangeKind }
func (Range) String() string { return "Int range" }

// IntRange is the single Range value; the type carries no parameters.
var IntRange Type = Range{}

// List's element nullability is a structural part of the type itself (List<Int>
// and List<Int?> are distinct, non-interchangeable types), unlike a binding's
// own top-level nullability, which stays a separate flow-tracked dimension.
type List struct {
	Element         Type
	ElementNullable bool
}

func (List) Kind() Kind { return ListKind }
func (list List) String() string {
	return fmt.Sprintf("List<%s>", displayNullable(list.Element, list.ElementNullable))
}

// Pair's value nullability is likewise structural. Keys may never be
// nullable (see IsPairKey), so Pair carries no KeyNullable.
type Pair struct {
	Key           Type
	Value         Type
	ValueNullable bool
}

func (Pair) Kind() Kind { return PairKind }
func (pair Pair) String() string {
	return fmt.Sprintf("Pair<%s, %s>", Display(pair.Key), displayNullable(pair.Value, pair.ValueNullable))
}

func displayNullable(value Type, nullable bool) string {
	if nullable {
		return Display(value) + "?"
	}
	return Display(value)
}

// Parameter is one concrete callable signature parameter.
type Parameter struct {
	Name       string
	Type       Type
	HasDefault bool
}

// Signature is compiler-internal. AhdCode source still spells its public type
// only as Function.
type Signature struct {
	Parameters []Parameter
	Return     Type
}

type Function struct{ Signature *Signature }

func (Function) Kind() Kind { return FunctionKind }
func (function Function) String() string {
	if function.Signature == nil {
		return "Function"
	}
	parts := make([]string, len(function.Signature.Parameters))
	for index, parameter := range function.Signature.Parameters {
		parts[index] = Display(parameter.Type)
	}
	return "Function(" + strings.Join(parts, ", ") + ") -> " + Display(function.Signature.Return)
}

// ClassSymbol is the identity carried by class instance/reference types. It is
// deliberately free of semantic-package symbols so module metadata can reuse
// it later without a package cycle.
type ClassSymbol struct {
	ModuleID string
	Name     string
	Parent   *ClassSymbol
}

type Class struct {
	Symbol    *ClassSymbol
	Reference bool
}

func (Class) Kind() Kind { return ClassKind }
func (class Class) String() string {
	if class.Symbol == nil {
		return "Class<?>"
	}
	if class.Reference {
		return fmt.Sprintf("Class<%s> reference", class.Symbol.Name)
	}
	return class.Symbol.Name
}

func Display(value Type) string {
	if value == nil {
		return "<invalid>"
	}
	return value.String()
}

func IsInvalid(value Type) bool { return value == nil || value.Kind() == InvalidKind }

// IterationYield reports the type an expression yields when iterated with
// for, whether that yielded value may be null, and whether the type is
// iterable at all. It is the one place that answers those questions for
// every iterable kind. Pair iteration yields keys, which are never nullable.
func IterationYield(value Type) (element Type, elementNullable bool, ok bool) {
	if value == nil {
		return Invalid, false, false
	}
	switch typed := value.(type) {
	case List:
		return typed.Element, typed.ElementNullable, true
	case Pair:
		return typed.Key, false, true
	case Range:
		return Int, false, true
	default:
		if value.Kind() == StringKind {
			return String, false, true
		}
		return Invalid, false, false
	}
}

// IsPairKey reports whether a type may be used as a v0.1 Pair key. Keys are
// limited to the stable simple scalar types; Real, Class instances, Function
// values, collections, and null are not Pair keys.
func IsPairKey(value Type) bool {
	if value == nil {
		return false
	}
	switch value.Kind() {
	case StringKind, IntKind, BoolKind:
		return true
	default:
		return false
	}
}
func IsNumeric(value Type) bool {
	return value != nil && (value.Kind() == IntKind || value.Kind() == RealKind)
}

// Equal compares semantic type identity, including invariant generic
// arguments and concrete callable signatures.
func Equal(left, right Type) bool {
	if left == nil || right == nil || left.Kind() != right.Kind() {
		return false
	}
	switch leftValue := left.(type) {
	case List:
		rightValue, ok := right.(List)
		return ok && Equal(leftValue.Element, rightValue.Element) && leftValue.ElementNullable == rightValue.ElementNullable
	case Pair:
		rightValue, ok := right.(Pair)
		return ok && Equal(leftValue.Key, rightValue.Key) && Equal(leftValue.Value, rightValue.Value) && leftValue.ValueNullable == rightValue.ValueNullable
	case Function:
		rightValue, ok := right.(Function)
		if !ok {
			return false
		}
		return equalSignature(leftValue.Signature, rightValue.Signature)
	case Class:
		rightValue, ok := right.(Class)
		return ok && sameClassIdentity(leftValue.Symbol, rightValue.Symbol) && leftValue.Reference == rightValue.Reference
	default:
		return true
	}
}

func equalSignature(left, right *Signature) bool {
	if left == nil || right == nil {
		return left == right
	}
	if len(left.Parameters) != len(right.Parameters) || !Equal(left.Return, right.Return) {
		return false
	}
	for index := range left.Parameters {
		// Parameter names belong to the callable declaration and named-call
		// surface, not to Function type identity. Keep them on the signature for
		// diagnostics and argument binding, but compare only semantic type
		// properties here.
		if left.Parameters[index].HasDefault != right.Parameters[index].HasDefault ||
			!Equal(left.Parameters[index].Type, right.Parameters[index].Type) {
			return false
		}
	}
	return true
}

// Assignable reports type-only assignment compatibility. Nullability is a
// separate semantic dimension and is intentionally absent here.
func Assignable(target, value Type) bool {
	if IsInvalid(target) || IsInvalid(value) {
		return true // avoid cascading diagnostics after an earlier error
	}
	if Equal(target, value) {
		return true
	}
	if target.Kind() == RealKind && value.Kind() == IntKind {
		return true
	}
	targetFunction, targetFunctionOK := target.(Function)
	valueFunction, valueFunctionOK := value.(Function)
	if targetFunctionOK && valueFunctionOK && targetFunction.Signature == nil && valueFunction.Signature != nil {
		return true
	}
	targetClass, targetOK := target.(Class)
	valueClass, valueOK := value.(Class)
	if targetOK && valueOK && !targetClass.Reference && !valueClass.Reference {
		visited := make(map[*ClassSymbol]bool)
		for current := valueClass.Symbol; current != nil && !visited[current]; current = current.Parent {
			visited[current] = true
			if sameClassIdentity(current, targetClass.Symbol) {
				return true
			}
		}
	}
	return false
}

// SameClassIdentity compares canonical cross-module Class identity. Legacy
// single-file symbols without a ModuleID retain pointer identity.
func SameClassIdentity(left, right *ClassSymbol) bool { return sameClassIdentity(left, right) }

func sameClassIdentity(left, right *ClassSymbol) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.ModuleID == "" || right.ModuleID == "" {
		return left == right
	}
	return left.ModuleID == right.ModuleID && left.Name == right.Name
}
