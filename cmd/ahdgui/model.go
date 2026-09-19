package main

import (
	"errors"
	"image/color"
	"sync"
	"unicode/utf8"
)

// Widget kinds. column and row are Containers; the others are leaves.
const (
	kindColumn    = "column"
	kindRow       = "row"
	kindLabel     = "label"
	kindButton    = "button"
	kindTextInput = "textInput"
	kindCheckbox  = "checkbox"
)

// Fixed metrics in logical pixels. One unit is one window point; on a
// high-density display everything is drawn at the display's scale.
const (
	fontSize        = 15.0
	rowHeight       = 32 // every leaf widget is one row tall
	buttonPadding   = 14 // horizontal space around a Button's text
	minButtonWidth  = 64
	textInputWidth  = 260
	textInputInset  = 8
	checkboxBox     = 18
	checkboxGap     = 8
	windowMargin    = 0
	caretBlinkTicks = 15
)

type widget struct {
	id       int64
	kind     string
	parent   int64
	children []int64

	text        []rune // label, button, checkbox caption, or TextInput value
	placeholder string
	checked     bool
	spacing     int
	padding     int

	caret  int // TextInput caret position in runes
	scroll int // TextInput horizontal scroll in logical pixels

	// foreground and background are the program's colors; nil keeps the
	// default look. disabled Buttons, TextInputs, and Checkboxes ignore the
	// user and cannot take the focus.
	foreground *color.NRGBA
	background *color.NRGBA
	disabled   bool

	// Layout, in logical pixels, relative to the window's top-left.
	x, y, w, h int
}

func (w *widget) container() bool { return w.kind == kindColumn || w.kind == kindRow }

// interactive reports whether the widget can be enabled and disabled.
func (w *widget) interactive() bool {
	return w.kind == kindButton || w.kind == kindTextInput || w.kind == kindCheckbox
}

// focusable reports whether the widget can take the focus: an enabled
// Button, TextInput, or Checkbox.
func (w *widget) focusable() bool {
	return w.interactive() && !w.disabled
}

// model is the whole window state. mu guards it: the protocol goroutine
// changes it on requests and the window goroutine on user input and drawing.
type model struct {
	mu      sync.Mutex
	title   string
	width   int
	height  int
	widgets map[int64]*widget
	order   []int64 // creation order, which is also focus (Tab) order
	root    int64
	next    int64
	focus   int64
	pressed int64
	dirty   bool
	measure func(text string) int
	// background is the Window's color; nil keeps the default.
	background *color.NRGBA
}

func newModel(title string, width, height int, measure func(string) int) *model {
	return &model{title: title, width: width, height: height, widgets: map[int64]*widget{}, measure: measure, dirty: true}
}

var (
	errSecondRoot   = errors.New("the Window already has its root Container")
	errNoParent     = errors.New("the parent Container does not exist")
	errNotContainer = errors.New("the parent is not a Container")
	errTooMany      = errors.New("the Window has too many widgets")
	errNoWidget     = errors.New("the widget does not exist")
	errWrongKind    = errors.New("the widget does not support that request")
)

// add creates a widget; parent 0 creates the root Container.
func (m *model) add(parent int64, kind, text, placeholder string, checked bool, spacing, padding int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.widgets) >= maxWidgets {
		return 0, errTooMany
	}
	switch kind {
	case kindColumn, kindRow, kindLabel, kindButton, kindTextInput, kindCheckbox:
	default:
		return 0, errors.New("unknown widget kind " + kind)
	}
	if !validText(text, maxTextRunes) || !validText(placeholder, maxTextRunes) {
		return 0, errors.New("invalid widget text")
	}
	if spacing < 0 || padding < 0 || spacing > maxSpacing || padding > maxSpacing {
		return 0, errors.New("spacing and padding must be between 0 and 1000")
	}
	if parent == 0 {
		if m.root != 0 {
			return 0, errSecondRoot
		}
		if kind != kindColumn && kind != kindRow {
			return 0, errNotContainer
		}
	} else {
		owner := m.widgets[parent]
		if owner == nil {
			return 0, errNoParent
		}
		if !owner.container() {
			return 0, errNotContainer
		}
	}
	m.next++
	created := &widget{id: m.next, kind: kind, parent: parent, text: []rune(text), placeholder: placeholder,
		checked: checked, spacing: spacing, padding: padding}
	created.caret = len(created.text)
	m.widgets[created.id] = created
	m.order = append(m.order, created.id)
	if parent == 0 {
		m.root = created.id
	} else {
		m.widgets[parent].children = append(m.widgets[parent].children, created.id)
	}
	m.dirty = true
	return created.id, nil
}

