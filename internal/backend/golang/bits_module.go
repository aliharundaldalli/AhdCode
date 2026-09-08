package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const bitsModulePrefix = "builtin:Bits::"

var bitsErrorClass = ir.ClassID("builtin:Bits::class::BitsError")

// bitsCall lowers the Bits module's functions. Like Security, Bits publishes no
// data-carrying Class -- every call maps to a plain Int runtime function. The
// runtime implementation lives in ahdruntime/bits.go.
//
// Only the shift and rotate helpers receive the error class: they are the only
// operations that can be given an out-of-range argument.
func (generator *generator) bitsCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), bitsModulePrefix)
	errorClass := generator.descriptorName(bitsErrorClass)
	number := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	switch name {
	case "bitAnd":
		return "AhdBitsAnd(" + number(0) + ", " + number(1) + ")"
	case "bitOr":
		return "AhdBitsOr(" + number(0) + ", " + number(1) + ")"
	case "bitXor":
		return "AhdBitsXor(" + number(0) + ", " + number(1) + ")"
	case "bitNot":
		return "AhdBitsNot(" + number(0) + ")"
	case "shiftLeft":
		return "AhdBitsShiftLeft(" + errorClass + ", " + number(0) + ", " + number(1) + ")"
	case "shiftRight":
		return "AhdBitsShiftRight(" + errorClass + ", " + number(0) + ", " + number(1) + ")"
	case "shiftRightUnsigned":
		return "AhdBitsShiftRightUnsigned(" + errorClass + ", " + number(0) + ", " + number(1) + ")"
	case "rotateLeft":
		return "AhdBitsRotateLeft(" + errorClass + ", " + number(0) + ", " + number(1) + ")"
	case "rotateRight":
		return "AhdBitsRotateRight(" + errorClass + ", " + number(0) + ", " + number(1) + ")"
	case "count":
		return "AhdBitsCount(" + number(0) + ")"
	case "leadingZeros":
		return "AhdBitsLeadingZeros(" + number(0) + ")"
	case "trailingZeros":
		return "AhdBitsTrailingZeros(" + number(0) + ")"
	default:
		return generator.unsupported("Bits function "+name, meta.Span)
	}
}
