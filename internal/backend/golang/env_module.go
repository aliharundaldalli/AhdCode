package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const envModulePrefix = "builtin:Env::"

var envErrorClass = ir.ClassID("builtin:Env::class::EnvError")

// envCall lowers the Env module's functions. Env publishes no data-carrying
// Class, so every call maps straight to a plain String/Bool/Nothing/Pair
// runtime function - there is no receiver-wrapping helper machinery to
// build, unlike Word/JSON/XML.
func (generator *generator) envCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), envModulePrefix)
	errorClass := generator.descriptorName(envErrorClass)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	boolean := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	switch name {
	case "get":
		return "AhdEnvGet(" + text(0, `""`) + ")"
	case "getOr":
		return "AhdEnvGetOr(" + text(0, `""`) + ", " + text(1, `""`) + ")"
	case "exists":
		return "AhdEnvHas(" + text(0, `""`) + ")"
	case "set":
		return "AhdEnvSet(" + errorClass + ", " + text(0, `""`) + ", " + text(1, `""`) + ")"
	case "unset":
		return "AhdEnvUnset(" + errorClass + ", " + text(0, `""`) + ")"
	case "read":
		return "func(entries []AhdEnvEntry) *AhdPair[string, string] { " +
			"keys := make([]string, len(entries)); values := make([]string, len(entries)); " +
			"for index, entry := range entries { keys[index] = entry.Key; values[index] = entry.Value }; " +
			"return AhdBuildPair(keys, values) }(AhdEnvReadEntries(" + errorClass + ", " + text(0, `""`) + "))"
	case "load":
		return "AhdEnvLoad(" + errorClass + ", " + text(0, `""`) + ", " + boolean(1, "false") + ")"
	default:
		return generator.unsupported("Env function "+name, meta.Span)
	}
}
