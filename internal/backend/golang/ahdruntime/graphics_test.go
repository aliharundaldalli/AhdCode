package ahdruntime

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	graphicsHelperOnce sync.Once
	graphicsHelperPath string
	graphicsHelperErr  error
)

// useGraphicsHelper builds the real ahdgraphics helper once and points the
// runtime at it in headless mode, so these tests need no display.
func useGraphicsHelper(t *testing.T) {
	t.Helper()
	graphicsHelperOnce.Do(func() {
		directory, err := os.MkdirTemp("", "ahdgraphics-test-")
		if err != nil {
			graphicsHelperErr = err
			return
		}
		name := "ahdgraphics"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		graphicsHelperPath = filepath.Join(directory, name)
		root, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "cmd", "ahdgraphics"))
		command := exec.Command("go", "build", "-o", graphicsHelperPath, ".")
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			graphicsHelperErr = err
			graphicsHelperPath = string(output)
		}
	})
	if graphicsHelperErr != nil {
		t.Fatalf("building ahdgraphics: %v\n%s", graphicsHelperErr, graphicsHelperPath)
	}
	t.Setenv("AHDCODE_GRAPHICS_RUNTIME", graphicsHelperPath)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS", "1")
}

func openTestCanvas(t *testing.T) int64 {
	t.Helper()
	useGraphicsHelper(t)
	handle, problem := AhdGraphicsOpen(200, 100, "test", "white")
	if problem != "" {
		t.Fatalf("open: %s", problem)
	}
	t.Cleanup(func() { AhdGraphicsClose(handle) })
	return handle
}

func must(t *testing.T, problem string) {
	t.Helper()
	if problem != "" {
		t.Fatalf("unexpected GraphicsError: %s", problem)
	}
}

// savedSVG saves the Canvas as SVG and returns the document.
func savedSVG(t *testing.T, canvas int64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "canvas.svg")
	must(t, AhdGraphicsSave(canvas, path))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// turtleState reads a live Turtle's state, failing the test if it was released.
func turtleState(t *testing.T, handle int64) (float64, float64, float64) {
	t.Helper()
	x, y, heading, problem := AhdGraphicsTurtleState(handle)
	must(t, problem)
	return x, y, heading
}

func TestGraphicsColorParser(t *testing.T) {
	named := map[string]ahdGraphicsColor{
		"black": {0, 0, 0, 255}, "white": {255, 255, 255, 255}, "red": {255, 0, 0, 255},
		"green": {0, 128, 0, 255}, "blue": {0, 0, 255, 255}, "yellow": {255, 255, 0, 255},
		"cyan": {0, 255, 255, 255}, "magenta": {255, 0, 255, 255}, "gray": {128, 128, 128, 255},
	}
	for name, want := range named {
		if got, ok := AhdGraphicsParseColor(name); !ok || got != want {
			t.Errorf("%s -> %v %v", name, got, ok)
		}
	}
	for text, want := range map[string]ahdGraphicsColor{
		"#1a2B3c": {0x1a, 0x2b, 0x3c, 255}, "#FFFFFF": {255, 255, 255, 255}, "#11223380": {0x11, 0x22, 0x33, 0x80},
	} {
		if got, ok := AhdGraphicsParseColor(text); !ok || got != want {
			t.Errorf("%s -> %v %v", text, got, ok)
		}
	}
	for _, text := range []string{"", "Red", "BLACK", "purple", "grey", "#fff", "#12345", "#1234567", "#GGGGGG", "123456", "#1122334455", " red"} {
		if _, ok := AhdGraphicsParseColor(text); ok {
			t.Errorf("%q accepted", text)
		}
	}
}

func TestGraphicsHeadingNormalization(t *testing.T) {
	for input, want := range map[float64]float64{0: 0, 360: 0, 450: 90, -90: 270, -360: 0, 720.5: 0.5, 1e-300: 1e-300, math.Copysign(0, -1) - 1: 359} {
		if got := ahdGraphicsNormalizeHeading(input); got != want || math.Signbit(got) {
			t.Errorf("normalize(%v) = %v, want %v", input, got, want)
		}
	}
	if got := ahdGraphicsNormalizeHeading(math.Copysign(0, -1)); got != 0 || math.Signbit(got) {
		t.Errorf("negative zero normalized to %v", got)
	}
	if got := ahdGraphicsNormalizeHeading(-1e-15); got < 0 || got >= 360 {
		t.Errorf("tiny negative normalized to %v", got)
	}
	for heading, want := range map[float64][2]float64{0: {1, 0}, 90: {0, 1}, 180: {-1, 0}, 270: {0, -1}} {
		dx, dy := ahdGraphicsDirection(heading)
		if dx != want[0] || dy != want[1] {
			t.Errorf("direction(%v) = %v, %v", heading, dx, dy)
		}
	}
}

