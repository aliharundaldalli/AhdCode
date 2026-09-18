package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	white = rgba{255, 255, 255, 255}
	black = rgba{0, 0, 0, 255}
	red   = rgba{255, 0, 0, 255}
	blue  = rgba{0, 0, 255, 255}
)

func testSnapshot(width, height int, commands ...command) snapshot {
	return snapshot{title: "t", width: width, height: height, background: white, commands: commands}
}

func pixel(t *testing.T, value snapshot, x, y int) color.NRGBA {
	t.Helper()
	img := render(value, 1)
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

func TestRasterBackgroundAndSize(t *testing.T) {
	value := testSnapshot(40, 30)
	value.background = rgba{10, 20, 30, 255}
	img := render(value, 1)
	if img.Bounds().Dx() != 40 || img.Bounds().Dy() != 30 {
		t.Fatalf("size %v", img.Bounds())
	}
	if got := pixel(t, value, 5, 5); got != (color.NRGBA{10, 20, 30, 255}) {
		t.Fatalf("background %v", got)
	}
}

func TestRasterUsesCartesianCoordinates(t *testing.T) {
	// A thick horizontal line at y = +10 lies ABOVE the center row in pixels.
	value := testSnapshot(100, 100, command{kind: lineCommand, a: -40, b: 10, c: 40, d: 10, stroke: red, width: 6})
	if got := pixel(t, value, 50, 40); got != (color.NRGBA{255, 0, 0, 255}) {
		t.Fatalf("pixel above center = %v, want red", got)
	}
	if got := pixel(t, value, 50, 60); got != (color.NRGBA{255, 255, 255, 255}) {
		t.Fatalf("pixel below center = %v, want white", got)
	}
}

func TestRasterRectangleCornerIsLowerLeft(t *testing.T) {
	value := testSnapshot(100, 100, command{kind: rectangleCommand, a: 0, b: 0, c: 30, d: 20, stroke: blue, fill: blue, hasFill: true, width: 1})
	// Filled region spans x 50..80 and, going up from the center row, y 30..50.
	if got := pixel(t, value, 65, 40); got != (color.NRGBA{0, 0, 255, 255}) {
		t.Fatalf("inside = %v", got)
	}
	if got := pixel(t, value, 65, 60); got != (color.NRGBA{255, 255, 255, 255}) {
		t.Fatalf("below the corner = %v", got)
	}
}

func TestRasterCircleFillAndStrokeOnly(t *testing.T) {
	filled := testSnapshot(100, 100, command{kind: circleCommand, a: 0, b: 0, c: 30, stroke: black, fill: red, hasFill: true, width: 2})
	if got := pixel(t, filled, 50, 50); got != (color.NRGBA{255, 0, 0, 255}) {
		t.Fatalf("filled center = %v", got)
	}
	ring := testSnapshot(100, 100, command{kind: circleCommand, a: 0, b: 0, c: 30, stroke: black, width: 4})
	if got := pixel(t, ring, 50, 50); got != (color.NRGBA{255, 255, 255, 255}) {
		t.Fatalf("stroke-only center = %v", got)
	}
	if got := pixel(t, ring, 80, 50); got != (color.NRGBA{0, 0, 0, 255}) {
		t.Fatalf("stroke on the circle = %v", got)
	}
}

func TestRasterDegenerateShapesDrawNothing(t *testing.T) {
	value := testSnapshot(20, 20,
		command{kind: circleCommand, a: 0, b: 0, c: 0, stroke: black, fill: black, hasFill: true, width: 5},
		command{kind: rectangleCommand, a: -5, b: -5, c: 0, d: 10, stroke: black, width: 5})
	img := render(value, 1)
	for index := 0; index < len(img.Pix); index++ {
		if img.Pix[index] != 255 {
			t.Fatalf("degenerate shape changed a pixel")
		}
	}
}

func TestRasterZeroLengthLineIsARoundDot(t *testing.T) {
	value := testSnapshot(20, 20, command{kind: lineCommand, a: 0, b: 0, c: 0, d: 0, stroke: black, width: 6})
	if got := pixel(t, value, 10, 10); got != (color.NRGBA{0, 0, 0, 255}) {
		t.Fatalf("dot center = %v", got)
	}
}

func TestRasterClipsFarGeometry(t *testing.T) {
	value := testSnapshot(50, 50,
		command{kind: lineCommand, a: -1e12, b: 0, c: 1e12, d: 0, stroke: black, width: 2},
		command{kind: circleCommand, a: 1e9, b: 1e9, c: 10, stroke: black, width: 1})
	if got := pixel(t, value, 25, 25); got.A == 0 || got.R != 0 {
		t.Fatalf("clipped line missing: %v", got)
	}
}

func TestPNGIsValidWithCanvasDimensions(t *testing.T) {
	data, err := encodePNG(testSnapshot(64, 48, command{kind: lineCommand, a: -10, b: 0, c: 10, d: 0, stroke: black, width: 1}))
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != 64 || config.Height != 48 {
		t.Fatalf("config %+v err %v", config, err)
	}
	again, _ := encodePNG(testSnapshot(64, 48, command{kind: lineCommand, a: -10, b: 0, c: 10, d: 0, stroke: black, width: 1}))
	if !bytes.Equal(data, again) {
		t.Fatalf("PNG export is not deterministic")
	}
}

func TestSVGStructureAndTransform(t *testing.T) {
	value := testSnapshot(200, 100,
		command{kind: lineCommand, a: -100, b: 25, c: 100, d: 25, stroke: red, width: 2},
		command{kind: circleCommand, a: 10, b: -10, c: 5, stroke: black, fill: rgba{0, 0, 255, 128}, hasFill: true, width: 1},
		command{kind: rectangleCommand, a: -50, b: -40, c: 20, d: 10, stroke: blue, width: 3},
		command{kind: circleCommand, a: 0, b: 0, c: 0, stroke: black, width: 1})
	value.title = `a <b> & "c"`
	text := string(encodeSVG(value))
	for _, want := range []string{
		`width="200" height="100" viewBox="0 0 200 100"`,
		`<title>a &lt;b&gt; &amp; &#34;c&#34;</title>`,
		`<rect x="0" y="0" width="200" height="100" fill="#ffffff"/>`,
		`<line x1="0" y1="25" x2="200" y2="25" stroke="#ff0000" stroke-width="2" stroke-linecap="round"/>`,
		`<circle cx="110" cy="60" r="5" fill="#0000ff" fill-opacity="0.502" stroke="#000000" stroke-width="1"/>`,
		// lower-left (-50, -40) with height 10: top edge at y = -30 -> pixel 80.
		`<rect x="50" y="80" width="20" height="10" fill="none" stroke="#0000ff" stroke-width="3"/>`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("SVG lacks %s\n%s", want, text)
		}
	}
	if strings.Count(text, "<circle") != 1 {
		t.Errorf("a zero-radius circle was serialized")
	}
	lower := strings.ToLower(text)
	for _, forbidden := range []string{"<script", "href", "url(", "<image", "<foreignobject", "@import"} {
		if strings.Contains(lower, forbidden) {
			t.Errorf("SVG contains %q", forbidden)
		}
	}
	if strings.Count(lower, "http") != 1 { // only the SVG namespace URI
		t.Errorf("SVG references an external URL")
	}
	decoder := xml.NewDecoder(strings.NewReader(text))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("SVG is not well-formed XML: %v", err)
		}
	}
}

