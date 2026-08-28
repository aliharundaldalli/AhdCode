package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

// goType maps an IR type to its Go runtime representation.
//
// Scalar slots are boxed only when the declared null-state allows null.
// Reference types are already nil-able, so both representations coincide.
// Collection elements and Pair values always use the nullable representation
// because element access is MaybeNull in the frontend null-state model; Pair
// keys are never null and always use the non-null representation.
func (generator *generator) goType(value ir.Type, nullable bool) string {
	base := generator.plainType(value)
	if base == "" {
		return ""
	}
	if !nullable || !isScalar(value) {
		return base
	}
	return "*" + base
}

func isScalar(value ir.Type) bool {
	switch value.Kind {
	case ir.IntType, ir.RealType, ir.StringType, ir.BoolType:
		return true
	default:
		return false
	}
}

func (generator *generator) plainType(value ir.Type) string {
	switch value.Kind {
	case ir.IntType:
		return "int64"
	case ir.RealType:
		return "float64"
	case ir.StringType:
		return "string"
	case ir.BoolType:
		return "bool"
	case ir.ListType:
		if value.Element == nil {
			return ""
		}
		element := generator.goType(*value.Element, true)
		if element == "" {
			return ""
		}
		return "*AhdList[" + element + "]"
	case ir.PairType:
		if value.Key == nil || value.Value == nil {
			return ""
		}
		key := generator.goType(*value.Key, false)
		item := generator.goType(*value.Value, true)
		if key == "" || item == "" {
			return ""
		}
		return "*AhdPair[" + key + ", " + item + "]"
	case ir.ClassType:
		if value.Class == "" {
			return ""
		}
		return "*" + generator.className(value.Class)
	case ir.FunctionType:
		return generator.functionType(value.Signature)
	default:
		return ""
	}
}

// functionType maps a concrete Function signature. Signature-level null-state
// is not part of the IR Function type contract, so Function values use the
// non-null representation for their parameters and result.
func (generator *generator) functionType(signature *ir.Signature) string {
	if signature == nil {
		return ""
	}
	parts := make([]string, 0, len(signature.Parameters))
	for _, parameter := range signature.Parameters {
		rendered := generator.goType(parameter.Type, false)
		if rendered == "" {
			return ""
		}
		parts = append(parts, rendered)
	}
	result := "func(" + strings.Join(parts, ", ") + ")"
	if signature.Return.Kind == ir.NothingType {
		return result
	}
	returned := generator.goType(signature.Return, false)
	if returned == "" {
		return ""
	}
	return result + " " + returned
}

func (generator *generator) className(id ir.ClassID) string {
	return mangle(classPrefix, string(id))
}

func (generator *generator) fieldName(id ir.FieldID) string {
	return mangle(fieldPrefix, string(id))
}
