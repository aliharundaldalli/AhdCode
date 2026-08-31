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
	realArgument := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.RealType}, false)
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
	case "chapter":
		return "AhdLatexChapter(" + argument(0) + ")"
	case "frame":
		return "AhdLatexFrame(" + argument(0) + ", " + argument(1) + ")"
	case "equation":
		return "AhdLatexEquation(" + argument(0) + ", " + argument(1) + ")"
	case "theorem":
		return "AhdLatexTheorem(" + argument(0) + ", " + argument(1) + ", " + argument(2) + ")"
	case "ref":
		return "AhdLatexRef(" + argument(0) + ")"
	case "cite":
		return "AhdLatexCite(" + argument(0) + ")"
	case "center":
		return "AhdLatexCenter(" + argument(0) + ")"
	case "pageBreak":
		return "AhdLatexPageBreak()"
	case "contents":
		return "AhdLatexContents()"
	case "minipage":
		return "AhdLatexMinipage(" + argument(0) + ", " + realArgument(1, "0") + ", " + func() string {
			if len(value.Arguments) <= 2 || value.Arguments[2].UsesDefault {
				return `"left"`
			}
			return argument(2)
		}() + ")"
	case "image":
		return "AhdLatexImage(" + argument(0) + ", " + generator.latexSizePair(value, 1) + ")"
	case "figure":
		return "AhdLatexFigure(" + argument(0) + ", " + argument(1) + ", " + argument(2) + ", " + generator.latexSizePair(value, 3) + ")"
	case "bibliography":
		return "AhdLatexBibliography(" + generator.expr(value.Arguments[0].Value) + ")"
	case "document":
		typeArg := `"Article"`
		if len(value.Arguments) > 4 && !value.Arguments[4].UsesDefault {
			typeArg = argument(4)
		}
		theorems := "AhdBuildPair([]string{}, []string{})"
		if len(value.Arguments) > 8 && !value.Arguments[8].UsesDefault && value.Arguments[8].Value != nil {
			theorems = generator.expr(value.Arguments[8].Value)
		}
		theme := `"Default"`
		if len(value.Arguments) > 9 && !value.Arguments[9].UsesDefault {
			theme = argument(9)
		}
		return "AhdLatexDocumentFull(" + argument(0) + ", " + argument(1) + ", " + argument(2) + ", " + argument(3) + ", " + typeArg + ", " + realArgument(5, "2.54") + ", " + argument(6) + ", " + argument(7) + ", " + theorems + ", " + theme + ")"
	case "table":
		if len(value.Arguments) < 2 || value.Arguments[0].Value == nil || value.Arguments[1].Value == nil {
			generator.fail(CodeGenerationFailure, "Latex.table has a missing argument", meta.Span, "the IR call is malformed")
			return `""`
		}
		// An omitted mathColumns is an empty List, so every cell stays text.
		columns := "AhdNewList[int64]()"
		if len(value.Arguments) > 2 && !value.Arguments[2].UsesDefault && value.Arguments[2].Value != nil {
			columns = generator.expr(value.Arguments[2].Value)
		}
		return "AhdLatexTable(" + generator.expr(value.Arguments[0].Value) + ", " +
			generator.expr(value.Arguments[1].Value) + ", " + columns + ")"
	default:
		return generator.unsupported("Latex function "+name, meta.Span)
	}
}

func (generator *generator) latexSizePair(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
		return "AhdBuildPair([]string{}, []float64{})"
	}
	return generator.expr(value.Arguments[index].Value)
}
