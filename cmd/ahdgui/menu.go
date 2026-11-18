package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
)

// Menus are drawn inside the window on every platform. The helper does not
// install an NSMenu, HMENU, or GTK menu, so the same bar, items, and
// shortcuts exist on macOS, Windows, and Linux.

const (
	menuKindMenu      = "menu"
	menuKindItem      = "item"
	menuKindCheck     = "check"
	menuKindSeparator = "separator"

	menuBarHeight  = 28
	menuItemHeight = 26
	menuPad        = 12
	maxMenus       = 32
	maxMenuItems   = 64
	maxMenuRunes   = 128
	maxMenuDepth   = 2
)

var (
	colorMenuBar  = color.RGBA{226, 226, 226, 255}
	colorMenuOpen = color.RGBA{210, 224, 242, 255}
	colorMenuDrop = color.RGBA{255, 255, 255, 255}

	errBadShortcut = errors.New(`a menu shortcut is a key such as "S", "Shift+S", or "Alt+S"`)
	errCommandKey  = errors.New("a menu shortcut already includes Command on macOS and Ctrl on Windows and Linux; do not write it in the shortcut")
	errNotMenu     = errors.New("the menu does not exist")
	errNotItem     = errors.New("only a menu item supports that request")
	errNotCheck    = errors.New("only a checkable menu item has a check")
)

// menuNode is one menu title, item, check item, or separator.
type menuNode struct {
	id       int64
	parent   int64
	kind     string
	text     string
	shortcut string
	enabled  bool
	checked  bool
	children []int64
}

// menuBox is one drawn menu title or item, in logical pixels.
type menuBox struct {
	id   int64
	kind string
	x, y int
	w, h int
}

// canonicalMenuShortcut accepts a key, optionally with Alt (or Option) and
// Shift, in either order. The primary modifier is not part of the text: "S"
// means Command+S on macOS and Ctrl+S elsewhere. The result is Alt, then
// Shift, then the key, with a letter in upper case.
func canonicalMenuShortcut(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	if strings.Contains(text, " ") || !utf8.ValidString(text) {
		return "", errBadShortcut
	}
	alt, shift := false, false
	key := ""
	for _, part := range strings.Split(text, "+") {
		switch part {
		case "Alt", "Option":
			if alt {
				return "", errBadShortcut
			}
			alt = true
		case "Shift":
			if shift {
				return "", errBadShortcut
			}
			shift = true
		case "Ctrl", "Control", "Command", "Cmd", "Meta":
			return "", errCommandKey
		default:
			if key != "" {
				return "", errBadShortcut
			}
			key = menuKeyName(part)
			if key == "" {
				return "", errBadShortcut
			}
		}
	}
	if key == "" {
		return "", errBadShortcut
	}
	return liveShortcut(alt, shift, key), nil
}

func menuKeyName(part string) string {
	runes := []rune(part)
	if len(runes) == 1 && runes[0] >= 'a' && runes[0] <= 'z' {
		part = string(runes[0] - 32)
	}
	if keyNameKnown(part) {
		return part
	}
	return ""
}

// liveShortcut is the canonical form of the modifiers held with a key.
func liveShortcut(alt, shift bool, key string) string {
	var b strings.Builder
	if alt {
		b.WriteString("Alt+")
	}
	if shift {
		b.WriteString("Shift+")
	}
	b.WriteString(key)
	return b.String()
}

func splitShortcut(text string) (alt, shift bool, key string, err error) {
	canon, err := canonicalMenuShortcut(text)
	if err != nil || canon == "" {
		if err == nil {
			err = errBadShortcut
		}
		return false, false, "", err
	}
	rest := canon
	if strings.HasPrefix(rest, "Alt+") {
		alt = true
		rest = strings.TrimPrefix(rest, "Alt+")
	}
	if strings.HasPrefix(rest, "Shift+") {
		shift = true
		rest = strings.TrimPrefix(rest, "Shift+")
	}
	return alt, shift, rest, nil
}

func (m *model) menuCount() int { return len(m.menus) }

func (m *model) addMenu(parent int64, title string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !validText(title, maxMenuRunes) || title == "" {
		return 0, errors.New("a menu title must be text of at most 128 characters")
	}
	if parent == 0 {
		if len(m.bar) >= maxMenus {
			return 0, errors.New("the Window has too many menus")
		}
	} else {
		owner := m.menus[parent]
		if owner == nil || owner.kind != menuKindMenu {
			return 0, errNotMenu
		}
		if m.menuDepth(parent) >= maxMenuDepth {
			return 0, errors.New("a submenu cannot contain another menu")
		}
		if len(owner.children) >= maxMenuItems {
			return 0, errors.New("the menu has too many items")
		}
	}
	if len(m.widgets)+m.menuCount() >= maxWidgets {
		return 0, errTooMany
	}
	return m.adopt(parent, menuKindMenu, title, "", false), nil
}

