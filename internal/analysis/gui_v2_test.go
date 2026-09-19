package analysis

import (
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/semantic"
	"ahdcode/internal/types"
)

// Every v2.0 GUI and Plot member reaches editors from the compiler's own
// member Symbols: completion lists each published name, and hover and
// signature help render its checked signature.

const guiV2Source = `bring GUI
bring Plot
bring Numeric
window := GUI.window()
window.setResizable(true)
form := window.column()
list := form.listBox(["a", "b"])
choice := form.select(["x", "y"], selectedIndex: 0)
notes := form.textArea(placeholder: "notes")
secret := form.passwordInput()
table := form.table(["Name"], [["Ayşe"]])
list.onChange(lambda (index: Int?, text: String?) -> write(index))
choice.select(null)
table.onSelect(lambda (row: Int?) -> write(row))
notes.onChange(lambda (text: String) -> write(text))
picked := GUI.openFile(extensions: ["csv"])
target := GUI.saveFile(suggestedName: "out.csv")
ok := GUI.confirm("Delete", "Sure?")
surface := Plot.surface([0, 1], [0, 1], Numeric.matrix([[0.0, 1.0], [1.0, 0.0]]))
surface.wireframe(true).save("s.png")
`

func TestCompletionOffersEveryV2Member(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	receivers := map[string]string{
		"ListBox": "form.listBox()", "Select": "form.select([\"a\"])", "TextArea": "form.textArea()",
		"PasswordInput": "form.passwordInput()", "TableView": "form.table([\"a\"])",
	}
	for class, create := range receivers {
		prefix := "bring GUI\nform := GUI.window().column()\nw := " + create + "\nw."
		store.Open(path, prefix+"\n")
		items := store.Completion(path, len(prefix))
		for _, name := range semantic.GUIOperations[class] {
			if !hasLabel(items, name) {
				t.Fatalf("%s completion lacks %q: %#v", class, name, items)
			}
		}
	}
	prefix := "bring Plot\nbring Numeric\ns := Plot.surface([0, 1], [0, 1], Numeric.matrix([[0.0, 0.0], [0.0, 0.0]]))\ns."
	store.Open(path, prefix+"\n")
	items := store.Completion(path, len(prefix))
	for _, name := range semantic.PlotSurfaceOperations {
		if !hasLabel(items, name) {
			t.Fatalf("Surface completion lacks %q", name)
		}
	}
	if hasLabel(items, "rotate") || hasLabel(items, "camera") {
		t.Fatal("Surface offers a camera API")
	}
}

func TestHoverAndSignatureHelpForV2Members(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, guiV2Source)
	for _, testCase := range []struct{ target, want string }{
		{"window.setResizable", "setResizable: (resizable: Bool) -> Nothing"},
		{"form.listBox", "listBox: (items: List<String> := default) -> ListBox"},
		{"form.select", "select: (items: List<String>, selectedIndex: Int? := default) -> Select"},
		{"form.textArea", "textArea: (placeholder: String := default) -> TextArea"},
		{"form.passwordInput", "passwordInput: (placeholder: String := default) -> PasswordInput"},
		{"form.table", "table: (columns: List<String>, rows: List<List<String>> := default) -> TableView"},
		{"list.onChange", "onChange: (handler: Function(Int, String) -> Nothing) -> Nothing"},
		{"choice.select", "select: (index: Int?) -> Nothing"},
		{"table.onSelect", "onSelect: (handler: Function(Int) -> Nothing) -> Nothing"},
		{"notes.onChange", "onChange: (handler: Function(String) -> Nothing) -> Nothing"},
		{"GUI.openFile", "openFile: (title: String := default, extensions: List<String> := default) -> String?"},
		{"GUI.saveFile", "saveFile: (title: String := default, suggestedName: String := default, extensions: List<String> := default) -> String?"},
		{"GUI.confirm", "confirm: (title: String, text: String) -> Bool"},
		{"Plot.surface", "surface"},
		{"surface.wireframe", "wireframe: (enabled: Bool) -> Surface"},
	} {
		dot := strings.Index(testCase.target, ".")
		hover, ok := store.Hover(path, offsetOf(t, guiV2Source, testCase.target)+dot+2)
		if !ok || !strings.HasPrefix(hover.Text, testCase.want) {
			t.Fatalf("hover on %s = %q (%v), want %q", testCase.target, hover.Text, ok, testCase.want)
		}
	}
	for _, testCase := range []struct{ call, want string }{
		{"form.select(", "(items: List<String>, selectedIndex: Int? := default) -> Select"},
		{"list.onChange(", "(handler: Function(Int, String) -> Nothing) -> Nothing"},
		{"GUI.openFile(", "(title: String := default, extensions: List<String> := default) -> String?"},
		{"Plot.surface(", "(x: List<Int>, y: List<Int>, z: Matrix) -> Surface"},
	} {
		help, ok := store.SignatureHelp(path, offsetOf(t, guiV2Source, testCase.call)+len(testCase.call))
		if !ok || help.Label != testCase.want {
			t.Fatalf("signature help for %s = %#v (%v), want %q", testCase.call, help, ok, testCase.want)
		}
	}
}

// TestEveryPublishedGUIMemberHasACheckedSymbol keeps completion aligned
// with the compiler for every GUI and Plot Class.
func TestEveryPublishedGUIMemberHasACheckedSymbol(t *testing.T) {
	for class, names := range semantic.GUIOperations {
		for _, name := range names {
			if !semantic.TypeOperationBindsArguments(semantic.TypeOperation(class + "." + name)) {
				t.Fatalf("GUI %s.%s is listed but has no published Symbol", class, name)
			}
		}
	}
	members := semantic.BuiltinClassMembers(semantic.PlotSurfaceIdentity())
	if len(members) != len(semantic.PlotSurfaceOperations) {
		t.Fatalf("Surface publishes %d of %d members", len(members), len(semantic.PlotSurfaceOperations))
	}
	var _ *types.ClassSymbol = semantic.PlotSurfaceIdentity()
}
