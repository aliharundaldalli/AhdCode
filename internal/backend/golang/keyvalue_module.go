package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const keyValueModulePrefix = "builtin:KeyValue::"

var keyValueErrorClass = ir.ClassID("builtin:KeyValue::class::KeyValueError")

// keyValueCall lowers the KeyValue module's type-directed operations, the same
// way listsCall does for Lists.
func (generator *generator) keyValueCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), keyValueModulePrefix)
	errorClass := generator.descriptorName(keyValueErrorClass)
	if name == "combine" {
		keys, _, keysOK := generator.collectionOperand(value, 0, "KeyValue.combine", ir.ListType)
		values, _, valuesOK := generator.collectionOperand(value, 1, "KeyValue.combine", ir.ListType)
		if !keysOK || !valuesOK {
			return "nil"
		}
		return "AhdKeyValueCombine(" + errorClass + ", " + keys + ", " + values + ")"
	}
	pair, pairType, ok := generator.collectionOperand(value, 0, "KeyValue."+name, ir.PairType)
	if !ok {
		return "nil"
	}
	key, item, itemNullable := *pairType.Key, *pairType.Value, pairType.ValueNullable
	// keyAt and valueAt render one argument in the exact representation the
	// Pair's own key or value slot uses, so no boxed/unboxed mismatch reaches
	// the runtime helper.
	keyAt := func(index int) string { return generator.value(value.Arguments[index].Value, key, false) }
	switch name {
	case "keys":
		return "AhdKeyValueKeys(" + pair + ")"
	case "values":
		return "AhdKeyValueValues(" + pair + ")"
	case "with":
		if len(value.Arguments) != 3 || value.Arguments[1].Value == nil || value.Arguments[2].Value == nil {
			return generator.malformed("KeyValue.with", meta)
		}
		return "AhdKeyValueWith(" + pair + ", " + keyAt(1) + ", " +
			generator.value(value.Arguments[2].Value, item, itemNullable) + ")"
	case "without":
		if len(value.Arguments) != 2 || value.Arguments[1].Value == nil {
			return generator.malformed("KeyValue.without", meta)
		}
		return "AhdKeyValueWithout(" + pair + ", " + keyAt(1) + ")"
	case "select", "drop":
		keys, _, keysOK := generator.collectionOperand(value, 1, "KeyValue."+name, ir.ListType)
		if !keysOK {
			return "nil"
		}
		helper := "AhdKeyValueSelect"
		if name == "drop" {
			helper = "AhdKeyValueDrop"
		}
		return helper + "(" + errorClass + ", " + pair + ", " + keys + ")"
	case "rename":
		if len(value.Arguments) != 3 || value.Arguments[1].Value == nil || value.Arguments[2].Value == nil {
			return generator.malformed("KeyValue.rename", meta)
		}
		return "AhdKeyValueRename(" + errorClass + ", " + pair + ", " + keyAt(1) + ", " + keyAt(2) + ")"
	case "mapValues":
		if len(value.Arguments) != 2 || value.Arguments[1].Value == nil || meta.Type.Value == nil {
			return generator.malformed("KeyValue.mapValues", meta)
		}
		transform := generator.adaptElementCallback(value.Arguments[1].Value, item, itemNullable,
			*meta.Type.Value, meta.Type.ValueNullable)
		return "AhdKeyValueMapValues(" + pair + ", " + transform + ")"
	case "merge", "overlay":
		other, _, otherOK := generator.collectionOperand(value, 1, "KeyValue."+name, ir.PairType)
		if !otherOK {
			return "nil"
		}
		if name == "overlay" {
			return "AhdKeyValueOverlay(" + pair + ", " + other + ")"
		}
		return "AhdKeyValueMerge(" + errorClass + ", " + pair + ", " + other + ")"
	default:
		return generator.unsupported("KeyValue operation "+name, meta.Span)
	}
}
