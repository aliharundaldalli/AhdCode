package ahdruntime

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var (
	guiHelperOnce sync.Once
	guiHelperPath string
	guiHelperErr  error
)

// useGUIHelper builds the real ahdgui helper once and points the runtime at
// it in headless mode, so these tests need no display. script is the user
// input the helper replays while the program waits.
func useGUIHelper(t *testing.T, script string) {
	t.Helper()
	guiHelperOnce.Do(func() {
		directory, err := os.MkdirTemp("", "ahdgui-test-")
		if err != nil {
			guiHelperErr = err
			return
		}
		name := "ahdgui"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		guiHelperPath = filepath.Join(directory, name)
		root, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "cmd", "ahdgui"))
		command := exec.Command("go", "build", "-o", guiHelperPath, ".")
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			guiHelperErr = errors.New(err.Error() + "\n" + string(output))
		}
	})
	if guiHelperErr != nil {
		t.Fatalf("building ahdgui: %v", guiHelperErr)
	}
	t.Setenv("AHDCODE_GUI_RUNTIME", guiHelperPath)
	t.Setenv("AHDCODE_GUI_HEADLESS", "1")
	t.Setenv("AHDCODE_GUI_HEADLESS_EVENTS", script)
}

func openTestWindow(t *testing.T, script string) int64 {
	t.Helper()
	useGUIHelper(t, script)
	handle, problem := AhdGUIWindowOpen("test", 400, 300)
	if problem != "" {
		t.Fatalf("open: %s", problem)
	}
	t.Cleanup(func() { AhdGUIClose(handle) })
	return handle
}

func guiString(t *testing.T) func(string, string) string {
	return func(value, problem string) string {
		t.Helper()
		if problem != "" {
			t.Fatalf("unexpected GUIError: %s", problem)
		}
		return value
	}
}

func guiBool(t *testing.T) func(bool, string) bool {
	return func(value bool, problem string) bool {
		t.Helper()
		if problem != "" {
			t.Fatalf("unexpected GUIError: %s", problem)
		}
		return value
	}
}

func guiMustNot(t *testing.T, problem, fragment string) {
	t.Helper()
	if !strings.Contains(problem, fragment) {
		t.Fatalf("problem %q, want one containing %q", problem, fragment)
	}
}

func TestGUIWindowValidation(t *testing.T) {
	useGUIHelper(t, "")
	for _, size := range [][2]int64{{0, 100}, {100, 0}, {-5, 100}, {100, -1}, {4097, 10}, {10, 4097}} {
		if _, problem := AhdGUIWindowOpen("t", size[0], size[1]); !strings.Contains(problem, "between 1 and 4096") {
			t.Errorf("size %v: %q", size, problem)
		}
	}
	if _, problem := AhdGUIWindowOpen(strings.Repeat("x", 257), 10, 10); !strings.Contains(problem, "title") {
		t.Errorf("long title: %q", problem)
	}
	handle := guiMust2(t)(AhdGUIWindowOpen("Başlık ✓", 4096, 1))
	defer AhdGUIClose(handle)
	if !AhdGUIIsOpen(handle) {
		t.Fatal("a new Window is not open")
	}
}

func guiMust2(t *testing.T) func(int64, string) int64 {
	return func(value int64, problem string) int64 {
		t.Helper()
		if problem != "" {
			t.Fatalf("unexpected GUIError: %s", problem)
		}
		return value
	}
}

func TestGUIRootAndLayoutRules(t *testing.T) {
	ok := guiMust2(t)
	window := openTestWindow(t, "")
	if _, problem := AhdGUIRoot(window, "column", -1, 0); problem == "" {
		t.Fatal("negative spacing accepted")
	}
	if _, problem := AhdGUIRoot(window, "row", 0, -1); problem == "" {
		t.Fatal("negative padding accepted")
	}
	root := ok(AhdGUIRoot(window, "row", 0, 0))
	_, problem := AhdGUIRoot(window, "column", 8, 12)
	guiMustNot(t, problem, "already has its root Container")
	column := ok(AhdGUIContainerLayout(root, "column", 8, 0))
	ok(AhdGUIContainerLayout(column, "row", 0, 0))
	ok(AhdGUIContainerLayout(column, "column", 0, 0))
	row := ok(AhdGUIContainerLayout(root, "row", 4, 2))
	label := ok(AhdGUIAddLeaf(row, "label", "L", "", false))
	if _, problem := AhdGUIContainerLayout(label, "row", 0, 0); !strings.Contains(problem, "only a Container") {
		t.Fatalf("a Label holding widgets: %q", problem)
	}
	if _, problem := AhdGUIAddLeaf(label, "button", "x", "", false); !strings.Contains(problem, "only a Container") {
		t.Fatalf("a widget inside a Label: %q", problem)
	}
}

