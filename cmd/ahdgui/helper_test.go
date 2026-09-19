package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"image/color"
	"strings"
	"testing"
)

// fixedMeasure makes layout tests independent of the font: every character
// is 10 logical pixels wide.
func fixedMeasure(text string) int { return 10 * len([]rune(text)) }

func mustAdd(t *testing.T, m *model, parent int64, kind, text string) int64 {
	t.Helper()
	id, err := m.add(parent, kind, text, "", false, 8, 0)
	if err != nil {
		t.Fatalf("add %s: %v", kind, err)
	}
	return id
}

func TestRootRules(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	if _, err := m.add(0, kindLabel, "x", "", false, 0, 0); err == nil {
		t.Fatal("a Label became the root")
	}
	root, err := m.add(0, kindColumn, "", "", false, 8, 12)
	if err != nil || root != 1 {
		t.Fatalf("root: %d %v", root, err)
	}
	if _, err := m.add(0, kindRow, "", "", false, 8, 12); err != errSecondRoot {
		t.Fatalf("second root: %v", err)
	}
	label := mustAdd(t, m, root, kindLabel, "hi")
	if _, err := m.add(label, kindButton, "x", "", false, 0, 0); err != errNotContainer {
		t.Fatalf("child of a Label: %v", err)
	}
	if _, err := m.add(99, kindButton, "x", "", false, 0, 0); err != errNoParent {
		t.Fatalf("missing parent: %v", err)
	}
	for _, bad := range [][2]int{{-1, 0}, {0, -1}, {1001, 0}} {
		if _, err := m.add(root, kindRow, "", "", false, bad[0], bad[1]); err == nil {
			t.Fatalf("spacing/padding %v accepted", bad)
		}
	}
	if _, err := m.add(root, "grid", "", "", false, 0, 0); err == nil {
		t.Fatal("unknown kind accepted")
	}
}

func TestColumnAndRowLayout(t *testing.T) {
	m := newModel("t", 800, 600, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 8, 12)
	label := mustAdd(t, m, root, kindLabel, "Name")                 // 40 x 32
	input, _ := m.add(root, kindTextInput, "", "hint", false, 0, 0) // 260 x 32
	row, _ := m.add(root, kindRow, "", "", false, 6, 0)
	check := mustAdd(t, m, row, kindCheckbox, "Paid") // 18+8+40 = 66
	button := mustAdd(t, m, row, kindButton, "Go")    // max(20+28, 64) = 64
	nested, _ := m.add(row, kindColumn, "", "", false, 0, 4)
	inner := mustAdd(t, m, nested, kindLabel, "a") // 10 x 32
	m.mu.Lock()
	m.layout()
	w := m.widgets
	m.mu.Unlock()
	expect := func(id int64, x, y, width, height int) {
		t.Helper()
		got := w[id]
		if got.x != x || got.y != y || got.w != width || got.h != height {
			t.Fatalf("widget %d (%s) at %d,%d %dx%d; want %d,%d %dx%d", id, got.kind, got.x, got.y, got.w, got.h, x, y, width, height)
		}
	}
	expect(label, 12, 12, 40, 32)
	expect(input, 12, 52, 260, 32)
	// The Row: 66 + 6 + 64 + 6 + (10 + 8 padding) = 160 wide, 40 tall
	// (the nested Column is 32 + 8 padding); Row children are centered.
	expect(row, 12, 92, 160, 40)
	expect(check, 12, 96, 66, 32)
	expect(button, 84, 96, 64, 32)
	expect(nested, 154, 92, 18, 40)
	expect(inner, 158, 96, 10, 32)
	expect(root, 0, 0, 260+24, 32*2+40+8*2+24)
}

func TestTextEditing(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	input, _ := m.add(root, kindTextInput, "", "", false, 0, 0)
	if m.insert([]rune("x")) {
		t.Fatal("typing with no focus reached a TextInput")
	}
	m.focus = input
	m.insert([]rune("Ayşe"))
	m.edit("ArrowLeft")
	m.edit("ArrowLeft")
	m.edit("Backspace") // removes y
	m.insert([]rune("i"))
	m.edit("Home")
	m.edit("Delete") // removes A
	m.edit("End")
	m.insert([]rune{'!', '\n', 0x7f})
	text, _, _ := m.read(input)
	if text != "işe!" {
		t.Fatalf("edited text %q", text)
	}
	m.insert([]rune(strings.Repeat("a", maxTextRunes+10)))
	if text, _, _ = m.read(input); len([]rune(text)) != maxTextRunes {
		t.Fatalf("text grew past the bound: %d", len([]rune(text)))
	}
}

