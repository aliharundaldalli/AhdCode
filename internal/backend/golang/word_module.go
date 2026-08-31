package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const wordModulePrefix = "builtin:Word::"

var (
	wordDocumentClass       = ir.ClassID("builtin:Word::class::Document")
	wordDocumentBlocksField = ir.FieldID("builtin:Word::class::Document::field::blocks")
)

// wordCall lowers the Word module's two factory functions.
func (generator *generator) wordCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), wordModulePrefix)
	argument := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
			return `""`
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "new":
		return generator.documentFrom("AhdWordNew()", meta)
	case "read":
		return generator.documentFrom("AhdWordRead("+argument(0)+")", meta)
	default:
		return generator.unsupported("Word function "+name, meta.Span)
	}
}

// documentFrom builds a Document instance from one runtime AhdWordDocument
// reading, the same way tableFrom builds a Table.
func (generator *generator) documentFrom(document string, meta ir.ExprBase) string {
	helper, ok := generator.wordHelper()
	if !ok {
		return generator.unsupported("a Document value without its Class declaration", meta.Span)
	}
	return helper + "(" + document + ")"
}

// documentOf evaluates one Document expression exactly once and reads its
// one hidden storage field into the runtime interchange shape.
func (generator *generator) documentOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	blocks := "value." + generator.fieldName(wordDocumentBlocksField) + "_get().Snapshot()"
	return "func(value " + generator.interfaceName(wordDocumentClass) + ") AhdWordDocument { " +
		"return AhdWordDocument{Blocks: " + blocks + "} }(" + rendered + ")"
}

func (generator *generator) wordHelper() (string, bool) {
	if generator.layouts[wordDocumentClass] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[wordDocumentClass]; known {
		return name, true
	}
	name := mangleNamed("wh_", generator.classDisplayName(wordDocumentClass), string(wordDocumentClass))
	generator.timeHelpers[wordDocumentClass] = name
	return name, true
}

// emitWordHelpers writes the Document wrapper, turning a runtime reading
// into a constructed AhdCode value.
func (generator *generator) emitWordHelpers(writer *emitter) {
	name, known := generator.timeHelpers[wordDocumentClass]
	if !known {
		return
	}
	layout := generator.layouts[wordDocumentClass]
	if layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	writer.line("// Document value built from one runtime block-list reading.")
	writer.open("func " + name + "(document AhdWordDocument) " + generator.interfaceName(wordDocumentClass) + " {")
	writer.line("return " + generator.callableName(constructor) + "(AhdNewList(document.Blocks...))")
	writer.close("}")
	writer.blank()
}

// wordOperation lowers the built-in members of Document. Every member
// reaches this through the ordinary type-operation path, so Word adds no
// static-method or operator semantics to the language.
func (generator *generator) wordOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	receiver := generator.documentOf(value.Callee)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	integer := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	boolean := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	list := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.expr(value.Arguments[index].Value)
	}
	switch name {
	case "Document.heading":
		return generator.documentFrom("AhdWordHeading("+receiver+", "+text(0, `""`)+", "+integer(1, "int64(0)")+")", meta)
	case "Document.paragraph":
		return generator.documentFrom("AhdWordParagraph("+receiver+", "+text(0, `""`)+", "+text(1, `"left"`)+", "+
			boolean(2, "false")+", "+boolean(3, "false")+", "+boolean(4, "false")+")", meta)
	case "Document.table":
		return generator.documentFrom("AhdWordTable("+receiver+", "+list(0, "AhdNewList[string]()")+", "+
			list(1, "AhdNewList[*AhdList[string]]()")+", "+list(2, "AhdNewList[*AhdList[int64]]()")+", "+
			text(3, `"left"`)+")", meta)
	case "Document.image":
		return generator.documentFrom("AhdWordImage("+receiver+", "+text(0, `""`)+", "+
			generator.wordSizePair(value, 1)+")", meta)
	case "Document.pageBreak":
		return generator.documentFrom("AhdWordPageBreak("+receiver+")", meta)
	case "Document.save":
		return "AhdWordSave(" + receiver + ", " + text(0, `""`) + ")"
	case "Document.text":
		return "AhdWordText(" + receiver + ")"
	case "Document.paragraphs":
		return "AhdWordParagraphs(" + receiver + ")"
	case "Document.headings":
		return "AhdWordHeadings(" + receiver + ")"
	case "Document.tables":
		return "AhdWordTables(" + receiver + ")"
	default:
		return generator.unsupported("Document operation "+name, meta.Span)
	}
}

func (generator *generator) wordSizePair(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
		return "AhdBuildPair([]string{}, []float64{})"
	}
	return generator.expr(value.Arguments[index].Value)
}
