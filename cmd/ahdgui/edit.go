package main

import (
	"unicode"
	"unicode/utf8"
)

// editSnap is one undo step: the text and both ends of the selection.
type editSnap struct {
	text          []rune
	caret, anchor int
}

// textShortcut is the editing command for a platform shortcut, or "".
// macOS uses Command; Windows and Linux use Control. Shift+Z redoes on
// every platform, and Y redoes on Windows and Linux.
func textShortcut(goos string, command, shift bool, key string) string {
	if !command {
		return ""
	}
	switch key {
	case "A":
		return "selectAll"
	case "C":
		return "copy"
	case "X":
		return "cut"
	case "V":
		return "paste"
	case "Z":
		if shift {
			return "redo"
		}
		return "undo"
	case "Y":
		if goos != "darwin" {
			return "redo"
		}
	}
	return ""
}

func editActionKnown(key string) bool {
	switch key {
	case "selectAll", "copy", "cut", "paste", "undo", "redo", "word",
		"left", "right", "up", "down", "home", "end", "backspace", "delete",
		"shiftLeft", "shiftRight", "shiftUp", "shiftDown", "shiftHome", "shiftEnd":
		return true
	}
	return false
}

func (w *widget) selection() (int, int) {
	start, end := w.anchor, w.caret
	if start > end {
		start, end = end, start
	}
	return start, end
}

func (w *widget) hasSelection() bool {
	start, end := w.selection()
	return start != end
}

func (m *model) editable(target *widget) bool {
	if target == nil || target.disabled || !m.shown(target.id) {
		return false
	}
	return target.textField() || target.kind == kindTextArea
}

func (m *model) closeGroup(id int64) {
	if target := m.widgets[id]; target != nil {
		target.group = ""
	}
}

func (m *model) begin(target *widget, group string) {
	if target.group == group {
		return
	}
	target.undo = append(target.undo, editSnap{
		text:   append([]rune(nil), target.text...),
		caret:  target.caret,
		anchor: target.anchor,
	})
	if len(target.undo) > maxUndo {
		target.undo = append([]editSnap(nil), target.undo[len(target.undo)-maxUndo:]...)
	}
	target.group = group
	target.redo = nil
}

func (m *model) restore(target *widget, snap editSnap) {
	target.text = append([]rune(nil), snap.text...)
	target.caret, target.anchor = snap.caret, snap.anchor
	target.group = ""
	m.noteChange(target.id)
	m.revealCaret(target)
	m.dirty = true
}

func (m *model) undoEdit(target *widget) {
	if len(target.undo) == 0 {
		return
	}
	target.redo = append(target.redo, editSnap{
		text:   append([]rune(nil), target.text...),
		caret:  target.caret,
		anchor: target.anchor,
	})
	snap := target.undo[len(target.undo)-1]
	target.undo = target.undo[:len(target.undo)-1]
	m.restore(target, snap)
}

func (m *model) redoEdit(target *widget) {
	if len(target.redo) == 0 {
		return
	}
	target.undo = append(target.undo, editSnap{
		text:   append([]rune(nil), target.text...),
		caret:  target.caret,
		anchor: target.anchor,
	})
	snap := target.redo[len(target.redo)-1]
	target.redo = target.redo[:len(target.redo)-1]
	m.restore(target, snap)
}

func (m *model) removeSelection(target *widget) {
	start, end := target.selection()
	if start == end {
		return
	}
	target.text = append(target.text[:start], target.text[end:]...)
	target.caret, target.anchor = start, start
	m.noteChange(target.id)
}

func fieldLimit(target *widget) int {
	if target.kind == kindTextArea {
		return maxAreaRunes
	}
	return maxTextRunes
}

