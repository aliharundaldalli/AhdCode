package main

import "testing"

func TestWrappingLabelReflowsToANarrowWindow(t *testing.T) {
	m := newModel("t", 420, 280, fixedMeasure)
	root := mustSpec(t, m, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	note := mustAdd(t, m, root, kindLabel, "hello world")
	if err := m.setBox("wrap", note, 1, 0, "", ""); err != nil || m.setBox("preferred", note, 200, 0, "", "") != nil {
		t.Fatal("wrap")
	}
	below := mustAdd(t, m, root, kindLabel, "next")
	wide := mustAdd(t, m, root, kindButton, "WIDE")
	if err := m.setBox("min", wide, 180, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{200, 32} {
		t.Fatalf("wide label %v", sizeOf(m, note))
	}
	if sizeOf(m, wide)[0] < 180 {
		t.Fatalf("button %v", sizeOf(m, wide))
	}
	nextY := m.widgets[below].y

	m.resize(80, 280)
	layout(t, m)
	got := sizeOf(m, note)
	if got[0] > 80 || got[0] >= 200 || got[1] <= 32 {
		t.Fatalf("narrow label %v", got)
	}
	lines := m.wrappedLines(m.widgets[note])
	if len(lines)*labelHeight("") > got[1] {
		t.Fatalf("lines do not fit: %#v in %v", lines, got)
	}
	if m.widgets[below].y < m.widgets[note].y+m.widgets[note].h || m.widgets[below].y <= nextY {
		t.Fatalf("sibling note %v next %v wide-y %d", widgetBox(m, note), widgetBox(m, below), nextY)
	}
	if sizeOf(m, wide)[0] < 180 {
		t.Fatalf("a narrow window shrank the button %v", sizeOf(m, wide))
	}

	m.resize(420, 280)
	layout(t, m)
	if sizeOf(m, note) != [2]int{200, 32} {
		t.Fatalf("restored %v", sizeOf(m, note))
	}
}

func TestGrowingColumnReflowsALabelWithoutShrinkingAField(t *testing.T) {
	m := newModel("t", 80, 300, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	note := mustAdd(t, m, root, kindLabel, "hello world")
	if err := m.setBox("wrap", note, 1, 0, "", ""); err != nil || m.setBox("preferred", note, 200, 0, "", "") != nil {
		t.Fatal(err)
	}
	field := mustAdd(t, m, root, kindTextArea, "")
	layout(t, m)
	if sizeOf(m, note)[0] > 80 || sizeOf(m, note)[1] <= 32 {
		t.Fatalf("label %v", sizeOf(m, note))
	}
	if sizeOf(m, field)[0] != textAreaWidth {
		t.Fatalf("field %v", sizeOf(m, field))
	}
	if m.widgets[field].y < m.widgets[note].y+m.widgets[note].h {
		t.Fatalf("overlap %v %v", widgetBox(m, note), widgetBox(m, field))
	}
}

func TestWrappingLabelKeepsItsMinimumAndItsLetters(t *testing.T) {
	m := newModel("t", 80, 240, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	note := mustAdd(t, m, root, kindLabel, "İstanbul")
	if err := m.setBox("wrap", note, 1, 0, "", ""); err != nil || m.setBox("preferred", note, 200, 0, "", "") != nil || m.setBox("min", note, 60, 0, "", "") != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note)[0] < 60 {
		t.Fatalf("minimum was ignored %v", sizeOf(m, note))
	}
	lines := m.wrappedLines(m.widgets[note])
	joined := ""
	for _, line := range lines {
		joined += line
	}
	if joined != "İstanbul" {
		t.Fatalf("Turkish text changed %#v", lines)
	}

	m.widgets[note].text = []rune("one\r\ntwo")
	if err := m.setBox("preferred", note, 30, 0, "", ""); err != nil || m.setBox("min", note, 0, 0, "", "") != nil {
		t.Fatal(err)
	}
	m.resize(20, 240)
	layout(t, m)
	lines = m.wrappedLines(m.widgets[note])
	if len(lines) < 2 || lines[0] == "" || lines[len(lines)-1] == "" {
		t.Fatalf("explicit break %v lines %#v", sizeOf(m, note), lines)
	}
	plain := mustAdd(t, m, root, kindLabel, "abcdefghij")
	layout(t, m)
	if sizeOf(m, plain)[0] != 100 {
		t.Fatalf("a non-wrapping label shrank %v", sizeOf(m, plain))
	}
}

func TestGridReflowsAWrappingLabelWithoutShrinkingAField(t *testing.T) {
	m := newModel("t", 300, 320, fixedMeasure)
	root := mustSpec(t, m, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	grid := mustSpec(t, m, root, spec{kind: kindGrid, gridColumns: 2, spacing: 0, rowGap: 0, selected: -1})
	field := mustAdd(t, m, grid, kindTextInput, "")
	note := mustAdd(t, m, grid, kindLabel, "word word word word word word word word word word")
	if err := m.setBox("wrap", note, 1, 0, "", ""); err != nil || m.setBox("preferred", note, 400, 0, "", "") != nil || m.setBox("span", note, 2, 1, "", "") != nil {
		t.Fatal(err)
	}
	below := mustAdd(t, m, grid, kindLabel, "under")
	if err := m.setBox("span", below, 2, 1, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, field)[0] != textInputWidth {
		t.Fatalf("field %v", sizeOf(m, field))
	}
	narrow := sizeOf(m, note)
	if narrow[0] >= 400 || narrow[0] < textInputWidth {
		t.Fatalf("label width %v", narrow)
	}
	if narrow[1] <= 32 {
		t.Fatalf("label did not grow %v", narrow)
	}
	if m.widgets[below].y < m.widgets[note].y+m.widgets[note].h {
		t.Fatalf("grid overlap %v %v", widgetBox(m, note), widgetBox(m, below))
	}

	m.resize(640, 320)
	layout(t, m)
	if sizeOf(m, note)[0] != 400 || sizeOf(m, field)[0] != textInputWidth {
		t.Fatalf("wide grid label %v field %v", sizeOf(m, note), sizeOf(m, field))
	}
}

func TestTabsReflowTheSelectedPage(t *testing.T) {
	m := newModel("t", 90, 240, fixedMeasure)
	root := mustSpec(t, m, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	tabs := mustAdd(t, m, root, kindTabs, "")
	one := mustAdd(t, m, tabs, kindPage, "One")
	note := mustAdd(t, m, one, kindLabel, "hello world")
	if err := m.setBox("wrap", note, 1, 0, "", ""); err != nil || m.setBox("preferred", note, 200, 0, "", "") != nil {
		t.Fatal(err)
	}
	two := mustAdd(t, m, tabs, kindPage, "Two")
	other := mustAdd(t, m, two, kindButton, "Stay")
	layout(t, m)
	if sizeOf(m, note)[0] >= 200 || sizeOf(m, note)[1] <= 32 {
		t.Fatalf("tab label %v", sizeOf(m, note))
	}
	if sizeOf(m, other) != [2]int{0, 0} {
		t.Fatalf("hidden page %v", sizeOf(m, other))
	}
}
