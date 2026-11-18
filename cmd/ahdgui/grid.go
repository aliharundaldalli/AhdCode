package main

import (
	"fmt"
	"image"
)

// Grid, alignment, and scroll containers extend the Column and Row layout.
// Hidden children are skipped, so a later child moves into the hole. The
// caller holds m.mu.

func validateGrid(s spec) error {
	if s.kind != kindGrid {
		return nil
	}
	if s.gridColumns < 1 || s.gridColumns > maxGridColumns {
		return fmt.Errorf("a grid needs between 1 and %d columns", maxGridColumns)
	}
	if s.spacing < 0 || s.rowGap < 0 || s.spacing > maxSpacing || s.rowGap > maxSpacing {
		return fmt.Errorf("column and row gaps must be between 0 and %d", maxSpacing)
	}
	return nil
}

func validAlign(value string) bool {
	switch value {
	case "start", "center", "end", "stretch":
		return true
	default:
		return false
	}
}

// setBox applies a minimum width, a preferred width, an alignment, or a span.
func (m *model) setBox(kind string, id int64, width, height int, horizontal, vertical string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	switch kind {
	case "min":
		if width < 0 || width > maxDimension {
			return fmt.Errorf("minimum width must be between 0 and %d", maxDimension)
		}
		target.minW = width
	case "preferred":
		if width < 0 || width > maxDimension {
			return fmt.Errorf("preferred width must be between 0 and %d", maxDimension)
		}
		target.prefW = width
	case "align":
		if !validAlign(horizontal) || (vertical != "" && !validAlign(vertical)) {
			return fmt.Errorf(`alignment must be "start", "center", "end", or "stretch"`)
		}
		target.hAlign = horizontal
		target.vAlign = vertical
	case "span":
		if width < 1 || width > maxGridColumns || height < 1 || height > maxGridColumns {
			return fmt.Errorf("a span must be between 1 and %d", maxGridColumns)
		}
		target.spanC = width
		target.spanR = height
	case "wrap":
		if target.kind != kindLabel {
			return errWrongKind
		}
		target.wrap = width != 0
	case "fit":
		if target.kind != kindImage {
			return errWrongKind
		}
		if width < 0 || height < 0 || width > maxDimension || height > maxDimension {
			return badImageSize()
		}
		target.fitW, target.fitH = width, height
		target.scaled = nil
	default:
		return fmt.Errorf("unknown layout request")
	}
	m.dirty = true
	return nil
}

// finishSize raises a widget's natural width to its preferred or minimum
// width. A preferred width below the content does nothing, and nothing
// shrinks below the size its content measured.
func (m *model) finishSize(target *widget) {
	if target.prefW > target.natW {
		target.natW = target.prefW
	}
	if target.minW > target.natW {
		target.natW = target.minW
	}
}

// alignAxis places size inside a slot. Stretch grows up to the slot.
// A widget larger than the slot stays at the start and keeps its size.
func alignAxis(align string, origin, slot, size int) (int, int) {
	if slot > size {
		switch align {
		case "center":
			return origin + (slot-size)/2, size
		case "end":
			return origin + slot - size, size
		case "stretch":
			return origin, slot
		}
	}
	return origin, size
}

type gridCell struct {
	id           int64
	col, row     int
	spanC, spanR int
}

// gridFlow measures the visible children and packs them left to right.
// A hidden child leaves no cell. A span that does not fit wraps, and a cell
// already taken by a row span is skipped.
func (m *model) gridFlow(target *widget) ([]gridCell, int, int) {
	columns := target.gridColumns
	if columns < 1 {
		columns = 1
	}
	occupied := map[[2]int]bool{}
	var cells []gridCell
	col, row := 0, 0
	for _, child := range target.children {
		placed := m.widgets[child]
		m.measureWidget(child)
		if placed.hidden {
			continue
		}
		if placed.grows {
			target.grows = true
		}
		spanC, spanR := placed.spanC, placed.spanR
		if spanC < 1 {
			spanC = 1
		}
		if spanR < 1 {
			spanR = 1
		}
		if spanC > columns {
			spanC = columns
		}
		for {
			if col+spanC > columns {
				col, row = 0, row+1
			}
			fits := true
			for r := 0; r < spanR && fits; r++ {
				for c := 0; c < spanC; c++ {
					if occupied[[2]int{col + c, row + r}] {
						fits = false
						break
					}
				}
			}
			if fits {
				break
			}
			col++
			if col >= columns {
				col, row = 0, row+1
			}
		}
		cells = append(cells, gridCell{id: child, col: col, row: row, spanC: spanC, spanR: spanR})
		for r := 0; r < spanR; r++ {
			for c := 0; c < spanC; c++ {
				occupied[[2]int{col + c, row + r}] = true
			}
		}
		col += spanC
		if col >= columns {
			col, row = 0, row+1
		}
	}
	rows := 0
	for _, cell := range cells {
		if cell.row+cell.spanR > rows {
			rows = cell.row + cell.spanR
		}
	}
	return cells, columns, rows
}

