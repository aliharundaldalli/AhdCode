package golang

import (
	"strings"
	"testing"
)

func TestLatexProfessionalHelpersLowerToRuntimeCalls(t *testing.T) {
	program := generate(t, `bring Latex as L

footer: String := L.footer(L.link("site", "https://ahdcode.org"), "Page " + L.pageNumber() + " / " + L.pageCount(), "")
header: String := L.header(left: L.image("logo.svg", {"height": 0.8}, {"opacity": 0.9}), right: L.qr("https://ahdcode.org", 1.2))
body: String := header + footer + L.bookmark("Intro") + L.bookmark("Detail", 2)
body += L.barcode("Code128", "AHD-42") + L.place("Top", 1.0, 1.0) + L.place("Corner", 20.0, 28.0, "south east")
body += L.figure("chart.png", "Chart", "fig:c", {"width": 12.0}, {"trimTop": 0.5})
source: String := L.document(body: body, paper: "A4", pageSize: {}, margins: {"top": 3.0}, subject: "S", keywords: ["a"], creator: "C")
L.pdf(source, "out.pdf")
`)
	generated := programSource(t, program)
	for _, call := range []string{
		"AhdLatexFooter(", "AhdLatexLink(", "AhdLatexPageNumberText()", "AhdLatexPageCountText()", "AhdLatexHeader(",
		"AhdLatexImageComplete(", "AhdLatexQR(", `"M")`, "AhdLatexBookmark(", "int64(1)", "AhdLatexBarcode(", "8.0, 2.0)",
		"AhdLatexPlace(", `"north west"`, `"south east"`, "AhdLatexFigureComplete(", "AhdLatexDocumentComplete(",
	} {
		if !strings.Contains(generated, call) {
			t.Fatalf("generated program does not contain %s", call)
		}
	}
	if !program.RequiresLatex || !program.RequiresCodes {
		t.Fatalf("Latex.qr and Latex.barcode need both the Latex runtime and the codes runtime: %+v", program)
	}
	plain := generate(t, "bring Latex as L\nwrite(L.document(L.header(\"x\", \"\", \"\") + L.link(\"a\", \"https://ahdcode.org\")))\n")
	if plain.RequiresCodes {
		t.Fatal("a Latex program without qr or barcode received the codes runtime")
	}
}
