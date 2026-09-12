package evaluator

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
)

func latexEscape(text string) string {
	var b strings.Builder
	for _, c := range text {
		switch c {
		case '\\':
			b.WriteString(`\textbackslash{}`)
		case '{':
			b.WriteString(`\{`)
		case '}':
			b.WriteString(`\}`)
		case '$':
			b.WriteString(`\$`)
		case '&':
			b.WriteString(`\&`)
		case '#':
			b.WriteString(`\#`)
		case '%':
			b.WriteString(`\%`)
		case '_':
			b.WriteString(`\_`)
		case '^':
			b.WriteString(`\textasciicircum{}`)
		case '~':
			b.WriteString(`\textasciitilde{}`)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}
func latexLabel(label string) string {
	if label == "" {
		return ""
	}
	return "\\label{" + latexEscape(label) + "}\n"
}
func latexTheoremID(name string) string {
	sum := sha256.Sum256([]byte(name))
	return fmt.Sprintf("ahdthm%x", sum[:6])
}

var latexTheoremPattern = regexp.MustCompile(`\\begin\{(ahdthm[0-9a-f]+)\}`)

var latexBeamerThemes = map[string]bool{"Default": true, "Madrid": true, "Warsaw": true}

func (s *Session) latexBuiltin(name string, args []any) any {
	str := func(i int, fallback string) string {
		if i >= len(args) || args[i] == nil {
			return fallback
		}
		return args[i].(string)
	}
	switch name {
	case "pdf", "pdfFile":
		s.raise("LatexError", "Latex PDF compilation is not available in the interactive evaluator")
	case "escape":
		return latexEscape(str(0, ""))
	case "chapter":
		return "\\chapter{" + latexEscape(str(0, "")) + "}\n"
	case "section":
		return "\\section{" + latexEscape(str(0, "")) + "}\n"
	case "subsection":
		return "\\subsection{" + latexEscape(str(0, "")) + "}\n"
	case "frame":
		return "\\begin{frame}{" + latexEscape(str(0, "")) + "}\n" + ensureNewline(str(1, "")) + "\\end{frame}\n"
	case "equation":
		return "\\begin{equation}\n" + str(0, "") + "\n" + latexLabel(str(1, "")) + "\\end{equation}\n"
	case "theorem":
		kind := str(0, "")
		if kind == "" {
			s.raise("ValueError", "Latex.theorem type must not be empty")
		}
		return "\\begin{" + latexTheoremID(kind) + "}\n" + ensureNewline(str(1, "")) + latexLabel(str(2, "")) + "\\end{" + latexTheoremID(kind) + "}\n"
	case "ref":
		return "\\ref{" + latexEscape(str(0, "")) + "}"
	case "cite":
		return "\\cite{" + latexEscape(str(0, "")) + "}"
	case "center":
		return "\\begin{center}\n" + ensureNewline(str(0, "")) + "\\end{center}\n"
	case "pageBreak":
		return "\\clearpage\n"
	case "contents":
		return "\\tableofcontents\n"
	case "minipage":
		width := numericFloat(args[1])
		align := str(2, "left")
		command := map[string]string{"left": "\\raggedright", "center": "\\centering", "right": "\\raggedleft"}[align]
		if width <= 0 || command == "" {
			s.raise("ValueError", "invalid Latex.minipage width or alignment")
		}
		return "\\begin{minipage}{" + formatReal(width) + "cm}\n" + command + "\n" + ensureNewline(str(0, "")) + "\\end{minipage}\n"
	case "image", "figure":
		// The runtime builds image and figure fragments -- including v1.3.0
		// transforms and SVG assets -- so evaluator output is identical to a
		// compiled program's.
		sizeIndex, transformIndex := 1, 2
		caption, label := "", ""
		if name == "figure" {
			sizeIndex, transformIndex = 3, 4
			caption, label = str(1, ""), str(2, "")
		}
		sizeKeys, sizeValues := s.latexRealEntries(pairArg(args, sizeIndex))
		transformKeys, transformValues := s.latexRealEntries(pairArg(args, transformIndex))
		text, problem := ahdruntime.AhdLatexImageText(str(0, ""), sizeKeys, sizeValues, transformKeys, transformValues,
			name == "figure", caption, label)
		return s.latexText(text, problem)
	case "qr":
		return s.latexText(ahdruntime.AhdLatexQRText("Latex.qr", str(0, ""), latexReal(args, 1, 3.0), str(2, "M")))
	case "barcode":
		return s.latexText(ahdruntime.AhdLatexBarcodeText("Latex.barcode", str(0, ""), str(1, ""), latexReal(args, 2, 8.0), latexReal(args, 3, 2.0)))
	case "place":
		return s.latexText(ahdruntime.AhdLatexPlaceText(str(0, ""), latexReal(args, 1, 0), latexReal(args, 2, 0), str(3, "north west")))
	case "header":
		return ahdruntime.AhdLatexRunningText("head", str(0, ""), str(1, ""), str(2, ""))
	case "footer":
		return ahdruntime.AhdLatexRunningText("foot", str(0, ""), str(1, ""), str(2, ""))
	case "pageNumber":
		return ahdruntime.AhdLatexPageNumberText()
	case "pageCount":
		return ahdruntime.AhdLatexPageCountText()
	case "link":
		return s.latexText(ahdruntime.AhdLatexLinkText("Latex.link", str(0, ""), str(1, "")))
	case "bookmark":
		level := int64(1)
		if len(args) > 1 && args[1] != nil {
			level = args[1].(int64)
		}
		return s.latexText(ahdruntime.AhdLatexBookmarkText("Latex.bookmark", str(0, ""), level))
	case "bibliography":
		return s.latexBibliography(pairArg(args, 0))
	case "document":
		return s.latexDocument(args)
	case "table":
		return s.latexTable(args)
	case "tikz", "overlay":
		// The runtime builds the fragment, so evaluator output is identical to
		// a compiled program's.
		text, problem := ahdruntime.AhdLatexTikZText(name, str(0, ""), s.latexStrings(args, 1), name == "overlay")
		if problem != "" {
			s.raise("ValueError", problem)
		}
		return text
	case "border":
		inset, thickness := 1.0, 1.0
		if len(args) > 0 && args[0] != nil {
			inset = numericFloat(args[0])
		}
		if len(args) > 1 && args[1] != nil {
			thickness = numericFloat(args[1])
		}
		text, problem := ahdruntime.AhdLatexBorderText(inset, thickness, str(2, ""))
		if problem != "" {
			s.raise("ValueError", problem)
		}
		return text
	}
	s.raise("Error", "unsupported Latex function "+name)
	return nil
}

// latexStrings reads an optional List<String> argument; an omitted list is empty.
func (s *Session) latexStrings(args []any, i int) []string {
	if i >= len(args) || args[i] == nil {
		return nil
	}
	list := s.requireList(args[i])
	items := make([]string, len(list.Items))
	for index, item := range list.Items {
		items[index] = item.(string)
	}
	return items
}

func ensureNewline(text string) string {
	if text != "" && !strings.HasSuffix(text, "\n") {
		return text + "\n"
	}
	return text
}
func pairArg(args []any, i int) *Pair {
	if i >= len(args) || args[i] == nil {
		return &Pair{Values: map[any]any{}}
	}
	return args[i].(*Pair)
}

// latexText returns a runtime-built fragment, raising ValueError for a
// problem the way a compiled program's Latex helpers do.
func (s *Session) latexText(text, problem string) string {
	if problem != "" {
		s.raise("ValueError", problem)
	}
	return text
}

func latexReal(args []any, index int, fallback float64) float64 {
	if index < len(args) && args[index] != nil {
		return numericFloat(args[index])
	}
	return fallback
}

// latexRealEntries reads a Pair<String, Real> argument in insertion order.
func (s *Session) latexRealEntries(p *Pair) ([]string, []float64) {
	p = s.requirePair(p)
	keys := make([]string, len(p.Keys))
	values := make([]float64, len(p.Keys))
	for index, key := range p.Keys {
		keys[index] = key.(string)
		values[index] = numericFloat(p.Values[key])
	}
	return keys, values
}
func (s *Session) latexBibliography(p *Pair) string {
	p = s.requirePair(p)
	var b strings.Builder
	b.WriteString("\\begin{thebibliography}{99}\n")
	for _, key := range p.Keys {
		b.WriteString("\\bibitem{" + latexEscape(key.(string)) + "} " + latexEscape(p.Values[key].(string)) + "\n")
	}
	b.WriteString("\\end{thebibliography}\n")
	return b.String()
}
func (s *Session) latexDocument(args []any) string {
	get := func(i int, def string) string {
		if i >= len(args) || args[i] == nil {
			return def
		}
		return args[i].(string)
	}
	// The runtime builds the whole document, so evaluator output is identical
	// to a compiled program's for every parameter.
	theorems := s.requirePair(pairArg(args, 8))
	names := make([]string, len(theorems.Keys))
	rules := make([]string, len(theorems.Keys))
	for index, key := range theorems.Keys {
		names[index], rules[index] = key.(string), theorems.Values[key].(string)
	}
	sizeKeys, sizeValues := s.latexRealEntries(pairArg(args, 12))
	marginKeys, marginValues := s.latexRealEntries(pairArg(args, 13))
	return s.latexText(ahdruntime.AhdLatexDocumentText(ahdruntime.AhdLatexDocumentOptions{
		Body: get(0, ""), Title: get(1, ""), Author: get(2, ""), Date: get(3, ""), Type: get(4, "Article"),
		Margin: latexReal(args, 5, 2.54), Color: get(6, ""), Cover: get(7, ""),
		TheoremNames: names, TheoremRules: rules, Theme: get(9, "Default"),
		Landscape: len(args) > 10 && args[10] != nil && args[10].(bool), Paper: get(11, "Letter"),
		PageSizeKeys: sizeKeys, PageSizeValues: sizeValues, MarginKeys: marginKeys, MarginValues: marginValues,
		Subject: get(14, ""), Keywords: s.latexStrings(args, 15), Creator: get(16, ""),
	}))
}
func (s *Session) latexTable(args []any) string {
	headers := s.requireList(args[0])
	rows := s.requireList(args[1])
	mathColumns := &List{}
	if len(args) > 2 && args[2] != nil {
		mathColumns = s.requireList(args[2])
	}
	if len(headers.Items) == 0 {
		s.raise("ValueError", "Latex.table requires at least one header")
	}
	mathSet := map[int64]bool{}
	for _, x := range mathColumns.Items {
		index := x.(int64)
		if index < 0 || index >= int64(len(headers.Items)) {
			s.raise("ValueError", fmt.Sprintf("Latex.table math column %d is outside 0..%d", index, len(headers.Items)-1))
		}
		mathSet[index] = true
	}
	var b strings.Builder
	b.WriteString("\\begin{tabular}{" + strings.Repeat("l", len(headers.Items)) + "}\n\\toprule\n")
	for i, x := range headers.Items {
		if i > 0 {
			b.WriteString(" & ")
		}
		b.WriteString(latexEscape(x.(string)))
	}
	b.WriteString(" \\\\\n\\midrule\n")
	for _, item := range rows.Items {
		row := s.requireList(item)
		if len(row.Items) != len(headers.Items) {
			s.raise("ValueError", "Latex.table row column count does not match headers")
		}
		for i, x := range row.Items {
			if i > 0 {
				b.WriteString(" & ")
			}
			if mathSet[int64(i)] {
				b.WriteString("\\(" + x.(string) + "\\)")
			} else {
				b.WriteString(latexEscape(x.(string)))
			}
		}
		b.WriteString(" \\\\\n")
	}
	b.WriteString("\\bottomrule\n\\end{tabular}\n")
	return b.String()
}
