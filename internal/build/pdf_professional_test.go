package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"unicode/utf16"
)

// Compiles every v1.3.0 PDFDocument operation with the staged offline runtime
// and checks the PDF itself: outline, link annotation, document properties,
// and vector-only content for the QR symbol, barcode, and SVG logo.
func TestPDFProfessionalDocumentCompilesWithStagedOfflineBundle(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	logo := filepath.Join(directory, "logo.svg")
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 40" width="120" height="40"><rect x="2" y="2" width="116" height="36" rx="8" fill="#1F4E79"/><circle cx="22" cy="20" r="11" fill="#B08D57"/></svg>`
	if err := os.WriteFile(logo, []byte(svg), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "label.pdf")
	source := `bring PDF
from PDF bring PDFDocument

doc: PDFDocument := PDF.new().layout("A5", true, {}, {"top": 1.5, "bottom": 1.5})
doc = doc.header("AhdCode Labs", "", "Product Label & Report")
doc = doc.footer("Confidential").pageNumbers("right", true)
doc = doc.metadata("Product Label", "AhdCode QA", "Professional PDF QA", ["label", "qa"], "AhdCode")
doc = doc.bookmark("Label").heading("Label", 1)
doc = doc.image(` + strconv.Quote(logo) + `, {"height": 1.0}, {"rotation": 5.0, "opacity": 0.8})
doc = doc.qr("https://ahdcode.org/verify?id=42", 3.0, "Q", "center")
doc = doc.barcode("Code128", "AHD-42", 7.0, 1.8, "center")
doc = doc.link("Verify online", "https://ahdcode.org/verify?id=42&lang=tr", "center")
doc = doc.pageBreak().bookmark("Details", 1).bookmark("Specification", 2).paragraph("Second page.", "left", false, false, false)
doc.save(` + strconv.Quote(output) + `)
write("saved")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "saved\n" {
		t.Fatalf("PDF program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	pdf, err := os.ReadFile(output)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("no valid PDF was produced: %v", err)
	}
	objects := inflatedPDF(pdf)
	subject := "/Subject<feff"
	for _, unit := range utf16.Encode([]rune("Professional PDF QA")) {
		subject += fmt.Sprintf("%04x", unit)
	}
	for _, want := range []string{"/Outlines", "/Annots", "/URI(https://ahdcode.org/verify?id=42&lang=tr)", "/D(ahdbookmark.3.1)", subject + ">"} {
		if !bytes.Contains(objects, []byte(want)) {
			t.Fatalf("the PDF lacks %q", want)
		}
	}
	if bytes.Contains(objects, []byte("/Subtype/Image")) || bytes.Contains(objects, []byte("/Subtype /Image")) {
		t.Fatal("the QR symbol, barcode, or SVG logo was rasterized")
	}
	// The outline is exactly the bookmark() calls; the "Label" heading adds
	// no automatic entry of its own.
	if bytes.Contains(objects, []byte("/D(section.")) {
		t.Fatal("a heading added an automatic outline entry")
	}
}