func (m *model) addItem(parent int64, kind, label, shortcut string, checked bool) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := m.menus[parent]
	if owner == nil || owner.kind != menuKindMenu {
		return 0, errNotMenu
	}
	if kind != menuKindItem && kind != menuKindCheck {
		return 0, errors.New("unknown menu item")
	}
	if !validText(label, maxMenuRunes) || label == "" {
		return 0, errors.New("a menu item label must be text of at most 128 characters")
	}
	canon, err := canonicalMenuShortcut(shortcut)
	if err != nil {
		return 0, err
	}
	if canon != "" {
		if _, used := m.shortcuts[canon]; used {
			return 0, fmt.Errorf("shortcut %q is already used", canon)
		}
	}
	if len(owner.children) >= maxMenuItems {
		return 0, errors.New("the menu has too many items")
	}
	if len(m.widgets)+m.menuCount() >= maxWidgets {
		return 0, errTooMany
	}
	id := m.adopt(parent, kind, label, canon, checked)
	if canon != "" {
		m.shortcuts[canon] = id
	}
	return id, nil
}

func (m *model) addSeparator(parent int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := m.menus[parent]
	if owner == nil || owner.kind != menuKindMenu {
		return errNotMenu
	}
	if len(owner.children) >= maxMenuItems {
		return errors.New("the menu has too many items")
	}
	if len(m.widgets)+m.menuCount() >= maxWidgets {
		return errTooMany
	}
	m.adopt(parent, menuKindSeparator, "", "", false)
	return nil
}

func (m *model) adopt(parent int64, kind, text, shortcut string, checked bool) int64 {
	m.next++
	node := &menuNode{id: m.next, parent: parent, kind: kind, text: text, shortcut: shortcut, enabled: true, checked: checked}
	m.menus[node.id] = node
	if parent == 0 {
		m.bar = append(m.bar, node.id)
	} else {
		owner := m.menus[parent]
		owner.children = append(owner.children, node.id)
	}
	m.dirty = true
	return node.id
}

func (m *model) menuDepth(id int64) int {
	depth := 0
	for id != 0 {
		node := m.menus[id]
		if node == nil {
			break
		}
		depth++
		id = node.parent
	}
	return depth
}

func (m *model) setMenuEnabled(id int64, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	node := m.itemNode(id)
	if node == nil {
		return errNotItem
	}
	node.enabled = enabled
	m.dirty = true
	return nil
}

func (m *model) menuEnabled(id int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	node := m.itemNode(id)
	if node == nil {
		return false, errNotItem
	}
	return node.enabled, nil
}

func (m *model) setMenuChecked(id int64, checked bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	node := m.menus[id]
	if node == nil || node.kind != menuKindCheck {
		return errNotCheck
	}
	node.checked = checked
	m.dirty = true
	return nil
}

func (m *model) menuChecked(id int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	node := m.menus[id]
	if node == nil || node.kind != menuKindCheck {
		return false, errNotCheck
	}
	return node.checked, nil
}

func (m *model) setMenuText(id int64, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	node := m.menus[id]
	if node == nil || (node.kind != menuKindMenu && node.kind != menuKindItem && node.kind != menuKindCheck) {
		return errNotItem
	}
	if !validText(text, maxMenuRunes) || text == "" {
		return errors.New("a menu label must be text of at most 128 characters")
	}
	node.text = text
	m.dirty = true
	return nil
}

func (m *model) itemNode(id int64) *menuNode {
	node := m.menus[id]
	if node == nil || (node.kind != menuKindItem && node.kind != menuKindCheck) {
		return nil
	}
	return node
}

func (m *model) itemChecked(id int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	node := m.menus[id]
	return node != nil && node.checked
}

func (m *model) menuExists(id int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.itemNode(id) != nil
}

// activateMenu runs one item: a check item flips, a disabled item does not
// run. It reports whether a callback should run.
func (m *model) activateMenu(id int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fire(id)
}

func (m *model) fire(id int64) bool {
	node := m.itemNode(id)
	if node == nil || !node.enabled {
		return false
	}
	if node.kind == menuKindCheck {
		node.checked = !node.checked
	}
	m.openMenu, m.openSub = 0, 0
	m.dirty = true
	return true
}

// menuShortcut claims the item that owns a canonical shortcut. A disabled
// item keeps the shortcut and does not run. The bool is true when some item
// owns it.
func (m *model) menuShortcut(canonical string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, owned := m.shortcuts[canonical]
	if !owned {
		return 0, false
	}
	if !m.fire(id) {
		return 0, true
	}
	return id, true
}

