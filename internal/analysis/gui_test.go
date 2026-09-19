package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// GUI and the Canvas events reach editors through the compiler's own module
// interface and member Symbols; there is no separate editor catalog.

const guiSource = `bring GUI
bring Graphics
window := GUI.window(title: "Form", width: 400, height: 300)
form := window.column(spacing: 8, padding: 12)
name := form.textInput(placeholder: "name")
paid := form.checkbox("Paid", true)
save := form.button("Save")
status := form.label("Ready")
save.onClick(lambda () -> status.setText("x"))
window.onKey(lambda (key: String) -> write(key))
text := name.text()
flag := paid.checked()
canvas := Graphics.open()
canvas.onClick(lambda (x: Real, y: Real) -> write(x))
canvas.onKey(lambda (key: String) -> write(key))
window.setBackground("#F0F4F8")
form.setBackground("white")
status.setForeground("blue")
save.setBackground(color: "#0066CC")
save.setEnabled(false)
active := save.isEnabled()
`

func TestCompletionOffersGUIModuleAndMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, "bring \n")
	if items := store.Completion(path, len("bring ")); !hasLabel(items, "GUI") {
		t.Fatalf("expected GUI among module completions")
	}
	for _, testCase := range []struct {
		prefix string
		want   []string
		absent []string
	}{
		{"bring GUI\nGUI.", []string{"window", "Window", "Container", "Label", "Button", "TextInput", "Checkbox", "GUIError",
			"ListBox", "Select", "TextArea", "PasswordInput", "TableView", "openFile", "openFiles", "selectFolder", "saveFile", "message", "confirm"},
			[]string{"alert", "fileDialog", "Table", "RadioGroup", "TreeView"}},
		{"bring GUI\nw := GUI.window()\nw.", []string{"column", "row", "onKey", "wait", "close", "isOpen", "setTitle", "setBackground", "setResizable", "isResizable"}, []string{"onClick", "menu", "setForeground", "setEnabled", "setTheme", "resize"}},
		{"bring GUI\nc := GUI.window().column()\nc.", []string{"column", "row", "label", "button", "textInput", "checkbox", "setBackground",
			"passwordInput", "textArea", "listBox", "select", "table"}, []string{"dropdown", "tree", "setForeground", "setEnabled", "scrollColumn"}},
		{"bring GUI\nb := GUI.window().column().button(\"x\")\nb.", []string{"text", "setText", "onClick", "setForeground", "setBackground", "setEnabled", "isEnabled"}, []string{"setVisible", "setFont", "setStyle"}},
		{"bring GUI\ni := GUI.window().column().textInput()\ni.", []string{"text", "setText", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled"}, []string{"onKey", "setFont"}},
		{"bring GUI\nk := GUI.window().column().checkbox(\"x\")\nk.", []string{"checked", "setChecked", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled"}, []string{"onClick", "setFont"}},
		{"bring GUI\nl := GUI.window().column().label(\"x\")\nl.", []string{"text", "setText", "setForeground", "setBackground"}, []string{"onClick", "setEnabled", "isEnabled", "setVisible", "setFont"}},
		{"bring Graphics\ncanvas := Graphics.open()\ncanvas.", []string{"onClick", "onKey", "wait", "turtle"}, []string{"mouseX", "isKeyDown"}},
	} {
		store.Open(path, testCase.prefix+"\n")
		items := store.Completion(path, len(testCase.prefix))
		for _, name := range testCase.want {
			if !hasLabel(items, name) {
				t.Fatalf("after %q expected %q, got %#v", testCase.prefix, name, items)
			}
		}
		for _, name := range testCase.absent {
			if hasLabel(items, name) {
				t.Fatalf("after %q unexpected %q", testCase.prefix, name)
			}
		}
	}
}

func TestHoverAndSignatureHelpForGUI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, guiSource)
	for _, testCase := range []struct{ target, want string }{
		{"GUI.window", "window: (title: String := default, width: Int := default, height: Int := default) -> Window"},
		{"window.column", "column: (spacing: Int := default, padding: Int := default) -> Container"},
		{"form.textInput", "textInput: (placeholder: String := default) -> TextInput"},
		{"form.checkbox", "checkbox: (text: String, checked: Bool := default) -> Checkbox"},
		{"save.onClick", "onClick: (handler: Function() -> Nothing) -> Nothing"},
		{"window.onKey", "onKey: (handler: Function(String) -> Nothing) -> Nothing"},
		{"name.text", "text: () -> String"},
		{"paid.checked", "checked: () -> Bool"},
		{"canvas.onClick", "onClick: (handler: Function(Real, Real) -> Nothing) -> Nothing"},
		{"canvas.onKey", "onKey: (handler: Function(String) -> Nothing) -> Nothing"},
		{"window.setBackground", "setBackground: (color: String) -> Nothing"},
		{"form.setBackground", "setBackground: (color: String) -> Nothing"},
		{"status.setForeground", "setForeground: (color: String) -> Nothing"},
		{"save.setEnabled", "setEnabled: (enabled: Bool) -> Nothing"},
		{"save.isEnabled", "isEnabled: () -> Bool"},
	} {
		dot := strings.Index(testCase.target, ".")
		hover, ok := store.Hover(path, offsetOf(t, guiSource, testCase.target)+dot+2)
		if !ok || hover.Text != testCase.want {
			t.Fatalf("hover on %s = %q (%v), want %q", testCase.target, hover.Text, ok, testCase.want)
		}
	}
	for _, testCase := range []struct{ call, want string }{
		{"GUI.window(", "(title: String := default, width: Int := default, height: Int := default) -> Window"},
		{"form.checkbox(", "(text: String, checked: Bool := default) -> Checkbox"},
		{"save.onClick(", "(handler: Function() -> Nothing) -> Nothing"},
		{"window.onKey(", "(handler: Function(String) -> Nothing) -> Nothing"},
		{"canvas.onClick(", "(handler: Function(Real, Real) -> Nothing) -> Nothing"},
		{"save.setBackground(", "(color: String) -> Nothing"},
		{"save.setEnabled(", "(enabled: Bool) -> Nothing"},
	} {
		help, ok := store.SignatureHelp(path, offsetOf(t, guiSource, testCase.call)+len(testCase.call))
		if !ok || help.Label != testCase.want {
			t.Fatalf("signature help for %s = %#v (%v), want %q", testCase.call, help, ok, testCase.want)
		}
	}
}