func TestGUIWidgetValues(t *testing.T) {
	ok := guiMust2(t)
	window := openTestWindow(t, "")
	root := ok(AhdGUIRoot(window, "column", 8, 12))
	label := ok(AhdGUIAddLeaf(root, "label", "Hello", "", false))
	button := ok(AhdGUIAddLeaf(root, "button", "Save", "", false))
	input := ok(AhdGUIAddLeaf(root, "textInput", "", "placeholder", false))
	off := ok(AhdGUIAddLeaf(root, "checkbox", "Off", "", false))
	on := ok(AhdGUIAddLeaf(root, "checkbox", "On", "", true))
	for _, check := range []struct {
		handle int64
		want   string
	}{{label, "Hello"}, {button, "Save"}, {input, ""}} {
		if got := guiString(t)(AhdGUIText(check.handle)); got != check.want {
			t.Fatalf("text %q, want %q", got, check.want)
		}
	}
	if guiBool(t)(AhdGUIChecked(off)) || !guiBool(t)(AhdGUIChecked(on)) {
		t.Fatal("initial Checkbox states")
	}
	for _, handle := range []int64{label, button, input} {
		if problem := AhdGUISetText(handle, "ğüşİ ✓"); problem != "" {
			t.Fatal(problem)
		}
		if got := guiString(t)(AhdGUIText(handle)); got != "ğüşİ ✓" {
			t.Fatalf("setText/text %q", got)
		}
	}
	if problem := AhdGUISetChecked(off, true); problem != "" || !guiBool(t)(AhdGUIChecked(off)) {
		t.Fatal("setChecked")
	}
	if problem := AhdGUISetText(label, strings.Repeat("x", AhdGUIMaxTextRunes+1)); !strings.Contains(problem, "at most") {
		t.Fatalf("oversized text: %q", problem)
	}
	if problem := AhdGUISetTitle(window, "Yeni başlık"); problem != "" {
		t.Fatal(problem)
	}
}

func TestGUIWindowsAreIndependentAndCloseCleanly(t *testing.T) {
	ok := guiMust2(t)
	first := openTestWindow(t, "")
	second := openTestWindow(t, "")
	firstInput := ok(AhdGUIAddLeaf(ok(AhdGUIRoot(first, "column", 0, 0)), "textInput", "", "", false))
	secondRoot := ok(AhdGUIRoot(second, "column", 0, 0))
	must(t, AhdGUISetText(firstInput, "kept"))
	record, _ := ahdGUIWindowOf(first)
	if AhdGUIClose(first) != "" || AhdGUIClose(first) != "" {
		t.Fatal("close is not idempotent")
	}
	if AhdGUIIsOpen(first) || !AhdGUIIsOpen(second) {
		t.Fatal("closing one Window affected the other")
	}
	if record.link.process.ProcessState == nil {
		t.Fatal("the closed Window's helper was not reaped")
	}
	if got := guiString(t)(AhdGUIText(firstInput)); got != "kept" {
		t.Fatalf("a closed Window's TextInput reads %q", got)
	}
	guiMustNot(t, AhdGUISetText(firstInput, "late"), "closed")
	_, problem := AhdGUIAddLeaf(secondRoot, "label", "still alive", "", false)
	if problem != "" {
		t.Fatal(problem)
	}
	if AhdGUIWait(first) != "" {
		t.Fatal("wait on a closed Window")
	}
}

