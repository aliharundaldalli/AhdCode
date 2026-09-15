package golang

import (
	"strings"

	"ahdcode/internal/ir"
	"ahdcode/internal/source"
)

const terminalModulePrefix = "builtin:Terminal::"

var terminalErrorClass = ir.ClassID("builtin:Terminal::class::TerminalError")

// terminalCall lowers the Terminal module. Every fixed-signature function maps
// to one runtime function over standard output or standard error; an omitted
// default argument arrives without a value and takes the documented default
// here. pretty is the module's one type-directed operation.
func (generator *generator) terminalCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), terminalModulePrefix)
	errorClass := generator.descriptorName(terminalErrorClass)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	boolean := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return "false"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	switch name {
	case "emit":
		if len(value.Arguments) == 0 || value.Arguments[0].Value == nil {
			return generator.malformed("Terminal.emit", meta)
		}
		return "AhdTerminalEmit(" + generator.expr(value.Arguments[0].Value) + ", " + text(1, `" "`) + ", " + text(2, `"\n"`) + ")"
	case "error":
		return "AhdTerminalError(" + text(0, `""`) + ", " + text(1, `"\n"`) + ")"
	case "flush":
		return "AhdTerminalFlush(" + errorClass + ")"
	case "isInteractive":
		return "AhdTerminalIsInteractive()"
	case "width":
		return "AhdTerminalWidth()"
	case "height":
		return "AhdTerminalHeight()"
	case "supportsColor":
		return "AhdTerminalSupportsColor()"
	case "style":
		return "AhdTerminalStyleText(" + errorClass + ", " + text(0, `""`) + ", " + text(1, `"default"`) + ", " +
			text(2, `"default"`) + ", " + boolean(3) + ", " + boolean(4) + ")"
	case "pretty":
		return generator.terminalPretty(value)
	default:
		return generator.unsupported("Terminal function "+name, meta.Span)
	}
}

// terminalPretty lays out a List or Pair from its static IR type and writes
// every other value exactly as write does, through the same generator.text
// choke point.
func (generator *generator) terminalPretty(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
		return generator.malformed("Terminal.pretty", meta)
	}
	argument := value.Arguments[0].Value
	argumentMeta := argument.ExprMeta()
	switch argumentMeta.Type.Kind {
	case ir.ListType, ir.PairType:
		layout := generator.prettyFunc(argumentMeta.Type, naturalNullable(argument), argumentMeta.Span)
		return "AhdTerminalPretty(" + layout + "(" + generator.expr(argument) + ", \"\"))"
	}
	return "AhdTerminalPretty(" + generator.text(argument) + ")"
}

// prettyFunc returns a Go expression of type func(T, string) string that lays
// out one value at a given indentation. It follows renderFunc's structure: a
// List or Pair recurses, and every other value is its nested canonical text.
func (generator *generator) prettyFunc(value ir.Type, nullable bool, span source.Span) string {
	switch value.Kind {
	case ir.ListType:
		if value.Element == nil {
			return generator.unsupported("pretty layout for an untyped List", span)
		}
		element := generator.goType(*value.Element, value.ElementNullable)
		return "AhdTerminalPrettyList[" + element + "](" + generator.prettyFunc(*value.Element, value.ElementNullable, span) + ")"
	case ir.PairType:
		if value.Key == nil || value.Value == nil {
			return generator.unsupported("pretty layout for an untyped Pair", span)
		}
		key, item := generator.goType(*value.Key, false), generator.goType(*value.Value, value.ValueNullable)
		return "AhdTerminalPrettyPair[" + key + ", " + item + "](" + generator.renderFunc(*value.Key, false, true, span) + ", " +
			generator.prettyFunc(*value.Value, value.ValueNullable, span) + ")"
	}
	return "AhdTerminalPrettyLeaf[" + generator.goType(value, nullable) + "](" + generator.renderFunc(value, nullable, true, span) + ")"
}