func TestGraphicsCanvasValidation(t *testing.T) {
	useGraphicsHelper(t)
	for _, size := range [][2]int64{{0, 100}, {100, 0}, {-1, 100}, {100, -5}, {AhdGraphicsMaxDimension + 1, 10}, {10, AhdGraphicsMaxDimension + 1}} {
		if _, problem := AhdGraphicsOpen(size[0], size[1], "t", "white"); problem == "" || !strings.Contains(problem, "Canvas size") {
			t.Errorf("size %v: %q", size, problem)
		}
	}
	if _, problem := AhdGraphicsOpen(10, 10, strings.Repeat("x", 257), "white"); problem == "" {
		t.Errorf("long title accepted")
	}
	if _, problem := AhdGraphicsOpen(10, 10, "t", "White"); !strings.Contains(problem, `background color "White"`) {
		t.Errorf("bad background: %q", problem)
	}
	handle, problem := AhdGraphicsOpen(AhdGraphicsMaxDimension, 1, "edge", "#00000000")
	if problem != "" {
		t.Fatalf("maximum width refused: %s", problem)
	}
	AhdGraphicsClose(handle)

	canvas := openTestCanvas(t)
	for _, problem := range []string{
		AhdGraphicsLine(canvas, 0, 0, 1, 1, "black", 0),
		AhdGraphicsLine(canvas, 0, 0, 1, 1, "black", -1),
		AhdGraphicsLine(canvas, 0, 0, 1, 1, "purple", 1),
		AhdGraphicsCircle(canvas, 0, 0, -1, "black", nil, 1),
		AhdGraphicsCircle(canvas, 0, 0, 1, "black", nil, 0),
		AhdGraphicsCircle(canvas, 0, 0, 1, "nope", nil, 1),
		AhdGraphicsRectangle(canvas, 0, 0, -1, 5, "black", nil, 1),
		AhdGraphicsRectangle(canvas, 0, 0, 5, -1, "black", nil, 1),
		AhdGraphicsRectangle(canvas, 0, 0, 5, 5, "black", nil, 0),
		AhdGraphicsClear(canvas, "transparent"),
		AhdGraphicsSave(canvas, "out.jpg"),
		AhdGraphicsSave(canvas, "out.pdf"),
		AhdGraphicsSave(canvas, ""),
		AhdGraphicsSave(canvas, strings.Repeat("a", ahdGraphicsMaxPathBytes)+".png"),
	} {
		if problem == "" {
			t.Errorf("an invalid drawing was accepted")
		}
	}
	bad := "#12"
	if problem := AhdGraphicsRectangle(canvas, 0, 0, 5, 5, "black", &bad, 1); !strings.Contains(problem, "fill color") {
		t.Errorf("bad fill: %q", problem)
	}
	// Valid edge cases draw without error.
	must(t, AhdGraphicsLine(canvas, 3, 3, 3, 3, "black", 1))
	must(t, AhdGraphicsCircle(canvas, 0, 0, 0, "black", nil, 1))
	must(t, AhdGraphicsRectangle(canvas, 0, 0, 0, 0, "black", nil, 1))
	must(t, AhdGraphicsLine(canvas, -1e9, 0, 1e9, 0, "#12345678", 0.25))
	for name := range ahdGraphicsNamedColors {
		fill := name
		must(t, AhdGraphicsCircle(canvas, 0, 0, 5, name, &fill, 1))
	}
}

