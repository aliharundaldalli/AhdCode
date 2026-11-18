package main

import (
	"bufio"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// session serves one Window after its open request. In window mode the
// protocol runs on its own goroutine while the window owns the main thread;
// headless mode has no window and serves on the main goroutine.
type session struct {
	model    *model
	in       *bufio.Reader
	out      *bufio.Writer
	headless bool
	openID   int64

	// closed is true once the Window is gone: the user closed it, or the
	// program asked to close it. A closed Window takes no further changes.
	closed         atomic.Bool
	closeRequested atomic.Bool
	windowGone     chan struct{}

	// writeMu serializes the output stream shared by responses and events.
	writeMu    sync.Mutex
	closedSent atomic.Bool

	// The events the program listens for: keys, clicks on some Buttons, and
	// changes of some widgets.
	listenKey atomic.Bool
	clicksMu  sync.Mutex
	clicks    map[int64]bool
	moves     map[int64]bool
	drags     map[int64]bool
	changes   map[int64]bool
	// menuClicks are the menu items whose activation the program listens to.
	menuClicks map[int64]bool

	// sink, when set, receives the events instead of the output stream: a
	// fallback dialog window handles its own clicks.
	sink func(event)

	// script is a headless test Window's input, replayed one event at a
	// time: wait starts it and each next request continues it.
	script []event
	cursor int
}

const closedMessage = "the Window is closed"

func (s *session) send(value any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return writeLine(s.out, value)
}

func (s *session) emit(value event) {
	if s.sink != nil {
		s.sink(value)
		return
	}
	if s.send(value) != nil {
		s.closeRequested.Store(true)
	}
}

// emitClosed tells the program, once, that the Window is gone, with the
// final TextInput and Checkbox readings.
func (s *session) emitClosed() {
	s.closed.Store(true)
	if s.closedSent.CompareAndSwap(false, true) {
		s.emit(event{Event: "closed", Values: s.model.values()})
	}
}

func (s *session) listensToMenu(id int64) bool {
	s.clicksMu.Lock()
	defer s.clicksMu.Unlock()
	return s.menuClicks[id]
}

func (s *session) menuActivated(id int64) {
	if id != 0 && s.listensToMenu(id) {
		s.emit(event{Event: "menu", Widget: id, Checked: s.model.itemChecked(id)})
	}
}

func (s *session) listensToClick(id int64) bool {
	s.clicksMu.Lock()
	defer s.clicksMu.Unlock()
	return s.clicks[id]
}

func (s *session) listensPointer(id int64, kind string) bool {
	s.clicksMu.Lock()
	defer s.clicksMu.Unlock()
	switch kind {
	case "move":
		return s.moves[id]
	case "drag":
		return s.drags[id]
	default:
		return s.clicks[id]
	}
}

// clicked reports a Button click the program listens to.
func (s *session) clicked(id int64) {
	if id != 0 && s.listensToClick(id) {
		s.emit(event{Event: "click", Widget: id})
	}
}

func (s *session) emitPointer(kind string, id int64, x, y float64) {
	if id != 0 && s.listensPointer(id, kind) {
		s.emit(event{Event: kind, Widget: id, X: x, Y: y})
	}
}

func (s *session) listensToChange(id int64) bool {
	s.clicksMu.Lock()
	defer s.clicksMu.Unlock()
	return s.changes[id]
}

// reportSort sends the header sort, when there is one, and reports whether
// it sent it. A header sort is not a selection change.
func (s *session) reportSort() bool {
	id, column, descending, ok := s.model.takeSort()
	if !ok {
		return false
	}
	selected := column
	s.emit(event{Event: "sort", Widget: id, Selected: &selected, Checked: descending})
	return true
}

// changed sends one change event for every widget the user changed that the
// program listens to, with the widget's new reading, and reports whether it
// sent any.
func (s *session) changed() bool {
	sent := false
	for _, id := range s.model.takeChanges() {
		if !s.listensToChange(id) {
			continue
		}
		reading, ok := s.model.valueNow(id)
		if !ok {
			continue
		}
		s.emit(event{Event: "change", Widget: id, Text: reading.Text, Checked: reading.Checked, Selected: reading.Selected})
		sent = true
	}
	return sent
}

// pressedKey reports a key press when the program listens for keys.
func (s *session) pressedKey(name string) {
	if s.listenKey.Load() {
		s.emit(event{Event: "key", Key: name})
	}
}

// serve answers requests until the runtime closes the pipe or sends close.
func (s *session) serve() {
	for {
		parsed, err := readRequest(s.in)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			_ = s.send(response{Error: err.Error()})
			if errors.Is(err, errRequestTooLarge) {
				return
			}
			continue
		}
		reply, stop := s.handle(parsed)
		reply.ID = parsed.ID
		if s.send(reply) != nil || stop {
			return
		}
		if s.headless && (parsed.Op == "wait" || parsed.Op == "next") {
			s.advance()
		}
	}
}