func TestExportFormatCasePolicy(t *testing.T) {
	for path, want := range map[string]string{"a.png": "png", "a.PNG": "png", "a.Svg": "svg", "dir.x/a.svg": "svg"} {
		if got, err := exportFormat(path); err != nil || got != want {
			t.Errorf("%s -> %q, %v", path, got, err)
		}
	}
	for _, path := range []string{"a.jpg", "a.pdf", "a", "", "a.png.txt"} {
		if _, err := exportFormat(path); err == nil {
			t.Errorf("%q accepted", path)
		}
	}
}

func TestModelLimitsCommands(t *testing.T) {
	m := newModel("t", 10, 10, white)
	m.commands = make([]command, maxCommands)
	if err := m.add(command{kind: lineCommand, stroke: black, width: 1}); err == nil {
		t.Fatalf("command limit not enforced")
	}
	if err := m.apply(request{Op: "clear", Color: &white}); err != nil || len(m.commands) != 0 {
		t.Fatalf("clear did not reset commands")
	}
}

// protocolRun serves a headless session over in-memory pipes and returns
// every response line.
func protocolRun(t *testing.T, lines ...string) []response {
	t.Helper()
	input := strings.NewReader(strings.Join(lines, "\n") + "\n")
	var output bytes.Buffer
	in := bufio.NewReaderSize(input, maxRequestBytes+1)
	out := bufio.NewWriter(&output)
	first, err := readRequest(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateOpen(first); err != nil {
		_ = writeResponse(out, response{Error: err.Error()})
	} else {
		s := &session{model: newModel(first.Title, first.Width, first.Height, *first.Background), in: in, out: out,
			headless: true, windowGone: make(chan struct{})}
		close(s.windowGone)
		_ = writeResponse(out, response{OK: true})
		s.serve()
	}
	var replies []response
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var reply response
		if err := json.Unmarshal([]byte(line), &reply); err != nil {
			t.Fatalf("bad response line %q", line)
		}
		replies = append(replies, reply)
	}
	return replies
}