func TestGraphicsCanvasPrimitivesSerialize(t *testing.T) {
	canvas := openTestCanvas(t)
	fill := "#00ff0080"
	must(t, AhdGraphicsLine(canvas, -100, 0, 100, 0, "red", 2))
	must(t, AhdGraphicsCircle(canvas, 0, 0, 20, "blue", &fill, 3))
	must(t, AhdGraphicsRectangle(canvas, -90, -40, 30, 20, "black", nil, 1))
	text := savedSVG(t, canvas)
	for _, want := range []string{
		`<line x1="0" y1="50" x2="200" y2="50" stroke="#ff0000" stroke-width="2"`,
		`<circle cx="100" cy="50" r="20" fill="#00ff00" fill-opacity="0.502" stroke="#0000ff" stroke-width="3"/>`,
		`<rect x="10" y="70" width="30" height="20" fill="none" stroke="#000000" stroke-width="1"/>`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("SVG lacks %s\n%s", want, text)
		}
	}
	must(t, AhdGraphicsClear(canvas, "#101010"))
	text = savedSVG(t, canvas)
	if strings.Contains(text, "<line") || strings.Contains(text, "<circle") || !strings.Contains(text, `fill="#101010"`) {
		t.Errorf("clear did not reset drawing and background:\n%s", text)
	}
}

func TestGraphicsTurtleInitialStateAndSquare(t *testing.T) {
	canvas := openTestCanvas(t)
	turtle, problem := AhdGraphicsTurtle(canvas)
	must(t, problem)
	if x, y, heading := turtleState(t, turtle); x != 0 || y != 0 || heading != 0 {
		t.Fatalf("initial state %v %v %v", x, y, heading)
	}
	for side := 0; side < 4; side++ {
		must(t, AhdGraphicsTurtleForward(turtle, 100))
		must(t, AhdGraphicsTurtleLeft(turtle, 90))
	}
	if x, y, heading := turtleState(t, turtle); x != 0 || y != 0 || heading != 0 {
		t.Fatalf("square did not close exactly: %v %v %v", x, y, heading)
	}
	text := savedSVG(t, canvas)
	if strings.Count(text, "<line") != 4 || !strings.Contains(text, `stroke="#000000" stroke-width="1"`) {
		t.Fatalf("square is not four black width-1 lines:\n%s", text)
	}
	// The first side runs right from the center, the second one up.
	if !strings.Contains(text, `<line x1="100" y1="50" x2="200" y2="50"`) || !strings.Contains(text, `<line x1="200" y1="50" x2="200" y2="-50"`) {
		t.Fatalf("square sides are not Cartesian:\n%s", text)
	}
}

func TestGraphicsTurtleGeometry(t *testing.T) {
	canvas := openTestCanvas(t)
	turtle, _ := AhdGraphicsTurtle(canvas)
	for side := 0; side < 3; side++ {
		must(t, AhdGraphicsTurtleForward(turtle, 100))
		must(t, AhdGraphicsTurtleLeft(turtle, 120))
	}
	x, y, heading := turtleState(t, turtle)
	if !near(x, 0) || !near(y, 0) || !near(heading, 0) && !near(heading, 360) {
		t.Fatalf("triangle did not close: %v %v %v", x, y, heading)
	}
	must(t, AhdGraphicsTurtleSetHeading(turtle, 30))
	must(t, AhdGraphicsTurtleMoveTo(turtle, 0, 0))
	must(t, AhdGraphicsTurtleForward(turtle, 10))
	x, y, _ = turtleState(t, turtle)
	if !near(x, 10*math.Sqrt(3)/2) || !near(y, 5) {
		t.Fatalf("forward at 30 degrees: %v %v", x, y)
	}
	must(t, AhdGraphicsTurtleMoveTo(turtle, 0, 0))
	must(t, AhdGraphicsTurtleSetHeading(turtle, 0))
	must(t, AhdGraphicsTurtleForward(turtle, -20))
	must(t, AhdGraphicsTurtleBackward(turtle, 5))
	if x, y, _ := turtleState(t, turtle); x != -25 || y != 0 {
		t.Fatalf("negative movement: %v %v", x, y)
	}
	must(t, AhdGraphicsTurtleBackward(turtle, -25))
	if x, _, _ := turtleState(t, turtle); x != 0 {
		t.Fatalf("backward(-25) should equal forward(25): %v", x)
	}
	must(t, AhdGraphicsTurtleLeft(turtle, -90))
	if _, _, heading := turtleState(t, turtle); heading != 270 {
		t.Fatalf("left(-90) = %v", heading)
	}
	must(t, AhdGraphicsTurtleSetHeading(turtle, 0))
	must(t, AhdGraphicsTurtleLeft(turtle, 450))
	if _, _, heading := turtleState(t, turtle); heading != 90 {
		t.Fatalf("left(450) = %v", heading)
	}
	must(t, AhdGraphicsTurtleRight(turtle, 180))
	if _, _, heading := turtleState(t, turtle); heading != 270 {
		t.Fatalf("right(180) from 90 = %v", heading)
	}
	must(t, AhdGraphicsTurtleSetHeading(turtle, -45))
	if _, _, heading := turtleState(t, turtle); heading != 315 {
		t.Fatalf("setHeading(-45) = %v", heading)
	}
	if problem := AhdGraphicsTurtleForward(turtle, math.MaxFloat64); problem != "" {
		// cos(315) * MaxFloat64 is finite; moving again overflows.
		t.Fatalf("large finite move refused: %s", problem)
	}
	if problem := AhdGraphicsTurtleForward(turtle, math.MaxFloat64); !strings.Contains(problem, "finite") {
		t.Fatalf("overflowing move: %q", problem)
	}
}

