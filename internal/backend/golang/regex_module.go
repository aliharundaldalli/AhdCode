package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const regexModulePrefix = "builtin:Regex::"

var (
	regexClassID      = ir.ClassID("builtin:Regex::class::Pattern")
	regexErrorClass   = ir.ClassID("builtin:Regex::class::RegexError")
	regexPatternField = ir.FieldID(string(regexClassID) + "::field::pattern")
)

// regexCall lowers the Regex standard module's one module-root function,
// compile. The pattern is validated (and cached) before the ordinary
// generated Regex constructor runs, so an invalid pattern never reaches a
// half-built instance.
func (generator *generator) regexCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), regexModulePrefix)
	if name != "compile" {
		return generator.unsupported("Regex function "+name, meta.Span)
	}
	if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
		generator.fail(CodeGenerationFailure, "Regex.compile has a missing argument", meta.Span, "the IR call is malformed")
		return "nil"
	}
	constructor := generator.functions[ir.CallableID(string(regexClassID)+"::constructor::(pattern:String)->Nothing")]
	if constructor == nil {
		return generator.unsupported("a Regex value without its Class declaration", meta.Span)
	}
	pattern := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
	errorClass := generator.descriptorName(regexErrorClass)
	return generator.callableName(constructor) + "(AhdRegexValidate(" + errorClass + ", " + pattern + "))"
}

// regexOperation lowers the built-in members of the Regex Class. Reached
// through the ordinary type-operation path, so AhdCode gains no
// static-method or operator-overloading semantics from it.
func (generator *generator) regexOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	if value.Callee == nil {
		generator.fail(CodeGenerationFailure, name+" has no receiver", meta.Span, "the IR call is malformed")
		return "nil"
	}
	errorClass := generator.descriptorName(regexErrorClass)
	pattern := generator.expr(value.Callee) + "." + generator.fieldName(regexPatternField) + "_get()"
	argument := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return `""`
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "Regex.matches":
		return "AhdRegexMatches(" + errorClass + ", " + pattern + ", " + argument(0) + ")"
	case "Regex.find":
		return "AhdRegexFind(" + errorClass + ", " + pattern + ", " + argument(0) + ")"
	case "Regex.findAll":
		return "AhdRegexFindAll(" + errorClass + ", " + pattern + ", " + argument(0) + ")"
	case "Regex.groups":
		return "AhdRegexGroups(" + errorClass + ", " + pattern + ", " + argument(0) + ")"
	case "Regex.replace":
		return "AhdRegexReplace(" + errorClass + ", " + pattern + ", " + argument(0) + ", " + argument(1) + ")"
	case "Regex.split":
		return "AhdRegexSplit(" + errorClass + ", " + pattern + ", " + argument(0) + ")"
	default:
		return generator.unsupported("Regex operation "+name, meta.Span)
	}
}
