package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestPDFDocumentInThePersistentREPL exercises PDF.new/heading/paragraph/
// table construction (pure, no Tectonic needed) through the persistent
// evaluator, matching the same block-building contract native builds use.
// save() cannot invoke the offline renderer interactively, so it must raise
// PDFError with a clear message rather than hang, crash, or silently no-op --
// the same contract Latex.pdf/pdfFile already have in the REPL.
func TestPDFDocumentInThePersistentREPL(t *testing.T) {
	program := `bring PDF
from PDF bring PDFDocument
doc: PDFDocument := PDF.new()
doc = doc.heading("Report", 1)
doc = doc.paragraph("Body text.", "center", true, false, false)
doc = doc.table(["A", "B"], [["1", "2"]], "left")
write("built")
doc.save("unused.pdf")
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(program), &output, &errorOutput, "AhdCode v0.1.19")
	if !strings.Contains(output.String(), "built") {
		t.Fatalf("REPL output missing %q:\n%s", "built", output.String())
	}
	if !strings.Contains(errorOutput.String(), "PDFError") {
		t.Fatalf("expected PDFError from save() in the REPL: %s", errorOutput.String())
	}
}

// TestPDFFromWordAndFromExcelInThePersistentREPL checks the pure conversion
// path (no Tectonic involved) works identically to native builds.
func TestPDFFromWordAndFromExcelInThePersistentREPL(t *testing.T) {
	program := `bring Word
bring Excel
bring PDF
from Word bring Document
from Excel bring Workbook
from PDF bring PDFDocument

word: Document := Word.new()
word = word.heading("From Word", 1)
wordPDF: PDFDocument := PDF.fromWord(word)
write("word-ok")

book: Workbook := Excel.new().addSheet("S")
excelPDF: PDFDocument := PDF.fromExcel(book)
write("excel-ok")
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(program), &output, &errorOutput, "AhdCode v0.1.19")
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL errors: %s", errorOutput.String())
	}
	for _, want := range []string{"word-ok", "excel-ok"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("REPL output omitted %q:\n%s", want, output.String())
		}
	}
}

// TestArchiveInThePersistentREPL checks Archive.zip/tar/tarGzip run for real
// in the REPL (unlike PDF.save/Latex.pdf, Archive needs no external process),
// producing the same bytes a native build would.
func TestArchiveInThePersistentREPL(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "a.txt")
	if err := os.WriteFile(sourcePath, []byte("repl archive data"), 0o600); err != nil {
		t.Fatal(err)
	}
	zipOutput := filepath.Join(directory, "out.zip")
	tarOutput := filepath.Join(directory, "out.tar")
	tarGzipOutput := filepath.Join(directory, "out.tar.gz")
	program := `bring Archive
files := {"a.txt": ` + strconv.Quote(sourcePath) + `}
Archive.zip(` + strconv.Quote(zipOutput) + `, files)
Archive.tar(` + strconv.Quote(tarOutput) + `, files)
Archive.tarGzip(` + strconv.Quote(tarGzipOutput) + `, files)
write("archived")
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(program), &output, &errorOutput, "AhdCode v0.1.19")
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL errors: %s", errorOutput.String())
	}
	if !strings.Contains(output.String(), "archived") {
		t.Fatalf("REPL output missing %q:\n%s", "archived", output.String())
	}
	for _, path := range []string{zipOutput, tarOutput, tarGzipOutput} {
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			t.Fatalf("%s missing or empty: %v", path, err)
		}
	}
}
