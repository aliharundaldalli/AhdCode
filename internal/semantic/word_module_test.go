package semantic

import (
	"strings"
	"testing"
)

const wordPreamble = "bring Word\nfrom Word bring Document\nfrom Word bring WordError\n\n"

func TestWordModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, wordPreamble+`document: Document := Word.new()
document = document.heading("Report", 1)
document = document.paragraph("Summary")
document = document.paragraph("Centered", "center", true, true, true)
document = document.table(["A", "B"], [["1", "2"]])
document = document.table(["A", "B"], [["1", "2"]], [[0, 0, 1, 2]], "center")
document = document.image("chart.png")
document = document.image("chart.png", {"width": 8.0})
document = document.pageBreak()
text: String := document.text()
paragraphs: List<String> := document.paragraphs()
headings: List<String> := document.headings()
tables: List<List<List<String>>> := document.tables()
document.save("report.docx")
loaded: Document := Word.read("report.docx")
`)
	requireSemanticClean(t, result)
}

func TestWordOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`document.heading("Title")`,
		`document.heading("Title", 1, 2)`,
		`document.heading("Title", "one")`,
		`document.paragraph("Text", 1)`,
		`document.table(["A"], [["1"]], "merge")`,
		`document.image("x.png", {"width": "wide"})`,
		`document.pageBreak(1)`,
		`document.save()`,
		`document.text(1)`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, wordPreamble+"document: Document := Word.new()\n"+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestWordDocumentMembersArePositionalOnly(t *testing.T) {
	result := analyzeWithStandardModules(t, wordPreamble+`document: Document := Word.new()
document.paragraph(text: "Summary", bold: true)
`)
	requireSemanticCode(t, result, codeCallArguments)
}

func TestWordDocumentConstructionHintNamesFactories(t *testing.T) {
	result := analyzeWithStandardModules(t, wordPreamble+`document: Document := Document()
`)
	requireSemanticCode(t, result, codeCallArguments)
	found := false
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(diagnostic.Hint, "Word.new()") && strings.Contains(diagnostic.Hint, "Word.read(path)") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Document construction diagnostic omitted the Word factories: %+v", result.Diagnostics)
	}
}

func TestWordHiddenStorageAndUnknownMembersAreRejected(t *testing.T) {
	for _, member := range []string{"blocks", "styles", "describe"} {
		result := analyzeWithStandardModules(t, wordPreamble+
			"document: Document := Word.new()\nwrite(document."+member+")\n")
		requireSemanticFailure(t, result)
	}
}
