package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const latexModulePrefix = "builtin:Latex::"

func (generator *generator) latexCall(value *ir.CallExpr) string {
	generator.usesLatex = true
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), latexModulePrefix)
	argument := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
			return `""`
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "pdf":
		return "AhdLatexPDF(" + argument(0) + ", " + argument(1) + ")"
	case "pdfFile":
		return "AhdLatexPDFFile(" + argument(0) + ", " + argument(1) + ")"
	case "escape":
		return "AhdLatexEscape(" + argument(0) + ")"
	case "section":
		return "AhdLatexSection(" + argument(0) + ")"
	case "subsection":
		return "AhdLatexSubsection(" + argument(0) + ")"
	case "equation":
		return "AhdLatexEquation(" + argument(0) + ")"
	case "document":
		return "AhdLatexDocument(" + argument(0) + ", " + argument(1) + ", " + argument(2) + ")"
	case "table":
		if len(value.Arguments) != 2 || value.Arguments[0].Value == nil || value.Arguments[1].Value == nil {
			generator.fail(CodeGenerationFailure, "Latex.table has a missing argument", meta.Span, "the IR call is malformed")
			return `""`
		}
		return "AhdLatexTable(" + generator.expr(value.Arguments[0].Value) + ", " + generator.expr(value.Arguments[1].Value) + ")"
	default:
		return generator.unsupported("Latex function "+name, meta.Span)
	}
}
