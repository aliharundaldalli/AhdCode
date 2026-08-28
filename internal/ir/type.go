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
)

// Type is a pointer-address-independent semantic type representation.
type Type struct {
	Kind      TypeKind
	Element   *Type
	Key       *Type
	Value     *Type
	Signature *Signature
	Class     ClassID
	Reference bool
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
		return fmt.Sprintf("List<%s>", typePointerString(value.Element))
	case PairType:
		return fmt.Sprintf("Pair<%s, %s>", typePointerString(value.Key), typePointerString(value.Value))
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

func EqualType(left, right Type) bool {
	if left.Kind != right.Kind || left.Class != right.Class || left.Reference != right.Reference {
		return false
	}
	switch left.Kind {
	case ListType:
		return equalTypePointer(left.Element, right.Element)
	case PairType:
		return equalTypePointer(left.Key, right.Key) && equalTypePointer(left.Value, right.Value)
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

type NullState string

const (
	MaybeNull NullState = "MaybeNull"
	Null      NullState = "Null"
	NonNull   NullState = "NonNull"
)
