package main

// Math text is deliberately a whole-string opt-in. Gonum already provides a
// vector-capable LaTeX text handler for every canvas it supports, so Plot
// labels can share one handler across titles, axes, legends, and categorical
// ticks without adding a second chart-specific renderer or dependency.

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"ahdcode/internal/plotproto"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
)

type mathTextHandler struct {
	plain text.Handler
	latex text.Latex
	mu    sync.Mutex
	valid map[string]error
	order []string
}

var superscriptRunes = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴', '5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'+': '⁺', '-': '⁻', '=': '⁼', '(': '⁽', ')': '⁾', 'n': 'ⁿ', 'i': 'ⁱ',
}

var subscriptRunes = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄', '5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎', 'i': 'ᵢ', 'n': 'ₙ',
}

func newMathTextHandler() *mathTextHandler {
	plain := plot.DefaultTextHandler
	return &mathTextHandler{plain: plain, latex: text.Latex{Fonts: plain.Cache()}, valid: make(map[string]error)}
}

func (h *mathTextHandler) Cache() *font.Cache { return h.plain.Cache() }

func (h *mathTextHandler) Extents(fnt font.Font) font.Extents { return h.plain.Extents(fnt) }

func (h *mathTextHandler) Lines(value string) []string {
	if formula, ok := mathTextFormula(value); ok {
		return []string{formula}
	}
	return h.plain.Lines(value)
}

func (h *mathTextHandler) Box(value string, fnt font.Font) (vg.Length, vg.Length, vg.Length) {
	if formula, ok := mathTextFormula(value); ok {
		if width, height, depth, ok := latexBox(h.latex, formula, fnt); ok {
			return width, height, depth
		}
		return h.plain.Box(mathFallback(formula), fnt)
	}
	return h.plain.Box(value, fnt)
}

func (h *mathTextHandler) Draw(canvas vg.Canvas, value string, style text.Style, point vg.Point) {
	if formula, ok := mathTextFormula(value); ok {
		if latexDraw(h.latex, canvas, formula, style, point) {
			return
		}
		h.plain.Draw(canvas, mathFallback(formula), style, point)
		return
	}
	h.plain.Draw(canvas, value, style, point)
}

func mathTextFormula(value string) (string, bool) {
	if len(value) < 2 || !strings.HasPrefix(value, "$") || !strings.HasSuffix(value, "$") {
		return "", false
	}
	// Gonum's existing LaTeX handler expects the delimiters as part of the
	// expression; keeping them also preserves its normal math-mode parsing.
	return value, true
}

func validateMathText(value, subject string, handler *mathTextHandler) error {
	formula, ok := mathTextFormula(value)
	if !ok {
		return nil
	}
	if strings.TrimSpace(formula) == "" {
		return fmt.Errorf("%s contains an empty math fragment", subject)
	}
	if utf8.RuneCountInString(formula) > 512 {
		return fmt.Errorf("%s is too long (math fragments are limited to 512 characters)", subject)
	}
	if !hasBalancedBraces(formula) {
		return fmt.Errorf("%s contains unmatched braces in math text %q", subject, formula)
	}
	return handler.validateFormula(formula)
}

func hasBalancedBraces(formula string) bool {
	depth := 0
	for i := 0; i < len(formula); i++ {
		if formula[i] == '\\' {
			i++
			continue
		}
		if formula[i] == '{' {
			depth++
		} else if formula[i] == '}' {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func (h *mathTextHandler) validateFormula(formula string) error {
	h.mu.Lock()
	if result, ok := h.valid[formula]; ok {
		h.mu.Unlock()
		return result
	}
	h.mu.Unlock()
	result := validateMathBox(formula, h)
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.valid[formula]; ok {
		return existing
	}
	if len(h.order) >= 128 {
		delete(h.valid, h.order[0])
		h.order = h.order[1:]
	}
	h.valid[formula], h.order = result, append(h.order, formula)
	return result
}

func validateMathBox(formula string, handler *mathTextHandler) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message := fmt.Sprint(recovered)
			if strings.Contains(message, "unknown ast node *ast.Sup") ||
				strings.Contains(message, "unknown ast node *ast.Sub") ||
				(strings.Contains(message, "not implemented") && mathFallbackAllowed(formula)) {
				// The bundled Gonum LaTeX handler accepts the syntax but its
				// current mtex visitor does not lower every small expression
				// node. Those formulas use the controlled plain fallback at draw
				// time. Unknown commands are still rejected below.
				err = nil
				return
			}
			err = fmt.Errorf("invalid math text %q: %v", formula, recovered)
		}
	}()
	_, _, _ = handler.latex.Box(formula, plot.DefaultFont)
	return nil
}

