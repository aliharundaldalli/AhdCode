package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runV280Example(t *testing.T, name, script string) (string, string) {
	t.Helper()
	useHeadlessGraphics(t)
	if script != "" {
		t.Setenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS", script)
	}
	entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v2.8", name, "main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	executable, result := BuildProgram(entry, filepath.Join(t.TempDir(), name))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	directory := t.TempDir()
	out, errorOutput, code := runIn(t, executable, directory)
	if code != 0 || errorOutput != "" {
		t.Fatalf("exit %d, stderr %q, stdout %q", code, errorOutput, out)
	}
	return out, directory
}

func requireFiles(t *testing.T, directory string, names ...string) {
	t.Helper()
	for _, name := range names {
		if info, err := os.Stat(filepath.Join(directory, name)); err != nil || info.Size() == 0 {
			t.Fatalf("%s missing or empty: %v", name, err)
		}
	}
}

// The timer example: two frames, a pause (the tick during it draws nothing),
// a resume, one more frame, a save, and Escape.
func TestV280BouncingBallExampleRuns(t *testing.T) {
	out, directory := runV280Example(t, "bouncing_ball", `[{"event":"tick"},{"event":"tick"},{"event":"key","key":"Space"},`+
		`{"event":"tick"},{"event":"key","key":"Space"},{"event":"tick"},{"event":"key","key":"S"},{"event":"key","key":"Escape"}]`)
	want := "Space pauses, S saves, Escape ends.\nPaused after 0 bounces.\nResumed.\nSaved bouncing-ball.png\nThe ball bounced 0 times.\n"
	if out != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out, want)
	}
	requireFiles(t, directory, "bouncing-ball.png")
}

// The drag example: a click starts a stroke of two segments, a plain move
// ends it, and the next two drags add one segment.
func TestV280FreeDrawingExampleRuns(t *testing.T) {
	out, directory := runV280Example(t, "free_drawing", `[{"event":"click","x":0,"y":0},{"event":"drag","x":10,"y":10},`+
		`{"event":"drag","x":20,"y":0},{"event":"move","x":50,"y":50},{"event":"drag","x":60,"y":60},{"event":"drag","x":70,"y":60},`+
		`{"event":"key","key":"S"},{"event":"key","key":"C"},{"event":"key","key":"Escape"}]`)
	if want := "Saved drawing.png with 3 segments.\nCleared.\nDone.\n"; out != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out, want)
	}
	requireFiles(t, directory, "drawing.png")
}

func TestV280ShapesAndTextExampleRuns(t *testing.T) {
	out, directory := runV280Example(t, "shapes_and_text", "")
	if want := "Saved shapes.png, shapes.svg and shapes-with-image.png\n"; out != want {
		t.Fatalf("stdout: %q", out)
	}
	requireFiles(t, directory, "shapes.png", "shapes.svg", "shapes-with-image.png")
	svg, _ := os.ReadFile(filepath.Join(directory, "shapes.svg"))
	for _, want := range []string{`<path aria-label="AhdCode Graphics"`, "<polygon points=", "<ellipse ", `stroke-linecap="round"`} {
		if !strings.Contains(string(svg), want) {
			t.Errorf("shapes.svg lacks %s", want)
		}
	}
}