// applyChord is the one shortcut decision. A focused text field, including a
// password field, takes an editing chord and the menu item does not run.
// Otherwise the menu item that owns the chord runs, once. A disabled item
// reserves the chord. The last bool is true when the chord was consumed.
func (m *model) applyChord(goos string, command, alt, shift bool, key string) (menuID int64, copied string, didCopy, consumed bool) {
	if !command {
		return 0, "", false, false
	}
	if !alt {
		if action := textShortcut(goos, true, shift, key); action != "" && m.textFocused() {
			copied, didCopy = m.commandFocused(action)
			return 0, copied, didCopy, true
		}
	}
	id, owned := m.menuShortcut(liveShortcut(alt, shift, key))
	return id, "", false, owned
}

func (m *model) textFocused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[m.focus]
	if target == nil || target.disabled || !m.shown(m.focus) {
		return false
	}
	switch target.kind {
	case kindTextInput, kindPassword, kindTextArea:
		return true
	default:
		return false
	}
}

func (m *model) focusWidget(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil || target.disabled || !target.focusable() || !m.shown(id) {
		return
	}
	if m.focus != id {
		m.closeGroup(m.focus)
	}
	m.focus = id
	m.revealWidget(id)
}

func (m *model) closeMenus() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.openMenu == 0 && m.openSub == 0 {
		return false
	}
	m.openMenu, m.openSub = 0, 0
	m.dirty = true
	return true
}

func (m *model) takeMenu() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.menuFired
	m.menuFired = 0
	return id
}

func (m *model) menuOp(r request) (int64, *bool, *bool, error) {
	switch r.Kind {
	case "menu":
		id, err := m.addMenu(r.Parent, r.Text)
		return id, nil, nil, err
	case menuKindItem, menuKindCheck:
		checked := false
		if r.Checked != nil {
			checked = *r.Checked
		}
		id, err := m.addItem(r.Parent, r.Kind, r.Text, r.Name, checked)
		return id, nil, nil, err
	case menuKindSeparator:
		return 0, nil, nil, m.addSeparator(r.Parent)
	case "enabled":
		if r.Enabled == nil {
			value, err := m.menuEnabled(r.Widget)
			return 0, nil, &value, err
		}
		return 0, nil, nil, m.setMenuEnabled(r.Widget, *r.Enabled)
	case "checked":
		if r.Checked == nil {
			value, err := m.menuChecked(r.Widget)
			return 0, &value, nil, err
		}
		return 0, nil, nil, m.setMenuChecked(r.Widget, *r.Checked)
	case "text":
		return 0, nil, nil, m.setMenuText(r.Widget, r.Text)
	default:
		return 0, nil, nil, errors.New("unknown menu request")
	}
}

// layoutMenus fills the bar and any open dropdown. The caller holds m.mu,
// or is the only user of the model.
func (m *model) layoutMenus() {
	m.menuBar = m.menuBar[:0]
	m.menuDrop = m.menuDrop[:0]
	m.menuSub = m.menuSub[:0]
	if len(m.bar) == 0 || m.measure == nil {
		return
	}
	x := 0
	for _, id := range m.bar {
		node := m.menus[id]
		width := m.measure(node.text) + 2*menuPad
		if width < 48 {
			width = 48
		}
		m.menuBar = append(m.menuBar, menuBox{id: id, kind: node.kind, x: x, y: 0, w: width, h: menuBarHeight})
		x += width
	}
	if m.openMenu != 0 {
		box := boxOf(m.menuBar, m.openMenu)
		m.menuDrop = m.dropdown(m.openMenu, box.x, menuBarHeight)
	}
	if m.openSub != 0 {
		box := boxOf(m.menuDrop, m.openSub)
		m.menuSub = m.dropdown(m.openSub, box.x+box.w, box.y)
	}
}

func (m *model) dropdown(menuID int64, x, y int) []menuBox {
	node := m.menus[menuID]
	if node == nil {
		return nil
	}
	width := 160
	for _, child := range node.children {
		item := m.menus[child]
		label := item.text
		if item.kind == menuKindCheck {
			label = "✓ " + label
		}
		extra := ""
		if item.shortcut != "" {
			extra = shortcutLabel(item.shortcut)
		}
		if item.kind == menuKindMenu {
			extra = "▸"
		}
		w := m.measure(label) + m.measure(extra) + 3*menuPad
		if w > width {
			width = w
		}
	}
	boxes := make([]menuBox, 0, len(node.children))
	for _, child := range node.children {
		item := m.menus[child]
		h := menuItemHeight
		if item.kind == menuKindSeparator {
			h = 9
		}
		boxes = append(boxes, menuBox{id: child, kind: item.kind, x: x, y: y, w: width, h: h})
		y += h
	}
	return boxes
}

func boxOf(boxes []menuBox, id int64) menuBox {
	for _, box := range boxes {
		if box.id == id {
			return box
		}
	}
	return menuBox{}
}

