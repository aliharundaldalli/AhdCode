package build

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestXMLStandardLibraryRunsAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "report.xml")
	source := `bring XML
from XML bring XMLNode
from XML bring XMLDocument
from XML bring XMLError

student: XMLNode := XML.element(
    "student"
    {"id": "42"}
    [
        XML.element("name", {}, [XML.text("Ali")])
        XML.element("score", {}, [XML.text("91")])
    ]
)

document: XMLDocument := XML.document(student)
text: String := XML.stringify(document, true)
write(text)

XML.write(document, ` + strconv.Quote(output) + `, true)
loaded: XMLDocument := XML.read(` + strconv.Quote(output) + `)
root: XMLNode := loaded.root()
write(root.name())
identifier: String? := root.attribute("id")
if identifier != null {
    write(identifier)
}

elements: List<XMLNode> := root.elements()
write(len(elements))
write(elements[0].name())
write(elements[0].text())

mixed: XMLDocument := XML.parse(r'<p>one<b>two</b>three</p>')
mixedRoot: XMLNode := mixed.root()
write(mixedRoot.text())

attempt {
    XML.parse("<a><b></a></b>")
}
except XMLError as error {
    write(error.message)
}
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("XML program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{
		"<student id=\"42\">\n  <name>Ali</name>\n  <score>91</score>\n</student>",
		"student\n", "42\n", "2\n", "name\n", "Ali\n", "onethree\n",
		"element <b> closed by </a>",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("native XML output omitted %q:\n%s", want, stdout)
		}
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `id="42"`) {
		t.Fatalf("written XML file content = %q", string(data))
	}
}
