package main

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
)

// Drawing of the scrolling and choosing widgets: TextArea, ListBox, Select
// and its open list, and TableView. Only the visible rows of a list or table
// are drawn, so a TableView of many rows costs no more than one screenful.

// maskRune is what a PasswordInput shows for each character.
const maskRune = "•"

var (
	colorSelection = color.RGBA{214, 226, 242, 255} // a selection without the focus
	colorHeader    = color.RGBA{232, 232, 232, 255}
	colorStripe    = color.RGBA{247, 247, 247, 255}
	colorGrid      = color.RGBA{218, 218, 218, 255}
	colorThumb     = color.RGBA{170, 170, 170, 255}
	colorOnFocus   = color.RGBA{255, 255, 255, 255}
)

// fieldFrame fills a widget's background and draws its border, blue while
// it has the focus.
func fieldFrame(img *image.RGBA, w *widget, box image.Rectangle, line int, focused bool) {
	fill(img, box, or(w.background, colorField))
	border := colorBorder
	if focused {
		border = colorFocus
	}
	outline(img, box, border, line)
}

// drawScrollbar draws a vertical scroll bar when the content is taller than
// the view, and a thin horizontal position bar when it is wider.
func drawScrollbar(img *image.RGBA, s float64, v viewport, scrollX, scrollY int) {
	if v.vertical {
		fill(img, scaled(s, v.x+v.w, v.y, scrollbar, v.h), colorHeader)
		top, height := thumb(v, scrollY)
		fill(img, scaled(s, v.x+v.w+2, top+1, scrollbar-4, max(height-2, 1)), colorThumb)
	}
	if v.horizontalScroll && v.contentW > 0 {
		width := max(v.w*v.w/v.contentW, 16)
		left := 0
		if span := v.contentW - v.w; span > 0 {
			left = (v.w - width) * scrollX / span
		}
		fill(img, scaled(s, v.x+left, v.y+v.h-4, width, 3), colorThumb)
	}
}

// rowColors are the background and text colors of one list row or table
// row.
func rowColors(selected, focused bool, foreground color.Color) (color.Color, color.Color) {
	switch {
	case selected && focused:
		return colorFocus, colorOnFocus
	case selected:
		return colorSelection, foreground
	}
	return nil, foreground
}

func drawTextArea(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, line int, focused bool, foreground color.Color) {
	s, m := f.scale, f.model
	fieldFrame(img, w, box, line, focused)
	v := m.viewportOf(w)
	view := scaled(s, v.x, v.y, v.w, v.h)
	if len(w.text) == 0 && !focused {
		text(img, face, view, view.Min.X, scaled(s, v.x, v.y, v.w, lineHeight), w.placeholder, colorMuted)
		return
	}
	lines := areaLines(w.text)
	caretLine, caretColumn := areaPosition(w.text, w.caret)
	// Keep the caret inside the view: scroll the text horizontally.
	caretX := font.MeasureString(face, string(lines[caretLine][:caretColumn])).Ceil()
	if caretX-w.scroll > view.Dx()-line {
		w.scroll = caretX - view.Dx() + line
	}
	if caretX < w.scroll {
		w.scroll = caretX
	}
	first := w.scrollY / lineHeight
	for index := first; index < len(lines) && index*lineHeight-w.scrollY < v.h; index++ {
		row := scaled(s, v.x, v.y+index*lineHeight-w.scrollY, v.w, lineHeight)
		text(img, face, view, view.Min.X-w.scroll, row, string(lines[index]), foreground)
	}
	if focused && f.caretVisible {
		row := scaled(s, v.x, v.y+caretLine*lineHeight-w.scrollY, v.w, lineHeight)
		x := view.Min.X + caretX - w.scroll
		fill(img, image.Rect(x, row.Min.Y+2*line, x+line, row.Max.Y-2*line).Intersect(view), foreground)
	}
	drawScrollbar(img, s, v, 0, w.scrollY)
}

func drawListBox(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, line int, focused bool, foreground color.Color) {
	s, m := f.scale, f.model
	fieldFrame(img, w, box, line, focused)
	v := m.viewportOf(w)
	view := scaled(s, v.x, v.y, v.w, v.h)
	inset := int(math.Round(itemInset * s))
	for index := w.scrollY / itemHeight; index < len(w.items) && index*itemHeight-w.scrollY < v.h; index++ {
		row := scaled(s, v.x, v.y+index*itemHeight-w.scrollY, v.w, itemHeight)
		background, ink := rowColors(index == w.selected, focused, foreground)
		if background != nil {
			fill(img, row.Intersect(view), background)
		}
		text(img, face, view, row.Min.X+inset, row, w.items[index], ink)
	}
	drawScrollbar(img, s, v, 0, w.scrollY)
}

