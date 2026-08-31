package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const envModuleID = "builtin:Env"

var envErrorClass = &types.ClassSymbol{
	ModuleID: envModuleID, Name: "EnvError",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}},
}

// EnvErrorIdentity exposes the standard module's catchable error identity to
// lowering without coupling the public module interface to a backend.
func EnvErrorIdentity() *types.ClassSymbol { return envErrorClass }

func envModuleInterface() *ModuleInterface {
	module := standardInterface(envModuleID, "Env")
	errorSymbol := &Symbol{
		Name: "EnvError", Kind: ClassSymbol, Class: envErrorClass,
		Type: types.Class{Symbol: envErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: envModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[envModuleID+"\x00EnvError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	name := func(label string) types.Parameter { return types.Parameter{Name: label, Type: types.String} }
	override := types.Parameter{Name: "override", Type: types.Bool, HasDefault: true}
	record := types.Pair{Key: types.String, Value: types.String}

	addStandardExport(module, standardNullableFunction(envModuleID, "get", types.String, name("name")))
	addStandardExport(module, standardFunction(envModuleID, "getOr", types.String, name("name"), name("fallback")))
	// exists, not has: `has` is a reserved keyword (§2.1, the `x has y`
	// protocol operator) and cannot appear as a member name after `.`;
	// exists matches the existing File.exists naming precedent.
	addStandardExport(module, standardFunction(envModuleID, "exists", types.Bool, name("name")))
	addStandardExport(module, standardFunction(envModuleID, "set", types.Nothing, name("name"), name("value")))
	addStandardExport(module, standardFunction(envModuleID, "unset", types.Nothing, name("name")))
	addStandardExport(module, standardFunction(envModuleID, "read", record, name("path")))
	addStandardExport(module, standardFunction(envModuleID, "load", types.Nothing, name("path"), override))

	sort.Strings(module.ExportNames)
	return module
}

// standardNullableFunction is standardFunction's one-off variant for a
// standard-module function whose *result* (not the function binding itself)
// is statically MaybeNull - Env.get is the only such function across the
// v0.1.17 modules, so this stays local rather than becoming a third shared
// helper alongside standardFunction/standardInterface.
func standardNullableFunction(moduleID, name string, result types.Type, parameters ...types.Parameter) *Symbol {
	signature := &types.Signature{Parameters: parameters, Return: result}
	return &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{Signature: signature},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull, OriginModuleID: moduleID,
		Callable: &Callable{Signature: signature, ParameterNull: nonNullParameters(len(parameters)), ReturnNull: MaybeNull},
	}
}
