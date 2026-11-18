package main

import (
	"errors"
	"image"
	"image/color"
	"sync"
)

// Widget kinds. column and row are Containers; the others are leaves.
const (
	kindColumn     = "column"
	kindRow        = "row"
	kindGrid       = "grid"
	kindScroll     = "scroll"
	kindLabel      = "label"
	kindImage      = "image"
	kindButton     = "button"
	kindTextInput  = "textInput"
	kindCheckbox   = "checkbox"
	kindPassword   = "passwordInput"
	kindTextArea   = "textArea"
	kindListBox    = "listBox"
	kindSelect     = "select"
	kindTable      = "table"
	kindTabs       = "tabs"
	kindPage       = "page"
	kindRadioGroup = "radioGroup"
	kindRadio      = "radio"
	kindProgress   = "progress"
	kindDate       = "date"
	kindCanvas     = "canvas"
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

	// maxGridColumns bounds a grid and a widget's column or row span.
	// scrollViewport is the natural height of a scroll container when its
	// content is taller; a scroll root is then placed in the whole window.
	maxGridColumns = 32
	scrollViewport = 8 * rowHeight
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
	// gridColumns and rowGap belong to a grid. spacing is the column gap.
	// hAlign and vAlign are start, center, end, or stretch; empty keeps the
	// historical Column and Row placement. minW and prefW only raise the
	// natural width. spanC and spanR are the grid span, at least 1.
	gridColumns int
	rowGap      int
	hAlign      string
	vAlign      string
	minW        int
	prefW       int
	spanC       int
	spanR       int

	caret  int // text field caret position in runes
	anchor int // selection anchor; equal to caret when nothing is selected
	// undo and redo remember text and the caret. group names the open
	// editing gesture so a run of typing is one step.
	undo   []editSnap
	redo   []editSnap
	group  string
	scroll int // text field horizontal scroll in device pixels

	// items are a ListBox's or Select's texts; columns, rows, and widths a
	// TableView's header, cells, and column widths. selected is the chosen
	// item or row, -1 for none.
	items    []string
	columns  []string
	rows     [][]string
	widths   []int
	selected int
	// scrollX and scrollY scroll a ListBox, TableView, TextArea, or scroll
	// Container, in logical pixels. highlight is, while a Select's list is
	// open, the choice the keyboard points at, and listScroll the list's scroll.
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
	// hidden widgets take no space and ignore input, including through a
	// hidden Container. labelSize is small, normal, large, or title; empty
	// means normal. bold draws a Label twice, one pixel apart.
	hidden    bool
	labelSize string
	bold      bool
	// wrap breaks a Label on words. pixels is an Image's decoded picture,
	// read once; fitW and fitH are the requested box, 0 for the picture's
	// own size. scaled is that picture at the last destination size.
	wrap             bool
	pixels           image.Image
	pixelW, pixelH   int
	fitW, fitH       int
	scaled           *image.RGBA
	scaledW, scaledH int

	// maximum is a progress bar's upper bound. spin draws it without a
	// value. year, month, and day are a date field's calendar day; segment
	// is 0 for the year, 1 for the month, and 2 for the day, and digits is
	// the number being typed. remember is the last focused descendant of a
	// tab page.
	maximum  int
	spin     bool
	year     int
	month    int
	day      int
	segment  int
	digits   string
	remember int64
	// commands are an embedded Canvas's drawing list, in Canvas coordinates.
	commands []canvasCommand

	// Layout, in logical pixels, relative to the window's top-left. natW
	// and natH are the natural size; grows marks a scrolling widget, or a
	// Container holding one, which takes a resizable Window's extra space.
	x, y, w, h int
	natW, natH int
	grows      bool
	// sortColumn is the TableView column the rows are ordered by, or -1
	// when they are in the order setRows last gave them. sortDesc is the
	// direction of that order.
	sortColumn int
	sortDesc   bool
}

func (w *widget) container() bool {
	return w.kind == kindColumn || w.kind == kindRow || w.kind == kindGrid || w.kind == kindScroll ||
		w.kind == kindTabs || w.kind == kindPage || w.kind == kindRadioGroup
}

// stackedDown reports a container that places children from top to bottom.
func stackedDown(kind string) bool {
	return kind == kindColumn || kind == kindPage || kind == kindRadioGroup
}

// textField reports whether the widget edits text on one line.
func (w *widget) textField() bool { return w.kind == kindTextInput || w.kind == kindPassword }

