package build

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLatexHelpersRunThroughNativeBackend(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring Latex as L

body: String := L.section("Türkçe & Matematik")
body += L.equation("\\sum_\{k=1\}^\{n\} k")
source: String := L.document(body: body, title: "AhdCode", author: "Ali")
write(L.escape("ç ğ ı İ & _"))
write(source)
`})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("Latex helper program failed: code=%d stderr=%q", code, stderr)
	}
	for _, expected := range []string{"ç ğ ı İ \\& \\_", "lmroman10-regular.otf", "\\section{Türkçe \\& Matematik}", "\\sum_{k=1}^{n} k"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("Latex helper output omitted %q:\n%s", expected, stdout)
		}
	}
}

func TestLatexBundledRuntimeCompilesSupportedOfflineProfile(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := filepath.Join(t.TempDir(), "Latex space Türkçe")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	imageSource := filepath.Join("..", "..", "editors", "vscode", "images", "ahdcode-icon.png")
	imageBytes, err := os.ReadFile(imageSource)
	if err != nil {
		t.Fatalf("read image fixture: %v", err)
	}
	imagePath := filepath.Join(directory, "ahd icon.png")
	if err := os.WriteFile(imagePath, imageBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	fileSource := `\documentclass{article}
\usepackage{graphicx}
\begin{document}
\includegraphics[width=2cm]{ahd icon.png}
\end{document}
`
	texPath := filepath.Join(directory, "image source.tex")
	if err := os.WriteFile(texPath, []byte(fileSource), 0o600); err != nil {
		t.Fatal(err)
	}
	generatedPDF := filepath.Join(directory, "generated output.pdf")
	imagePDF := filepath.Join(directory, "image output.pdf")
	main := `bring Latex as L

body: String := L.section("Türkçe karakter testi: ç ğ ı İ ö ş ü Ç Ğ Ö Ş Ü")
body += L.equation("\\sum_\{k=1\}^\{n\} k = \\frac\{n(n+1)\}\{2\}")
body += """
\\[
\\int_0^1 x^2\\,dx = \\frac\{1\}\{3\}
\\qquad
\\begin\{pmatrix\}1&2\\\\3&4\\end\{pmatrix\}
\\]
\\begin\{align\}
a+b &= c \\\\
d-e &= f
\\end\{align\}
"""
body += L.table(["Ad", "Değer"], [["Ali", "1"]])
source: String := L.document(body: body, title: "AhdCode", author: "Ali")
L.pdf(source: source, output: ` + strconv.Quote(generatedPDF) + `)
L.pdfFile(input: ` + strconv.Quote(texPath) + `, output: ` + strconv.Quote(imagePDF) + `)
write("ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(main), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(t.TempDir(), "program")
	path, result := BuildProgram(entry, executable)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if path != executable {
		t.Fatalf("built path = %s", path)
	}
	command := exec.Command(executable)
	command.Dir = directory
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "PATH=") {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "PATH=/usr/bin:/bin")
	output, err := command.CombinedOutput()
	if err != nil || string(output) != "ok\n" {
		t.Fatalf("offline Latex run failed: %v\n%s", err, output)
	}
	for _, path := range []string{generatedPDF, imagePDF} {
		content, err := os.ReadFile(path)
		if err != nil || len(content) < 5 || string(content[:5]) != "%PDF-" {
			t.Fatalf("invalid PDF %s: size=%d err=%v", path, len(content), err)
		}
	}
}

func TestLatexRealEngineFailureIsCatchable(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to exercise the real engine")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.ahd")
	text := `bring Latex
from Latex bring LatexError

attempt {
    Latex.pdf("\\documentclass\{article\}\\begin\{document\}\\notACommand\\end\{document\}", "bad.pdf")
}
except LatexError as error {
    write("caught")
}
`
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "caught\n" || stderr != "" {
		t.Fatalf("LatexError handling = stdout %q stderr %q code %d", stdout, stderr, code)
	}
}

