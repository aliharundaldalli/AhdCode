package semantic

import (
	"fmt"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const wordModuleID = "builtin:Word"

var (
	wordErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	wordErrorClass    = &types.ClassSymbol{ModuleID: wordModuleID, Name: "WordError", Parent: wordErrorParent}
	wordDocumentClass = &types.ClassSymbol{ModuleID: wordModuleID, Name: "Document"}
)

// WordErrorIdentity and WordDocumentIdentity expose the canonical identities
// to the lowering layer without coupling the public module interface to a
// backend.
func WordErrorIdentity() *types.ClassSymbol    { return wordErrorClass }
func WordDocumentIdentity() *types.ClassSymbol { return wordDocumentClass }

// WordDocumentOperations names the members a Document publishes through
// built-in type operations, so has/has not reports what a Document really
// offers, and the lowering layer's IR Class stays in agreement with the
// frontend about the published surface.
var WordDocumentOperations = []string{
	"heading", "paragraph", "table", "image", "pageBreak", "save",
	"text", "paragraphs", "headings", "tables",
}

func wordDocumentType() types.Type { return types.Class{Symbol: wordDocumentClass} }

func wordModuleInterface() *ModuleInterface {
	module := standardInterface(wordModuleID, "Word")

	errorSymbol := &Symbol{
		Name: "WordError", Kind: ClassSymbol, Class: wordErrorClass,
		Type: types.Class{Symbol: wordErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: wordModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[wordModuleID+"\x00WordError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	documentSymbol := &Symbol{
		Name: "Document", Kind: ClassSymbol, Class: wordDocumentClass,
		Type: types.Class{Symbol: wordDocumentClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: wordModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[wordModuleID+"\x00Document"] = documentSymbol
	addStandardExport(module, documentSymbol)

	document := wordDocumentType()
	addStandardExport(module, standardFunction(wordModuleID, "new", document))
	addStandardExport(module, standardFunction(wordModuleID, "read", document,
		types.Parameter{Name: "path", Type: types.String}))

	return module
}

// wordConstructionHint names the Word functions that produce a Document, so
// direct construction has an actionable message instead of a generic
// missing-constructor diagnostic.
func wordConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != wordModuleID || identity.Name != "Document" {
		return "", false
	}
	return "create a Document with Word.new() or Word.read(path), or derive one from an existing Document", true
}

// wordOperationShape is the call shape of one Document member. Every member
// is positional-only: AhdCode's built-in type operations publish no
// parameter names (the same restriction String, List, Table, and Chart
// already carry), so a Document member follows that established convention
// rather than introducing named-argument support for one module.
type wordOperationShape struct {
	parameters []types.Type
	minimum    int
	result     types.Type
	hint       string
}

func wordSizePairType() types.Type { return types.Pair{Key: types.String, Value: types.Real} }

func wordMergesType() types.Type {
	return types.List{Element: types.List{Element: types.Int}}
}

func wordOperationShapes() map[TypeOperation]wordOperationShape {
	document := wordDocumentType()
	none := []types.Type{}
	headers := types.List{Element: types.String}
	rows := types.List{Element: types.List{Element: types.String}}
	tables := types.List{Element: types.List{Element: types.List{Element: types.String}}}
	return map[TypeOperation]wordOperationShape{
		WordDocumentHeading: {[]types.Type{types.String, types.Int}, 2, document,
			"pass the heading text and an Int level from 1 to 6"},
		WordDocumentParagraph: {[]types.Type{types.String, types.String, types.Bool, types.Bool, types.Bool}, 1, document,
			"pass the paragraph text, and optionally align, bold, italic, and underline in that order"},
		WordDocumentTable: {[]types.Type{headers, rows, wordMergesType(), types.String}, 2, document,
			"pass headers and rows, and optionally a merge descriptor list and align"},
		WordDocumentImage: {[]types.Type{types.String, wordSizePairType()}, 1, document,
			"pass the image path, and optionally a Pair<String, Real> of width/height in centimeters"},
		WordDocumentPageBreak: {none, 0, document, "call pageBreak with no argument"},
		WordDocumentSave:      {[]types.Type{types.String}, 1, types.Nothing, "pass the destination .docx path"},
		WordDocumentText:      {none, 0, types.String, "call text with no argument"},
		WordDocumentParagraphs: {none, 0, types.List{Element: types.String},
			"call paragraphs with no argument"},
		WordDocumentHeadings: {none, 0, types.List{Element: types.String}, "call headings with no argument"},
		WordDocumentTables:   {none, 0, tables, "call tables with no argument"},
	}
}

var wordOperationNames = map[string]TypeOperation{
	"heading": WordDocumentHeading, "paragraph": WordDocumentParagraph,
	"table": WordDocumentTable, "image": WordDocumentImage,
	"pageBreak": WordDocumentPageBreak, "save": WordDocumentSave,
	"text": WordDocumentText, "paragraphs": WordDocumentParagraphs,
	"headings": WordDocumentHeadings, "tables": WordDocumentTables,
}

// wordOperationFor names the built-in member a Document instance publishes.
// Only the compiler-supplied Document identity matches, so a user Class
// named Document never collides with it.
func wordOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil ||
		class.Symbol.ModuleID != wordModuleID || class.Symbol.Name != "Document" {
		return "", false
	}
	operation, known := wordOperationNames[name]
	return operation, known
}

// analyzeWordOperation checks one Document member. Arguments are NonNull
// values of the declared type; a trailing defaulted argument may be omitted,
// but only from the end - paragraph's align/bold/italic/underline and
// table's merges/align can only be provided in their declared order.
func (a *analyzer) analyzeWordOperation(call *ast.CallExpr, operation TypeOperation, shape wordOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) < shape.minimum || len(call.Arguments) > len(shape.parameters) {
		a.error(codeCallArguments, wordArityMessage(operation, shape, len(call.Arguments)), call.Span(), shape.hint)
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

func wordArityMessage(operation TypeOperation, shape wordOperationShape, received int) string {
	if shape.minimum == len(shape.parameters) {
		return fmt.Sprintf("%s expects %d argument(s); received %d", operation, shape.minimum, received)
	}
	return fmt.Sprintf("%s expects %d to %d argument(s); received %d",
		operation, shape.minimum, len(shape.parameters), received)
}