// scrolling reports whether the widget scrolls its own content.
func (w *widget) scrolling() bool {
	return w.kind == kindListBox || w.kind == kindTable || w.kind == kindTextArea
}

// interactive reports whether the widget can take the focus.
func (w *widget) interactive() bool {
	switch w.kind {
	case kindButton, kindTextInput, kindCheckbox, kindPassword, kindTextArea, kindListBox, kindSelect, kindTable,
		kindTabs, kindRadio, kindDate:
		return true
	}
	return false
}

// canDisable reports whether setEnabled applies. An Image or a progress bar
// can be faded without becoming a Tab stop.
func (w *widget) canDisable() bool {
	return w.interactive() || w.kind == kindImage || w.kind == kindProgress || w.kind == kindCanvas
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
	mu     sync.Mutex
	title  string
	width  int
	height int
	// wrapOffer is the visible width a scroll container is offering its
	// children while they are placed. Zero means no scroll offer is active.
	wrapOffer int
	widgets   map[int64]*widget
	order     []int64 // creation order, which is also focus (Tab) order
	root      int64
	next      int64
	focus     int64
	pressed   int64
	dirty     bool
	// phase moves an indeterminate progress bar. It advances only while one
	// is visible, on the same frame tick that blinks the caret.
	phase   int
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
	grabX    int
	// dragHorizontal is set while a scroll container's horizontal bar is dragged.
	dragHorizontal bool
	// changes are the widgets the user changed since the last takeChanges,
	// in order, each once: the session sends their change events.
	// sortNotice is a TableView the user sorted from a header, or 0.
	changes    []int64
	sortNotice int64
	// selecting is the text widget being dragged, and wordDrag extends by
	// words after a double click. clipboard is this window's copy of the
	// program clipboard.
	selecting int64
	wordDrag  bool
	wordFrom  int
	wordTo    int
	clipboard string
	// menus are the menu bar and its items. shortcuts maps a canonical
	// shortcut to the one item that owns it. menuFired is the item a click
	// just activated. The open menu and submenu are drawn over the widgets.
	menus     map[int64]*menuNode
	shortcuts map[string]int64
	bar       []int64
	menuFired int64
	openMenu  int64
	openSub   int64
	menuBar   []menuBox
	menuDrop  []menuBox
	menuSub   []menuBox
	// canvasPress is the Canvas a drag began on. hoverX and hoverY are the
	// last pointer position, so a still pointer is not reported again.
	// pointer is one event waiting for the session to send.
	canvasPress        int64
	hoverX, hoverY     int
	hoverReady         bool
	pointerKind        string
	pointerID          int64
	pointerX, pointerY float64
	pointerReady       bool
}

func newModel(title string, width, height int, measure func(string) int) *model {
	return &model{title: title, width: width, height: height, widgets: map[int64]*widget{}, measure: measure, dirty: true, scale: 1,
		menus: map[int64]*menuNode{}, shortcuts: map[string]int64{}}
}

var (
	errSecondRoot   = errors.New("the Window already has its root Container")
	errNoParent     = errors.New("the parent Container does not exist")
	errNotContainer = errors.New("the parent is not a Container")
	errTooMany      = errors.New("the Window has too many widgets")
	errNoWidget     = errors.New("the widget does not exist")
	errWrongKind    = errors.New("the widget does not support that request")
	errIndex        = errors.New("the index is outside the items")
	errLabelSize    = errors.New(`label size must be "small", "normal", "large", or "title"`)
)

// spec is everything an add request can carry.
type spec struct {
	kind, text, placeholder string
	checked                 bool
	spacing, padding        int
	items, columns          []string
	rows                    [][]string
	selected                int
	gridColumns, rowGap     int
	// fitW and fitH are an Image's requested size. 0 means not given.
	fitW, fitH int
	// maximum, year, month, and day belong to a progress bar or a date field.
	maximum, year, month, day int
}

// add creates a widget without items; parent 0 creates the root Container.
func (m *model) add(parent int64, kind, text, placeholder string, checked bool, spacing, padding int) (int64, error) {
	return m.addSpec(parent, spec{kind: kind, text: text, placeholder: placeholder, checked: checked,
		spacing: spacing, padding: padding, selected: -1})
}

