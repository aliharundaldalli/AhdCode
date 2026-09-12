package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// QRModuleID and BarcodeModuleID are the compiler-supplied identities of the
// machine-readable code modules.
const (
	QRModuleID      = "builtin:QR"
	BarcodeModuleID = "builtin:Barcode"
)

const (
	qrCodeClassID       = ir.ClassID(QRModuleID + "::class::QRCode")
	qrErrorClassID      = ir.ClassID(QRModuleID + "::class::QRError")
	barcodeCodeClassID  = ir.ClassID(BarcodeModuleID + "::class::BarcodeCode")
	barcodeErrorClassID = ir.ClassID(BarcodeModuleID + "::class::BarcodeError")
)

// QRCodeDataFieldID and BarcodeCodeDataFieldID are the hidden String holding a
// validated symbol, the representation SMTPClient and Scheduler use for their
// own runtime state.
var (
	QRCodeDataFieldID      = ir.FieldID(string(qrCodeClassID) + "::field::data")
	BarcodeCodeDataFieldID = ir.FieldID(string(barcodeCodeClassID) + "::field::data")
)

func qrModule(id ir.ModuleID, name, path string) *ir.Module {
	return codesModule(id, name, path, qrCodeClassID, "QRCode", QRCodeDataFieldID, semantic.QRCodeOperations, qrErrorClassID, "QRError")
}

func barcodeModule(id ir.ModuleID, name, path string) *ir.Module {
	return codesModule(id, name, path, barcodeCodeClassID, "BarcodeCode", BarcodeCodeDataFieldID, semantic.BarcodeCodeOperations, barcodeErrorClassID, "BarcodeError")
}

func codesModule(id ir.ModuleID, name, path string, codeClassID ir.ClassID, codeName string, field ir.FieldID, operations []string, errorClassID ir.ClassID, errorName string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	data := ir.Field{ID: field, Name: "data", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
	code := &ir.Class{
		ID: codeClassID, Symbol: ir.SymbolID(string(codeClassID) + "::symbol"), Name: codeName,
		Operations: operations, Fields: []ir.Field{data},
		Constructor: ir.CallableID(string(codeClassID) + "::constructor::(data:String)->Nothing"),
	}
	module.Classes = append(module.Classes, code)
	module.Functions = append(module.Functions, smtpValueConstructor(code))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: errorClassID, Symbol: ir.SymbolID(string(errorClassID) + "::symbol"),
		Name: errorName, Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(errorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}
