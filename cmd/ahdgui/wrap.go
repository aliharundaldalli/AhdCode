package main

import (
	"image"
	"image/color"
	"math"
	"strings"
)

// A wrapping Label breaks its text to a width. Wrapping is off until
// setWrap, so an ordinary Label stays one line and keeps the Phase 1 size
// and weight. The wrap width is the larger of the preferred and minimum
// widths when either is set, even when that is narrower than the text.
// With neither set, only explicit line breaks split the text.

func (m *model) measureLabel(target *widget) {
	if !target.wrap {
		target.natW = labelTextWidth(m.measure, string(target.text), target.labelSize, target.bold)
		target.natH = labelHeight(target.labelSize)
		return
	}
	wrapWidth := 0
	if target.prefW > 0 || target.minW > 0 {
		wrapWidth = max(target.prefW, target.minW)
	}
	lines := labelLines(m.measure, target.text, target.labelSize, target.bold, wrapWidth)
	width := 0
	for _, line := range lines {
		width = max(width, labelTextWidth(m.measure, line, target.labelSize, target.bold))
	}
	if wrapWidth > 0 {
		width = wrapWidth
	}
	target.natW = width
	target.natH = labelHeight(target.labelSize) * max(len(lines), 1)
}

// labelTextWidth is the width Label measurement and wrapping both use, so a
// bold or larger Label wraps where it will be drawn.
func labelTextWidth(measure func(string) int, text, size string, bold bool) int {
	width := int(float64(measure(text))*labelScale(size) + 0.5)
	if bold && width > 0 {
		width += max(1, width/20)
	}
	return width
}

// labelFloor is the narrowest width a wrapping Label can use. A minimum
// width is kept, and so is the widest grapheme cluster, which is drawn
// whole even when it is wider than the line.
func (m *model) labelFloor(w *widget) int {
	floor := w.minW
	runes := []rune(w.text)
	for start := 0; start < len(runes); {
		end := nextCluster(runes, start)
		if end <= start {
			end = start + 1
		}
		floor = max(floor, labelTextWidth(m.measure, string(runes[start:end]), w.labelSize, w.bold))
		start = end
	}
	if floor < 1 {
		floor = 1
	}
	return floor
}

// labelBox is a wrapping Label's size at offer. A non-positive offer, or an
// offer at least as wide as the measured width, keeps the measured size.
// A narrower offer reflows onto more lines and never shortens the Label.
func (m *model) labelBox(w *widget, offer int) (int, int) {
	if w == nil || !w.wrap {
		if w == nil {
			return 0, 0
		}
		return w.natW, w.natH
	}
	width := w.natW
	if offer > 0 && offer < width {
		width = max(offer, m.labelFloor(w))
	}
	if width >= w.natW {
		return w.natW, w.natH
	}
	lines := labelLines(m.measure, w.text, w.labelSize, w.bold, width)
	height := labelHeight(w.labelSize) * max(len(lines), 1)
	if height < w.natH {
		height = w.natH
	}
	return width, height
}

// flowLimit is the width a vertical stack offers its children. Inside a
// scroll container the offer is the viewport, inset by this stack's padding,
// when that is narrower than the stack itself.
func (m *model) flowLimit(padding, inner int) int {
	if m.wrapOffer <= 0 {
		return inner
	}
	visible := m.wrapOffer
	if padding > 0 {
		visible = max(m.wrapOffer-2*padding, 0)
	}
	if inner <= 0 || visible < inner {
		return visible
	}
	return inner
}

// childSize is the size a child takes when its parent offers it a width.
// Only a wrapping Label, and a container that holds one, can become
// narrower than its measured width. Every other widget keeps its natural size.
func (m *model) childSize(w *widget, offer int) (int, int) {
	if w == nil || w.hidden {
		return 0, 0
	}
	switch w.kind {
	case kindLabel:
		return m.labelBox(w, offer)
	case kindColumn, kindPage, kindRadioGroup:
		return m.flowDown(w, offer)
	case kindGrid:
		return m.flowGrid(w, offer)
	case kindTabs:
		return m.flowTabs(w, offer)
	case kindScroll:
		width := w.natW
		if offer > 0 && offer < width {
			width = offer
		}
		return width, w.natH
	default:
		return w.natW, w.natH
	}
}

// flowBox is childSize for an id.
func (m *model) flowBox(id int64, offer int) (int, int) {
	return m.childSize(m.widgets[id], offer)
}

