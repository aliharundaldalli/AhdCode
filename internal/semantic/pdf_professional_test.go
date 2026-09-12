package semantic

import "testing"

func TestPDFProfessionalOperationsValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring PDF
from PDF bring PDFDocument

doc: PDFDocument := PDF.new()
doc = doc.layout("A4")
doc = doc.layout("Custom", true, {"width": 10.0, "height": 6.0}, {"top": 1.0})
doc = doc.header("Left")
doc = doc.header("Left", "Center", "Right")
doc = doc.footer("", "Confidential")
doc = doc.pageNumbers()
doc = doc.pageNumbers("right", true)
doc = doc.qr("https://ahdcode.org")
doc = doc.qr("x", 4.0, "H", "right")
doc = doc.barcode("Code128", "AHD-42")
doc = doc.barcode("EAN13", "590123412345", 6.0, 2.0, "left")
doc = doc.link("Site", "https://ahdcode.org")
doc = doc.link("Mail", "mailto:info@ahdcode.org", "center")
doc = doc.bookmark("Intro")
doc = doc.bookmark("Detail", 2)
doc = doc.metadata("Title")
doc = doc.metadata("Title", "Author", "Subject", ["a", "b"], "AhdCode")
doc = doc.image("logo.svg")
doc = doc.image("logo.svg", {"width": 3.0}, {"rotation": 15.0, "opacity": 0.5, "trimLeft": 0.2})
doc.save("out.pdf")
`)
	requireSemanticClean(t, result)
}

func TestPDFProfessionalOperationsRejectStaticMistakes(t *testing.T) {
	for _, call := range []string{
		`doc.layout()`,
		`doc.layout(4)`,
		`doc.layout("A4", "yes")`,
		`doc.header()`,
		`doc.header(1)`,
		`doc.footer("a", "b", "c", "d")`,
		`doc.pageNumbers("right", "yes")`,
		`doc.qr(42)`,
		`doc.qr("x", "3cm")`,
		`doc.barcode("EAN13")`,
		`doc.link("x")`,
		`doc.bookmark("x", 1.5)`,
		`doc.metadata()`,
		`doc.metadata("t", "a", "s", [1], "c")`,
		`doc.image("a.png", {}, {"opacity": "half"})`,
		// PDF is not Latex: it publishes no way to write renderer source.
		`doc.raw(r"\newpage")`,
		`doc.tex(r"\newpage")`,
		`doc.tikz(r"\draw (0,0) -- (1,1);")`,
		`doc.command("newpage")`,
	} {
		source := "bring PDF\nfrom PDF bring PDFDocument\ndoc: PDFDocument := PDF.new()\nresult := " + call + "\n"
		requireSemanticFailure(t, analyzeWithStandardModules(t, source))
	}
}