// insertLocked types runes at the caret. A selection is replaced by the
// whole run. Typing fills up to the field limit; a paste that does not fit
// is refused whole.
func (m *model) insertLocked(target *widget, runes []rune, typing bool) {
	if target == nil {
		return
	}
	limit := fieldLimit(target)
	accepted := make([]rune, 0, len(runes))
	for _, character := range runes {
		newline := character == '\n' && target.kind == kindTextArea
		if target.kind != kindTextArea && (character == '\n' || character == '\r') {
			continue
		}
		if (!newline && (character < 0x20 || character == 0x7f)) || !utf8.ValidRune(character) {
			continue
		}
		accepted = append(accepted, character)
	}
	if len(accepted) == 0 {
		return
	}
	start, end := target.selection()
	room := limit - (len(target.text) - (end - start))
	if room < 0 {
		room = 0
	}
	if !typing && len(accepted) > room {
		return
	}
	if typing && room == 0 && start == end {
		return
	}
	if typing {
		m.begin(target, "type")
	} else {
		m.begin(target, "paste")
	}
	if start != end {
		m.removeSelection(target)
	}
	wrote := false
	for _, character := range accepted {
		if len(target.text) >= limit {
			break
		}
		target.text = append(target.text[:target.caret], append([]rune{character}, target.text[target.caret:]...)...)
		target.caret++
		wrote = true
	}
	target.anchor = target.caret
	if !typing {
		// Each paste is its own undo step.
		target.group = ""
	}
	if wrote || start != end {
		m.noteChange(target.id)
		m.revealCaret(target)
	}
	m.dirty = true
}

func (m *model) deleteBackward(target *widget) {
	if target.hasSelection() {
		m.begin(target, "delete")
		m.removeSelection(target)
		m.revealCaret(target)
		m.dirty = true
		return
	}
	if target.caret == 0 {
		return
	}
	m.begin(target, "delete")
	start := prevCluster(target.text, target.caret)
	target.text = append(target.text[:start], target.text[target.caret:]...)
	target.caret, target.anchor = start, start
	m.noteChange(target.id)
	m.revealCaret(target)
	m.dirty = true
}

func (m *model) deleteForward(target *widget) {
	if target.hasSelection() {
		m.begin(target, "delete")
		m.removeSelection(target)
		m.revealCaret(target)
		m.dirty = true
		return
	}
	if target.caret >= len(target.text) {
		return
	}
	m.begin(target, "delete")
	end := nextCluster(target.text, target.caret)
	target.text = append(target.text[:target.caret], target.text[end:]...)
	target.anchor = target.caret
	m.noteChange(target.id)
	m.revealCaret(target)
	m.dirty = true
}

func (m *model) moveCaret(target *widget, caret int, shift bool) {
	caret = min(max(caret, 0), len(target.text))
	target.caret = snapCluster(target.text, caret)
	if !shift {
		target.anchor = target.caret
	}
	target.group = ""
	m.revealCaret(target)
	m.dirty = true
}

// editLocked applies one key to a text widget. The caller holds m.mu.
func (m *model) editLocked(target *widget, key string, shift bool) bool {
	if !m.editable(target) {
		return false
	}
	line, column := areaPosition(target.text, target.caret)
	switch key {
	case "Enter":
		if target.kind == kindTextArea {
			m.insertLocked(target, []rune{'\n'}, true)
		}
	case "Backspace":
		m.deleteBackward(target)
	case "Delete":
		m.deleteForward(target)
	case "ArrowLeft":
		if !shift && target.hasSelection() {
			start, _ := target.selection()
			m.moveCaret(target, start, false)
			break
		}
		m.moveCaret(target, prevCluster(target.text, target.caret), shift)
	case "ArrowRight":
		if !shift && target.hasSelection() {
			_, end := target.selection()
			m.moveCaret(target, end, false)
			break
		}
		m.moveCaret(target, nextCluster(target.text, target.caret), shift)
	case "ArrowUp":
		if target.kind != kindTextArea {
			break
		}
		next := 0
		if line > 0 {
			next = areaOffset(target.text, line-1, column)
		}
		m.moveCaret(target, next, shift)
	case "ArrowDown":
		if target.kind != kindTextArea {
			break
		}
		lines := areaLines(target.text)
		next := len(target.text)
		if line < len(lines)-1 {
			next = areaOffset(target.text, line+1, column)
		}
		m.moveCaret(target, next, shift)
	case "Home":
		next := 0
		if target.kind == kindTextArea {
			next = areaOffset(target.text, line, 0)
		}
		m.moveCaret(target, next, shift)
	case "End":
		next := len(target.text)
		if target.kind == kindTextArea {
			next = areaOffset(target.text, line, 1<<30)
		}
		m.moveCaret(target, next, shift)
	case "PageUp":
		if target.kind == kindTextArea {
			m.moveCaret(target, areaOffset(target.text, line-pageItems, column), shift)
		}
	case "PageDown":
		if target.kind == kindTextArea {
			m.moveCaret(target, areaOffset(target.text, line+pageItems, column), shift)
		}
	}
	return true
}

