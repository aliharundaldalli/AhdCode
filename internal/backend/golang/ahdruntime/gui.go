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
	ahdGUIProtocolVersion = 3
	// AhdGUIMaxDimension bounds a Window's width and height.
	AhdGUIMaxDimension  = 4096
	ahdGUIMaxTitleRunes = 256
	// AhdGUIMaxTextRunes bounds every widget text.
	AhdGUIMaxTextRunes = 4096
	// AhdGUIMaxSpacing bounds spacing and padding.
	AhdGUIMaxSpacing = 1000
	// ahdGUIMaxLineBytes bounds one helper line; the closed event carries
	// every text field's value, each bounded below.
	ahdGUIMaxLineBytes    = 32 << 20
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

	Parent      int64   `json:"parent,omitempty"`
	Widget      int64   `json:"widget,omitempty"`
	Kind        string  `json:"kind,omitempty"`
	Text        string  `json:"text,omitempty"`
	Placeholder string  `json:"placeholder,omitempty"`
	Checked     *bool   `json:"checked,omitempty"`
	Spacing     int     `json:"spacing,omitempty"`
	Padding     int     `json:"padding,omitempty"`
	Part        string  `json:"part,omitempty"`
	Color       string  `json:"color,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	Visible     *bool   `json:"visible,omitempty"`
	Size        string  `json:"size,omitempty"`
	Bold        *bool   `json:"bold,omitempty"`
	Clip        *string `json:"clip,omitempty"`

	Items      []string        `json:"items,omitempty"`
	Columns    []string        `json:"columns,omitempty"`
	Rows       [][]string      `json:"rows,omitempty"`
	Selected   *int            `json:"selected,omitempty"`
	Name       string          `json:"name,omitempty"`
	Extensions []string        `json:"extensions,omitempty"`
	Answer     json.RawMessage `json:"answer,omitempty"`

	// Drawing fields for an embedded Canvas. Zero is a real coordinate, so
	// the helper treats a missing number as zero. Stroke is a line width;
	// Fill is empty when a shape is not filled.
	X       float64 `json:"x,omitempty"`
	Y       float64 `json:"y,omitempty"`
	X2      float64 `json:"x2,omitempty"`
	Y2      float64 `json:"y2,omitempty"`
	Radius  float64 `json:"radius,omitempty"`
	StrokeW float64 `json:"stroke,omitempty"`
	Fill    string  `json:"fill,omitempty"`
}

type ahdGUIValue struct {
	Widget   int64  `json:"widget"`
	Text     string `json:"text,omitempty"`
	Checked  bool   `json:"checked,omitempty"`
	Selected *int   `json:"selected,omitempty"`
}

type ahdGUIResponse struct {
	OK        bool          `json:"ok"`
	Error     string        `json:"error,omitempty"`
	Closed    bool          `json:"closed,omitempty"`
	Open      *bool         `json:"open,omitempty"`
	Widget    int64         `json:"widget,omitempty"`
	Text      *string       `json:"text,omitempty"`
	Checked   *bool         `json:"checked,omitempty"`
	Enabled   *bool         `json:"enabled,omitempty"`
	Selected  *int          `json:"selected,omitempty"`
	Rows      [][]string    `json:"rows,omitempty"`
	Values    []ahdGUIValue `json:"values,omitempty"`
	Paths     []string      `json:"paths,omitempty"`
	Cancelled bool          `json:"cancelled,omitempty"`
	Confirmed *bool         `json:"confirmed,omitempty"`
}

type ahdGUIEvent struct {
	Event    string        `json:"event"`
	Widget   int64         `json:"widget,omitempty"`
	Key      string        `json:"key,omitempty"`
	Text     string        `json:"text,omitempty"`
	Checked  bool          `json:"checked,omitempty"`
	Selected *int          `json:"selected,omitempty"`
	Values   []ahdGUIValue `json:"values,omitempty"`
	// X and Y are a Canvas pointer position. Timer names one timer tick;
	// zero means every timer, which is what a scripted tick asks for.
	// Task, Value, and Text carry a finished background task.
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	Timer int64   `json:"timer,omitempty"`
	Task  int64   `json:"task,omitempty"`
	Value *int64  `json:"value,omitempty"`
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
	// buttons are the Button click callbacks, by helper widget id, and
	// changes the change and selection callbacks.
	buttons map[int64]func()
	// menus are menu-item callbacks, by helper item id. menuChecked is
	// the last check state, so checked() still answers after the Window
	// closes.
	menus       map[int64]func()
	menuChecked map[int64]bool
	changes     map[int64]func(ahdGUIEvent)
	// finals are the last readings of every widget the user can change,
	// sent by the helper when the Window closed, so they stay readable
	// afterwards.
	finals map[int64]ahdGUIValue
	// resizable records Window.setResizable.
	resizable bool
	// Timers repeat while the program waits. Tasks are this window's
	// background work. Canvas callbacks are pointer events, by helper id.
	timers      map[int64]*ahdGUITimer
	timerOrder  []int64
	nextTimer   int64
	waiting     bool
	pending     []*ahdGUITask
	tasks       []*ahdGUITask
	canvasMove  map[int64]func(float64, float64)
	canvasDrag  map[int64]func(float64, float64)
	canvasClick map[int64]func(float64, float64)
}

// ahdGUIWidget is one widget: its Window, its helper id, and its kind.
// Label and Button texts, ListBox and Select items, and TableView columns
// and rows are kept here, because only the program changes them.
type ahdGUIWidget struct {
	window   int64
	id       int64
	kind     string
	text     string
	disabled bool
	hidden   bool
	parent   int64
	items    []string
	columns  []string
	rows     [][]string
	// sortColumn is the TableView column the cached rows are ordered by,
	// or -1 when they are in the order last given to the helper. sortDesc
	// is that direction. The helper is the display; this copy stays aligned
	// with it.
	sortColumn int
	sortDesc   bool
	// maximum is a progress bar's upper bound. The helper keeps the value.
	maximum int
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
	if AhdPackagedApplication {
		if path, ok := ahdPackagedHelper(name); ok {
			return path, nil
		}
		return "", errors.New("the GUI window helper (ahdgui) is missing from this application")
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
		buttons: map[int64]func(){}, menus: map[int64]func(){}, menuChecked: map[int64]bool{}, changes: map[int64]func(ahdGUIEvent){}, finals: map[int64]ahdGUIValue{},
		canvasMove: map[int64]func(float64, float64){}, canvasDrag: map[int64]func(float64, float64){}, canvasClick: map[int64]func(float64, float64){}}
	ahdClipboard.Lock()
	existing := ahdClipboard.text
	ahdClipboard.Unlock()
	openRequest := ahdGUIRequest{Op: "open", Version: ahdGUIProtocolVersion, Width: int(width),
		Height: int(height), Title: title, Headless: headless, Script: script}
	if existing != "" {
		openRequest.Clip = &existing
	}
	if _, problem := window.request(openRequest, ahdGUIOpenTimeout); problem != "" {
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
	if ahdGUIInTask() {
		return ahdGUIResponse{}, "a background task cannot change the window"
	}
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
	w.stopTimers()
	w.cancelUnfinishedTasks()
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
func ahdGUIAdd(windowHandle, parentID, parentHandle int64, kind string, value ahdGUIRequest) (int64, string) {
	window, problem := ahdGUIWindowOf(windowHandle)
	if problem != "" {
		return 0, problem
	}
	window.mu.Lock()
	if !window.usable {
		window.mu.Unlock()
		return 0, ahdGUIClosedMessage
	}
	if parentID == 0 && window.root {
		window.mu.Unlock()
		return 0, "the Window already has its root Container; add further Containers to it"
	}
	value.Op, value.Parent, value.Kind = "add", parentID, kind
	reply, problem := window.request(value, ahdGUIRequestTimeout)
	if problem == "" && parentID == 0 {
		window.root = true
	}
	window.mu.Unlock()
	if problem != "" {
		return 0, problem
	}
	ahdGUI.Lock()
	ahdGUI.next++
	handle := ahdGUI.next
	ahdGUI.widgets[handle] = &ahdGUIWidget{window: windowHandle, id: reply.Widget, kind: kind, text: value.Text,
		parent: parentHandle, items: value.Items, columns: value.Columns, rows: value.Rows, sortColumn: -1}
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
	return ahdGUIAdd(window, 0, 0, kind, ahdGUIRequest{Spacing: int(spacing), Padding: int(padding)})
}

// ahdGUIContainer checks that a handle is a Container and returns its widget.
func ahdGUIContainer(container int64) (*ahdGUIWidget, string) {
	widget, _, problem := ahdGUIWidgetOf(container)
	if problem != "" {
		return nil, problem
	}
	if widget.kind != "column" && widget.kind != "row" && widget.kind != "grid" && widget.kind != "scroll" && widget.kind != "page" {
		return nil, "only a Container holds widgets"
	}
	return widget, ""
}

// AhdGUIGrid creates a grid Container. topLevel makes it the Window's root.
// columns is at least 1. columnGap is the request spacing and rowGap the
// request height; both use the spacing range.
func AhdGUIGrid(owner int64, topLevel bool, columns, columnGap, rowGap int64) (int64, string) {
	if columns < 1 || columns > 32 {
		return 0, "a grid needs between 1 and 32 columns"
	}
	if columnGap < 0 || rowGap < 0 || columnGap > AhdGUIMaxSpacing || rowGap > AhdGUIMaxSpacing {
		return 0, fmt.Sprintf("column and row gaps must be between 0 and %d", AhdGUIMaxSpacing)
	}
	value := ahdGUIRequest{Spacing: int(columnGap), Width: int(columns), Height: int(rowGap)}
	if topLevel {
		return ahdGUIAdd(owner, 0, 0, "grid", value)
	}
	parent, problem := ahdGUIContainer(owner)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(parent.window, parent.id, owner, "grid", value)
}

// AhdGUIContainerLayout adds a nested Column, Row, or scroll Container.
func AhdGUIContainerLayout(container int64, kind string, spacing, padding int64) (int64, string) {
	if problem := ahdGUILayoutProblem(spacing, padding); problem != "" {
		return 0, problem
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(parent.window, parent.id, container, kind, ahdGUIRequest{Spacing: int(spacing), Padding: int(padding)})
}

// AhdGUIAddLeaf adds a Label, Button, TextInput, Checkbox, PasswordInput,
// or TextArea.
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
	return ahdGUIAdd(parent.window, parent.id, container, kind, value)
}

// ahdGUIMaxImagePath bounds an Image path. ahdGUIMaxImageBytes bounds the
// file. The helper applies the same limits when it decodes the picture.
const (
	ahdGUIMaxImagePath  = 4096
	ahdGUIMaxImageBytes = 64 << 20
)

// ahdGUIImagePath checks a picture path and returns its absolute form.
// A relative path is the program's working directory. The helper's folder
// and an application bundle are not searched; GUI.resource is the explicit
// path into a resource folder.
func ahdGUIImagePath(path string) (string, string) {
	if path == "" || strings.ContainsRune(path, 0) || len(path) > ahdGUIMaxImagePath {
		return "", "image needs a file path"
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg":
	default:
		return "", "GUI images are .png, .jpg, and .jpeg files"
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "the image path cannot be resolved"
	}
	info, err := os.Stat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "could not read the image: the file does not exist"
		}
		return "", "could not read the image"
	}
	if !info.Mode().IsRegular() {
		return "", "could not read the image: it is not a file"
	}
	if info.Size() > ahdGUIMaxImageBytes {
		return "", "the image file is larger than 64 MiB"
	}
	return absolute, ""
}

// AhdGUIResource resolves one file inside the program's resource folder and
// returns that file's canonical absolute path. name is relative to the
// folder: it is not the working directory and it is not the user's home
// directory. The folder itself and the file are both resolved through any
// symbolic links, and the file's canonical path must stay inside the
// folder's canonical path. A link that points outside, including a directory
// link in the middle of the path, is a GUIError. A missing file, and a link
// whose target does not exist, are a GUIError. image() is unchanged and
// still uses the working directory.
//
// Resolving the links and the later open are not one filesystem operation.
// The returned path has no symbolic-link component at the moment it is
// resolved. A directory on that path can still be replaced before image()
// opens it, and the walk that resolves the links is not atomic.
func AhdGUIResource(name string) (string, string) {
	relative, problem := ahdResourceRelative(name)
	if problem != "" {
		return "", problem
	}
	root, problem := ahdResourceRoot()
	if problem != "" {
		return "", problem
	}
	canonicalRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "the resource folder cannot be found"
	}
	canonicalRoot, err = filepath.EvalSymlinks(canonicalRoot)
	if err != nil {
		return "", "the resource folder cannot be found"
	}
	info, err := os.Stat(canonicalRoot)
	if err != nil || !info.IsDir() {
		return "", "the resource folder cannot be found"
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(canonicalRoot, filepath.FromSlash(relative)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Sprintf("could not read the resource %s: the file does not exist", name)
		}
		return "", "could not read the resource " + name
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil || !ahdResourceInside(canonicalRoot, absolute) {
		return "", "the resource path leaves the resource folder"
	}
	info, err = os.Stat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Sprintf("could not read the resource %s: the file does not exist", name)
		}
		return "", "could not read the resource " + name
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Sprintf("could not read the resource %s: it is not a file", name)
	}
	return absolute, ""
}

// ahdResourceInside reports whether candidate is root or a path beneath it.
// Both paths must already be canonical. Components of the relative path
// decide, so a sibling such as resources-evil is not inside resources.
func ahdResourceInside(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	if relative == "." {
		return true
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func AhdGUIResourceChecked(name string) string {
	path, problem := AhdGUIResource(name)
	AhdGUICheck(problem)
	return path
}

// ahdResourceRelative rejects an empty, absolute, or parent-relative name.
func ahdResourceRelative(name string) (string, string) {
	if name == "" || strings.ContainsRune(name, 0) || len(name) > ahdGUIMaxImagePath {
		return "", "a resource needs a file path inside the resource folder"
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, `\`) || strings.HasPrefix(name, "/") {
		return "", "a resource path is relative to the resource folder"
	}
	slashed := strings.ReplaceAll(name, `\`, "/")
	parts := strings.Split(slashed, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", "a resource path cannot contain .."
		}
	}
	return slashed, ""
}

// ahdResourceRoot is the resource folder, never the working directory and
// never the home directory. AHDCODE_RESOURCE_ROOT wins. A packaged macOS
// application uses Contents/Resources/app beside its executable. Windows and
// Linux use resources beside the executable. Otherwise a resources folder
// beside the executable is used when it exists.
func ahdResourceRoot() (string, string) {
	if root := strings.TrimSpace(os.Getenv("AHDCODE_RESOURCE_ROOT")); root != "" {
		return filepath.Clean(root), ""
	}
	executable, err := os.Executable()
	if err != nil {
		return "", "the resource folder cannot be found"
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return ahdResourceRootFrom(executable, AhdPackagedApplication, runtime.GOOS)
}

// ahdResourceRootFrom is the platform half of ahdResourceRoot, so a packaged
// layout can be checked without launching a bundle.
func ahdResourceRootFrom(executable string, packaged bool, goos string) (string, string) {
	directory := filepath.Dir(executable)
	if packaged {
		if goos == "darwin" {
			return filepath.Clean(filepath.Join(directory, "..", "Resources", "app")), ""
		}
		return filepath.Join(directory, "resources"), ""
	}
	candidate := filepath.Join(directory, "resources")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate, ""
	}
	return "", "GUI.resource needs a resources folder next to the program, or AHDCODE_RESOURCE_ROOT"
}

// AhdGUIAddImage shows a PNG or JPEG file. width and height are the box in
// points; 0 means that side is not given. The picture is read once.
func AhdGUIAddImage(container int64, path string, width, height int64) (int64, string) {
	if width < 0 || height < 0 || width > AhdGUIMaxDimension || height > AhdGUIMaxDimension {
		return 0, fmt.Sprintf("image width and height must be between 0 and %d", AhdGUIMaxDimension)
	}
	absolute, problem := ahdGUIImagePath(path)
	if problem != "" {
		return 0, problem
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(parent.window, parent.id, container, "image", ahdGUIRequest{Text: absolute, Width: int(width), Height: int(height)})
}

// ahdGUITextField reports whether a kind holds text the user edits.
func ahdGUITextField(kind string) bool {
	return kind == "textInput" || kind == "passwordInput" || kind == "textArea"
}

// AhdGUIText reads a widget's text. A text field is asked for its current
// text; after its Window closed, the last text is returned.
func AhdGUIText(handle int64) (string, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return "", problem
	}
	if !ahdGUITextField(widget.kind) {
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

// AhdGUISetText changes a widget's text. A TextArea's line breaks are
// normalized to "\n".
func AhdGUISetText(handle int64, text string) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind == "textArea" {
		text = ahdGUIAreaText(text)
		if !ahdGUIValidText(text, AhdGUIMaxAreaRunes) {
			return fmt.Sprintf("a TextArea holds text of at most %d characters", AhdGUIMaxAreaRunes)
		}
	} else if !ahdGUIValidText(text, AhdGUIMaxTextRunes) {
		return ahdGUITextProblem("text")
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

// ahdGUISetBox sends one layout request: min, preferred, align, or span.
func ahdGUISetBox(handle int64, kind string, width, height int, horizontal, vertical string) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "box", Kind: kind, Widget: widget.id, Width: width, Height: height, Text: horizontal, Size: vertical}, ahdGUIRequestTimeout)
	return problem
}

// AhdGUISetMinWidth raises a widget's natural width. Zero clears it.
func AhdGUISetMinWidth(handle int64, width int64) string {
	if width < 0 || width > AhdGUIMaxDimension {
		return fmt.Sprintf("minimum width must be between 0 and %d", AhdGUIMaxDimension)
	}
	return ahdGUISetBox(handle, "min", int(width), 0, "", "")
}

// AhdGUISetPreferredWidth raises a widget's natural width. A value below the
// content's own width does nothing. Zero clears it.
func AhdGUISetPreferredWidth(handle int64, width int64) string {
	if width < 0 || width > AhdGUIMaxDimension {
		return fmt.Sprintf("preferred width must be between 0 and %d", AhdGUIMaxDimension)
	}
	return ahdGUISetBox(handle, "preferred", int(width), 0, "", "")
}

// AhdGUISetAlign sets horizontal and vertical alignment. vertical may be empty.
func AhdGUISetAlign(handle int64, horizontal, vertical string) string {
	switch horizontal {
	case "start", "center", "end", "stretch":
	default:
		return `alignment must be "start", "center", "end", or "stretch"`
	}
	switch vertical {
	case "", "start", "center", "end", "stretch":
	default:
		return `alignment must be "start", "center", "end", or "stretch"`
	}
	return ahdGUISetBox(handle, "align", 0, 0, horizontal, vertical)
}

// AhdGUISetSpan sets how many grid columns and rows a widget covers.
func AhdGUISetSpan(handle int64, columns, rows int64) string {
	if columns < 1 || columns > 32 || rows < 1 || rows > 32 {
		return "a span must be between 1 and 32"
	}
	return ahdGUISetBox(handle, "span", int(columns), int(rows), "", "")
}

// AhdGUISetVisible hides or shows a widget or Container. A hidden widget
// takes no space in its Column, Row, or grid.
func AhdGUISetVisible(handle int64, visible bool) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if _, problem := window.request(ahdGUIRequest{Op: "visible", Widget: widget.id, Visible: &visible}, ahdGUIRequestTimeout); problem != "" {
		return problem
	}
	ahdGUI.Lock()
	widget.hidden = !visible
	ahdGUI.Unlock()
	return ""
}

// AhdGUIIsVisible reports whether a widget is shown. A hidden Container
// hides everything inside it. The answer comes from the runtime's own
// record, so it also works after the Window closed.
func AhdGUIIsVisible(handle int64) (bool, string) {
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	for handle != 0 {
		widget := ahdGUI.widgets[handle]
		if widget == nil {
			return false, ahdGUIReleasedMessage
		}
		if widget.hidden {
			return false, ""
		}
		handle = widget.parent
	}
	return true, ""
}

// AhdGUISetLabelSize sets a Label to small, normal, large, or title.
func AhdGUISetLabelSize(handle int64, size string) string {
	switch size {
	case "small", "normal", "large", "title":
	default:
		return `label size must be "small", "normal", "large", or "title"`
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
	_, problem = window.request(ahdGUIRequest{Op: "label", Widget: widget.id, Size: size}, ahdGUIRequestTimeout)
	return problem
}

// AhdGUISetWrap turns wrapping on or off for a Label. The default is off.
func AhdGUISetWrap(handle int64, wrap bool) string {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "label" {
		return "setWrap is for a Label"
	}
	width := 0
	if wrap {
		width = 1
	}
	return ahdGUISetBox(handle, "wrap", width, 0, "", "")
}

// AhdGUISetImageSize changes an Image's box. Zero on a side means that side
// is not given, so the picture's proportions decide it.
func AhdGUISetImageSize(handle int64, width, height int64) string {
	if width < 0 || height < 0 || width > AhdGUIMaxDimension || height > AhdGUIMaxDimension {
		return fmt.Sprintf("image width and height must be between 0 and %d", AhdGUIMaxDimension)
	}
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "image" {
		return "setSize with a width and a height is for an Image"
	}
	return ahdGUISetBox(handle, "fit", int(width), int(height), "", "")
}

// AhdGUISetLabelBold draws a Label heavier, or returns it to the regular weight.
func AhdGUISetLabelBold(handle int64, bold bool) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "label", Widget: widget.id, Bold: &bold}, ahdGUIRequestTimeout)
	return problem
}

// ahdGUIMaxClipboardRunes bounds GUI.setClipboard.
const ahdGUIMaxClipboardRunes = 100_000

var ahdClipboard struct {
	sync.Mutex
	text string
}

// AhdGUIClipboard is the program's text clipboard.
func AhdGUIClipboard() string {
	ahdClipboard.Lock()
	defer ahdClipboard.Unlock()
	return ahdClipboard.text
}

// AhdGUISetClipboard replaces the program's text clipboard.
func AhdGUISetClipboard(text string) string {
	if !ahdGUIValidText(text, ahdGUIMaxClipboardRunes) {
		return fmt.Sprintf("clipboard text is at most %d characters", ahdGUIMaxClipboardRunes)
	}
	ahdClipboard.Lock()
	ahdClipboard.text = text
	ahdClipboard.Unlock()
	ahdGUIPushClipboard(text)
	return ""
}

// ahdGUIPushClipboard copies the program clipboard into every open window.
func ahdGUIPushClipboard(text string) { ahdGUIPushClipboardExcept(0, text) }

// ahdGUIPushClipboardExcept updates every open window except one. The
// window a copy came from already holds the text.
func ahdGUIPushClipboardExcept(except int64, text string) {
	ahdGUI.Lock()
	windows := make([]struct {
		handle int64
		window *ahdGUIWindow
	}, 0, len(ahdGUI.windows))
	for handle, window := range ahdGUI.windows {
		if handle != except {
			windows = append(windows, struct {
				handle int64
				window *ahdGUIWindow
			}{handle, window})
		}
	}
	ahdGUI.Unlock()
	clip := text
	for _, item := range windows {
		item.window.mu.Lock()
		if item.window.usable {
			_, _ = item.window.request(ahdGUIRequest{Op: "clipboard", Clip: &clip}, ahdGUIRequestTimeout)
		}
		item.window.mu.Unlock()
	}
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

// AhdGUIWait blocks until the Window is closed by its user, running click,
// change, selection, and key callbacks one at a time, in order, on the calling goroutine. On a
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
	window.startTasks()
	defer ahdGUIJoinTasks(handle)
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
			window.stopTimers()
			window.cancelUnfinishedTasks()
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
		var ticks []func()
		advance := true
		sorted := false
		switch {
		case event.Event == "tick":
			ticks = window.dueTimers(event.Timer)
		case event.Event == "task":
			advance = false
			if handler := window.taskHandler(event.Task); handler != nil && !window.taskCancelled(event.Task) {
				value, problemText := event.Value, event.Text
				callback = func() {
					var problem *string
					if problemText != "" {
						problem = &problemText
					}
					handler(value, problem)
				}
			}
		case event.Event == "move" && window.canvasMove[event.Widget] != nil:
			handler, x, y := window.canvasMove[event.Widget], event.X, event.Y
			callback = func() { handler(x, y) }
		case event.Event == "drag" && window.canvasDrag[event.Widget] != nil:
			handler, x, y := window.canvasDrag[event.Widget], event.X, event.Y
			callback = func() { handler(x, y) }
		case event.Event == "click" && window.canvasClick[event.Widget] != nil:
			handler, x, y := window.canvasClick[event.Widget], event.X, event.Y
			callback = func() { handler(x, y) }
		case event.Event == "click" && window.buttons[event.Widget] != nil:
			handler := window.buttons[event.Widget]
			callback = handler
		case event.Event == "menu" && window.menus[event.Widget] != nil:
			window.menuChecked[event.Widget] = event.Checked
			handler := window.menus[event.Widget]
			callback = handler
		case event.Event == "sort":
			ahdGUINoteSort(handle, event)
			sorted = true
		case event.Event == "change" && window.changes[event.Widget] != nil:
			handler, changed := window.changes[event.Widget], event
			callback = func() { handler(changed) }
		case event.Event == "key" && window.onKey != nil:
			handler, key := window.onKey, event.Key
			callback = func() { handler(key) }
		case event.Event == "clipboard":
			copied := event.Text
			callback = func() {
				if !ahdGUIValidText(copied, ahdGUIMaxClipboardRunes) {
					return
				}
				ahdClipboard.Lock()
				ahdClipboard.text = copied
				ahdClipboard.Unlock()
				ahdGUIPushClipboardExcept(handle, copied)
			}
		}
		window.mu.Unlock()
		if !alive {
			return ""
		}
		if stale {
			continue
		}
		if event.Event == "tick" {
			for _, tick := range ticks {
				ahdGUIDispatch(handle, tick)
				window.mu.Lock()
				alive = window.alive
				window.mu.Unlock()
				if !alive {
					return ""
				}
			}
		} else if callback != nil {
			ahdGUIDispatch(handle, callback)
			window.mu.Lock()
			alive = window.alive
			window.mu.Unlock()
			if !alive {
				return ""
			}
		} else if event.Event != "tick" && !sorted {
			continue
		}
		if advance && window.headless {
			// A headless test Window replays its script one event at a time.
			// A task completion is not one of those events.
			window.mu.Lock()
			if window.alive {
				_, problem = window.request(ahdGUIRequest{Op: "next"}, ahdGUIRequestTimeout)
			}
			alive = window.alive
			window.mu.Unlock()
			if !alive {
				return ""
			}
			if problem != "" {
				return problem
			}
		}
	}
}

// ahdGUIDispatch runs one callback. If it raises, the Window is closed and
// its helper released, and the error propagates unchanged.
func ahdGUIDispatch(handle int64, callback func()) {
	// The callback lock is the GUI turn. A background task holds it while
	// its function runs and lets it go at GUI.checkpoint, so callbacks and
	// the task never touch AhdCode values at the same time.
	ahdGUICallbackLock.Lock()
	defer ahdGUICallbackLock.Unlock()
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
const ahdGUIMaxMenuRunes = 128

func ahdGUIRemember(windowHandle int64, widget ahdGUIWidget) int64 {
	ahdGUI.Lock()
	ahdGUI.next++
	handle := ahdGUI.next
	widget.window = windowHandle
	ahdGUI.widgets[handle] = &widget
	ahdGUI.Unlock()
	return handle
}

// AhdGUIMenu adds a menu to a Window, or a submenu to a Menu. topLevel
// selects which handle owner is.
func AhdGUIMenu(owner int64, title string, topLevel bool) (int64, string) {
	if title == "" || !ahdGUIValidText(title, ahdGUIMaxMenuRunes) {
		return 0, "a menu title must be text of at most 128 characters"
	}
	var window *ahdGUIWindow
	var parentID, windowHandle, parentHandle int64
	var problem string
	if topLevel {
		window, problem = ahdGUIWindowOf(owner)
		windowHandle = owner
	} else {
		var widget *ahdGUIWidget
		widget, window, problem = ahdGUIWidgetOf(owner)
		if problem == "" && widget.kind != "menu" {
			return 0, "a submenu belongs to a Menu"
		}
		if problem == "" {
			parentID = widget.id
			windowHandle = widget.window
			parentHandle = owner
		}
	}
	if problem != "" {
		return 0, problem
	}
	window.mu.Lock()
	if !window.usable {
		window.mu.Unlock()
		return 0, ahdGUIClosedMessage
	}
	reply, problem := window.request(ahdGUIRequest{Op: "menu", Kind: "menu", Parent: parentID, Text: title}, ahdGUIRequestTimeout)
	window.mu.Unlock()
	if problem != "" {
		return 0, problem
	}
	return ahdGUIRemember(windowHandle, ahdGUIWidget{id: reply.Widget, kind: "menu", text: title, parent: parentHandle}), ""
}

// AhdGUIMenuItem adds an item or a check item. kind is "item" or "check".
func AhdGUIMenuItem(menu int64, kind, label, shortcut string, checked bool) (int64, string) {
	if label == "" || !ahdGUIValidText(label, ahdGUIMaxMenuRunes) {
		return 0, "a menu item label must be text of at most 128 characters"
	}
	if utf8.RuneCountInString(shortcut) > 64 {
		return 0, "a menu shortcut is too long"
	}
	widget, window, problem := ahdGUIWidgetOf(menu)
	if problem != "" {
		return 0, problem
	}
	if widget.kind != "menu" {
		return 0, "a menu item belongs to a Menu"
	}
	window.mu.Lock()
	if !window.usable {
		window.mu.Unlock()
		return 0, ahdGUIClosedMessage
	}
	reply, problem := window.request(ahdGUIRequest{Op: "menu", Kind: kind, Parent: widget.id, Text: label, Name: shortcut, Checked: &checked}, ahdGUIRequestTimeout)
	if problem == "" && kind == "check" {
		window.menuChecked[reply.Widget] = checked
	}
	window.mu.Unlock()
	if problem != "" {
		return 0, problem
	}
	stored := "menuItem"
	if kind == "check" {
		stored = "menuCheck"
	}
	return ahdGUIRemember(widget.window, ahdGUIWidget{id: reply.Widget, kind: stored, text: label, parent: menu}), ""
}

// AhdGUIMenuSeparator adds a separator to a Menu.
func AhdGUIMenuSeparator(menu int64) string {
	widget, window, problem := ahdGUIWidgetOf(menu)
	if problem != "" {
		return problem
	}
	if widget.kind != "menu" {
		return "a menu item belongs to a Menu"
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "menu", Kind: "separator", Parent: widget.id}, ahdGUIRequestTimeout)
	return problem
}

func ahdGUIMenuItem(handle int64) (*ahdGUIWidget, *ahdGUIWindow, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return nil, nil, problem
	}
	if widget.kind != "menuItem" && widget.kind != "menuCheck" {
		return nil, nil, "only a menu item supports that request"
	}
	return widget, window, ""
}

// AhdGUIMenuSetEnabled enables or disables a menu item. A disabled item
// keeps its shortcut and the shortcut does not run.
func AhdGUIMenuSetEnabled(handle int64, enabled bool) string {
	widget, window, problem := ahdGUIMenuItem(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "menu", Kind: "enabled", Widget: widget.id, Enabled: &enabled}, ahdGUIRequestTimeout)
	return problem
}

// AhdGUIMenuIsEnabled reports whether a menu item is enabled.
func AhdGUIMenuIsEnabled(handle int64) (bool, string) {
	widget, window, problem := ahdGUIMenuItem(handle)
	if problem != "" {
		return false, problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return false, ahdGUIClosedMessage
	}
	reply, problem := window.request(ahdGUIRequest{Op: "menu", Kind: "enabled", Widget: widget.id}, ahdGUIRequestTimeout)
	if problem != "" {
		return false, problem
	}
	return reply.Enabled != nil && *reply.Enabled, ""
}

// AhdGUIMenuSetChecked sets a check item. A plain item reports an error.
func AhdGUIMenuSetChecked(handle int64, checked bool) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "menuCheck" {
		return "only a checkable menu item has a check"
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "menu", Kind: "checked", Widget: widget.id, Checked: &checked}, ahdGUIRequestTimeout)
	if problem == "" {
		window.menuChecked[widget.id] = checked
	}
	return problem
}

// AhdGUIMenuChecked reads a check item.
func AhdGUIMenuChecked(handle int64) (bool, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return false, problem
	}
	if widget.kind != "menuCheck" {
		return false, "only a checkable menu item has a check"
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return window.menuChecked[widget.id], ""
	}
	reply, problem := window.request(ahdGUIRequest{Op: "menu", Kind: "checked", Widget: widget.id}, ahdGUIRequestTimeout)
	if problem != "" {
		return false, problem
	}
	checked := reply.Checked != nil && *reply.Checked
	window.menuChecked[widget.id] = checked
	return checked, ""
}

// AhdGUIMenuSetText replaces a menu title or an item label.
func AhdGUIMenuSetText(handle int64, text string) string {
	if text == "" || !ahdGUIValidText(text, ahdGUIMaxMenuRunes) {
		return "a menu label must be text of at most 128 characters"
	}
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "menu" && widget.kind != "menuItem" && widget.kind != "menuCheck" {
		return "only a menu item supports that request"
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	_, problem = window.request(ahdGUIRequest{Op: "menu", Kind: "text", Widget: widget.id, Text: text}, ahdGUIRequestTimeout)
	if problem == "" {
		widget.text = text
	}
	return problem
}

// AhdGUIOnMenu registers a menu item's callback, replacing any earlier one.
func AhdGUIOnMenu(handle int64, handler func()) string {
	if handler == nil {
		return "MenuItem.onClick needs a Function"
	}
	widget, window, problem := ahdGUIMenuItem(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if _, known := window.menus[widget.id]; !known {
		if _, problem := window.request(ahdGUIRequest{Op: "listen", Kind: "menu", Widget: widget.id}, ahdGUIRequestTimeout); problem != "" {
			return problem
		}
	}
	window.menus[widget.id] = handler
	return ""
}

func AhdGUIAddMenuChecked(owner int64, title string, topLevel bool) int64 {
	handle, problem := AhdGUIMenu(owner, title, topLevel)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIMenuItemChecked(menu int64, kind, label, shortcut string, checked bool) int64 {
	handle, problem := AhdGUIMenuItem(menu, kind, label, shortcut, checked)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIMenuSeparatorChecked(menu int64) { AhdGUICheck(AhdGUIMenuSeparator(menu)) }

func AhdGUIMenuSetEnabledChecked(handle int64, enabled bool) {
	AhdGUICheck(AhdGUIMenuSetEnabled(handle, enabled))
}

func AhdGUIMenuIsEnabledChecked(handle int64) bool {
	enabled, problem := AhdGUIMenuIsEnabled(handle)
	AhdGUICheck(problem)
	return enabled
}

func AhdGUIMenuSetCheckedChecked(handle int64, checked bool) {
	AhdGUICheck(AhdGUIMenuSetChecked(handle, checked))
}

func AhdGUIMenuCheckedChecked(handle int64) bool {
	checked, problem := AhdGUIMenuChecked(handle)
	AhdGUICheck(problem)
	return checked
}

func AhdGUIMenuSetTextChecked(handle int64, text string) {
	AhdGUICheck(AhdGUIMenuSetText(handle, text))
}

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

func AhdGUISetVisibleChecked(handle int64, visible bool) {
	AhdGUICheck(AhdGUISetVisible(handle, visible))
}

func AhdGUIGridChecked(owner int64, topLevel bool, columns, columnGap, rowGap int64) int64 {
	handle, problem := AhdGUIGrid(owner, topLevel, columns, columnGap, rowGap)
	AhdGUICheck(problem)
	return handle
}

func AhdGUISetMinWidthChecked(handle int64, width int64) {
	AhdGUICheck(AhdGUISetMinWidth(handle, width))
}

func AhdGUISetPreferredWidthChecked(handle int64, width int64) {
	AhdGUICheck(AhdGUISetPreferredWidth(handle, width))
}

func AhdGUISetAlignChecked(handle int64, horizontal, vertical string) {
	AhdGUICheck(AhdGUISetAlign(handle, horizontal, vertical))
}

func AhdGUISetSpanChecked(handle int64, columns, rows int64) {
	AhdGUICheck(AhdGUISetSpan(handle, columns, rows))
}

func AhdGUIIsVisibleChecked(handle int64) bool {
	visible, problem := AhdGUIIsVisible(handle)
	AhdGUICheck(problem)
	return visible
}

func AhdGUISetLabelSizeChecked(handle int64, size string) {
	AhdGUICheck(AhdGUISetLabelSize(handle, size))
}

func AhdGUISetLabelBoldChecked(handle int64, bold bool) {
	AhdGUICheck(AhdGUISetLabelBold(handle, bold))
}

func AhdGUIAddImageChecked(container int64, path string, width, height int64) int64 {
	handle, problem := AhdGUIAddImage(container, path, width, height)
	AhdGUICheck(problem)
	return handle
}

func AhdGUISetWrapChecked(handle int64, wrap bool) {
	AhdGUICheck(AhdGUISetWrap(handle, wrap))
}

func AhdGUISetImageSizeChecked(handle int64, width, height int64) {
	AhdGUICheck(AhdGUISetImageSize(handle, width, height))
}

func AhdGUISetClipboardChecked(text string) {
	AhdGUICheck(AhdGUISetClipboard(text))
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
