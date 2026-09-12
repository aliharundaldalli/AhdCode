package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// QR and Barcode (v1.3.0) are machine-readable codes. Each module creates one
// immutable value -- a QRCode or a BarcodeCode -- that exposes the logical
// symbol (a module matrix or a bar pattern) and writes it as PNG or SVG. The
// same encoder serves Latex.qr, Latex.barcode, PDFDocument.qr, and
// PDFDocument.barcode, so a symbol means the same thing everywhere.

const (
	qrModuleID      = "builtin:QR"
	barcodeModuleID = "builtin:Barcode"
)

var (
	codesErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	qrErrorClass      = &types.ClassSymbol{ModuleID: qrModuleID, Name: "QRError", Parent: codesErrorParent}
	qrCodeClass       = &types.ClassSymbol{ModuleID: qrModuleID, Name: "QRCode"}
	barcodeErrorClass = &types.ClassSymbol{ModuleID: barcodeModuleID, Name: "BarcodeError", Parent: codesErrorParent}
	barcodeCodeClass  = &types.ClassSymbol{ModuleID: barcodeModuleID, Name: "BarcodeCode"}
)

// Identities exposed to lowering without coupling the public module
// interfaces to a backend.
func QRErrorIdentity() *types.ClassSymbol      { return qrErrorClass }
func QRCodeIdentity() *types.ClassSymbol       { return qrCodeClass }
func BarcodeErrorIdentity() *types.ClassSymbol { return barcodeErrorClass }
func BarcodeCodeIdentity() *types.ClassSymbol  { return barcodeCodeClass }

const (
	QRCodeValue   TypeOperation = "QRCode.value"
	QRCodeLevel   TypeOperation = "QRCode.level"
	QRCodeSize    TypeOperation = "QRCode.size"
	QRCodeMatrix  TypeOperation = "QRCode.matrix"
	QRCodeSavePNG TypeOperation = "QRCode.savePNG"
	QRCodeSaveSVG TypeOperation = "QRCode.saveSVG"

	BarcodeCodeKind    TypeOperation = "BarcodeCode.kind"
	BarcodeCodeValue   TypeOperation = "BarcodeCode.value"
	BarcodeCodePattern TypeOperation = "BarcodeCode.pattern"
	BarcodeCodeSavePNG TypeOperation = "BarcodeCode.savePNG"
	BarcodeCodeSaveSVG TypeOperation = "BarcodeCode.saveSVG"
)

// QRCodeOperations and BarcodeCodeOperations name the members each value
// publishes through built-in type operations, so has/has not reports the real
// surface and the IR Class agrees with the frontend.
var (
	QRCodeOperations      = []string{"value", "level", "size", "matrix", "savePNG", "saveSVG"}
	BarcodeCodeOperations = []string{"kind", "value", "pattern", "savePNG", "saveSVG"}
)

func codesModuleClasses(module *ModuleInterface, moduleID string, classes ...*types.ClassSymbol) {
	for _, identity := range classes {
		symbol := &Symbol{
			Name: identity.Name, Kind: ClassSymbol, Class: identity,
			Type: types.Class{Symbol: identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: moduleID,
			Members: make(map[string]*Symbol),
		}
		if identity.Parent != nil {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[moduleID+"\x00"+identity.Name] = symbol
		addStandardExport(module, symbol)
	}
}

func qrModuleInterface() *ModuleInterface {
	module := standardInterface(qrModuleID, "QR")
	codesModuleClasses(module, qrModuleID, qrErrorClass, qrCodeClass)
	addStandardExport(module, standardFunction(qrModuleID, "create", types.Class{Symbol: qrCodeClass},
		types.Parameter{Name: "value", Type: types.String},
		types.Parameter{Name: "level", Type: types.String, HasDefault: true}))
	sort.Strings(module.ExportNames)
	return module
}

func barcodeModuleInterface() *ModuleInterface {
	module := standardInterface(barcodeModuleID, "Barcode")
	codesModuleClasses(module, barcodeModuleID, barcodeErrorClass, barcodeCodeClass)
	for _, name := range []string{"code128", "ean13", "upca"} {
		addStandardExport(module, standardFunction(barcodeModuleID, name, types.Class{Symbol: barcodeCodeClass},
			types.Parameter{Name: "value", Type: types.String}))
	}
	sort.Strings(module.ExportNames)
	return module
}

func qrConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity != qrCodeClass {
		return "", false
	}
	return "create a QRCode with QR.create(value) or QR.create(value, level)", true
}

func barcodeConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity != barcodeCodeClass {
		return "", false
	}
	return "create a BarcodeCode with Barcode.code128(value), Barcode.ean13(value), or Barcode.upca(value)", true
}