func trackSpan(tracks []int, start, span, gap int) int {
	total := 0
	for i := 0; i < span; i++ {
		if i > 0 {
			total += gap
		}
		total += tracks[start+i]
	}
	return total
}

func shareDeficit(tracks []int, start, span, deficit int) {
	if span < 1 || deficit <= 0 {
		return
	}
	share, extra := deficit/span, deficit%span
	for i := 0; i < span; i++ {
		tracks[start+i] += share
		if i < extra {
			tracks[start+i]++
		}
	}
}

// gridTracks sizes each column and row from the children that span one
// track, then lets a spanning child raise those tracks until it fits.
func (m *model) gridTracks(target *widget, cells []gridCell, columns, rows int) ([]int, []int) {
	colW := make([]int, columns)
	rowH := make([]int, rows)
	for _, cell := range cells {
		child := m.widgets[cell.id]
		if cell.spanC == 1 {
			colW[cell.col] = max(colW[cell.col], child.natW)
		}
		if cell.spanR == 1 {
			rowH[cell.row] = max(rowH[cell.row], child.natH)
		}
	}
	for _, cell := range cells {
		child := m.widgets[cell.id]
		if cell.spanC > 1 {
			have := trackSpan(colW, cell.col, cell.spanC, target.spacing)
			shareDeficit(colW, cell.col, cell.spanC, child.natW-have)
		}
		if cell.spanR > 1 {
			have := trackSpan(rowH, cell.row, cell.spanR, target.rowGap)
			shareDeficit(rowH, cell.row, cell.spanR, child.natH-have)
		}
	}
	return colW, rowH
}

func (m *model) measureGrid(target *widget) {
	target.grows = false
	cells, columns, rows := m.gridFlow(target)
	colW, rowH := m.gridTracks(target, cells, columns, rows)
	target.natW = trackSpan(colW, 0, columns, target.spacing) + 2*target.padding
	target.natH = trackSpan(rowH, 0, rows, target.rowGap) + 2*target.padding
}

func (m *model) placeGrid(target *widget) {
	cells, columns, rows := m.gridFlow(target)
	colW, rowH := m.gridTracks(target, cells, columns, rows)
	innerW := target.w - 2*target.padding
	innerH := target.h - 2*target.padding
	if columns > 0 && target.natW > 0 && (target.w < target.natW || (m.wrapOffer > 0 && m.wrapOffer < target.natW)) {
		colW = m.gridColumnsOffered(target, cells, columns, m.gridOffer(target, innerW))
	}
	extraW := innerW - trackSpan(colW, 0, columns, target.spacing)
	if columns > 0 && extraW > 0 {
		share, extra := extraW/columns, extraW%columns
		for i := range colW {
			colW[i] += share
			if i < extra {
				colW[i]++
			}
		}
	}
	growRow := make([]bool, rows)
	growing := 0
	for _, cell := range cells {
		if !m.widgets[cell.id].grows {
			continue
		}
		for r := 0; r < cell.spanR && cell.row+r < rows; r++ {
			if !growRow[cell.row+r] {
				growRow[cell.row+r] = true
				growing++
			}
		}
	}
	if columns > 0 && target.natW > 0 && (target.w < target.natW || (m.wrapOffer > 0 && m.wrapOffer < target.natW)) {
		rowH = m.gridRowHeights(target, cells, rows, colW)
	}
	extraH := innerH - trackSpan(rowH, 0, rows, target.rowGap)
	if growing > 0 && extraH > 0 {
		share, extra := extraH/growing, extraH%growing
		given := 0
		for r := 0; r < rows; r++ {
			if !growRow[r] {
				continue
			}
			rowH[r] += share
			if given < extra {
				rowH[r]++
				given++
			}
		}
	}
	colX := make([]int, columns)
	rowY := make([]int, rows)
	cursor := target.x + target.padding
	for i := 0; i < columns; i++ {
		colX[i] = cursor
		cursor += colW[i]
		if i+1 < columns {
			cursor += target.spacing
		}
	}
	cursor = target.y + target.padding
	for i := 0; i < rows; i++ {
		rowY[i] = cursor
		cursor += rowH[i]
		if i+1 < rows {
			cursor += target.rowGap
		}
	}
	used := map[int64]bool{}
	for _, cell := range cells {
		used[cell.id] = true
		child := m.widgets[cell.id]
		slotW := trackSpan(colW, cell.col, cell.spanC, target.spacing)
		slotH := trackSpan(rowH, cell.row, cell.spanR, target.rowGap)
		hAlign, vAlign := child.hAlign, child.vAlign
		if hAlign == "" {
			hAlign = "start"
		}
		if vAlign == "" {
			vAlign = "start"
		}
		if child.grows {
			hAlign, vAlign = "stretch", "stretch"
		}
		sizeW, sizeH := child.natW, child.natH
		if child.kind == kindLabel && child.wrap {
			sizeW, sizeH = m.labelBox(child, slotW)
		}
		x, w := alignAxis(hAlign, colX[cell.col], slotW, sizeW)
		y, h := alignAxis(vAlign, rowY[cell.row], slotH, sizeH)
		m.place(cell.id, x, y, w, h)
	}
	for _, child := range target.children {
		if !used[child] {
			m.widgets[child].w, m.widgets[child].h = 0, 0
		}
	}
}

