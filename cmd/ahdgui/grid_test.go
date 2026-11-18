package main

import (
	"image/color"
	"testing"
)

func mustSpec(t *testing.T, m *model, parent int64, s spec) int64 {
	t.Helper()
	id, err := m.addSpec(parent, s)
	if err != nil {
		t.Fatalf("add %s: %v", s.kind, err)
	}
	return id
}

func layout(t *testing.T, m *model) {
	t.Helper()
	m.mu.Lock()
	m.layout()
	m.mu.Unlock()
}

func TestGridLayout(t *testing.T) {
	if _, err := newModel("t", 200, 200, fixedMeasure).addSpec(0, spec{kind: kindGrid, selected: -1}); err == nil {
		t.Fatal("a grid with no columns was accepted")
	}
	m := newModel("t", 400, 300, fixedMeasure)
	root := mustSpec(t, m, 0, spec{kind: kindGrid, gridColumns: 2, spacing: 8, rowGap: 4, selected: -1})
	a := mustAdd(t, m, root, kindLabel, "A")
	b := mustAdd(t, m, root, kindLabel, "BB")
	hidden := mustAdd(t, m, root, kindLabel, "H")
	c := mustAdd(t, m, root, kindLabel, "C")
	if err := m.setVisible(hidden, false); err != nil {
		t.Fatal(err)
	}
	wide := mustAdd(t, m, root, kindButton, "OK")
	if err := m.setBox("span", wide, 2, 1, "", ""); err != nil {
		t.Fatal(err)
	}
	tall := mustAdd(t, m, root, kindLabel, "T")
	if err := m.setBox("span", tall, 1, 2, "", ""); err != nil {
		t.Fatal(err)
	}
	side := mustAdd(t, m, root, kindLabel, "S")
	below := mustAdd(t, m, root, kindLabel, "Z")
	layout(t, m)

	// A and BB share row 0. H is hidden, so C moves into the next cell,
	// which is the start of row 1. The button spans that row's two columns
	// and therefore wraps to the following row.
	// The spanned button is wider than the two columns, so each column
	// takes half of the deficit (the first column takes the extra pixel).
	if m.widgets[a].x != 0 || m.widgets[a].y != 0 || m.widgets[a].w != 10 {
		t.Fatalf("A at %v", widgetBox(m, a))
	}
	if m.widgets[b].x != 31 || m.widgets[b].y != 0 || m.widgets[b].w != 20 {
		t.Fatalf("BB at %v", widgetBox(m, b))
	}
	if m.widgets[hidden].w != 0 || m.widgets[hidden].h != 0 {
		t.Fatal("a hidden label kept a cell")
	}
	if m.widgets[c].x != 0 || m.widgets[c].y != 36 {
		t.Fatalf("C did not close the hole: %v", widgetBox(m, c))
	}
	if m.widgets[wide].x != 0 || m.widgets[wide].y != 72 || m.widgets[wide].w != 64 {
		t.Fatalf("spanned button at %v", widgetBox(m, wide))
	}
	// T occupies column 0 of the next two rows. S skips that cell.
	if m.widgets[tall].x != 0 || m.widgets[tall].y != 108 || m.widgets[tall].h != rowHeight {
		t.Fatalf("row span at %v", widgetBox(m, tall))
	}
	if m.widgets[side].y != m.widgets[tall].y || m.widgets[side].x == m.widgets[tall].x {
		t.Fatalf("S should sit beside T: T %v S %v", widgetBox(m, tall), widgetBox(m, side))
	}
	if m.widgets[below].y <= m.widgets[side].y {
		t.Fatalf("Z should follow the row span: %v", widgetBox(m, below))
	}
	if m.hit(2, 2) != a || m.hit(32, 2) != b || m.hit(2, 40) != c {
		t.Fatalf("hits A=%d B=%d C=%d", m.hit(2, 2), m.hit(32, 2), m.hit(2, 40))
	}
	if m.hit(2, m.widgets[hidden].y) == hidden {
		t.Fatal("a hidden label was hit")
	}

	// Extra width is shared by the columns. The centered label moves inside
	// its cell and does not change the other column's start.
	if err := m.setBox("preferred", root, 84, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := m.setBox("align", b, 0, 0, "center", "start"); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[root].w != 84 {
		t.Fatalf("preferred width %d", m.widgets[root].w)
	}
	if m.widgets[a].x != 0 || m.widgets[b].x != 52 {
		t.Fatalf("centered cell A %v B %v", widgetBox(m, a), widgetBox(m, b))
	}

	nested := newModel("t", 300, 200, fixedMeasure)
	column := mustSpec(t, nested, 0, spec{kind: kindColumn, selected: -1})
	inner := mustSpec(t, nested, column, spec{kind: kindGrid, gridColumns: 2, spacing: 0, rowGap: 0, selected: -1})
	left := mustAdd(t, nested, inner, kindLabel, "L")
	right := mustAdd(t, nested, inner, kindLabel, "R")
	layout(t, nested)
	if nested.widgets[left].y != nested.widgets[right].y || nested.widgets[right].x != 10 {
		t.Fatalf("nested grid L %v R %v", widgetBox(nested, left), widgetBox(nested, right))
	}
}

func TestAlignmentAndSizing(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	wide := mustAdd(t, m, root, kindButton, "OK")
	label := mustAdd(t, m, root, kindLabel, "A")
	if err := m.setBox("align", label, 0, 0, "center", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[wide].x != 0 || m.widgets[wide].w != 64 {
		t.Fatalf("button %v", widgetBox(m, wide))
	}
	if m.widgets[label].x != (64-10)/2 || m.widgets[label].w != 10 {
		t.Fatalf("centered label %v", widgetBox(m, label))
	}
	if err := m.setBox("align", label, 0, 0, "end", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[label].x != 64-10 {
		t.Fatalf("end label %v", widgetBox(m, label))
	}
	if err := m.setBox("align", label, 0, 0, "stretch", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[label].x != 0 || m.widgets[label].w != 64 {
		t.Fatalf("stretched label %v", widgetBox(m, label))
	}

	row, _ := m.add(0, kindRow, "", "", false, 0, 0)
	// The column is already the root; a second root is rejected. Build the
	// row as its own window.
	rowModel := newModel("t", 400, 200, fixedMeasure)
	row = mustSpec(t, rowModel, 0, spec{kind: kindRow, selected: -1})
	title := mustAdd(t, rowModel, row, kindLabel, "Title")
	if err := rowModel.setLabelStyle(title, "title", nil); err != nil {
		t.Fatal(err)
	}
	button := mustAdd(t, rowModel, row, kindButton, "OK")
	layout(t, rowModel)
	titleH := rowModel.widgets[title].h
	if rowModel.widgets[button].y != (titleH-rowHeight)/2 {
		t.Fatalf("default row centers vertically: %v titleH %d", widgetBox(rowModel, button), titleH)
	}
	if err := rowModel.setBox("align", button, 0, 0, "start", "start"); err != nil {
		t.Fatal(err)
	}
	layout(t, rowModel)
	if rowModel.widgets[button].y != 0 {
		t.Fatalf("vertical start %v", widgetBox(rowModel, button))
	}
	if err := rowModel.setBox("align", button, 0, 0, "start", "end"); err != nil {
		t.Fatal(err)
	}
	layout(t, rowModel)
	if rowModel.widgets[button].y != titleH-rowHeight {
		t.Fatalf("vertical end %v", widgetBox(rowModel, button))
	}
	if err := rowModel.setBox("align", button, 0, 0, "start", "stretch"); err != nil {
		t.Fatal(err)
	}
	layout(t, rowModel)
	if rowModel.widgets[button].y != 0 || rowModel.widgets[button].h != titleH {
		t.Fatalf("vertical stretch %v", widgetBox(rowModel, button))
	}

	sized := newModel("t", 200, 100, fixedMeasure)
	column := mustSpec(t, sized, 0, spec{kind: kindColumn, selected: -1})
	text := mustAdd(t, sized, column, kindLabel, "A")
	if err := sized.setBox("min", text, 50, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, sized)
	if sized.widgets[text].w != 50 {
		t.Fatalf("minimum width %d", sized.widgets[text].w)
	}
	if err := sized.setBox("preferred", text, 80, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, sized)
	if sized.widgets[text].w != 80 {
		t.Fatalf("preferred width %d", sized.widgets[text].w)
	}
	if err := sized.setBox("preferred", text, 4, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, sized)
	if sized.widgets[text].w != 50 {
		t.Fatalf("a preferred width below the content shrank the label to %d", sized.widgets[text].w)
	}
	if err := sized.setBox("align", text, 0, 0, "middle", ""); err == nil {
		t.Fatal("middle alignment was accepted")
	}
	if err := sized.setBox("min", text, -1, 0, "", ""); err == nil {
		t.Fatal("a negative minimum was accepted")
	}
	if err := sized.setBox("span", text, 0, 1, "", ""); err == nil {
		t.Fatal("a zero span was accepted")
	}
}

func TestScrollContainer(t *testing.T) {
	m := newModel("t", 120, 80, fixedMeasure)
	_, _ = m.addMenu(0, "File")
	root := mustSpec(t, m, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	grid := mustSpec(t, m, root, spec{kind: kindGrid, gridColumns: 2, spacing: 0, rowGap: 0, selected: -1})
	var labels []int64
	for i := 0; i < 8; i++ {
		labels = append(labels, mustAdd(t, m, grid, kindLabel, "L"))
	}
	layout(t, m)
	if m.widgets[root].y != menuBarHeight {
		t.Fatalf("scroll root starts at %d, want below the menu bar", m.widgets[root].y)
	}
	if m.widgets[root].h != 80-menuBarHeight {
		t.Fatalf("scroll root height %d", m.widgets[root].h)
	}
	if m.widgets[labels[0]].y != menuBarHeight || m.hit(2, menuBarHeight+2) != labels[0] {
		t.Fatalf("first cell %v hit %d", widgetBox(m, labels[0]), m.hit(2, menuBarHeight+2))
	}

	// The viewport starts below the menu bar and is shorter than eight rows.
	viewTop := menuBarHeight
	m.wheel(4, viewTop+4, 0, -1)
	layout(t, m)
	if m.widgets[root].scrollY != 3*itemHeight {
		t.Fatalf("wheel scroll %d", m.widgets[root].scrollY)
	}
	moved := m.hit(2, viewTop+2)
	if moved == labels[0] || moved == 0 {
		t.Fatalf("after scrolling, the hit was %d; first label is %d", moved, labels[0])
	}
	if m.hit(2, viewTop-4) != 0 {
		t.Fatal("a click above the scroll viewport hit a widget")
	}
	// A point that used to land on the first label now lands on whichever
	// cell the scroll offset brought there, and nowhere else.
	if m.widgets[labels[0]].y+m.widgets[labels[0]].h <= viewTop && m.hit(2, m.widgets[labels[0]].y+2) != 0 {
		t.Fatal("a scrolled-out label still received the hit")
	}

	fieldModel := newModel("t", 100, 80, fixedMeasure)
	scroller := mustSpec(t, fieldModel, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	field := mustAdd(t, fieldModel, scroller, kindTextInput, "")
	if err := fieldModel.setText(field, "abcd"); err != nil {
		t.Fatal(err)
	}
	fieldModel.mu.Lock()
	fieldModel.widgets[scroller].scrollX = 20
	fieldModel.mu.Unlock()
	fieldModel.press(13, 10)
	fieldModel.mu.Lock()
	got := fieldModel.widgets[field].caret
	focused := fieldModel.focus
	fieldModel.mu.Unlock()
	if focused != field || got != 2 {
		t.Fatalf("caret after a scrolled click: focus %d caret %d", focused, got)
	}

	// Tab reveals a button that starts below the viewport.
	buttons := newModel("t", 80, 80, fixedMeasure)
	body := mustSpec(t, buttons, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	var keys []int64
	for i := 0; i < 5; i++ {
		keys = append(keys, mustAdd(t, buttons, body, kindButton, "B"))
	}
	if err := buttons.setEnabled(keys[1], false); err != nil {
		t.Fatal(err)
	}
	buttons.press(4, 4)
	if buttons.focus != keys[0] {
		t.Fatalf("focus %d, want the first button", buttons.focus)
	}
	buttons.focusWidget(keys[len(keys)-1])
	layout(t, buttons)
	last := buttons.widgets[keys[len(keys)-1]]
	vx, vy, _, vh, _, _ := buttons.scrollMetrics(buttons.widgets[body])
	if last.y < vy || last.y+last.h > vy+vh {
		t.Fatalf("focused button %v outside viewport y %d h %d", widgetBox(buttons, keys[len(keys)-1]), vy, vh)
	}
	buttons.press(4, 4)
	if buttons.focus == keys[1] {
		t.Fatal("a disabled button took the focus")
	}
	_ = vx

	// Content wider than the window scrolls horizontally, and the click
	// follows the offset.
	wide := newModel("t", 80, 100, fixedMeasure)
	band := mustSpec(t, wide, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	line := mustSpec(t, wide, band, spec{kind: kindRow, spacing: 0, selected: -1})
	left := mustAdd(t, wide, line, kindLabel, "A")
	right := mustAdd(t, wide, line, kindLabel, "B")
	if err := wide.setBox("min", left, 100, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := wide.setBox("min", right, 100, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, wide)
	if wide.hit(40, 10) != left {
		t.Fatalf("before horizontal scroll hit %d", wide.hit(40, 10))
	}
	wide.wheel(10, 10, -1, 0)
	layout(t, wide)
	if wide.widgets[band].scrollX == 0 {
		t.Fatal("horizontal wheel did not scroll")
	}
	if wide.hit(40, 10) != right {
		t.Fatalf("after horizontal scroll hit %d, want %d", wide.hit(40, 10), right)
	}

	// A label scrolled into the padding is not drawn there.
	clip := newModel("t", 120, 100, fixedMeasure)
	pane := mustSpec(t, clip, 0, spec{kind: kindScroll, spacing: 0, padding: 20, selected: -1})
	for i := 0; i < 4; i++ {
		id := mustAdd(t, clip, pane, kindLabel, "A")
		if i == 0 {
			if err := clip.setColor(id, partBackground, color.NRGBA{R: 200, G: 0, B: 0, A: 255}); err != nil {
				t.Fatal(err)
			}
		}
	}
	clip.mu.Lock()
	clip.widgets[pane].scrollY = 20
	clip.mu.Unlock()
	img := render(frame{model: clip, scale: 1})
	if img.RGBAAt(22, 8) != colorBackground {
		t.Fatalf("pixel in the padding was %#v", img.RGBAAt(22, 8))
	}
	if img.RGBAAt(22, 24) != (color.RGBA{R: 200, G: 0, B: 0, A: 255}) {
		t.Fatalf("pixel in the viewport was %#v", img.RGBAAt(22, 24))
	}

	// Dragging the vertical bar moves the same scroll offset the wheel uses.
	before := m.widgets[root].scrollY
	m.press(112, 40)
	m.drag(112, 70)
	m.release(112, 70)
	if m.widgets[root].scrollY == before {
		t.Fatal("dragging the scroll bar did not scroll")
	}

	// Growing the window shows content that had been scrolled away and
	// clamps the offset.
	m.resize(120, 400)
	layout(t, m)
	if m.widgets[root].scrollY != 0 || m.widgets[labels[0]].y != menuBarHeight {
		t.Fatalf("after resize scroll %d first %v", m.widgets[root].scrollY, widgetBox(m, labels[0]))
	}
}

func widgetBox(m *model, id int64) [4]int {
	w := m.widgets[id]
	return [4]int{w.x, w.y, w.w, w.h}
}
