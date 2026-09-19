package main

import (
	"image/color"
	"testing"

	"golang.org/x/image/font"
)

// over is "source over" compositing of one 8-bit channel, the rule the
// renderer documents for translucent colors.
func over(source, alpha, destination uint8) uint8 {
	return uint8((uint32(source)*uint32(alpha) + uint32(destination)*(255-uint32(alpha)) + 127) / 255)
}

func near(t *testing.T, what string, have, want color.RGBA) {
	t.Helper()
	diff := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	if diff(have.R, want.R) > 1 || diff(have.G, want.G) > 1 || diff(have.B, want.B) > 1 || have.A != 255 {
		t.Fatalf("%s: have %v, want %v", what, have, want)
	}
}

func mustColor(t *testing.T, m *model, id int64, part, text string) {
	t.Helper()
	value, ok := parseColor(text)
	if !ok {
		t.Fatalf("parseColor(%q)", text)
	}
	if err := m.setColor(id, part, value); err != nil {
		t.Fatalf("setColor %d %s %s: %v", id, part, text, err)
	}
}

// styledForm is a Column with a Label, a Button, a TextInput, and a
// Checkbox, at fixed positions: root padding 10 and spacing 8.
func styledForm(t *testing.T) (m *model, root, label, button, input, check int64) {
	t.Helper()
	m = newModel("t", 400, 240, fixedMeasure)
	root, _ = m.add(0, kindColumn, "", "", false, 8, 10)
	label = mustAdd(t, m, root, kindLabel, "Label")
	button = mustAdd(t, m, root, kindButton, "Save")
	input, _ = m.add(root, kindTextInput, "", "", false, 0, 0)
	check = mustAdd(t, m, root, kindCheckbox, "Paid")
	return
}

func renderAt(m *model) func(x, y int) color.RGBA {
	m.mu.Lock()
	img := render(frame{model: m, scale: 1})
	m.mu.Unlock()
	return img.RGBAAt
}

func TestDefaultColorsAreTheV18Look(t *testing.T) {
	m, _, label, button, input, check := styledForm(t)
	pixel := renderAt(m)
	m.mu.Lock()
	lw, bw, iw, cw := *m.widgets[label], *m.widgets[button], *m.widgets[input], *m.widgets[check]
	m.mu.Unlock()
	near(t, "window", pixel(395, 235), colorBackground)
	near(t, "root padding", pixel(2, 2), colorBackground)
	near(t, "label box corner", pixel(lw.x+lw.w-1, lw.y+1), colorBackground)
	near(t, "button fill", pixel(bw.x+3, bw.y+3), colorButton)
	near(t, "button border", pixel(bw.x, bw.y+bw.h/2), colorBorder)
	near(t, "field fill", pixel(iw.x+3, iw.y+3), colorField)
	near(t, "checkbox caption area", pixel(cw.x+cw.w-1, cw.y+1), colorBackground)
}

func TestColorsRenderAtTheirWidgets(t *testing.T) {
	m, root, label, button, input, check := styledForm(t)
	mustColor(t, m, 0, partBackground, "#203040FF")
	mustColor(t, m, root, partBackground, "#FFFFFFFF")
	mustColor(t, m, label, partBackground, "#00FF00FF")
	mustColor(t, m, button, partBackground, "#0000FFFF")
	mustColor(t, m, input, partBackground, "#FFFF00FF")
	mustColor(t, m, check, partBackground, "#FF00FFFF")
	pixel := renderAt(m)
	m.mu.Lock()
	rw, lw, bw, iw, cw := *m.widgets[root], *m.widgets[label], *m.widgets[button], *m.widgets[input], *m.widgets[check]
	m.mu.Unlock()
	near(t, "window background outside the root", pixel(395, 235), color.RGBA{0x20, 0x30, 0x40, 255})
	near(t, "root background in its padding", pixel(rw.x+2, rw.y+2), color.RGBA{255, 255, 255, 255})
	near(t, "label background", pixel(lw.x+lw.w-1, lw.y+1), color.RGBA{0, 255, 0, 255})
	near(t, "button background", pixel(bw.x+3, bw.y+3), color.RGBA{0, 0, 255, 255})
	near(t, "field background", pixel(iw.x+3, iw.y+3), color.RGBA{255, 255, 0, 255})
	near(t, "checkbox background", pixel(cw.x+cw.w-1, cw.y+1), color.RGBA{255, 0, 255, 255})

	// A second color replaces the first.
	mustColor(t, m, button, partBackground, "#FF0000FF")
	near(t, "button background after a second update", renderAt(m)(bw.x+3, bw.y+3), color.RGBA{255, 0, 0, 255})
}

