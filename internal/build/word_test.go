package build

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWordStandardLibraryRunsAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "native.docx")
	source := `bring Word
from Word bring Document
from Word bring WordError

base: Document := Word.new()
headers: List<String> := ["A", "B"]
rows: List<List<String>> := [["1", "2"], ["3", "4"]]
merges: List<List<Int>> := [[0, 0, 1, 2], [1, 0, 2, 1]]
document: Document := base.heading("Report", 1)
document = document.paragraph("Summary", "center", true, true, true)
document = document.table(headers, rows, merges, "center")
document = document.pageBreak()
headers[0] = "changed"
rows[0][0] = "changed"
merges[0][3] = 1
write(base.text() == "")
write(document.text())
write(document.headings())
write(document.paragraphs())
write(document.tables())
document.save(` + strconv.Quote(output) + `)
loaded: Document := Word.read(` + strconv.Quote(output) + `)
write(loaded.text())
write(loaded.tables())
resaved: String := ` + strconv.Quote(filepath.Join(directory, "resaved.docx")) + `
loaded.save(resaved)
reloaded: Document := Word.read(resaved)
write(reloaded.tables())
attempt {
    base.table(["A"], [["1"]], [[0, 0, 1, 1]])
}
except WordError as error {
    write(error.message)
}
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Word program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{
		"true\n", "Report\nSummary\n", `["Report"]`, `["Summary"]`,
		`[[["A", "B"], ["1", "2"], ["3", "4"]]]`,
		// The header's horizontal merge (gridSpan) and the first column's
		// vertical merge (vMerge) mean Word.read() recovers this table as
		// ["A", ""] / ["1", "2"] / ["", "4"]: every logical column survives,
		// including the "2" and "4" cells a pre-fix reader would have
		// silently dropped by capping every row at the 1-cell merged header.
		`[[["A", ""], ["1", "2"], ["", "4"]]]`,
		"a 1x1 table merge is meaningless\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("native Word output omitted %q:\n%s", want, stdout)
		}
	}
	data, err := os.ReadFile(output)
	if err != nil || len(data) < 100 || !bytes.HasPrefix(data, []byte("PK")) {
		t.Fatalf("native Word output is not a DOCX: size=%d err=%v", len(data), err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var documentXML string
	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		var content bytes.Buffer
		if _, err := content.ReadFrom(opened); err != nil {
			t.Fatal(err)
		}
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
		documentXML = content.String()
	}
	for _, want := range []string{`<w:gridSpan w:val="2"/>`, `<w:vMerge w:val="restart"/>`, `<w:vMerge/>`, `<w:br w:type="page"/>`} {
		if !strings.Contains(documentXML, want) {
			t.Fatalf("native document.xml omitted %q:\n%s", want, documentXML)
		}
	}
}