// advance replays the headless script up to and including its next event.
// Typing, toggling, and selecting change the model and send a change event
// when the program listens for one; a click on a Button nobody listens to,
// or a key when nobody listens for keys, produces nothing. When the script
// is exhausted the Window reports itself closed.
func (s *session) advance() {
	if s.changed() {
		return
	}
	for s.cursor < len(s.script) {
		step := s.script[s.cursor]
		s.cursor++
		switch step.Event {
		case "type", "toggle", "select", "sort":
			switch step.Event {
			case "type":
				_ = s.model.typeInto(step.Widget, step.Text)
			case "toggle":
				_ = s.model.toggle(step.Widget)
			case "sort":
				if step.Selected != nil {
					s.model.sortHeader(step.Widget, *step.Selected)
				}
			default:
				_ = s.model.choose(step.Widget, *step.Selected)
			}
			if s.reportSort() {
				return
			}
			if s.changed() {
				return
			}
		case "resize":
			if s.model.isResizable() {
				s.model.resize(step.Width, step.Height)
			}
		case "tick":
			s.emit(event{Event: "tick", Timer: step.Timer})
			return
		case "move", "drag":
			outside := step.Event == "drag"
			gx, gy, ok := s.model.canvasCoords(step.Widget, int(step.X), int(step.Y), outside)
			if ok && s.listensPointer(step.Widget, step.Event) {
				s.emit(event{Event: step.Event, Widget: step.Widget, X: gx, Y: gy})
				return
			}
		case "click":
			if s.model.kindOf(step.Widget) == kindCanvas {
				if gx, gy, ok := s.model.canvasCoords(step.Widget, int(step.X), int(step.Y), false); ok && s.listensToClick(step.Widget) {
					s.emit(event{Event: "click", Widget: step.Widget, X: gx, Y: gy})
					return
				}
				break
			}
			// A disabled Button ignores the user, scripted or not.
			if s.listensToClick(step.Widget) && s.model.enabled(step.Widget) {
				s.clicked(step.Widget)
				return
			}
		case "key":
			if s.listenKey.Load() {
				s.pressedKey(step.Key)
				return
			}
		case "edit":
			text, copied := s.model.command(step.Widget, step.Key)
			if copied {
				s.emit(event{Event: "clipboard", Text: text})
				return
			}
			if s.changed() {
				return
			}
		case "focus":
			s.model.focusWidget(step.Widget)
		case "menu":
			if s.model.activateMenu(step.Widget) && s.listensToMenu(step.Widget) {
				s.menuActivated(step.Widget)
				return
			}
		case "shortcut":
			alt, shift, key, err := splitShortcut(step.Key)
			if err != nil {
				continue
			}
			id, text, copied, _ := s.model.applyChord(runtime.GOOS, true, alt, shift, key)
			if copied {
				s.emit(event{Event: "clipboard", Text: text})
				return
			}
			if id != 0 && s.listensToMenu(id) {
				s.menuActivated(id)
				return
			}
		}
	}
	s.emitClosed()
}

