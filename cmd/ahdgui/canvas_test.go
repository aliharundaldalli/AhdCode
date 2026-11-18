package main

import (
	"image/color"
	"testing"
)

func TestCanvasDrawAndPointer(t *testing.T) {
	m := newModel("t", 240, 200, fixedMeasure)
	root := mustSpec(t, m, 0, spec{kind: kindColumn, spacing: 0, padding: 0, selected: -1})
	board := mustSpec(t, m, root, spec{kind: kindCanvas, fitW: 100, fitH: 80, selected: -1})
	layout(t, m)
	box := widgetBox(m, board)
	gx, gy, ok := m.canvasCoords(board, box[0]+box[2]/2, box[1]+box[3]/2, false)
	if !ok || gx != 0 || gy != 0 {
		t.Fatalf("center (%v, %v) ok %v box %v", gx, gy, ok, box)
	}
	left, top, ok := m.canvasCoords(board, box[0], box[1], false)
	if !ok || left != float64(-box[2]/2) || top != float64(box[3]/2) {
		t.Fatalf("corner (%v, %v) box %v", left, top, box)
	}
	if err := m.setEnabled(board, false); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := m.canvasCoords(board, box[0]+1, box[1]+1, false); ok {
		t.Fatal("a disabled Canvas reported a pointer")
	}
	if err := m.setEnabled(board, true); err != nil || m.setVisible(board, false) != nil {
		t.Fatal("restore")
	}
	if _, _, ok := m.canvasCoords(board, box[0]+1, box[1]+1, false); ok {
		t.Fatal("a hidden Canvas reported a pointer")
	}
	if err := m.setVisible(board, true); err != nil {
		t.Fatal(err)
	}
	if err := m.drawCanvas(request{Widget: board, Kind: "clear", Color: "#ffffffff"}); err != nil {
		t.Fatal(err)
	}
	if err := m.drawCanvas(request{Widget: board, Kind: "circle", X: 0, Y: 0, Radius: 8, StrokeW: 1, Color: "#0000ffff", Fill: "#0000ffff"}); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	img := render(frame{model: m, scale: 1})
	center := widgetBox(m, board)
	pixel := img.RGBAAt(center[0]+center[2]/2, center[1]+center[3]/2)
	if pixel.R > 20 || pixel.B < 200 {
		t.Fatalf("center pixel %#v", pixel)
	}
	_ = color.White
}

func TestCanvasInGridScrollAndTabs(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	grid := mustSpec(t, m, 0, spec{kind: kindGrid, gridColumns: 2, spacing: 0, rowGap: 0, selected: -1})
	left := mustSpec(t, m, grid, spec{kind: kindCanvas, fitW: 40, fitH: 30, selected: -1})
	right := mustSpec(t, m, grid, spec{kind: kindCanvas, fitW: 40, fitH: 30, selected: -1})
	layout(t, m)
	if m.widgets[left].x == m.widgets[right].x || m.widgets[left].w == 0 {
		t.Fatalf("grid canvases %v %v", widgetBox(m, left), widgetBox(m, right))
	}
	box := widgetBox(m, right)
	if _, _, ok := m.canvasCoords(right, box[0]+box[2]/2, box[1]+box[3]/2, false); !ok {
		t.Fatal("grid canvas missed its center")
	}

	scrolled := newModel("t", 120, 50, fixedMeasure)
	body := mustSpec(t, scrolled, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	for i := 0; i < 4; i++ {
		mustAdd(t, scrolled, body, kindLabel, "spacer")
	}
	board := mustSpec(t, scrolled, body, spec{kind: kindCanvas, fitW: 60, fitH: 40, selected: -1})
	layout(t, scrolled)
	before := widgetBox(scrolled, board)
	if scrolled.hit(before[0]+2, before[1]+2) == board {
		t.Fatal("canvas below the viewport was hit")
	}
	scrolled.mu.Lock()
	scrolled.widgets[body].scrollY = before[1]
	scrolled.mu.Unlock()
	layout(t, scrolled)
	moved := widgetBox(scrolled, board)
	gx, gy, ok := scrolled.canvasCoords(board, moved[0]+moved[2]/2, moved[1]+moved[3]/2, false)
	if !ok || gx != 0 || gy != 0 {
		t.Fatalf("scrolled canvas center (%v, %v) box %v", gx, gy, moved)
	}

	tabsModel := newModel("t", 200, 160, fixedMeasure)
	root := mustSpec(t, tabsModel, 0, spec{kind: kindColumn, spacing: 0, padding: 0, selected: -1})
	tabs := mustAdd(t, tabsModel, root, kindTabs, "")
	first := mustSpec(t, tabsModel, tabs, spec{kind: kindPage, text: "One", selected: -1})
	second := mustSpec(t, tabsModel, tabs, spec{kind: kindPage, text: "Two", selected: -1})
	hidden := mustSpec(t, tabsModel, first, spec{kind: kindCanvas, fitW: 50, fitH: 40, selected: -1})
	shown := mustSpec(t, tabsModel, second, spec{kind: kindCanvas, fitW: 50, fitH: 40, selected: -1})
	layout(t, tabsModel)
	shownBox := widgetBox(tabsModel, shown)
	if _, _, ok := tabsModel.canvasCoords(shown, shownBox[0]+4, shownBox[1]+4, false); ok {
		t.Fatal("a canvas on an inactive page reported a pointer")
	}
	if err := tabsModel.selectIndex(tabs, 1); err != nil {
		t.Fatal(err)
	}
	layout(t, tabsModel)
	shownBox = widgetBox(tabsModel, shown)
	if _, _, ok := tabsModel.canvasCoords(shown, shownBox[0]+shownBox[2]/2, shownBox[1]+shownBox[3]/2, false); !ok {
		t.Fatal("the active page canvas missed its pointer")
	}
	if _, _, ok := tabsModel.canvasCoords(hidden, 4, 4, false); ok {
		t.Fatal("the inactive page canvas still reported a pointer")
	}
}