// gridOffer is the width wrapping labels in this grid may use. A scroll
// viewport narrower than the grid wins over the grid's own inner width.
func (m *model) gridOffer(target *widget, inner int) int {
	inner = max(inner, 0)
	if m.wrapOffer <= 0 {
		return inner
	}
	visible := m.wrapOffer
	if target.padding > 0 {
		visible = max(m.wrapOffer-2*target.padding, 0)
	}
	if inner == 0 || visible < inner {
		return visible
	}
	return inner
}

// gridColumnsOffered sizes columns as gridTracks does, except a wrapping
// Label asks for its height-for-width size at offer instead of its measured
// width. A button, field, or picture still keeps its natural width.
func (m *model) gridColumnsOffered(target *widget, cells []gridCell, columns, offer int) []int {
	colW := make([]int, columns)
	widths := make(map[int64]int, len(cells))
	for _, cell := range cells {
		child := m.widgets[cell.id]
		width := child.natW
		if child.kind == kindLabel && child.wrap {
			width, _ = m.labelBox(child, offer)
		}
		widths[cell.id] = width
		if cell.spanC == 1 {
			colW[cell.col] = max(colW[cell.col], width)
		}
	}
	for _, cell := range cells {
		if cell.spanC > 1 {
			have := trackSpan(colW, cell.col, cell.spanC, target.spacing)
			shareDeficit(colW, cell.col, cell.spanC, widths[cell.id]-have)
		}
	}
	return colW
}

// gridRowHeights sizes rows from the final column widths. A wrapping Label
// uses the height of the text at its real slot, so a narrower slot grows
// the row before the next row is placed.
func (m *model) gridRowHeights(target *widget, cells []gridCell, rows int, colW []int) []int {
	rowH := make([]int, rows)
	heights := make(map[int64]int, len(cells))
	for _, cell := range cells {
		child := m.widgets[cell.id]
		height := child.natH
		if child.kind == kindLabel && child.wrap {
			slot := trackSpan(colW, cell.col, cell.spanC, target.spacing)
			_, height = m.labelBox(child, slot)
		}
		heights[cell.id] = height
		if cell.spanR == 1 {
			rowH[cell.row] = max(rowH[cell.row], height)
		}
	}
	for _, cell := range cells {
		if cell.spanR > 1 {
			have := trackSpan(rowH, cell.row, cell.spanR, target.rowGap)
			shareDeficit(rowH, cell.row, cell.spanR, heights[cell.id]-have)
		}
	}
	return rowH
}

// flowStack is the content size of a scroll stack when its children are
// offered a viewport width.
func (m *model) flowStack(target *widget, offer int) (int, int) {
	width, height, seen := 0, 0, 0
	for _, child := range target.children {
		placed := m.widgets[child]
		if placed.hidden {
			continue
		}
		childW, childH := m.childSize(placed, offer)
		if seen > 0 {
			height += target.spacing
		}
		seen++
		width = max(width, childW)
		height += childH
	}
	return width, height
}

