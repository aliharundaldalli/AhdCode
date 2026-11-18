package main

import "testing"

func TestTableSortIsStableAndKeepsTheSelectedRow(t *testing.T) {
	m := newModel("t", 480, 360, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	table := mustSpec(t, m, root, spec{
		kind: kindTable, columns: []string{"Name", "Note"}, selected: -1,
		rows: [][]string{{"b", ""}, {"a", "m"}, {"b", "m"}, {"a", ""}, {"", "z"}},
	})
	if err := m.selectIndex(table, 0); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	box := widgetBox(m, table)
	m.press(box[0]+4, box[1]+4)
	rows, selected, err := m.rowsNow(table)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"", "z"}, {"a", "m"}, {"a", ""}, {"b", ""}, {"b", "m"}}
	if !sameRows(rows, want) || selected != 3 {
		t.Fatalf("ascending rows %v selected %d", rows, selected)
	}
	if column, desc, err := m.sortNow(table); err != nil || column != 0 || desc {
		t.Fatalf("sort state %d %v %v", column, desc, err)
	}
	if len(m.takeChanges()) != 0 {
		t.Fatal("a header sort notified onSelect")
	}
	id, column, desc, ok := m.takeSort()
	if !ok || id != table || column != 0 || desc {
		t.Fatalf("sort notice %d %d %v %v", id, column, desc, ok)
	}
	if _, _, _, again := m.takeSort(); again {
		t.Fatal("the sort notice was delivered twice")
	}

	m.press(box[0]+4, box[1]+4)
	rows, selected, _ = m.rowsNow(table)
	want = [][]string{{"b", ""}, {"b", "m"}, {"a", "m"}, {"a", ""}, {"", "z"}}
	if !sameRows(rows, want) || selected != 0 {
		t.Fatalf("descending rows %v selected %d", rows, selected)
	}

	noteX := box[0] + 4 + 320
	if m.headerColumn(m.widgets[table], noteX, box[1]+4) != 1 && m.headerColumn(m.widgets[table], box[0]+80, box[1]+4) != 1 {
		// The second column starts after the first. Ask the hitter rather than guessing a pixel.
	}
	m.mu.Lock()
	second := m.headerColumn(m.widgets[table], box[0]+m.widgets[table].widths[0]+4, box[1]+4)
	m.mu.Unlock()
	if second != 1 {
		t.Fatalf("header column %d", second)
	}
	m.press(box[0]+m.widgets[table].widths[0]+4, box[1]+4)
	rows, selected, _ = m.rowsNow(table)
	// Empty notes first, in the order they already had, then the "m" notes
	// in the order they already had, then z.
	want = [][]string{{"b", ""}, {"a", ""}, {"b", "m"}, {"a", "m"}, {"", "z"}}
	if !sameRows(rows, want) || selected != 0 {
		t.Fatalf("note column %v selected %d", rows, selected)
	}

	m.takeSort()
	if err := m.sortBy(table, 0, false); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := m.takeSort(); ok {
		t.Fatal("a programmatic sort notified the header")
	}
	if err := m.sortBy(table, 9, false); err == nil {
		t.Fatal("accepted a column outside the table")
	}
	if err := m.setRows(table, [][]string{{"c", "1"}, {"a", "1"}}); err != nil {
		t.Fatal(err)
	}
	rows, _, _ = m.rowsNow(table)
	if !sameRows(rows, [][]string{{"a", "1"}, {"c", "1"}}) {
		t.Fatalf("setRows did not keep the sort: %v", rows)
	}

	if err := m.setEnabled(table, false); err != nil {
		t.Fatal(err)
	}
	before, _, _ := m.rowsNow(table)
	m.press(box[0]+4, box[1]+4)
	after, _, _ := m.rowsNow(table)
	if !sameRows(before, after) {
		t.Fatal("a disabled table sorted from the header")
	}
	if err := m.sortBy(table, 0, true); err != nil {
		t.Fatal(err)
	}
	rows, _, _ = m.rowsNow(table)
	if !sameRows(rows, [][]string{{"c", "1"}, {"a", "1"}}) {
		t.Fatalf("programmatic sort on a disabled table: %v", rows)
	}
}

func TestTableHeaderRespectsHorizontalScroll(t *testing.T) {
	m := newModel("t", 200, 240, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	table := mustSpec(t, m, root, spec{
		kind: kindTable, columns: []string{"One", "Two"}, selected: -1,
		rows: [][]string{{"aaaa", "bbbb"}},
	})
	layout(t, m)
	m.mu.Lock()
	target := m.widgets[table]
	target.scrollX = target.widths[0]
	column := m.headerColumn(target, target.x+4, target.y+4)
	m.mu.Unlock()
	if column != 1 {
		t.Fatalf("scrolled header hit %d", column)
	}
}

func TestWrappingLabelKeepsMeasuredLinesWhenPlacedNarrower(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	note := mustAdd(t, m, root, kindLabel, "hello world")
	if err := m.setBox("wrap", note, 1, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := m.setBox("preferred", note, 50, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{50, 64} {
		t.Fatalf("measured %v", sizeOf(m, note))
	}
	m.resize(80, 80)
	layout(t, m)
	if sizeOf(m, note) != [2]int{50, 64} {
		t.Fatalf("a narrower window changed the label %v", sizeOf(m, note))
	}
	m.resize(900, 700)
	layout(t, m)
	if sizeOf(m, note) != [2]int{50, 64} {
		t.Fatalf("a wider window changed the label %v", sizeOf(m, note))
	}
	m.widgets[note].text = []rune("İstanbul")
	layout(t, m)
	if sizeOf(m, note) != [2]int{50, 64} {
		t.Fatalf("Turkish %v", sizeOf(m, note))
	}
	m.widgets[note].text = []rune("one\r\ntwo")
	if err := m.setBox("preferred", note, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{30, 64} {
		t.Fatalf("explicit break %v", sizeOf(m, note))
	}

	m.widgets[note].text = []rune("hello world")
	if err := m.setBox("preferred", note, 50, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	label := m.widgets[note]
	label.w = 20
	lines := m.wrappedLines(label)
	if len(lines) != 2 || lines[0] != "hello" || lines[1] != "world" {
		t.Fatalf("narrow placement dropped lines: %#v", lines)
	}
}

func sameRows(got, want [][]string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if len(got[index]) != len(want[index]) {
			return false
		}
		for cell := range want[index] {
			if got[index][cell] != want[index][cell] {
				return false
			}
		}
	}
	return true
}
