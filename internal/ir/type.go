package ir

import (
	"fmt"
	"strings"
)

type ModuleID string
type SymbolID string
type CallableID string
type ClassID string
type FieldID string
type TempID string

type TypeKind string

const (
	InvalidType  TypeKind = "Invalid"
	IntType      TypeKind = "Int"
	RealType     TypeKind = "Real"
	StringType   TypeKind = "String"
	BoolType     TypeKind = "Bool"
	NothingType  TypeKind = "Nothing"
	ListType     TypeKind = "List"
	PairType     TypeKind = "Pair"
	FunctionType TypeKind = "Function"
	ClassType    TypeKind = "Class"
	// RangeType is the lazy integer iteration produced by between. It has no
	// AhdCode type syntax and exists only as iteration state.
	RangeType TypeKind = "Range"
)

// Type is a pointer-address-independent semantic type representation.
// ElementNullable/ValueNullable are structural: List<Int> and List<Int?> are
// distinct, non-interchangeable types, unlike a binding's own top-level
// nullability, which the backend tracks separately via NullState.
type Type struct {
	Kind            TypeKind
	Element         *Type
	ElementNullable bool
	Key             *Type
	Value           *Type
	ValueNullable   bool
	Signature       *Signature
	Class           ClassID
	Reference       bool
}

type ParameterType struct {
	Name       string
	Type       Type
	HasDefault bool
}

type Signature struct {
	Parameters []ParameterType
	Return     Type
}

func (value Type) String() string {
	switch value.Kind {
	case ListType:
		return fmt.Sprintf("List<%s%s>", typePointerString(value.Element), nullableSuffix(value.ElementNullable))
	case PairType:
		return fmt.Sprintf("Pair<%s, %s%s>", typePointerString(value.Key), typePointerString(value.Value), nullableSuffix(value.ValueNullable))
	case FunctionType:
		if value.Signature == nil {
			return "Function<?>"
		}
		parts := make([]string, len(value.Signature.Parameters))
		for index, parameter := range value.Signature.Parameters {
			parts[index] = parameter.Name + ":" + parameter.Type.String()
		}
		return "Function(" + strings.Join(parts, ",") + ")->" + value.Signature.Return.String()
	case ClassType:
		if value.Reference {
			return "ClassRef<" + string(value.Class) + ">"
		}
		return string(value.Class)
	default:
		return string(value.Kind)
	}
}

func typePointerString(value *Type) string {
	if value == nil {
		return "Invalid"
	}
	return value.String()
}

func nullableSuffix(nullable bool) string {
	if nullable {
		return "?"
	}
	return ""
}

func EqualType(left, right Type) bool {
	if left.Kind != right.Kind || left.Class != right.Class || left.Reference != right.Reference {
		return false
	}
	switch left.Kind {
	case ListType:
		return left.ElementNullable == right.ElementNullable && equalTypePointer(left.Element, right.Element)
	case PairType:
		return left.ValueNullable == right.ValueNullable && equalTypePointer(left.Key, right.Key) && equalTypePointer(left.Value, right.Value)
	case FunctionType:
		return EqualSignature(left.Signature, right.Signature)
	default:
		return true
	}
}

func EqualSignature(left, right *Signature) bool {
	if left == nil || right == nil {
		return left == right
	}
	if len(left.Parameters) != len(right.Parameters) || !EqualType(left.Return, right.Return) {
		return false
	}
	for index := range left.Parameters {
		// Names are retained for named-argument lowering and diagnostics, but do
		// not distinguish callable types.
		if left.Parameters[index].HasDefault != right.Parameters[index].HasDefault || !EqualType(left.Parameters[index].Type, right.Parameters[index].Type) {
			return false
		}
	}
	return true
}

func equalTypePointer(left, right *Type) bool {
	if left == nil || right == nil {
		return left == right
	}
	return EqualType(*left, *right)
}

func IsValidType(value Type) bool {
	if value.Kind == "" || value.Kind == InvalidType {
		return false
	}
	switch value.Kind {
	case ListType:
		return value.Element != nil && IsValidType(*value.Element)
	case PairType:
		return value.Key != nil && value.Value != nil && IsValidType(*value.Key) && IsValidType(*value.Value)
	case FunctionType:
		return value.Signature != nil
	case ClassType:
		return value.Class != ""
	default:
		return true
	}
}

// IsPairKeyType reports whether a lowered type is a valid v0.1 Pair key. It
// mirrors the frontend rule so a backend never has to trust an unchecked IR.
func IsPairKeyType(value Type) bool {
	switch value.Kind {
	case StringType, IntType, BoolType:
		return true
	default:
		return false
	}
}

type NullState string

const (
	MaybeNull NullState = "MaybeNull"
	Null      NullState = "Null"
	NonNull   NullState = "NonNull"
)