// flowGrid is a grid's size when it is offered a width. An offer that is
// at least the measured width keeps that measured size.
func (m *model) flowGrid(target *widget, offer int) (int, int) {
	if offer <= 0 || target.natW == 0 || offer >= target.natW {
		return target.natW, target.natH
	}
	cells, columns, rows := m.gridFlow(target)
	if columns == 0 {
		return target.natW, target.natH
	}
	inner := max(offer-2*target.padding, 0)
	colW := m.gridColumnsOffered(target, cells, columns, inner)
	rowH := m.gridRowHeights(target, cells, rows, colW)
	width := trackSpan(colW, 0, columns, target.spacing) + 2*target.padding
	height := trackSpan(rowH, 0, rows, target.rowGap) + 2*target.padding
	return width, height
}

// flowTabs is a tab strip's size at an offered width. The selected page
// reflows; the title row does not shrink below its titles.
func (m *model) flowTabs(target *widget, offer int) (int, int) {
	if offer <= 0 || target.natW == 0 || offer >= target.natW {
		return target.natW, target.natH
	}
	width := max(offer, m.tabTitleWidth(target))
	if width > target.natW {
		width = target.natW
	}
	pages := m.pages(target)
	pageH := 0
	if target.selected >= 0 && target.selected < len(pages) && !pages[target.selected].hidden {
		_, pageH = m.flowBox(pages[target.selected].id, width)
	}
	height := rowHeight + pageH
	if height < target.natH {
		height = target.natH
	}
	return width, height
}

func (m *model) tabTitleWidth(target *widget) int {
	titles, shown := 0, 0
	for _, page := range m.pages(target) {
		if page.hidden {
			continue
		}
		if shown > 0 {
			titles += 4
		}
		shown++
		titles += m.measure(string(page.text)) + 2*buttonPadding
	}
	return titles
}

// measureScroll stacks its children at their natural size. The container is
// as wide as that stack. Its natural height is the stack, or scrollViewport
// when the stack is taller; it then grows so a Column can give it more room.
// Children inside a scroll container do not grow.
func (m *model) measureScroll(target *widget) {
	contentW, contentH := m.measureStack(target)
	target.natW = contentW + 2*target.padding
	viewH := contentH
	target.grows = false
	if contentH > scrollViewport {
		viewH = scrollViewport
		target.grows = true
	}
	target.natH = viewH + 2*target.padding
}

// measureStack measures a vertical stack and returns its content size.
// Hidden children leave no gap. A child's own growth is not propagated:
// the scroll container keeps every child at its natural size.
func (m *model) measureStack(target *widget) (int, int) {
	width, height, seen := 0, 0, 0
	for _, child := range target.children {
		childW, childH := m.measureWidget(child)
		if m.widgets[child].hidden {
			continue
		}
		if seen > 0 {
			height += target.spacing
		}
		seen++
		width = max(width, childW)
		height += childH
	}
	return width, height
}

func (m *model) stackSize(target *widget) (int, int) {
	width, height, seen := 0, 0, 0
	for _, child := range target.children {
		placed := m.widgets[child]
		if placed.hidden {
			continue
		}
		if seen > 0 {
			height += target.spacing
		}
		seen++
		cw, ch := placed.natW, placed.natH
		if placed.w > 0 {
			cw = placed.w
		}
		if placed.h > 0 {
			ch = placed.h
		}
		width = max(width, cw)
		height += ch
	}
	return width, height
}

func (m *model) placeStack(target *widget, x, y int) {
	cursor, seen := y, 0
	for _, child := range target.children {
		placed := m.widgets[child]
		if placed.hidden {
			placed.w, placed.h = 0, 0
			continue
		}
		if seen > 0 {
			cursor += target.spacing
		}
		seen++
		childW, childH := placed.natW, placed.natH
		if m.wrapOffer > 0 {
			childW, childH = m.childSize(placed, m.wrapOffer)
		}
		m.place(child, x, cursor, childW, childH)
		cursor += childH
	}
}

// scrollMetrics is the content viewport of a scroll container, with the
// scroll bars taken out of the side they occupy. It matches placeScroll.
func (m *model) scrollMetrics(target *widget) (x, y, w, h, contentW, contentH int) {
	contentW, contentH = m.stackSize(target)
	x = target.x + target.padding
	y = target.y + target.padding
	w = max(target.w-2*target.padding, 0)
	h = max(target.h-2*target.padding, 0)
	vertical := contentH > h
	if vertical {
		w = max(w-scrollbar, 0)
	}
	if contentW > w {
		h = max(h-scrollbar, 0)
		if !vertical && contentH > h {
			w = max(target.w-2*target.padding-scrollbar, 0)
		}
	}
	return x, y, w, h, contentW, contentH
}

