package ahdruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// GUI standard module (v1.8.0)
// ---------------------------------------------------------------------------
//
// GUI is a deliberately small desktop toolkit: a Window with one root
// Container, Columns and Rows, and Label, Button, TextInput, and Checkbox
// widgets, plus two callbacks, Button.onClick and Window.onKey. This file is
// the whole AhdCode side of it and uses only the standard library. The
// window, layout, drawing, focus, and text editing live in the bundled ahdgui
// helper, one process per open Window, reached through helperlink.go's
// request/response and event protocol (see cmd/ahdgui/protocol.go; the
// request shape is duplicated here because this file is embedded into every
// compiled program and cannot import that module).
//
// Both compiled programs and the REPL evaluator call these same functions.
// Functions report a problem as a non-empty message; AhdGUICheck turns it
// into a GUIError in a compiled program, and the evaluator raises its own
// GUIError from the same message. A callback's own error is never turned into
// a GUIError: it propagates unchanged.

// AhdClassGUIError is the runtime descriptor of GUIError.
var AhdClassGUIError = &AhdClass{Name: "GUIError", Parent: AhdClassError}

// AhdGUIRuntimeHint is the directory of the bundled ahdgui helper, filled in
// by the compiler for programs that use GUI, exactly like
// AhdGraphicsRuntimeHint. AHDCODE_GUI_RUNTIME overrides it for packaging and
// tests.
var AhdGUIRuntimeHint string

const (
	ahdGUIProtocolVersion = 2
	// AhdGUIMaxDimension bounds a Window's width and height.
	AhdGUIMaxDimension  = 4096
	ahdGUIMaxTitleRunes = 256
	// AhdGUIMaxTextRunes bounds every widget text.
	AhdGUIMaxTextRunes = 4096
	// AhdGUIMaxSpacing bounds spacing and padding.
	AhdGUIMaxSpacing = 1000
	// ahdGUIMaxLineBytes bounds one helper line; the closed event carries
	// every TextInput value, each at most AhdGUIMaxTextRunes characters.
	ahdGUIMaxLineBytes    = 8 << 20
	ahdGUIOpenTimeout     = 30 * time.Second
	ahdGUIRequestTimeout  = 60 * time.Second
	ahdGUICloseTimeout    = 5 * time.Second
	ahdGUIClosedMessage   = "the Window is closed"
	ahdGUIStoppedMessage  = "the GUI window helper stopped unexpectedly"
	ahdGUIReleasedMessage = "the widget's Window was closed with close()"
)

