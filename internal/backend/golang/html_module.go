package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const htmlModulePrefix = "builtin:HTML::"

var (
	htmlNodeClass         = ir.ClassID("builtin:HTML::class::HTMLNode")
	htmlNodeDataField     = ir.FieldID("builtin:HTML::class::HTMLNode::field::data")
	htmlDocumentClass     = ir.ClassID("builtin:HTML::class::HTMLDocument")
	htmlDocumentDataField = ir.FieldID("builtin:HTML::class::HTMLDocument::field::data")
	htmlElementClass      = ir.ClassID("builtin:HTML::class::HTMLElement")
	htmlElementDataField  = ir.FieldID("builtin:HTML::class::HTMLElement::field::data")
	htmlErrorClass        = ir.ClassID("builtin:HTML::class::HTMLError")
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
	case "parse":
		return generator.htmlDocumentFrom("AhdHTMLParse("+errorClass+", "+text(0, `""`)+")", meta)
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
	helper, ok := generator.htmlHelper(htmlNodeClass, "html_")
	if !ok {
		return generator.unsupported("an HTMLNode without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) htmlDocumentFrom(data string, meta ir.ExprBase) string {
	helper, ok := generator.htmlHelper(htmlDocumentClass, "hd_")
	if !ok {
		return generator.unsupported("an HTMLDocument without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) htmlElementFrom(data string, meta ir.ExprBase) string {
	helper, ok := generator.htmlHelper(htmlElementClass, "he_")
	if !ok {
		return generator.unsupported("an HTMLElement without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) htmlDocumentOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	data := "value." + generator.fieldName(htmlDocumentDataField) + "_get()"
	return "func(value " + generator.interfaceName(htmlDocumentClass) + ") string { return " + data + " }(" + rendered + ")"
}

func (generator *generator) htmlElementOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	data := "value." + generator.fieldName(htmlElementDataField) + "_get()"
	return "func(value " + generator.interfaceName(htmlElementClass) + ") string { return " + data + " }(" + rendered + ")"
}

func (generator *generator) htmlHelper(classID ir.ClassID, prefix string) (string, bool) {
	if generator.layouts[classID] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[classID]; known {
		return name, true
	}
	name := mangleNamed(prefix, generator.classDisplayName(classID), string(classID))
	generator.timeHelpers[classID] = name
	return name, true
}

func (generator *generator) emitHTMLHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{htmlNodeClass, htmlDocumentClass, htmlElementClass} {
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
		writer.open("func " + name + "(data string) " + generator.interfaceName(class) + " {")
		writer.line("return " + generator.callableName(constructor) + "(data)")
		writer.close("}")
		writer.blank()
	}
}

// htmlOperation lowers the built-in members of HTMLDocument and HTMLElement.
func (generator *generator) htmlOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(htmlErrorClass)
	selector := func() string {
		return generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "HTMLDocument.select":
		return generator.htmlElementListResult("AhdHTMLDocumentSelect("+errorClass+", "+generator.htmlDocumentOf(value.Callee)+", "+selector()+")", meta)
	case "HTMLDocument.first":
		return generator.htmlOptionalElementResult("AhdHTMLDocumentFirst("+errorClass+", "+generator.htmlDocumentOf(value.Callee)+", "+selector()+")", meta)
	case "HTMLElement.tag":
		return "AhdHTMLElementTag(" + errorClass + ", " + generator.htmlElementOf(value.Callee) + ")"
	case "HTMLElement.text":
		return "AhdHTMLElementText(" + errorClass + ", " + generator.htmlElementOf(value.Callee) + ")"
	case "HTMLElement.attr":
		return "AhdHTMLElementAttr(" + errorClass + ", " + generator.htmlElementOf(value.Callee) + ", " + selector() + ")"
	case "HTMLElement.hasAttr":
		return "AhdHTMLElementHasAttr(" + errorClass + ", " + generator.htmlElementOf(value.Callee) + ", " + selector() + ")"
	case "HTMLElement.select":
		return generator.htmlElementListResult("AhdHTMLElementSelect("+errorClass+", "+generator.htmlElementOf(value.Callee)+", "+selector()+")", meta)
	case "HTMLElement.first":
		return generator.htmlOptionalElementResult("AhdHTMLElementFirst("+errorClass+", "+generator.htmlElementOf(value.Callee)+", "+selector()+")", meta)
	default:
		return generator.unsupported("HTML operation "+name, meta.Span)
	}
}

// htmlElementListResult wraps a []string of encoded element data back into
// a List<HTMLElement>.
func (generator *generator) htmlElementListResult(dataExpr string, meta ir.ExprBase) string {
	helper, ok := generator.htmlHelper(htmlElementClass, "he_")
	if !ok {
		return generator.unsupported("an HTMLElement without its Class declaration", meta.Span)
	}
	element := generator.interfaceName(htmlElementClass)
	return "func(items []string) *AhdList[" + element + "] { " +
		"result := make([]" + element + ", len(items)); " +
		"for index, data := range items { result[index] = " + helper + "(data) }; " +
		"return AhdNewList(result...) }(" + dataExpr + ")"
}

// htmlOptionalElementResult wraps a *string (nil when no match) back into a
// nullable HTMLElement, represented as a nil interface when absent.
func (generator *generator) htmlOptionalElementResult(dataExpr string, meta ir.ExprBase) string {
	helper, ok := generator.htmlHelper(htmlElementClass, "he_")
	if !ok {
		return generator.unsupported("an HTMLElement without its Class declaration", meta.Span)
	}
	element := generator.interfaceName(htmlElementClass)
	return "func(data *string) " + element + " { if data == nil { return nil }; return " + helper + "(*data) }(" + dataExpr + ")"
}
