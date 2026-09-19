package main

// Scrolling, selection lists, the Select's open list, and TextArea editing.
// Every function here expects the caller to hold m.mu.

// viewport is the part of a scrolling widget that shows content, in logical
// pixels, with the content's full size.
type viewport struct {
	x, y, w, h       int
	contentW         int
	contentH         int
	vertical         bool // a vertical scroll bar is shown
	horizontalScroll bool // the content is wider than the view
}

// viewportOf measures a ListBox, TableView, or TextArea. A vertical scroll
// bar takes scrollbar pixels on the right when the content is taller than
// the view.
func (m *model) viewportOf(target *widget) viewport {
	var v viewport
	switch target.kind {
	case kindListBox:
		v = viewport{x: target.x + 1, y: target.y + 1, w: target.w - 2, h: target.h - 2, contentH: len(target.items) * itemHeight}
		v.contentW = v.w
	case kindTable:
		v = viewport{x: target.x + 1, y: target.y + 1 + headerHeight, w: target.w - 2, h: target.h - 2 - headerHeight,
			contentH: len(target.rows) * itemHeight}
		for _, width := range target.widths {
			v.contentW += width
		}
	case kindTextArea:
		lines := areaLines(target.text)
		v = viewport{x: target.x + textInputInset, y: target.y + textInputInset, w: target.w - 2*textInputInset,
			h: target.h - 2*textInputInset, contentH: len(lines) * lineHeight}
		v.contentW = v.w
	default:
		return v
	}
	v.w, v.h = max(v.w, 0), max(v.h, 0)
	v.vertical = v.contentH > v.h
	if v.vertical {
		v.w = max(v.w-scrollbar, 0)
	}
	v.horizontalScroll = v.contentW > v.w
	return v
}

// clampScroll keeps a widget's scroll inside its content.
func (m *model) clampScroll(target *widget) {
	if !target.scrolling() {
		return
	}
	v := m.viewportOf(target)
	target.scrollY = min(max(target.scrollY, 0), max(v.contentH-v.h, 0))
	if target.kind == kindTable {
		target.scrollX = min(max(target.scrollX, 0), max(v.contentW-v.w, 0))
	}
}

// reveal scrolls a ListBox or TableView so its selection is visible.
func (m *model) reveal(target *widget) {
	if target.kind != kindListBox && target.kind != kindTable {
		return
	}
	if target.selected >= 0 {
		v := m.viewportOf(target)
		top := target.selected * itemHeight
		if top < target.scrollY {
			target.scrollY = top
		}
		if top+itemHeight > target.scrollY+v.h {
			target.scrollY = top + itemHeight - v.h
		}
	}
	m.clampScroll(target)
}

// thumb is the scroll bar thumb's top and height.
func thumb(v viewport, scrollY int) (int, int) {
	height := max(v.h*v.h/max(v.contentH, 1), 16)
	height = min(height, v.h)
	travel := v.h - height
	top := 0
	if span := v.contentH - v.h; span > 0 {
		top = travel * scrollY / span
	}
	return v.y + top, height
}

// pressScrollbar starts dragging a scroll bar under a point.
func (m *model) pressScrollbar(target *widget, x, y int) bool {
	if !target.scrolling() {
		return false
	}
	v := m.viewportOf(target)
	if !v.vertical || x < v.x+v.w || x >= v.x+v.w+scrollbar || y < v.y || y >= v.y+v.h {
		return false
	}
	top, height := thumb(v, target.scrollY)
	if y < top || y >= top+height {
		// A press beside the thumb jumps it there first.
		target.scrollY = (y - v.y - height/2) * (v.contentH - v.h) / max(v.h-height, 1)
		m.clampScroll(target)
		top, _ = thumb(v, target.scrollY)
	}
	m.dragging, m.grabY = target.id, y-top
	return true
}

