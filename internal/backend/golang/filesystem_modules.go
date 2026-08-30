package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const (
	fileModulePrefix = "builtin:File::"
	pathModulePrefix = "builtin:Path::"
)

var fileErrorClass = ir.ClassID("builtin:File::class::FileError")

func (generator *generator) fileCall(value *ir.CallExpr) string {
	name := strings.TrimPrefix(string(value.Callable), fileModulePrefix)
	arguments := generator.standardStringArguments(value)
	if arguments == nil {
		return "nil"
	}
	errorClass := generator.descriptorName(fileErrorClass)
	switch name {
	case "exists":
		return "AhdFileExists(" + errorClass + ", " + arguments[0] + ")"
	case "readText":
		return "AhdFileReadText(" + errorClass + ", " + arguments[0] + ")"
	case "writeText":
		return "AhdFileWriteText(" + errorClass + ", " + arguments[0] + ", " + arguments[1] + ")"
	case "append":
		return "AhdFileAppend(" + errorClass + ", " + arguments[0] + ", " + arguments[1] + ")"
	case "delete":
		return "AhdFileDelete(" + errorClass + ", " + arguments[0] + ")"
	case "createDir":
		return "AhdFileCreateDir(" + errorClass + ", " + arguments[0] + ")"
	case "list":
		return "AhdFileList(" + errorClass + ", " + arguments[0] + ")"
	default:
		return generator.unsupported("File function "+name, value.ExprMeta().Span)
	}
}

func (generator *generator) pathCall(value *ir.CallExpr) string {
	name := strings.TrimPrefix(string(value.Callable), pathModulePrefix)
	if name == "join" {
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported("Path.join arguments", value.ExprMeta().Span)
		}
		return "AhdPathJoin(" + generator.expr(value.Arguments[0].Value) + ")"
	}
	arguments := generator.standardStringArguments(value)
	if arguments == nil {
		return "nil"
	}
	switch name {
	case "ext":
		return "AhdPathExt(" + arguments[0] + ")"
	case "base":
		return "AhdPathBase(" + arguments[0] + ")"
	case "dir":
		return "AhdPathDir(" + arguments[0] + ")"
	default:
		return generator.unsupported("Path function "+name, value.ExprMeta().Span)
	}
}

func (generator *generator) standardStringArguments(value *ir.CallExpr) []string {
	arguments := make([]string, len(value.Arguments))
	for index, argument := range value.Arguments {
		if argument.Value == nil {
			generator.fail(CodeGenerationFailure, "standard module call has a missing argument", value.ExprMeta().Span, "the IR call is malformed")
			return nil
		}
		arguments[index] = generator.value(argument.Value, ir.Type{Kind: ir.StringType}, false)
	}
	return arguments
}
