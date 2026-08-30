package lowering

import "ahdcode/internal/ir"

// StatisticsModuleID is the synthetic module that carries the Statistics
// standard library's Class declarations into the IR.
const StatisticsModuleID = "builtin:Statistics"

const statisticsErrorClassID = ir.ClassID(StatisticsModuleID + "::class::StatisticsError")

// statisticsModule emits StatisticsError. Statistics itself publishes only
// functions, so the module carries no other Class.
func statisticsModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: statisticsErrorClassID, Symbol: ir.SymbolID(string(statisticsErrorClassID) + "::symbol"),
		Name: "StatisticsError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(statisticsErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}