// v280Program uses every v2.8 Canvas member with positional and named
// arguments, both polygon overloads, and the errors a wrong argument raises.
const v280Program = `bring Graphics
from Graphics bring (Canvas, GraphicsError)

canvas: Canvas := Graphics.open(200, 160, "v2.8")
canvas.text(0, 50, "Merhaba ğüşiöç", "blue", 20, "center")
canvas.text(x: -90, y: -70, text: "default")
canvas.polygon([-80, -20, -20, -20, -50, 30], "black", "#ff000080", 2)
canvas.polygon(points: [10.5, -20.0, 70.0, -20.0, 40.0, 30.0], fill: "yellow")
canvas.ellipse(0, 0, 30, 10)
canvas.ellipse(x: 0, y: 0, radiusX: 10, radiusY: 30, stroke: "green", fill: null, width: 2)
canvas.arc(0, 0, 40, 0, 270)
canvas.arc(x: 0, y: 0, radius: 50, start: 90, end: -90, color: "red", width: 3)
canvas.save("PNGPATH")
canvas.image("PNGPATH", -100, -80, 50)
canvas.image(path: "PNGPATH", x: 50, y: -80, width: 0, height: 20)
canvas.save("SVGPATH")

problems := [
    lambda [@canvas] () -> canvas.text(0, 0, "a", "black", 0.0),
    lambda [@canvas] () -> canvas.text(0, 0, "a", "black", 12, "middle"),
    lambda [@canvas] () -> canvas.polygon([0, 0, 1, 1]),
    lambda [@canvas] () -> canvas.polygon([0, 0, 1, 1, 2]),
    lambda [@canvas] () -> canvas.ellipse(0, 0, -1, 2),
    lambda [@canvas] () -> canvas.arc(0, 0, -1, 0, 90),
    lambda [@canvas] () -> canvas.image("absent.png", 0, 0),
    lambda [@canvas] () -> canvas.cancel(99)
]
for problem in problems {
    attempt {
        problem()
        write("no error")
    } except GraphicsError as error {
        write(error.message)
    }
}

ticks: Int := 0
frame: Function := () -> Nothing uses [@canvas, @ticks] {
    ticks += 1
    canvas.clear("white")
    canvas.text(0, 0, "tick {ticks}")
}
timer: Int := canvas.every(20, frame)
canvas.onMove(lambda (x: Real, y: Real) -> write("move {x} {y}"))
dragged: Function := (x: Real, y: Real) -> Nothing uses [@canvas, @timer] {
    write("drag {x} {y}")
    canvas.cancel(timer)
}
canvas.onDrag(dragged)
canvas.batch(lambda [@canvas] () -> canvas.circle(0, 0, 5))
canvas.wait()
write("ticks {ticks} open {canvas.isOpen()}")
`

func v280Source(directory string) string {
	source := strings.ReplaceAll(v280Program, "SVGPATH", filepath.ToSlash(filepath.Join(directory, "v280.svg")))
	return strings.ReplaceAll(source, "PNGPATH", filepath.ToSlash(filepath.Join(directory, "v280.png")))
}

func TestV280GraphicsNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGraphics(t)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS", `[{"event":"tick"},{"event":"move","x":5,"y":-6},{"event":"tick"},`+
		`{"event":"drag","x":1.5,"y":2},{"event":"tick"}]`)
	nativeDirectory, evaluatorDirectory := t.TempDir(), t.TempDir()

	directory := writeSources(t, map[string]string{"main.ahd": v280Source(nativeDirectory)})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("native program failed: code=%d stderr=%q stdout=%q", code, stderr, stdout)
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 11 {
		t.Fatalf("stdout:\n%s", stdout)
	}
	for index, line := range lines[:7] {
		if line == "no error" || line == "" {
			t.Errorf("problem %d raised nothing", index)
		}
	}
	// Cancelling a timer that is not running is harmless, so a program may
	// cancel twice or cancel after the timer was never started.
	if lines[7] != "no error" {
		t.Errorf("cancelling an unknown timer raised %q", lines[7])
	}
	if tail := strings.Join(lines[8:], "\n"); tail != "move 5.0 -6.0\ndrag 1.5 2.0\nticks 2 open false" {
		t.Fatalf("events:\n%s", tail)
	}

	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, v280Source(evaluatorDirectory), &output, &errorOutput)
	normalize := func(text, own string) string { return strings.ReplaceAll(text, filepath.ToSlash(own), "DIR") }
	if normalize(output.String(), evaluatorDirectory) != normalize(stdout, nativeDirectory) {
		t.Fatalf("evaluator stdout differs from native:\n evaluator: %q\n    native: %q", output.String(), stdout)
	}
	for _, name := range []string{"v280.svg", "v280.png"} {
		native, err := os.ReadFile(filepath.Join(nativeDirectory, name))
		if err != nil {
			t.Fatalf("native %s: %v", name, err)
		}
		evaluated, err := os.ReadFile(filepath.Join(evaluatorDirectory, name))
		if err != nil {
			t.Fatalf("evaluator %s: %v", name, err)
		}
		if !bytes.Equal(native, evaluated) {
			t.Fatalf("%s differs between the native program and the evaluator", name)
		}
	}
}
