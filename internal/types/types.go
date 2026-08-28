// Package types defines AhdCode semantic types independently from syntax and
// code generation.
package types

import "fmt"

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

type List struct{ Element Type }

func (List) Kind() Kind { return ListKind }
func (list List) String() string {
	return fmt.Sprintf("List<%s>", Display(list.Element))
}

type Pair struct {
	Key   Type
	Value Type
}

func (Pair) Kind() Kind { return PairKind }
func (pair Pair) String() string {
	return fmt.Sprintf("Pair<%s, %s>", Display(pair.Key), Display(pair.Value))
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
	return "Function<resolved>"
}

// ClassSymbol is the identity carried by class instance/reference types. It is
// deliberately free of semantic-package symbols so module metadata can reuse
// it later without a package cycle.
type ClassSymbol struct {
	Name   string
	Parent *ClassSymbol
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
		return ok && Equal(leftValue.Element, rightValue.Element)
	case Pair:
		rightValue, ok := right.(Pair)
		return ok && Equal(leftValue.Key, rightValue.Key) && Equal(leftValue.Value, rightValue.Value)
	case Function:
		rightValue, ok := right.(Function)
		if !ok {
			return false
		}
		return equalSignature(leftValue.Signature, rightValue.Signature)
	case Class:
		rightValue, ok := right.(Class)
		return ok && leftValue.Symbol == rightValue.Symbol && leftValue.Reference == rightValue.Reference
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
		if left.Parameters[index].Name != right.Parameters[index].Name ||
			left.Parameters[index].HasDefault != right.Parameters[index].HasDefault ||
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
			if current == targetClass.Symbol {
				return true
			}
		}
	}
	return false
}
