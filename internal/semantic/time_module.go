package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const timeModuleID = "builtin:Time"

// TimeDateTimeFields are the read-only Int attributes every DateTime exposes,
// in constructor order. The runtime derives every one of them, including
// weekday, when a DateTime is created.
var TimeDateTimeFields = []string{
	"year", "month", "day", "hour", "minute", "second", "millisecond", "weekday",
}

// Canonical identities of the Classes the Time standard module publishes.
var (
	timeDateTimeClass = &types.ClassSymbol{ModuleID: timeModuleID, Name: "DateTime"}
	timeDurationClass = &types.ClassSymbol{ModuleID: timeModuleID, Name: "Duration"}
	timeCalendarClass = &types.ClassSymbol{ModuleID: timeModuleID, Name: "Calendar"}
)

// TimeClassNames maps each Time Class identity to its canonical name, so
// lowering and the backend can publish the same declarations the frontend
// promises without duplicating the identity rules.
func TimeClassNames() map[*types.ClassSymbol]string {
	return map[*types.ClassSymbol]string{
		timeDateTimeClass: "DateTime",
		timeDurationClass: "Duration",
		timeCalendarClass: "Calendar",
	}
}

// TimeDateTimeIdentity, TimeDurationIdentity, and TimeCalendarIdentity expose
// the canonical Class identities to the lowering layer.
func TimeDateTimeIdentity() *types.ClassSymbol { return timeDateTimeClass }
func TimeDurationIdentity() *types.ClassSymbol { return timeDurationClass }
func TimeCalendarIdentity() *types.ClassSymbol { return timeCalendarClass }

// timeModuleInterface builds the Time standard module. It follows the same
// ModuleInterface model as Math, and additionally publishes the DateTime,
// Duration, and Calendar Class identities.
//
// DateTime and Duration attributes are Constant, so the existing Constant rule
// already rejects assignment through an instance and no new immutability
// concept is needed. Calendar is published as a Class reference whose members
// are built-in type operations, so AhdCode gains no static-method semantics.
func timeModuleInterface() *ModuleInterface {
	module := &ModuleInterface{
		ModuleID: timeModuleID,
		Name:     "Time",
		Exports:  make(map[string]*Symbol),
		Symbols:  make(map[string]*Symbol),
		Classes:  make(map[string]*Symbol),
	}
	add := func(symbol *Symbol) {
		module.Symbols[symbol.Name] = symbol
		module.Exports[symbol.Name] = symbol
		module.ExportNames = append(module.ExportNames, symbol.Name)
	}

	dateTime := timeClass(timeDateTimeClass, timeIntAttributes(timeDateTimeClass, TimeDateTimeFields...))
	duration := timeClass(timeDurationClass, append(
		timeIntAttributes(timeDurationClass, "milliseconds"),
		timeAttribute(timeDurationClass, "seconds", types.Real),
	))
	calendar := timeClass(timeCalendarClass, nil)
	for _, class := range []*Symbol{dateTime, duration, calendar} {
		module.Classes[timeModuleID+"\x00"+class.Name] = class
		add(class)
	}

	instant := types.Class{Symbol: timeDateTimeClass}
	elapsed := types.Class{Symbol: timeDurationClass}
	add(timeFunction("now", timeSignature(instant)))
	add(timeFunction("monotonic", timeSignature(types.Real)))
	add(timeFunction("sleep", timeSignature(types.Nothing, timeParameter("milliseconds", types.Int))))
	add(timeFunction("duration", timeSignature(elapsed, timeParameter("milliseconds", types.Int))))
	add(timeFunction("between", timeSignature(elapsed,
		timeParameter("first", instant), timeParameter("second", instant))))
	add(timeFunction("dateTime", timeDateTimeSignature(instant)))

	sort.Strings(module.ExportNames)
	return module
}

// timeDateTimeSignature is the civil-construction contract. hour, minute,
// second, and millisecond use the ordinary default-argument mechanics the
// compiler already implements.
func timeDateTimeSignature(result types.Type) *types.Signature {
	parameters := []types.Parameter{
		timeParameter("year", types.Int),
		timeParameter("month", types.Int),
		timeParameter("day", types.Int),
	}
	for _, name := range []string{"hour", "minute", "second", "millisecond"} {
		parameters = append(parameters, types.Parameter{Name: name, Type: types.Int, HasDefault: true})
	}
	return &types.Signature{Parameters: parameters, Return: result}
}

// timeClass publishes one compiler-supplied Class. It deliberately has no
// constructor: DateTime and Duration values are produced by Time functions, so
// direct construction is rejected rather than bypassing validation.
func timeClass(identity *types.ClassSymbol, members []*Symbol) *Symbol {
	symbol := &Symbol{
		Name: identity.Name, Kind: ClassSymbol, Class: identity,
		Type: types.Class{Symbol: identity, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: timeModuleID,
		Members: make(map[string]*Symbol, len(members)),
	}
	for _, member := range members {
		symbol.Members[member.Name] = member
	}
	return symbol
}

func timeIntAttributes(owner *types.ClassSymbol, names ...string) []*Symbol {
	members := make([]*Symbol, 0, len(names))
	for _, name := range names {
		members = append(members, timeAttribute(owner, name, types.Int))
	}
	return members
}

// timeAttribute is one read-only instance attribute. Constant is what makes it
// read-only: assigning it is the existing Constant diagnostic.
func timeAttribute(owner *types.ClassSymbol, name string, value types.Type) *Symbol {
	return &Symbol{
		Name: name, Kind: MemberSymbol, Type: value, Constant: true,
		Builtin: true, InitialNull: NonNull, OwnerClass: owner,
		OriginModuleID: timeModuleID,
	}
}

func timeFunction(name string, signature *types.Signature) *Symbol {
	return &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{Signature: signature},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull,
		OriginModuleID: timeModuleID,
		Callable: &Callable{
			Signature: signature, ParameterNull: nonNullParameters(len(signature.Parameters)),
			ReturnNull: NonNull,
		},
	}
}

func timeSignature(result types.Type, parameters ...types.Parameter) *types.Signature {
	return &types.Signature{Parameters: parameters, Return: result}
}

func timeParameter(name string, value types.Type) types.Parameter {
	return types.Parameter{Name: name, Type: value}
}