// flowDown stacks children at the offered width and returns the stack's size.
func (m *model) flowDown(target *widget, offer int) (int, int) {
	limit := offer
	if limit > 0 {
		limit = max(limit-2*target.padding, 0)
	}
	limit = m.flowLimit(target.padding, limit)
	width, height, seen := 0, 0, 0
	for _, child := range target.children {
		placed := m.widgets[child]
		if placed.hidden {
			continue
		}
		childW, childH := m.childSize(placed, limit)
		if seen > 0 {
			height += target.spacing
		}
		seen++
		width = max(width, childW)
		height += childH
	}
	return width + 2*target.padding, height + 2*target.padding
}

// wrappedLines is the text a wrapping Label draws. A placed width that
// would need more lines than the box holds falls back to the measured
// width, whose lines fit the measured height.
func (m *model) wrappedLines(w *widget) []string {
	width := w.w
	if width <= 0 {
		width = w.natW
	}
	lines := labelLines(m.measure, w.text, w.labelSize, w.bold, width)
	lineH := labelHeight(w.labelSize)
	if lineH < 1 {
		lineH = 1
	}
	fit := 0
	if w.h > 0 {
		fit = w.h / lineH
		if fit < 1 {
			fit = 1
		}
	}
	if fit > 0 && len(lines) > fit && w.natW > width {
		measured := labelLines(m.measure, w.text, w.labelSize, w.bold, w.natW)
		if len(measured) <= fit {
			return measured
		}
	}
	return lines
}

// labelLines splits text. A non-positive wrapWidth keeps explicit line
// breaks and does not wrap words. \r\n and \r are line breaks.
func labelLines(measure func(string) int, text []rune, size string, bold bool, wrapWidth int) []string {
	normalized := strings.ReplaceAll(string(text), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	parts := strings.Split(normalized, "\n")
	var lines []string
	for _, part := range parts {
		if wrapWidth <= 0 {
			lines = append(lines, part)
			continue
		}
		lines = append(lines, wrapParagraph(measure, part, size, bold, wrapWidth)...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// wrapParagraph breaks one paragraph on spaces. A run of spaces becomes one
// space. A word wider than the line is split on grapheme clusters, and one
// cluster wider than the line is kept whole so it can be clipped.
func wrapParagraph(measure func(string) int, paragraph, size string, bold bool, wrapWidth int) []string {
	words := strings.Fields(paragraph)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := ""
	for _, word := range words {
		for _, piece := range breakWord(measure, word, size, bold, wrapWidth) {
			if current == "" {
				current = piece
				continue
			}
			candidate := current + " " + piece
			if labelTextWidth(measure, candidate, size, bold) <= wrapWidth {
				current = candidate
			} else {
				lines = append(lines, current)
				current = piece
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func breakWord(measure func(string) int, word, size string, bold bool, wrapWidth int) []string {
	if labelTextWidth(measure, word, size, bold) <= wrapWidth {
		return []string{word}
	}
	runes := []rune(word)
	var pieces []string
	start := 0
	for start < len(runes) {
		end := start
		next := start
		for next < len(runes) {
			cluster := nextCluster(runes, next)
			if cluster <= next {
				cluster = next + 1
			}
			if labelTextWidth(measure, string(runes[start:cluster]), size, bold) > wrapWidth && end > start {
				break
			}
			end = cluster
			next = cluster
			if labelTextWidth(measure, string(runes[start:end]), size, bold) > wrapWidth {
				break
			}
		}
		if end == start {
			end = nextCluster(runes, start)
			if end <= start {
				end = start + 1
			}
		}
		pieces = append(pieces, string(runes[start:end]))
		start = end
	}
	return pieces
}

// drawWrappedLabel paints each line from the top of the box and clips to
// the box. A wider placed width reflows and the allocated height stays the
// measured height. A narrower placed width keeps the measured line breaks
// when those lines fit, so a lower measured line is not dropped.
func (m *model) drawWrappedLabel(img *image.RGBA, scale float64, box image.Rectangle, w *widget, foreground color.Color) {
	lines := m.wrappedLines(w)
	face := faceAt(scale * labelScale(w.labelSize))
	linePx := max(1, int(math.Round(float64(labelHeight(w.labelSize))*scale)))
	offset := 0
	if w.bold {
		offset = max(1, int(math.Round(scale)))
	}
	for index, line := range lines {
		top := box.Min.Y + index*linePx
		if top >= box.Max.Y {
			break
		}
		lineBox := image.Rect(box.Min.X, top, box.Max.X, min(top+linePx, box.Max.Y))
		text(img, face, box, box.Min.X, lineBox, line, foreground)
		if w.bold {
			text(img, face, box, box.Min.X+offset, lineBox, line, foreground)
		}
	}
}
