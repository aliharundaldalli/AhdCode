package ahdruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pdfTestSquareSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"><rect width="100" height="100" fill="#1F4E79"/></svg>`

func pdfTestFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A document built only from v1.2.0 operations stores the same blocks and
// compiles from exactly the v1.2.0 source.
func TestPDFV12DocumentSourceIsUnchanged(t *testing.T) {
	png := pdfTestFile(t, "icon.png", pdfTestPNGBytes(t, 20, 10))
	doc := AhdPDFHeading(AhdPDFNew(), "Report & 100%", 1)
	doc = AhdPDFParagraph(doc, "Body", "center", true, false, false)
	doc = AhdPDFTable(doc, AhdNewList("A", "B"), AhdNewList(AhdNewList("1", "2")), "right")
	doc = AhdPDFImage(doc, png, AhdBuildPair([]string{"width"}, []float64{4}))
	doc = AhdPDFPageBreak(doc)
	if strings.Contains(doc.Blocks[3], "transform") || doc.Blocks[3] != ahdPDFBlockText(ahdPDFBlock{
		Kind: "image", Media: pdfTestPNGBytes(t, 20, 10), MediaExt: "png", WidthCM: 4, HeightCM: 2,
	}) {
		t.Fatalf("a v1.2.0 image block changed: %s", doc.Blocks[3][:80])
	}
	source := ahdPDFSource(ahdPDFDecodeBlocks(doc), t.TempDir())
	want := "\\documentclass[a4paper]{article}\n\\usepackage{fontspec}\n" +
		"\\setmainfont{lmroman10-regular.otf}[BoldFont=lmroman10-bold.otf,ItalicFont=lmroman10-italic.otf,BoldItalicFont=lmroman10-bolditalic.otf]\n" +
		"\\usepackage{geometry,graphicx,booktabs,array}\n\\geometry{a4paper,margin=2.54cm}\n\\begin{document}\n" +
		"\\section{Report \\& 100\\%}\n\\begin{center}\n\\textbf{Body}\n\\end{center}\n"
	if !strings.HasPrefix(source, want) {
		t.Fatalf("v1.2.0 source changed:\n%s", source)
	}
	if !strings.Contains(source, "\\includegraphics[width=4.0cm,height=2.0cm]{ahdpdf-image-0.png}\n\\clearpage\n\\end{document}\n") {
		t.Fatalf("v1.2.0 image and page break changed:\n%s", source)
	}
	for _, unexpected := range []string{"hyperref", "tikz", "fancyhdr", "lastpage", "ahdbookmark", "ahdimage"} {
		if strings.Contains(source, unexpected) {
			t.Fatalf("a v1.2.0 document loaded %q:\n%s", unexpected, source)
		}
	}
}