// dragScrollbar moves a dragged thumb to follow the pointer.
func (m *model) dragScrollbar(target *widget, y int) {
	v := m.viewportOf(target)
	_, height := thumb(v, target.scrollY)
	travel := max(v.h-height, 1)
	target.scrollY = (y - m.grabY - v.y) * (v.contentH - v.h) / travel
	m.clampScroll(target)
	m.dirty = true
}

// rowAt is the ListBox item or TableView row at a logical y, or -1.
func (m *model) rowAt(target *widget, y int) int {
	v := m.viewportOf(target)
	if y < v.y || y >= v.y+v.h {
		return -1
	}
	row := (y - v.y + target.scrollY) / itemHeight
	if row >= target.count() {
		return -1
	}
	return row
}

// wheel scrolls the open Select list or the scrolling widget under a point,
// by dx and dy wheel notches. It reports whether anything scrolled.
func (m *model) wheel(x, y int, dx, dy float64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.layout()
	const notch = 3 * itemHeight
	if m.popup != 0 {
		if !m.inPopup(x, y) {
			return false
		}
		target := m.widgets[m.popup]
		target.listScroll -= int(dy * notch)
		m.clampPopup(target)
		m.dirty = true
		return true
	}
	id := m.hit(x, y)
	if id == 0 || !m.widgets[id].scrolling() || m.widgets[id].disabled {
		return false
	}
	target := m.widgets[id]
	target.scrollY -= int(dy * notch)
	if target.kind == kindTable {
		target.scrollX -= int(dx * notch)
	}
	m.clampScroll(target)
	m.dirty = true
	return true
}

// ---- the Select's open list ------------------------------------------------

// popupBox is where the open list is drawn; it is placed by placePopup.
type popupBox struct{ x, y, w, h int }

// The open list is kept in model.list.

// placePopup puts the open list below its Select, or above it when there is
// no room below.
func (m *model) placePopup(target *widget) {
	visible := min(max(len(target.items), 1), selectVisible)
	height := visible*itemHeight + 2
	y := target.y + target.h
	if y+height > m.height && target.y-height >= 0 {
		y = target.y - height
	}
	m.list = popupBox{x: target.x, y: y, w: target.w, h: height}
	m.clampPopup(target)
}

func (m *model) clampPopup(target *widget) {
	visible := min(max(len(target.items), 1), selectVisible)
	target.listScroll = min(max(target.listScroll, 0), max(len(target.items)-visible, 0)*itemHeight)
}

func (m *model) inPopup(x, y int) bool {
	return x >= m.list.x && y >= m.list.y && x < m.list.x+m.list.w && y < m.list.y+m.list.h
}

// popupChoiceAt is the choice at a logical y in the open list, or -1.
func (m *model) popupChoiceAt(y int) int {
	target := m.widgets[m.popup]
	choice := (y - m.list.y - 1 + target.listScroll) / itemHeight
	if y < m.list.y+1 || choice >= len(target.items) {
		return -1
	}
	return choice
}

func (m *model) openPopup(target *widget) {
	if target.disabled {
		return
	}
	m.popup = target.id
	target.highlight = target.selected
	target.listScroll = 0
	m.placePopup(target)
	m.revealPopup(target)
	m.dirty = true
}

func (m *model) closePopup() {
	if target := m.widgets[m.popup]; target != nil {
		target.highlight = -1
	}
	m.popup = 0
	m.dirty = true
}

// revealPopup scrolls the open list to its highlighted choice.
func (m *model) revealPopup(target *widget) {
	if target.highlight < 0 {
		return
	}
	visible := min(max(len(target.items), 1), selectVisible)
	top := target.highlight * itemHeight
	if top < target.listScroll {
		target.listScroll = top
	}
	if top+itemHeight > target.listScroll+visible*itemHeight {
		target.listScroll = top + itemHeight - visible*itemHeight
	}
	m.clampPopup(target)
}

// ---- TextArea --------------------------------------------------------------

