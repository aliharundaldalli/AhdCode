package evaluator

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
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
	case "image":
		return s.latexImage(str(0, ""), pairArg(args, 1))
	case "figure":
		marker, image := s.latexAsset(str(0, ""))
		return marker + "\\begin{figure}[!ht]\n\\centering\n\\includegraphics" + s.latexSizes(pairArg(args, 3)) + "{" + image + "}\n\\caption{" + latexEscape(str(1, "")) + "}\n" + latexLabel(str(2, "")) + "\\end{figure}\n"
	case "bibliography":
		return s.latexBibliography(pairArg(args, 0))
	case "document":
		return s.latexDocument(args)
	case "table":
		return s.latexTable(args)
	}
	s.raise("Error", "unsupported Latex function "+name)
	return nil
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
func (s *Session) latexAsset(path string) (string, string) {
	ext := strings.ToLower(filepath.Ext(path))
	if path == "" || (ext != ".png" && ext != ".pdf" && ext != ".jpg" && ext != ".jpeg") {
		s.raise("ValueError", "Latex image supports PNG, PDF, and JPEG assets")
	}
	sum := sha256.Sum256([]byte(path))
	staged := fmt.Sprintf("ahdasset-%x%s", sum[:8], ext)
	return "% AHDCODE_ASSET " + base64.RawStdEncoding.EncodeToString([]byte(path)) + " " + staged + "\n", staged
}
func (s *Session) latexSizes(p *Pair) string {
	p = s.requirePair(p)
	options := []string{}
	for _, key := range p.Keys {
		k := key.(string)
		if k != "width" && k != "height" {
			s.raise("ValueError", "Latex image size supports only width and height")
		}
		value := p.Values[key].(float64)
		if value <= 0 {
			s.raise("ValueError", "Latex image dimensions must be positive")
		}
		options = append(options, k+"="+formatReal(value)+"cm")
	}
	if len(options) == 0 {
		return ""
	}
	return "[" + strings.Join(options, ",") + "]"
}
func (s *Session) latexImage(path string, size *Pair) string {
	marker, image := s.latexAsset(path)
	return marker + "\\includegraphics" + s.latexSizes(size) + "{" + image + "}\n"
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
	body, title, author, date, kind := get(0, ""), get(1, ""), get(2, ""), get(3, ""), get(4, "Article")
	classes := map[string]string{"Article": "article", "Report": "report", "Beamer": "beamer"}
	class := classes[kind]
	if class == "" {
		s.raise("ValueError", "Latex.document type must be Article, Report, or Beamer")
	}
	margin := 2.54
	if len(args) > 5 && args[5] != nil {
		margin = numericFloat(args[5])
	}
	color, cover := get(6, ""), get(7, "")
	if margin <= 0 {
		s.raise("ValueError", "Latex.document margin must be positive")
	}
	if color != "" && !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(color) {
		s.raise("ValueError", "Latex.document color must use #RRGGBB")
	}
	theorems := pairArg(args, 8)
	var b strings.Builder
	b.WriteString("\\documentclass{" + class + "}\n\\usepackage{fontspec}\n\\setmainfont{lmroman10-regular.otf}[BoldFont=lmroman10-bold.otf,ItalicFont=lmroman10-italic.otf,BoldItalicFont=lmroman10-bolditalic.otf]\n\\usepackage{amsmath,amssymb,mathtools}\n\\usepackage{geometry,graphicx,booktabs,array,xcolor,hyperref}\n\\geometry{margin=" + formatReal(margin) + "cm}\n\\hypersetup{hidelinks}\n")
	declared := map[string]string{}
	for _, key := range theorems.Keys {
		display, rule := key.(string), theorems.Values[key].(string)
		if display == "" {
			s.raise("ValueError", "theorem type name must not be empty")
		}
		id := latexTheoremID(display)
		switch {
		case rule == "":
			b.WriteString("\\newtheorem{" + id + "}{" + latexEscape(display) + "}\n")
		case rule == "section" || rule == "subsection":
			b.WriteString("\\newtheorem{" + id + "}{" + latexEscape(display) + "}[" + rule + "]\n")
		case rule == "chapter":
			if kind != "Report" {
				s.raise("ValueError", "chapter theorem counters require a Report document")
			}
			b.WriteString("\\newtheorem{" + id + "}{" + latexEscape(display) + "}[chapter]\n")
		default:
			shared := declared[rule]
			if shared == "" {
				s.raise("ValueError", "theorem counter references an unknown or later type: "+rule)
			}
			b.WriteString("\\newtheorem{" + id + "}[" + shared + "]{" + latexEscape(display) + "}\n")
		}
		declared[display] = id
	}
	knownTheorems := map[string]bool{}
	for _, id := range declared {
		knownTheorems[id] = true
	}
	for _, match := range latexTheoremPattern.FindAllStringSubmatch(body, -1) {
		if !knownTheorems[match[1]] {
			s.raise("ValueError", "document body uses an undeclared theorem type")
		}
	}
	if color != "" {
		b.WriteString("\\definecolor{ahdaccent}{HTML}{" + strings.ToUpper(strings.TrimPrefix(color, "#")) + "}\n")
	}
	if title != "" {
		b.WriteString("\\title{" + latexEscape(title) + "}\n")
	}
	if author != "" {
		b.WriteString("\\author{" + latexEscape(author) + "}\n")
	}
	b.WriteString("\\date{" + latexEscape(date) + "}\n\\begin{document}\n")
	if cover != "" {
		b.WriteString(ensureNewline(cover) + "\\clearpage\n")
	}
	if title != "" {
		if kind == "Beamer" {
			b.WriteString("\\begin{frame}\n\\titlepage\n\\end{frame}\n")
		} else {
			b.WriteString("\\maketitle\n")
		}
	}
	b.WriteString(ensureNewline(body) + "\\end{document}\n")
	return b.String()
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
