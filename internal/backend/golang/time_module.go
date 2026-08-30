package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const timeModulePrefix = "builtin:Time::"

var (
	timeDateTimeClass = ir.ClassID("builtin:Time::class::DateTime")
	timeDurationClass = ir.ClassID("builtin:Time::class::Duration")
)

// timeCall lowers the explicitly imported Time standard module. The frontend
// has already checked every signature, so this layer only maps stable builtin
// identities onto runtime helpers and the generated constructors.
func (generator *generator) timeCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), timeModulePrefix)
	argument := func(index int) string {
		// An omitted default is an explicit zero: every defaulted Time
		// parameter is a zero-based clock component.
		if index >= len(value.Arguments) || value.Arguments[index].UsesDefault ||
			value.Arguments[index].Value == nil {
			return "int64(0)"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	switch name {
	case "now":
		return generator.dateTimeFrom("AhdTimeNow()", meta)
	case "utc":
		return generator.dateTimeFrom("AhdTimeUTC()", meta)
	case "timestamp":
		return "AhdTimeTimestamp()"
	case "fromTimestamp":
		return generator.dateTimeFrom("AhdTimeFromTimestamp("+argument(0)+")", meta)
	case "monotonic":
		return "AhdTimeMonotonic()"
	case "sleep":
		return "AhdTimeSleep(" + argument(0) + ")"
	case "dateTime":
		fields := make([]string, 7)
		for index := range fields {
			fields[index] = argument(index)
		}
		return generator.dateTimeFrom("AhdTimeCivil("+strings.Join(fields, ", ")+")", meta)
	case "dateTimeUTC":
		fields := make([]string, 7)
		for index := range fields {
			fields[index] = argument(index)
		}
		return generator.dateTimeFrom("AhdTimeCivilUTC("+strings.Join(fields, ", ")+")", meta)
	case "dateTimeOffset":
		fields := make([]string, 8)
		for index := range fields {
			fields[index] = argument(index)
		}
		return generator.dateTimeFrom("AhdTimeCivilOffset("+strings.Join(fields, ", ")+")", meta)
	case "duration":
		return generator.durationFrom(argument(0), meta)
	case "between":
		if len(value.Arguments) != 2 || value.Arguments[0].Value == nil || value.Arguments[1].Value == nil {
			generator.fail(CodeGenerationFailure, "Time.between has a missing argument", meta.Span, "the IR call is malformed")
			return "nil"
		}
		first := generator.instantOf(value.Arguments[0].Value)
		second := generator.instantOf(value.Arguments[1].Value)
		return generator.durationFrom("AhdTimeDifference("+first+", "+second+")", meta)
	default:
		return generator.unsupported("Time function "+name, meta.Span)
	}
}

// dateTimeFrom builds a DateTime instance from one civil-time reading. The
// reading is taken once and spread across the constructor by a generated
// helper, so a clock call is never repeated per field.
func (generator *generator) dateTimeFrom(civil string, meta ir.ExprBase) string {
	helper, ok := generator.timeHelper(timeDateTimeClass)
	if !ok {
		return generator.unsupported("a DateTime value without its Class declaration", meta.Span)
	}
	return helper + "(" + civil + ")"
}

func (generator *generator) durationFrom(milliseconds string, meta ir.ExprBase) string {
	helper, ok := generator.timeHelper(timeDurationClass)
	if !ok {
		return generator.unsupported("a Duration value without its Class declaration", meta.Span)
	}
	return helper + "(" + milliseconds + ")"
}

// instantOf rebuilds the instant a DateTime expression denotes from its
// published attributes, so comparison and difference need no hidden state.
func (generator *generator) instantOf(expression ir.Expr) string {
	return "AhdTimeInstantCivil(" + generator.civilOf(expression) + ")"
}

// civilOf evaluates one DateTime expression exactly once and snapshots its
// civil fields into the runtime interchange shape. offsetSeconds is runtime
// representation rather than a published attribute, so it travels with the
// reading and keeps a historical sub-minute offset exact.
func (generator *generator) civilOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	names := []string{"year", "month", "day", "hour", "minute", "second", "millisecond", "offsetMinutes", "offsetSeconds"}
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, "value."+generator.fieldName(ir.FieldID(string(timeDateTimeClass)+"::field::"+name))+"_get()")
	}
	return "func(value " + generator.interfaceName(timeDateTimeClass) + ") AhdCivilTime { " +
		"return AhdCivilTime{Year: " + parts[0] + ", Month: " + parts[1] + ", Day: " + parts[2] +
		", Hour: " + parts[3] + ", Minute: " + parts[4] + ", Second: " + parts[5] +
		", Millisecond: " + parts[6] + ", OffsetMinutes: " + parts[7] +
		", OffsetSeconds: " + parts[8] + "} }(" + rendered + ")"
}

