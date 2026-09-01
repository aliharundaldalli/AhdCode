package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestPDFStandardLibraryRunsAsNativeExecutable exercises the full PDFDocument
// surface (heading/paragraph/table/image/pageBreak/save) through a real
// native build and a real offline Tectonic compile, gated exactly like the
// existing Latex runtime tests.
func TestPDFStandardLibraryRunsAsNativeExecutable(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	imageSource := filepath.Join("..", "..", "editors", "vscode", "images", "ahdcode-icon.png")
	imageBytes, err := os.ReadFile(imageSource)
	if err != nil {
		t.Fatalf("read image fixture: %v", err)
	}
	imageCopy := filepath.Join(directory, "icon.png")
	if err := os.WriteFile(imageCopy, imageBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "report.pdf")
	source := `bring PDF
from PDF bring PDFDocument

doc: PDFDocument := PDF.new()
doc = doc.heading("Report Ünïcödé & <tag>", 1)
doc = doc.paragraph("Bold paragraph.", "center", true, false, false)
doc = doc.paragraph("Justified body text with $ & # % _ ^ ~ reserved chars.", "justify", false, false, false)
doc = doc.table(["Name", "Score"], [["Ada", "91"], ["Grace", "88"]], "left")
doc = doc.image(` + strconv.Quote(imageCopy) + `, {"width": 3.0})
doc = doc.pageBreak()
doc = doc.paragraph("Second page.", "left", false, true, true)
doc.save(` + strconv.Quote(output) + `)
write("saved")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "saved\n" || stderr != "" {
		t.Fatalf("PDF program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	content, err := os.ReadFile(output)
	if err != nil || len(content) < 1024 || !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("produced invalid PDF: size=%d err=%v", len(content), err)
	}
}

// TestPDFInvalidDocumentsRaisePDFErrorNotGoOrCompilerDefects checks the
// compile-time-safe (semantic) and runtime error paths do not leak a native
// build failure, a Go panic, or a stack trace.
func TestPDFInvalidDocumentsRaisePDFErrorNotGoOrCompilerDefects(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	cases := []struct {
		name   string
		source string
	}{
		{"invalid heading level", `
doc: PDFDocument := PDF.new()
doc = doc.heading("x", 0)
`},
		{"invalid align", `
doc: PDFDocument := PDF.new()
doc = doc.paragraph("x", "top", false, false, false)
`},
		{"ragged table", `
doc: PDFDocument := PDF.new()
doc = doc.table(["A", "B"], [["1"]], "left")
`},
		{"invalid output extension", `
doc: PDFDocument := PDF.new()
doc = doc.heading("x", 1)
doc.save("out.txt")
`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			entry := filepath.Join(directory, "main.ahd")
			source := "bring PDF\nfrom PDF bring PDFDocument\n" + testCase.source + `write("unreachable")` + "\n"
			if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := buildAndRun(t, entry, "")
			if code == 0 {
				t.Fatalf("expected a nonzero exit; stdout=%q stderr=%q", stdout, stderr)
			}
			for _, forbidden := range []string{"BCK005", "panic", "goroutine ", "runtime error"} {
				if strings.Contains(stdout+stderr, forbidden) {
					t.Fatalf("leaked an internal (%q): stdout=%q stderr=%q", forbidden, stdout, stderr)
				}
			}
			if !strings.Contains(stderr, "PDFError") {
				t.Fatalf("did not raise PDFError: stdout=%q stderr=%q", stdout, stderr)
			}
		})
	}
}

// TestPDFFromWordProducesValidPDFWithoutMutatingSource is the required
// Word->PDF workflow: heading, paragraph, formatted paragraph, table, image,
// page break, converted to PDF, with the source Document verified unchanged
// by continuing to use it afterward.
func TestPDFFromWordProducesValidPDFWithoutMutatingSource(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	imageSource := filepath.Join("..", "..", "editors", "vscode", "images", "ahdcode-icon.png")
	imageBytes, err := os.ReadFile(imageSource)
	if err != nil {
		t.Fatalf("read image fixture: %v", err)
	}
	imageCopy := filepath.Join(directory, "icon.png")
	if err := os.WriteFile(imageCopy, imageBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	docxPath := filepath.Join(directory, "report.docx")
	pdfPath := filepath.Join(directory, "report.pdf")
	source := `bring Word
bring PDF
from Word bring Document
from PDF bring PDFDocument

word: Document := Word.new()
word = word.heading("Report", 1)
word = word.paragraph("Summary paragraph.")
word = word.paragraph("Formatted.", "center", true, true, true)
word = word.table(["A", "B"], [["1", "2"]])
word = word.image(` + strconv.Quote(imageCopy) + `)
word = word.pageBreak()
word.save(` + strconv.Quote(docxPath) + `)

pdf: PDFDocument := PDF.fromWord(word)
pdf.save(` + strconv.Quote(pdfPath) + `)

reloaded: Document := Word.read(` + strconv.Quote(docxPath) + `)
write(reloaded.headings())
write("done")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Word->PDF program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Report") {
		t.Fatalf("Word Document lost its heading after PDF.fromWord: %q", stdout)
	}
	for _, path := range []string{docxPath, pdfPath} {
		content, err := os.ReadFile(path)
		if err != nil || len(content) == 0 {
			t.Fatalf("%s missing or empty: %v", path, err)
		}
	}
	pdfContent, err := os.ReadFile(pdfPath)
	if err != nil || !bytes.HasPrefix(pdfContent, []byte("%PDF-")) {
		t.Fatalf("PDF.fromWord did not produce a valid PDF: %v", err)
	}
}