// TestGUICallbacksRunSeriallyAndCanUpdateWidgets replays typing, a toggle,
// Button clicks, and keys, and checks the callbacks see the values at the
// moment of each click, can change widgets synchronously, and never overlap.
func TestGUICallbacksRunSeriallyAndCanUpdateWidgets(t *testing.T) {
	ok := guiMust2(t)
	// Widget ids follow creation order: root 1, input 2, check 3, button 4, label 5.
	window := openTestWindow(t, `[{"event":"type","widget":2,"text":"Ali"},{"event":"click","widget":4},`+
		`{"event":"toggle","widget":3},{"event":"key","key":"ArrowUp"},{"event":"click","widget":4},{"event":"key","key":"A"}]`)
	root := ok(AhdGUIRoot(window, "column", 8, 12))
	input := ok(AhdGUIAddLeaf(root, "textInput", "", "name", false))
	check := ok(AhdGUIAddLeaf(root, "checkbox", "Paid", "", false))
	button := ok(AhdGUIAddLeaf(root, "button", "Save", "", false))
	label := ok(AhdGUIAddLeaf(root, "label", "Ready", "", false))
	var seen []string
	var running atomic.Int32
	must(t, AhdGUIOnClick(button, func() { seen = append(seen, "replaced") }))
	must(t, AhdGUIOnClick(button, func() {
		if running.Add(1) != 1 {
			t.Error("callbacks overlapped")
		}
		defer running.Add(-1)
		text := guiString(t)(AhdGUIText(input))
		checked := guiBool(t)(AhdGUIChecked(check))
		must(t, AhdGUISetText(label, "Saved "+text))
		seen = append(seen, "click:"+text+":"+map[bool]string{true: "paid", false: "open"}[checked]+":"+guiString(t)(AhdGUIText(label)))
	}))
	must(t, AhdGUIOnKey(window, func(key string) { seen = append(seen, "key:"+key) }))
	if problem := AhdGUIWait(window); problem != "" {
		t.Fatal(problem)
	}
	want := []string{"click:Ali:open:Saved Ali", "key:ArrowUp", "click:Ali:paid:Saved Ali", "key:A"}
	if strings.Join(seen, "|") != strings.Join(want, "|") {
		t.Fatalf("callbacks: %v", seen)
	}
	if AhdGUIIsOpen(window) {
		t.Fatal("the Window is open after wait")
	}
	if guiString(t)(AhdGUIText(input)) != "Ali" || !guiBool(t)(AhdGUIChecked(check)) {
		t.Fatal("final values after wait")
	}
}

// TestGUICallbackErrorClosesTheWindowAndPropagates checks a callback's own
// error is neither swallowed nor turned into a GUIError, and that no later
// callback runs.
func TestGUICallbackErrorClosesTheWindowAndPropagates(t *testing.T) {
	ok := guiMust2(t)
	window := openTestWindow(t, `[{"event":"click","widget":2},{"event":"click","widget":2}]`)
	button := ok(AhdGUIAddLeaf(ok(AhdGUIRoot(window, "column", 0, 0)), "button", "Go", "", false))
	calls := 0
	must(t, AhdGUIOnClick(button, func() {
		calls++
		panic("user error")
	}))
	record, _ := ahdGUIWindowOf(window)
	func() {
		defer func() {
			if recovered := recover(); recovered != "user error" {
				t.Fatalf("recovered %v", recovered)
			}
		}()
		AhdGUIWait(window)
		t.Fatal("wait returned normally")
	}()
	if calls != 1 {
		t.Fatalf("callback ran %d times", calls)
	}
	if AhdGUIIsOpen(window) || record.link.process.ProcessState == nil {
		t.Fatal("the Window was not closed and reaped after the callback error")
	}
}

func TestGUICallbackCanCloseTheWindow(t *testing.T) {
	ok := guiMust2(t)
	window := openTestWindow(t, `[{"event":"click","widget":2},{"event":"click","widget":2}]`)
	button := ok(AhdGUIAddLeaf(ok(AhdGUIRoot(window, "column", 0, 0)), "button", "Quit", "", false))
	calls := 0
	must(t, AhdGUIOnClick(button, func() {
		calls++
		AhdGUIClose(window)
	}))
	if problem := AhdGUIWait(window); problem != "" || calls != 1 {
		t.Fatalf("wait after a closing callback: %q, %d calls", problem, calls)
	}
}

