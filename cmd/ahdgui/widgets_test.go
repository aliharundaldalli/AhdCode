package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Tests of the v2.0 widgets: ListBox, Select, TextArea, PasswordInput,
// TableView, change events, scrolling, and resizable layout.

func addItems(t *testing.T, m *model, parent int64, kind string, items []string, selected int) int64 {
	t.Helper()
	id, err := m.addSpec(parent, spec{kind: kind, items: items, selected: selected})
	if err != nil {
		t.Fatalf("add %s: %v", kind, err)
	}
	return id
}

func tableRowsOf(count, columns int) [][]string {
	rows := make([][]string, count)
	for row := range rows {
		rows[row] = make([]string, columns)
		for column := range rows[row] {
			rows[row][column] = fmt.Sprintf("r%dc%d", row, column)
		}
	}
	return rows
}

func TestListBoxSelectionItemsAndScrolling(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	empty := addItems(t, m, root, kindListBox, nil, -1)
	if _, _, selected, _ := m.readAll(empty); selected != -1 {
		t.Fatal("an empty ListBox has a selection")
	}
	if err := m.selectIndex(empty, 0); err != errIndex {
		t.Fatalf("selecting in an empty ListBox: %v", err)
	}
	var items []string
	for index := range 50 {
		items = append(items, fmt.Sprintf("item %d", index))
	}
	list := addItems(t, m, root, kindListBox, items, -1)
	if err := m.selectIndex(list, 49); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.layout()
	target := m.widgets[list]
	v := m.viewportOf(target)
	m.mu.Unlock()
	// Selecting the last item scrolls it into view.
	if target.scrollY != 50*itemHeight-v.h || !v.vertical {
		t.Fatalf("scrollY %d, viewport %+v", target.scrollY, v)
	}
	if len(m.takeChanges()) != 0 {
		t.Fatal("a selection by the program counted as a user change")
	}
	// Replacing the items clears a selection that no longer exists.
	if err := m.setItems(list, items[:10]); err != nil {
		t.Fatal(err)
	}
	if _, _, selected, _ := m.readAll(list); selected != -1 {
		t.Fatalf("stale selection %d kept", selected)
	}
	_ = m.selectIndex(list, 3)
	_ = m.setItems(list, items[:5])
	if _, _, selected, _ := m.readAll(list); selected != 3 {
		t.Fatalf("a still valid selection was lost: %d", selected)
	}
	if err := m.selectIndex(list, -1); err != nil {
		t.Fatal(err)
	}
	// A click on the third row selects it as the user.
	m.mu.Lock()
	m.layout()
	rowY := m.widgets[list].y + 1 + 2*itemHeight + 3
	m.mu.Unlock()
	m.press(20, rowY)
	m.release(20, rowY)
	if _, _, selected, _ := m.readAll(list); selected != 2 {
		t.Fatalf("click selected %d", selected)
	}
	if changes := m.takeChanges(); len(changes) != 1 || changes[0] != list {
		t.Fatalf("changes %v", changes)
	}
	// The keyboard moves the selection of the focused ListBox.
	for _, key := range []string{"ArrowDown", "ArrowDown", "End", "Home", "PageDown"} {
		m.navigate(key)
	}
	if _, _, selected, _ := m.readAll(list); selected != 4 {
		t.Fatalf("keyboard selection %d", selected)
	}
	if err := m.setItems(list, []string{strings.Repeat("x", maxItemRunes+1)}); err == nil {
		t.Fatal("an overlong item was accepted")
	}
}

