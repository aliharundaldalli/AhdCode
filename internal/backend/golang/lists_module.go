package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const listsModulePrefix = "builtin:Lists::"

var listsErrorClass = ir.ClassID("builtin:Lists::class::ListsError")

// listsCall lowers the Lists module's type-directed operations. The frontend
// has already specialized every call site to one exact concrete signature, so
// this layer reads the argument and result IR types it recorded and lets Go's
// own generic inference bind the runtime helper's type parameters. Nothing is
// erased to an interface: each emitted call is a fully typed instantiation.
func (generator *generator) listsCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), listsModulePrefix)
	source, sourceType, ok := generator.collectionOperand(value, 0, "Lists."+name, ir.ListType)
	if !ok {
		return "nil"
	}
	element, elementNullable := *sourceType.Element, sourceType.ElementNullable
	errorClass := generator.descriptorName(listsErrorClass)
	switch name {
	case "chunk":
		if len(value.Arguments) != 2 || value.Arguments[1].Value == nil {
			return generator.malformed("Lists.chunk", meta)
		}
		size := generator.value(value.Arguments[1].Value, ir.Type{Kind: ir.IntType}, false)
		return "AhdListsChunk(" + errorClass + ", " + source + ", " + size + ")"
	case "flatten":
		return "AhdListsFlatten(" + source + ")"
	case "transpose":
		return "AhdListsTranspose(" + errorClass + ", " + source + ")"
	case "unique":
		return "AhdListsUnique(" + source + ", " + generator.equalFunc(element, elementNullable, meta.Span) + ")"
	case "valueCounts":
		return "AhdListsValueCounts(" + source + ")"
	case "groupBy":
		if len(value.Arguments) != 2 || value.Arguments[1].Value == nil || meta.Type.Key == nil {
			return generator.malformed("Lists.groupBy", meta)
		}
		// A Pair key is never null, so the adapted key Function returns the
		// unboxed representation the runtime Pair indexes by.
		key := generator.adaptElementCallback(value.Arguments[1].Value, element, elementNullable, *meta.Type.Key, false)
		return "AhdListsGroupBy(" + source + ", " + key + ")"
	default:
		return generator.unsupported("Lists operation "+name, meta.Span)
	}
}

// collectionOperand renders one List or Pair argument and hands back its fully
// parameterized IR type, which the caller uses to derive element, key, and
// value representations.
func (generator *generator) collectionOperand(value *ir.CallExpr, index int, operation string, kind ir.TypeKind) (string, ir.Type, bool) {
	meta := value.ExprMeta()
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		generator.fail(CodeGenerationFailure, operation+" has a missing argument", meta.Span, "the IR call is malformed")
		return "nil", ir.Type{}, false
	}
	operand := value.Arguments[index].Value.ExprMeta().Type
	if operand.Kind != kind ||
		(kind == ir.ListType && operand.Element == nil) ||
		(kind == ir.PairType && (operand.Key == nil || operand.Value == nil)) {
		generator.fail(CodeGenerationFailure, operation+" has no fully typed collection argument", meta.Span, "the IR call is malformed")
		return "nil", ir.Type{}, false
	}
	return generator.expr(value.Arguments[index].Value), operand, true
}

func (generator *generator) malformed(operation string, meta ir.ExprBase) string {
	generator.fail(CodeGenerationFailure, operation+" has a malformed argument list", meta.Span, "the IR call is malformed")
	return "nil"
}
