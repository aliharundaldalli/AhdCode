package ahdruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Graphics standard module (v1.6.0)
// ---------------------------------------------------------------------------
//
// Graphics is a Cartesian 2D Canvas and a Turtle pen for visual programming
// and mathematics. This file is the whole AhdCode side of it and uses only the
// standard library: it validates every argument, parses colors, keeps every
// Turtle's state, and reduces each drawing to a Canvas primitive. The window,
// rasterizer, and PNG/SVG writers live in the bundled ahdgraphics helper, one
// process per open Canvas, reached through a line-based JSON protocol over the
// helper's standard input and output (see cmd/ahdgraphics/protocol.go; the
// request shape is duplicated here field for field because this file is
// embedded into every compiled program and cannot import that module).
//
// Both `ahdcode run`/`ahdcode build` programs and the REPL evaluator call these
// same functions, so Turtle geometry and every validation rule are shared.
// Functions report a problem as a non-empty message; AhdGraphicsCheck turns it
// into a GraphicsError in a compiled program, and the evaluator raises its own
// GraphicsError from the same message.
//
// Since v1.8 a Canvas also reports clicks and key presses: Canvas.onClick and
// Canvas.onKey register one callback each, and Canvas.wait runs them, one at a
// time on the program's own goroutine, until the window is closed. The
// transport, event routing, and queue bound live in helperlink.go, shared with
// the GUI module.

// AhdClassGraphicsError is the runtime descriptor of GraphicsError.
var AhdClassGraphicsError = &AhdClass{Name: "GraphicsError", Parent: AhdClassError}

// AhdGraphicsRuntimeHint is the directory of the bundled ahdgraphics helper,
// filled in by the compiler for programs that use Graphics, exactly like
// AhdPlotRuntimeHint. AHDCODE_GRAPHICS_RUNTIME overrides it for packaging and
// tests.
var AhdGraphicsRuntimeHint string

const (
	ahdGraphicsProtocolVersion = 2
	// AhdGraphicsMaxDimension bounds a Canvas side. A 4096x4096 Canvas is a
	// 64 MiB image, the largest the helper will allocate for one export.
	AhdGraphicsMaxDimension  = 4096
	ahdGraphicsMaxTitleRunes = 256
	ahdGraphicsMaxPathBytes  = 4096
	// ahdGraphicsMaxResponseBytes bounds one helper response line.
	ahdGraphicsMaxResponseBytes = 16 << 10
	ahdGraphicsOpenTimeout      = 30 * time.Second
	ahdGraphicsRequestTimeout   = 60 * time.Second
	ahdGraphicsCloseTimeout     = 5 * time.Second
)

const (
	ahdGraphicsReleasedMessage = "the Turtle's Canvas was closed with close()"
	ahdGraphicsClosedMessage   = "the Canvas is closed"
	ahdGraphicsStoppedMessage  = "the Graphics window helper stopped unexpectedly"
)

// ahdGraphicsColor is a parsed, non-premultiplied RGBA color.
type ahdGraphicsColor [4]uint8

var ahdGraphicsNamedColors = map[string]ahdGraphicsColor{
	"black":   {0, 0, 0, 255},
	"white":   {255, 255, 255, 255},
	"red":     {255, 0, 0, 255},
	"green":   {0, 128, 0, 255},
	"blue":    {0, 0, 255, 255},
	"yellow":  {255, 255, 0, 255},
	"cyan":    {0, 255, 255, 255},
	"magenta": {255, 0, 255, 255},
	"gray":    {128, 128, 128, 255},
}

// AhdGraphicsParseColor is the one color parser: one of nine case-sensitive
// names, #RRGGBB, or #RRGGBBAA (hex digits in either case). Anything else is
// rejected; nothing is silently replaced by black.
func AhdGraphicsParseColor(text string) (ahdGraphicsColor, bool) {
	if named, ok := ahdGraphicsNamedColors[text]; ok {
		return named, true
	}
	if len(text) != 7 && len(text) != 9 || text[0] != '#' {
		return ahdGraphicsColor{}, false
	}
	var value ahdGraphicsColor
	value[3] = 255
	for index := 0; index < (len(text)-1)/2; index++ {
		high, okHigh := ahdGraphicsHexDigit(text[1+2*index])
		low, okLow := ahdGraphicsHexDigit(text[2+2*index])
		if !okHigh || !okLow {
			return ahdGraphicsColor{}, false
		}
		value[index] = high<<4 | low
	}
	return value, true
}

