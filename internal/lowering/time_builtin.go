package lowering

import (
	"strings"

	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// TimeModuleID is the synthetic module that carries the Time standard
// library's Class declarations into the IR.
const TimeModuleID = "builtin:Time"

// timeClassID mirrors the frontend identity rule, so a Class named in AhdCode
// source and the declaration emitted here are the same Class.
func timeClassID(name string) ir.ClassID {
	return ir.ClassID(TimeModuleID + "::class::" + name)
}

func timeFieldID(class, field string) ir.FieldID {
	return ir.FieldID(string(timeClassID(class)) + "::field::" + field)
}

// timeDateTimeOperations and the other operation lists name the members each
// Time Class publishes through built-in type operations. They exist so member
// existence reports what the value really offers.
var (
	timeDateTimeOperations = []string{"before", "after", "sameMoment", "timestamp", "toUTC", "toLocal", "toOffset", "toString"}
	timeCalendarOperations = []string{"isLeapYear", "daysInMonth", "weekday"}
)

// timeModule emits DateTime, Duration, and Calendar as ordinary IR classes.
// Their attributes are real fields, so reading them needs no special path and
// the frontend's Constant rule already makes them read-only.
func timeModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	dateTime := timeClassDeclaration("DateTime", timeIntFields("DateTime", semantic.TimeDateTimeFields)...)
	dateTime.Operations = timeDateTimeOperations
	duration := timeClassDeclaration("Duration",
		append(timeIntFields("Duration", []string{"milliseconds"}),
			ir.Field{ID: timeFieldID("Duration", "seconds"), Name: "seconds",
				Type: ir.Type{Kind: ir.RealType}, NullState: ir.NonNull})...)
	calendar := timeClassDeclaration("Calendar")
	calendar.Operations = timeCalendarOperations

	for _, class := range []*ir.Class{dateTime, duration, calendar} {
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, timeConstructor(class))
	}
	return module
}

func timeClassDeclaration(name string, fields ...ir.Field) *ir.Class {
	identity := timeClassID(name)
	return &ir.Class{
		ID: identity, Symbol: ir.SymbolID(string(identity) + "::symbol"),
		Name: name, Fields: fields,
		Constructor: timeConstructorID(name, fields),
	}
}

func timeIntFields(class string, names []string) []ir.Field {
	fields := make([]ir.Field, 0, len(names))
	for _, name := range names {
		fields = append(fields, ir.Field{
			ID: timeFieldID(class, name), Name: name,
			Type: ir.Type{Kind: ir.IntType}, NullState: ir.NonNull,
		})
	}
	return fields
}

func timeConstructorID(name string, fields []ir.Field) ir.CallableID {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, field.Name+":"+field.Type.String())
	}
	return ir.CallableID(string(timeClassID(name)) + "::constructor::(" + strings.Join(parts, ",") + ")->Nothing")
}

// timeConstructor builds the constructor the backend uses to materialize one
// value. AhdCode source cannot reach it: the frontend publishes these Classes
// without a constructor, so values come only from the Time functions, which
// validate their arguments first.
func timeConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	function := &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: ir.SymbolID(string(id) + "::receiver"),
		Signature: ir.Signature{Return: ir.Type{Kind: ir.NothingType}}, ReturnNull: ir.NonNull,
	}
	var statements []ir.Statement
	for _, field := range class.Fields {
		parameter := ir.Parameter{
			ID: ir.SymbolID(string(id) + "::parameter::" + field.Name), Name: field.Name,
			Type: field.Type, NullState: ir.NonNull,
		}
		function.Signature.Parameters = append(function.Signature.Parameters,
			ir.ParameterType{Name: field.Name, Type: field.Type})
		function.Parameters = append(function.Parameters, parameter)
		statements = append(statements, &ir.AssignStmt{
			Target: ir.Target{
				Kind: ir.FieldTarget, Type: field.Type, Field: field.ID,
				Receiver: &ir.LoadExpr{
					ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull},
					Symbol:   function.Receiver,
				},
			},
			Value: &ir.LoadExpr{
				ExprBase: ir.ExprBase{Type: field.Type, NullState: ir.NonNull}, Symbol: parameter.ID,
			},
		})
	}
	function.Body = ir.Block{Statements: statements}
	return function
}