func TestPDFProfessionalDocumentSource(t *testing.T) {
	svg := pdfTestFile(t, "logo.svg", []byte(pdfTestSquareSVG))
	doc := AhdPDFLayout(AhdPDFNew(), "Letter", false, nil, nil)
	doc = AhdPDFLayout(doc, "A5", true, AhdBuildPair([]string{}, []float64{}), AhdBuildPair([]string{"top"}, []float64{1.5}))
	doc = AhdPDFRunning(doc, "header", "Left", "Rapor & 100%", "")
	doc = AhdPDFRunning(doc, "footer", "Old", "", "")
	doc = AhdPDFRunning(doc, "footer", "Confidential", "", "")
	doc = AhdPDFPageNumbers(doc, "right", true)
	doc = AhdPDFBookmark(doc, "Giriş", 1)
	doc = AhdPDFHeading(doc, "Intro", 1)
	doc = AhdPDFQR(doc, "https://ahdcode.org", 3, "M", "center")
	doc = AhdPDFBarcode(doc, "EAN13", "590123412345", 6, 2, "left")
	doc = AhdPDFLink(doc, "Site", "https://ahdcode.org/?a=1&b=2", "right")
	doc = AhdPDFImageComplete(doc, svg, AhdBuildPair([]string{"width"}, []float64{4}), AhdBuildPair([]string{"rotation", "opacity"}, []float64{90, 0.5}))
	doc = AhdPDFMetadata(doc, "Rapor & Özet", "Ayşe", "Subject", AhdNewList("pdf", "Türkçe"), "AhdCode")
	directory := t.TempDir()
	source := ahdPDFSource(ahdPDFDecodeBlocks(doc), directory)
	for _, want := range []string{
		"\\documentclass[a4paper]{article}\n",
		"\\geometry{landscape,paperwidth=14.8cm,paperheight=21.0cm,top=1.5cm,right=2.54cm,bottom=2.54cm,left=2.54cm}\n" +
			"\\usepackage{hyperref}\n\\hypersetup{hidelinks,bookmarksdepth=-2}\n" +
			"\\hypersetup{pdftitle={Rapor \\& Özet},pdfauthor={Ayşe},pdfsubject={Subject},pdfkeywords={pdf, Türkçe},pdfcreator={AhdCode}}\n" +
			"\\usepackage{tikz}\n\\usepackage{fancyhdr}\n",
		"\\usepackage{lastpage}\n\\newcounter{ahdbookmark}\n",
		"\\newcommand{\\ahdimageopacity}",
		"\\begin{document}\n%\n% AHDCODE_PAGESTYLE\n\\fancyhead[L]{%\nLeft}%\n\\fancyhead[C]{%\nRapor \\& 100\\%}%\n",
		"\\fancyfoot[L]{%\nConfidential}%\n\\fancyfoot[C]{%\n}%\n\\fancyfoot[R]{%\n\\thepage{} / %\n% AHDCODE_LASTPAGE\n\\pageref*{LastPage}}%\n",
		"\\ahdbookmark{0}{Giriş}%\n\\section{Intro}\n\\begin{center}\n",
		"\\begin{flushleft}\n",
		"\\begin{flushright}\n\\href{https://ahdcode.org/?a=1\\&b=2}{Site}\n\\end{flushright}\n",
		"%\n% AHDCODE_TIKZ\n% AHDCODE_IMAGE\n\\ahdimageopacity{0.5}{\\rotatebox{90.0}{\\resizebox{4.0cm}{4.0cm}{\\input{ahdpdf-image-0-svg.tex}}}}\n\\end{document}\n",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("source lacks %q:\n%s", want, source)
		}
	}
	if strings.Contains(source, "paperwidth=21.59cm") || strings.Contains(source, "Old") {
		t.Fatal("an earlier layout or footer was used instead of the last one")
	}
	if strings.Count(source, "\\fill[black]") != 2 {
		t.Fatalf("expected the QR symbol and the barcode as vector fills:\n%s", source)
	}
	staged, err := os.ReadFile(filepath.Join(directory, "ahdpdf-image-0-svg.tex"))
	if err != nil || !strings.HasPrefix(string(staged), "\\begin{pgfpicture}") {
		t.Fatalf("the SVG was not staged as vector PGF: %v %q", err, staged)
	}
}

func TestPDFRunningTextDefaultsAndOverlap(t *testing.T) {
	header := AhdPDFRunning(AhdPDFNew(), "header", "", "Title", "")
	source := ahdPDFSource(ahdPDFDecodeBlocks(header), t.TempDir())
	if !strings.Contains(source, "\\fancyfoot[C]{\\thepage}\n") || strings.Contains(source, "\\fancyfoot[L]") || strings.Contains(source, "lastpage") {
		t.Fatalf("a header alone must keep the default page number:\n%s", source)
	}
	numbers := AhdPDFPageNumbers(AhdPDFNew(), "left", false)
	source = ahdPDFSource(ahdPDFDecodeBlocks(numbers), t.TempDir())
	if !strings.Contains(source, "\\fancyfoot[L]{%\n\\thepage{}}%\n\\fancyfoot[C]{%\n}%\n") {
		t.Fatalf("pageNumbers left:\n%s", source)
	}
	overlap := AhdPDFPageNumbers(AhdPDFRunning(AhdPDFNew(), "footer", "", "Center", ""), "center", false)
	expectRaise(t, AhdClassPDFError, func() { ahdPDFSource(ahdPDFDecodeBlocks(overlap), t.TempDir()) })
	if _, problem := AhdPDFRunningBlock("footer", "", "", ""); problem != "" {
		t.Fatalf("an empty footer is valid: %s", problem)
	}
}