func TestSelectOpensChoosesAndCloses(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	choice := addItems(t, m, root, kindSelect, []string{"Cash", "Card", "Transfer"}, 1)
	if _, _, selected, _ := m.readAll(choice); selected != 1 {
		t.Fatal("initial selection")
	}
	if _, err := m.addSpec(root, spec{kind: kindSelect, items: []string{"a"}, selected: 1}); err != errIndex {
		t.Fatalf("a selection outside the items was accepted: %v", err)
	}
	// Click opens the list; a click on its third choice picks it.
	m.press(5, 5)
	m.release(5, 5)
	if !m.popupOpen() {
		t.Fatal("click did not open the list")
	}
	m.mu.Lock()
	listY := m.list.y + 1 + 2*itemHeight + 2
	m.mu.Unlock()
	m.press(5, listY)
	m.release(5, listY)
	if m.popupOpen() {
		t.Fatal("picking a choice left the list open")
	}
	if _, _, selected, _ := m.readAll(choice); selected != 2 {
		t.Fatalf("picked %d", selected)
	}
	if changes := m.takeChanges(); len(changes) != 1 {
		t.Fatalf("changes %v", changes)
	}
	// Keyboard: a closed Select moves its selection with the arrows; Enter
	// opens, arrows move the highlight, Enter picks it.
	m.navigate("ArrowUp")
	if _, _, selected, _ := m.readAll(choice); selected != 1 {
		t.Fatalf("ArrowUp chose %d", selected)
	}
	m.activateFocused(false)
	m.navigate("ArrowUp")
	m.activateFocused(false)
	if _, _, selected, _ := m.readAll(choice); selected != 0 || m.popupOpen() {
		t.Fatalf("Enter chose %d", selected)
	}
	m.activateFocused(true)
	if !m.escape() || m.popupOpen() {
		t.Fatal("Escape did not close the list")
	}
	// An empty Select opens an empty list and chooses nothing.
	empty := addItems(t, m, root, kindSelect, nil, -1)
	if err := m.setItems(empty, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if err := m.setEnabled(choice, false); err != nil || m.focusedKind() != "" {
		t.Fatal("disabling the focused Select kept the focus")
	}
}

func TestTextAreaEditsLinesAndScrolls(t *testing.T) {
	m := newModel("t", 600, 400, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	area, err := m.addSpec(root, spec{kind: kindTextArea, placeholder: "notes", selected: -1})
	if err != nil {
		t.Fatal(err)
	}
	m.focus = area
	m.insert([]rune("Ayşe"))
	m.edit("Enter")
	m.insert([]rune("Çağrı\n"))
	m.insert([]rune("üç"))
	m.edit("ArrowUp") // to the end of line 2's first 2 columns
	m.edit("Home")
	m.insert([]rune(">"))
	m.edit("End")
	m.edit("Backspace")
	text, _, _ := m.read(area)
	if text != "Ayşe\n>Çağr\nüç" {
		t.Fatalf("edited text %q", text)
	}
	if changes := m.takeChanges(); len(changes) != 1 || changes[0] != area {
		t.Fatalf("one widget changed, reported %v", changes)
	}
	// Many lines scroll so the caret stays visible.
	m.edit("ArrowDown")
	m.edit("End")
	m.insert([]rune(strings.Repeat("\nline", 40)))
	m.mu.Lock()
	m.layout()
	target := m.widgets[area]
	v := m.viewportOf(target)
	m.mu.Unlock()
	lines := len(areaLines(target.text))
	if !v.vertical || target.scrollY != lines*lineHeight-v.h {
		t.Fatalf("scrollY %d for %d lines, viewport %+v", target.scrollY, lines, v)
	}
	m.edit("PageUp")
	m.edit("PageUp")
	if line, _ := areaPosition(target.text, target.caret); line != lines-1-2*pageItems {
		t.Fatalf("PageUp reached line %d", line)
	}
	if err := m.setText(area, "a\rb"); err == nil {
		t.Fatal("a carriage return was accepted")
	}
	if err := m.setText(area, strings.Repeat("x", maxAreaRunes)); err != nil {
		t.Fatal(err)
	}
	m.insert([]rune("y"))
	if text, _, _ := m.read(area); len([]rune(text)) != maxAreaRunes {
		t.Fatal("a TextArea grew past its bound")
	}
}

func TestPasswordInputMasksItsText(t *testing.T) {
	images := map[string][]byte{}
	for _, secret := range []string{"hunter2", "abcdefg"} {
		m := newModel("t", 300, 60, measureText)
		root, _ := m.add(0, kindColumn, "", "", false, 0, 12)
		password, err := m.addSpec(root, spec{kind: kindPassword, selected: -1})
		if err != nil {
			t.Fatal(err)
		}
		m.focus = password
		m.insert([]rune(secret))
		if text, _, _ := m.read(password); text != secret {
			t.Fatalf("value %q", text)
		}
		m.mu.Lock()
		img := render(frame{model: m, scale: 1})
		m.mu.Unlock()
		images[secret] = img.Pix
	}
	// Two different secrets of the same length draw exactly the same pixels.
	if !bytes.Equal(images["hunter2"], images["abcdefg"]) {
		t.Fatal("a PasswordInput drew its characters")
	}
}

func TestTableViewShapeSelectionAndLargeRows(t *testing.T) {
	m := newModel("t", 800, 600, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	for _, bad := range []spec{
		{kind: kindTable},
		{kind: kindTable, columns: []string{"a", "b"}, rows: [][]string{{"1"}}},
		{kind: kindTable, columns: make([]string, maxColumns+1)},
	} {
		bad.selected = -1
		if _, err := m.addSpec(root, bad); err == nil {
			t.Fatalf("table %+v accepted", bad)
		}
	}
	empty, err := m.addSpec(root, spec{kind: kindTable, columns: []string{"Name", "Name"}, selected: -1})
	if err != nil {
		t.Fatalf("duplicate column names are display labels: %v", err)
	}
	if err := m.selectIndex(empty, 0); err != errIndex {
		t.Fatal("selected a row of an empty table")
	}
	for _, count := range []int{10, 1000, 10000} {
		table, err := m.addSpec(root, spec{kind: kindTable, columns: []string{"Customer", "Amount", "Date"}, rows: tableRowsOf(count, 3), selected: -1})
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		m.mu.Lock()
		img := render(frame{model: m, scale: 2})
		m.mu.Unlock()
		if elapsed := time.Since(start); elapsed > 2*time.Second || img.Bounds().Dx() != 1600 {
			t.Fatalf("%d rows rendered in %v", count, elapsed)
		}
		if err := m.selectIndex(table, count-1); err != nil {
			t.Fatal(err)
		}
		m.mu.Lock()
		target := m.widgets[table]
		v := m.viewportOf(target)
		m.mu.Unlock()
		if target.scrollY != max(count*itemHeight-v.h, 0) {
			t.Fatalf("%d rows: the last row is not revealed (%d)", count, target.scrollY)
		}
		_ = m.setRows(table, tableRowsOf(5, 3))
		if _, _, selected, _ := m.readAll(table); selected != -1 {
			t.Fatal("setRows kept a row that no longer exists")
		}
	}
	if err := m.setRows(empty, [][]string{{"only one"}}); err == nil {
		t.Fatal("a row of the wrong width was accepted")
	}
	if _, err := m.addSpec(root, spec{kind: kindTable, columns: []string{"a"}, rows: tableRowsOf(maxRows+1, 1), selected: -1}); err == nil {
		t.Fatal("too many rows accepted")
	}
}

func TestTableColumnWidthsAndHorizontalScroll(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	wide := strings.Repeat("w", 60)
	table, err := m.addSpec(root, spec{kind: kindTable, columns: []string{"a", "Long header", "c", "d", "e"},
		rows: [][]string{{wide, "x", "", wide, wide}}, selected: -1})
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.layout()
	target := m.widgets[table]
	widths := append([]int(nil), target.widths...)
	m.mu.Unlock()
	// Widths: the widest text plus insets, between 48 and 320.
	if widths[0] != maxColumnWidth || widths[1] != 110+2*itemInset || widths[2] != minColumnWidth {
		t.Fatalf("column widths %v", widths)
	}
	m.wheel(10, 60, -5, 0)
	m.mu.Lock()
	scrolled := target.scrollX
	v := m.viewportOf(target)
	m.mu.Unlock()
	if scrolled <= 0 || scrolled > v.contentW-v.w {
		t.Fatalf("horizontal scroll %d of %d", scrolled, v.contentW-v.w)
	}
}

func TestResizableLayoutGivesExtraSpaceToScrollingWidgets(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 8, 12)
	label := mustAdd(t, m, root, kindLabel, "Customers")
	table, _ := m.addSpec(root, spec{kind: kindTable, columns: []string{"a"}, selected: -1})
	row, _ := m.add(root, kindRow, "", "", false, 8, 0)
	list := addItems(t, m, row, kindListBox, []string{"x"}, -1)
	button := mustAdd(t, m, row, kindButton, "Save")
	m.setResizable(true)
	m.resize(1000, 900)
	m.mu.Lock()
	m.layout()
	w := m.widgets
	m.mu.Unlock()
	naturalTable := headerHeight + tableRows*itemHeight + 2
	naturalList := listBoxRows*itemHeight + 2
	used := 12*2 + rowHeight + 8 + naturalTable + 8 + naturalList
	extra := 900 - used
	if w[root].w != 1000 || w[root].h != 900 {
		t.Fatalf("the root fills the window: %dx%d", w[root].w, w[root].h)
	}
	// The table and the Row holding the ListBox share the extra height.
	if w[table].h != naturalTable+extra-extra/2 || w[row].h != naturalList+extra/2 {
		t.Fatalf("table %d, row %d, extra %d", w[table].h, w[row].h, extra)
	}
	// A growing child fills the Column's width; others keep their size.
	if w[table].w != 1000-24 || w[label].w != 90 || w[button].h != rowHeight {
		t.Fatalf("widths: table %d label %d button height %d", w[table].w, w[label].w, w[button].h)
	}
	// In the Row, the ListBox takes the extra width; the Button keeps its own.
	if w[list].w != 1000-24-8-w[button].w || w[list].h != w[row].h {
		t.Fatalf("list %dx%d", w[list].w, w[list].h)
	}
	// Smaller than the content: nothing shrinks, the rest is clipped.
	m.resize(100, 100)
	m.mu.Lock()
	m.layout()
	m.mu.Unlock()
	if w[table].h != naturalTable || w[root].w < 24+tableMinWidth {
		t.Fatal("widgets shrank below their natural size")
	}
}

func TestTabAndShiftTabVisitEveryEnabledControl(t *testing.T) {
	m := newModel("t", 800, 600, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	var order []int64
	for _, kind := range []string{kindTextInput, kindPassword, kindTextArea, kindSelect, kindListBox, kindTable, kindCheckbox, kindButton} {
		s := spec{kind: kind, selected: -1}
		if kind == kindTable {
			s.columns = []string{"a"}
		}
		id, err := m.addSpec(root, s)
		if err != nil {
			t.Fatal(err)
		}
		order = append(order, id)
		mustAdd(t, m, root, kindLabel, "skip me")
	}
	_ = m.setEnabled(order[3], false)
	var forward []int64
	for range 8 {
		m.focusNext()
		forward = append(forward, m.focus)
	}
	want := []int64{order[0], order[1], order[2], order[4], order[5], order[6], order[7], order[0]}
	for index := range want {
		if forward[index] != want[index] {
			t.Fatalf("Tab order %v, want %v", forward, want)
		}
	}
	m.focusPrevious()
	if m.focus != order[7] {
		t.Fatalf("Shift+Tab reached %d", m.focus)
	}
	m.focusPrevious()
	m.focusPrevious()
	if m.focus != order[5] {
		t.Fatalf("Shift+Tab skipped wrongly to %d", m.focus)
	}
}

func TestProtocolChangeEventsAndNewKinds(t *testing.T) {
	open := `{"op":"open","id":1,"version":3,"width":500,"height":400,"title":"t","headless":true,"script":[` +
		`{"event":"type","widget":2,"text":"Ali"},{"event":"type","widget":3,"text":"s3cret"},{"event":"toggle","widget":4},` +
		`{"event":"select","widget":5,"selected":1},{"event":"select","widget":6,"selected":2},{"event":"select","widget":7,"selected":0},` +
		`{"event":"type","widget":8,"text":"a\nb"},{"event":"resize","width":900,"height":700}]}`
	replies, events := run(t, open,
		`{"op":"add","id":2,"kind":"column"}`,
		`{"op":"add","id":3,"parent":1,"kind":"textInput"}`,
		`{"op":"add","id":4,"parent":1,"kind":"passwordInput"}`,
		`{"op":"add","id":5,"parent":1,"kind":"checkbox","text":"Paid"}`,
		`{"op":"add","id":6,"parent":1,"kind":"listBox","items":["a","b"]}`,
		`{"op":"add","id":7,"parent":1,"kind":"select","items":["x","y","z"],"selected":0}`,
		`{"op":"add","id":8,"parent":1,"kind":"table","columns":["n"],"rows":[["1"],["2"]]}`,
		`{"op":"add","id":9,"parent":1,"kind":"textArea"}`,
		`{"op":"resizable","id":10,"enabled":true}`,
		`{"op":"listen","id":11,"kind":"change","widget":2}`,
		`{"op":"listen","id":12,"kind":"change","widget":3}`,
		`{"op":"listen","id":13,"kind":"change","widget":5}`,
		`{"op":"listen","id":14,"kind":"change","widget":6}`,
		`{"op":"listen","id":15,"kind":"change","widget":7}`,
		`{"op":"listen","id":16,"kind":"change","widget":8}`,
		`{"op":"wait","id":17}`,
		`{"op":"next","id":18}`, `{"op":"next","id":19}`, `{"op":"next","id":20}`, `{"op":"next","id":21}`,
		`{"op":"next","id":22}`, `{"op":"next","id":23}`,
		`{"op":"close","id":24}`)
	for index, reply := range replies {
		if !reply.OK {
			t.Fatalf("reply %d: %+v", index, reply)
		}
	}
	// The toggle of widget 4 is not listened to, so it sends nothing.
	want := []string{"change:2:Ali", "change:3:s3cret", "change:5:1", "change:6:2", "change:7:0", "change:8:a\nb", "closed"}
	if len(events) != len(want) {
		t.Fatalf("events %+v", events)
	}
	for index, e := range events {
		got := e.Event
		if e.Event == "change" {
			got += fmt.Sprintf(":%d:", e.Widget)
			if e.Selected != nil {
				got += fmt.Sprint(*e.Selected)
			} else {
				got += e.Text
			}
		}
		if got != want[index] {
			t.Fatalf("event %d = %q, want %q", index, got, want[index])
		}
	}
	final := events[len(events)-1].Values
	if len(final) != 7 || final[2].Checked != true || *final[4].Selected != 2 {
		t.Fatalf("closed values %+v", final)
	}
}