// TestLatexScriptScriptMathResolvesPhysicalFonts is the regression test for the
// bundled font-closure defect: the minimal bundle shipped cmsy7.tfm without
// cmsy7.pfb, so an ordinary mathematics document reached xdvipdfmx and failed
// with "Cannot proceed without .vf or physical font for PDF output".
//
// The mathematics below is taken from the document that exposed the defect. Its
// nested fractions inside \left|...\right| push symbols down to the
// script-script size, which is what requests the smaller physical font.
func TestLatexScriptScriptMathResolvesPhysicalFonts(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	output := filepath.Join(directory, "series.pdf")
	entry := filepath.Join(directory, "main.ahd")
	text := `bring Latex as L

body: String := L.section("Sonlu Geometrik Seriler")
body += L.escape("r gerçek bir sayı ve r ≠ 1 olmak üzere:")
body += "\n"
body += L.equation("S_n = \\sum_\{k=0\}^\{n\} r^k")
body += L.equation("S_n = \\frac\{1-r^\{n+1\}\}\{1-r\}")
body += L.equation("\\sum_\{k=0\}^\{\\infty\} r^k = \\frac\{1\}\{1-r\}, \\qquad \\left|\\frac\{1\}\{1-r\} - S_n\\right| = \\frac\{|r|^\{n+1\}\}\{|1-r|\}")
body += L.table(
    ["n", "S_n", "2 - S_n"],
    [
        ["2", "1.750000", "0.250000"],
        ["12", "1.999756", "0.000244"]
    ]
)

source: String := L.document(body: body, title: "Matematiksel Kısa Rapor", author: "Ali Harun Daldallı")
L.pdf(source: source, output: ` + strconv.Quote(output) + `)
write("ok")
`
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" {
		t.Fatalf("script-script mathematics failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	content, err := os.ReadFile(output)
	if err != nil || len(content) < 1024 || string(content[:5]) != "%PDF-" {
		t.Fatalf("invalid PDF: size=%d err=%v", len(content), err)
	}
}

// TestLatexBundleShipsAPhysicalFontForEveryMetric keeps the closure defect from
// returning: a metric whose Type 1 outline is absent from the bundle is exactly
// what made xdvipdfmx fail. lcircle10, lcirclew10, and pzdr are excluded on
// purpose — the first two are MetaFont-only picture fonts and pzdr maps to a
// standard PDF font, so neither has a Type 1 outline to ship.
func TestLatexBundleShipsAPhysicalFontForEveryMetric(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "tooling", "latex", "resources.json"))
	if err != nil {
		t.Fatal(err)
	}
	var resources []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(content, &resources); err != nil {
		t.Fatal(err)
	}
	outlines := make(map[string]bool)
	var metrics []string
	for _, item := range resources {
		switch {
		case strings.HasSuffix(item.Name, ".pfb"):
			outlines[strings.TrimSuffix(item.Name, ".pfb")] = true
		case strings.HasSuffix(item.Name, ".tfm"):
			metrics = append(metrics, strings.TrimSuffix(item.Name, ".tfm"))
		}
	}
	withoutOutline := map[string]bool{"lcircle10": true, "lcirclew10": true, "pzdr": true}
	for _, metric := range metrics {
		if !outlines[metric] && !withoutOutline[metric] {
			t.Fatalf("metric %s.tfm has no physical font %s.pfb in the bundle", metric, metric)
		}
	}
	if len(metrics) == 0 {
		t.Fatal("resource manifest lists no font metrics")
	}
}

// TestLatexMathColumnTableCompilesToPDF is the real-engine regression for
// mathematical table cells. The table mirrors the bigeometric-derivative
// document that exposed the gap: two math columns beside ordinary Turkish text.
func TestLatexMathColumnTableCompilesToPDF(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	output := filepath.Join(directory, "bigeometrik.pdf")
	entry := filepath.Join(directory, "main.ahd")
	text := `bring Latex as L

body: String := L.section("Bigeometrik türev")
body += L.table(
    ["Fonksiyon", "Bigeometrik türev", "Yorum"],
    [
        ["g(x)=x^a", "e^a", "İlk türev sabittir"],
        ["g(x)=e^\{cx\}", "e^\{cx\}", "Fonksiyon sabit noktadır"],
        ["g(x)=C", "1", "Pozitif sabitlerin türevi birdir"],
        ["g(x)=e^\{a(\\ln x)^m\}", "e^\{am(\\ln x)^\{m-1\}\}", "Logaritmik & polinom ailesi"]
    ],
    [0, 1]
)

source: String := L.document(body: body, title: "Bigeometrik", author: "Ali")
L.pdf(source: source, output: ` + strconv.Quote(output) + `)
write("ok")
`
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" {
		t.Fatalf("math-column table failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	content, err := os.ReadFile(output)
	if err != nil || len(content) < 1024 || string(content[:5]) != "%PDF-" {
		t.Fatalf("invalid PDF: size=%d err=%v", len(content), err)
	}
}
