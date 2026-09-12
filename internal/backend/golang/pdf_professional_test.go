package golang

import (
	"strings"
	"testing"
)

func TestPDFProfessionalOperationsLowerToRuntimeCalls(t *testing.T) {
	program := generate(t, `bring PDF
from PDF bring PDFDocument

doc: PDFDocument := PDF.new()
doc = doc.layout("A4", false, {}, {"top": 3.0})
doc = doc.header("Left", "Center", "Right")
doc = doc.footer("Footer")
doc = doc.pageNumbers("right", true)
doc = doc.qr("https://ahdcode.org")
doc = doc.barcode("EAN13", "590123412345", 6.0, 2.0, "left")
doc = doc.link("Site", "https://ahdcode.org")
doc = doc.bookmark("Intro")
doc = doc.metadata("Title", "Author", "Subject", ["a", "b"], "AhdCode")
doc = doc.image("logo.svg", {"width": 3.0}, {"opacity": 0.5})
doc = doc.image("photo.png")
doc.save("out.pdf")
`)
	generated := programSource(t, program)
	for _, call := range []string{
		"AhdPDFLayout(", "AhdPDFRunning(", `"header", "Left"`, `"footer", "Footer", "", "")`,
		"AhdPDFPageNumbers(", `"right", true)`, "AhdPDFQR(", `3.0, "M", "center")`, "AhdPDFBarcode(", `"left")`,
		"AhdPDFLink(", "AhdPDFBookmark(", "int64(1))", "AhdPDFMetadata(", "AhdPDFImageComplete(",
		"AhdBuildPair([]string{}, []float64{}), AhdBuildPair([]string{}, []float64{}))", "AhdPDFSave(",
	} {
		if !strings.Contains(generated, call) {
			t.Fatalf("generated program does not contain %s", call)
		}
	}
	if !program.RequiresLatex || !program.RequiresCodes {
		t.Fatalf("PDFDocument.qr and save need the codes and Latex runtimes: %+v", program)
	}
	plain := generate(t, "bring PDF\nfrom PDF bring PDFDocument\ndoc: PDFDocument := PDF.new().link(\"a\", \"https://ahdcode.org\").bookmark(\"b\")\n")
	if plain.RequiresCodes {
		t.Fatal("a PDF program without qr or barcode received the codes runtime")
	}
}