func ahdGraphicsHexDigit(character byte) (uint8, bool) {
	switch {
	case character >= '0' && character <= '9':
		return character - '0', true
	case character >= 'a' && character <= 'f':
		return character - 'a' + 10, true
	case character >= 'A' && character <= 'F':
		return character - 'A' + 10, true
	}
	return 0, false
}

func ahdGraphicsColorProblem(role, text string) string {
	return fmt.Sprintf("unsupported %s color %q; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA", role, text)
}

// ahdGraphicsRequest mirrors the helper's request field for field.
type ahdGraphicsRequest struct {
	Op string `json:"op"`

	Version    int               `json:"version,omitempty"`
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	Title      string            `json:"title,omitempty"`
	Headless   bool              `json:"headless,omitempty"`
	Background *ahdGraphicsColor `json:"background,omitempty"`

	X1 float64 `json:"x1,omitempty"`
	Y1 float64 `json:"y1,omitempty"`
	X2 float64 `json:"x2,omitempty"`
	Y2 float64 `json:"y2,omitempty"`

	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Radius float64 `json:"radius,omitempty"`
	W      float64 `json:"w,omitempty"`
	H      float64 `json:"h,omitempty"`

	Color     *ahdGraphicsColor `json:"color,omitempty"`
	Stroke    *ahdGraphicsColor `json:"stroke,omitempty"`
	Fill      *ahdGraphicsColor `json:"fill,omitempty"`
	LineWidth float64           `json:"lineWidth,omitempty"`

	Path string `json:"path,omitempty"`

	Kind   string             `json:"kind,omitempty"`
	Script []ahdGraphicsEvent `json:"script,omitempty"`
}