func TestTranslucentColorsBlendOverWhatIsBeneath(t *testing.T) {
	m, root, label, _, _, _ := styledForm(t)
	mustColor(t, m, 0, partBackground, "#FF000080")
	mustColor(t, m, label, partBackground, "#0000FF80")
	pixel := renderAt(m)
	m.mu.Lock()
	lw := *m.widgets[label]
	m.mu.Unlock()
	window := color.RGBA{over(255, 0x80, 242), over(0, 0x80, 242), over(0, 0x80, 242), 255}
	near(t, "window over the opaque default", pixel(395, 235), window)
	label50 := color.RGBA{over(0, 0x80, window.R), over(0, 0x80, window.G), over(255, 0x80, window.B), 255}
	near(t, "label over the window", pixel(lw.x+lw.w-1, lw.y+1), label50)
	_ = root
}

func TestForegroundColorsTheText(t *testing.T) {
	m := newModel("t", 400, 80, measureText)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	label := mustAdd(t, m, root, kindLabel, "MMMMMMMM")
	mustColor(t, m, label, partForeground, "#FF0000FF")
	m.mu.Lock()
	img := render(frame{model: m, scale: 1})
	m.mu.Unlock()
	red, dark := 0, 0
	for y := 0; y < rowHeight; y++ {
		for x := 0; x < 120; x++ {
			c := img.RGBAAt(x, y)
			if c.R > 200 && c.G < 80 && c.B < 80 {
				red++
			}
			if c.R < 80 && c.G < 80 && c.B < 80 {
				dark++
			}
		}
	}
	if red < 40 || dark != 0 {
		t.Fatalf("red text pixels %d, default dark pixels %d", red, dark)
	}
}

func TestColorTargetsAreChecked(t *testing.T) {
	m, root, label, _, _, _ := styledForm(t)
	value, _ := parseColor("#000000FF")
	if m.setColor(0, partForeground, value) == nil {
		t.Fatal("the Window accepted a foreground")
	}
	if m.setColor(root, partForeground, value) == nil {
		t.Fatal("a Container accepted a foreground")
	}
	if m.setColor(label, "border", value) == nil {
		t.Fatal("an unknown color part was accepted")
	}
	if m.setColor(999, partBackground, value) == nil {
		t.Fatal("a missing widget accepted a color")
	}
	for _, bad := range []string{"#fff", "#FFFFFF", "red", "#GG0000FF", "#FF0000FF0", ""} {
		if _, ok := parseColor(bad); ok {
			t.Fatalf("the helper accepted %q; the runtime always sends #RRGGBBAA", bad)
		}
	}
}

func TestDisabledWidgetsIgnoreTheUser(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	button := mustAdd(t, m, root, kindButton, "B")
	check := mustAdd(t, m, root, kindCheckbox, "C")
	input, _ := m.add(root, kindTextInput, "", "", false, 0, 0)
	other := mustAdd(t, m, root, kindButton, "O")
	// Button 0..32, Checkbox 32..64, TextInput 64..96, other Button 96..128.
	if !m.enabled(button) || !m.enabled(check) || !m.enabled(input) {
		t.Fatal("widgets start enabled")
	}
	m.press(5, 5)
	if m.focus != button {
		t.Fatal("focus before disabling")
	}
	for _, id := range []int64{button, check, input} {
		if err := m.setEnabled(id, false); err != nil {
			t.Fatal(err)
		}
	}
	if m.focus != 0 {
		t.Fatal("disabling the focused Button kept the focus")
	}
	m.press(5, 5)
	if m.focus != 0 || m.release(5, 5) != 0 {
		t.Fatal("a disabled Button took the focus or was clicked")
	}
	m.press(5, 40)
	m.release(5, 40)
	if _, checked, _ := m.read(check); checked {
		t.Fatal("clicking a disabled Checkbox toggled it")
	}
	if m.toggle(check) != nil {
		t.Fatal("scripted toggle")
	}
	if _, checked, _ := m.read(check); checked {
		t.Fatal("a scripted toggle changed a disabled Checkbox")
	}
	m.press(5, 70)
	if m.focus != 0 || m.insert([]rune("x")) {
		t.Fatal("a disabled TextInput took the focus or text")
	}
	if m.typeInto(input, "typed") != nil {
		t.Fatal("scripted typing")
	}
	if text, _, _ := m.read(input); text != "" {
		t.Fatalf("a disabled TextInput received %q", text)
	}
	// Tab skips disabled widgets.
	m.focus = 0
	m.focusNext()
	if m.focus != other {
		t.Fatalf("Tab reached %d, not the enabled Button %d", m.focus, other)
	}
	m.focusNext()
	if m.focus != other {
		t.Fatal("Tab left the only enabled widget")
	}
	// The program can still change disabled widgets.
	if m.setText(button, "Renamed") != nil || m.setChecked(check, true) != nil || m.setText(input, "set") != nil {
		t.Fatal("programmatic changes to disabled widgets")
	}
	if text, _, _ := m.read(input); text != "set" {
		t.Fatalf("setText on a disabled TextInput gave %q", text)
	}
	// Re-enabling restores everything.
	for _, id := range []int64{button, check, input} {
		if err := m.setEnabled(id, true); err != nil {
			t.Fatal(err)
		}
	}
	m.press(5, 5)
	if m.focus != button || m.release(5, 5) != button {
		t.Fatal("a re-enabled Button")
	}
	if m.activateFocused(true) != button {
		t.Fatal("Space on a re-enabled Button")
	}
	m.press(5, 40)
	m.release(5, 40)
	if _, checked, _ := m.read(check); checked {
		t.Fatal("clicking the re-enabled checked Checkbox did not uncheck it")
	}
	m.press(5, 70)
	if !m.insert([]rune("!")) {
		t.Fatal("a re-enabled TextInput")
	}
	if text, _, _ := m.read(input); text != "set!" {
		t.Fatalf("typing into a re-enabled TextInput gave %q", text)
	}
	if m.setEnabled(root, false) == nil {
		t.Fatal("a Container accepted enabled")
	}
}

