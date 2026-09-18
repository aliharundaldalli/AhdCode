package semantic

import (
	"strings"
	"testing"
)

const graphicsPreamble = "bring Graphics\nfrom Graphics bring (Canvas, Turtle, GraphicsError)\n\ncanvas: Canvas := Graphics.open()\npen: Turtle := canvas.turtle()\n"

func TestGraphicsModuleRegistered(t *testing.T) {
	module, ok := StandardModuleInterfaces()["Graphics"]
	if !ok {
		t.Fatal("Graphics module not registered")
	}
	want := []string{"Canvas", "GraphicsError", "Turtle", "open"}
	if strings.Join(module.ExportNames, ",") != strings.Join(want, ",") {
		t.Fatalf("Graphics exports %v, want exactly %v", module.ExportNames, want)
	}
	// Graphics is 2D drawing, not a game engine or a GUI toolkit.
	for _, name := range []string{"Sprite", "Scene", "Game", "loop", "update", "draw", "tick", "fps", "onClick", "mouseX",
		"isKeyDown", "Button", "TextInput", "sound", "text", "image", "Turtle.speed"} {
		if module.Exports[name] != nil {
			t.Fatalf("Graphics must not export %q", name)
		}
	}
	canvas := strings.Join(GraphicsCanvasOperations, ",")
	turtle := strings.Join(GraphicsTurtleOperations, ",")
	if canvas != "clear,line,circle,rectangle,save,wait,close,isOpen,turtle" ||
		turtle != "forward,backward,left,right,moveTo,setHeading,penUp,penDown,setColor,setWidth,home,x,y,heading" {
		t.Fatalf("unexpected member surface:\n%s\n%s", canvas, turtle)
	}
}

func TestGraphicsValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, graphicsPreamble+`small: Canvas := Graphics.open(400, 300)
named: Canvas := Graphics.open(width: 640, height: 480, title: "Shapes", background: "#101010")
canvas.clear()
canvas.clear("black")
canvas.clear(color: "#ffffff")
canvas.line(0, 0, 100, 50)
canvas.line(0, 0, 100, 50, "red", 2.5)
canvas.line(x1: -1.5, y1: 0, x2: 10, y2: 0, width: 3)
canvas.circle(0, 0, 50)
canvas.circle(0, 0, 50, "blue", "yellow", 2)
canvas.circle(0, 0, 50, "blue", null, 2)
canvas.circle(x: 0, y: 0, radius: 10, fill: "red")
maybeFill: String? := null
canvas.circle(x: 0, y: 0, radius: 10, fill: maybeFill)
canvas.rectangle(-50, -50, 100, 60)
canvas.rectangle(x: 0, y: 0, width: 20, height: 10, stroke: "green", fill: "#00ff0080", lineWidth: 4)
canvas.save("out.png")
open: Bool := canvas.isOpen()
distance: Int := 25
pen.forward(distance)
pen.forward(12.5)
pen.backward(-3)
pen.left(90)
pen.right(-45.5)
pen.moveTo(10, 20)
pen.moveTo(x: -5, y: 5)
pen.setHeading(180)
pen.penUp()
pen.penDown()
pen.setColor("magenta")
pen.setWidth(2)
pen.home()
x: Real := pen.x()
y: Real := pen.y()
heading: Real := pen.heading()
second: Turtle := small.turtle()
attempt {
    canvas.line(0, 0, 1, 1, "purple")
} except GraphicsError as error {
    write(error.message)
}
canvas.wait()
canvas.close()
`)
	requireSemanticClean(t, result)
}

// Static mistakes are compiler diagnostics, never a runtime GraphicsError.
func TestGraphicsRejectsStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`Graphics.open("800")`,
		`Graphics.open(800, 600, 1)`,
		`Graphics.open(width: 800.5)`,
		`Graphics.open(size: 800)`,
		`Graphics.open(800, 600, "t", "white", "extra")`,
		`Canvas()`,
		`Turtle()`,
		`canvas.line(0, 0, 1)`,
		`canvas.line("0", 0, 1, 1)`,
		`canvas.line(0, 0, 1, 1, 2)`,
		`canvas.line(x1: 0, y1: 0, x2: 1)`,
		`canvas.line(x1: 0, y1: 0, x2: 1, y2: 1, thickness: 2)`,
		`canvas.line(x1: 0, x1: 1, y1: 0, x2: 1, y2: 1)`,
		`canvas.circle(0, 0)`,
		`canvas.circle(0, 0, 5, null)`,
		`canvas.circle(x: 0, y: 0, radius: 5, stroke: null)`,
		`canvas.circle(0, 0, 5, "red", 3)`,
		"maybe: Real? := null\ncanvas.circle(0, 0, maybe)",
		`canvas.rectangle(0, 0, 5)`,
		`canvas.rectangle(x: 0, y: 0, width: 1, height: 1, width2: 3)`,
		`canvas.save()`,
		`canvas.save(1)`,
		`canvas.wait(1)`,
		`canvas.close(true)`,
		`flag: Int := canvas.isOpen()`,
		`canvas.turtle(1)`,
		`canvas.fill(0, 0)`,
		`canvas.text("hello")`,
		`canvas.onClick()`,
		`pen.forward()`,
		`pen.forward("10")`,
		`pen.forward(10, 20)`,
		`pen.forward(steps: 10)`,
		`pen.moveTo(1)`,
		`pen.setColor(1)`,
		`pen.setColor(null)`,
		`pen.setWidth("2")`,
		`pen.penUp(true)`,
		`count: Int := pen.x()`,
		`pen.speed(5)`,
		`pen.stamp()`,
		`mover := pen.forward`,
		`result: Real := pen.forward(1)`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, graphicsPreamble+source+"\n"))
	}
}

func TestGraphicsSelectiveAndAliasedImports(t *testing.T) {
	for _, source := range []string{
		"from Graphics bring open\ncanvas := open(100, 100)\ncanvas.line(0, 0, 1, 1)\n",
		"bring Graphics as G\ncanvas := G.open()\nturtle := canvas.turtle()\nturtle.forward(10)\n",
	} {
		requireSemanticClean(t, analyzeWithStandardModules(t, source))
	}
}
