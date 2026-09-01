package build

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestArchiveStandardLibraryRunsAsNativeExecutable is the required real
// workflow (spec section 29): package a PDF-shaped file, a JSON file, and an
// image into ZIP, TAR, and TAR.GZ, then independently verify every member's
// exact bytes against its source.
func TestArchiveStandardLibraryRunsAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	reportPath := filepath.Join(directory, "report.pdf")
	dataPath := filepath.Join(directory, "data.json")
	imagePath := filepath.Join(directory, "chart.png")
	for path, content := range map[string]string{
		reportPath: "%PDF-fake-report",
		dataPath:   `{"score":91}`,
		imagePath:  "fake-png-bytes",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	zipOutput := filepath.Join(directory, "submission.zip")
	tarOutput := filepath.Join(directory, "submission.tar")
	tarGzipOutput := filepath.Join(directory, "submission.tar.gz")

	source := `bring Archive
files := {
    "report/report.pdf": ` + strconv.Quote(reportPath) + `
    "data/data.json": ` + strconv.Quote(dataPath) + `
    "images/chart.png": ` + strconv.Quote(imagePath) + `
}
Archive.zip(` + strconv.Quote(zipOutput) + `, files)
Archive.tar(` + strconv.Quote(tarOutput) + `, files)
Archive.tarGzip(` + strconv.Quote(tarGzipOutput) + `, files)
write("packaged")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "packaged\n" || stderr != "" {
		t.Fatalf("Archive program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	sources := map[string]string{
		"report/report.pdf": reportPath,
		"data/data.json":    dataPath,
		"images/chart.png":  imagePath,
	}

	zipData, err := os.ReadFile(zipOutput)
	if err != nil {
		t.Fatal(err)
	}
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatal(err)
	}
	if len(zipReader.File) != len(sources) {
		t.Fatalf("zip member count = %d; want %d", len(zipReader.File), len(sources))
	}
	for _, file := range zipReader.File {
		sourcePath, known := sources[file.Name]
		if !known {
			t.Fatalf("unexpected zip member %q", file.Name)
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(content, expected) || sha256.Sum256(content) != sha256.Sum256(expected) {
			t.Fatalf("zip member %q content mismatch", file.Name)
		}
	}

	tarData, err := os.ReadFile(tarOutput)
	if err != nil {
		t.Fatal(err)
	}
	verifyTarMembers(t, bytes.NewReader(tarData), sources)

	tarGzipData, err := os.ReadFile(tarGzipOutput)
	if err != nil {
		t.Fatal(err)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(tarGzipData))
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	verifyTarMembers(t, gzipReader, sources)
}

func verifyTarMembers(t *testing.T, reader io.Reader, sources map[string]string) {
	t.Helper()
	tarReader := tar.NewReader(reader)
	seen := map[string]bool{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		sourcePath, known := sources[header.Name]
		if !known {
			t.Fatalf("unexpected tar member %q", header.Name)
		}
		seen[header.Name] = true
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(content, expected) {
			t.Fatalf("tar member %q content mismatch", header.Name)
		}
	}
	if len(seen) != len(sources) {
		t.Fatalf("tar member count = %d; want %d", len(seen), len(sources))
	}
}

// TestArchiveInvalidUsageRaisesArchiveErrorNotGoOrCompilerDefects checks the
// runtime error paths (traversal, symlink, collision-adjacent, missing
// source) never leak a Go panic or a native build failure.
func TestArchiveInvalidUsageRaisesArchiveErrorNotGoOrCompilerDefects(t *testing.T) {
	directory := t.TempDir()
	present := filepath.Join(directory, "a.txt")
	if err := os.WriteFile(present, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		member string
		source string
	}{
		{"absolute member", "/etc/passwd", present},
		{"traversal member", "../escape.txt", present},
		{"missing source", "a.txt", filepath.Join(directory, "missing.txt")},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "out.zip")
			source := `bring Archive
files := {
    ` + strconv.Quote(testCase.member) + `: ` + strconv.Quote(testCase.source) + `
}
Archive.zip(` + strconv.Quote(output) + `, files)
write("unreachable")
`
			entry := filepath.Join(t.TempDir(), "main.ahd")
			if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := buildAndRun(t, entry, "")
			if code == 0 {
				t.Fatalf("expected failure; stdout=%q stderr=%q", stdout, stderr)
			}
			for _, forbidden := range []string{"BCK005", "panic", "goroutine ", "runtime error"} {
				if strings.Contains(stdout+stderr, forbidden) {
					t.Fatalf("leaked an internal (%q): stdout=%q stderr=%q", forbidden, stdout, stderr)
				}
			}
			if !strings.Contains(stderr, "ArchiveError") {
				t.Fatalf("did not raise ArchiveError: stderr=%q", stderr)
			}
		})
	}
}

// TestPDFAndArchiveModulesAreNotShadowedByASiblingFile compiles a program
// whose own directory holds conflicting PDF.ahd and Archive.ahd files, and
// proves the built-in modules still win, all the way through to a running
// binary -- the same guarantee TestCollectionModulesAreNotShadowedByASiblingFile
// already proves for Lists/KeyValue.
func TestPDFAndArchiveModulesAreNotShadowedByASiblingFile(t *testing.T) {
	directory := t.TempDir()
	files := map[string]string{
		"PDF.ahd":     "new: Function := () -> String {\n    return \"user PDF.new\"\n}\n",
		"Archive.ahd": "zip: Function := (a: String, b: String) -> String {\n    return \"user Archive.zip\"\n}\n",
		"main.ahd": `bring PDF
bring Archive
from PDF bring PDFDocument
doc: PDFDocument := PDF.new()
doc = doc.heading("real builtin PDFDocument", 1)
write("pdf-ok")
`,
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("shadowing program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stdout != "pdf-ok\n" {
		t.Fatalf("a sibling file shadowed a built-in module: %q", stdout)
	}
}

// TestArchiveNativeExecutableRelocatesWithoutSourceOrRuntimeBundle proves
// Archive needs no staged runtime at all: it works the same way Excel/Word
// do (embed-verbatim, no libexec dependency).
func TestArchiveNativeExecutableRelocatesWithoutSourceOrRuntimeBundle(t *testing.T) {
	output := filepath.Join(t.TempDir(), "relocated.zip")
	sourceDirectory := t.TempDir()
	dataPath := filepath.Join(sourceDirectory, "data.txt")
	if err := os.WriteFile(dataPath, []byte("relocated archive data"), 0o600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(sourceDirectory, "main.ahd")
	source := `bring Archive
files := {"data.txt": ` + strconv.Quote(dataPath) + `}
Archive.zip(` + strconv.Quote(output) + `, files)
write("ok")
`
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout := buildRelocateAndRun(t, entry, "archive-relocated")
	if stdout != "ok\n" {
		t.Fatalf("relocated output = %q", stdout)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(reader.File) != 1 {
		t.Fatalf("relocated zip invalid: err=%v", err)
	}
}
