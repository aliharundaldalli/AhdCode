package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const templatesModulePrefix = "builtin:Templates::"

var (
	templateDataClass     = ir.ClassID("builtin:Templates::class::TemplateData")
	templateDataTextField = ir.FieldID("builtin:Templates::class::TemplateData::field::text")
	templateSetClass      = ir.ClassID("builtin:Templates::class::TemplateSet")
	templateSetTextField  = ir.FieldID("builtin:Templates::class::TemplateSet::field::text")
	templateErrorClass    = ir.ClassID("builtin:Templates::class::TemplateError")
)

func (generator *generator) templatesCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), templatesModulePrefix)
	errorClass := generator.descriptorName(templateErrorClass)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	boolean := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	integer := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	real := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.RealType}, false)
	}
	switch name {
	case "nullValue":
		return generator.templateDataFrom("AhdTemplatesNull()", meta)
	case "fromBool":
		return generator.templateDataFrom("AhdTemplatesFromBool("+boolean(0)+")", meta)
	case "fromInt":
		return generator.templateDataFrom("AhdTemplatesFromInt("+integer(0)+")", meta)
	case "fromReal":
		return generator.templateDataFrom("AhdTemplatesFromReal("+errorClass+", "+real(0)+")", meta)
	case "fromString":
		return generator.templateDataFrom("AhdTemplatesFromString("+text(0, `""`)+")", meta)
	case "trusted":
		return generator.templateDataFrom("AhdTemplatesTrusted("+text(0, `""`)+")", meta)
	case "list":
		return generator.templateDataFrom("AhdTemplatesList("+generator.templateDataTexts(value, 0)+")", meta)
	case "object":
		return generator.templateDataFrom(generator.templateObjectCall(errorClass, value, 0), meta)
	case "parse":
		return generator.templateSetFrom("AhdTemplatesParse("+errorClass+", "+text(0, `""`)+", "+text(1, `""`)+")", meta)
	case "sources":
		return generator.templateSetFrom("AhdTemplatesSources("+errorClass+", "+generator.templateStringList(value, 0)+", "+generator.templateStringList(value, 1)+")", meta)
	case "files":
		return generator.templateSetFrom("AhdTemplatesFiles("+errorClass+", "+text(0, `""`)+")", meta)
	default:
		return generator.unsupported("Templates function "+name, meta.Span)
	}
}

func (generator *generator) templatesOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(templateErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	handle := generator.templateFieldOf(templateSetClass, templateSetTextField, value.Callee)
	data := func(index int) string {
		return generator.templateFieldOf(templateDataClass, templateDataTextField, value.Arguments[index].Value)
	}
	switch name {
	case "TemplateSet.render":
		return "AhdTemplatesRender(" + errorClass + ", " + handle + ", " + text(0) + ", " + data(1) + ")"
	case "TemplateSet.layout":
		return "AhdTemplatesLayout(" + errorClass + ", " + handle + ", " + text(0) + ", " + text(1) + ", " + data(2) + ")"
	default:
		return generator.unsupported("Templates operation "+name, meta.Span)
	}
}

func (generator *generator) templateDataFrom(text string, meta ir.ExprBase) string {
	return generator.templateWrap(templateDataClass, text, meta)
}

func (generator *generator) templateSetFrom(text string, meta ir.ExprBase) string {
	return generator.templateWrap(templateSetClass, text, meta)
}

func (generator *generator) templateWrap(class ir.ClassID, text string, meta ir.ExprBase) string {
	helper, ok := generator.templateHelper(class)
	if !ok {
		return generator.unsupported("a template value without its Class declaration", meta.Span)
	}
	return helper + "(" + text + ")"
}

func (generator *generator) templateHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("th_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

func (generator *generator) emitTemplateHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{templateDataClass, templateSetClass} {
		name, known := generator.timeHelpers[class]
		if !known {
			continue
		}
		layout := generator.layouts[class]
		if layout == nil {
			continue
		}
		constructor := generator.functions[layout.class.Constructor]
		if constructor == nil {
			continue
		}
		writer.line("// Template value built from one runtime text.")
		writer.open("func " + name + "(text string) " + generator.interfaceName(class) + " {")
		writer.line("return " + generator.callableName(constructor) + "(text)")
		writer.close("}")
		writer.blank()
	}
}

func (generator *generator) templateFieldOf(class ir.ClassID, field ir.FieldID, expression ir.Expr) string {
	rendered := generator.expr(expression)
	getter := "value." + generator.fieldName(field) + "_get()"
	return "func(value " + generator.interfaceName(class) + ") string { return " + getter + " }(" + rendered + ")"
}

func (generator *generator) templateDataTexts(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(templateDataClass)
	getter := generator.fieldName(templateDataTextField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { " +
		"items := list.Snapshot(); result := make([]string, len(items)); " +
		"for index, item := range items { result[index] = item." + getter + " }; " +
		"return result }(" + rendered + ")"
}

// templateObjectCall evaluates the Pair argument exactly once, then reads
// keys and values from that one result. Splitting Keys() and Get() across two
// expressions would re-run a call that builds the map.
func (generator *generator) templateObjectCall(errorClass string, value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "AhdTemplatesObject(" + errorClass + ", nil, nil)"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(templateDataClass)
	getter := generator.fieldName(templateDataTextField) + "_get()"
	return "func(pair *AhdPair[string, " + element + "]) string { " +
		"keys := pair.Keys(); values := make([]string, len(keys)); " +
		"for index, key := range keys { values[index] = pair.Get(key)." + getter + " }; " +
		"return AhdTemplatesObject(" + errorClass + ", keys, values) }(" + rendered + ")"
}

func (generator *generator) templateStringList(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	return "func(list *AhdList[string]) []string { return list.Snapshot() }(" + rendered + ")"
}
