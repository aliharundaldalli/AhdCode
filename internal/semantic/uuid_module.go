package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

// UUID (v1.4.0) creates RFC 9562 identifiers. A UUIDValue is an immutable,
// opaque value produced by UUID.v4, UUID.v7, UUID.parse, or UUID.zero; it never
// converts implicitly to or from String. Like DateTime, a UUIDValue has no
// Class protocol methods: == keeps built-in Class identity semantics, and
// values compare through equals and compare.

const uuidModuleID = "builtin:UUID"

var (
	uuidErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	uuidErrorClass = &types.ClassSymbol{ModuleID: uuidModuleID, Name: "UUIDError", Parent: uuidErrorParent}
	uuidValueClass = &types.ClassSymbol{ModuleID: uuidModuleID, Name: "UUIDValue"}
)

// UUIDErrorIdentity and UUIDValueIdentity expose the canonical identities to
// lowering without coupling the public module interface to a backend.
func UUIDErrorIdentity() *types.ClassSymbol { return uuidErrorClass }
func UUIDValueIdentity() *types.ClassSymbol { return uuidValueClass }

const (
	UUIDValueString  TypeOperation = "UUIDValue.string"
	UUIDValueVersion TypeOperation = "UUIDValue.version"
	UUIDValueIsZero  TypeOperation = "UUIDValue.isZero"
	UUIDValueEquals  TypeOperation = "UUIDValue.equals"
	UUIDValueCompare TypeOperation = "UUIDValue.compare"
)

// UUIDValueOperations names the members a UUIDValue publishes through built-in
// type operations, so has/has not reports the real surface and the IR Class
// agrees with the frontend.
var UUIDValueOperations = []string{"string", "version", "isZero", "equals", "compare"}

func uuidModuleInterface() *ModuleInterface {
	module := standardInterface(uuidModuleID, "UUID")
	codesModuleClasses(module, uuidModuleID, uuidErrorClass, uuidValueClass)
	value := types.Class{Symbol: uuidValueClass}
	for _, name := range []string{"v4", "v7", "zero"} {
		addStandardExport(module, standardFunction(uuidModuleID, name, value))
	}
	addStandardExport(module, standardFunction(uuidModuleID, "parse", value,
		types.Parameter{Name: "text", Type: types.String}))
	addStandardExport(module, standardFunction(uuidModuleID, "isValid", types.Bool,
		types.Parameter{Name: "text", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}

func uuidConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity != uuidValueClass {
		return "", false
	}
	return "create a UUIDValue with UUID.v4(), UUID.v7(), UUID.parse(text), or UUID.zero()", true
}

// uuidOperationShapes reuses the positional member shape of QRCode and
// BarcodeCode: every argument is a NonNull value of the declared type.
func uuidOperationShapes() map[TypeOperation]codesOperationShape {
	none := []types.Type{}
	other := []types.Type{types.Class{Symbol: uuidValueClass}}
	return map[TypeOperation]codesOperationShape{
		UUIDValueString:  {none, 0, types.String, "call string with no argument"},
		UUIDValueVersion: {none, 0, types.Int, "call version with no argument"},
		UUIDValueIsZero:  {none, 0, types.Bool, "call isZero with no argument"},
		UUIDValueEquals:  {other, 1, types.Bool, "pass the UUIDValue to compare with"},
		UUIDValueCompare: {other, 1, types.Int, "pass the UUIDValue to compare with"},
	}
}

var uuidOperationNames = map[string]TypeOperation{
	"string": UUIDValueString, "version": UUIDValueVersion, "isZero": UUIDValueIsZero,
	"equals": UUIDValueEquals, "compare": UUIDValueCompare,
}

// uuidOperationFor names the built-in member a UUIDValue publishes. Only the
// compiler-supplied identity matches, so a user Class named UUIDValue never
// collides with it.
func uuidOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol != uuidValueClass {
		return "", false
	}
	operation, known := uuidOperationNames[name]
	return operation, known
}