// timeHelper registers the generated constructor wrapper of one Time Class.
func (generator *generator) timeHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("th_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

// emitTimeHelpers writes one wrapper per Time Class actually used, turning a
// runtime reading into a constructed AhdCode value.
func (generator *generator) emitTimeHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{timeDateTimeClass, timeDurationClass} {
		name, known := generator.timeHelpers[class]
		if !known {
			continue
		}
		layout := generator.layouts[class]
		constructor := generator.functions[layout.class.Constructor]
		if constructor == nil {
			continue
		}
		result := generator.interfaceName(class)
		if class == timeDateTimeClass {
			writer.line("// DateTime value built from one civil-time reading.")
			writer.open("func " + name + "(civil AhdCivilTime) " + result + " {")
			writer.line("return " + generator.callableName(constructor) +
				"(civil.Year, civil.Month, civil.Day, civil.Hour, civil.Minute, civil.Second, civil.Millisecond, civil.Weekday, civil.OffsetMinutes, civil.OffsetSeconds)")
			writer.close("}")
			writer.blank()
			continue
		}
		writer.line("// Duration value built from a signed millisecond count.")
		writer.open("func " + name + "(milliseconds int64) " + result + " {")
		writer.line("return " + generator.callableName(constructor) + "(milliseconds, AhdDurationSeconds(milliseconds))")
		writer.close("}")
		writer.blank()
	}
}

// timeOperation lowers the built-in members of the Time Classes. DateTime and
// Calendar reach these through the ordinary type-operation path, so AhdCode
// gains no static-method or operator-overloading semantics.
func (generator *generator) timeOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	other := func() string {
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "AhdTimeInstant(1, 1, 1, 0, 0, 0, 0, 0, 0)"
		}
		return generator.instantOf(value.Arguments[0].Value)
	}
	integer := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "int64(0)"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	switch name {
	case "DateTime.before":
		return "(AhdTimeCompare(" + generator.instantOf(value.Callee) + ", " + other() + ") < 0)"
	case "DateTime.after":
		return "(AhdTimeCompare(" + generator.instantOf(value.Callee) + ", " + other() + ") > 0)"
	case "DateTime.sameMoment":
		return "(AhdTimeCompare(" + generator.instantOf(value.Callee) + ", " + other() + ") == 0)"
	case "DateTime.timestamp":
		return "AhdTimeInstantTimestamp(" + generator.instantOf(value.Callee) + ")"
	case "DateTime.toUTC":
		return generator.dateTimeFrom("AhdTimeToUTC("+generator.instantOf(value.Callee)+")", meta)
	case "DateTime.toLocal":
		return generator.dateTimeFrom("AhdTimeToLocal("+generator.instantOf(value.Callee)+")", meta)
	case "DateTime.toOffset":
		return generator.dateTimeFrom("AhdTimeToOffset("+generator.instantOf(value.Callee)+", "+integer(0)+")", meta)
	case "DateTime.toString":
		return "AhdTimeCivilText(" + generator.civilOf(value.Callee) + ")"
	case "Calendar.isLeapYear":
		return "AhdCalendarIsLeapYear(" + integer(0) + ")"
	case "Calendar.daysInMonth":
		return "AhdCalendarDaysInMonth(" + integer(0) + ", " + integer(1) + ")"
	case "Calendar.weekday":
		return "AhdCalendarWeekday(" + integer(0) + ", " + integer(1) + ", " + integer(2) + ")"
	default:
		return generator.unsupported("Time operation "+name, meta.Span)
	}
}