func TestPDFProfessionalValidationMessages(t *testing.T) {
	gif := pdfTestFile(t, "a.gif", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"))
	png := pdfTestFile(t, "a.png", pdfTestPNGBytes(t, 10, 10))
	svg := pdfTestFile(t, "a.svg", []byte(pdfTestSquareSVG))
	text := pdfTestFile(t, "text.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><text>x</text></svg>`))
	exact := []struct {
		name    string
		problem string
		want    string
	}{
		{"paper", second(AhdPDFLayoutBlock("B5", false, nil, nil, nil, nil)), `PDFDocument.layout paper must be A3, A4, A5, Letter, Legal, or Custom; received "B5"`},
		{"custom", second(AhdPDFLayoutBlock("Custom", false, nil, nil, nil, nil)), `PDFDocument.layout paper "Custom" requires pageSize with both width and height`},
		{"margin", second(AhdPDFLayoutBlock("A4", false, nil, nil, []string{"top"}, []float64{0})), "PDFDocument.layout margins top must be greater than 0 and at most 1000 centimeters"},
		{"header line", second(AhdPDFRunningBlock("header", "a\nb", "", "")), "PDFDocument.header text must be a single line"},
		{"footer line", second(AhdPDFRunningBlock("footer", "", "", "a\r")), "PDFDocument.footer text must be a single line"},
		{"numbers", second(AhdPDFPageNumbersBlock("top", false)), `PDFDocument.pageNumbers align must be left, center, or right; received "top"`},
		{"qr align", second(AhdPDFQRBlock("x", 3, "M", "middle")), `PDFDocument.qr align must be left, center, or right; received "middle"`},
		{"qr value", second(AhdPDFQRBlock("", 3, "M", "center")), "PDFDocument.qr value must not be empty"},
		{"barcode kind", second(AhdPDFBarcodeBlock("Code39", "X", 8, 2, "center")), `PDFDocument.barcode kind must be Code128, EAN13, or UPCA; received "Code39"`},
		{"link url", second(AhdPDFLinkBlock("x", "ftp://ahdcode.org", "left")), `PDFDocument.link url must start with https://, http://, or mailto:; received "ftp://ahdcode.org"`},
		{"link script", second(AhdPDFLinkBlock("x", "javascript:alert(1)", "left")), `PDFDocument.link url must start with https://, http://, or mailto:; received "javascript:alert(1)"`},
		{"link align", second(AhdPDFLinkBlock("x", "https://ahdcode.org", "justify")), `PDFDocument.link align must be left, center, or right; received "justify"`},
		{"link text", second(AhdPDFLinkBlock("", "https://ahdcode.org", "left")), "PDFDocument.link text must not be empty"},
		{"bookmark", second(AhdPDFBookmarkBlock("x", 5)), "PDFDocument.bookmark level must be between 1 and 4; received 5"},
		{"keywords", second(AhdPDFMetadataBlock("t", "", "", []string{"a,b"}, "")), "PDFDocument.metadata keywords must be non-empty and must not contain commas"},
		{"metadata line", second(AhdPDFMetadataBlock("t\nx", "", "", nil, "")), "PDFDocument.metadata values must be single lines"},
		{"image path", second(AhdPDFImageBlock("", nil, nil, nil, nil)), "image path must not be empty"},
		{"image format", second(AhdPDFImageBlock(gif, nil, nil, nil, nil)), "unsupported image format: PDF supports PNG, JPEG, and SVG"},
		{"image size", second(AhdPDFImageBlock(png, []string{"depth"}, []float64{1}, nil, nil)), "image size supports only width and height"},
		{"image width", second(AhdPDFImageBlock(png, []string{"width"}, []float64{0}, nil, nil)), "image width must be positive"},
		{"opacity", second(AhdPDFImageBlock(png, nil, nil, []string{"opacity"}, []float64{2})), "PDFDocument.image transform opacity must be between 0.0 and 1.0; received 2.0"},
		{"transform key", second(AhdPDFImageBlock(svg, nil, nil, []string{"scale"}, []float64{2})), `PDFDocument.image transform supports only rotation, opacity, trimLeft, trimTop, trimRight, and trimBottom; received "scale"`},
		{"trim", second(AhdPDFImageBlock(png, []string{"height"}, []float64{2}, []string{"trimTop", "trimBottom"}, []float64{1, 1})), "PDFDocument.image transform trimTop and trimBottom remove the whole 2.0 cm height"},
	}
	for _, test := range exact {
		if test.problem != test.want {
			t.Fatalf("%s\n got %q\nwant %q", test.name, test.problem, test.want)
		}
	}
	for name, problem := range map[string]string{
		"missing image":  second(AhdPDFImageBlock(filepath.Join(t.TempDir(), "missing.png"), nil, nil, nil, nil)),
		"svg text":       second(AhdPDFImageBlock(text, nil, nil, nil, nil)),
		"svg trim":       second(AhdPDFImageBlock(svg, nil, nil, []string{"trimLeft", "trimRight"}, []float64{2, 2})),
		"barcode digits": second(AhdPDFBarcodeBlock("EAN13", "123", 8, 2, "center")),
	} {
		if problem == "" {
			t.Fatalf("%s was accepted", name)
		}
	}
	// Native entry points raise the builders' problems as PDFError.
	expectRaise(t, AhdClassPDFError, func() { AhdPDFLink(AhdPDFNew(), "x", "file:///etc/passwd", "left") })
	expectRaise(t, AhdClassPDFError, func() { AhdPDFLayout(AhdPDFNew(), "Tabloid", false, nil, nil) })
	expectRaise(t, AhdClassPDFError, func() { AhdPDFImageComplete(AhdPDFNew(), text, nil, nil) })
	expectRaise(t, AhdClassPDFError, func() { AhdPDFMetadata(AhdPDFNew(), "", "", "", AhdNewList(""), "") })
}

func second(_ string, problem string) string { return problem }