func TestFocusClickAndToggle(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	mustAdd(t, m, root, kindLabel, "L")
	button := mustAdd(t, m, root, kindButton, "B")
	check := mustAdd(t, m, root, kindCheckbox, "C")
	input, _ := m.add(root, kindTextInput, "", "", false, 0, 0)
	// Label at y 0..32, Button 32..64, Checkbox 64..96, TextInput 96..128.
	m.press(5, 40)
	if m.focus != button || m.release(5, 40) != button {
		t.Fatal("clicking the Button")
	}
	m.press(5, 40)
	if m.release(5, 80) != 0 {
		t.Fatal("a press on the Button released elsewhere clicked it")
	}
	m.press(5, 70)
	m.release(5, 70)
	if _, checked, _ := m.read(check); !checked {
		t.Fatal("clicking the Checkbox did not check it")
	}
	m.focus = 0
	m.focusNext()
	if m.focus != button {
		t.Fatalf("Tab from nothing focused %d", m.focus)
	}
	m.focusNext()
	m.focusNext()
	if m.focus != input {
		t.Fatalf("Tab order reached %d", m.focus)
	}
	m.focusNext()
	if m.focus != button {
		t.Fatal("Tab did not wrap around")
	}
	if m.activateFocused(false) != button {
		t.Fatal("Enter on a focused Button")
	}
	m.focus = check
	m.activateFocused(false)
	if _, checked, _ := m.read(check); !checked {
		t.Fatal("Enter toggled a Checkbox")
	}
	m.activateFocused(true)
	if _, checked, _ := m.read(check); checked {
		t.Fatal("Space did not toggle the focused Checkbox")
	}
	m.press(300, 290)
	if m.focus != 0 {
		t.Fatal("clicking empty space kept the focus")
	}
}

// run serves a headless session over in-memory pipes.
func run(t *testing.T, lines ...string) ([]response, []event) {
	t.Helper()
	input := strings.NewReader(strings.Join(lines, "\n") + "\n")
	var output bytes.Buffer
	in := bufio.NewReaderSize(input, maxRequestBytes+1)
	out := bufio.NewWriter(&output)
	first, err := readRequest(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateOpen(first); err != nil {
		_ = writeLine(out, response{ID: first.ID, Error: err.Error()})
	} else {
		s := newSession(first, in, out)
		s.model.measure = fixedMeasure
		close(s.windowGone)
		_ = s.send(response{ID: first.ID, OK: true})
		s.serve()
	}
	var replies []response
	var events []event
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if strings.HasPrefix(line, `{"event":`) {
			var value event
			if json.Unmarshal([]byte(line), &value) != nil {
				t.Fatalf("bad event %q", line)
			}
			events = append(events, value)
			continue
		}
		var reply response
		if json.Unmarshal([]byte(line), &reply) != nil {
			t.Fatalf("bad response %q", line)
		}
		replies = append(replies, reply)
	}
	return replies, events
}

func TestProtocolScriptStepsOneEventAtATime(t *testing.T) {
	open := `{"op":"open","id":1,"version":3,"width":300,"height":200,"title":"t","headless":true,"script":[` +
		`{"event":"type","widget":2,"text":"Ali"},{"event":"click","widget":3},{"event":"toggle","widget":4},` +
		`{"event":"click","widget":4},{"event":"key","key":"Enter"},{"event":"click","widget":3}]}`
	replies, events := run(t, open,
		`{"op":"add","id":2,"kind":"column","spacing":8,"padding":12}`,
		`{"op":"add","id":3,"parent":1,"kind":"textInput","placeholder":"name"}`,
		`{"op":"add","id":4,"parent":1,"kind":"button","text":"Save"}`,
		`{"op":"add","id":5,"parent":1,"kind":"checkbox","text":"Paid"}`,
		`{"op":"listen","id":6,"kind":"click","widget":3}`,
		`{"op":"listen","id":7,"kind":"key"}`,
		`{"op":"wait","id":8}`,
		`{"op":"get","id":9,"widget":2}`,
		`{"op":"next","id":10}`,
		`{"op":"next","id":11}`,
		`{"op":"get","id":12,"widget":4}`,
		`{"op":"next","id":13}`,
		`{"op":"set","id":14,"widget":1,"kind":"text","text":"x"}`,
		`{"op":"close","id":15}`)
	if len(replies) != 15 {
		t.Fatalf("replies: %+v", replies)
	}
	for index, reply := range replies[:12] {
		if !reply.OK {
			t.Fatalf("reply %d: %+v", index, reply)
		}
	}
	if replies[1].Widget != 1 || replies[2].Widget != 2 || replies[3].Widget != 3 || replies[4].Widget != 4 {
		t.Fatalf("widget ids follow creation order: %+v", replies[1:5])
	}
	// The first step types "Ali" and stops at the first listened click.
	if *replies[8].Text != "Ali" {
		t.Fatalf("typed text at the first click: %q", *replies[8].Text)
	}
	if *replies[11].Checked != true {
		t.Fatal("the toggle step did not check the Checkbox")
	}
	if replies[13].OK || !replies[13].Closed {
		t.Fatalf("a change after the script ended was accepted: %+v", replies[13])
	}
	want := []string{"click:3", "key:Enter", "click:3", "closed"}
	if len(events) != len(want) {
		t.Fatalf("events: %+v", events)
	}
	for index, e := range events {
		got := e.Event
		if e.Event == "click" {
			got += ":" + string(rune('0'+e.Widget))
		}
		if e.Event == "key" {
			got += ":" + e.Key
		}
		if got != want[index] {
			t.Fatalf("event %d = %s, want %s", index, got, want[index])
		}
	}
	final := events[3].Values
	if len(final) != 2 || final[0].Text != "Ali" || !final[1].Checked {
		t.Fatalf("closed event values: %+v", final)
	}
	if len(replies[14].Values) != 2 {
		t.Fatalf("close response values: %+v", replies[14])
	}
}