func (m *model) edit(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.editLocked(m.focusedInput(), key, false)
}

func (m *model) editMove(key string, shift bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.editLocked(m.focusedInput(), key, shift)
}

func (m *model) insert(runes []rune) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if date := m.widgets[m.focus]; date != nil && date.kind == kindDate && date.focusable() && m.shown(date.id) && m.tabShown(date.id) {
		m.typeDate(date, runes)
		return true
	}
	target := m.focusedInput()
	if target == nil {
		return false
	}
	m.insertLocked(target, runes, true)
	return true
}

// setProgramClipboard replaces the window copy of the program clipboard.
func (m *model) setProgramClipboard(text string) {
	m.mu.Lock()
	m.clipboard = text
	m.mu.Unlock()
}

func (m *model) rememberClipboard(text string) bool {
	if !validText(text, maxClipboardRunes) {
		return false
	}
	m.clipboard = text
	return true
}

// command runs one editing action on a widget. When it replaces the
// clipboard it returns that text.
func (m *model) command(id int64, action string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target != nil && target.kind == kindDate {
		if target.disabled || !m.shown(id) || !m.tabShown(id) {
			return "", false
		}
		m.dateKey(target, action)
		return "", false
	}
	if !m.editable(target) {
		return "", false
	}
	if move, shift, ok := editMoveOf(action); ok {
		m.editLocked(target, move, shift)
		return "", false
	}
	switch action {
	case "selectAll":
		target.anchor, target.caret = 0, len(target.text)
		target.group = ""
		m.dirty = true
	case "word":
		start, end := wordSpan(target.text, target.caret)
		target.anchor, target.caret = start, end
		target.group = ""
		m.dirty = true
	case "copy":
		return m.copySelection(target)
	case "cut":
		if target.kind == kindPassword {
			return "", false
		}
		text, copied := m.copySelection(target)
		if !copied {
			return "", false
		}
		m.begin(target, "delete")
		m.removeSelection(target)
		m.revealCaret(target)
		m.dirty = true
		return text, true
	case "paste":
		m.insertLocked(target, []rune(m.clipboard), false)
	case "undo":
		m.undoEdit(target)
	case "redo":
		m.redoEdit(target)
	}
	return "", false
}

func (m *model) commandFocused(action string) (string, bool) {
	m.mu.Lock()
	id := m.focus
	m.mu.Unlock()
	return m.command(id, action)
}

func editMoveOf(action string) (string, bool, bool) {
	switch action {
	case "left":
		return "ArrowLeft", false, true
	case "right":
		return "ArrowRight", false, true
	case "up":
		return "ArrowUp", false, true
	case "down":
		return "ArrowDown", false, true
	case "home":
		return "Home", false, true
	case "end":
		return "End", false, true
	case "shiftLeft":
		return "ArrowLeft", true, true
	case "shiftRight":
		return "ArrowRight", true, true
	case "shiftUp":
		return "ArrowUp", true, true
	case "shiftDown":
		return "ArrowDown", true, true
	case "shiftHome":
		return "Home", true, true
	case "shiftEnd":
		return "End", true, true
	case "backspace":
		return "Backspace", false, true
	case "delete":
		return "Delete", false, true
	}
	return "", false, false
}

func (m *model) copySelection(target *widget) (string, bool) {
	if target.kind == kindPassword || !target.hasSelection() {
		return "", false
	}
	start, end := target.selection()
	text := string(target.text[start:end])
	if !m.rememberClipboard(text) {
		return "", false
	}
	return text, true
}

// placeTextCaret moves the caret to a click, or selects a word on a double
// click, and starts a drag. The caller holds m.mu.
func (m *model) placeTextCaret(target *widget, x, y int, double bool) {
	index := m.caretAtPoint(target, x, y)
	if double {
		start, end := wordSpan(target.text, index)
		target.anchor, target.caret = start, end
		m.wordDrag = true
		m.wordFrom, m.wordTo = start, end
	} else {
		target.anchor, target.caret = index, index
		m.wordDrag = false
	}
	target.group = ""
	m.selecting = target.id
	m.revealCaret(target)
	m.dirty = true
}

func (m *model) dragSelection(x, y int) {
	target := m.widgets[m.selecting]
	if !m.editable(target) {
		m.selecting = 0
		return
	}
	index := m.caretAtPoint(target, x, y)
	if m.wordDrag {
		start, end := wordSpan(target.text, index)
		target.anchor = min(m.wordFrom, start)
		target.caret = max(m.wordTo, end)
	} else {
		target.caret = index
	}
	m.revealCaret(target)
	m.dirty = true
}