func drawSelect(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, line int, focused bool, foreground color.Color) {
	s := f.scale
	fieldFrame(img, w, box, line, focused)
	inset := int(math.Round(itemInset * s))
	arrow := int(math.Round(selectArrow * s))
	inner := image.Rect(box.Min.X+inset, box.Min.Y+line, box.Max.X-arrow, box.Max.Y-line)
	if w.selected >= 0 && w.selected < len(w.items) {
		text(img, face, inner, inner.Min.X, box, w.items[w.selected], foreground)
	}
	// A small downward triangle marks the Select.
	centerX := float64(box.Max.X) - float64(arrow)/2
	centerY := float64(box.Min.Y+box.Max.Y) / 2
	half := 4 * s
	for y := 0; y < int(math.Ceil(half)); y++ {
		span := half * (1 - float64(y)/half)
		row := int(math.Round(centerY - half/2 + float64(y)))
		fill(img, image.Rect(int(math.Round(centerX-span)), row, int(math.Round(centerX+span)), row+1), foreground)
	}
}

// drawPopup draws a Select's open list over everything else.
func drawPopup(img *image.RGBA, face font.Face, f frame, w *widget) {
	s, m := f.scale, f.model
	line := max(1, int(math.Round(s)))
	box := scaled(s, m.list.x, m.list.y, m.list.w, m.list.h)
	fill(img, box, colorField)
	outline(img, box, colorFocus, line)
	view := box.Inset(line)
	inset := int(math.Round(itemInset * s))
	foreground := or(w.foreground, colorText)
	for index := w.listScroll / itemHeight; index < len(w.items) && index*itemHeight-w.listScroll < m.list.h-2; index++ {
		row := scaled(s, m.list.x+1, m.list.y+1+index*itemHeight-w.listScroll, m.list.w-2, itemHeight)
		background, ink := rowColors(index == w.highlight, true, foreground)
		if background == nil && index == w.selected {
			background = colorSelection
		}
		if background != nil {
			fill(img, row.Intersect(view), background)
		}
		text(img, face, view, row.Min.X+inset, row, w.items[index], ink)
	}
}

func drawTable(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, line int, focused bool, foreground color.Color) {
	s, m := f.scale, f.model
	fieldFrame(img, w, box, line, focused)
	v := m.viewportOf(w)
	inset := int(math.Round(itemInset * s))
	header := scaled(s, v.x, w.y+1, v.w, headerHeight)
	fill(img, header, colorHeader)
	fill(img, scaled(s, v.x, v.y-1, v.w, 1), colorBorder)
	view := scaled(s, v.x, v.y, v.w, v.h)
	// Column edges, shared by the header and the body.
	lefts := make([]int, len(w.widths))
	left := v.x - w.scrollX
	for index, width := range w.widths {
		lefts[index] = left
		left += width
	}
	for index, title := range w.columns {
		cell := scaled(s, lefts[index], w.y+1, w.widths[index], headerHeight)
		text(img, face, cell.Intersect(header), cell.Min.X+inset, cell, title, foreground)
		fill(img, scaled(s, lefts[index]+w.widths[index]-1, w.y+1, 1, headerHeight).Intersect(header), colorGrid)
	}
	for row := w.scrollY / itemHeight; row < len(w.rows) && row*itemHeight-w.scrollY < v.h; row++ {
		band := scaled(s, v.x, v.y+row*itemHeight-w.scrollY, v.w, itemHeight)
		background, ink := rowColors(row == w.selected, focused, foreground)
		if background == nil && row%2 == 1 {
			background = colorStripe
		}
		if background != nil {
			fill(img, band.Intersect(view), background)
		}
		for index, value := range w.rows[row] {
			cell := scaled(s, lefts[index], v.y+row*itemHeight-w.scrollY, w.widths[index], itemHeight)
			if cell.Max.X < view.Min.X || cell.Min.X > view.Max.X {
				continue
			}
			clip := image.Rect(cell.Min.X, cell.Min.Y, cell.Max.X-inset/2, cell.Max.Y).Intersect(view)
			text(img, face, clip, cell.Min.X+inset, cell, value, ink)
		}
	}
	for index := range w.widths {
		fill(img, scaled(s, lefts[index]+w.widths[index]-1, v.y, 1, v.h).Intersect(view), colorGrid)
	}
	drawScrollbar(img, s, v, w.scrollX, w.scrollY)
}
