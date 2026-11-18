package main

import (
	"runtime"
	"testing"
)

func TestMenuBarItemsAndClicks(t *testing.T) {
	m := newModel("t", 480, 280, fixedMeasure)
	file, err := m.addMenu(0, "File")
	if err != nil {
		t.Fatal(err)
	}
	save, err := m.addItem(file, menuKindItem, "Save", "s", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.addItem(file, menuKindItem, "Also", "S", false); err == nil {
		t.Fatal("a second item accepted the same shortcut")
	}
	if _, err := m.addItem(file, menuKindItem, "Bad", "Ctrl+S", false); err == nil {
		t.Fatal("the primary modifier was accepted in a shortcut")
	}
	bold, err := m.addItem(file, menuKindCheck, "Bold", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.addSeparator(file); err != nil {
		t.Fatal(err)
	}
	edit, err := m.addMenu(file, "Edit")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.addMenu(edit, "More"); err == nil {
		t.Fatal("a nested submenu was accepted")
	}
	copyItem, err := m.addItem(edit, menuKindItem, "Copy", "Alt+Shift+C", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.setMenuEnabled(save, false); err != nil {
		t.Fatal(err)
	}
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	m.mu.Lock()
	m.layout()
	if m.widgets[root].y != menuBarHeight {
		t.Fatalf("content starts at %d, want %d", m.widgets[root].y, menuBarHeight)
	}
	m.mu.Unlock()

	m.press(8, 8)
	if m.openMenu != file || m.takeMenu() != 0 {
		t.Fatalf("bar click open=%d", m.openMenu)
	}
	// The disabled Save item is the first row of the open menu.
	m.press(8, menuBarHeight+4)
	if got := m.takeMenu(); got != 0 {
		t.Fatalf("disabled item ran: %d", got)
	}
	enabled, err := m.menuEnabled(save)
	if err != nil || enabled {
		t.Fatalf("enabled = %v %v", enabled, err)
	}
	if err := m.setMenuEnabled(save, true); err != nil {
		t.Fatal(err)
	}
	m.press(8, menuBarHeight+4)
	if got := m.takeMenu(); got != save {
		t.Fatalf("activated %d, want %d", got, save)
	}
	if m.openMenu != 0 {
		t.Fatal("the menu stayed open after an item ran")
	}

	m.press(8, 8)
	// Save, Bold, separator, Edit.
	editY := menuBarHeight + menuItemHeight + menuItemHeight + 9 + 4
	m.press(8, editY)
	if m.openSub != edit {
		t.Fatalf("submenu open=%d", m.openSub)
	}
	m.mu.Lock()
	m.layout()
	sub := m.menuSub[0]
	m.mu.Unlock()
	m.press(sub.x+4, sub.y+4)
	if got := m.takeMenu(); got != copyItem {
		t.Fatalf("submenu item %d, want %d", got, copyItem)
	}

	m.press(8, 8)
	m.press(8, menuBarHeight+menuItemHeight+4)
	if got := m.takeMenu(); got != bold {
		t.Fatalf("check item %d", got)
	}
	checked, err := m.menuChecked(bold)
	if err != nil || !checked {
		t.Fatalf("checked = %v %v", checked, err)
	}
	if err := m.setMenuChecked(save, true); err == nil {
		t.Fatal("a plain item accepted a check")
	}

	img := render(frame{model: m, scale: 1})
	if img.RGBAAt(2, 2) != colorMenuBar {
		t.Fatalf("menu bar pixel = %v", img.RGBAAt(2, 2))
	}
}

func TestMenuShortcutPrecedence(t *testing.T) {
	m := newModel("t", 480, 280, fixedMeasure)
	file, _ := m.addMenu(0, "File")
	save, _ := m.addItem(file, menuKindItem, "Save", "S", false)
	copyItem, _ := m.addItem(file, menuKindItem, "Copy", "C", false)
	again, _ := m.addItem(file, menuKindItem, "Redo", "Shift+Z", false)
	alt, _ := m.addItem(file, menuKindItem, "Other", "Alt+S", false)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	field := mustAdd(t, m, root, kindTextInput, "")
	secret := mustAdd(t, m, root, kindPassword, "")
	button := mustAdd(t, m, root, kindButton, "Go")
	m.focusWidget(field)
	m.insert([]rune("Ali"))
	m.command(field, "selectAll")

	id, copied, did, consumed := m.applyChord(runtime.GOOS, true, false, false, "C")
	if id != 0 || !did || copied != "Ali" || !consumed {
		t.Fatalf("text copy = id %d %q did %v consumed %v", id, copied, did, consumed)
	}
	if got := m.clipboard; got != "Ali" {
		t.Fatalf("clipboard %q", got)
	}
	id, _, _, consumed = m.applyChord(runtime.GOOS, true, false, false, "S")
	if id != save || !consumed {
		t.Fatalf("save chord = %d consumed %v", id, consumed)
	}
	id, _, _, consumed = m.applyChord("linux", true, false, false, "Y")
	if id != 0 || !consumed {
		t.Fatalf("linux Ctrl+Y should be redo, id %d consumed %v", id, consumed)
	}
	id, _, _, consumed = m.applyChord("darwin", true, false, true, "Z")
	if id != 0 || !consumed {
		t.Fatalf("focused Shift+Z should redo in the field, id %d consumed %v", id, consumed)
	}

	m.focusWidget(button)
	id, _, _, _ = m.applyChord(runtime.GOOS, true, false, false, "C")
	if id != copyItem {
		t.Fatalf("menu Copy = %d, want %d", id, copyItem)
	}
	id, _, _, _ = m.applyChord(runtime.GOOS, true, true, false, "S")
	if id != alt {
		t.Fatalf("Alt+S = %d, want %d", id, alt)
	}
	id, _, _, _ = m.applyChord("darwin", true, false, true, "Z")
	if id != again {
		t.Fatalf("menu Shift+Z = %d, want %d", id, again)
	}
	id, _, _, consumed = m.applyChord("darwin", true, false, false, "Y")
	if id != 0 || consumed {
		t.Fatalf("macOS Command+Y is not redo and has no menu item: %d %v", id, consumed)
	}

	if err := m.setMenuEnabled(save, false); err != nil {
		t.Fatal(err)
	}
	id, _, _, consumed = m.applyChord(runtime.GOOS, true, false, false, "S")
	if id != 0 || !consumed {
		t.Fatalf("disabled shortcut reserved = id %d consumed %v", id, consumed)
	}

	m.focusWidget(secret)
	m.insert([]rune("secret"))
	m.command(secret, "selectAll")
	id, copied, did, consumed = m.applyChord(runtime.GOOS, true, false, false, "C")
	if id != 0 || did || copied != "" || !consumed {
		t.Fatalf("password copy = id %d %q did %v consumed %v", id, copied, did, consumed)
	}
	if m.clipboard != "Ali" {
		t.Fatalf("password changed the clipboard to %q", m.clipboard)
	}

	if err := m.setVisible(field, false); err != nil {
		t.Fatal(err)
	}
	m.focusWidget(field)
	if m.focus == field {
		t.Fatal("a hidden field took the focus")
	}
	m.focusWidget(button)
	id, _, _, _ = m.applyChord(runtime.GOOS, true, false, false, "C")
	if id != copyItem {
		t.Fatalf("hidden field still owned Copy, menu id %d", id)
	}
}

func TestMenuScriptShortcutAndClick(t *testing.T) {
	open := `{"op":"open","id":1,"version":3,"width":480,"height":280,"title":"t","headless":true,"script":[` +
		`{"event":"type","widget":6,"text":"Ali"},{"event":"edit","widget":6,"key":"selectAll"},` +
		`{"event":"focus","widget":6},{"event":"shortcut","key":"C"},{"event":"shortcut","key":"S"},` +
		`{"event":"shortcut","key":"D"},{"event":"menu","widget":3}]}`
	replies, events := run(t, open,
		`{"op":"menu","id":2,"kind":"menu","text":"File"}`,
		`{"op":"menu","id":3,"kind":"item","parent":1,"text":"Save","name":"S"}`,
		`{"op":"menu","id":4,"kind":"check","parent":1,"text":"Bold"}`,
		`{"op":"menu","id":5,"kind":"separator","parent":1}`,
		`{"op":"add","id":6,"kind":"column"}`,
		`{"op":"add","id":7,"parent":5,"kind":"textInput"}`,
		`{"op":"menu","id":8,"kind":"item","parent":1,"text":"Drop","name":"D"}`,
		`{"op":"menu","id":9,"kind":"enabled","widget":7,"enabled":false}`,
		`{"op":"listen","id":10,"kind":"menu","widget":2}`,
		`{"op":"listen","id":11,"kind":"menu","widget":3}`,
		`{"op":"wait","id":12}`,
		`{"op":"next","id":13}`,
		`{"op":"next","id":14}`,
		`{"op":"menu","id":15,"kind":"checked","widget":3}`,
		`{"op":"get","id":16,"widget":6}`)
	for index, reply := range replies {
		if !reply.OK {
			t.Fatalf("reply %d: %+v", index, reply)
		}
	}
	if replies[1].Widget != 1 || replies[2].Widget != 2 || replies[6].Widget != 6 {
		t.Fatalf("ids: menu %d item %d field %d", replies[1].Widget, replies[2].Widget, replies[6].Widget)
	}
	want := []string{"clipboard", "menu", "menu"}
	if len(events) < 3 {
		t.Fatalf("events: %+v", events)
	}
	for index, event := range events[:3] {
		if event.Event != want[index] {
			t.Fatalf("event %d = %s, want %s (%+v)", index, event.Event, want[index], events)
		}
	}
	if events[0].Text != "Ali" {
		t.Fatalf("copied %q", events[0].Text)
	}
	if events[1].Widget != 2 || events[2].Widget != 3 {
		t.Fatalf("menu events: %+v %+v", events[1], events[2])
	}
	if replies[14].Checked == nil || !*replies[14].Checked {
		t.Fatalf("check state %+v", replies[14])
	}
	if replies[15].Text == nil || *replies[15].Text != "Ali" {
		t.Fatalf("field %+v", replies[15].Text)
	}
}
