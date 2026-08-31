package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

// goType maps an IR type to its Go runtime representation.
//
// Scalar slots are boxed only when the declared null-state allows null.
// Reference types are already nil-able, so both representations coincide.
// A List's element and a Pair's value are boxed only when the type itself
// says so (List<T?>/Pair<K, V?>); List<T>/Pair<K, V> store unboxed values.
// Pair keys are never null and always use the non-null representation.
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
	case ir.IntType, ir.RealType, ir.ComplexType, ir.StringType, ir.BoolType:
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
	case ir.ComplexType:
		return "complex128"
	case ir.StringType:
		return "string"
	case ir.BoolType:
		return "bool"
	case ir.ListType:
		if value.Element == nil {
			return ""
		}
		element := generator.goType(*value.Element, value.ElementNullable)
		if element == "" {
			return ""
		}
		return "*AhdList[" + element + "]"
	case ir.PairType:
		if value.Key == nil || value.Value == nil {
			return ""
		}
		key := generator.goType(*value.Key, false)
		item := generator.goType(*value.Value, value.ValueNullable)
		if key == "" || item == "" {
			return ""
		}
		return "*AhdPair[" + key + ", " + item + "]"
	case ir.ClassType:
		if value.Class == "" {
			return ""
		}
		// A Class instance is represented by its generated interface, so a
		// subclass value keeps one object identity when it is upcast.
		return generator.interfaceName(value.Class)
	case ir.FunctionType:
		return generator.functionType(value.Signature)
	case ir.RangeType:
		return "*AhdRange"
	default:
		return ""
	}
}

// functionType is the uniform Go type of a Function value. Its scalar
// parameters and result always use the nullable representation, because an IR
// Function type carries no per-parameter or return null-state; a concrete
// callable is adapted to this shape when it is taken as a value.
func (generator *generator) functionType(signature *ir.Signature) string {
	if signature == nil {
		return ""
	}
	parts := make([]string, 0, len(signature.Parameters))
	for _, parameter := range signature.Parameters {
		rendered := generator.goType(parameter.Type, true)
		if rendered == "" {
			return ""
		}
		parts = append(parts, rendered)
	}
	result := "func(" + strings.Join(parts, ", ") + ")"
	if signature.Return.Kind == ir.NothingType {
		return result
	}
	returned := generator.goType(signature.Return, true)
	if returned == "" {
		return ""
	}
	return result + " " + returned
}

func (generator *generator) className(id ir.ClassID) string {
	return mangleNamed(classPrefix, generator.classDisplayName(id), string(id))
}

func (generator *generator) interfaceName(id ir.ClassID) string {
	return mangleNamed(interfacePrefix, generator.classDisplayName(id), string(id))
}

func (generator *generator) descriptorName(id ir.ClassID) string {
	return mangleNamed(descriptorPrefix, generator.classDisplayName(id), string(id))
}

func (generator *generator) classDisplayName(id ir.ClassID) string {
	if class := generator.classes[id]; class != nil {
		return class.Name
	}
	return string(id)
}

func (generator *generator) fieldName(id ir.FieldID) string {
	name := string(id)
	if field, known := generator.fields[id]; known {
		name = field.Name
	}
	return mangleNamed(fieldPrefix, name, string(id))
}

// slotName is the interface method that implements one dispatch slot. It is
// derived from the callable that introduced the slot, so a parent method and
// every override share one Go method name.
func (generator *generator) slotName(id ir.CallableID) string {
	root := generator.slotRoot(id)
	name := string(root)
	if function := generator.functions[root]; function != nil {
		name = function.Name
	}
	return mangleNamed(slotPrefix, name, string(root))
}

// slotRoot walks the recorded Override chain to the callable that introduced
// the dispatch slot.
func (generator *generator) slotRoot(id ir.CallableID) ir.CallableID {
	seen := make(map[ir.CallableID]bool)
	for {
		function := generator.functions[id]
		if function == nil || function.Overrides == "" || seen[id] {
			return id
		}
		seen[id] = true
		id = function.Overrides
	}
}