func TestGraphicsTurtlePenColorWidthAndHome(t *testing.T) {
	canvas := openTestCanvas(t)
	turtle, _ := AhdGraphicsTurtle(canvas)
	must(t, AhdGraphicsTurtlePen(turtle, false))
	must(t, AhdGraphicsTurtleForward(turtle, 50))
	must(t, AhdGraphicsTurtleMoveTo(turtle, -30, 20))
	if x, y, _ := turtleState(t, turtle); x != -30 || y != 20 {
		t.Fatalf("pen-up movement did not move: %v %v", x, y)
	}
	if strings.Contains(savedSVG(t, canvas), "<line") {
		t.Fatalf("pen-up movement drew a line")
	}
	must(t, AhdGraphicsTurtlePen(turtle, true))
	must(t, AhdGraphicsTurtleSetColor(turtle, "#ff000080"))
	must(t, AhdGraphicsTurtleSetWidth(turtle, 3.5))
	must(t, AhdGraphicsTurtleSetHeading(turtle, 90))
	must(t, AhdGraphicsTurtleForward(turtle, 10))
	text := savedSVG(t, canvas)
	if strings.Count(text, "<line") != 1 || !strings.Contains(text, `stroke="#ff0000" stroke-opacity="0.502" stroke-width="3.5"`) {
		t.Fatalf("pen-down line with color and width:\n%s", text)
	}
	if problem := AhdGraphicsTurtleSetColor(turtle, "Orange"); problem == "" {
		t.Fatalf("invalid Turtle color accepted")
	}
	for _, width := range []float64{0, -1} {
		if problem := AhdGraphicsTurtleSetWidth(turtle, width); problem == "" {
			t.Fatalf("Turtle width %v accepted", width)
		}
	}
	must(t, AhdGraphicsTurtleHome(turtle))
	x, y, heading := turtleState(t, turtle)
	if x != 0 || y != 0 || heading != 0 {
		t.Fatalf("home: %v %v %v", x, y, heading)
	}
	if strings.Count(savedSVG(t, canvas), "<line") != 2 {
		t.Fatalf("home with the pen down should draw its movement")
	}
	must(t, AhdGraphicsTurtlePen(turtle, false))
	must(t, AhdGraphicsTurtleMoveTo(turtle, 10, 10))
	must(t, AhdGraphicsTurtleHome(turtle))
	if strings.Count(savedSVG(t, canvas), "<line") != 2 {
		t.Fatalf("home with the pen up drew a line")
	}
}

func TestGraphicsClearKeepsTurtleState(t *testing.T) {
	canvas := openTestCanvas(t)
	turtle, _ := AhdGraphicsTurtle(canvas)
	must(t, AhdGraphicsTurtleSetColor(turtle, "red"))
	must(t, AhdGraphicsTurtleSetWidth(turtle, 4))
	must(t, AhdGraphicsTurtleLeft(turtle, 30))
	must(t, AhdGraphicsTurtleForward(turtle, 40))
	must(t, AhdGraphicsTurtlePen(turtle, false))
	before := [3]float64{}
	before[0], before[1], before[2] = turtleState(t, turtle)
	must(t, AhdGraphicsClear(canvas, "black"))
	after := [3]float64{}
	after[0], after[1], after[2] = turtleState(t, turtle)
	if before != after {
		t.Fatalf("clear moved the Turtle: %v -> %v", before, after)
	}
	must(t, AhdGraphicsTurtlePen(turtle, true))
	must(t, AhdGraphicsTurtleForward(turtle, 10))
	if !strings.Contains(savedSVG(t, canvas), `stroke="#ff0000" stroke-width="4"`) {
		t.Fatalf("clear changed the Turtle color or width")
	}
}