const openLine = `{"op":"open","version":1,"width":100,"height":80,"title":"x","headless":true,"background":[255,255,255,255]}`

func TestProtocolLifecycle(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "out.PNG")
	replies := protocolRun(t, openLine,
		`{"op":"line","x1":0,"y1":0,"x2":10,"y2":0,"color":[0,0,0,255],"lineWidth":1}`,
		`{"op":"status"}`,
		`{"op":"save","path":`+strings.ReplaceAll(`"`+path+`"`, `\`, `\\`)+`}`,
		`{"op":"line","x1":0,"y1":0,"x2":10,"y2":0,"color":[0,0,0,255],"lineWidth":0}`,
		`{"op":"unknownField","bogus":1}`,
		`{"op":"wait"}`,
		`{"op":"status"}`,
		`{"op":"circle","x":0,"y":0,"radius":1,"stroke":[0,0,0,255],"lineWidth":1}`,
		`{"op":"close"}`,
		`{"op":"status"}`)
	if len(replies) != 10 {
		t.Fatalf("replies after close were served: %+v", replies)
	}
	if !replies[0].OK || !replies[1].OK || !replies[2].OK || replies[2].Open == nil || !*replies[2].Open || !replies[3].OK {
		t.Fatalf("open/draw/status/save: %+v", replies[:4])
	}
	if replies[4].OK || replies[5].OK {
		t.Fatalf("invalid width or unknown field accepted: %+v", replies[4:6])
	}
	if !replies[6].OK || replies[7].Open == nil || *replies[7].Open {
		t.Fatalf("wait/status after wait: %+v", replies[6:8])
	}
	if replies[8].OK || !replies[8].Closed {
		t.Fatalf("drawing after wait was accepted: %+v", replies[8])
	}
	if !replies[9].OK {
		t.Fatalf("close: %+v", replies[9])
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("save did not write the PNG: %v", err)
	}
}

func TestProtocolRejectsBadOpen(t *testing.T) {
	for _, line := range []string{
		`{"op":"open","version":2,"width":100,"height":80,"background":[0,0,0,255]}`,
		`{"op":"open","version":1,"width":0,"height":80,"background":[0,0,0,255]}`,
		`{"op":"open","version":1,"width":5000,"height":80,"background":[0,0,0,255]}`,
		`{"op":"open","version":1,"width":10,"height":10}`,
		`{"op":"line"}`,
	} {
		replies := protocolRun(t, line)
		if len(replies) != 1 || replies[0].OK {
			t.Errorf("%s accepted: %+v", line, replies)
		}
	}
}

func TestProtocolBoundsRequestSize(t *testing.T) {
	huge := `{"op":"save","path":"` + strings.Repeat("a", maxRequestBytes+10) + `.png"}`
	replies := protocolRun(t, openLine, huge, `{"op":"status"}`)
	if len(replies) != 2 || replies[1].OK || replies[1].Error != errRequestTooLarge.Error() {
		t.Fatalf("oversized request: %+v", replies)
	}
}
