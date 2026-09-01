package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestPDFArchiveLatexWordExcelJSONCoexistInOneProgram brings every module the
// v0.1.20 release touches (plus the standard-library modules the task's
// coexistence check names) into one program and exercises at least
// PDF/Archive/Latex/Word/Excel/JSON, proving no canonical module identity
// collision exists between them.
func TestPDFArchiveLatexWordExcelJSONCoexistInOneProgram(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	pdfOutput := filepath.Join(directory, "article.pdf")
	zipOutput := filepath.Join(directory, "package.zip")
	notePath := filepath.Join(directory, "note.txt")

	source := `bring PDF
bring Archive
bring Latex as L
bring Word
bring Excel
bring JSON
bring Lists
bring KeyValue
bring Env
bring Data
bring Numeric
bring Statistics
bring File
bring Path
from PDF bring PDFDocument
from Word bring Document
from Excel bring Workbook

document: String := L.document(body: L.section("Coexistence"), title: "Coexistence")
L.pdf(document, ` + strconv.Quote(pdfOutput) + `)

word: Document := Word.new().heading("Word coexists", 1)
excel: Workbook := Excel.new().addSheet("S")
pdf: PDFDocument := PDF.fromWord(word)
pdf = PDF.fromExcel(excel)

write(JSON.fromBool(true).bool())
write(Lists.flatten([[1, 2], [3]]))
write(KeyValue.keys({"a": 1}))

File.writeText(` + strconv.Quote(notePath) + `, "hello")
files := {"note.txt": ` + strconv.Quote(notePath) + `}
Archive.zip(` + strconv.Quote(zipOutput) + `, files)

write("coexist-ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("coexistence program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "coexist-ok") {
		t.Fatalf("coexistence program did not complete: %q", stdout)
	}
	pdfContent, err := os.ReadFile(pdfOutput)
	if err != nil || !bytes.HasPrefix(pdfContent, []byte("%PDF-")) {
		t.Fatalf("Latex.pdf did not produce a valid PDF: %v", err)
	}
	if info, err := os.Stat(zipOutput); err != nil || info.Size() == 0 {
		t.Fatalf("Archive.zip did not produce output: %v", err)
	}
}