func TestGUIHelperFailures(t *testing.T) {
	useGUIHelper(t, "")
	t.Setenv("AHDCODE_GUI_RUNTIME", filepath.Join(t.TempDir(), "missing"))
	previous := AhdGUIRuntimeHint
	AhdGUIRuntimeHint = ""
	defer func() { AhdGUIRuntimeHint = previous }()
	if _, problem := AhdGUIWindowOpen("t", 10, 10); !strings.Contains(problem, "ahdgui) was not found") {
		t.Fatalf("missing helper: %q", problem)
	}
	if runtime.GOOS == "windows" {
		return
	}
	crashing := filepath.Join(t.TempDir(), "crashing-helper")
	if err := os.WriteFile(crashing, []byte("#!/bin/sh\nread line\necho '{\"id\":1,\"ok\":true}'\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_GUI_RUNTIME", crashing)
	window := guiMust2(t)(AhdGUIWindowOpen("t", 10, 10))
	record, _ := ahdGUIWindowOf(window)
	_, problem := AhdGUIRoot(window, "column", 0, 0)
	guiMustNot(t, problem, "stopped unexpectedly")
	if AhdGUIIsOpen(window) || AhdGUIWait(window) != "" {
		t.Fatal("a crashed Window reports open or waits")
	}
	AhdGUIClose(window)
	if record.link.process.ProcessState == nil {
		t.Fatal("the crashed helper was not reaped")
	}
	garbage := filepath.Join(t.TempDir(), "garbage-helper")
	if err := os.WriteFile(garbage, []byte("#!/bin/sh\nread line\necho 'not json'\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_GUI_RUNTIME", garbage)
	if _, problem := AhdGUIWindowOpen("t", 10, 10); !strings.Contains(problem, "invalid response") {
		t.Fatalf("malformed protocol: %q", problem)
	}
}

// ---- Canvas events ----

func TestCanvasEventsUseCartesianCoordinatesAndReplaceHandlers(t *testing.T) {
	useGraphicsHelper(t)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS", `[{"event":"click","x":0,"y":0},{"event":"click","x":150,"y":80},`+
		`{"event":"key","key":"ArrowUp"},{"event":"key","key":"ArrowLeft"}]`)
	canvas, problem := AhdGraphicsOpen(400, 300, "t", "white")
	must(t, problem)
	defer AhdGraphicsClose(canvas)
	turtle, problem := AhdGraphicsTurtle(canvas)
	must(t, problem)
	var seen []string
	must(t, AhdGraphicsOnClick(canvas, func(x, y float64) { seen = append(seen, "old") }))
	must(t, AhdGraphicsOnClick(canvas, func(x, y float64) {
		must(t, AhdGraphicsTurtleMoveTo(turtle, x, y))
		seen = append(seen, "click:"+ahdGraphicsNumber(x)+","+ahdGraphicsNumber(y))
	}))
	must(t, AhdGraphicsOnKey(canvas, func(key string) {
		heading := map[string]float64{"ArrowUp": 90, "ArrowLeft": 180}[key]
		must(t, AhdGraphicsTurtleSetHeading(turtle, heading))
		must(t, AhdGraphicsTurtleForward(turtle, 20))
		must(t, AhdGraphicsLine(canvas, 0, 0, 1, 1, "red", 1))
		x, y, _ := turtleState(t, turtle)
		seen = append(seen, "key:"+key+":"+ahdGraphicsNumber(x)+","+ahdGraphicsNumber(y))
	}))
	must(t, AhdGraphicsWait(canvas))
	want := "click:0.0,0.0|click:150.0,80.0|key:ArrowUp:150.0,100.0|key:ArrowLeft:130.0,100.0"
	if strings.Join(seen, "|") != want {
		t.Fatalf("events: %v", seen)
	}
	if AhdGraphicsIsOpen(canvas) {
		t.Fatal("the Canvas is open after wait")
	}
}

func TestCanvasWaitWithoutHandlersIsUnchanged(t *testing.T) {
	useGraphicsHelper(t)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS", `[{"event":"key","key":"ArrowUp"}]`)
	canvas, problem := AhdGraphicsOpen(100, 100, "t", "white")
	must(t, problem)
	defer AhdGraphicsClose(canvas)
	must(t, AhdGraphicsLine(canvas, 0, 0, 10, 10, "black", 1))
	must(t, AhdGraphicsSave(canvas, filepath.Join(t.TempDir(), "before.png")))
	must(t, AhdGraphicsWait(canvas))
	must(t, AhdGraphicsWait(canvas))
}

func TestCanvasCallbackErrorClosesTheCanvasAndPropagates(t *testing.T) {
	useGraphicsHelper(t)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS", `[{"event":"key","key":"A"},{"event":"key","key":"B"}]`)
	canvas, problem := AhdGraphicsOpen(100, 100, "t", "white")
	must(t, problem)
	record := graphicsCanvasRecord(canvas)
	calls := 0
	must(t, AhdGraphicsOnKey(canvas, func(key string) {
		calls++
		panic("boom " + key)
	}))
	func() {
		defer func() {
			if recovered := recover(); recovered != "boom A" {
				t.Fatalf("recovered %v", recovered)
			}
		}()
		AhdGraphicsWait(canvas)
	}()
	if calls != 1 || AhdGraphicsIsOpen(canvas) || record.link.process.ProcessState == nil {
		t.Fatalf("after a callback error: %d calls, helper reaped %v", calls, record.link.process.ProcessState != nil)
	}
}
