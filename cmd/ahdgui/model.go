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
	kindPassword  = "passwordInput"
	kindTextArea  = "textArea"
	kindListBox   = "listBox"
	kindSelect    = "select"
	kindTable     = "table"
)

// Fixed metrics in logical pixels. One unit is one window point; on a
// high-density display everything is drawn at the display's scale.
const (
	fontSize        = 15.0
	rowHeight       = 32 // every single-line widget is one row tall
	buttonPadding   = 14 // horizontal space around a Button's text
	minButtonWidth  = 64
	textInputWidth  = 260
	textInputInset  = 8
	checkboxBox     = 18
	checkboxGap     = 8
	windowMargin    = 0
	caretBlinkTicks = 15

	// itemHeight is one ListBox item, Select choice, or TableView row;
	// headerHeight is a TableView's header; lineHeight one TextArea line.
	itemHeight   = 24
	headerHeight = 28
	lineHeight   = 20
	itemInset    = 8
	scrollbar    = 10 // scroll bar width

	// Natural sizes of the scrolling widgets; a resizable Window gives them
	// its extra space.
	listBoxWidth   = 260
	listBoxRows    = 6
	textAreaWidth  = 360
	textAreaLines  = 6
	tableRows      = 8
	tableMinWidth  = 200
	tableMaxWidth  = 640
	minColumnWidth = 48
	maxColumnWidth = 320
	// measuredRows bounds the rows whose cells decide column widths, so
	// setRows stays fast for large tables; the widths stay deterministic.
	measuredRows = 1000
	// selectVisible is how many Select choices the open list shows at once.
	selectVisible  = 8
	selectMinWidth = 200
	selectMaxWidth = 480
	selectArrow    = 28
	pageItems      = 6 // PageUp and PageDown in lists and TextAreas
)

type widget struct {
	id       int64
	kind     string
	parent   int64
	children []int64

	text        []rune // label, button, checkbox caption, or text field value
	placeholder string
	checked     bool
	spacing     int
	padding     int

	caret  int // text field caret position in runes
	scroll int // text field horizontal scroll in device pixels

	// items are a ListBox's or Select's texts; columns, rows, and widths a
	// TableView's header, cells, and column widths. selected is the chosen
	// item or row, -1 for none.
	items    []string
	columns  []string
	rows     [][]string
	widths   []int
	selected int
	// scrollX and scrollY scroll a ListBox, TableView, or TextArea, in
	// logical pixels; highlight is, while a Select's list is open, the choice the keyboard points at and listScroll the list's scroll.
	scrollX    int
	scrollY    int
	highlight  int
	listScroll int

	// foreground and background are the program's colors; nil keeps the
	// default look. disabled widgets ignore the user and cannot take the
	// focus.
	foreground *color.NRGBA
	background *color.NRGBA
	disabled   bool

	// Layout, in logical pixels, relative to the window's top-left. natW
	// and natH are the natural size; grows marks a scrolling widget, or a
	// Container holding one, which takes a resizable Window's extra space.
	x, y, w, h int
	natW, natH int
	grows      bool
}

func (w *widget) container() bool { return w.kind == kindColumn || w.kind == kindRow }

// textField reports whether the widget edits text on one line.
func (w *widget) textField() bool { return w.kind == kindTextInput || w.kind == kindPassword }

// scrolling reports whether the widget scrolls its own content.
func (w *widget) scrolling() bool {
	return w.kind == kindListBox || w.kind == kindTable || w.kind == kindTextArea
}

// interactive reports whether the widget can be enabled and disabled.
func (w *widget) interactive() bool {
	switch w.kind {
	case kindButton, kindTextInput, kindCheckbox, kindPassword, kindTextArea, kindListBox, kindSelect, kindTable:
		return true
	}
	return false
}

// focusable reports whether the widget can take the focus: an enabled
// interactive widget.
func (w *widget) focusable() bool {
	return w.interactive() && !w.disabled
}

// count is the number of items or rows a selecting widget offers.
func (w *widget) count() int {
	if w.kind == kindTable {
		return len(w.rows)
	}
	return len(w.items)
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
	// resizable lets the user resize the window.
	resizable bool
	// scale is the display scale of the last drawing; a TextArea's
	// horizontal scroll is in its device pixels.
	scale float64
	// popup is the Select whose list is open, or 0; dragging is the widget
	// whose scroll bar is being dragged, grabbed at grabY within the thumb.
	popup    int64
	list     popupBox
	dragging int64
	grabY    int
	// changes are the widgets the user changed since the last takeChanges,
	// in order, each once: the session sends their change events.
	changes []int64
}

