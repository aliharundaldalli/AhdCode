package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const cronModuleID = "builtin:Cron"

var (
	cronErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	cronErrorClass     = &types.ClassSymbol{ModuleID: cronModuleID, Name: "CronError", Parent: cronErrorParent}
	cronSchedulerClass = &types.ClassSymbol{ModuleID: cronModuleID, Name: "Scheduler"}
)

// CronErrorIdentity and CronSchedulerIdentity expose the canonical identities
// to lowering without coupling the public module interface to a backend.
func CronErrorIdentity() *types.ClassSymbol     { return cronErrorClass }
func CronSchedulerIdentity() *types.ClassSymbol { return cronSchedulerClass }

// CronSchedulerOperations names the members Scheduler publishes through
// built-in type operations, so has/has not reports the real surface and the
// IR Class agrees with the frontend.
var CronSchedulerOperations = []string{"add", "run", "stop"}

func cronSchedulerType() types.Type { return types.Class{Symbol: cronSchedulerClass} }

// cronTaskType is the one task shape a Scheduler accepts: a Function that takes
// nothing and returns Nothing. It is checked statically like an HTTP handler.
func cronTaskType() types.Type {
	return types.Function{Signature: &types.Signature{Return: types.Nothing}}
}

// cronModuleInterface declares the Cron standard module.
//
// Cron is bounded in-process scheduling: a classic five-field schedule, a
// Scheduler that runs AhdCode Functions in the calling program, and Cron.next,
// the pure calendar computation behind it. It is not a job queue, a daemon, or
// an orchestration framework.
func cronModuleInterface() *ModuleInterface {
	module := standardInterface(cronModuleID, "Cron")
	for _, entry := range []struct {
		name     string
		identity *types.ClassSymbol
	}{{"CronError", cronErrorClass}, {"Scheduler", cronSchedulerClass}} {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: cronModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "CronError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[cronModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	dateTime := types.Class{Symbol: timeDateTimeClass}
	addStandardExport(module, standardFunction(cronModuleID, "scheduler", cronSchedulerType()))
	addStandardExport(module, standardFunction(cronModuleID, "next", dateTime,
		types.Parameter{Name: "expression", Type: types.String},
		types.Parameter{Name: "after", Type: dateTime}))
	sort.Strings(module.ExportNames)
	return module
}

func cronConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != cronModuleID || identity.Name != "Scheduler" {
		return "", false
	}
	return "create a Scheduler with Cron.scheduler()", true
}

type cronOperationShape struct {
	parameters []types.Type
	hint       string
}

func cronOperationShapes() map[TypeOperation]cronOperationShape {
	return map[TypeOperation]cronOperationShape{
		CronSchedulerAdd:  {[]types.Type{types.String, cronTaskType()}, "pass a schedule String and a () -> Nothing Function"},
		CronSchedulerRun:  {[]types.Type{}, "call run with no argument"},
		CronSchedulerStop: {[]types.Type{}, "call stop with no argument"},
	}
}

var cronOperationNames = map[string]TypeOperation{
	"add": CronSchedulerAdd, "run": CronSchedulerRun, "stop": CronSchedulerStop,
}

func cronOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol != cronSchedulerClass {
		return "", false
	}
	operation, known := cronOperationNames[name]
	return operation, known
}

// analyzeCronOperation checks one Scheduler member call. Every Scheduler
// operation returns Nothing. The task argument must be a Function whose
// signature is exactly compatible with () -> Nothing; any other shape is the
// existing SEM004 type mismatch, never a runtime failure.
func (a *analyzer) analyzeCronOperation(call *ast.CallExpr, operation TypeOperation, shape cronOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, argument := range call.Arguments {
		expected := shape.parameters[index]
		reported := a.bag.Len()
		info := a.analyzeExpressionExpected(argument.Value, current, flow, expected)
		// A Function value checked against its expected shape reports its own
		// mismatch; a second diagnostic here would only repeat it.
		if info.invalid() || a.bag.Len() > reported {
			continue
		}
		if info.nullState != NonNull {
			a.nullableError(string(operation), argument.Value, info.nullState)
			continue
		}
		if expectedFunction, isFunction := expected.(types.Function); isFunction {
			got, ok := info.typeValue.(types.Function)
			if !ok || got.Signature == nil {
				a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" task")
				continue
			}
			if _, compatible := functionValueScore(got.Signature, expectedFunction.Signature); !compatible {
				a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" task")
			}
			continue
		}
		if !types.Assignable(expected, info.typeValue) {
			a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" argument")
		}
	}
	parameters := make([]types.Parameter, len(shape.parameters))
	for index, expected := range shape.parameters {
		parameters[index] = types.Parameter{Type: expected}
	}
	a.result.SelectedCallables[call] = &Callable{
		Signature:  &types.Signature{Parameters: parameters, Return: types.Nothing},
		ReturnNull: NonNull,
	}
	return result
}
