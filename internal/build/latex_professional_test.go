package build

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
)

var pdfStreamPattern = regexp.MustCompile(`(?s)stream\r?\n(.*?)endstream`)

// inflatedPDF returns the raw PDF followed by every Flate stream it can
// inflate, so dictionaries inside object streams become searchable.
func inflatedPDF(pdf []byte) []byte {
	var result bytes.Buffer
	result.Write(pdf)
	for _, match := range pdfStreamPattern.FindAllSubmatch(pdf, -1) {
		reader, err := zlib.NewReader(bytes.NewReader(match[1]))
		if err != nil {
			continue
		}
		inflated, _ := io.ReadAll(reader)
		result.Write(inflated)
	}
	return result.Bytes()
}

// Compiles the v1.3.0 professional-document helpers with the staged offline
// runtime: a multi-page A4 report with custom margins, an SVG logo and a
// vector QR in the header, a page-count footer with a link, bookmarks, PDF
// properties, an EAN-13, a transformed SVG, and absolute placement. It checks
// the PDF itself, not the generated source.
func TestLatexProfessionalReportCompilesWithStagedOfflineBundle(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	logo := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 40" width="120" height="40"><rect x="2" y="2" width="116" height="36" rx="8" fill="#1F4E79"/><circle cx="22" cy="20" r="11" fill="#B08D57"/></svg>`
	if err := os.WriteFile(filepath.Join(directory, "logo.svg"), []byte(logo), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "report.pdf")
	// The test executable runs in the package directory, so every asset path
	// is absolute.
	logoPath := strconv.Quote(filepath.Join(directory, "logo.svg"))
	codePath := strconv.Quote(filepath.Join(directory, "code.svg"))
	source := `bring Latex as L
bring QR

QR.create("https://ahdcode.org/verify?id=42").saveSVG(` + codePath + `, 3.0)
filler: String := ""
for index in [1, 2, 3, 4, 5, 6, 7, 8, 9] {
    filler += L.section("Section " + str(index)) + "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.\n\n"
}
header: String := L.header(L.image(` + logoPath + `, {"height": 0.8}), L.escape("Report & Summary"), L.qr("https://ahdcode.org", 1.0))
footer: String := L.footer(L.link("ahdcode.org", "https://ahdcode.org/?a=1&b=2#top"), "Page " + L.pageNumber() + " of " + L.pageCount(), "")
body: String := header + footer + L.bookmark("Summary") + L.bookmark("Detail", 2)
body += L.image(` + codePath + `, {"width": 3.0}, {"rotation": 10.0, "opacity": 0.7, "trimLeft": 0.2}) + "\n\n"
body += L.barcode("EAN13", "590123412345") + "\n\n" + filler
body += L.place("Placed", 19.0, 1.0, "north east")
source: String := L.document(
    body: body
    title: "Professional Report"
    author: "AhdCode QA"
    paper: "A4"
    margins: {"top": 3.0, "bottom": 2.5, "left": 2.5, "right": 2.0}
    subject: "Professional QA"
    keywords: ["report", "qa"]
    creator: "AhdCode"
)
L.pdf(source, ` + strconv.Quote(output) + `, "tex")
write("ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" {
		t.Fatalf("report did not compile: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	pdf, err := os.ReadFile(output)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("no valid PDF was produced: %v", err)
	}
	// xdvipdfmx packs dictionaries into compressed object streams, so the
	// checks read the inflated streams.
	objects := inflatedPDF(pdf)
	subject := "/Subject<feff"
	for _, unit := range utf16.Encode([]rune("Professional QA")) {
		subject += fmt.Sprintf("%04x", unit)
	}
	for _, want := range []string{"/Outlines", "/Annots", "/URI(https://ahdcode.org/?a=1&b=2#top)", "/D(ahdbookmark.2.1)", subject + ">"} {
		if !bytes.Contains(objects, []byte(want)) {
			t.Fatalf("the PDF lacks %q", want)
		}
	}
	if bytes.Contains(objects, []byte("/Subtype/Image")) || bytes.Contains(objects, []byte("/Subtype /Image")) {
		t.Fatal("the vector report contains a raster image")
	}
	tex, _ := os.ReadFile(strings.TrimSuffix(output, ".pdf") + ".tex")
	for _, want := range []string{"\\usepackage{fancyhdr}", "\\usepackage{lastpage}", "\\pdfbookmark", "paperwidth=21.0cm"} {
		if !strings.Contains(string(tex), want) && !strings.Contains(string(tex), strings.TrimPrefix(want, "\\")) {
			t.Fatalf("the source sidecar lacks %q", want)
		}
	}
}