func newModel(title string, width, height int, measure func(string) int) *model {
	return &model{title: title, width: width, height: height, widgets: map[int64]*widget{}, measure: measure, dirty: true, scale: 1}
}

var (
	errSecondRoot   = errors.New("the Window already has its root Container")
	errNoParent     = errors.New("the parent Container does not exist")
	errNotContainer = errors.New("the parent is not a Container")
	errTooMany      = errors.New("the Window has too many widgets")
	errNoWidget     = errors.New("the widget does not exist")
	errWrongKind    = errors.New("the widget does not support that request")
	errIndex        = errors.New("the index is outside the items")
)

// spec is everything an add request can carry.
type spec struct {
	kind, text, placeholder string
	checked                 bool
	spacing, padding        int
	items, columns          []string
	rows                    [][]string
	selected                int
}

// add creates a widget without items; parent 0 creates the root Container.
func (m *model) add(parent int64, kind, text, placeholder string, checked bool, spacing, padding int) (int64, error) {
	return m.addSpec(parent, spec{kind: kind, text: text, placeholder: placeholder, checked: checked,
		spacing: spacing, padding: padding, selected: -1})
}

// addSpec creates a widget; parent 0 creates the root Container.
func (m *model) addSpec(parent int64, s spec) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.widgets) >= maxWidgets {
		return 0, errTooMany
	}
	switch s.kind {
	case kindColumn, kindRow, kindLabel, kindButton, kindTextInput, kindCheckbox,
		kindPassword, kindTextArea, kindListBox, kindSelect, kindTable:
	default:
		return 0, errors.New("unknown widget kind " + s.kind)
	}
	valid := validText(s.text, maxTextRunes)
	if s.kind == kindTextArea {
		valid = validAreaText(s.text, maxAreaRunes)
	}
	if !valid || !validText(s.placeholder, maxTextRunes) {
		return 0, errors.New("invalid widget text")
	}
	if s.spacing < 0 || s.padding < 0 || s.spacing > maxSpacing || s.padding > maxSpacing {
		return 0, errors.New("spacing and padding must be between 0 and 1000")
	}
	if err := checkItems(s.items); err != nil {
		return 0, err
	}
	if s.kind == kindTable {
		if err := checkTable(s.columns, s.rows); err != nil {
			return 0, err
		}
	}
	created := &widget{kind: s.kind, parent: parent, text: []rune(s.text), placeholder: s.placeholder,
		checked: s.checked, spacing: s.spacing, padding: s.padding, selected: -1, highlight: -1}
	switch s.kind {
	case kindListBox, kindSelect:
		created.items = append([]string(nil), s.items...)
		if s.selected < -1 || s.selected >= len(created.items) {
			return 0, errIndex
		}
		created.selected = s.selected
	case kindTable:
		created.columns = append([]string(nil), s.columns...)
		created.rows = copyRows(s.rows)
		created.widths = m.columnWidths(created.columns, created.rows)
	}
	if parent == 0 {
		if m.root != 0 {
			return 0, errSecondRoot
		}
		if !created.container() {
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
	created.id = m.next
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

func checkItems(items []string) error {
	if len(items) > maxItems {
		return errors.New("too many items")
	}
	for _, item := range items {
		if !validText(item, maxItemRunes) {
			return errors.New("invalid item text")
		}
	}
	return nil
}

func checkTable(columns []string, rows [][]string) error {
	if len(columns) == 0 || len(columns) > maxColumns {
		return errors.New("a TableView needs between 1 and 64 columns")
	}
	if len(rows) > maxRows || len(rows)*len(columns) > maxCells {
		return errors.New("too many rows")
	}
	for _, column := range columns {
		if !validText(column, maxCellRunes) {
			return errors.New("invalid column text")
		}
	}
	for _, row := range rows {
		if len(row) != len(columns) {
			return errors.New("every row needs one cell per column")
		}
		for _, cell := range row {
			if !validText(cell, maxCellRunes) {
				return errors.New("invalid cell text")
			}
		}
	}
	return nil
}

func copyRows(rows [][]string) [][]string {
	copied := make([][]string, len(rows))
	for index, row := range rows {
		copied[index] = append([]string(nil), row...)
	}
	return copied
}

// columnWidths is each column's width: its widest header or cell among the
// first measuredRows rows, plus insets, kept between minColumnWidth and
// maxColumnWidth. The caller holds m.mu.
func (m *model) columnWidths(columns []string, rows [][]string) []int {
	widths := make([]int, len(columns))
	for index, column := range columns {
		widths[index] = m.measure(column)
	}
	for _, row := range rows[:min(len(rows), measuredRows)] {
		for index, cell := range row {
			if len(cell) > 0 {
				widths[index] = max(widths[index], m.measure(cell))
			}
		}
	}
	for index := range widths {
		widths[index] = min(max(widths[index]+2*itemInset, minColumnWidth), maxColumnWidth)
	}
	return widths
}

func (m *model) setText(id int64, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	switch target.kind {
	case kindLabel, kindButton, kindCheckbox, kindTextInput, kindPassword:
		if !validText(text, maxTextRunes) {
			return errors.New("invalid widget text")
		}
	case kindTextArea:
		if !validAreaText(text, maxAreaRunes) {
			return errors.New("invalid widget text")
		}
	default:
		return errWrongKind
	}
	target.text = []rune(text)
	target.caret, target.scroll, target.scrollY = len(target.text), 0, 0
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

// setItems replaces a ListBox's or Select's items. A selection that no
// longer exists is cleared.
func (m *model) setItems(id int64, items []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindListBox && target.kind != kindSelect {
		return errWrongKind
	}
	if err := checkItems(items); err != nil {
		return err
	}
	target.items = append([]string(nil), items...)
	m.afterReplace(target)
	return nil
}

// setRows replaces a TableView's rows, keeping its columns.
func (m *model) setRows(id int64, rows [][]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindTable {
		return errWrongKind
	}
	if err := checkTable(target.columns, rows); err != nil {
		return err
	}
	target.rows = copyRows(rows)
	target.widths = m.columnWidths(target.columns, target.rows)
	m.afterReplace(target)
	return nil
}

// afterReplace keeps a selection only while it still exists and closes an
// open list. The caller holds m.mu.
func (m *model) afterReplace(target *widget) {
	if target.selected >= target.count() {
		target.selected = -1
	}
	target.highlight = -1
	if m.popup == target.id {
		m.closePopup()
	}
	m.clampScroll(target)
	m.dirty = true
}

// selectIndex selects an item or row for the program, -1 for none. It never
// counts as a change by the user.
func (m *model) selectIndex(id int64, index int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindListBox && target.kind != kindSelect && target.kind != kindTable {
		return errWrongKind
	}
	if index < -1 || index >= target.count() {
		return errIndex
	}
	target.selected = index
	m.reveal(target)
	m.dirty = true
	return nil
}

func (m *model) read(id int64) (string, bool, error) {
	text, checked, _, err := m.readAll(id)
	return text, checked, err
}

// readAll reads a widget's text, check state, and selection.
func (m *model) readAll(id int64) (string, bool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return "", false, -1, errNoWidget
	}
	return string(target.text), target.checked, target.selected, nil
}

// valueOf is one widget's reading. The caller holds m.mu.
func valueOf(target *widget) (value, bool) {
	switch target.kind {
	case kindTextInput, kindPassword, kindTextArea:
		return value{Widget: target.id, Text: string(target.text)}, true
	case kindCheckbox:
		return value{Widget: target.id, Checked: target.checked}, true
	case kindListBox, kindSelect, kindTable:
		selected := target.selected
		return value{Widget: target.id, Selected: &selected}, true
	}
	return value{}, false
}

// values reads every widget the user can change, in creation order.
func (m *model) values() []value {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []value
	for _, id := range m.order {
		if reading, ok := valueOf(m.widgets[id]); ok {
			result = append(result, reading)
		}
	}
	return result
}

// valueNow reads one widget, for a change event.
func (m *model) valueNow(id int64) (value, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return value{}, false
	}
	return valueOf(target)
}

// noteChange records a change the user made. The caller holds m.mu.
func (m *model) noteChange(id int64) {
	for _, known := range m.changes {
		if known == id {
			return
		}
	}
	m.changes = append(m.changes, id)
}

// takeChanges returns and forgets the widgets the user changed.
func (m *model) takeChanges() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	changes := m.changes
	m.changes = nil
	return changes
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

// setEnabled enables or disables an interactive widget. A widget that is
// disabled while it has the focus loses it, and an open Select closes.
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
		if m.popup == id {
			m.closePopup()
		}
		if m.dragging == id {
			m.dragging = 0
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

// setResizable lets the user resize the window, or stops it.
func (m *model) setResizable(resizable bool) {
	m.mu.Lock()
	m.resizable = resizable
	m.mu.Unlock()
}

func (m *model) isResizable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resizable
}

// resize gives the window a new size in logical pixels.
func (m *model) resize(width, height int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	width, height = min(max(width, 1), maxDimension), min(max(height, 1), maxDimension)
	if width != m.width || height != m.height {
		m.width, m.height = width, height
		m.closePopup()
		m.dirty = true
	}
}

// ---- layout --------------------------------------------------------------

// layout places every widget. A Column stacks its children top to bottom and
// a Row places them left to right, each separated by the Container's
// spacing, inside its padding. Children keep their natural size: a Column's
// children are left-aligned and a Row's children are centered vertically.
// The root Container starts at the window's top-left corner; what does not
// fit in the window is clipped.
//
// A ListBox, TableView, or TextArea, and every Container holding one, grows:
// when the window is larger than the content, the extra height of a Column
// (and the extra width of a Row) is shared equally by its growing children,
// the first ones taking any remainder, and a growing child also fills the
// Container's full width in a Column (height in a Row). Nothing else ever
// stretches, and nothing shrinks below its natural size. The caller holds
// m.mu.
func (m *model) layout() {
	if m.root == 0 {
		return
	}
	root := m.widgets[m.root]
	m.measureWidget(m.root)
	width, height := root.natW, root.natH
	if root.grows {
		width, height = max(width, m.width-2*windowMargin), max(height, m.height-2*windowMargin)
	}
	m.place(m.root, windowMargin, windowMargin, width, height)
	if m.popup != 0 {
		m.placePopup(m.widgets[m.popup])
	}
}

func (m *model) measureWidget(id int64) (int, int) {
	target := m.widgets[id]
	target.grows = target.scrolling()
	switch target.kind {
	case kindLabel:
		target.natW, target.natH = m.measure(string(target.text)), rowHeight
	case kindButton:
		target.natW, target.natH = max(m.measure(string(target.text))+2*buttonPadding, minButtonWidth), rowHeight
	case kindTextInput, kindPassword:
		target.natW, target.natH = textInputWidth, rowHeight
	case kindCheckbox:
		target.natW, target.natH = checkboxBox+checkboxGap+m.measure(string(target.text)), rowHeight
	case kindTextArea:
		target.natW, target.natH = textAreaWidth, textAreaLines*lineHeight+2*textInputInset
	case kindListBox:
		target.natW, target.natH = listBoxWidth, listBoxRows*itemHeight+2
	case kindSelect:
		widest := m.measure(target.placeholder)
		for _, item := range target.items[:min(len(target.items), measuredRows)] {
			widest = max(widest, m.measure(item))
		}
		target.natW = min(max(widest+2*itemInset+selectArrow, selectMinWidth), selectMaxWidth)
		target.natH = rowHeight
	case kindTable:
		total := 0
		for _, width := range target.widths {
			total += width
		}
		target.natW = min(max(total+2+scrollbar, tableMinWidth), tableMaxWidth)
		target.natH = headerHeight + tableRows*itemHeight + 2
	case kindColumn, kindRow:
		width, height := 0, 0
		for index, child := range target.children {
			childWidth, childHeight := m.measureWidget(child)
			if m.widgets[child].grows {
				target.grows = true
			}
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
		target.natW, target.natH = width+2*target.padding, height+2*target.padding
	}
	return target.natW, target.natH
}

// place gives a widget its position and size, and lays out a Container's
// children inside it.
func (m *model) place(id int64, x, y, width, height int) {
	target := m.widgets[id]
	target.x, target.y, target.w, target.h = x, y, width, height
	if !target.container() {
		m.clampScroll(target)
		return
	}
	innerW, innerH := width-2*target.padding, height-2*target.padding
	growing, used := 0, 0
	for index, child := range target.children {
		placed := m.widgets[child]
		if placed.grows {
			growing++
		}
		if index > 0 {
			used += target.spacing
		}
		if target.kind == kindColumn {
			used += placed.natH
		} else {
			used += placed.natW
		}
	}
	extra := innerH - used
	if target.kind == kindRow {
		extra = innerW - used
	}
	extra = max(extra, 0)
	share := func(index int) int {
		if growing == 0 {
			return 0
		}
		portion := extra / growing
		if index < extra%growing {
			portion++
		}
		return portion
	}
	cursorX, cursorY := x+target.padding, y+target.padding
	growingIndex := 0
	for _, child := range target.children {
		placed := m.widgets[child]
		childW, childH := placed.natW, placed.natH
		if placed.grows {
			if target.kind == kindColumn {
				childW, childH = max(childW, innerW), childH+share(growingIndex)
			} else {
				childW, childH = childW+share(growingIndex), max(childH, innerH)
			}
			growingIndex++
		}
		if target.kind == kindColumn {
			m.place(child, cursorX, cursorY, childW, childH)
			cursorY += childH + target.spacing
		} else {
			m.place(child, cursorX, cursorY+(innerH-childH)/2, childW, childH)
			cursorX += childW + target.spacing
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

// press starts a click at a logical point and moves the focus there. A
// press on a ListBox or TableView row selects it; a press on a scroll bar
// starts dragging it; a press on a TextArea moves its caret; a press outside
// an open Select list closes it.
func (m *model) press(x, y int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.layout()
	m.dirty = true
	m.pressed = 0
	if m.popup != 0 {
		if m.inPopup(x, y) {
			m.pressed = m.popup
			return
		}
		m.closePopup()
	}
	id := m.hit(x, y)
	m.focus = 0
	if id == 0 || !m.widgets[id].focusable() {
		return
	}
	target := m.widgets[id]
	m.pressed, m.focus = id, id
	if m.pressScrollbar(target, x, y) {
		return
	}
	switch target.kind {
	case kindListBox, kindTable:
		if row := m.rowAt(target, y); row >= 0 {
			m.userSelect(target, row)
		}
	case kindTextArea:
		m.placeAreaCaret(target, x, y)
	}
}

// drag moves a scroll bar being dragged.
func (m *model) drag(x, y int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dragging == 0 {
		return
	}
	target := m.widgets[m.dragging]
	if target == nil {
		m.dragging = 0
		return
	}
	m.dragScrollbar(target, y)
}

// release ends a click. A click is a press and a release on the same Button,
// Checkbox, or Select; it returns the Button clicked, toggles a Checkbox,
// and opens or closes a Select's list or picks a choice in it.
func (m *model) release(x, y int) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.layout()
	m.dirty = true
	pressed := m.pressed
	m.pressed = 0
	if m.dragging != 0 {
		m.dragging = 0
		return 0
	}
	if pressed != 0 && pressed == m.popup {
		if m.inPopup(x, y) {
			if choice := m.popupChoiceAt(y); choice >= 0 {
				m.userSelect(m.widgets[m.popup], choice)
				m.closePopup()
			}
		}
		return 0
	}
	id := m.hit(x, y)
	if id == 0 || id != pressed {
		return 0
	}
	return m.activate(id)
}

// activate clicks a Button (returned), toggles a Checkbox, or opens and
// closes a Select's list. The caller holds m.mu.
func (m *model) activate(id int64) int64 {
	target := m.widgets[id]
	switch target.kind {
	case kindButton:
		return id
	case kindCheckbox:
		target.checked = !target.checked
		m.noteChange(id)
		m.dirty = true
	case kindSelect:
		if m.popup == id {
			m.closePopup()
		} else {
			m.openPopup(target)
		}
	}
	return 0
}

// focusNext moves the focus to the next enabled interactive widget in
// creation order, wrapping around; disabled widgets are skipped.
func (m *model) focusNext() { m.focusStep(1) }

// focusPrevious moves the focus backwards, for Shift+Tab.
func (m *model) focusPrevious() { m.focusStep(-1) }

func (m *model) focusStep(direction int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closePopup()
	start := -1
	if direction < 0 {
		start = len(m.order)
	}
	for index, id := range m.order {
		if id == m.focus {
			start = index
		}
	}
	for step := 1; step <= len(m.order); step++ {
		id := m.order[((start+direction*step)%len(m.order)+len(m.order))%len(m.order)]
		if m.widgets[id].focusable() {
			m.focus = id
			m.dirty = true
			return
		}
	}
}

// activateFocused handles Enter or Space on the focused widget: a Button is
// clicked (returned), a Checkbox toggled by Space, and a Select's list
// opened, or its highlighted choice picked. A text field keeps the key for
// editing.
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
	case target.kind == kindSelect && m.popup == target.id:
		if target.highlight >= 0 {
			m.userSelect(target, target.highlight)
		}
		m.closePopup()
	case target.kind == kindSelect:
		m.openPopup(target)
	}
	return 0
}

// focusedKind is the kind of the focused, enabled widget, or "".
func (m *model) focusedKind() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[m.focus]
	if target == nil || target.disabled {
		return ""
	}
	return target.kind
}

// popupOpen reports whether a Select's list is open.
func (m *model) popupOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.popup != 0
}

