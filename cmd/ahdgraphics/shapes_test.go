package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	redPixel   = color.NRGBA{255, 0, 0, 255}
	whitePixel = color.NRGBA{255, 255, 255, 255}
)

func TestRasterEllipseHasTwoRadii(t *testing.T) {
	value := testSnapshot(100, 100, command{kind: ellipseCommand, a: 0, b: 0, c: 40, d: 10, stroke: red, fill: red, hasFill: true, width: 1})
	if got := pixel(t, value, 85, 50); got != redPixel {
		t.Fatalf("inside the horizontal radius = %v", got)
	}
	if got := pixel(t, value, 50, 30); got != whitePixel {
		t.Fatalf("outside the vertical radius = %v", got)
	}
	outline := testSnapshot(100, 100, command{kind: ellipseCommand, a: 0, b: 0, c: 40, d: 10, stroke: red, width: 2})
	if got := pixel(t, outline, 50, 50); got != whitePixel {
		t.Fatalf("an ellipse without a fill has a filled center: %v", got)
	}
}

func TestRasterPolygonFillsAndClosesItsOutline(t *testing.T) {
	// A triangle whose apex points up: Cartesian y grows upward.
	points := []float64{-30, -20, 30, -20, 0, 30}
	filled := testSnapshot(100, 100, command{kind: polygonCommand, points: points, stroke: red, fill: red, hasFill: true, width: 1})
	if got := pixel(t, filled, 50, 55); got != redPixel {
		t.Fatalf("inside = %v", got)
	}
	if got := pixel(t, filled, 25, 25); got != whitePixel {
		t.Fatalf("beside the apex = %v", got)
	}
	outline := testSnapshot(100, 100, command{kind: polygonCommand, points: points, stroke: red, width: 3})
	if got := pixel(t, outline, 50, 55); got != whitePixel {
		t.Fatalf("an outline-only polygon is filled: %v", got)
	}
	// The closing edge runs from the last vertex back to the first.
	if got := pixel(t, outline, 35, 45); got != redPixel {
		t.Fatalf("closing edge = %v", got)
	}
}

func TestRasterArcRunsCounterclockwiseFromPositiveX(t *testing.T) {
	// 0 to 90 degrees is the upper-right quarter only.
	value := testSnapshot(100, 100, command{kind: arcCommand, a: 0, b: 0, c: 30, start: 0, end: 90, stroke: red, width: 5})
	if got := pixel(t, value, 71, 29); got != redPixel {
		t.Fatalf("upper right = %v", got)
	}
	for _, point := range [][2]int{{29, 29}, {29, 71}, {71, 71}} {
		if got := pixel(t, value, point[0], point[1]); got != whitePixel {
			t.Fatalf("pixel %v outside the quarter = %v", point, got)
		}
	}
	// A negative sweep goes clockwise, into the lower-right quarter.
	clockwise := testSnapshot(100, 100, command{kind: arcCommand, a: 0, b: 0, c: 30, start: 0, end: -90, stroke: red, width: 5})
	if got := pixel(t, clockwise, 71, 71); got != redPixel {
		t.Fatalf("lower right = %v", got)
	}
	if got := pixel(t, clockwise, 71, 29); got != whitePixel {
		t.Fatalf("upper right of a clockwise arc = %v", got)
	}
}

// inked returns the horizontal extent of every non-background pixel.
func inked(img *image.RGBA) (left, right, count int) {
	left, right = img.Bounds().Dx(), -1
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA) != whitePixel {
				count++
				left, right = min(left, x), max(right, x)
			}
		}
	}
	return left, right, count
}

func TestRasterTextAlignsAroundItsAnchor(t *testing.T) {
	draw := func(align string) (int, int, int) {
		return inked(render(testSnapshot(300, 100, command{kind: textCommand, a: 0, b: 0, c: 24, text: "AhdCode", align: align, stroke: black}), 1))
	}
	left, _, count := draw("left")
	if count == 0 || left < 150 {
		t.Fatalf("left-aligned text starts at %d with %d pixels", left, count)
	}
	_, right, _ := draw("right")
	if right > 150 {
		t.Fatalf("right-aligned text ends at %d", right)
	}
	from, to, _ := draw("center")
	if middle := (from + to) / 2; middle < 145 || middle > 155 {
		t.Fatalf("centered text spans %d..%d", from, to)
	}
	// The anchor is on the baseline: nothing of "A" hangs below it.
	img := render(testSnapshot(100, 100, command{kind: textCommand, a: 0, b: 0, c: 30, text: "A", align: "left", stroke: black}), 1)
	for y := 53; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA) != whitePixel {
				t.Fatalf("ink below the baseline at %d,%d", x, y)
			}
		}
	}
}

