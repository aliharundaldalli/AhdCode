package build

import (
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