func (s *session) handle(r request) (response, bool) {
	switch r.Op {
	case "status":
		open := !s.closed.Load()
		return response{OK: true, Open: &open}, false
	case "close":
		s.requestClose()
		return response{OK: true, Values: s.model.values()}, true
	case "wait", "next":
		// wait never blocks the protocol: the program keeps sending requests
		// from its callbacks and learns that the Window is gone from the
		// closed event.
		return response{OK: true}, false
	case "get":
		if r.Kind == "rows" {
			rows, selected, err := s.model.rowsNow(r.Widget)
			if err != nil {
				return response{Error: err.Error()}, false
			}
			return response{OK: true, Rows: rows, Selected: &selected}, false
		}
		if r.Kind == "sort" {
			column, descending, err := s.model.sortNow(r.Widget)
			if err != nil {
				return response{Error: err.Error()}, false
			}
			return response{OK: true, Selected: &column, Checked: &descending}, false
		}
		text, checked, selected, err := s.model.readAll(r.Widget)
		if err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true, Text: &text, Checked: &checked, Selected: &selected}, false
	}
	if s.closed.Load() {
		return response{Error: closedMessage, Closed: true}, false
	}
	switch r.Op {
	case "draw":
		if err := s.model.drawCanvas(r); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "add":
		selected := -1
		if r.Selected != nil {
			selected = *r.Selected
		}
		day := 0
		if r.Kind == kindDate && r.Selected != nil {
			day = *r.Selected
		}
		id, err := s.model.addSpec(r.Parent, spec{kind: r.Kind, text: r.Text, placeholder: r.Placeholder,
			checked: r.Checked != nil && *r.Checked, spacing: r.Spacing, padding: r.Padding,
			items: r.Items, columns: r.Columns, rows: r.Rows, selected: selected,
			gridColumns: r.Width, rowGap: r.Height, fitW: r.Width, fitH: r.Height,
			maximum: r.Width, year: r.Width, month: r.Height, day: day})
		if err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true, Widget: id}, false
	case "set":
		var err error
		switch {
		case r.Kind == "text":
			err = s.model.setText(r.Widget, r.Text)
		case r.Kind == "checked" && r.Checked != nil:
			err = s.model.setChecked(r.Widget, *r.Checked)
		case r.Kind == "items":
			err = s.model.setItems(r.Widget, r.Items)
		case r.Kind == "rows":
			err = s.model.setRows(r.Widget, r.Rows)
		case r.Kind == "sort" && r.Selected != nil && r.Checked != nil:
			err = s.model.sortBy(r.Widget, *r.Selected, *r.Checked)
		case r.Kind == "selected" && r.Selected != nil:
			err = s.model.selectIndex(r.Widget, *r.Selected)
		case r.Kind == "amount" && r.Selected != nil:
			err = s.model.setAmount(r.Widget, *r.Selected)
		case r.Kind == "indeterminate" && r.Checked != nil:
			err = s.model.setSpin(r.Widget, *r.Checked)
		case r.Kind == "date" && r.Selected != nil:
			err = s.model.setDate(r.Widget, r.Width, r.Height, *r.Selected)
		default:
			err = errors.New("set needs text, checked, items, rows, selected, sort, amount, indeterminate, or date")
		}
		if err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "color":
		value, ok := parseColor(r.Color)
		if !ok {
			return response{Error: "invalid color " + r.Color}, false
		}
		if err := s.model.setColor(r.Widget, r.Part, value); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "enabled":
		if r.Enabled == nil {
			return response{Error: "enabled needs a value"}, false
		}
		if err := s.model.setEnabled(r.Widget, *r.Enabled); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "visible":
		if r.Visible == nil {
			return response{Error: "visible needs a value"}, false
		}
		if err := s.model.setVisible(r.Widget, *r.Visible); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "box":
		if err := s.model.setBox(r.Kind, r.Widget, r.Width, r.Height, r.Text, r.Size); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "label":
		if err := s.model.setLabelStyle(r.Widget, r.Size, r.Bold); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "clipboard":
		if r.Clip == nil || !validText(*r.Clip, maxClipboardRunes) {
			return response{Error: "clipboard text is too long"}, false
		}
		s.model.setProgramClipboard(*r.Clip)
		return response{OK: true}, false
	case "resizable":
		if r.Enabled == nil {
			return response{Error: "resizable needs a value"}, false
		}
		s.model.setResizable(*r.Enabled)
		return response{OK: true}, false
	case "title":
		if !validText(r.Text, maxTitleRunes) {
			return response{Error: "invalid window title"}, false
		}
		s.model.setTitle(r.Text)
		return response{OK: true}, false
	case "menu":
		id, checked, enabled, err := s.model.menuOp(r)
		if err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true, Widget: id, Checked: checked, Enabled: enabled}, false
	case "listen":
		switch r.Kind {
		case "key":
			s.listenKey.Store(true)
		case "click":
			if _, _, err := s.model.read(r.Widget); err != nil {
				return response{Error: err.Error()}, false
			}
			s.clicksMu.Lock()
			s.clicks[r.Widget] = true
			s.clicksMu.Unlock()
		case "move", "drag":
			if _, _, err := s.model.read(r.Widget); err != nil {
				return response{Error: err.Error()}, false
			}
			s.clicksMu.Lock()
			if r.Kind == "move" {
				s.moves[r.Widget] = true
			} else {
				s.drags[r.Widget] = true
			}
			s.clicksMu.Unlock()
		case "change":
			if _, _, err := s.model.read(r.Widget); err != nil {
				return response{Error: err.Error()}, false
			}
			s.clicksMu.Lock()
			s.changes[r.Widget] = true
			s.clicksMu.Unlock()
		case "menu":
			if !s.model.menuExists(r.Widget) {
				return response{Error: "the menu item does not exist"}, false
			}
			s.clicksMu.Lock()
			s.menuClicks[r.Widget] = true
			s.clicksMu.Unlock()
		default:
			return response{Error: "unknown event kind " + r.Kind}, false
		}
		return response{OK: true}, false
	}
	return response{Error: "unknown request " + r.Op}, false
}

// requestClose ends the window, if any, and waits briefly for it to go.
func (s *session) requestClose() {
	s.closed.Store(true)
	if s.headless {
		return
	}
	s.closeRequested.Store(true)
	select {
	case <-s.windowGone:
	case <-time.After(5 * time.Second):
	}
}
