package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestLatexPDFTwoArgumentCallIsUnchanged is the backward-compatibility
// regression for the existing published contract: an ordinary two-argument
// Latex.pdf(source, output) call must lower and behave exactly as before --
// no sidecar .tex may appear.
func TestLatexPDFTwoArgumentCallIsUnchanged(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	output := filepath.Join(directory, "old.pdf")
	source := `bring Latex as L
document: String := L.document(body: L.section("Old call"), title: "Old")
L.pdf(document, ` + strconv.Quote(output) + `)
write("ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" || stderr != "" {
		t.Fatalf("2-arg pdf() failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	content, err := os.ReadFile(output)
	if err != nil || !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("2-arg pdf() did not produce a valid PDF: %v", err)
	}
	if _, err := os.Stat(strings.TrimSuffix(output, ".pdf") + ".tex"); !os.IsNotExist(err) {
		t.Fatalf("2-arg pdf() unexpectedly produced a .tex sidecar (err=%v)", err)
	}
}

// TestLatexPDFExplicitEmptySourceOutputMatchesTwoArgumentForm checks the
// explicit third argument "" behaves identically to omitting it.
func TestLatexPDFExplicitEmptySourceOutputMatchesTwoArgumentForm(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	output := filepath.Join(directory, "explicit.pdf")
	source := `bring Latex as L
document: String := L.document(body: L.section("Explicit empty"), title: "Explicit")
L.pdf(document, ` + strconv.Quote(output) + `, "")
write("ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" || stderr != "" {
		t.Fatalf(`pdf(..., "") failed: code=%d stdout=%q stderr=%q`, code, stdout, stderr)
	}
	if _, err := os.Stat(strings.TrimSuffix(output, ".pdf") + ".tex"); !os.IsNotExist(err) {
		t.Fatalf(`pdf(..., "") unexpectedly produced a .tex sidecar (err=%v)`, err)
	}
}

// TestLatexPDFTexModeProducesExactSourceSidecar is the canonical "tex" mode
// contract: the PDF publishes, and the .tex sibling contains the caller's
// exact source bytes -- not a transformed temporary source, not compiler
// metadata, not a temporary path.
func TestLatexPDFTexModeProducesExactSourceSidecar(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	pdfOutput := filepath.Join(directory, "article.pdf")
	texOutput := filepath.Join(directory, "article.tex")
	source := `bring Latex as L
document: String := L.document(body: L.section("Tex sidecar Türkçe & <tag>"), title: "Sidecar")
L.pdf(document, ` + strconv.Quote(pdfOutput) + `, "tex")
write(document)
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf(`pdf(..., "tex") failed: code=%d stderr=%q`, code, stderr)
	}
	expectedSource := strings.TrimSuffix(stdout, "\n")

	pdfContent, err := os.ReadFile(pdfOutput)
	if err != nil || !bytes.HasPrefix(pdfContent, []byte("%PDF-")) {
		t.Fatalf("did not produce a valid PDF: %v", err)
	}
	texContent, err := os.ReadFile(texOutput)
	if err != nil {
		t.Fatalf("did not produce a .tex sidecar: %v", err)
	}
	if string(texContent) != expectedSource {
		t.Fatalf(".tex sidecar bytes did not match the exact source:\ngot:  %q\nwant: %q", texContent, expectedSource)
	}
	for _, forbidden := range []string{directory, os.TempDir(), "ahdcode-latex-source-"} {
		if strings.Contains(string(texContent), forbidden) {
			t.Fatalf(".tex sidecar leaked a temporary/absolute path: contains %q", forbidden)
		}
	}
}

// TestLatexPDFFolderSiblingPathDerivation checks the sibling .tex path is
// derived by replacing the trailing .pdf, including for a nested folder.
func TestLatexPDFFolderSiblingPathDerivation(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	sub := filepath.Join(directory, "folder")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	pdfOutput := filepath.Join(sub, "report.pdf")
	source := `bring Latex as L
document: String := L.document(body: L.section("Nested"), title: "Nested")
L.pdf(document, ` + strconv.Quote(pdfOutput) + `, "tex")
write("ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" || stderr != "" {
		t.Fatalf("failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(sub, "report.tex")); err != nil {
		t.Fatalf("sibling .tex was not derived correctly: %v", err)
	}
}

// TestLatexPDFInvalidSourceOutputModeRaisesLatexError checks an arbitrary
// third-argument String is rejected rather than silently accepted, and that
// the mode comparison is case-sensitive.
func TestLatexPDFInvalidSourceOutputModeRaisesLatexError(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	for _, mode := range []string{"yes", "TEX", "Tex", "pdf+tex"} {
		t.Run(mode, func(t *testing.T) {
			directory := t.TempDir()
			output := filepath.Join(directory, "x.pdf")
			source := `bring Latex as L
document: String := L.document(body: L.section("x"), title: "x")
L.pdf(document, ` + strconv.Quote(output) + `, ` + strconv.Quote(mode) + `)
write("unreachable")
`
			entry := filepath.Join(directory, "main.ahd")
			if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := buildAndRun(t, entry, "")
			if code == 0 {
				t.Fatalf("expected failure for mode %q; stdout=%q stderr=%q", mode, stdout, stderr)
			}
			if !strings.Contains(stderr, "LatexError") {
				t.Fatalf("mode %q did not raise LatexError: stderr=%q", mode, stderr)
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("mode %q published a PDF despite being invalid", mode)
			}
		})
	}
}

// TestLatexPDFTexModeRequiresPDFOutputExtension checks the sibling-derivation
// precondition is enforced before any compilation is attempted.
func TestLatexPDFTexModeRequiresPDFOutputExtension(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	output := filepath.Join(directory, "x.output")
	source := `bring Latex as L
document: String := L.document(body: L.section("x"), title: "x")
L.pdf(document, ` + strconv.Quote(output) + `, "tex")
write("unreachable")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code == 0 {
		t.Fatalf("expected failure; stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(stderr, "LatexError") {
		t.Fatalf("did not raise LatexError: stderr=%q", stderr)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("published an output despite the invalid extension")
	}
}

// TestLatexPDFCompileFailurePublishesNeitherOutput is the dual-output
// atomicity regression: a failing compile must not create a new PDF or a new
// .tex sidecar, and must not disturb an already-existing valid PDF.
func TestLatexPDFCompileFailurePublishesNeitherOutput(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	output := filepath.Join(directory, "existing.pdf")
	texOutput := filepath.Join(directory, "existing.tex")
	if err := os.WriteFile(output, []byte("%PDF-existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := `bring Latex as L
L.pdf("\\documentclass\{article\}\\begin\{document\}\\undefinedcommandthatdoesnotexist\\end\{document\}", ` + strconv.Quote(output) + `, "tex")
write("unreachable")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code == 0 {
		t.Fatalf("expected a compile failure; stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(stderr, "LatexError") {
		t.Fatalf("did not raise LatexError: stderr=%q", stderr)
	}
	content, err := os.ReadFile(output)
	if err != nil || string(content) != "%PDF-existing" {
		t.Fatalf("a failed compile changed the existing PDF destination: %q, %v", content, err)
	}
	if _, err := os.Stat(texOutput); !os.IsNotExist(err) {
		t.Fatalf("a failed compile published a .tex sidecar anyway (err=%v)", err)
	}
}

// TestLatexPDFFileStillTakesTwoArguments is the pdfFile regression: it must
// remain untouched by the pdf() extension.
func TestLatexPDFFileStillTakesTwoArguments(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	input := filepath.Join(directory, "existing.tex")
	if err := os.WriteFile(input, []byte(`\documentclass{article}\begin{document}pdfFile still works\end{document}`), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "existing.pdf")
	source := `bring Latex as L
L.pdfFile(` + strconv.Quote(input) + `, ` + strconv.Quote(output) + `)
write("ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" || stderr != "" {
		t.Fatalf("pdfFile regression failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	content, err := os.ReadFile(output)
	if err != nil || !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("pdfFile did not produce a valid PDF: %v", err)
	}
}
