package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// UUIDModuleID is the compiler-supplied identity of the UUID module.
const UUIDModuleID = "builtin:UUID"

const (
	uuidValueClassID = ir.ClassID(UUIDModuleID + "::class::UUIDValue")
	uuidErrorClassID = ir.ClassID(UUIDModuleID + "::class::UUIDError")
)

// UUIDValueDataFieldID is the hidden String holding a UUIDValue's canonical
// lowercase text, the same representation QRCode and BarcodeCode use.
var UUIDValueDataFieldID = ir.FieldID(string(uuidValueClassID) + "::field::data")

func uuidModule(id ir.ModuleID, name, path string) *ir.Module {
	return codesModule(id, name, path, uuidValueClassID, "UUIDValue", UUIDValueDataFieldID, semantic.UUIDValueOperations, uuidErrorClassID, "UUIDError")
}