func TestGraphicsTwoTurtlesAreIndependent(t *testing.T) {
	canvas := openTestCanvas(t)
	first, _ := AhdGraphicsTurtle(canvas)
	second, _ := AhdGraphicsTurtle(canvas)
	must(t, AhdGraphicsTurtleForward(first, 30))
	must(t, AhdGraphicsTurtleLeft(second, 90))
	must(t, AhdGraphicsTurtleForward(second, 20))
	x1, y1, h1 := turtleState(t, first)
	x2, y2, h2 := turtleState(t, second)
	if x1 != 30 || y1 != 0 || h1 != 0 || x2 != 0 || y2 != 20 || h2 != 90 {
		t.Fatalf("turtles share state: (%v %v %v) (%v %v %v)", x1, y1, h1, x2, y2, h2)
	}
}

func TestGraphicsLifecycle(t *testing.T) {
	canvas := openTestCanvas(t)
	turtle, _ := AhdGraphicsTurtle(canvas)
	if !AhdGraphicsIsOpen(canvas) {
		t.Fatalf("new Canvas reports closed")
	}
	// Headless: there is no window, so wait returns at once and closes it.
	must(t, AhdGraphicsWait(canvas))
	if AhdGraphicsIsOpen(canvas) {
		t.Fatalf("Canvas still open after wait")
	}
	if problem := AhdGraphicsLine(canvas, 0, 0, 1, 1, "black", 1); problem != ahdGraphicsClosedMessage {
		t.Fatalf("drawing on a closed Canvas: %q", problem)
	}
	if problem := AhdGraphicsSave(canvas, filepath.Join(t.TempDir(), "x.png")); problem != ahdGraphicsClosedMessage {
		t.Fatalf("saving a closed Canvas: %q", problem)
	}
	if problem := AhdGraphicsTurtleForward(turtle, 10); problem != ahdGraphicsClosedMessage {
		t.Fatalf("Turtle drawing on a closed Canvas: %q", problem)
	}
	if x, _, _ := turtleState(t, turtle); x != 0 {
		t.Fatalf("a failed Turtle draw moved the Turtle")
	}
	must(t, AhdGraphicsTurtlePen(turtle, false))
	must(t, AhdGraphicsTurtleForward(turtle, 10))
	must(t, AhdGraphicsWait(canvas))

	record := graphicsCanvasRecord(canvas)
	must(t, AhdGraphicsClose(canvas))
	must(t, AhdGraphicsClose(canvas))
	if record.link.process.ProcessState == nil {
		t.Fatalf("closed Canvas left its helper unreaped")
	}
}

func TestGraphicsMultipleCanvasesAreIndependent(t *testing.T) {
	first := openTestCanvas(t)
	second := openTestCanvas(t)
	must(t, AhdGraphicsClose(first))
	if !AhdGraphicsIsOpen(second) {
		t.Fatalf("closing one Canvas closed another")
	}
	must(t, AhdGraphicsLine(second, 0, 0, 5, 5, "black", 1))
}

func graphicsCanvasRecord(handle int64) *ahdGraphicsCanvas {
	ahdGraphics.Lock()
	defer ahdGraphics.Unlock()
	return ahdGraphics.canvases[handle]
}

