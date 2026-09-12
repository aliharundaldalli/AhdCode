package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// CronModuleID is the compiler-supplied Cron standard module identity.
const CronModuleID = "builtin:Cron"

const (
	cronSchedulerClassID = ir.ClassID(CronModuleID + "::class::Scheduler")
	cronErrorClassID     = ir.ClassID(CronModuleID + "::class::CronError")
)

// CronSchedulerHandleFieldID is the hidden String handle naming one runtime
// scheduler, the same representation SMTPClient uses.
var CronSchedulerHandleFieldID = ir.FieldID(string(cronSchedulerClassID) + "::field::handle")

func cronModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	field := ir.Field{ID: CronSchedulerHandleFieldID, Name: "handle", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
	scheduler := &ir.Class{
		ID: cronSchedulerClassID, Symbol: ir.SymbolID(string(cronSchedulerClassID) + "::symbol"), Name: "Scheduler",
		Operations: semantic.CronSchedulerOperations, Fields: []ir.Field{field},
		Constructor: ir.CallableID(string(cronSchedulerClassID) + "::constructor::(handle:String)->Nothing"),
	}
	module.Classes = append(module.Classes, scheduler)
	module.Functions = append(module.Functions, smtpValueConstructor(scheduler))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: cronErrorClassID, Symbol: ir.SymbolID(string(cronErrorClassID) + "::symbol"),
		Name: "CronError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(cronErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}
