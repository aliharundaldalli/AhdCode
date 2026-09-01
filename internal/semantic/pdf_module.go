package semantic

import (
	"fmt"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const pdfModuleID = "builtin:PDF"

var (
	pdfErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	pdfErrorClass    = &types.ClassSymbol{ModuleID: pdfModuleID, Name: "PDFError", Parent: pdfErrorParent}
	pdfDocumentClass = &types.ClassSymbol{ModuleID: pdfModuleID, Name: "PDFDocument"}
)

// PDFErrorIdentity and PDFDocumentIdentity expose the canonical identities to
// the lowering layer without coupling the public module interface to a
// backend.
func PDFErrorIdentity() *types.ClassSymbol    { return pdfErrorClass }
func PDFDocumentIdentity() *types.ClassSymbol { return pdfDocumentClass }

// PDFDocumentOperations names the members a PDFDocument publishes through
// built-in type operations, so has/has not reports what a PDFDocument really
// offers, and the lowering layer's IR Class stays in agreement with the
// frontend about the published surface. PDFDocument is deliberately smaller
// than Word's Document: it publishes no read-only accessors, keeping the
// public surface to the minimal construction/publication operations the
// v0.1.20 release calls for.
var PDFDocumentOperations = []string{"heading", "paragraph", "table", "image", "pageBreak", "save"}

func pdfDocumentType() types.Type { return types.Class{Symbol: pdfDocumentClass} }

func pdfModuleInterface() *ModuleInterface {
	module := standardInterface(pdfModuleID, "PDF")

	errorSymbol := &Symbol{
		Name: "PDFError", Kind: ClassSymbol, Class: pdfErrorClass,
		Type: types.Class{Symbol: pdfErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: pdfModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[pdfModuleID+"\x00PDFError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	documentSymbol := &Symbol{
		Name: "PDFDocument", Kind: ClassSymbol, Class: pdfDocumentClass,
		Type: types.Class{Symbol: pdfDocumentClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: pdfModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[pdfModuleID+"\x00PDFDocument"] = documentSymbol
	addStandardExport(module, documentSymbol)

	document := pdfDocumentType()
	addStandardExport(module, standardFunction(pdfModuleID, "new", document))
	addStandardExport(module, standardFunction(pdfModuleID, "fromWord", document,
		types.Parameter{Name: "document", Type: types.Class{Symbol: WordDocumentIdentity()}}))
	addStandardExport(module, standardFunction(pdfModuleID, "fromExcel", document,
		types.Parameter{Name: "workbook", Type: types.Class{Symbol: ExcelWorkbookIdentity()}}))

	return module
}

// pdfConstructionHint names the PDF functions that produce a PDFDocument, so
// direct construction has an actionable message instead of a generic
// missing-constructor diagnostic.
func pdfConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != pdfModuleID || identity.Name != "PDFDocument" {
		return "", false
	}
	return "create a PDFDocument with PDF.new(), PDF.fromWord(document), or PDF.fromExcel(workbook)", true
}

// pdfOperationShape is the call shape of one PDFDocument member. Every member
// is positional-only, matching Word's Document, String, List, Table, and
// Chart convention.
type pdfOperationShape struct {
	parameters []types.Type
	minimum    int
	result     types.Type
	hint       string
}

func pdfSizePairType() types.Type { return types.Pair{Key: types.String, Value: types.Real} }

func pdfOperationShapes() map[TypeOperation]pdfOperationShape {
	document := pdfDocumentType()
	none := []types.Type{}
	headers := types.List{Element: types.String}
	rows := types.List{Element: types.List{Element: types.String}}
	return map[TypeOperation]pdfOperationShape{
		PDFDocumentHeading: {[]types.Type{types.String, types.Int}, 2, document,
			"pass the heading text and an Int level from 1 to 6"},
		PDFDocumentParagraph: {[]types.Type{types.String, types.String, types.Bool, types.Bool, types.Bool}, 1, document,
			"pass the paragraph text, and optionally align, bold, italic, and underline in that order"},
		PDFDocumentTable: {[]types.Type{headers, rows, types.String}, 2, document,
			"pass headers and rows, and optionally align"},
		PDFDocumentImage: {[]types.Type{types.String, pdfSizePairType()}, 1, document,
			"pass the image path, and optionally a Pair<String, Real> of width/height in centimeters"},
		PDFDocumentPageBreak: {none, 0, document, "call pageBreak with no argument"},
		PDFDocumentSave:      {[]types.Type{types.String}, 1, types.Nothing, "pass the destination .pdf path"},
	}
}

var pdfOperationNames = map[string]TypeOperation{
	"heading": PDFDocumentHeading, "paragraph": PDFDocumentParagraph,
	"table": PDFDocumentTable, "image": PDFDocumentImage,
	"pageBreak": PDFDocumentPageBreak, "save": PDFDocumentSave,
}

// pdfOperationFor names the built-in member a PDFDocument instance publishes.
// Only the compiler-supplied PDFDocument identity matches, so a user Class
// named PDFDocument never collides with it.
func pdfOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil ||
		class.Symbol.ModuleID != pdfModuleID || class.Symbol.Name != "PDFDocument" {
		return "", false
	}
	operation, known := pdfOperationNames[name]
	return operation, known
}

// analyzePDFOperation checks one PDFDocument member. Arguments are NonNull
// values of the declared type; a trailing defaulted argument may be omitted,
// but only from the end - paragraph's align/bold/italic/underline and
// table's align can only be provided in their declared order.
func (a *analyzer) analyzePDFOperation(call *ast.CallExpr, operation TypeOperation, shape pdfOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) < shape.minimum || len(call.Arguments) > len(shape.parameters) {
		a.error(codeCallArguments, pdfArityMessage(operation, shape, len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, argument := range call.Arguments {
		expected := shape.parameters[index]
		info := a.analyzeExpressionExpected(argument.Value, current, flow, expected)
		if info.invalid() {
			continue
		}
		if info.nullState != NonNull {
			a.nullableError(string(operation), argument.Value, info.nullState)
			continue
		}
		if !types.Assignable(expected, info.typeValue) {
			a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" argument")
		}
	}
	return result
}

func pdfArityMessage(operation TypeOperation, shape pdfOperationShape, received int) string {
	if shape.minimum == len(shape.parameters) {
		return fmt.Sprintf("%s expects %d argument(s); received %d", operation, shape.minimum, received)
	}
	return fmt.Sprintf("%s expects %d to %d argument(s); received %d",
		operation, shape.minimum, len(shape.parameters), received)
}