func (m *model) placeScroll(target *widget) {
	previous := m.wrapOffer
	vw := max(target.w-2*target.padding, 0)
	vh := max(target.h-2*target.padding, 0)
	contentW, contentH := 0, 0
	for range 3 {
		m.wrapOffer = vw
		contentW, contentH = m.flowStack(target, vw)
		vw, vh = m.fittedViewport(target, contentW, contentH)
	}
	target.scrollX = min(max(target.scrollX, 0), max(contentW-vw, 0))
	target.scrollY = min(max(target.scrollY, 0), max(contentH-vh, 0))
	m.wrapOffer = vw
	m.placeStack(target, target.x+target.padding-target.scrollX, target.y+target.padding-target.scrollY)
	m.wrapOffer = previous
}

// fittedViewport is the content viewport after scroll bars take their side.
func (m *model) fittedViewport(target *widget, contentW, contentH int) (int, int) {
	w := max(target.w-2*target.padding, 0)
	h := max(target.h-2*target.padding, 0)
	vertical := contentH > h
	if vertical {
		w = max(w-scrollbar, 0)
	}
	if contentW > w {
		h = max(h-scrollbar, 0)
		if !vertical && contentH > h {
			w = max(target.w-2*target.padding-scrollbar, 0)
		}
	}
	return w, h
}

func (m *model) clampScrollContainer(target *widget) {
	_, _, vw, vh, contentW, contentH := m.scrollMetrics(target)
	target.scrollX = min(max(target.scrollX, 0), max(contentW-vw, 0))
	target.scrollY = min(max(target.scrollY, 0), max(contentH-vh, 0))
}

// insideScrolls reports whether a point is inside every scroll viewport
// that contains the widget. A point on a scroll bar, or on a child that has
// been scrolled outside the viewport, is not a hit.
func (m *model) insideScrolls(id int64, x, y int) bool {
	for parent := m.widgets[id].parent; parent != 0; parent = m.widgets[parent].parent {
		box := m.widgets[parent]
		if box.kind != kindScroll {
			continue
		}
		vx, vy, vw, vh, _, _ := m.scrollMetrics(box)
		if x < vx || y < vy || x >= vx+vw || y >= vy+vh {
			return false
		}
	}
	return true
}

// scrollUnder is the innermost scroll container whose box contains the point.
func (m *model) scrollUnder(x, y int) *widget {
	var found *widget
	for _, id := range m.order {
		target := m.widgets[id]
		if target.kind != kindScroll || !m.shown(id) {
			continue
		}
		if x >= target.x && y >= target.y && x < target.x+target.w && y < target.y+target.h {
			found = target
		}
	}
	return found
}

// revealWidget scrolls every scroll ancestor so the widget is inside the
// viewport. A widget taller or wider than the viewport is aligned to the
// start of that viewport. The caller holds m.mu.
func (m *model) revealWidget(id int64) {
	target := m.widgets[id]
	if target == nil {
		return
	}
	scrolls := false
	for parent := target.parent; parent != 0; parent = m.widgets[parent].parent {
		if m.widgets[parent].kind == kindScroll {
			scrolls = true
			break
		}
	}
	if !scrolls {
		return
	}
	m.layout()
	for parent := m.widgets[id].parent; parent != 0; parent = m.widgets[parent].parent {
		box := m.widgets[parent]
		if box.kind != kindScroll {
			continue
		}
		target = m.widgets[id]
		vx, vy, vw, vh, _, _ := m.scrollMetrics(box)
		if target.h >= vh {
			box.scrollY += target.y - vy
		} else if target.y < vy {
			box.scrollY -= vy - target.y
		} else if target.y+target.h > vy+vh {
			box.scrollY += target.y + target.h - (vy + vh)
		}
		if target.w >= vw {
			box.scrollX += target.x - vx
		} else if target.x < vx {
			box.scrollX -= vx - target.x
		} else if target.x+target.w > vx+vw {
			box.scrollX += target.x + target.w - (vx + vw)
		}
		m.clampScrollContainer(box)
		m.layout()
	}
}