func TestGraphicsHelperCrashBecomesGraphicsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in helper is a POSIX shell script")
	}
	script := filepath.Join(t.TempDir(), "crashing-helper")
	// Acknowledges open, then exits: the next request finds it gone.
	if err := os.WriteFile(script, []byte("#!/bin/sh\nread line\necho '{\"id\":1,\"ok\":true}'\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_GRAPHICS_RUNTIME", script)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS", "1")
	canvas, problem := AhdGraphicsOpen(10, 10, "t", "white")
	must(t, problem)
	record := graphicsCanvasRecord(canvas)
	if problem := AhdGraphicsLine(canvas, 0, 0, 1, 1, "black", 1); problem != ahdGraphicsStoppedMessage {
		t.Fatalf("crash: %q", problem)
	}
	if AhdGraphicsIsOpen(canvas) {
		t.Fatalf("a crashed Canvas reports open")
	}
	must(t, AhdGraphicsClose(canvas))
	if record.link.process.ProcessState == nil || record.link.process.ProcessState.ExitCode() != 3 {
		t.Fatalf("crashed helper was not reaped: %v", record.link.process.ProcessState)
	}

	silent := filepath.Join(t.TempDir(), "silent-helper")
	if err := os.WriteFile(silent, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_GRAPHICS_RUNTIME", silent)
	if _, problem := AhdGraphicsOpen(10, 10, "t", "white"); !strings.HasPrefix(problem, "could not open the Canvas") {
		t.Fatalf("helper that never answers open: %q", problem)
	}
}

func TestGraphicsMissingHelper(t *testing.T) {
	t.Setenv("AHDCODE_GRAPHICS_RUNTIME", filepath.Join(t.TempDir(), "missing"))
	previous := AhdGraphicsRuntimeHint
	AhdGraphicsRuntimeHint = ""
	defer func() { AhdGraphicsRuntimeHint = previous }()
	if _, problem := AhdGraphicsOpen(10, 10, "t", "white"); !strings.Contains(problem, "was not found") {
		t.Fatalf("missing helper: %q", problem)
	}
}

func TestGraphicsConcurrentCanvases(t *testing.T) {
	canvases := []int64{openTestCanvas(t), openTestCanvas(t)}
	var group sync.WaitGroup
	for _, canvas := range canvases {
		turtle, _ := AhdGraphicsTurtle(canvas)
		group.Add(1)
		go func(canvas, turtle int64) {
			defer group.Done()
			for step := 0; step < 50; step++ {
				_ = AhdGraphicsTurtleForward(turtle, 3)
				_ = AhdGraphicsTurtleLeft(turtle, 7)
				_ = AhdGraphicsIsOpen(canvas)
			}
		}(canvas, turtle)
	}
	group.Wait()
	AhdGraphicsCloseAll()
	for _, canvas := range canvases {
		if AhdGraphicsIsOpen(canvas) {
			t.Fatalf("CloseAll left a Canvas open")
		}
	}
}

func TestGraphicsCloseReleasesCanvasAndItsTurtles(t *testing.T) {
	first := openTestCanvas(t)
	second := openTestCanvas(t)
	pen, _ := AhdGraphicsTurtle(first)
	other, _ := AhdGraphicsTurtle(first)
	kept, _ := AhdGraphicsTurtle(second)
	must(t, AhdGraphicsTurtleForward(pen, 10))
	must(t, AhdGraphicsClose(first))

	ahdGraphics.Lock()
	_, canvasKept := ahdGraphics.canvases[first]
	_, penKept := ahdGraphics.turtles[pen]
	_, otherKept := ahdGraphics.turtles[other]
	_, secondKept := ahdGraphics.turtles[kept]
	ahdGraphics.Unlock()
	if canvasKept || penKept || otherKept || !secondKept {
		t.Fatalf("close released the wrong state: canvas %v, turtles %v %v, other Canvas turtle %v", canvasKept, penKept, otherKept, secondKept)
	}

	// Stale values fail cleanly, never with a panic.
	for _, problem := range []string{
		AhdGraphicsTurtleForward(pen, 5),
		AhdGraphicsTurtlePen(pen, false),
		AhdGraphicsTurtleLeft(other, 90),
		AhdGraphicsTurtleSetColor(other, "red"),
		AhdGraphicsTurtleHome(pen),
	} {
		if problem != ahdGraphicsReleasedMessage {
			t.Fatalf("stale Turtle: %q", problem)
		}
	}
	if _, _, _, problem := AhdGraphicsTurtleState(pen); problem != ahdGraphicsReleasedMessage {
		t.Fatalf("stale Turtle state: %q", problem)
	}
	if problem := AhdGraphicsLine(first, 0, 0, 1, 1, "black", 1); problem != ahdGraphicsClosedMessage {
		t.Fatalf("stale Canvas: %q", problem)
	}
	if _, problem := AhdGraphicsTurtle(first); problem != ahdGraphicsClosedMessage {
		t.Fatalf("turtle() on a closed Canvas: %q", problem)
	}
	if AhdGraphicsIsOpen(first) {
		t.Fatalf("closed Canvas reports open")
	}
	must(t, AhdGraphicsWait(first))
	must(t, AhdGraphicsClose(first))

	// The other Canvas and its Turtle are untouched.
	must(t, AhdGraphicsTurtleForward(kept, 20))
	if x, _, _ := turtleState(t, kept); x != 20 {
		t.Fatalf("other Canvas turtle: %v", x)
	}
}
