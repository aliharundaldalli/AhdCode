package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const xmlModulePrefix = "builtin:XML::"

var (
	xmlNodeClass         = ir.ClassID("builtin:XML::class::XMLNode")
	xmlNodeDataField     = ir.FieldID("builtin:XML::class::XMLNode::field::data")
	xmlDocumentClass     = ir.ClassID("builtin:XML::class::XMLDocument")
	xmlDocumentDataField = ir.FieldID("builtin:XML::class::XMLDocument::field::data")
	xmlErrorClass        = ir.ClassID("builtin:XML::class::XMLError")
)

// xmlCall lowers the XML module's plain functions.
func (generator *generator) xmlCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), xmlModulePrefix)
	errorClass := generator.descriptorName(xmlErrorClass)
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
	case "text":
		return generator.xmlNodeFrom("AhdXMLText("+text(0, `""`)+")", meta)
	case "element":
		return generator.xmlNodeFrom("AhdXMLElement("+errorClass+", "+text(0, `""`)+", "+
			generator.xmlAttributeKeys(value, 1)+", "+generator.xmlAttributeValues(value, 1)+", "+
			generator.xmlNodeTexts(value, 2)+")", meta)
	case "document":
		return generator.xmlDocumentFrom("AhdXMLDocument("+errorClass+", "+generator.xmlNodeOf(value.Arguments[0].Value)+")", meta)
	case "parse":
		return generator.xmlDocumentFrom("AhdXMLParse("+errorClass+", "+text(0, `""`)+")", meta)
	case "read":
		return generator.xmlDocumentFrom("AhdXMLRead("+errorClass+", "+text(0, `""`)+")", meta)
	case "stringify":
		return "AhdXMLStringify(" + errorClass + ", " + generator.xmlDocumentOf(value.Arguments[0].Value) + ", " + boolean(1, "false") + ")"
	case "write":
		return "AhdXMLWrite(" + errorClass + ", " + generator.xmlDocumentOf(value.Arguments[0].Value) + ", " +
			text(1, `""`) + ", " + boolean(2, "false") + ")"
	default:
		return generator.unsupported("XML function "+name, meta.Span)
	}
}

func (generator *generator) xmlNodeOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	data := "value." + generator.fieldName(xmlNodeDataField) + "_get()"
	return "func(value " + generator.interfaceName(xmlNodeClass) + ") string { return " + data + " }(" + rendered + ")"
}

func (generator *generator) xmlDocumentOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	data := "value." + generator.fieldName(xmlDocumentDataField) + "_get()"
	return "func(value " + generator.interfaceName(xmlDocumentClass) + ") string { return " + data + " }(" + rendered + ")"
}

// xmlAttributeKeys/xmlAttributeValues evaluate a Pair<String, String>
// argument exactly once each and return its ordered keys/values.
func (generator *generator) xmlAttributeKeys(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	return "func(pair *AhdPair[string, string]) []string { return pair.Keys() }(" + rendered + ")"
}

func (generator *generator) xmlAttributeValues(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	return "func(pair *AhdPair[string, string]) []string { " +
		"keys := pair.Keys(); values := make([]string, len(keys)); " +
		"for index, key := range keys { values[index] = pair.Get(key) }; " +
		"return values }(" + rendered + ")"
}

// xmlNodeTexts evaluates a List<XMLNode> argument exactly once and returns
// a []string of every element's own encoded data, in order.
func (generator *generator) xmlNodeTexts(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(xmlNodeClass)
	getter := generator.fieldName(xmlNodeDataField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { " +
		"items := list.Snapshot(); result := make([]string, len(items)); " +
		"for index, item := range items { result[index] = item." + getter + " }; " +
		"return result }(" + rendered + ")"
}

func (generator *generator) xmlNodeFrom(data string, meta ir.ExprBase) string {
	helper, ok := generator.xmlHelper(xmlNodeClass, "xn_")
	if !ok {
		return generator.unsupported("an XMLNode without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) xmlDocumentFrom(data string, meta ir.ExprBase) string {
	helper, ok := generator.xmlHelper(xmlDocumentClass, "xd_")
	if !ok {
		return generator.unsupported("an XMLDocument without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) xmlHelper(classID ir.ClassID, prefix string) (string, bool) {
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

// emitXMLHelpers writes the XMLNode and XMLDocument wrappers, each turning
// one encoded-data reading into a constructed AhdCode value.
func (generator *generator) emitXMLHelpers(writer *emitter) {
	generator.emitXMLHelperFor(writer, xmlNodeClass)
	generator.emitXMLHelperFor(writer, xmlDocumentClass)
}

func (generator *generator) emitXMLHelperFor(writer *emitter, classID ir.ClassID) {
	name, known := generator.timeHelpers[classID]
	if !known {
		return
	}
	layout := generator.layouts[classID]
	if layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	writer.line("// " + generator.classDisplayName(classID) + " built from one runtime encoded-data reading.")
	writer.open("func " + name + "(data string) " + generator.interfaceName(classID) + " {")
	writer.line("return " + generator.callableName(constructor) + "(data)")
	writer.close("}")
	writer.blank()
}

// xmlOperation lowers the built-in members of XMLNode and XMLDocument.
func (generator *generator) xmlOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(xmlErrorClass)
	if name == "XMLDocument.root" {
		return generator.xmlNodeFrom(generator.xmlDocumentOf(value.Callee), meta)
	}
	receiver := generator.xmlNodeOf(value.Callee)
	switch name {
	case "XMLNode.kind":
		return "AhdXMLKind(" + errorClass + ", " + receiver + ")"
	case "XMLNode.name":
		return "AhdXMLName(" + errorClass + ", " + receiver + ")"
	case "XMLNode.namespace":
		return "AhdXMLNamespace(" + errorClass + ", " + receiver + ")"
	case "XMLNode.text":
		return "AhdXMLNodeText(" + errorClass + ", " + receiver + ")"
	case "XMLNode.attribute":
		key := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
		return "AhdXMLAttribute(" + errorClass + ", " + receiver + ", " + key + ")"
	case "XMLNode.attributes":
		return "AhdBuildPair(AhdXMLAttributeKeys(" + errorClass + ", " + receiver + "), AhdXMLAttributeValues(" + errorClass + ", " + receiver + "))"
	case "XMLNode.children":
		return generator.xmlNodeListResult("AhdXMLChildrenData("+errorClass+", "+receiver+")", meta)
	case "XMLNode.elements":
		return generator.xmlNodeListResult("AhdXMLElementsData("+errorClass+", "+receiver+")", meta)
	default:
		return generator.unsupported("XMLNode operation "+name, meta.Span)
	}
}

// xmlNodeListResult wraps a []string of encoded node data back into a
// List<XMLNode>.
func (generator *generator) xmlNodeListResult(dataExpr string, meta ir.ExprBase) string {
	helper, ok := generator.xmlHelper(xmlNodeClass, "xn_")
	if !ok {
		return generator.unsupported("an XMLNode without its Class declaration", meta.Span)
	}
	element := generator.interfaceName(xmlNodeClass)
	return "func(items []string) *AhdList[" + element + "] { " +
		"result := make([]" + element + ", len(items)); " +
		"for index, data := range items { result[index] = " + helper + "(data) }; " +
		"return AhdNewList(result...) }(" + dataExpr + ")"
}