// areaLines splits a TextArea's text into lines of runes.
func areaLines(text []rune) [][]rune {
	lines := [][]rune{{}}
	for _, character := range text {
		if character == '\n' {
			lines = append(lines, []rune{})
			continue
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], character)
	}
	return lines
}

// areaPosition is the caret's line and column, in runes.
func areaPosition(text []rune, caret int) (int, int) {
	line, column := 0, 0
	for _, character := range text[:caret] {
		if character == '\n' {
			line, column = line+1, 0
		} else {
			column++
		}
	}
	return line, column
}

// areaOffset is the rune offset of a line and column, the column kept
// inside the line.
func areaOffset(text []rune, line, column int) int {
	lines := areaLines(text)
	line = min(max(line, 0), len(lines)-1)
	offset := 0
	for index := 0; index < line; index++ {
		offset += len(lines[index]) + 1
	}
	return offset + min(max(column, 0), len(lines[line]))
}

// editArea applies an editing or caret key to a TextArea.
func (m *model) editArea(target *widget, key string) {
	line, column := areaPosition(target.text, target.caret)
	switch key {
	case "Enter":
		if len(target.text) < maxAreaRunes {
			target.text = append(target.text[:target.caret], append([]rune{'\n'}, target.text[target.caret:]...)...)
			target.caret++
			m.noteChange(target.id)
		}
	case "Backspace":
		if target.caret > 0 {
			target.text = append(target.text[:target.caret-1], target.text[target.caret:]...)
			target.caret--
			m.noteChange(target.id)
		}
	case "Delete":
		if target.caret < len(target.text) {
			target.text = append(target.text[:target.caret], target.text[target.caret+1:]...)
			m.noteChange(target.id)
		}
	case "ArrowLeft":
		target.caret = max(target.caret-1, 0)
	case "ArrowRight":
		target.caret = min(target.caret+1, len(target.text))
	case "ArrowUp":
		target.caret = areaOffset(target.text, line-1, column)
		if line == 0 {
			target.caret = 0
		}
	case "ArrowDown":
		lines := areaLines(target.text)
		target.caret = areaOffset(target.text, line+1, column)
		if line == len(lines)-1 {
			target.caret = len(target.text)
		}
	case "Home":
		target.caret = areaOffset(target.text, line, 0)
	case "End":
		target.caret = areaOffset(target.text, line, 1<<30)
	case "PageUp":
		target.caret = areaOffset(target.text, line-pageItems, column)
	case "PageDown":
		target.caret = areaOffset(target.text, line+pageItems, column)
	default:
		return
	}
	m.revealCaret(target)
	m.dirty = true
}

// revealCaret scrolls a TextArea vertically so its caret line is visible;
// the horizontal scroll follows the caret when the TextArea is drawn.
func (m *model) revealCaret(target *widget) {
	if target.kind != kindTextArea {
		return
	}
	line, _ := areaPosition(target.text, target.caret)
	v := m.viewportOf(target)
	top := line * lineHeight
	if top < target.scrollY {
		target.scrollY = top
	}
	if top+lineHeight > target.scrollY+v.h {
		target.scrollY = top + lineHeight - v.h
	}
	m.clampScroll(target)
}

// placeAreaCaret moves a TextArea's caret to a clicked point.
func (m *model) placeAreaCaret(target *widget, x, y int) {
	v := m.viewportOf(target)
	lines := areaLines(target.text)
	line := min(max((y-v.y+target.scrollY)/lineHeight, 0), len(lines)-1)
	want := x - v.x + int(float64(target.scroll)/max(m.scale, 1))
	column := 0
	for column < len(lines[line]) {
		next := m.measure(string(lines[line][:column+1]))
		if next > want {
			previous := m.measure(string(lines[line][:column]))
			if want-previous > next-want {
				column++
			}
			break
		}
		column++
	}
	target.caret = areaOffset(target.text, line, column)
}