func mathFallbackAllowed(formula string) bool {
	allowed := map[string]bool{
		"alpha": true, "beta": true, "gamma": true, "delta": true,
		"epsilon": true, "lambda": true, "mu": true, "pi": true,
		"sigma": true, "theta": true, "omega": true, "sum": true,
		"int": true, "infty": true, "cdot": true, "times": true,
	}
	for index := 0; index < len(formula); index++ {
		if formula[index] != '\\' {
			continue
		}
		start := index + 1
		end := start
		for end < len(formula) && ((formula[end] >= 'a' && formula[end] <= 'z') || (formula[end] >= 'A' && formula[end] <= 'Z')) {
			end++
		}
		if end == start || !allowed[formula[start:end]] {
			return false
		}
		index = end - 1
	}
	return true
}

func latexBox(handler text.Latex, formula string, fnt font.Font) (width, height, depth vg.Length, ok bool) {
	defer func() { _ = recover() }()
	width, height, depth = handler.Box(formula, fnt)
	return width, height, depth, true
}

func latexDraw(handler text.Latex, canvas vg.Canvas, formula string, style text.Style, point vg.Point) (ok bool) {
	defer func() { _ = recover() }()
	handler.Draw(canvas, formula, style, point)
	return true
}

func mathFallback(formula string) string {
	formula = strings.TrimSuffix(strings.TrimPrefix(formula, "$"), "$")
	formula = strings.ReplaceAll(formula, `\,`, " ")
	for command, replacement := range map[string]string{
		`\alpha`: "α", `\beta`: "β", `\gamma`: "γ", `\delta`: "δ", `\epsilon`: "ε",
		`\lambda`: "λ", `\mu`: "μ", `\pi`: "π", `\sigma`: "σ", `\theta`: "θ", `\omega`: "ω",
		`\sum`: "∑", `\int`: "∫", `\infty`: "∞", `\cdot`: "·", `\times`: "×",
	} {
		formula = strings.ReplaceAll(formula, command, replacement)
	}
	var out []rune
	runes := []rune(formula)
	for index := 0; index < len(runes); index++ {
		if runes[index] != '^' && runes[index] != '_' {
			if runes[index] != '{' && runes[index] != '}' {
				out = append(out, runes[index])
			}
			continue
		}
		superscript := runes[index] == '^'
		index++
		var value []rune
		if index < len(runes) && runes[index] == '{' {
			index++
			for index < len(runes) && runes[index] != '}' {
				value = append(value, runes[index])
				index++
			}
		} else if index < len(runes) {
			value = []rune{runes[index]}
		}
		for _, item := range value {
			if superscript {
				if mapped, ok := superscriptRune(item); ok {
					out = append(out, mapped)
				} else {
					out = append(out, '^', item)
				}
			} else if mapped, ok := subscriptRune(item); ok {
				out = append(out, mapped)
			} else {
				out = append(out, '_', item)
			}
		}
	}
	return string(out)
}

func superscriptRune(value rune) (rune, bool) {
	mapped, ok := superscriptRunes[value]
	return mapped, ok
}

func subscriptRune(value rune) (rune, bool) {
	mapped, ok := subscriptRunes[value]
	return mapped, ok
}

func validateChartMath(spec plotproto.ChartSpec, handler *mathTextHandler) error {
	check := func(value, subject string) error { return validateMathText(value, subject, handler) }
	if err := check(spec.Title, "chart title"); err != nil {
		return err
	}
	if err := check(spec.XLabel, "x label"); err != nil {
		return err
	}
	if err := check(spec.YLabel, "y label"); err != nil {
		return err
	}
	for index, series := range spec.Series {
		if err := check(series.Label, fmt.Sprintf("series %d label", index)); err != nil {
			return err
		}
	}
	labels := append([]string{}, spec.BarLabels...)
	labels = append(labels, spec.PieLabels...)
	labels = append(labels, spec.HeatmapXLabels...)
	labels = append(labels, spec.HeatmapYLabels...)
	for index, value := range labels {
		if err := check(value, fmt.Sprintf("category label %d", index)); err != nil {
			return err
		}
	}
	return nil
}