func (m *model) caretAtPoint(target *widget, x, y int) int {
	if target.kind == kindTextArea {
		return m.areaCaretAt(target, x, y)
	}
	shown := target.text
	if target.kind == kindPassword {
		shown = []rune(repeat(maskRune, len(target.text)))
	}
	want := x - (target.x + textInputInset) + int(float64(target.scroll)/max(m.scale, 1))
	if want < 0 {
		return 0
	}
	caret := 0
	for caret < len(shown) {
		next := nextCluster(shown, caret)
		width := m.measure(string(shown[:next]))
		if width > want {
			previous := m.measure(string(shown[:caret]))
			if want-previous > width-want {
				return next
			}
			return caret
		}
		caret = next
	}
	return len(target.text)
}

func repeat(text string, count int) string {
	out := make([]rune, count)
	one := []rune(text)
	for index := range out {
		out[index] = one[0]
	}
	return string(out)
}

func runeExtends(r rune) bool {
	if r == '\n' || r == '\r' {
		return false
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return true
	}
	switch r {
	case 0x200C, 0x200D, 0xFE0E, 0xFE0F, 0x20E3:
		return true
	}
	return r >= 0x1F3FB && r <= 0x1F3FF
}

func regionalIndicator(r rune) bool { return r >= 0x1F1E6 && r <= 0x1F1FF }

func clusterEnd(text []rune, start int) int {
	if start >= len(text) {
		return len(text)
	}
	index := start + 1
	for index < len(text) {
		character := text[index]
		if character == 0x200D && index+1 < len(text) {
			index += 2
			continue
		}
		if runeExtends(character) {
			index++
			continue
		}
		if index == start+1 && regionalIndicator(text[start]) && regionalIndicator(character) {
			index++
			continue
		}
		break
	}
	return index
}

// clusterStart is the first rune of the cluster that contains the rune just
// before index.
func clusterStart(text []rune, index int) int {
	if index <= 0 {
		return 0
	}
	if index > len(text) {
		index = len(text)
	}
	index--
	for index > 0 && (runeExtends(text[index]) || text[index] == 0x200D) {
		index--
	}
	if index > 0 && regionalIndicator(text[index]) && regionalIndicator(text[index-1]) {
		index--
	}
	return index
}

func nextCluster(text []rune, caret int) int {
	if caret >= len(text) {
		return len(text)
	}
	start := caret
	if caret > 0 {
		previous := clusterStart(text, caret)
		if previous < caret && caret < clusterEnd(text, previous) {
			start = previous
		}
	}
	return clusterEnd(text, start)
}

func prevCluster(text []rune, caret int) int {
	if caret <= 0 {
		return 0
	}
	start := clusterStart(text, caret)
	if start < caret {
		return start
	}
	return clusterStart(text, caret)
}

func snapCluster(text []rune, index int) int {
	if index <= 0 || index >= len(text) {
		return min(max(index, 0), len(text))
	}
	// index is inside a cluster when the rune before it and the rune at it
	// belong to the same cluster.
	start := clusterStart(text, index+1)
	if start == index {
		return index
	}
	end := clusterEnd(text, start)
	if index-start <= end-index {
		return start
	}
	return end
}

func baseRune(text []rune, index int) rune {
	if len(text) == 0 {
		return 0
	}
	index = min(max(index, 0), len(text)-1)
	return text[clusterStart(text, index+1)]
}

func runeClass(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_':
		return 1
	default:
		return 2
	}
}

func wordSpan(text []rune, index int) (int, int) {
	if len(text) == 0 {
		return 0, 0
	}
	if index >= len(text) {
		index = len(text) - 1
	}
	if index < 0 {
		index = 0
	}
	origin := clusterStart(text, index+1)
	if origin >= len(text) {
		origin = clusterStart(text, len(text))
	}
	class := runeClass(baseRune(text, origin))
	start := origin
	for start > 0 {
		previous := clusterStart(text, start)
		if previous == start || runeClass(baseRune(text, previous)) != class {
			break
		}
		start = previous
	}
	end := clusterEnd(text, origin)
	for end < len(text) && runeClass(baseRune(text, end)) == class {
		end = clusterEnd(text, end)
	}
	return start, end
}
