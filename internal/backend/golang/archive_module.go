package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const archiveModulePrefix = "builtin:Archive::"

// archiveCall lowers the Archive module's three creation functions. Each
// takes a destination path and a Pair<String,String> mapping an in-archive
// member path to a source filesystem path; Pair already crosses the codegen
// boundary as *AhdPair[string, string] (the same representation
// Latex.bibliography's references parameter uses), so no new interchange
// type is needed.
func (generator *generator) archiveCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), archiveModulePrefix)
	text := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return `""`
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	entries := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return "AhdBuildPair([]string{}, []string{})"
		}
		return generator.expr(value.Arguments[index].Value)
	}
	switch name {
	case "zip":
		return "AhdArchiveZip(" + text(0) + ", " + entries(1) + ")"
	case "tar":
		return "AhdArchiveTar(" + text(0) + ", " + entries(1) + ")"
	case "tarGzip":
		return "AhdArchiveTarGzip(" + text(0) + ", " + entries(1) + ")"
	default:
		return generator.unsupported("Archive function "+name, meta.Span)
	}
}