// menuPointer handles a click on the bar or an open menu. A click outside an
// open menu closes it and is not delivered to the widget under it. The id is
// the item to run, or 0.
func (m *model) menuPointer(x, y int) (int64, bool) {
	if len(m.bar) == 0 {
		return 0, false
	}
	if id, ok := hitMenu(m.menuSub, x, y); ok {
		return m.menuHit(id)
	}
	if id, ok := hitMenu(m.menuDrop, x, y); ok {
		return m.menuHit(id)
	}
	if id, ok := hitMenu(m.menuBar, x, y); ok {
		if m.openMenu == id && m.openSub == 0 {
			m.openMenu = 0
		} else {
			m.openMenu, m.openSub = id, 0
		}
		return 0, true
	}
	if m.openMenu != 0 {
		m.openMenu, m.openSub = 0, 0
		return 0, true
	}
	return 0, false
}

func (m *model) menuHit(id int64) (int64, bool) {
	node := m.menus[id]
	if node == nil {
		return 0, true
	}
	if node.kind == menuKindMenu {
		m.openSub = id
		return 0, true
	}
	if node.kind == menuKindSeparator || !node.enabled {
		return 0, true
	}
	if m.fire(id) {
		return id, true
	}
	return 0, true
}

func hitMenu(boxes []menuBox, x, y int) (int64, bool) {
	for _, box := range boxes {
		if x >= box.x && y >= box.y && x < box.x+box.w && y < box.y+box.h {
			return box.id, true
		}
	}
	return 0, false
}

// shortcutLabel is the indicator drawn beside an item. It includes the
// primary modifier, which the shortcut text itself omits.
func shortcutLabel(shortcut string) string {
	primary, alt, shift := "Ctrl+", "Alt+", "Shift+"
	if runtime.GOOS == "darwin" {
		primary, alt, shift = "⌘", "⌥", "⇧"
	}
	rest := shortcut
	var b strings.Builder
	b.WriteString(primary)
	if strings.HasPrefix(rest, "Alt+") {
		b.WriteString(alt)
		rest = strings.TrimPrefix(rest, "Alt+")
	}
	if strings.HasPrefix(rest, "Shift+") {
		b.WriteString(shift)
		rest = strings.TrimPrefix(rest, "Shift+")
	}
	b.WriteString(rest)
	return b.String()
}

// drawMenus paints the bar and any open dropdown over the widgets.
func drawMenus(img *image.RGBA, face font.Face, f frame) {
	m := f.model
	if len(m.menuBar) == 0 {
		return
	}
	s := f.scale
	fill(img, scaled(s, 0, 0, m.width, menuBarHeight), colorMenuBar)
	for _, box := range m.menuBar {
		node := m.menus[box.id]
		rect := scaled(s, box.x, box.y, box.w, box.h)
		if m.openMenu == box.id {
			fill(img, rect, colorMenuOpen)
		}
		text(img, face, img.Bounds(), rect.Min.X+int(math.Round(float64(menuPad)*s)), rect, node.text, colorText)
	}
	drawMenuList(img, face, f, m.menuDrop)
	drawMenuList(img, face, f, m.menuSub)
}

func drawMenuList(img *image.RGBA, face font.Face, f frame, boxes []menuBox) {
	if len(boxes) == 0 {
		return
	}
	m, s := f.model, f.scale
	union := boxes[0]
	for _, box := range boxes[1:] {
		union.w = box.w
		union.h = box.y + box.h - union.y
	}
	panel := scaled(s, union.x, union.y, union.w, union.h)
	fill(img, panel, colorMenuDrop)
	outline(img, panel, colorBorder, max(1, int(math.Round(s))))
	for _, box := range boxes {
		node := m.menus[box.id]
		rect := scaled(s, box.x, box.y, box.w, box.h)
		if node.kind == menuKindSeparator {
			mid := rect.Min.Y + rect.Dy()/2
			fill(img, image.Rect(rect.Min.X+4, mid, rect.Max.X-4, mid+max(1, int(math.Round(s)))), colorBorder)
			continue
		}
		ink := colorText
		if !node.enabled && node.kind != menuKindMenu {
			ink = colorMuted
		}
		label := node.text
		if node.kind == menuKindCheck && node.checked {
			label = "✓ " + label
		}
		text(img, face, panel, rect.Min.X+int(math.Round(float64(menuPad)*s)), rect, label, ink)
		extra := ""
		if node.shortcut != "" {
			extra = shortcutLabel(node.shortcut)
		} else if node.kind == menuKindMenu {
			extra = "▸"
		}
		if extra != "" {
			width := font.MeasureString(face, extra).Ceil()
			text(img, face, panel, rect.Max.X-width-int(math.Round(float64(menuPad)*s)), rect, extra, ink)
		}
	}
}
