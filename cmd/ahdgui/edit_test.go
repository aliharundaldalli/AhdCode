package main

import (
	"strings"
	"testing"
)

func TestTextSelectionUndoAndClipboard(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	input, _ := m.add(root, kindTextInput, "", "", false, 0, 0)
	area, _ := m.addSpec(root, spec{kind: kindTextArea, selected: -1})
	password, _ := m.addSpec(root, spec{kind: kindPassword, selected: -1})
	m.focus = input
	m.insert([]rune("abcd"))
	if m.editMove("ArrowLeft", true) == false || m.editMove("ArrowLeft", true) == false {
		t.Fatal("shift selection")
	}
	text, copied := m.command(input, "cut")
	if text != "cd" || !copied {
		t.Fatalf("cut %q %v", text, copied)
	}
	if got, _, _ := m.read(input); got != "ab" {
		t.Fatalf("after cut %q", got)
	}
	m.command(input, "undo")
	if got, _, _ := m.read(input); got != "abcd" {
		t.Fatalf("undo cut %q", got)
	}
	m.setText(input, "")
	m.focus = input
	m.insert([]rune("ab"))
	m.edit("ArrowLeft")
	m.insert([]rune("c"))
	m.command(input, "undo")
	if got, _, _ := m.read(input); got != "ab" {
		t.Fatalf("undo one typed run %q", got)
	}
	m.command(input, "undo")
	if got, _, _ := m.read(input); got != "" {
		t.Fatalf("undo previous run %q", got)
	}
	m.command(input, "redo")
	if got, _, _ := m.read(input); got != "ab" {
		t.Fatalf("redo %q", got)
	}

	m.focus = input
	m.setText(input, "e\u0301x")
	m.focus = input
	m.edit("Backspace")
	m.edit("Backspace")
	if got, _, _ := m.read(input); got != "" {
		t.Fatalf("grapheme delete left %q", got)
	}

	m.setProgramClipboard(strings.Repeat("x", maxTextRunes+1))
	m.setText(input, "ok")
	m.focus = input
	m.command(input, "paste")
	if got, _, _ := m.read(input); got != "ok" {
		t.Fatalf("an oversized paste changed the field to %q", got)
	}
	m.setProgramClipboard("dünya")
	m.command(input, "paste")
	if got, _, _ := m.read(input); got != "okdünya" {
		t.Fatalf("paste %q", got)
	}

	m.focus = password
	m.insert([]rune("secret"))
	m.command(password, "selectAll")
	if text, copied := m.command(password, "copy"); copied || text != "" {
		t.Fatal("a PasswordInput copied its text")
	}
	if got, _, _ := m.read(password); got != "secret" {
		t.Fatal("refusing copy changed the password")
	}

	m.setVisible(input, false)
	m.command(input, "paste")
	if got, _, _ := m.read(input); got != "okdünya" {
		t.Fatalf("a hidden field changed to %q", got)
	}
	m.setEnabled(area, false)
	m.focus = area
	if m.insert([]rune("x")) {
		t.Fatal("a disabled TextArea accepted typing")
	}

	if textShortcut("darwin", true, false, "Z") != "undo" || textShortcut("linux", true, true, "Z") != "redo" || textShortcut("windows", false, false, "V") != "" {
		t.Fatal("shortcut mapping")
	}
	if textShortcut("darwin", true, false, "Y") != "" || textShortcut("windows", true, false, "Y") != "redo" || textShortcut("linux", true, false, "Y") != "redo" {
		t.Fatal("Y redoes on Windows and Linux only")
	}
	if textShortcut("darwin", true, false, "A") != "selectAll" || textShortcut("linux", true, false, "C") != "copy" || textShortcut("windows", true, false, "X") != "cut" || textShortcut("darwin", true, false, "V") != "paste" {
		t.Fatal("copy shortcuts")
	}
}

func TestMouseSelectsText(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	input, _ := m.add(root, kindTextInput, "", "", false, 0, 0)
	m.focus = input
	m.insert([]rune("hello world"))
	m.mu.Lock()
	m.layout()
	field := m.widgets[input]
	x, y := field.x+textInputInset+15, field.y+8
	m.mu.Unlock()
	m.pressAt(x, y, true)
	m.mu.Lock()
	start, end := m.widgets[input].selection()
	selected := string(m.widgets[input].text[start:end])
	m.mu.Unlock()
	if selected != "hello" {
		t.Fatalf("double click selected %q", selected)
	}
	m.press(field.x+textInputInset+2, y)
	m.drag(field.x+textInputInset+55, y)
	m.release(field.x+textInputInset+55, y)
	m.mu.Lock()
	start, end = m.widgets[input].selection()
	selected = string(m.widgets[input].text[start:end])
	m.mu.Unlock()
	if selected == "" {
		t.Fatal("dragging did not select")
	}
}

func TestGraphemeClustersStayTogether(t *testing.T) {
	text := []rune("e\u0301👍a")
	if nextCluster(text, 0) != 2 {
		t.Fatalf("combining mark split at %d", nextCluster(text, 0))
	}
	if prevCluster(text, len(text)) != len(text)-1 {
		t.Fatal("previous cluster")
	}
	flag := []rune("🇹🇷")
	if nextCluster(flag, 0) != len(flag) {
		t.Fatalf("flag split at %d", nextCluster(flag, 0))
	}
}
