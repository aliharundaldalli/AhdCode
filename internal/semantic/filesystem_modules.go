package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const (
	fileModuleID = "builtin:File"
	pathModuleID = "builtin:Path"
)

var fileErrorClass = &types.ClassSymbol{
	ModuleID: fileModuleID,
	Name:     "FileError",
	Parent: &types.ClassSymbol{
		ModuleID: "builtin:core", Name: "IOError",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
			Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}},
	},
}

// FileErrorIdentity exposes the standard module's catchable error identity to
// lowering without coupling the public module interface to a backend.
func FileErrorIdentity() *types.ClassSymbol { return fileErrorClass }

func fileModuleInterface() *ModuleInterface {
	module := standardInterface(fileModuleID, "File")
	errorSymbol := &Symbol{
		Name: "FileError", Kind: ClassSymbol, Class: fileErrorClass,
		Type: types.Class{Symbol: fileErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: fileModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[fileModuleID+"\x00FileError"] = errorSymbol
	addStandardExport(module, errorSymbol)
	stringParameter := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	addStandardExport(module, standardFunction(fileModuleID, "exists", types.Bool, stringParameter("path")))
	addStandardExport(module, standardFunction(fileModuleID, "readText", types.String, stringParameter("path")))
	addStandardExport(module, standardFunction(fileModuleID, "writeText", types.Nothing, stringParameter("path"), stringParameter("content")))
	addStandardExport(module, standardFunction(fileModuleID, "append", types.Nothing, stringParameter("path"), stringParameter("content")))
	addStandardExport(module, standardFunction(fileModuleID, "delete", types.Nothing, stringParameter("path")))
	addStandardExport(module, standardFunction(fileModuleID, "createDir", types.Nothing, stringParameter("path")))
	addStandardExport(module, standardFunction(fileModuleID, "list", types.List{Element: types.String}, stringParameter("path")))
	sort.Strings(module.ExportNames)
	return module
}

func pathModuleInterface() *ModuleInterface {
	module := standardInterface(pathModuleID, "Path")
	stringParameter := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	addStandardExport(module, standardFunction(pathModuleID, "join", types.String,
		types.Parameter{Name: "parts", Type: types.List{Element: types.String}}))
	for _, name := range []string{"ext", "base", "dir"} {
		addStandardExport(module, standardFunction(pathModuleID, name, types.String, stringParameter("path")))
	}
	sort.Strings(module.ExportNames)
	return module
}

func standardInterface(id, name string) *ModuleInterface {
	return &ModuleInterface{ModuleID: id, Name: name, Exports: make(map[string]*Symbol), Symbols: make(map[string]*Symbol), Classes: make(map[string]*Symbol)}
}

func addStandardExport(module *ModuleInterface, symbol *Symbol) {
	module.Symbols[symbol.Name] = symbol
	module.Exports[symbol.Name] = symbol
	module.ExportNames = append(module.ExportNames, symbol.Name)
}

func standardFunction(moduleID, name string, result types.Type, parameters ...types.Parameter) *Symbol {
	signature := &types.Signature{Parameters: parameters, Return: result}
	return &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{Signature: signature},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull, OriginModuleID: moduleID,
		Callable: &Callable{Signature: signature, ParameterNull: nonNullParameters(len(parameters)), ReturnNull: NonNull},
	}
}