// addSpec creates a widget; parent 0 creates the root Container.
func (m *model) addSpec(parent int64, s spec) (int64, error) {
	var pixels image.Image
	var pixelW, pixelH int
	if s.kind == kindImage {
		// Decode before taking the lock. The pixels stay on the widget, so
		// drawing does not read the file again.
		if !validText(s.text, maxTextRunes) {
			return 0, errors.New("invalid widget text")
		}
		if s.fitW < 0 || s.fitH < 0 || s.fitW > maxDimension || s.fitH > maxDimension {
			return 0, badImageSize()
		}
		var err error
		pixels, pixelW, pixelH, err = loadGUIImage(s.text)
		if err != nil {
			return 0, err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.widgets) >= maxWidgets {
		return 0, errTooMany
	}
	switch s.kind {
	case kindColumn, kindRow, kindGrid, kindScroll, kindLabel, kindImage, kindButton, kindTextInput, kindCheckbox,
		kindPassword, kindTextArea, kindListBox, kindSelect, kindTable,
		kindTabs, kindPage, kindRadioGroup, kindRadio, kindProgress, kindDate, kindCanvas:
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
	if err := validateGrid(s); err != nil {
		return 0, err
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
		checked: s.checked, spacing: s.spacing, padding: s.padding, selected: -1, highlight: -1,
		gridColumns: s.gridColumns, rowGap: s.rowGap, spanC: 1, spanR: 1, sortColumn: -1}
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
	case kindImage:
		created.pixels = pixels
		created.pixelW, created.pixelH = pixelW, pixelH
		created.fitW, created.fitH = s.fitW, s.fitH
	case kindProgress:
		created.maximum = s.maximum
		created.selected = 0
	case kindDate:
		created.year, created.month, created.day = s.year, s.month, s.day
	case kindCanvas:
		created.fitW, created.fitH = s.fitW, s.fitH
	}
	if err := validateNewControl(m, parent, s); err != nil {
		return 0, err
	}
	if parent == 0 {
		if m.root != 0 {
			return 0, errSecondRoot
		}
		if !created.container() || s.kind == kindTabs || s.kind == kindPage || s.kind == kindRadioGroup {
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
	created.anchor = created.caret
	m.widgets[created.id] = created
	m.order = append(m.order, created.id)
	if parent == 0 {
		m.root = created.id
	} else {
		m.widgets[parent].children = append(m.widgets[parent].children, created.id)
		if s.kind == kindPage && len(m.widgets[parent].children) == 1 {
			m.widgets[parent].selected = 0
		}
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
	case kindLabel, kindButton, kindCheckbox, kindTextInput, kindPassword, kindRadio:
		if target.kind == kindRadio && (len([]rune(text)) == 0 || len([]rune(text)) > 256) {
			return errors.New("a radio option must be between 1 and 256 characters")
		}
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
	target.caret, target.anchor = len(target.text), len(target.text)
	target.scroll, target.scrollY = 0, 0
	target.undo, target.redo, target.group = nil, nil, ""
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
	if target.sortColumn >= 0 {
		m.orderRows(target)
	}
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
	switch target.kind {
	case kindTabs:
		return m.selectTab(target, index)
	case kindRadioGroup:
		return m.selectRadio(target, index)
	case kindListBox, kindSelect, kindTable:
		if index < -1 || index >= target.count() {
			return errIndex
		}
		target.selected = index
		m.reveal(target)
		m.dirty = true
		return nil
	default:
		return errWrongKind
	}
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
	switch target.kind {
	case kindDate:
		return formatDate(target.year, target.month, target.day), false, -1, nil
	case kindProgress:
		return "", target.spin, target.selected, nil
	case kindTabs, kindRadioGroup:
		return "", false, target.selected, nil
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
	case kindListBox, kindSelect, kindTable, kindTabs, kindRadioGroup:
		selected := target.selected
		return value{Widget: target.id, Selected: &selected}, true
	case kindProgress:
		selected := target.selected
		return value{Widget: target.id, Checked: target.spin, Selected: &selected}, true
	case kindDate:
		return value{Widget: target.id, Text: formatDate(target.year, target.month, target.day)}, true
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
	if !target.canDisable() {
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

// setVisible hides or shows a widget or Container. A hidden widget takes no
// space, and hiding the focused widget, or a Container that holds it, drops
// the focus.
func (m *model) setVisible(id int64, visible bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	target.hidden = !visible
	if target.hidden {
		m.dropFocusUnder(id)
	}
	m.dirty = true
	return nil
}

// dropFocusUnder clears the focus, press, popup, and drag when they sit on
// id or on one of its descendants. The caller holds m.mu.
func (m *model) dropFocusUnder(id int64) {
	if m.inside(m.focus, id) {
		m.focus = 0
	}
	if m.inside(m.pressed, id) {
		m.pressed = 0
	}
	if m.inside(m.popup, id) {
		m.closePopup()
	}
	if m.inside(m.dragging, id) {
		m.dragging = 0
	}
}

// inside reports whether id is ancestor or id itself. The caller holds m.mu.
func (m *model) inside(id, ancestor int64) bool {
	for id != 0 {
		if id == ancestor {
			return true
		}
		target := m.widgets[id]
		if target == nil {
			return false
		}
		id = target.parent
	}
	return false
}

// shown reports whether a widget and every Container above it are visible.
// The caller holds m.mu.
func (m *model) shown(id int64) bool {
	for id != 0 {
		target := m.widgets[id]
		if target == nil || target.hidden {
			return false
		}
		id = target.parent
	}
	return true
}

// setLabelStyle changes a Label's size, its weight, or both.
func (m *model) setLabelStyle(id int64, size string, bold *bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindLabel {
		return errWrongKind
	}
	if size == "" && bold == nil {
		return errors.New("label style needs a size or bold")
	}
	if size != "" {
		switch size {
		case "small", "normal", "large", "title":
			target.labelSize = size
		default:
			return errLabelSize
		}
	}
	if bold != nil {
		target.bold = *bold
	}
	m.dirty = true
	return nil
}

func labelScale(size string) float64 {
	switch size {
	case "small":
		return 0.85
	case "large":
		return 1.25
	case "title":
		return 1.6
	default:
		return 1
	}
}

func labelHeight(size string) int {
	switch size {
	case "small":
		return rowHeight - 6
	case "large":
		return rowHeight + 10
	case "title":
		return rowHeight + 18
	default:
		return rowHeight
	}
}

func (m *model) kindOf(id int64) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.widgets[id] == nil {
		return ""
	}
	return m.widgets[id].kind
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
// spacing, inside its padding. With no explicit alignment, a Column's
// children stay left-aligned and a Row's children stay centered vertically.
// The root starts at the top-left, or just below the menu bar when the
// window has one. What does not fit is clipped, unless the root is a scroll
// Container: that root fills the window and scrolls.
//
// A ListBox, TableView, or TextArea, and every Container holding one, grows:
// when the window is larger than the content, the extra height of a Column
// (and the extra width of a Row) is shared equally by its growing children,
// the first ones taking any remainder, and a growing child also fills the
// Container's full width in a Column (height in a Row). An explicit
// alignment can center, end-align, or stretch a child that does not grow.
// Nothing shrinks below its natural size. The caller holds m.mu.
func (m *model) layout() {
	m.layoutMenus()
	if m.root == 0 {
		return
	}
	root := m.widgets[m.root]
	m.measureWidget(m.root)
	top := windowMargin
	if len(m.bar) > 0 {
		top = menuBarHeight
	}
	width, height := root.natW, root.natH
	if root.kind == kindScroll {
		// A scroll root fills the window, below the menu bar, so a small
		// window scrolls instead of clipping the scroll bar away.
		width = max(m.width-2*windowMargin, 0)
		height = max(m.height-top-windowMargin, 0)
	} else if root.grows {
		width, height = max(width, m.width-2*windowMargin), max(height, m.height-top-windowMargin)
		if m.width > 0 && m.width < root.natW {
			// The column stays wide enough for its fields. Wrapping labels
			// still reflow to the window, and the extra lines push the
			// widgets below them down.
			m.wrapOffer = max(m.width-2*windowMargin, 0)
			if _, flowed := m.flowBox(m.root, m.wrapOffer); flowed > height {
				height = flowed
			}
		}
	} else if m.width > 0 && m.width < width {
		// A narrower window reflows wrapping labels. The root keeps this
		// window width; children that cannot wrap stay at their natural
		// width and the window clips them.
		width = m.width
		if _, flowed := m.flowBox(m.root, width); flowed > height {
			height = flowed
		}
	}
	m.place(m.root, windowMargin, top, width, height)
	m.wrapOffer = 0
	if m.popup != 0 {
		m.placePopup(m.widgets[m.popup])
	}
}

func (m *model) measureWidget(id int64) (int, int) {
	target := m.widgets[id]
	if target.hidden {
		target.grows = false
		target.natW, target.natH = 0, 0
		return 0, 0
	}
	target.grows = target.scrolling()
	switch target.kind {
	case kindLabel:
		m.measureLabel(target)
	case kindImage:
		m.measureImage(target)
	case kindButton:
		target.natW, target.natH = max(m.measure(string(target.text))+2*buttonPadding, minButtonWidth), rowHeight
	case kindTextInput, kindPassword:
		target.natW, target.natH = textInputWidth, rowHeight
	case kindCheckbox, kindRadio:
		target.natW, target.natH = checkboxBox+checkboxGap+m.measure(string(target.text)), rowHeight
	case kindProgress, kindDate:
		target.natW, target.natH = textInputWidth, rowHeight
	case kindCanvas:
		target.natW, target.natH = target.fitW, target.fitH
	case kindTabs:
		m.measureTabs(target)
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
	case kindColumn, kindRow, kindPage, kindRadioGroup:
		width, height := 0, 0
		seen := 0
		for _, child := range target.children {
			childWidth, childHeight := m.measureWidget(child)
			placed := m.widgets[child]
			if placed.hidden {
				continue
			}
			if placed.grows {
				target.grows = true
			}
			gap := 0
			if seen > 0 {
				gap = target.spacing
			}
			seen++
			if stackedDown(target.kind) {
				width, height = max(width, childWidth), height+gap+childHeight
			} else {
				width, height = width+gap+childWidth, max(height, childHeight)
			}
		}
		target.natW, target.natH = width+2*target.padding, height+2*target.padding
	case kindGrid:
		m.measureGrid(target)
	case kindScroll:
		m.measureScroll(target)
	}
	m.finishSize(target)
	return target.natW, target.natH
}

// place gives a widget its position and size, and lays out a Container's
// children inside it.
func (m *model) place(id int64, x, y, width, height int) {
	target := m.widgets[id]
	target.x, target.y, target.w, target.h = x, y, width, height
	if target.hidden {
		target.w, target.h = 0, 0
		return
	}
	switch target.kind {
	case kindGrid:
		m.placeGrid(target)
		return
	case kindScroll:
		m.placeScroll(target)
		return
	case kindTabs:
		m.placeTabs(target)
		return
	}
	if !target.container() {
		m.clampScroll(target)
		return
	}
	innerW, innerH := width-2*target.padding, height-2*target.padding
	growing, used := 0, 0
	seen := 0
	for _, child := range target.children {
		placed := m.widgets[child]
		if placed.hidden {
			continue
		}
		if placed.grows {
			growing++
		}
		if seen > 0 {
			used += target.spacing
		}
		seen++
		if stackedDown(target.kind) {
			used += placed.natH
		} else {
			used += placed.natW
		}
	}
	offer := innerW
	if stackedDown(target.kind) {
		offer = m.flowLimit(target.padding, innerW)
		used = 0
		seen = 0
		for _, child := range target.children {
			placed := m.widgets[child]
			if placed.hidden {
				continue
			}
			_, childH := m.childSize(placed, offer)
			if seen > 0 {
				used += target.spacing
			}
			seen++
			used += childH
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
		if placed.hidden {
			placed.w, placed.h = 0, 0
			continue
		}
		childW, childH := placed.natW, placed.natH
		if stackedDown(target.kind) {
			childW, childH = m.childSize(placed, offer)
		}
		if placed.grows {
			if stackedDown(target.kind) {
				childW, childH = max(childW, innerW), childH+share(growingIndex)
			} else {
				childW, childH = childW+share(growingIndex), max(childH, innerH)
			}
			growingIndex++
		}
		if stackedDown(target.kind) {
			childX := cursorX
			if placed.hAlign != "" && !placed.grows {
				childX, childW = alignAxis(placed.hAlign, cursorX, innerW, childW)
			}
			m.place(child, childX, cursorY, childW, childH)
			cursorY += childH + target.spacing
		} else {
			childY := cursorY + (innerH-childH)/2
			if placed.vAlign != "" {
				if placed.grows {
					switch placed.vAlign {
					case "start", "stretch":
						childY = cursorY
					case "end":
						childY = cursorY + innerH - childH
					default:
						childY = cursorY + (innerH-childH)/2
					}
				} else {
					childY, childH = alignAxis(placed.vAlign, cursorY, innerH, childH)
				}
			}
			m.place(child, cursorX, childY, childW, childH)
			cursorX += childW + target.spacing
		}
	}
}

// hit finds the leaf widget at a logical point, or 0.
func (m *model) hit(x, y int) int64 {
	for index := len(m.order) - 1; index >= 0; index-- {
		target := m.widgets[m.order[index]]
		if !target.container() && m.shown(target.id) && m.tabShown(target.id) && x >= target.x && y >= target.y && x < target.x+target.w && y < target.y+target.h && m.insideScrolls(target.id, x, y) {
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
func (m *model) press(x, y int) { m.pressAt(x, y, false) }

func (m *model) pressAt(x, y int, double bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.layout()
	m.dirty = true
	m.pressed = 0
	m.menuFired = 0
	if id, handled := m.menuPointer(x, y); handled {
		m.menuFired = id
		return
	}
	if m.popup != 0 {
		if m.inPopup(x, y) {
			m.pressed = m.popup
			return
		}
		m.closePopup()
	}
	if tabs, index, ok := m.tabAt(x, y); ok {
		m.userTab(m.widgets[tabs], index, true)
		m.pressed = tabs
		return
	}
	id := m.hit(x, y)
	if id != 0 && m.widgets[id].kind == kindCanvas {
		target := m.widgets[id]
		if !target.disabled {
			if gx, gy, ok := m.canvasPoint(target, x, y, false); ok {
				m.canvasPress = id
				m.queuePointer("click", id, gx, gy)
			}
		}
		return
	}
	if id == 0 && m.pressScrollBar(x, y) {
		return
	}
	if m.focus != id {
		m.closeGroup(m.focus)
	}
	m.focus = 0
	if id == 0 || !m.widgets[id].focusable() || !m.shown(id) {
		return
	}
	target := m.widgets[id]
	m.pressed, m.focus = id, id
	m.notePageFocus(id)
	if m.pressScrollbar(target, x, y) {
		return
	}
	switch target.kind {
	case kindListBox:
		if row := m.rowAt(target, y); row >= 0 {
			m.userSelect(target, row)
		}
	case kindTable:
		if column := m.headerColumn(target, x, y); column >= 0 {
			m.userSort(target, column)
			return
		}
		if row := m.rowAt(target, y); row >= 0 {
			m.userSelect(target, row)
		}
	case kindTextInput, kindPassword, kindTextArea:
		m.placeTextCaret(target, x, y, double)
	case kindDate:
		m.pickSegment(target, x)
	}
}

// drag moves a scroll bar being dragged.
func (m *model) drag(x, y int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.canvasPress != 0 {
		target := m.widgets[m.canvasPress]
		if target == nil || target.disabled || !m.shown(m.canvasPress) {
			m.canvasPress = 0
			return
		}
		gx, gy, _ := m.canvasPoint(target, x, y, true)
		m.queuePointer("drag", m.canvasPress, gx, gy)
		return
	}
	if m.selecting != 0 {
		m.dragSelection(x, y)
		return
	}
	if m.dragging == 0 {
		return
	}
	target := m.widgets[m.dragging]
	if target == nil {
		m.dragging = 0
		return
	}
	if target.kind == kindScroll {
		m.dragScrollContainer(target, x, y)
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
	if m.canvasPress != 0 {
		m.canvasPress = 0
		m.pressed = 0
		return 0
	}
	pressed := m.pressed
	m.pressed = 0
	m.selecting = 0
	m.wordDrag = false
	if m.dragging != 0 {
		m.dragging = 0
		m.dragHorizontal = false
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
	case kindRadio:
		m.chooseRadio(target)
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
		if m.widgets[id].focusable() && m.shown(id) && m.tabShown(id) {
			if m.focus != id {
				m.closeGroup(m.focus)
			}
			m.focus = id
			m.notePageFocus(id)
			m.revealWidget(id)
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
	case target.kind == kindRadio:
		m.chooseRadio(target)
	case target.kind == kindDate && !space:
		m.commitDigits(target)
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
	if !m.editable(target) {
		return nil
	}
	return target
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
	case kindTabs:
		return m.moveTabs(target, key)
	case kindRadio:
		return m.moveRadio(target, key)
	case kindDate:
		return m.dateKey(target, key)
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
	if target == nil || !(target.textField() || target.kind == kindTextArea || target.kind == kindDate) {
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
	if target == nil {
		return errNoWidget
	}
	switch target.kind {
	case kindTabs:
		pages := m.pages(target)
		if index < 0 || index >= len(pages) {
			return errTabIndex
		}
		m.userTab(target, index, false)
		return nil
	case kindRadioGroup:
		return m.userRadio(target, index)
	case kindListBox, kindSelect, kindTable:
	default:
		return errWrongKind
	}
	if index < -1 || index >= target.count() {
		return errIndex
	}
	m.userSelect(target, index)
	return nil
}
