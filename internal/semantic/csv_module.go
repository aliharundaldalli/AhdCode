package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const csvModuleID = "builtin:CSV"

var csvErrorClass = &types.ClassSymbol{
	ModuleID: csvModuleID, Name: "CSVError",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}},
}

func CSVErrorIdentity() *types.ClassSymbol { return csvErrorClass }

func csvModuleInterface() *ModuleInterface {
	module := standardInterface(csvModuleID, "CSV")
	errorSymbol := &Symbol{
		Name: "CSVError", Kind: ClassSymbol, Class: csvErrorClass,
		Type: types.Class{Symbol: csvErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: csvModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[csvModuleID+"\x00CSVError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	row := types.List{Element: types.String}
	rows := types.List{Element: row}
	record := types.Pair{Key: types.String, Value: types.String}
	records := types.List{Element: record}
	text := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	delimiter := types.Parameter{Name: "delimiter", Type: types.String, HasDefault: true}

	addStandardExport(module, standardFunction(csvModuleID, "parse", rows, text("text"), delimiter))
	addStandardExport(module, standardFunction(csvModuleID, "stringify", types.String,
		types.Parameter{Name: "rows", Type: rows}, delimiter))
	addStandardExport(module, standardFunction(csvModuleID, "read", rows, text("path"), delimiter))
	addStandardExport(module, standardFunction(csvModuleID, "write", types.Nothing,
		text("path"), types.Parameter{Name: "rows", Type: rows}, delimiter))
	addStandardExport(module, standardFunction(csvModuleID, "parseRecords", records, text("text"), delimiter))
	addStandardExport(module, standardFunction(csvModuleID, "readRecords", records, text("path"), delimiter))
	addStandardExport(module, standardFunction(csvModuleID, "stringifyRecords", types.String,
		types.Parameter{Name: "records", Type: records}, delimiter))
	addStandardExport(module, standardFunction(csvModuleID, "writeRecords", types.Nothing,
		text("path"), types.Parameter{Name: "records", Type: records}, delimiter))
	sort.Strings(module.ExportNames)
	return module
}
