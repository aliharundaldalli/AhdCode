package build

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	graphicsHelperBuild sync.Once
	graphicsHelperFile  string
	graphicsHelperFail  string
)

// useHeadlessGraphics builds the real ahdgraphics helper once and runs every
// Canvas in this test headless, so the parity checks need no display.
func useHeadlessGraphics(t *testing.T) {
	t.Helper()
	graphicsHelperBuild.Do(func() {
		directory, err := os.MkdirTemp("", "ahdgraphics-build-test-")
		if err != nil {
			graphicsHelperFail = err.Error()
			return
		}
		name := "ahdgraphics"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		graphicsHelperFile = filepath.Join(directory, name)
		source, _ := filepath.Abs(filepath.Join("..", "..", "cmd", "ahdgraphics"))
		command := exec.Command("go", "build", "-o", graphicsHelperFile, ".")
		command.Dir = source
		if output, err := command.CombinedOutput(); err != nil {
			graphicsHelperFail = err.Error() + "\n" + string(output)
		}
	})
	if graphicsHelperFail != "" {
		t.Fatalf("building ahdgraphics: %s", graphicsHelperFail)
	}
	t.Setenv("AHDCODE_GRAPHICS_RUNTIME", graphicsHelperFile)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS", "1")
}

// graphicsProgram exercises every Graphics operation with positional and
// named arguments, defaults, a null fill, errors, and the lifecycle, and
// saves one PNG and one SVG whose bytes both backends must reproduce.
const graphicsProgram = `bring Graphics
from Graphics bring (Canvas, Turtle, GraphicsError)

canvas: Canvas := Graphics.open(width: 320, height: 240, title: "Parity <&>", background: "#f0f0f0")
canvas.line(-160, 0, 160, 0)
canvas.line(x1: 0, y1: -120, x2: 0, y2: 120, color: "gray", width: 2)
canvas.circle(0, 0, 60, "red", "#ffff0080", 3)
canvas.circle(x: 40, y: 40, radius: 10, fill: null)
canvas.circle(x: 0, y: 0, radius: 0)
canvas.rectangle(-150, -110, 50, 30, "blue", "cyan", 2.5)
canvas.rectangle(x: 100, y: -110, width: 0, height: 20)

pen: Turtle := canvas.turtle()
write("start {pen.x()} {pen.y()} {pen.heading()}")
pen.setColor("magenta")
pen.setWidth(2)
side := 0
while side < 4 {
    pen.forward(50)
    pen.left(90)
    side = side + 1
}
write("square {pen.x()} {pen.y()} {pen.heading()}")
pen.penUp()
pen.moveTo(x: -100, y: 60)
pen.penDown()
pen.setHeading(-60)
pen.forward(40)
pen.backward(-10)
write("heading {pen.heading()}")
pen.left(450)
write("left 450 {pen.heading()}")
pen.right(720)
pen.home()
write("home {pen.x()} {pen.y()} {pen.heading()}")

other: Turtle := canvas.turtle()
other.right(90)
other.forward(30)
write("other {other.x()} {other.y()} {other.heading()} pen {pen.x()}")

canvas.save("SVGPATH")
canvas.save("PNGPATH")
write("open {canvas.isOpen()}")
for bad in ["purple", "Red", "#12345", ""] {
    attempt {
        canvas.line(0, 0, 1, 1, bad)
    } except GraphicsError as error {
        write(error.message)
    }
}
attempt {
    canvas.circle(x: 0, y: 0, radius: -1)
} except GraphicsError as error {
    write(error.message)
}
attempt {
    canvas.rectangle(0, 0, 1, -2)
} except GraphicsError as error {
    write(error.message)
}
attempt {
    canvas.save("drawing.jpg")
} except GraphicsError as error {
    write(error.message)
}
attempt {
    pen.setWidth(0)
} except GraphicsError as error {
    write(error.message)
}
attempt {
    Graphics.open(0, 10)
} except GraphicsError as error {
    write(error.message)
}
canvas.clear()
canvas.wait()
write("after wait {canvas.isOpen()}")
attempt {
    pen.forward(5)
} except GraphicsError as error {
    write(error.message)
}
canvas.close()
canvas.close()
attempt {
    write(str(pen.x()))
} except GraphicsError as error {
    write(error.message)
}
attempt {
    canvas.turtle()
} except GraphicsError as error {
    write(error.message)
}
write("done")
`