// scrollClip is the device rectangle a widget may draw into, the
// intersection of its scroll ancestors. limited is false when nothing
// clips it. An empty rectangle means the widget is scrolled out of view.
func (m *model) scrollClip(id int64, scale float64) (image.Rectangle, bool) {
	var clip image.Rectangle
	limited := false
	for parent := m.widgets[id].parent; parent != 0; parent = m.widgets[parent].parent {
		box := m.widgets[parent]
		if box.kind != kindScroll {
			continue
		}
		vx, vy, vw, vh, _, _ := m.scrollMetrics(box)
		rect := scaled(scale, vx, vy, vw, vh)
		if !limited {
			clip = rect
			limited = true
		} else {
			clip = clip.Intersect(rect)
		}
	}
	return clip, limited
}

// pressScrollBar starts a drag when the point is on a scroll container's bar.
// A click on the track jumps the thumb there first. The caller holds m.mu.
func (m *model) pressScrollBar(x, y int) bool {
	target := m.scrollUnder(x, y)
	if target == nil {
		return false
	}
	vx, vy, vw, vh, contentW, contentH := m.scrollMetrics(target)
	if contentH > vh && x >= vx+vw && x < vx+vw+scrollbar && y >= vy && y < vy+vh {
		top, height := thumb(viewport{y: vy, h: vh, contentH: contentH}, target.scrollY)
		if y < top || y >= top+height {
			target.scrollY = (y - vy - height/2) * max(contentH-vh, 0) / max(vh-height, 1)
			m.clampScrollContainer(target)
			top, _ = thumb(viewport{y: vy, h: vh, contentH: contentH}, target.scrollY)
		}
		m.dragging, m.grabY, m.dragHorizontal = target.id, y-top, false
		m.dirty = true
		return true
	}
	if contentW > vw && y >= vy+vh && y < vy+vh+scrollbar && x >= vx && x < vx+vw {
		width := min(max(vw*vw/max(contentW, 1), 16), vw)
		left := scrollThumb(vw, width, contentW, target.scrollX)
		if x < vx+left || x >= vx+left+width {
			target.scrollX = (x - vx - width/2) * max(contentW-vw, 0) / max(vw-width, 1)
			m.clampScrollContainer(target)
			left = scrollThumb(vw, width, contentW, target.scrollX)
		}
		m.dragging, m.grabX, m.dragHorizontal = target.id, x-(vx+left), true
		m.dirty = true
		return true
	}
	return false
}

func scrollThumb(view, thumbWidth, content, scroll int) int {
	if span := content - view; span > 0 && view > thumbWidth {
		return (view - thumbWidth) * scroll / span
	}
	return 0
}

// dragScrollContainer follows the pointer with a scroll container's thumb.
func (m *model) dragScrollContainer(target *widget, x, y int) {
	vx, vy, vw, vh, contentW, contentH := m.scrollMetrics(target)
	if m.dragHorizontal {
		width := min(max(vw*vw/max(contentW, 1), 16), vw)
		target.scrollX = (x - m.grabX - vx) * max(contentW-vw, 0) / max(vw-width, 1)
	} else {
		_, height := thumb(viewport{y: vy, h: vh, contentH: contentH}, target.scrollY)
		target.scrollY = (y - m.grabY - vy) * max(contentH-vh, 0) / max(vh-height, 1)
	}
	m.clampScrollContainer(target)
	m.dirty = true
}

// drawScrollBars draws the bars of a scroll container. Children are clipped
// to the viewport, so they do not paint over the bars.
func (m *model) drawScrollBars(img *image.RGBA, f frame, w *widget) {
	vx, vy, vw, vh, contentW, contentH := m.scrollMetrics(w)
	s := f.scale
	if contentH > vh && vh > 0 {
		fill(img, scaled(s, vx+vw, vy, scrollbar, vh), colorButton)
		top, height := thumb(viewport{y: vy, h: vh, contentH: contentH}, w.scrollY)
		fill(img, scaled(s, vx+vw, top, scrollbar, height), colorBorder)
	}
	if contentW > vw && vw > 0 {
		fill(img, scaled(s, vx, vy+vh, vw, scrollbar), colorButton)
		width := min(max(vw*vw/max(contentW, 1), 16), vw)
		left := 0
		if span := contentW - vw; span > 0 && vw > width {
			left = (vw - width) * w.scrollX / span
		}
		fill(img, scaled(s, vx+left, vy+vh, width, scrollbar), colorBorder)
	}
}
