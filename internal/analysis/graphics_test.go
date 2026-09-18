package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// Graphics reaches editors through the compiler's own module interface and
// the Canvas/Turtle member Symbols the compiler checks calls against; there
// is no separate editor catalog.

const graphicsSource = `bring Graphics
canvas := Graphics.open(width: 400, height: 300)
canvas.circle(x: 0, y: 0, radius: 40, fill: null)
canvas.rectangle(0, 0, 20, 10)
pen := canvas.turtle()
pen.forward(50)
position := pen.x()
`

func TestCompletionOffersGraphicsModuleAndMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, "bring \n")
	if items := store.Completion(path, len("bring ")); !hasLabel(items, "Graphics") {
		t.Fatalf("expected Graphics among module completions, got %#v", items)
	}
	text := "bring Graphics\nGraphics.\n"
	store.Open(path, text)
	items := store.Completion(path, len("bring Graphics\nGraphics."))
	for _, name := range []string{"open", "Canvas", "Turtle", "GraphicsError"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Graphics completions, got %#v", name, items)
		}
	}

	text = "bring Graphics\ncanvas := Graphics.open()\ncanvas.\n"
	store.Open(path, text)
	items = store.Completion(path, len(text)-1)
	for _, name := range []string{"clear", "line", "circle", "rectangle", "save", "wait", "close", "isOpen", "turtle"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Canvas completions, got %#v", name, items)
		}
	}
	for _, item := range items {
		if item.Label == "circle" && !strings.Contains(item.Detail, "fill: String?") {
			t.Fatalf("circle completion does not show the nullable fill: %q", item.Detail)
		}
	}

	text = "bring Graphics\ncanvas := Graphics.open()\npen := canvas.turtle()\npen.\n"
	store.Open(path, text)
	items = store.Completion(path, len(text)-1)
	for _, name := range []string{"forward", "backward", "left", "right", "moveTo", "setHeading", "penUp", "penDown",
		"setColor", "setWidth", "home", "x", "y", "heading"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Turtle completions, got %#v", name, items)
		}
	}
	if hasLabel(items, "speed") || hasLabel(items, "clear") {
		t.Fatalf("Turtle completion offers a member it does not have: %#v", items)
	}
}

func TestHoverShowsGraphicsSignatures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, graphicsSource)
	for _, testCase := range []struct{ target, want string }{
		{"Graphics.open", "open: (width: Int := default, height: Int := default, title: String := default, background: String := default) -> Canvas"},
		{"canvas.circle", "circle: (x: Real, y: Real, radius: Real, stroke: String := default, fill: String? := default, width: Real := default) -> Nothing"},
		{"canvas.rectangle", "rectangle: (x: Real, y: Real, width: Real, height: Real, stroke: String := default, fill: String? := default, lineWidth: Real := default) -> Nothing"},
		{"canvas.turtle", "turtle: () -> Turtle"},
		{"pen.forward", "forward: (distance: Real) -> Nothing"},
		{"pen.x", "x: () -> Real"},
	} {
		dot := strings.Index(testCase.target, ".")
		hover, ok := store.Hover(path, offsetOf(t, graphicsSource, testCase.target)+dot+2)
		if !ok || hover.Text != testCase.want {
			t.Fatalf("hover on %s = %q (%v), want %q", testCase.target, hover.Text, ok, testCase.want)
		}
	}
}

func TestSignatureHelpForGraphicsMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, graphicsSource)
	help, ok := store.SignatureHelp(path, offsetOf(t, graphicsSource, "canvas.circle(")+len("canvas.circle("))
	if !ok || !strings.Contains(help.Label, "fill: String?") || len(help.Parameters) != 6 || help.Parameters[4] != "fill: String? := default" {
		t.Fatalf("signature help for canvas.circle = %#v (%v)", help, ok)
	}
	help, ok = store.SignatureHelp(path, offsetOf(t, graphicsSource, "pen.forward(")+len("pen.forward("))
	if !ok || help.Label != "(distance: Real) -> Nothing" {
		t.Fatalf("signature help for pen.forward = %#v (%v)", help, ok)
	}
	help, ok = store.SignatureHelp(path, offsetOf(t, graphicsSource, "Graphics.open(")+len("Graphics.open("))
	if !ok || len(help.Parameters) != 4 || help.Parameters[0] != "width: Int := default" {
		t.Fatalf("signature help for Graphics.open = %#v (%v)", help, ok)
	}
}
