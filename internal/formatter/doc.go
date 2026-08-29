package formatter

import (
	"strings"
	"unicode/utf8"
)

// doc is a small Wadler/Lindig-style pretty-printing document. It is the
// intermediate form the AST is lowered into before canonical AhdCode source
// is rendered: text is opaque content, group tries to render its content on
// one line and only breaks (turning every contained line into a real
// newline) when it does not fit within maxLineWidth, and ifBreak lets a
// construct choose different content depending on whether its enclosing
// group ended up flat or broken.
type dKind uint8

const (
	dText dKind = iota
	dHard
	dRaw
	dConcat
	dIndent
	dGroup
	dBreakGroup
	dIfBreak
)

type doc struct {
	kind  dKind
	text  string
	items []doc // dConcat children; for dIfBreak: items[0]=flat, items[1]=broken
	inner *doc  // dIndent, dGroup, dBreakGroup
}

const maxLineWidth = 80

func text(s string) doc { return doc{kind: dText, text: s} }

func hardline() doc { return doc{kind: dHard} }

// rawNewline renders a bare "\n" with no indent padding. It exists only to
// rejoin the chunks a multiline block comment gets split into at each
// internal newline (see lexer.scanBlockComment): every chunk after the
// first already carries its own original leading whitespace, so padding it
// again with canonical indentation would double it up.
func rawNewline() doc { return doc{kind: dRaw} }

func concat(items ...doc) doc { return doc{kind: dConcat, items: items} }

func indent(inner doc) doc { return doc{kind: dIndent, inner: &inner} }

func group(inner doc) doc { return doc{kind: dGroup, inner: &inner} }

// breakGroup behaves like a group whose content always renders broken,
// bypassing the width check. Used when a group's interior contains a comment
// or another construct that can never be safely flattened to one line.
func breakGroup(inner doc) doc { return doc{kind: dBreakGroup, inner: &inner} }

// ifBreak renders flatDoc when the enclosing group is flat and breakDoc when
// the enclosing group is broken.
func ifBreak(breakDoc, flatDoc doc) doc {
	return doc{kind: dIfBreak, items: []doc{flatDoc, breakDoc}}
}

func runeWidth(s string) int { return utf8.RuneCountInString(s) }

type renderMode uint8

const (
	modeFlat renderMode = iota
	modeBreak
)

type workItem struct {
	ind  int
	mode renderMode
	d    doc
}

// render lays out doc for a page of the given width using the classic
// "does it fit" pretty-printing algorithm (Wadler 1998, as reformulated by
// Lindig's "Strictly Pretty"). Every dGroup independently attempts a flat
// rendering, consulting how much of the *rest* of the document (already
// queued after it) still needs to land on the same line before the next
// unavoidable break; it falls back to a broken rendering only when that
// does not fit.
func render(root doc, width int) string {
	var out strings.Builder
	col := 0
	stack := []workItem{{0, modeBreak, root}}
	for len(stack) > 0 {
		item := stack[0]
		stack = stack[1:]
		switch item.d.kind {
		case dText:
			out.WriteString(item.d.text)
			if idx := strings.LastIndexByte(item.d.text, '\n'); idx >= 0 {
				col = runeWidth(item.d.text[idx+1:])
			} else {
				col += runeWidth(item.d.text)
			}
		case dConcat:
			stack = spliceChildren(stack, item, item.d.items)
		case dHard:
			out.WriteByte('\n')
			pad := strings.Repeat("    ", item.ind)
			out.WriteString(pad)
			col = len(pad)
		case dRaw:
			out.WriteByte('\n')
			col = 0
		case dIndent:
			stack = append([]workItem{{item.ind + 1, item.mode, *item.d.inner}}, stack...)
		case dIfBreak:
			chosen := item.d.items[0]
			if item.mode == modeBreak {
				chosen = item.d.items[1]
			}
			stack = append([]workItem{{item.ind, item.mode, chosen}}, stack...)
		case dGroup:
			flatAttempt := workItem{item.ind, modeFlat, *item.d.inner}
			if fits(width-col, flatAttempt, stack) {
				stack = append([]workItem{flatAttempt}, stack...)
			} else {
				stack = append([]workItem{{item.ind, modeBreak, *item.d.inner}}, stack...)
			}
		case dBreakGroup:
			stack = append([]workItem{{item.ind, modeBreak, *item.d.inner}}, stack...)
		}
	}
	return out.String()
}

func spliceChildren(stack []workItem, parent workItem, children []doc) []workItem {
	expanded := make([]workItem, 0, len(children)+len(stack))
	for _, child := range children {
		expanded = append(expanded, workItem{parent.ind, parent.mode, child})
	}
	return append(expanded, stack...)
}

// fits reports whether rendering `first` (tentatively flat) followed by
// whatever is already queued in `rest` can complete the current output line
// without exceeding width. It stops as soon as it reaches a hard break or a
// line that is already in broken mode, since everything after that point is
// guaranteed to start a fresh line regardless of what this group decides.
func fits(width int, first workItem, rest []workItem) bool {
	stack := append([]workItem{first}, rest...)
	for {
		if width < 0 {
			return false
		}
		if len(stack) == 0 {
			return true
		}
		item := stack[0]
		stack = stack[1:]
		switch item.d.kind {
		case dText:
			width -= runeWidth(item.d.text)
			if strings.ContainsRune(item.d.text, '\n') {
				return true
			}
		case dConcat:
			stack = spliceChildren(stack, item, item.d.items)
		case dHard, dRaw:
			return true
		case dIndent:
			stack = append([]workItem{{item.ind + 1, item.mode, *item.d.inner}}, stack...)
		case dIfBreak:
			chosen := item.d.items[0]
			if item.mode == modeBreak {
				chosen = item.d.items[1]
			}
			stack = append([]workItem{{item.ind, item.mode, chosen}}, stack...)
		case dGroup:
			stack = append([]workItem{{item.ind, modeFlat, *item.d.inner}}, stack...)
		case dBreakGroup:
			return true
		}
	}
}