const graphicsExpectedStdout = "start 0.0 0.0 0.0\n" +
	"square 0.0 0.0 0.0\n" +
	"heading 300.0\n" +
	"left 450 30.0\n" +
	"home 0.0 0.0 0.0\n" +
	"other 0.0 -30.0 270.0 pen 0.0\n" +
	"open true\n" +
	"unsupported line color \"purple\"; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA\n" +
	"unsupported line color \"Red\"; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA\n" +
	"unsupported line color \"#12345\"; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA\n" +
	"unsupported line color \"\"; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA\n" +
	"circle radius must be 0 or greater, not -1.0\n" +
	"rectangle width and height must be 0 or greater, not 1.0 and -2.0\n" +
	"cannot save \"drawing.jpg\": Graphics saves only .png and .svg files\n" +
	"Turtle line width must be greater than 0, not 0.0\n" +
	"Canvas size 0x10 is not allowed; width and height must each be between 1 and 4096\n" +
	"after wait false\n" +
	"the Canvas is closed\n" +
	"the Turtle's Canvas was closed with close()\n" +
	"the Canvas is closed\n" +
	"done\n"

func graphicsSource(directory string) string {
	source := strings.ReplaceAll(graphicsProgram, "SVGPATH", filepath.ToSlash(filepath.Join(directory, "parity.svg")))
	return strings.ReplaceAll(source, "PNGPATH", filepath.ToSlash(filepath.Join(directory, "parity.png")))
}

func TestGraphicsNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGraphics(t)
	nativeDirectory, evaluatorDirectory := t.TempDir(), t.TempDir()

	directory := writeSources(t, map[string]string{"main.ahd": graphicsSource(nativeDirectory)})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("native program failed: code=%d stderr=%q", code, stderr)
	}
	if stdout != graphicsExpectedStdout {
		t.Fatalf("native stdout mismatch:\n got: %q\nwant: %q", stdout, graphicsExpectedStdout)
	}

	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, graphicsSource(evaluatorDirectory), &output, &errorOutput)
	if output.String() != stdout {
		t.Fatalf("evaluator stdout differs from native:\n evaluator: %q\n    native: %q", output.String(), stdout)
	}
	for _, name := range []string{"parity.svg", "parity.png"} {
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
	svg, _ := os.ReadFile(filepath.Join(nativeDirectory, "parity.svg"))
	for _, want := range []string{
		`width="320" height="240"`, `<title>Parity &lt;&amp;&gt;</title>`, `fill="#f0f0f0"`,
		`stroke="#ff00ff" stroke-width="2"`, `<circle cx="200" cy="80" r="10" fill="none"`,
	} {
		if !strings.Contains(string(svg), want) {
			t.Errorf("SVG lacks %s", want)
		}
	}
}

func TestGraphicsMissingHelperRaisesGraphicsError(t *testing.T) {
	t.Setenv("AHDCODE_GRAPHICS_RUNTIME", filepath.Join(t.TempDir(), "absent", "ahdgraphics"))
	source := `bring Graphics
from Graphics bring (GraphicsError)
attempt {
    Graphics.open()
} except GraphicsError as error {
    write(error.message)
}
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	stdout, _, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	want := "the Graphics window helper (ahdgraphics) was not found; reinstall AhdCode with its bundled helpers\n"
	if code != 0 || stdout != want {
		t.Fatalf("missing helper: code=%d stdout=%q", code, stdout)
	}
}

// A program that ends, or fails, with a Canvas still open leaves no helper
// process behind.
func TestGraphicsHelpersEndWithTheProgram(t *testing.T) {
	useHeadlessGraphics(t)
	for _, source := range []string{
		"bring Graphics\nfirst := Graphics.open(10, 10)\nsecond := Graphics.open(20, 20)\nwrite(\"opened\")\n",
		"bring Graphics\ncanvas := Graphics.open(10, 10)\ncanvas.line(0, 0, 1, 1, \"nope\")\n",
	} {
		directory := writeSources(t, map[string]string{"main.ahd": source})
		buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
		if runtime.GOOS == "windows" {
			continue
		}
		output, _ := exec.Command("pgrep", "-f", graphicsHelperFile).Output()
		if strings.TrimSpace(string(output)) != "" {
			t.Fatalf("helper processes outlived the program: %s", output)
		}
	}
}
