package semantic

import "testing"

func TestLatexProfessionalHelpersValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring Latex as L

code: String := L.qr("https://ahdcode.org")
large: String := L.qr(value: "x", size: 4.0, level: "H")
bars: String := L.barcode("EAN13", "590123412345")
wide: String := L.barcode(kind: "Code128", value: "AHD-42", width: 10.0, height: 2.5)
placed: String := L.place(code, 18.5, 2.0)
corner: String := L.place(content: bars, x: 20.0, y: 27.0, anchor: "south east")
header: String := L.header("Left", L.escape("Center & more"), code)
footer: String := L.footer(center: "Page " + L.pageNumber() + " / " + L.pageCount())
link: String := L.link("ahdcode.org", "https://ahdcode.org")
mark: String := L.bookmark("Intro")
deep: String := L.bookmark(title: "Detail", level: 3)
logo: String := L.image("logo.svg", {"width": 4.0}, {"rotation": 15.0, "opacity": 0.5, "trimLeft": 0.2})
figure: String := L.figure("chart.png", "Chart", "fig:c", {}, {"trimBottom": 1.0})
old: String := L.document(link)
source: String := L.document(
    body: header + footer + placed + corner + link + mark + deep + logo + figure + wide + large
    title: "Report"
    paper: "Custom"
    pageSize: {"width": 10.0, "height": 15.0}
    margins: {"top": 1.0, "right": 1.0, "bottom": 1.0, "left": 1.0}
    subject: "Subject"
    keywords: ["one", "two"]
    creator: "AhdCode"
)
positional: String := L.document("Body", "Title", "Author", "", "Report", 2.5, "", "", {}, "Default", true, "A4")
`)
	requireSemanticClean(t, result)
}

func TestLatexProfessionalHelpersRejectStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`L.qr(42)`,
		`L.qr("x", "3cm")`,
		`L.barcode("EAN13")`,
		`L.barcode("EAN13", 590123412345)`,
		`L.place("x")`,
		`L.place("x", "1", "2")`,
		`L.header(1, 2, 3)`,
		`L.pageNumber(1)`,
		`L.link("x")`,
		`L.bookmark("x", 1.5)`,
		`L.image("a.png", {}, {"opacity": "half"})`,
		`L.document(body: "b", paper: 4)`,
		`L.document(body: "b", keywords: [1, 2])`,
		`L.document(body: "b", margins: {"top": "3"})`,
		`L.tcpdf("x")`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, "bring Latex as L\n"+source+"\n"))
	}
}
