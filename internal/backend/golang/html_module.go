package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const htmlModulePrefix = "builtin:HTML::"

var (
	htmlNodeClass     = ir.ClassID("builtin:HTML::class::HTMLNode")
	htmlNodeDataField = ir.FieldID("builtin:HTML::class::HTMLNode::field::data")
	htmlErrorClass    = ir.ClassID("builtin:HTML::class::HTMLError")
)

func (generator *generator) htmlCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), htmlModulePrefix)
	errorClass := generator.descriptorName(htmlErrorClass)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "text":
		return generator.htmlNodeFrom("AhdHTMLText("+text(0, `""`)+")", meta)
	case "element":
		return generator.htmlNodeFrom("AhdHTMLElement("+errorClass+", "+text(0, `""`)+", "+
			generator.htmlAttributeKeys(value, 1)+", "+generator.htmlAttributeValues(value, 1)+", "+
			generator.htmlNodeTexts(value, 2)+")", meta)
	case "render":
		return "AhdHTMLRender(" + errorClass + ", " + generator.htmlNodeOf(value.Arguments[0].Value) + ")"
	case "document":
		return "AhdHTMLDocument(" + errorClass + ", " + text(0, `""`) + ", " + generator.htmlNodeTexts(value, 1) + ")"
	default:
		return generator.unsupported("HTML function "+name, meta.Span)
	}
}

func (generator *generator) htmlNodeOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	data := "value." + generator.fieldName(htmlNodeDataField) + "_get()"
	return "func(value " + generator.interfaceName(htmlNodeClass) + ") string { return " + data + " }(" + rendered + ")"
}

func (generator *generator) htmlAttributeKeys(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	return "func(pair *AhdPair[string, string]) []string { return pair.Keys() }(" + rendered + ")"
}

func (generator *generator) htmlAttributeValues(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	return "func(pair *AhdPair[string, string]) []string { " +
		"keys := pair.Keys(); values := make([]string, len(keys)); " +
		"for index, key := range keys { values[index] = pair.Get(key) }; " +
		"return values }(" + rendered + ")"
}

func (generator *generator) htmlNodeTexts(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(htmlNodeClass)
	getter := generator.fieldName(htmlNodeDataField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { " +
		"items := list.Snapshot(); result := make([]string, len(items)); " +
		"for index, item := range items { result[index] = item." + getter + " }; " +
		"return result }(" + rendered + ")"
}

func (generator *generator) htmlNodeFrom(data string, meta ir.ExprBase) string {
	helper, ok := generator.htmlHelper()
	if !ok {
		return generator.unsupported("an HTMLNode without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) htmlHelper() (string, bool) {
	if generator.layouts[htmlNodeClass] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[htmlNodeClass]; known {
		return name, true
	}
	name := mangleNamed("html_", generator.classDisplayName(htmlNodeClass), string(htmlNodeClass))
	generator.timeHelpers[htmlNodeClass] = name
	return name, true
}

func (generator *generator) emitHTMLHelpers(writer *emitter) {
	name, known := generator.timeHelpers[htmlNodeClass]
	if !known {
		return
	}
	layout := generator.layouts[htmlNodeClass]
	if layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	writer.open("func " + name + "(data string) " + generator.interfaceName(htmlNodeClass) + " {")
	writer.line("return " + generator.callableName(constructor) + "(data)")
	writer.close("}")
	writer.blank()
}
