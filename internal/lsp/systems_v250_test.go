package lsp

import (
	"strings"
	"testing"
)

const systemsV250LSPSource = `bring Archive
bring File
bring Process
bring SQLite
from Archive bring ArchiveEntry
from File bring FileEntry
from Process bring ProcessResult
from SQLite bring Database
entry: ArchiveEntry := Archive.list("release.zip")[0]
item: FileEntry := File.walk("site")[0]
result: ProcessResult := Process.run(command: "/bin/echo", args: ["hi"])
db: Database := SQLite.open("panel.db")
`

func TestCompletionOffersTheV250SystemsSurfaceOverTheWire(t *testing.T) {
	cases := []struct {
		prefix string
		labels []string
	}{
		{"Archive.", []string{"list", "extract", "zip", "ArchiveEntry"}},
		{"File.", []string{"copy", "atomicWrite", "atomicMove", "symlink", "readLink", "isSymlink", "permissions", "setPermissions", "walk"}},
		{"Process.", []string{"run", "ProcessResult", "ProcessError"}},
		{"entry.", []string{"path", "kind", "size"}},
		{"item.", []string{"path", "relativePath", "kind", "size", "isSymlink"}},
		{"result.", []string{"exitCode", "stdout", "stderr"}},
		{"db.", []string{"backupTo", "execute", "query", "close"}},
	}
	for _, testCase := range cases {
		source := systemsV250LSPSource + testCase.prefix
		items := completionAt(t, source, "file:///main.ahd", len(source))
		for _, label := range testCase.labels {
			if !hasCompletionLabel(items, label) {
				t.Fatalf("%s offers no %q; got %#v", testCase.prefix, label, items)
			}
		}
	}
	source := "bring "
	if items := completionAt(t, source, "file:///main.ahd", len(source)); !hasCompletionLabel(items, "Process") {
		t.Fatal("bring offers no Process module")
	}
}

func TestHoverAndSignatureHelpForV250SystemsOverTheWire(t *testing.T) {
	source := systemsV250LSPSource + `Process.run(command: "/bin/true")
Archive.extract("a.zip", "out")
db.backupTo("b.db")
code: Int := result.exitCode()
`
	hover := hoverAt(t, source, "file:///main.ahd", "Process.run(command", len("Process.")+1)
	for _, want := range []string{"command: String", "args: List<String>", "timeoutSeconds: Int", "maxOutputBytes: Int", "ProcessResult"} {
		if !strings.Contains(hover.Contents.Value, want) {
			t.Fatalf("Process.run hover lacks %q: %q", want, hover.Contents.Value)
		}
	}
	help, found := signatureHelpAt(t, source, "file:///main.ahd", `Archive.extract("a.zip"`, len("Archive.extract("))
	if !found || len(help.Signatures) == 0 || !strings.Contains(help.Signatures[0].Label, "destination: String") ||
		!strings.Contains(help.Signatures[0].Label, "maxBytes: Int") {
		t.Fatalf("Archive.extract signature help = %+v, found=%v", help, found)
	}
	help, found = signatureHelpAt(t, source, "file:///main.ahd", `db.backupTo("b.db")`, len("db.backupTo("))
	if !found || len(help.Signatures) == 0 || !strings.Contains(help.Signatures[0].Label, "String") {
		t.Fatalf("backupTo signature help = %+v, found=%v", help, found)
	}
	hover = hoverAt(t, source, "file:///main.ahd", "result.exitCode()", len("result.")+1)
	if !strings.Contains(hover.Contents.Value, "exitCode") || !strings.Contains(hover.Contents.Value, "Int") {
		t.Fatalf("exitCode hover = %q", hover.Contents.Value)
	}
}
