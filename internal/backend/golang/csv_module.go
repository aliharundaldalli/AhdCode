package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const csvModulePrefix = "builtin:CSV::"

var csvErrorClass = ir.ClassID("builtin:CSV::class::CSVError")

func (generator *generator) csvCall(value *ir.CallExpr) string {
	name := strings.TrimPrefix(string(value.Callable), csvModulePrefix)
	argument := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.expr(value.Arguments[index].Value)
	}
	csvError := generator.descriptorName(csvErrorClass)
	fileError := "AhdClassFileError"
	delimiterIndex := map[string]int{"parse": 1, "stringify": 1, "read": 1, "write": 2,
		"parseRecords": 1, "readRecords": 1, "stringifyRecords": 1, "writeRecords": 2}[name]
	delimiter := argument(delimiterIndex, `","`)
	switch name {
	case "parse":
		return "AhdCSVParse(" + csvError + ", " + argument(0, `""`) + ", " + delimiter + ")"
	case "stringify":
		return "AhdCSVStringify(" + csvError + ", " + argument(0, "nil") + ", " + delimiter + ")"
	case "read":
		return "AhdCSVRead(" + csvError + ", " + fileError + ", " + argument(0, `""`) + ", " + delimiter + ")"
	case "write":
		return "AhdCSVWrite(" + csvError + ", " + fileError + ", " + argument(0, `""`) + ", " + argument(1, "nil") + ", " + delimiter + ")"
	case "parseRecords":
		return "AhdCSVParseRecords(" + csvError + ", " + argument(0, `""`) + ", " + delimiter + ")"
	case "readRecords":
		return "AhdCSVReadRecords(" + csvError + ", " + fileError + ", " + argument(0, `""`) + ", " + delimiter + ")"
	case "stringifyRecords":
		return "AhdCSVStringifyRecords(" + csvError + ", " + argument(0, "nil") + ", " + delimiter + ")"
	case "writeRecords":
		return "AhdCSVWriteRecords(" + csvError + ", " + fileError + ", " + argument(0, `""`) + ", " + argument(1, "nil") + ", " + delimiter + ")"
	default:
		return generator.unsupported("CSV function "+name, value.ExprMeta().Span)
	}
}