// TestPDFFromExcelProducesValidPDFWithoutMutatingSource is the required
// Excel->PDF workflow: multiple Sheets, Unicode, every scalar Cell kind,
// merge, Formula, converted to PDF.
func TestPDFFromExcelProducesValidPDFWithoutMutatingSource(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	xlsxPath := filepath.Join(directory, "results.xlsx")
	pdfPath := filepath.Join(directory, "results.pdf")
	source := `bring Excel
bring PDF
from Excel bring Workbook
from Excel bring Sheet
from PDF bring PDFDocument

book: Workbook := Excel.new().addSheet("Öğrenciler").addSheet("Notlar")
sheet: Sheet := book.sheet("Öğrenciler")
sheet = sheet.setCell(1, 1, Excel.fromString("Ad"))
sheet = sheet.setCell(1, 2, Excel.fromString("Puan"))
sheet = sheet.setCell(2, 1, Excel.fromString("Ayşe"))
sheet = sheet.setCell(2, 2, Excel.fromInt(91))
sheet = sheet.setCell(3, 1, Excel.fromString("Can"))
sheet = sheet.setCell(3, 2, Excel.fromReal(88.5))
sheet = sheet.setCell(4, 1, Excel.fromString("Active"))
sheet = sheet.setCell(4, 2, Excel.fromBool(true))
sheet = sheet.setCell(5, 1, Excel.formula("=SUM(B2:B3)"))
book = book.withSheet(sheet)
book.save(` + strconv.Quote(xlsxPath) + `)

pdf: PDFDocument := PDF.fromExcel(book)
pdf.save(` + strconv.Quote(pdfPath) + `)

reloaded: Workbook := Excel.read(` + strconv.Quote(xlsxPath) + `)
write(reloaded.sheets())
write("done")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Excel->PDF program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Öğrenciler") || !strings.Contains(stdout, "Notlar") {
		t.Fatalf("Workbook lost a Sheet after PDF.fromExcel: %q", stdout)
	}
	pdfContent, err := os.ReadFile(pdfPath)
	if err != nil || !bytes.HasPrefix(pdfContent, []byte("%PDF-")) {
		t.Fatalf("PDF.fromExcel did not produce a valid PDF: %v", err)
	}
}

// TestPDFNativeExecutableRelocatesWithoutSourceOrRuntimeBundle proves PDF
// reuses Latex's existing staged-resource relocation contract exactly: a
// native binary built once still finds the same libexec/ahdcode/latex bundle
// when moved and run from an unrelated directory with a scrubbed PATH.
func TestPDFNativeExecutableRelocatesWithoutSourceOrRuntimeBundle(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	// The staged root is only consulted at build time (baked into the
	// generated program's own init file); the relocated executable is run
	// with a scrubbed environment further down, exactly like
	// TestExcelNativeExecutableRelocatesWithoutSourceOrRuntimeBundle.
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	output := filepath.Join(t.TempDir(), "relocated.pdf")
	sourceDirectory := t.TempDir()
	entry := filepath.Join(sourceDirectory, "main.ahd")
	source := `bring PDF
from PDF bring PDFDocument
doc: PDFDocument := PDF.new().heading("Relocated", 1)
doc.save(` + strconv.Quote(output) + `)
write("ok")
`
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout := buildRelocateAndRun(t, entry, "pdf-relocated")
	if stdout != "ok\n" {
		t.Fatalf("relocated output = %q", stdout)
	}
	content, err := os.ReadFile(output)
	if err != nil || !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("relocated PDF invalid: size=%d err=%v", len(content), err)
	}
}