// codesOperationShape is the call shape of one QRCode or BarcodeCode member.
// Members are positional-only; trailing defaulted arguments may be omitted.
type codesOperationShape struct {
	parameters []types.Type
	minimum    int
	result     types.Type
	hint       string
}

func codesOperationShapes() map[TypeOperation]codesOperationShape {
	none := []types.Type{}
	matrix := types.List{Element: types.List{Element: types.Bool}}
	return map[TypeOperation]codesOperationShape{
		QRCodeValue:  {none, 0, types.String, "call value with no argument"},
		QRCodeLevel:  {none, 0, types.String, "call level with no argument"},
		QRCodeSize:   {none, 0, types.Int, "call size with no argument"},
		QRCodeMatrix: {none, 0, matrix, "call matrix with no argument"},
		QRCodeSavePNG: {[]types.Type{types.String, types.Int}, 1, types.Nothing,
			"pass the destination .png path, and optionally the image side in pixels"},
		QRCodeSaveSVG: {[]types.Type{types.String, types.Real}, 1, types.Nothing,
			"pass the destination .svg path, and optionally the side length in centimeters"},
		BarcodeCodeKind:    {none, 0, types.String, "call kind with no argument"},
		BarcodeCodeValue:   {none, 0, types.String, "call value with no argument"},
		BarcodeCodePattern: {none, 0, types.List{Element: types.Bool}, "call pattern with no argument"},
		BarcodeCodeSavePNG: {[]types.Type{types.String, types.Int, types.Int}, 1, types.Nothing,
			"pass the destination .png path, and optionally the width and height in pixels"},
		BarcodeCodeSaveSVG: {[]types.Type{types.String, types.Real, types.Real}, 1, types.Nothing,
			"pass the destination .svg path, and optionally the width and height in centimeters"},
	}
}

var codesOperationNames = map[*types.ClassSymbol]map[string]TypeOperation{
	qrCodeClass: {
		"value": QRCodeValue, "level": QRCodeLevel, "size": QRCodeSize, "matrix": QRCodeMatrix,
		"savePNG": QRCodeSavePNG, "saveSVG": QRCodeSaveSVG,
	},
	barcodeCodeClass: {
		"kind": BarcodeCodeKind, "value": BarcodeCodeValue, "pattern": BarcodeCodePattern,
		"savePNG": BarcodeCodeSavePNG, "saveSVG": BarcodeCodeSaveSVG,
	},
}

// codesOperationFor names the built-in member a QRCode or BarcodeCode value
// publishes. Only the compiler-supplied identities match, so a user Class
// with the same name never collides with them.
func codesOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil {
		return "", false
	}
	operation, known := codesOperationNames[class.Symbol][name]
	return operation, known
}

// analyzeCodesOperation checks one QRCode or BarcodeCode member call.
// Arguments are NonNull values of the declared type; a trailing defaulted
// argument may be omitted, but only from the end.
func (a *analyzer) analyzeCodesOperation(call *ast.CallExpr, operation TypeOperation, shape codesOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) < shape.minimum || len(call.Arguments) > len(shape.parameters) {
		message := fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments))
		if shape.minimum != len(shape.parameters) {
			message = fmt.Sprintf("%s expects %d to %d argument(s); received %d", operation, shape.minimum, len(shape.parameters), len(call.Arguments))
		}
		a.error(codeCallArguments, message, call.Span(), shape.hint)
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