func (m *model) setText(id int64, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.container() {
		return errWrongKind
	}
	if !validText(text, maxTextRunes) {
		return errors.New("invalid widget text")
	}
	target.text = []rune(text)
	target.caret, target.scroll = len(target.text), 0
	m.dirty = true
	return nil
}

func (m *model) setChecked(id int64, checked bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindCheckbox {
		return errWrongKind
	}
	target.checked = checked
	m.dirty = true
	return nil
}

func (m *model) read(id int64) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return "", false, errNoWidget
	}
	return string(target.text), target.checked, nil
}

// values reads every TextInput and Checkbox, in creation order.
func (m *model) values() []value {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []value
	for _, id := range m.order {
		target := m.widgets[id]
		switch target.kind {
		case kindTextInput:
			result = append(result, value{Widget: id, Text: string(target.text)})
		case kindCheckbox:
			result = append(result, value{Widget: id, Checked: target.checked})
		}
	}
	return result
}

// Color parts.
const (
	partForeground = "foreground"
	partBackground = "background"
)

// setColor sets one color. Widget 0 is the Window, which has a background
// only; a Container has a background only; every other widget has both.
func (m *model) setColor(id int64, part string, value color.NRGBA) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if part != partForeground && part != partBackground {
		return errors.New("unknown color part " + part)
	}
	if id == 0 {
		if part != partBackground {
			return errWrongKind
		}
		m.background = &value
		m.dirty = true
		return nil
	}
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.container() && part != partBackground {
		return errWrongKind
	}
	if part == partForeground {
		target.foreground = &value
	} else {
		target.background = &value
	}
	m.dirty = true
	return nil
}

// setEnabled enables or disables a Button, TextInput, or Checkbox. A widget
// that is disabled while it has the focus loses it.
func (m *model) setEnabled(id int64, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if !target.interactive() {
		return errWrongKind
	}
	target.disabled = !enabled
	if target.disabled {
		if m.focus == id {
			m.focus = 0
		}
		if m.pressed == id {
			m.pressed = 0
		}
	}
	m.dirty = true
	return nil
}

// enabled reports whether a widget accepts user input.
func (m *model) enabled(id int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	return target != nil && !target.disabled
}

func (m *model) setTitle(title string) {
	m.mu.Lock()
	m.title = title
	m.mu.Unlock()
}

// ---- layout --------------------------------------------------------------

// layout places every widget. A Column stacks its children top to bottom and
// a Row places them left to right, each separated by the Container's
// spacing, inside its padding. Children keep their natural size: a Column's
// children are left-aligned and a Row's children are centered vertically.
// The root Container starts at the window's top-left corner; what does not
// fit in the window is clipped. The caller holds m.mu.
func (m *model) layout() {
	if m.root == 0 {
		return
	}
	m.measureWidget(m.root)
	m.place(m.root, windowMargin, windowMargin)
}

func (m *model) measureWidget(id int64) (int, int) {
	target := m.widgets[id]
	switch target.kind {
	case kindLabel:
		target.w, target.h = m.measure(string(target.text)), rowHeight
	case kindButton:
		target.w, target.h = max(m.measure(string(target.text))+2*buttonPadding, minButtonWidth), rowHeight
	case kindTextInput:
		target.w, target.h = textInputWidth, rowHeight
	case kindCheckbox:
		target.w, target.h = checkboxBox+checkboxGap+m.measure(string(target.text)), rowHeight
	case kindColumn, kindRow:
		width, height := 0, 0
		for index, child := range target.children {
			childWidth, childHeight := m.measureWidget(child)
			gap := 0
			if index > 0 {
				gap = target.spacing
			}
			if target.kind == kindColumn {
				width, height = max(width, childWidth), height+gap+childHeight
			} else {
				width, height = width+gap+childWidth, max(height, childHeight)
			}
		}
		target.w, target.h = width+2*target.padding, height+2*target.padding
	}
	return target.w, target.h
}

func (m *model) place(id int64, x, y int) {
	target := m.widgets[id]
	target.x, target.y = x, y
	if !target.container() {
		return
	}
	cursorX, cursorY := x+target.padding, y+target.padding
	inner := target.h - 2*target.padding
	for _, child := range target.children {
		placed := m.widgets[child]
		if target.kind == kindColumn {
			m.place(child, cursorX, cursorY)
			cursorY += placed.h + target.spacing
		} else {
			m.place(child, cursorX, cursorY+(inner-placed.h)/2)
			cursorX += placed.w + target.spacing
		}
	}
}

