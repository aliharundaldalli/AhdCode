package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const pdfModulePrefix = "builtin:PDF::"

var (
	pdfDocumentClass       = ir.ClassID("builtin:PDF::class::PDFDocument")
	pdfDocumentBlocksField = ir.FieldID("builtin:PDF::class::PDFDocument::field::blocks")
)

// pdfCall lowers the PDF module's factory functions: new(), fromWord(...),
// and fromExcel(...). fromWord/fromExcel read another module's own class
// value (Word.Document / Excel.Workbook) through that module's existing
// unwrap helper -- documentOf and excelDataOf already exist for exactly this
// (Word's and Excel's own operations), so PDF adds no new cross-module
// plumbing beyond calling them.
func (generator *generator) pdfCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), pdfModulePrefix)
	switch name {
	case "new":
		return generator.pdfDocumentFrom("AhdPDFNew()", meta)
	case "fromWord":
		if len(value.Arguments) == 0 || value.Arguments[0].Value == nil {
			return generator.unsupported("PDF.fromWord without a Document argument", meta.Span)
		}
		document := generator.documentOf(value.Arguments[0].Value)
		return generator.pdfDocumentFrom("AhdPDFFromWord("+document+")", meta)
	case "fromExcel":
		if len(value.Arguments) == 0 || value.Arguments[0].Value == nil {
			return generator.unsupported("PDF.fromExcel without a Workbook argument", meta.Span)
		}
		workbook := generator.excelDataOf(excelWorkbookClass, value.Arguments[0].Value)
		return generator.pdfDocumentFrom("AhdPDFFromExcel("+workbook+")", meta)
	default:
		return generator.unsupported("PDF function "+name, meta.Span)
	}
}

// pdfDocumentFrom builds a PDFDocument instance from one runtime
// AhdPDFDocument reading, the same way documentFrom builds a Document.
func (generator *generator) pdfDocumentFrom(document string, meta ir.ExprBase) string {
	helper, ok := generator.pdfHelper()
	if !ok {
		return generator.unsupported("a PDFDocument value without its Class declaration", meta.Span)
	}
	return helper + "(" + document + ")"
}

// pdfDocumentOf evaluates one PDFDocument expression exactly once and reads
// its one hidden storage field into the runtime interchange shape.
func (generator *generator) pdfDocumentOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	blocks := "value." + generator.fieldName(pdfDocumentBlocksField) + "_get().Snapshot()"
	return "func(value " + generator.interfaceName(pdfDocumentClass) + ") AhdPDFDocument { " +
		"return AhdPDFDocument{Blocks: " + blocks + "} }(" + rendered + ")"
}

func (generator *generator) pdfHelper() (string, bool) {
	if generator.layouts[pdfDocumentClass] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[pdfDocumentClass]; known {
		return name, true
	}
	name := mangleNamed("ph_", generator.classDisplayName(pdfDocumentClass), string(pdfDocumentClass))
	generator.timeHelpers[pdfDocumentClass] = name
	return name, true
}

// emitPDFHelpers writes the PDFDocument wrapper, turning a runtime reading
// into a constructed AhdCode value.
func (generator *generator) emitPDFHelpers(writer *emitter) {
	name, known := generator.timeHelpers[pdfDocumentClass]
	if !known {
		return
	}
	layout := generator.layouts[pdfDocumentClass]
	if layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	writer.line("// PDFDocument value built from one runtime block-list reading.")
	writer.open("func " + name + "(document AhdPDFDocument) " + generator.interfaceName(pdfDocumentClass) + " {")
	writer.line("return " + generator.callableName(constructor) + "(AhdNewList(document.Blocks...))")
	writer.close("}")
	writer.blank()
}

// pdfOperation lowers the built-in members of PDFDocument. Every member
// reaches this through the ordinary type-operation path, so PDF adds no
// static-method or operator semantics to the language. save() is the one
// operation that ever invokes the shared Tectonic renderer, so it alone
// marks the program as requiring the staged Latex/Tectonic runtime -- the
// exact same flag and staging Latex.pdf itself uses.
func (generator *generator) pdfOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	receiver := generator.pdfDocumentOf(value.Callee)
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
	case "PDFDocument.heading":
		return generator.pdfDocumentFrom("AhdPDFHeading("+receiver+", "+text(0, `""`)+", "+integer(1, "int64(0)")+")", meta)
	case "PDFDocument.paragraph":
		return generator.pdfDocumentFrom("AhdPDFParagraph("+receiver+", "+text(0, `""`)+", "+text(1, `"left"`)+", "+
			boolean(2, "false")+", "+boolean(3, "false")+", "+boolean(4, "false")+")", meta)
	case "PDFDocument.table":
		return generator.pdfDocumentFrom("AhdPDFTable("+receiver+", "+list(0, "AhdNewList[string]()")+", "+
			list(1, "AhdNewList[*AhdList[string]]()")+", "+text(2, `"left"`)+")", meta)
	case "PDFDocument.image":
		return generator.pdfDocumentFrom("AhdPDFImage("+receiver+", "+text(0, `""`)+", "+
			generator.pdfSizePair(value, 1)+")", meta)
	case "PDFDocument.pageBreak":
		return generator.pdfDocumentFrom("AhdPDFPageBreak("+receiver+")", meta)
	case "PDFDocument.save":
		generator.usesLatex = true
		return "AhdPDFSave(" + receiver + ", " + text(0, `""`) + ")"
	default:
		return generator.unsupported("PDFDocument operation "+name, meta.Span)
	}
}

func (generator *generator) pdfSizePair(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
		return "AhdBuildPair([]string{}, []float64{})"
	}
	return generator.expr(value.Arguments[index].Value)
}