func TestDisabledKeyboardActivationIsIgnored(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root, _ := m.add(0, kindColumn, "", "", false, 0, 0)
	button := mustAdd(t, m, root, kindButton, "B")
	m.focus = button
	m.widgets[button].disabled = true
	if m.activateFocused(false) != 0 || m.activateFocused(true) != 0 {
		t.Fatal("Enter or Space activated a disabled Button")
	}
}

func TestDisabledWidgetsAreFaded(t *testing.T) {
	m, _, _, button, _, _ := styledForm(t)
	if err := m.setEnabled(button, false); err != nil {
		t.Fatal(err)
	}
	pixel := renderAt(m)
	m.mu.Lock()
	bw := *m.widgets[button]
	m.mu.Unlock()
	want := color.RGBA{over(242, 153, colorButton.R), over(242, 153, colorButton.G), over(242, 153, colorButton.B), 255}
	near(t, "faded disabled Button", pixel(bw.x+3, bw.y+3), want)
}

func TestProtocolColorsAndEnabled(t *testing.T) {
	open := `{"op":"open","id":1,"version":3,"width":300,"height":200,"title":"t","headless":true}`
	replies, _ := run(t, open,
		`{"op":"add","id":2,"kind":"column"}`,
		`{"op":"add","id":3,"parent":1,"kind":"button","text":"B"}`,
		`{"op":"color","id":4,"widget":2,"part":"background","color":"#112233FF"}`,
		`{"op":"color","id":5,"widget":0,"part":"background","color":"#445566"}`,
		`{"op":"color","id":6,"widget":1,"part":"foreground","color":"#112233FF"}`,
		`{"op":"enabled","id":7,"widget":2,"enabled":false}`,
		`{"op":"enabled","id":8,"widget":2}`,
		`{"op":"enabled","id":9,"widget":1,"enabled":false}`,
		`{"op":"close","id":10}`,
		`{"op":"color","id":11,"widget":2,"part":"background","color":"#112233FF"}`)
	want := map[int64]bool{1: true, 2: true, 3: true, 4: true, 5: false, 6: false, 7: true, 8: false, 9: false, 10: true}
	for _, reply := range replies {
		if expected, ok := want[reply.ID]; ok && reply.OK != expected {
			t.Fatalf("reply %d ok=%v error=%q", reply.ID, reply.OK, reply.Error)
		}
	}
}

// Text laid out at scale 1 must not be clipped when drawn at a higher scale.
func TestTextFitsItsBoxAtEveryScale(t *testing.T) {
	for _, text := range []string{"New order", "Type a name, accept the terms, then press Check.", "Müşteri ĞÜŞİÖÇ", "Ready"} {
		width := measureText(text)
		for _, scale := range []float64{1, 1.25, 1.5, 2, 3} {
			drawn := font.MeasureString(faceAt(scale), text).Ceil()
			if float64(drawn) > float64(width)*scale+1 {
				t.Fatalf("%q at scale %v is %d device pixels wide; its box is %v", text, scale, drawn, float64(width)*scale)
			}
		}
	}
}