// hit finds the leaf widget at a logical point, or 0.
func (m *model) hit(x, y int) int64 {
	for index := len(m.order) - 1; index >= 0; index-- {
		target := m.widgets[m.order[index]]
		if !target.container() && x >= target.x && y >= target.y && x < target.x+target.w && y < target.y+target.h {
			return target.id
		}
	}
	return 0
}

// ---- input ---------------------------------------------------------------

// press starts a click at a logical point and moves the focus there.
func (m *model) press(x, y int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.layout()
	id := m.hit(x, y)
	m.pressed = 0
	m.focus = 0
	if id != 0 && m.widgets[id].focusable() {
		m.pressed, m.focus = id, id
	}
	m.dirty = true
}

// release ends a click. A click is a press and a release on the same Button
// or Checkbox; it returns the Button clicked, and toggles a Checkbox.
func (m *model) release(x, y int) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.layout()
	id := m.hit(x, y)
	pressed := m.pressed
	m.pressed = 0
	m.dirty = true
	if id == 0 || id != pressed {
		return 0
	}
	return m.activate(id)
}

// activate clicks a Button (returned) or toggles a Checkbox. The caller
// holds m.mu.
func (m *model) activate(id int64) int64 {
	target := m.widgets[id]
	switch target.kind {
	case kindButton:
		return id
	case kindCheckbox:
		target.checked = !target.checked
		m.dirty = true
	}
	return 0
}

// focusNext moves the focus to the next enabled Button, TextInput, or
// Checkbox in creation order, wrapping around; disabled widgets are skipped.
func (m *model) focusNext() {
	m.mu.Lock()
	defer m.mu.Unlock()
	start := -1
	for index, id := range m.order {
		if id == m.focus {
			start = index
		}
	}
	for step := 1; step <= len(m.order); step++ {
		id := m.order[(start+step+len(m.order))%len(m.order)]
		if m.widgets[id].focusable() {
			m.focus = id
			m.dirty = true
			return
		}
	}
}

// activateFocused handles Enter or Space on the focused widget: a Button is
// clicked (returned) and a Checkbox toggled by Space. A TextInput keeps the
// key for text editing.
func (m *model) activateFocused(space bool) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[m.focus]
	if target == nil || target.disabled {
		return 0
	}
	switch {
	case target.kind == kindButton:
		return m.activate(target.id)
	case target.kind == kindCheckbox && space:
		return m.activate(target.id)
	}
	return 0
}

// focusedInput is the focused TextInput, or nil. The caller holds m.mu.
func (m *model) focusedInput() *widget {
	target := m.widgets[m.focus]
	if target == nil || target.kind != kindTextInput || target.disabled {
		return nil
	}
	return target
}

// Editing operations on the focused TextInput. Each reports whether the
// focus is on a TextInput.

func (m *model) insert(runes []rune) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.focusedInput()
	if target == nil {
		return false
	}
	for _, character := range runes {
		if character < 0x20 || character == 0x7f || !utf8.ValidRune(character) || len(target.text) >= maxTextRunes {
			continue
		}
		target.text = append(target.text[:target.caret], append([]rune{character}, target.text[target.caret:]...)...)
		target.caret++
	}
	m.dirty = true
	return true
}

func (m *model) edit(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.focusedInput()
	if target == nil {
		return false
	}
	switch key {
	case "Backspace":
		if target.caret > 0 {
			target.text = append(target.text[:target.caret-1], target.text[target.caret:]...)
			target.caret--
		}
	case "Delete":
		if target.caret < len(target.text) {
			target.text = append(target.text[:target.caret], target.text[target.caret+1:]...)
		}
	case "ArrowLeft":
		target.caret = max(target.caret-1, 0)
	case "ArrowRight":
		target.caret = min(target.caret+1, len(target.text))
	case "Home":
		target.caret = 0
	case "End":
		target.caret = len(target.text)
	default:
		return true
	}
	m.dirty = true
	return true
}

// typeInto appends text to a TextInput as if typed, for headless scripts.
func (m *model) typeInto(id int64, text string) error {
	m.mu.Lock()
	target := m.widgets[id]
	if target == nil || target.kind != kindTextInput {
		m.mu.Unlock()
		return errWrongKind
	}
	if target.disabled {
		// A disabled TextInput ignores the user, scripted or not.
		m.mu.Unlock()
		return nil
	}
	previous := m.focus
	m.focus = id
	m.mu.Unlock()
	m.insert([]rune(text))
	m.mu.Lock()
	m.focus = previous
	m.mu.Unlock()
	return nil
}

// toggle clicks a Checkbox, for headless scripts.
func (m *model) toggle(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil || target.kind != kindCheckbox {
		return errWrongKind
	}
	if !target.disabled {
		m.activate(id)
	}
	return nil
}
