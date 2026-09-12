package repl

import (
	"bytes"
	"strings"
	"testing"
)

// The v1.3.0 PDFDocument operations build documents in the persistent REPL,
// report invalid arguments as catchable PDFError values, and still leave
// compilation to native programs.
func TestPDFProfessionalOperationsInPersistentREPL(t *testing.T) {
	program := `bring PDF
from PDF bring PDFDocument
from PDF bring PDFError
doc: PDFDocument := PDF.new().layout("A5", true).header("Report").footer("", "Confidential").pageNumbers("right", true)
doc = doc.qr("https://ahdcode.org").barcode("EAN13", "590123412345").link("Site", "https://ahdcode.org").bookmark("Intro")
doc = doc.metadata("Title", "Author", "Subject", ["a"], "AhdCode")
write("built")
attempt {
    doc = doc.link("Bad", "javascript:alert(1)")
}
except PDFError as problem {
    write("caught " + problem.message)
}
doc.pageNumbers("top")
doc.save("unused.pdf")
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(program), &output, &errors, "AhdCode v1.3.0")
	for _, want := range []string{"built\n", `caught PDFDocument.link url must start with https://, http://, or mailto:; received "javascript:alert(1)"`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("REPL output lacks %q:\n%s\nerrors:\n%s", want, output.String(), errors.String())
		}
	}
	for _, want := range []string{`PDFDocument.pageNumbers align must be left, center, or right; received "top"`, "PDF compilation is not available in the interactive evaluator"} {
		if !strings.Contains(errors.String(), want) {
			t.Fatalf("REPL errors lack %q:\n%s", want, errors.String())
		}
	}
}