// ahdGraphicsEvent is one event line from the helper, and one scripted event
// of a headless test Canvas.
type ahdGraphicsEvent struct {
	Event string  `json:"event"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	Key   string  `json:"key,omitempty"`
}

type ahdGraphicsResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Closed bool   `json:"closed,omitempty"`
	Open   *bool  `json:"open,omitempty"`
}

// ahdGraphicsCanvas is one open Canvas: its helper link and its state.
// usable is false once the Canvas can no longer be drawn on; alive is false
// once its helper process is gone. mu serializes requests; it is never held
// while an event callback runs.
type ahdGraphicsCanvas struct {
	mu       sync.Mutex
	link     *ahdHelperLink
	usable   bool
	alive    bool
	headless bool
	// The event callbacks; nil until registered. A second registration
	// replaces the first.
	onClick func(x, y float64)
	onKey   func(key string)
	// listening records the event kinds the helper was asked to report.
	listening map[string]bool
}

// ahdGraphicsTurtle is one Turtle's complete state. Geometry happens here,
// never in the helper: a Turtle only ever asks its Canvas for a line.
type ahdGraphicsTurtle struct {
	mu      sync.Mutex
	canvas  int64
	x, y    float64
	heading float64
	pen     bool
	color   ahdGraphicsColor
	width   float64
}

var ahdGraphics = struct {
	sync.Mutex
	next     int64
	canvases map[int64]*ahdGraphicsCanvas
	turtles  map[int64]*ahdGraphicsTurtle
}{canvases: map[int64]*ahdGraphicsCanvas{}, turtles: map[int64]*ahdGraphicsTurtle{}}

func init() { AhdOnExit(AhdGraphicsCloseAll) }

// ahdGraphicsDiscoverRuntime finds the bundled helper the same way Plot finds
// ahdplot: the explicit override, the compiler's baked-in hint, then paths
// relative to the running executable. PATH is never searched.
func ahdGraphicsDiscoverRuntime() (string, error) {
	name := "ahdgraphics"
	if runtime.GOOS == "windows" {
		name = "ahdgraphics.exe"
	}
	if AhdPackagedApplication {
		if path, ok := ahdPackagedHelper(name); ok {
			return path, nil
		}
		return "", errors.New("the Graphics window helper (ahdgraphics) is missing from this application")
	}
	candidates := []string{os.Getenv("AHDCODE_GRAPHICS_RUNTIME")}
	if AhdGraphicsRuntimeHint != "" {
		candidates = append(candidates, filepath.Join(AhdGraphicsRuntimeHint, name))
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
	return "", errors.New("the Graphics window helper (ahdgraphics) was not found; reinstall AhdCode with its bundled helpers")
}

// AhdGraphicsOpen validates a Canvas, starts its helper, and waits until the
// window is ready. It returns the Canvas handle.
func AhdGraphicsOpen(width, height int64, title, background string) (int64, string) {
	if width < 1 || height < 1 || width > AhdGraphicsMaxDimension || height > AhdGraphicsMaxDimension {
		return 0, fmt.Sprintf("Canvas size %dx%d is not allowed; width and height must each be between 1 and %d", width, height, AhdGraphicsMaxDimension)
	}
	if !utf8.ValidString(title) || strings.ContainsRune(title, 0) || utf8.RuneCountInString(title) > ahdGraphicsMaxTitleRunes {
		return 0, fmt.Sprintf("the Canvas title must be text of at most %d characters", ahdGraphicsMaxTitleRunes)
	}
	color, ok := AhdGraphicsParseColor(background)
	if !ok {
		return 0, ahdGraphicsColorProblem("background", background)
	}
	headless := os.Getenv("AHDCODE_GRAPHICS_HEADLESS") == "1"
	var script []ahdGraphicsEvent
	if text := os.Getenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS"); headless && text != "" {
		// Automated tests replay clicks and key presses on a headless Canvas.
		if json.Unmarshal([]byte(text), &script) != nil {
			return 0, "AHDCODE_GRAPHICS_HEADLESS_EVENTS is not a JSON list of events"
		}
	}
	path, err := ahdGraphicsDiscoverRuntime()
	if err != nil {
		return 0, err.Error()
	}
	link, err := ahdStartHelper(path, ahdGraphicsMaxResponseBytes)
	if err != nil {
		return 0, "could not start the Graphics window helper"
	}
	canvas := &ahdGraphicsCanvas{link: link, usable: true, alive: true, headless: headless, listening: map[string]bool{}}
	_, problem := canvas.request(ahdGraphicsRequest{Op: "open", Version: ahdGraphicsProtocolVersion,
		Width: int(width), Height: int(height), Title: title, Background: &color,
		Headless: headless, Script: script}, ahdGraphicsOpenTimeout)
	if problem != "" {
		canvas.mu.Lock()
		canvas.shutdown()
		canvas.mu.Unlock()
		return 0, "could not open the Canvas: " + problem
	}
	ahdGraphics.Lock()
	ahdGraphics.next++
	handle := ahdGraphics.next
	ahdGraphics.canvases[handle] = canvas
	ahdGraphics.Unlock()
	return handle, ""
}

// request sends one request and waits for its one response. The caller holds
// c.mu, except during open, before the Canvas is shared.
func (c *ahdGraphicsCanvas) request(value ahdGraphicsRequest, timeout time.Duration) (ahdGraphicsResponse, string) {
	if !c.alive {
		return ahdGraphicsResponse{}, ahdGraphicsClosedMessage
	}
	line, err := c.link.call(value, timeout)
	switch {
	case errors.Is(err, errAhdHelperSilent):
		c.shutdown()
		return ahdGraphicsResponse{}, "the Graphics window helper stopped responding"
	case errors.Is(err, errAhdHelperInvalid):
		c.shutdown()
		return ahdGraphicsResponse{}, "the Graphics window helper sent an invalid response"
	case err != nil:
		c.shutdown()
		return ahdGraphicsResponse{}, ahdGraphicsStoppedMessage
	}
	var reply ahdGraphicsResponse
	if json.Unmarshal(line, &reply) != nil {
		c.shutdown()
		return ahdGraphicsResponse{}, "the Graphics window helper sent an invalid response"
	}
	if reply.Closed {
		c.usable = false
	}
	if !reply.OK {
		if reply.Error == "" {
			reply.Error = "the Graphics window helper reported an unknown problem"
		}
		return reply, reply.Error
	}
	return reply, ""
}

// shutdown ends the helper process: its input is closed, which a helper
// treats as the program ending, and it is killed if it does not exit.
func (c *ahdGraphicsCanvas) shutdown() {
	if !c.alive {
		return
	}
	c.usable, c.alive = false, false
	c.link.shutdown(ahdGraphicsCloseTimeout)
}

func ahdGraphicsCanvasOf(handle int64) (*ahdGraphicsCanvas, string) {
	ahdGraphics.Lock()
	canvas := ahdGraphics.canvases[handle]
	ahdGraphics.Unlock()
	if canvas == nil {
		return nil, ahdGraphicsClosedMessage
	}
	return canvas, ""
}

// ahdGraphicsDraw sends one drawing request to an open Canvas.
func ahdGraphicsDraw(handle int64, value ahdGraphicsRequest) string {
	canvas, problem := ahdGraphicsCanvasOf(handle)
	if problem != "" {
		return problem
	}
	canvas.mu.Lock()
	defer canvas.mu.Unlock()
	if !canvas.usable {
		return ahdGraphicsClosedMessage
	}
	_, problem = canvas.request(value, ahdGraphicsRequestTimeout)
	return problem
}

// AhdGraphicsClear clears every drawing and sets the background.
func AhdGraphicsClear(handle int64, color string) string {
	parsed, ok := AhdGraphicsParseColor(color)
	if !ok {
		return ahdGraphicsColorProblem("clear", color)
	}
	return ahdGraphicsDraw(handle, ahdGraphicsRequest{Op: "clear", Color: &parsed})
}

// AhdGraphicsLine draws a straight line between two Cartesian points.
func AhdGraphicsLine(handle int64, x1, y1, x2, y2 float64, color string, width float64) string {
	parsed, ok := AhdGraphicsParseColor(color)
	if !ok {
		return ahdGraphicsColorProblem("line", color)
	}
	if !(width > 0) {
		return fmt.Sprintf("line width must be greater than 0, not %s", ahdGraphicsNumber(width))
	}
	return ahdGraphicsDraw(handle, ahdGraphicsRequest{Op: "line", X1: x1, Y1: y1, X2: x2, Y2: y2, Color: &parsed, LineWidth: width})
}

// AhdGraphicsCircle draws a circle around (x, y). fill is nil for no fill.
func AhdGraphicsCircle(handle int64, x, y, radius float64, stroke string, fill *string, width float64) string {
	if radius < 0 {
		return fmt.Sprintf("circle radius must be 0 or greater, not %s", ahdGraphicsNumber(radius))
	}
	if !(width > 0) {
		return fmt.Sprintf("circle stroke width must be greater than 0, not %s", ahdGraphicsNumber(width))
	}
	request := ahdGraphicsRequest{Op: "circle", X: x, Y: y, Radius: radius, LineWidth: width}
	if problem := ahdGraphicsPaint(&request, stroke, fill); problem != "" {
		return problem
	}
	return ahdGraphicsDraw(handle, request)
}

// AhdGraphicsRectangle draws a rectangle whose lower-left corner is (x, y).
func AhdGraphicsRectangle(handle int64, x, y, width, height float64, stroke string, fill *string, lineWidth float64) string {
	if width < 0 || height < 0 {
		return fmt.Sprintf("rectangle width and height must be 0 or greater, not %s and %s", ahdGraphicsNumber(width), ahdGraphicsNumber(height))
	}
	if !(lineWidth > 0) {
		return fmt.Sprintf("rectangle line width must be greater than 0, not %s", ahdGraphicsNumber(lineWidth))
	}
	request := ahdGraphicsRequest{Op: "rectangle", X: x, Y: y, W: width, H: height, LineWidth: lineWidth}
	if problem := ahdGraphicsPaint(&request, stroke, fill); problem != "" {
		return problem
	}
	return ahdGraphicsDraw(handle, request)
}

func ahdGraphicsPaint(request *ahdGraphicsRequest, stroke string, fill *string) string {
	parsed, ok := AhdGraphicsParseColor(stroke)
	if !ok {
		return ahdGraphicsColorProblem("stroke", stroke)
	}
	request.Stroke = &parsed
	if fill != nil {
		filled, ok := AhdGraphicsParseColor(*fill)
		if !ok {
			return ahdGraphicsColorProblem("fill", *fill)
		}
		request.Fill = &filled
	}
	return ""
}

// AhdGraphicsSave writes the Canvas as PNG or SVG, chosen by the path's
// extension without regard to case. A relative path is relative to the
// program's working directory.
func AhdGraphicsSave(handle int64, path string) string {
	if path == "" || len(path) > ahdGraphicsMaxPathBytes || strings.ContainsRune(path, 0) {
		return "save needs a file path"
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".svg":
	default:
		return fmt.Sprintf("cannot save %q: Graphics saves only .png and .svg files", path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("cannot save %q: the path cannot be resolved", path)
	}
	problem := ahdGraphicsDraw(handle, ahdGraphicsRequest{Op: "save", Path: absolute})
	if problem != "" && problem != ahdGraphicsClosedMessage && !strings.HasPrefix(problem, "the Graphics window helper") {
		return fmt.Sprintf("cannot save %q: %s", path, problem)
	}
	return problem
}

// AhdGraphicsWait blocks until the Canvas window is closed by its user,
// running the registered onClick and onKey callbacks one at a time, in order,
// on the calling goroutine. On a Canvas that is already closed it returns at
// once. A callback's own error closes the Canvas and propagates unchanged.
// Events still queued when the window closes are discarded: their window is
// gone.
func AhdGraphicsWait(handle int64) string {
	canvas, problem := ahdGraphicsCanvasOf(handle)
	if problem != "" {
		return ""
	}
	canvas.mu.Lock()
	// A Canvas whose window is already gone, or that an earlier wait already
	// finished, has nothing left to wait for.
	if !canvas.alive || !canvas.usable {
		canvas.mu.Unlock()
		return ""
	}
	// Output written before wait must be visible while the window is open.
	AhdFlush()
	_, problem = canvas.request(ahdGraphicsRequest{Op: "wait"}, ahdGraphicsRequestTimeout)
	canvas.mu.Unlock()
	if problem != "" {
		if problem == ahdGraphicsClosedMessage {
			return ""
		}
		return problem
	}
	for {
		line, kind := canvas.link.events.next(canvas.link.exited)
		canvas.mu.Lock()
		alive, onClick, onKey := canvas.alive, canvas.onClick, canvas.onKey
		stale := !canvas.headless && canvas.link.events.closedPending()
		switch kind {
		case ahdEventClosed:
			canvas.usable = false
			canvas.mu.Unlock()
			return ""
		case ahdEventOverflow:
			canvas.shutdown()
			canvas.mu.Unlock()
			return "the Canvas received more window events than the program handled"
		case ahdEventExited:
			canvas.mu.Unlock()
			if !alive {
				return ""
			}
			canvas.mu.Lock()
			canvas.shutdown()
			canvas.mu.Unlock()
			return ahdGraphicsStoppedMessage
		}
		canvas.mu.Unlock()
		if !alive {
			return ""
		}
		if stale {
			continue
		}
		var event ahdGraphicsEvent
		if json.Unmarshal(line, &event) != nil {
			canvas.mu.Lock()
			canvas.shutdown()
			canvas.mu.Unlock()
			return "the Graphics window helper sent an invalid event"
		}
		switch {
		case event.Event == "click" && onClick != nil:
			ahdGraphicsDispatch(handle, func() { onClick(event.X, event.Y) })
		case event.Event == "key" && onKey != nil:
			ahdGraphicsDispatch(handle, func() { onKey(event.Key) })
		}
	}
}

// ahdGraphicsDispatch runs one callback. If it raises, the Canvas is closed
// and its helper released, and the error propagates unchanged.
func ahdGraphicsDispatch(handle int64, callback func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			AhdGraphicsClose(handle)
			panic(recovered)
		}
	}()
	callback()
	// What a callback writes is visible at once, not when the window closes.
	AhdFlush()
}

// ahdGraphicsListen stores one callback and asks the helper, once per kind,
// to report that kind of event.
func ahdGraphicsListen(handle int64, kind string, store func(c *ahdGraphicsCanvas)) string {
	canvas, problem := ahdGraphicsCanvasOf(handle)
	if problem != "" {
		return problem
	}
	canvas.mu.Lock()
	defer canvas.mu.Unlock()
	if !canvas.usable {
		return ahdGraphicsClosedMessage
	}
	store(canvas)
	if canvas.listening[kind] {
		return ""
	}
	if _, problem := canvas.request(ahdGraphicsRequest{Op: "listen", Kind: kind}, ahdGraphicsRequestTimeout); problem != "" {
		return problem
	}
	canvas.listening[kind] = true
	return ""
}

// AhdGraphicsOnClick registers the Canvas click callback, replacing any
// earlier one. It receives the Cartesian point that was clicked.
func AhdGraphicsOnClick(handle int64, handler func(x, y float64)) string {
	if handler == nil {
		return "Canvas.onClick needs a Function"
	}
	return ahdGraphicsListen(handle, "click", func(c *ahdGraphicsCanvas) { c.onClick = handler })
}

// AhdGraphicsOnKey registers the Canvas key callback, replacing any earlier
// one. It receives the normalized name of each key pressed.
func AhdGraphicsOnKey(handle int64, handler func(key string)) string {
	if handler == nil {
		return "Canvas.onKey needs a Function"
	}
	return ahdGraphicsListen(handle, "key", func(c *ahdGraphicsCanvas) { c.onKey = handler })
}

func AhdGraphicsOnClickChecked(handle int64, handler func(x, y float64)) {
	AhdGraphicsCheck(AhdGraphicsOnClick(handle, handler))
}

func AhdGraphicsOnKeyChecked(handle int64, handler func(key string)) {
	AhdGraphicsCheck(AhdGraphicsOnKey(handle, handler))
}

// AhdGraphicsClose closes the Canvas window, ends its helper, and releases
// everything the Canvas owns: its own record and the state of every Turtle it
// handed out. A stale Canvas or Turtle value then finds nothing and reports a
// problem, never a crash. Closing a Canvas that is already closed does nothing.
func AhdGraphicsClose(handle int64) string {
	canvas, problem := ahdGraphicsCanvasOf(handle)
	if problem != "" {
		return ""
	}
	canvas.mu.Lock()
	if canvas.alive {
		_, _ = canvas.request(ahdGraphicsRequest{Op: "close"}, ahdGraphicsCloseTimeout)
		canvas.shutdown()
	}
	canvas.mu.Unlock()
	ahdGraphics.Lock()
	delete(ahdGraphics.canvases, handle)
	for turtle, state := range ahdGraphics.turtles {
		if state.canvas == handle {
			delete(ahdGraphics.turtles, turtle)
		}
	}
	ahdGraphics.Unlock()
	return ""
}

// AhdGraphicsIsOpen reports whether the Canvas can still be drawn on. It asks
// the helper, because the user may have closed the window since the last call.
func AhdGraphicsIsOpen(handle int64) bool {
	canvas, problem := ahdGraphicsCanvasOf(handle)
	if problem != "" {
		return false
	}
	canvas.mu.Lock()
	defer canvas.mu.Unlock()
	if !canvas.usable {
		return false
	}
	reply, problem := canvas.request(ahdGraphicsRequest{Op: "status"}, ahdGraphicsRequestTimeout)
	if problem != "" || reply.Open == nil || !*reply.Open {
		canvas.usable = false
		return false
	}
	return true
}

// AhdGraphicsCloseAll closes every Canvas. It runs when a program ends, so
// no window or helper outlives the program that opened it.
func AhdGraphicsCloseAll() {
	ahdGraphics.Lock()
	handles := make([]int64, 0, len(ahdGraphics.canvases))
	for handle := range ahdGraphics.canvases {
		handles = append(handles, handle)
	}
	ahdGraphics.Unlock()
	for _, handle := range handles {
		AhdGraphicsClose(handle)
	}
}

// AhdGraphicsTurtle creates a Turtle at the Canvas origin, facing +x, with
// its pen down, drawing black lines of width 1.
func AhdGraphicsTurtle(canvas int64) (int64, string) {
	if _, problem := ahdGraphicsCanvasOf(canvas); problem != "" {
		return 0, problem
	}
	ahdGraphics.Lock()
	defer ahdGraphics.Unlock()
	ahdGraphics.next++
	handle := ahdGraphics.next
	ahdGraphics.turtles[handle] = &ahdGraphicsTurtle{canvas: canvas, pen: true,
		color: ahdGraphicsNamedColors["black"], width: 1}
	return handle, ""
}

func ahdGraphicsTurtleOf(handle int64) *ahdGraphicsTurtle {
	ahdGraphics.Lock()
	defer ahdGraphics.Unlock()
	return ahdGraphics.turtles[handle]
}

// ahdGraphicsNormalizeHeading maps any angle into [0, 360).
func ahdGraphicsNormalizeHeading(degrees float64) float64 {
	heading := math.Mod(degrees, 360)
	if heading < 0 {
		heading += 360
	}
	if heading >= 360 {
		heading = 0
	}
	return heading + 0 // turns -0 into 0
}

// ahdGraphicsDirection is the unit vector of a heading in [0, 360). The four
// axis headings are exact, so a square drawn with 90-degree turns closes
// exactly instead of drifting by rounding error.
func ahdGraphicsDirection(heading float64) (float64, float64) {
	switch heading {
	case 0:
		return 1, 0
	case 90:
		return 0, 1
	case 180:
		return -1, 0
	case 270:
		return 0, -1
	}
	radians := heading * math.Pi / 180
	return math.Cos(radians), math.Sin(radians)
}

// moveTo moves the Turtle, drawing a Canvas line when its pen is down. The
// state changes only after the line is drawn, so a failed draw leaves the
// Turtle where it was.
func (t *ahdGraphicsTurtle) moveTo(x, y float64) string {
	if math.IsNaN(x) || math.IsInf(x, 0) || math.IsNaN(y) || math.IsInf(y, 0) {
		return "the Turtle cannot move that far: its position would not be a finite number"
	}
	if t.pen {
		color := fmt.Sprintf("#%02x%02x%02x%02x", t.color[0], t.color[1], t.color[2], t.color[3])
		if problem := AhdGraphicsLine(t.canvas, t.x, t.y, x, y, color, t.width); problem != "" {
			return problem
		}
	}
	t.x, t.y = x+0, y+0
	return ""
}

func (t *ahdGraphicsTurtle) advance(distance float64) string {
	dx, dy := ahdGraphicsDirection(t.heading)
	return t.moveTo(t.x+distance*dx, t.y+distance*dy)
}

func ahdGraphicsWithTurtle(handle int64, action func(t *ahdGraphicsTurtle) string) string {
	turtle := ahdGraphicsTurtleOf(handle)
	if turtle == nil {
		return ahdGraphicsReleasedMessage
	}
	turtle.mu.Lock()
	defer turtle.mu.Unlock()
	return action(turtle)
}

// AhdGraphicsTurtleForward moves distance units along the heading.
func AhdGraphicsTurtleForward(handle int64, distance float64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string { return t.advance(distance) })
}

// AhdGraphicsTurtleBackward moves distance units against the heading.
func AhdGraphicsTurtleBackward(handle int64, distance float64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string { return t.advance(-distance) })
}

// AhdGraphicsTurtleLeft turns counter-clockwise.
func AhdGraphicsTurtleLeft(handle int64, degrees float64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		t.heading = ahdGraphicsNormalizeHeading(t.heading + degrees)
		return ""
	})
}

// AhdGraphicsTurtleRight turns clockwise.
func AhdGraphicsTurtleRight(handle int64, degrees float64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		t.heading = ahdGraphicsNormalizeHeading(t.heading - degrees)
		return ""
	})
}

// AhdGraphicsTurtleMoveTo moves straight to (x, y).
func AhdGraphicsTurtleMoveTo(handle int64, x, y float64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string { return t.moveTo(x, y) })
}

// AhdGraphicsTurtleSetHeading faces an absolute direction without moving.
func AhdGraphicsTurtleSetHeading(handle int64, degrees float64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		t.heading = ahdGraphicsNormalizeHeading(degrees)
		return ""
	})
}

// AhdGraphicsTurtlePen lifts (false) or lowers (true) the pen.
func AhdGraphicsTurtlePen(handle int64, down bool) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		t.pen = down
		return ""
	})
}

// AhdGraphicsTurtleSetColor sets the color of later lines.
func AhdGraphicsTurtleSetColor(handle int64, color string) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		parsed, ok := AhdGraphicsParseColor(color)
		if !ok {
			return ahdGraphicsColorProblem("Turtle", color)
		}
		t.color = parsed
		return ""
	})
}

// AhdGraphicsTurtleSetWidth sets the width of later lines.
func AhdGraphicsTurtleSetWidth(handle int64, width float64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		if !(width > 0) {
			return fmt.Sprintf("Turtle line width must be greater than 0, not %s", ahdGraphicsNumber(width))
		}
		t.width = width
		return ""
	})
}

// AhdGraphicsTurtleHome moves to the origin, obeying the pen, then faces +x.
// It does not clear the Canvas or change the pen, color, or width.
func AhdGraphicsTurtleHome(handle int64) string {
	return ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		if problem := t.moveTo(0, 0); problem != "" {
			return problem
		}
		t.heading = 0
		return ""
	})
}

// AhdGraphicsTurtleState reads x, y, and heading, or reports that the
// Turtle's Canvas was closed and its state released.
func AhdGraphicsTurtleState(handle int64) (x, y, heading float64, problem string) {
	problem = ahdGraphicsWithTurtle(handle, func(t *ahdGraphicsTurtle) string {
		x, y, heading = t.x, t.y, t.heading
		return ""
	})
	return x, y, heading, problem
}

// AhdGraphicsTurtleXChecked, AhdGraphicsTurtleYChecked, and
// AhdGraphicsTurtleHeadingChecked read one part of a Turtle's state for a
// compiled program; the heading is always in [0, 360).
func AhdGraphicsTurtleXChecked(handle int64) float64 {
	x, _, _, problem := AhdGraphicsTurtleState(handle)
	AhdGraphicsCheck(problem)
	return x
}

func AhdGraphicsTurtleYChecked(handle int64) float64 {
	_, y, _, problem := AhdGraphicsTurtleState(handle)
	AhdGraphicsCheck(problem)
	return y
}

func AhdGraphicsTurtleHeadingChecked(handle int64) float64 {
	_, _, heading, problem := AhdGraphicsTurtleState(handle)
	AhdGraphicsCheck(problem)
	return heading
}

func ahdGraphicsNumber(value float64) string {
	return AhdStrReal(value)
}

// AhdGraphicsCheck raises a GraphicsError for a reported problem in a
// compiled program.
func AhdGraphicsCheck(problem string) {
	if problem != "" {
		AhdRaiseClass(AhdClassGraphicsError, problem)
	}
}

// AhdGraphicsOpenChecked is Graphics.open for a compiled program.
func AhdGraphicsOpenChecked(width, height int64, title, background string) int64 {
	handle, problem := AhdGraphicsOpen(width, height, title, background)
	AhdGraphicsCheck(problem)
	return handle
}

// AhdGraphicsTurtleChecked is Canvas.turtle for a compiled program.
func AhdGraphicsTurtleChecked(canvas int64) int64 {
	handle, problem := AhdGraphicsTurtle(canvas)
	AhdGraphicsCheck(problem)
	return handle
}