func TestTextCoversTurkishLetters(t *testing.T) {
	for _, letter := range "çğıİöşüÇĞÖŞÜ" {
		_, _, count := inked(render(testSnapshot(60, 60, command{kind: textCommand, a: -10, b: -10, c: 30, text: string(letter), align: "left", stroke: black}), 1))
		if count == 0 {
			t.Fatalf("%q draws nothing", letter)
		}
	}
}

func writePicture(t *testing.T, width, height int, fill color.NRGBA) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	path := filepath.Join(t.TempDir(), "picture.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImageSizesAndAnchorsAtTheLowerLeft(t *testing.T) {
	path := writePicture(t, 20, 10, redPixel)
	canvas := &model{title: "t", width: 100, height: 100, background: white}
	if err := canvas.apply(request{Op: "image", Path: path, X: 0, Y: 0}); err != nil {
		t.Fatal(err)
	}
	// Width alone keeps the 2:1 aspect ratio.
	if err := canvas.apply(request{Op: "image", Path: path, X: -40, Y: -40, W: 40}); err != nil {
		t.Fatal(err)
	}
	if len(canvas.commands) != 2 || canvas.commands[0].c != 20 || canvas.commands[0].d != 10 ||
		canvas.commands[1].c != 40 || canvas.commands[1].d != 20 {
		t.Fatalf("image sizes: %+v", canvas.commands)
	}
	if canvas.commands[0].picture != canvas.commands[1].picture {
		t.Fatal("one file was decoded twice")
	}
	img := render(canvas.snapshot(), 1)
	at := func(x, y int) color.NRGBA { return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA) }
	if at(60, 45) != redPixel || at(60, 55) != whitePixel || at(45, 45) != whitePixel {
		t.Fatalf("natural-size image: %v %v %v", at(60, 45), at(60, 55), at(45, 45))
	}
	if at(30, 80) != redPixel {
		t.Fatalf("scaled image: %v", at(30, 80))
	}

	for _, bad := range []request{
		{Op: "image", Path: filepath.Join(t.TempDir(), "absent.png")},
		{Op: "image", Path: path, W: -1},
		{Op: "image", Path: ""},
	} {
		if err := canvas.apply(bad); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	text := filepath.Join(t.TempDir(), "note.png")
	if err := os.WriteFile(text, []byte("not a picture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := canvas.apply(request{Op: "image", Path: text}); err == nil {
		t.Fatal("a file that is not a picture was accepted")
	}
}

func TestModelValidatesNewShapes(t *testing.T) {
	ink := &rgba{0, 0, 0, 255}
	canvas := &model{title: "t", width: 100, height: 100, background: white}
	for _, good := range []request{
		{Op: "ellipse", X: 0, Y: 0, RX: 10, RY: 5, Stroke: ink, LineWidth: 1},
		{Op: "polygon", Points: []float64{0, 0, 10, 0, 0, 10}, Stroke: ink, Fill: ink, LineWidth: 1},
		{Op: "arc", Radius: 10, Start: 0, End: 720, Color: ink, LineWidth: 1},
		{Op: "text", Text: "", Size: 12, Align: "left", Color: ink},
		{Op: "text", Text: "Merhaba", Size: 12, Align: "center", Color: ink},
	} {
		if err := canvas.apply(good); err != nil {
			t.Fatalf("%+v: %v", good, err)
		}
	}
	for _, bad := range []request{
		{Op: "ellipse", RX: -1, RY: 5, Stroke: ink, LineWidth: 1},
		{Op: "ellipse", RX: 1, RY: 5, LineWidth: 1},
		{Op: "polygon", Points: []float64{0, 0, 10, 0}, Stroke: ink, LineWidth: 1},
		{Op: "polygon", Points: []float64{0, 0, 10, 0, 0, 10, 5}, Stroke: ink, LineWidth: 1},
		{Op: "polygon", Points: make([]float64, 2*maxPolygonPoints+2), Stroke: ink, LineWidth: 1},
		{Op: "arc", Radius: -1, Color: ink, LineWidth: 1},
		{Op: "arc", Radius: 1, Color: ink},
		{Op: "text", Text: "a\nb", Size: 12, Align: "left", Color: ink},
		{Op: "text", Text: "a", Size: 0, Align: "left", Color: ink},
		{Op: "text", Text: "a", Size: 12, Align: "middle", Color: ink},
		{Op: "text", Text: strings.Repeat("a", maxTextRunes+1), Size: 12, Align: "left", Color: ink},
	} {
		if err := canvas.apply(bad); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestSVGCarriesTheNewShapes(t *testing.T) {
	path := writePicture(t, 4, 4, redPixel)
	canvas := &model{title: "t", width: 200, height: 100, background: white}
	ink := &rgba{0, 0, 0, 255}
	for _, value := range []request{
		{Op: "ellipse", X: 0, Y: 0, RX: 30, RY: 10, Stroke: ink, LineWidth: 2},
		{Op: "polygon", Points: []float64{0, 0, 10, 0, 0, 10}, Stroke: ink, Fill: ink, LineWidth: 1},
		{Op: "arc", Radius: 10, Start: 0, End: 90, Color: ink, LineWidth: 1},
		{Op: "text", Text: `a<b & "c"`, Size: 12, Align: "left", Color: ink},
		{Op: "image", Path: path, X: 0, Y: 0},
	} {
		if err := canvas.apply(value); err != nil {
			t.Fatal(err)
		}
	}
	text := string(encodeSVG(canvas.snapshot()))
	for _, want := range []string{
		`<ellipse cx="100" cy="50" rx="30" ry="10" fill="none"`,
		`<polygon points="100,50 110,50 100,40"`,
		`<path d="M110 50 A10 10 0 0 0 100 40" fill="none"`,
		`<path aria-label="a&lt;b &amp; &#34;c&#34;"`,
		`<image x="100" y="46" width="4" height="4" preserveAspectRatio="none" href="data:image/png;base64,`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("SVG lacks %s\n%s", want, text)
		}
	}
}

func TestProtocolNewShapesPointerEventsAndBatch(t *testing.T) {
	open := `{"op":"open","id":1,"version":3,"width":100,"height":80,"headless":true,"background":[255,255,255,255],` +
		`"script":[{"event":"move","x":1,"y":2},{"event":"tick"},{"event":"drag","x":3,"y":4}]}`
	replies, events := protocolRunEvents(t, open,
		`{"op":"ellipse","id":2,"x":0,"y":0,"rx":10,"ry":5,"stroke":[0,0,0,255],"lineWidth":1}`,
		`{"op":"polygon","id":3,"points":[0,0,10,0,0,10],"stroke":[0,0,0,255],"lineWidth":1}`,
		`{"op":"arc","id":4,"x":0,"y":0,"radius":10,"start":0,"end":90,"color":[0,0,0,255],"lineWidth":1}`,
		`{"op":"text","id":5,"x":0,"y":0,"text":"Merhaba","size":16,"align":"center","color":[0,0,0,255]}`,
		`{"op":"polygon","id":6,"points":[0,0,10,0],"stroke":[0,0,0,255],"lineWidth":1}`,
		`{"op":"hold","id":7}`,
		`{"op":"release","id":8}`,
		`{"op":"listen","id":9,"kind":"move"}`,
		`{"op":"listen","id":10,"kind":"drag"}`,
		`{"op":"wait","id":11}`,
		`{"op":"close","id":12}`)
	if len(replies) != 12 {
		t.Fatalf("replies: %+v", replies)
	}
	for index, reply := range replies {
		if reply.OK == (index == 5) {
			t.Fatalf("reply %d: %+v", index, reply)
		}
	}
	want := []event{{Event: "move", X: 1, Y: 2}, {Event: "tick"}, {Event: "drag", X: 3, Y: 4}, {Event: "closed"}}
	if len(events) != len(want) {
		t.Fatalf("events: %+v", events)
	}
	for index := range want {
		if events[index] != want[index] {
			t.Fatalf("event %d = %+v, want %+v", index, events[index], want[index])
		}
	}
}

func TestProtocolRejectsAnOlderRuntime(t *testing.T) {
	replies, _ := protocolRunEvents(t, strings.Replace(openLine, `"version":3`, `"version":2`, 1))
	if len(replies) != 1 || replies[0].OK {
		t.Fatalf("a version 2 open was accepted: %+v", replies)
	}
}