// escape closes an open Select list and reports whether one was open.
func (m *model) escape() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.popup == 0 {
		return false
	}
	m.closePopup()
	return true
}

// focusedInput is the focused one-line text field, or nil. The caller holds
// m.mu.
func (m *model) focusedInput() *widget {
	target := m.widgets[m.focus]
	if target == nil || !(target.textField() || target.kind == kindTextArea) || target.disabled {
		return nil
	}
	return target
}

// Editing operations on the focused text field. Each reports whether the
// focus is on a text field.

func (m *model) insert(runes []rune) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.focusedInput()
	if target == nil {
		return false
	}
	limit := maxTextRunes
	if target.kind == kindTextArea {
		limit = maxAreaRunes
	}
	changed := false
	for _, character := range runes {
		newline := character == '\n' && target.kind == kindTextArea
		if (!newline && (character < 0x20 || character == 0x7f)) || !utf8.ValidRune(character) || len(target.text) >= limit {
			continue
		}
		target.text = append(target.text[:target.caret], append([]rune{character}, target.text[target.caret:]...)...)
		target.caret++
		changed = true
	}
	if changed {
		m.noteChange(target.id)
		m.revealCaret(target)
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
	if target.kind == kindTextArea {
		m.editArea(target, key)
		return true
	}
	switch key {
	case "Backspace":
		if target.caret > 0 {
			target.text = append(target.text[:target.caret-1], target.text[target.caret:]...)
			target.caret--
			m.noteChange(target.id)
		}
	case "Delete":
		if target.caret < len(target.text) {
			target.text = append(target.text[:target.caret], target.text[target.caret+1:]...)
			m.noteChange(target.id)
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

// navigate moves the selection of the focused ListBox, TableView, or
// closed Select, or the highlight of an open Select list, with the arrow,
// Home, End, PageUp, and PageDown keys. It reports whether it used the key.
func (m *model) navigate(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[m.focus]
	if target == nil || target.disabled {
		return false
	}
	switch target.kind {
	case kindListBox, kindTable, kindSelect:
	default:
		return false
	}
	count := target.count()
	current := target.selected
	if m.popup == target.id {
		current = target.highlight
	}
	next, ok := stepIndex(current, count, key)
	if !ok {
		return false
	}
	if m.popup == target.id {
		target.highlight = next
		m.revealPopup(target)
		m.dirty = true
		return true
	}
	if next >= 0 {
		m.userSelect(target, next)
	}
	return true
}

// stepIndex moves an index for a navigation key among count entries.
func stepIndex(current, count int, key string) (int, bool) {
	if count == 0 {
		switch key {
		case "ArrowUp", "ArrowDown", "Home", "End", "PageUp", "PageDown":
			return -1, true
		}
		return -1, false
	}
	switch key {
	case "ArrowUp":
		if current < 0 {
			return 0, true
		}
		return max(current-1, 0), true
	case "ArrowDown":
		return min(current+1, count-1), true
	case "Home":
		return 0, true
	case "End":
		return count - 1, true
	case "PageUp":
		return max(current-pageItems, 0), true
	case "PageDown":
		return min(max(current, 0)+pageItems, count-1), true
	}
	return current, false
}

// userSelect selects an item or row as the user and records the change.
// The caller holds m.mu.
func (m *model) userSelect(target *widget, index int) {
	if target.disabled || index < -1 || index >= target.count() || target.selected == index {
		return
	}
	target.selected = index
	m.reveal(target)
	m.noteChange(target.id)
	m.dirty = true
}

// typeInto appends text to a text field as if typed, for headless scripts.
func (m *model) typeInto(id int64, text string) error {
	m.mu.Lock()
	target := m.widgets[id]
	if target == nil || !(target.textField() || target.kind == kindTextArea) {
		m.mu.Unlock()
		return errWrongKind
	}
	if target.disabled {
		// A disabled text field ignores the user, scripted or not.
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

// choose selects an item or row as the user would, for headless scripts.
func (m *model) choose(id int64, index int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil || (target.kind != kindListBox && target.kind != kindSelect && target.kind != kindTable) {
		return errWrongKind
	}
	if index < -1 || index >= target.count() {
		return errIndex
	}
	m.userSelect(target, index)
	return nil
}