func TestProtocolRejectsBadInput(t *testing.T) {
	for _, line := range []string{
		`{"op":"open","version":4,"width":10,"height":10}`,
		`{"op":"open","version":3,"width":0,"height":10}`,
		`{"op":"open","version":3,"width":10,"height":5000}`,
		`{"op":"open","version":3,"width":10,"height":10,"script":[{"event":"key","key":"A"}]}`,
		`{"op":"open","version":3,"width":10,"height":10,"headless":true,"script":[{"event":"key","key":"F1"}]}`,
		`{"op":"open","version":3,"width":10,"height":10,"headless":true,"script":[{"event":"closed"}]}`,
		`{"op":"add"}`,
	} {
		replies, _ := run(t, line)
		if len(replies) != 1 || replies[0].OK {
			t.Errorf("%s accepted: %+v", line, replies)
		}
	}
	open := `{"op":"open","id":1,"version":3,"width":100,"height":100,"headless":true}`
	huge := `{"op":"set","widget":1,"kind":"text","text":"` + strings.Repeat("a", maxRequestBytes) + `"}`
	replies, _ := run(t, open, `{"op":"add","id":2,"kind":"column"}`, `{"op":"unknown","id":3}`, `{"op":"listen","id":4,"kind":"wheel"}`,
		`{"op":"set","id":5,"widget":1,"kind":"text","text":"x"}`, `{"op":"set","id":6,"widget":9,"kind":"text","text":"x"}`, huge, `{"op":"status","id":7}`)
	if len(replies) != 7 || replies[2].OK || replies[3].OK || replies[4].OK || replies[5].OK || replies[6].Error != errRequestTooLarge.Error() {
		t.Fatalf("bad requests: %+v", replies)
	}
}

func TestKeyNamesAreTheDocumentedSet(t *testing.T) {
	want := []string{"ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Enter", "Escape", "Space", "Tab", "Backspace",
		"Delete", "Home", "End", "PageUp", "PageDown"}
	for letter := 'A'; letter <= 'Z'; letter++ {
		want = append(want, string(letter))
	}
	for digit := '0'; digit <= '9'; digit++ {
		want = append(want, string(digit))
	}
	names := map[string]bool{}
	for _, name := range keyNames {
		names[name] = true
	}
	for _, name := range want {
		if !names[name] {
			t.Errorf("key %q is not reported", name)
		}
		delete(names, name)
	}
	if len(names) != 0 {
		t.Errorf("undocumented key names: %v", names)
	}
}

func TestRenderDrawsWidgetsWithTheEmbeddedFont(t *testing.T) {
	m := newModel("t", 300, 120, measureText)
	root, _ := m.add(0, kindColumn, "", "", false, 8, 12)
	mustAdd(t, m, root, kindLabel, "Müşteri ĞÜŞİÖÇ")
	check, _ := m.add(root, kindCheckbox, "Paid", "", true, 0, 0)
	m.mu.Lock()
	img := render(frame{model: m, scale: 2})
	target := m.widgets[check]
	m.mu.Unlock()
	if img.Bounds().Dx() != 600 || img.Bounds().Dy() != 240 {
		t.Fatalf("image size %v at scale 2", img.Bounds())
	}
	textPixels := 0
	for y := 24; y < 88; y++ {
		for x := 24; x < 400; x++ {
			if c := img.RGBAAt(x, y); c.R < 100 && c.G < 100 && c.B < 100 {
				textPixels++
			}
		}
	}
	if textPixels < 200 {
		t.Fatalf("the label drew only %d dark pixels", textPixels)
	}
	// A checked Checkbox is a filled focus-colored square.
	center := img.RGBAAt(2*target.x+4, 2*target.y+2*rowHeight/2)
	if center != (color.RGBA{0, 110, 220, 255}) {
		t.Fatalf("checked box color %v", center)
	}
	if measureText("") != 0 || measureText("abc") <= 0 {
		t.Fatal("text measurement")
	}
}