type ahdGUIRequest struct {
	Op string `json:"op"`

	Version  int             `json:"version,omitempty"`
	Width    int             `json:"width,omitempty"`
	Height   int             `json:"height,omitempty"`
	Title    string          `json:"title,omitempty"`
	Headless bool            `json:"headless,omitempty"`
	Script   json.RawMessage `json:"script,omitempty"`

	Parent      int64  `json:"parent,omitempty"`
	Widget      int64  `json:"widget,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Text        string `json:"text,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Checked     *bool  `json:"checked,omitempty"`
	Spacing     int    `json:"spacing,omitempty"`
	Padding     int    `json:"padding,omitempty"`
	Part        string `json:"part,omitempty"`
	Color       string `json:"color,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

type ahdGUIValue struct {
	Widget  int64  `json:"widget"`
	Text    string `json:"text,omitempty"`
	Checked bool   `json:"checked,omitempty"`
}

type ahdGUIResponse struct {
	OK      bool          `json:"ok"`
	Error   string        `json:"error,omitempty"`
	Closed  bool          `json:"closed,omitempty"`
	Open    *bool         `json:"open,omitempty"`
	Widget  int64         `json:"widget,omitempty"`
	Text    *string       `json:"text,omitempty"`
	Checked *bool         `json:"checked,omitempty"`
	Values  []ahdGUIValue `json:"values,omitempty"`
}

type ahdGUIEvent struct {
	Event  string        `json:"event"`
	Widget int64         `json:"widget,omitempty"`
	Key    string        `json:"key,omitempty"`
	Values []ahdGUIValue `json:"values,omitempty"`
}

// ahdGUIWindow is one open Window. mu serializes requests; it is never held
// while a callback runs.
type ahdGUIWindow struct {
	mu       sync.Mutex
	link     *ahdHelperLink
	usable   bool // the Window can still be changed
	alive    bool // the helper process is still running
	headless bool
	root     bool
	onKey    func(key string)
	listened bool
	// buttons are the Button click callbacks, by helper widget id.
	buttons map[int64]func()
	// finals are the last TextInput and Checkbox readings the helper sent
	// when the Window closed, so they stay readable afterwards.
	finals map[int64]ahdGUIValue
}

// ahdGUIWidget is one widget: its Window, its helper id, and its kind.
// Label and Button texts are kept here, because only the program changes
// them.
type ahdGUIWidget struct {
	window   int64
	id       int64
	kind     string
	text     string
	disabled bool
}

var ahdGUI = struct {
	sync.Mutex
	next    int64
	windows map[int64]*ahdGUIWindow
	widgets map[int64]*ahdGUIWidget
}{windows: map[int64]*ahdGUIWindow{}, widgets: map[int64]*ahdGUIWidget{}}

func init() { AhdOnExit(AhdGUICloseAll) }

// ahdGUIDiscoverRuntime finds the bundled helper: the explicit override, the
// compiler's baked-in hint, then paths relative to the running executable.
// PATH is never searched.
func ahdGUIDiscoverRuntime() (string, error) {
	name := "ahdgui"
	if runtime.GOOS == "windows" {
		name = "ahdgui.exe"
	}
	candidates := []string{os.Getenv("AHDCODE_GUI_RUNTIME")}
	if AhdGUIRuntimeHint != "" {
		candidates = append(candidates, filepath.Join(AhdGUIRuntimeHint, name))
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(bin, name), filepath.Join(bin, "..", "libexec", "ahdcode", name))
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(filepath.Clean(candidate)); err == nil && info.Mode().IsRegular() {
			return filepath.Clean(candidate), nil
		}
	}
	return "", errors.New("the GUI window helper (ahdgui) was not found; reinstall AhdCode with its bundled helpers")
}

func ahdGUIValidText(text string, limit int) bool {
	return utf8.ValidString(text) && !strings.ContainsRune(text, 0) && utf8.RuneCountInString(text) <= limit
}

func ahdGUITextProblem(what string) string {
	return fmt.Sprintf("the %s must be text of at most %d characters", what, AhdGUIMaxTextRunes)
}

// AhdGUIWindowOpen validates a Window, starts its helper, and waits until the
// window is ready. It returns the Window handle.
func AhdGUIWindowOpen(title string, width, height int64) (int64, string) {
	if width < 1 || height < 1 || width > AhdGUIMaxDimension || height > AhdGUIMaxDimension {
		return 0, fmt.Sprintf("Window size %dx%d is not allowed; width and height must each be between 1 and %d", width, height, AhdGUIMaxDimension)
	}
	if !ahdGUIValidText(title, ahdGUIMaxTitleRunes) {
		return 0, fmt.Sprintf("the Window title must be text of at most %d characters", ahdGUIMaxTitleRunes)
	}
	headless := os.Getenv("AHDCODE_GUI_HEADLESS") == "1"
	var script json.RawMessage
	if text := os.Getenv("AHDCODE_GUI_HEADLESS_EVENTS"); headless && text != "" {
		// Automated tests stand in for the user on a headless Window.
		var list []json.RawMessage
		if json.Unmarshal([]byte(text), &list) != nil {
			return 0, "AHDCODE_GUI_HEADLESS_EVENTS is not a JSON list of events"
		}
		script = json.RawMessage(text)
	}
	path, err := ahdGUIDiscoverRuntime()
	if err != nil {
		return 0, err.Error()
	}
	link, err := ahdStartHelper(path, ahdGUIMaxLineBytes)
	if err != nil {
		return 0, "could not start the GUI window helper"
	}
	window := &ahdGUIWindow{link: link, usable: true, alive: true, headless: headless,
		buttons: map[int64]func(){}, finals: map[int64]ahdGUIValue{}}
	if _, problem := window.request(ahdGUIRequest{Op: "open", Version: ahdGUIProtocolVersion, Width: int(width),
		Height: int(height), Title: title, Headless: headless, Script: script}, ahdGUIOpenTimeout); problem != "" {
		window.mu.Lock()
		window.shutdown()
		window.mu.Unlock()
		return 0, "could not open the Window: " + problem
	}
	ahdGUI.Lock()
	ahdGUI.next++
	handle := ahdGUI.next
	ahdGUI.windows[handle] = window
	ahdGUI.Unlock()
	return handle, ""
}

// request sends one request and waits for its response. The caller holds
// w.mu, except during open, before the Window is shared.
func (w *ahdGUIWindow) request(value ahdGUIRequest, timeout time.Duration) (ahdGUIResponse, string) {
	if !w.alive {
		return ahdGUIResponse{}, ahdGUIClosedMessage
	}
	line, err := w.link.call(value, timeout)
	switch {
	case errors.Is(err, errAhdHelperSilent):
		w.shutdown()
		return ahdGUIResponse{}, "the GUI window helper stopped responding"
	case errors.Is(err, errAhdHelperInvalid):
		w.shutdown()
		return ahdGUIResponse{}, "the GUI window helper sent an invalid response"
	case err != nil:
		w.shutdown()
		return ahdGUIResponse{}, ahdGUIStoppedMessage
	}
	var reply ahdGUIResponse
	if json.Unmarshal(line, &reply) != nil {
		w.shutdown()
		return ahdGUIResponse{}, "the GUI window helper sent an invalid response"
	}
	if len(reply.Values) > 0 {
		w.remember(reply.Values)
	}
	if reply.Closed {
		w.usable = false
	}
	if !reply.OK {
		if reply.Error == "" {
			reply.Error = "the GUI window helper reported an unknown problem"
		}
		return reply, reply.Error
	}
	return reply, ""
}

func (w *ahdGUIWindow) remember(values []ahdGUIValue) {
	for _, value := range values {
		w.finals[value.Widget] = value
	}
}

func (w *ahdGUIWindow) shutdown() {
	if !w.alive {
		return
	}
	w.usable, w.alive = false, false
	w.link.shutdown(ahdGUICloseTimeout)
}

func ahdGUIWindowOf(handle int64) (*ahdGUIWindow, string) {
	ahdGUI.Lock()
	window := ahdGUI.windows[handle]
	ahdGUI.Unlock()
	if window == nil {
		return nil, ahdGUIClosedMessage
	}
	return window, ""
}

// ahdGUIWidgetOf finds a widget and its Window. A widget of a Window that was
// closed with close() is released and reports so.
func ahdGUIWidgetOf(handle int64) (*ahdGUIWidget, *ahdGUIWindow, string) {
	ahdGUI.Lock()
	widget := ahdGUI.widgets[handle]
	var window *ahdGUIWindow
	if widget != nil {
		window = ahdGUI.windows[widget.window]
	}
	ahdGUI.Unlock()
	if widget == nil || window == nil {
		return nil, nil, ahdGUIReleasedMessage
	}
	return widget, window, ""
}

// ahdGUIAdd creates one widget on a live Window and returns its handle.
func ahdGUIAdd(windowHandle int64, parent int64, kind string, value ahdGUIRequest) (int64, string) {
	window, problem := ahdGUIWindowOf(windowHandle)
	if problem != "" {
		return 0, problem
	}
	window.mu.Lock()
	if !window.usable {
		window.mu.Unlock()
		return 0, ahdGUIClosedMessage
	}
	if parent == 0 && window.root {
		window.mu.Unlock()
		return 0, "the Window already has its root Container; add further Containers to it"
	}
	value.Op, value.Parent, value.Kind = "add", parent, kind
	reply, problem := window.request(value, ahdGUIRequestTimeout)
	if problem == "" && parent == 0 {
		window.root = true
	}
	window.mu.Unlock()
	if problem != "" {
		return 0, problem
	}
	ahdGUI.Lock()
	ahdGUI.next++
	handle := ahdGUI.next
	ahdGUI.widgets[handle] = &ahdGUIWidget{window: windowHandle, id: reply.Widget, kind: kind, text: value.Text}
	ahdGUI.Unlock()
	return handle, ""
}

func ahdGUILayoutProblem(spacing, padding int64) string {
	if spacing < 0 || padding < 0 {
		return fmt.Sprintf("spacing and padding must be 0 or greater, not %d and %d", spacing, padding)
	}
	if spacing > AhdGUIMaxSpacing || padding > AhdGUIMaxSpacing {
		return fmt.Sprintf("spacing and padding must be at most %d", AhdGUIMaxSpacing)
	}
	return ""
}

// AhdGUIRoot creates a Window's one root Container, a "column" or a "row".
func AhdGUIRoot(window int64, kind string, spacing, padding int64) (int64, string) {
	if problem := ahdGUILayoutProblem(spacing, padding); problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(window, 0, kind, ahdGUIRequest{Spacing: int(spacing), Padding: int(padding)})
}

// ahdGUIContainer checks that a handle is a Container and returns its widget.
func ahdGUIContainer(container int64) (*ahdGUIWidget, string) {
	widget, _, problem := ahdGUIWidgetOf(container)
	if problem != "" {
		return nil, problem
	}
	if widget.kind != "column" && widget.kind != "row" {
		return nil, "only a Container holds widgets"
	}
	return widget, ""
}

// AhdGUIContainerLayout adds a nested Column or Row.
func AhdGUIContainerLayout(container int64, kind string, spacing, padding int64) (int64, string) {
	if problem := ahdGUILayoutProblem(spacing, padding); problem != "" {
		return 0, problem
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(parent.window, parent.id, kind, ahdGUIRequest{Spacing: int(spacing), Padding: int(padding)})
}

// AhdGUIAddLeaf adds a Label, Button, TextInput, or Checkbox.
func AhdGUIAddLeaf(container int64, kind, text, placeholder string, checked bool) (int64, string) {
	if !ahdGUIValidText(text, AhdGUIMaxTextRunes) {
		return 0, ahdGUITextProblem(kind + " text")
	}
	if !ahdGUIValidText(placeholder, AhdGUIMaxTextRunes) {
		return 0, ahdGUITextProblem("placeholder")
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	value := ahdGUIRequest{Text: text, Placeholder: placeholder}
	if kind == "checkbox" {
		value.Checked = &checked
	}
	return ahdGUIAdd(parent.window, parent.id, kind, value)
}

// AhdGUIText reads a Label's, Button's, or TextInput's text. A TextInput is
// asked for its current text; after its Window closed, the last text is
// returned.
func AhdGUIText(handle int64) (string, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return "", problem
	}
	if widget.kind != "textInput" {
		return widget.text, ""
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if final, ok := window.finals[widget.id]; ok || !window.alive {
		return final.Text, ""
	}
	reply, problem := window.request(ahdGUIRequest{Op: "get", Widget: widget.id}, ahdGUIRequestTimeout)
	if problem != "" {
		return "", problem
	}
	if reply.Text == nil {
		return "", "the GUI window helper sent an invalid response"
	}
	return *reply.Text, ""
}

// AhdGUISetText changes a Label's, Button's, or TextInput's text.
func AhdGUISetText(handle int64, text string) string {
	if !ahdGUIValidText(text, AhdGUIMaxTextRunes) {
		return ahdGUITextProblem("text")
	}
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if _, problem := window.request(ahdGUIRequest{Op: "set", Kind: "text", Widget: widget.id, Text: text}, ahdGUIRequestTimeout); problem != "" {
		return problem
	}
	ahdGUI.Lock()
	widget.text = text
	ahdGUI.Unlock()
	return ""
}

// AhdGUIChecked reads a Checkbox; after its Window closed, the last state.
func AhdGUIChecked(handle int64) (bool, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return false, problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if final, ok := window.finals[widget.id]; ok || !window.alive {
		return final.Checked, ""
	}
	reply, problem := window.request(ahdGUIRequest{Op: "get", Widget: widget.id}, ahdGUIRequestTimeout)
	if problem != "" {
		return false, problem
	}
	if reply.Checked == nil {
		return false, "the GUI window helper sent an invalid response"
	}
	return *reply.Checked, ""
}

// AhdGUISetChecked checks or clears a Checkbox.
func AhdGUISetChecked(handle int64, checked bool) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "set", Kind: "checked", Widget: widget.id, Checked: &checked}, ahdGUIRequestTimeout)
	return problem
}

// AhdGUISetTitle changes the Window title.
func AhdGUISetTitle(handle int64, title string) string {
	if !ahdGUIValidText(title, ahdGUIMaxTitleRunes) {
		return fmt.Sprintf("the Window title must be text of at most %d characters", ahdGUIMaxTitleRunes)
	}
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "title", Text: title}, ahdGUIRequestTimeout)
	return problem
}

// ahdGUIColor reads a color exactly as Graphics does -- one of the nine
// case-sensitive names, #RRGGBB, or #RRGGBBAA -- and returns the helper's
// normalized #RRGGBBAA form.
func ahdGUIColor(role, text string) (string, string) {
	value, ok := AhdGraphicsParseColor(text)
	if !ok {
		return "", ahdGraphicsColorProblem(role, text)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", value[0], value[1], value[2], value[3]), ""
}

// AhdGUIWindowSetBackground colors the Window behind every Container.
func AhdGUIWindowSetBackground(handle int64, color string) string {
	normalized, problem := ahdGUIColor("background", color)
	if problem != "" {
		return problem
	}
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "color", Part: "background", Color: normalized}, ahdGUIRequestTimeout)
	return problem
}

// AhdGUISetColor sets a Container's background, or a Label's, Button's,
// TextInput's, or Checkbox's foreground or background.
func AhdGUISetColor(handle int64, part, color string) string {
	normalized, problem := ahdGUIColor(part, color)
	if problem != "" {
		return problem
	}
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "color", Widget: widget.id, Part: part, Color: normalized}, ahdGUIRequestTimeout)
	return problem
}

// AhdGUISetEnabled enables or disables a Button, TextInput, or Checkbox.
func AhdGUISetEnabled(handle int64, enabled bool) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if _, problem := window.request(ahdGUIRequest{Op: "enabled", Widget: widget.id, Enabled: &enabled}, ahdGUIRequestTimeout); problem != "" {
		return problem
	}
	ahdGUI.Lock()
	widget.disabled = !enabled
	ahdGUI.Unlock()
	return ""
}

// AhdGUIIsEnabled reports whether a widget accepts user input. It answers
// from the runtime's own record, so it also works after the Window closed.
func AhdGUIIsEnabled(handle int64) (bool, string) {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return false, problem
	}
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	return !widget.disabled, ""
}

// AhdGUIOnClick registers a Button's click callback, replacing any earlier
// one.
func AhdGUIOnClick(handle int64, handler func()) string {
	if handler == nil {
		return "Button.onClick needs a Function"
	}
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if _, known := window.buttons[widget.id]; !known {
		if _, problem := window.request(ahdGUIRequest{Op: "listen", Kind: "click", Widget: widget.id}, ahdGUIRequestTimeout); problem != "" {
			return problem
		}
	}
	window.buttons[widget.id] = handler
	return ""
}

// AhdGUIOnKey registers a Window's key callback, replacing any earlier one.
func AhdGUIOnKey(handle int64, handler func(key string)) string {
	if handler == nil {
		return "Window.onKey needs a Function"
	}
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if !window.listened {
		if _, problem := window.request(ahdGUIRequest{Op: "listen", Kind: "key"}, ahdGUIRequestTimeout); problem != "" {
			return problem
		}
		window.listened = true
	}
	window.onKey = handler
	return ""
}

// AhdGUIWait blocks until the Window is closed by its user, running Button
// and key callbacks one at a time, in order, on the calling goroutine. On a
// Window that is already closed it returns at once. A callback's own error
// closes the Window and propagates unchanged. Events still queued when the
// window closes are discarded: their window is gone.
func AhdGUIWait(handle int64) string {
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return ""
	}
	window.mu.Lock()
	if !window.alive || !window.usable {
		window.mu.Unlock()
		return ""
	}
	// Output written before wait must be visible while the window is open.
	AhdFlush()
	_, problem = window.request(ahdGUIRequest{Op: "wait"}, ahdGUIRequestTimeout)
	window.mu.Unlock()
	if problem != "" {
		if problem == ahdGUIClosedMessage {
			return ""
		}
		return problem
	}
	for {
		line, kind := window.link.events.next(window.link.exited)
		window.mu.Lock()
		alive := window.alive
		stale := !window.headless && window.link.events.closedPending()
		switch kind {
		case ahdEventClosed:
			var closed ahdGUIEvent
			if json.Unmarshal(line, &closed) == nil {
				window.remember(closed.Values)
			}
			window.usable = false
			window.mu.Unlock()
			return ""
		case ahdEventOverflow:
			window.shutdown()
			window.mu.Unlock()
			return "the Window received more events than the program handled"
		case ahdEventExited:
			if alive {
				window.shutdown()
				window.mu.Unlock()
				return ahdGUIStoppedMessage
			}
			window.mu.Unlock()
			return ""
		}
		var event ahdGUIEvent
		if json.Unmarshal(line, &event) != nil {
			window.shutdown()
			window.mu.Unlock()
			return "the GUI window helper sent an invalid event"
		}
		var callback func()
		switch {
		case event.Event == "click" && window.buttons[event.Widget] != nil:
			handler := window.buttons[event.Widget]
			callback = handler
		case event.Event == "key" && window.onKey != nil:
			handler, key := window.onKey, event.Key
			callback = func() { handler(key) }
		}
		window.mu.Unlock()
		if !alive {
			return ""
		}
		if stale || callback == nil {
			continue
		}
		ahdGUIDispatch(handle, callback)
		window.mu.Lock()
		alive = window.alive
		if alive && window.headless {
			// A headless test Window replays its script one event at a time:
			// the next input happens only after this callback finished.
			_, problem = window.request(ahdGUIRequest{Op: "next"}, ahdGUIRequestTimeout)
		}
		window.mu.Unlock()
		if !alive {
			return ""
		}
		if problem != "" {
			return problem
		}
	}
}

// ahdGUIDispatch runs one callback. If it raises, the Window is closed and
// its helper released, and the error propagates unchanged.
func ahdGUIDispatch(handle int64, callback func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			AhdGUIClose(handle)
			panic(recovered)
		}
	}()
	callback()
	// What a callback writes is visible at once, not when the window closes.
	AhdFlush()
}

// AhdGUIClose closes the Window and ends its helper. Its widgets stay
// readable: TextInput and Checkbox values are their final readings. Closing
// a Window that is already closed does nothing.
func AhdGUIClose(handle int64) string {
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return ""
	}
	window.mu.Lock()
	if window.alive {
		_, _ = window.request(ahdGUIRequest{Op: "close"}, ahdGUICloseTimeout)
		window.shutdown()
	}
	window.usable = false
	window.mu.Unlock()
	return ""
}

// AhdGUIIsOpen reports whether the Window is still open. It asks the helper,
// because the user may have closed the window since the last call.
func AhdGUIIsOpen(handle int64) bool {
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return false
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return false
	}
	reply, problem := window.request(ahdGUIRequest{Op: "status"}, ahdGUIRequestTimeout)
	if problem != "" || reply.Open == nil || !*reply.Open {
		window.usable = false
		return false
	}
	return true
}

// AhdGUICloseAll closes every Window. It runs when a program ends, so no
// window or helper outlives the program that opened it.
func AhdGUICloseAll() {
	ahdGUI.Lock()
	handles := make([]int64, 0, len(ahdGUI.windows))
	for handle := range ahdGUI.windows {
		handles = append(handles, handle)
	}
	ahdGUI.Unlock()
	for _, handle := range handles {
		AhdGUIClose(handle)
	}
}

// ---- compiled-program wrappers ----

// AhdGUICheck raises a GUIError for a reported problem in a compiled program.
func AhdGUICheck(problem string) {
	if problem != "" {
		AhdRaiseClass(AhdClassGUIError, problem)
	}
}

func AhdGUIWindowChecked(title string, width, height int64) int64 {
	handle, problem := AhdGUIWindowOpen(title, width, height)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIRootChecked(window int64, kind string, spacing, padding int64) int64 {
	handle, problem := AhdGUIRoot(window, kind, spacing, padding)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIContainerLayoutChecked(container int64, kind string, spacing, padding int64) int64 {
	handle, problem := AhdGUIContainerLayout(container, kind, spacing, padding)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIAddLeafChecked(container int64, kind, text, placeholder string, checked bool) int64 {
	handle, problem := AhdGUIAddLeaf(container, kind, text, placeholder, checked)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIWindowSetBackgroundChecked(handle int64, color string) {
	AhdGUICheck(AhdGUIWindowSetBackground(handle, color))
}

func AhdGUISetColorChecked(handle int64, part, color string) {
	AhdGUICheck(AhdGUISetColor(handle, part, color))
}

func AhdGUISetEnabledChecked(handle int64, enabled bool) {
	AhdGUICheck(AhdGUISetEnabled(handle, enabled))
}

func AhdGUIIsEnabledChecked(handle int64) bool {
	enabled, problem := AhdGUIIsEnabled(handle)
	AhdGUICheck(problem)
	return enabled
}

func AhdGUITextChecked(handle int64) string {
	text, problem := AhdGUIText(handle)
	AhdGUICheck(problem)
	return text
}

func AhdGUICheckedChecked(handle int64) bool {
	checked, problem := AhdGUIChecked(handle)
	AhdGUICheck(problem)
	return checked
}
