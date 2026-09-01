package build

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestExcelStandardLibraryRunsAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "University Results.xlsx")
	source := `bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell
from Excel bring Range
from Excel bring CellStyle
from Excel bring ExcelError

base: Workbook := Excel.new()
book: Workbook := base.addSheet("Students").addSheet("Statistics").addSheet("Configuration")
sheet: Sheet := book.sheet("Students")
sheet = sheet.setCell(1, 1, Excel.fromString("Üniversite Sonuçları"))
sheet = sheet.setCell(1, 2, Excel.fromString("Do not lose me"))
title: Range := sheet.range(1, 1, 1, 5)
attempt { sheet.merge(title) } except ExcelError as error { write(error.message) }
write(sheet.cell(1, 2).string())
sheet = sheet.setCell(1, 2, Excel.blank())
sheet = sheet.merge(title)
sheet = sheet.setRow(2, 1, [Excel.fromString("Name"), Excel.fromString("Score"), Excel.fromString("Average"), Excel.fromString("Active"), Excel.fromString("Formula")])
sheet = sheet.setRow(3, 1, [Excel.fromString("Ali"), Excel.fromInt(91), Excel.fromReal(91.5), Excel.fromBool(true), Excel.formula("=SUM(B3:C3)")])
header: CellStyle := Excel.style().bold(true).fillColor("#1F4E79").textColor("#FFFFFF").horizontal("center").vertical("center").wrap(true).border("thin", "#000000")
sheet = sheet.style(sheet.range(1, 1, 2, 5), header)
sheet = sheet.style(sheet.range(3, 2, 3, 3), Excel.style().numberFormat("0.00"))
sheet = sheet.columnWidth(1, 24.0).columnWidth(2, 12.0).rowHeight(1, 28.0)
attempt { sheet.setRange(sheet.range(5, 1, 6, 3), [[Excel.blank(), Excel.blank(), Excel.blank()], [Excel.blank(), Excel.blank()]]) } except ExcelError as error { write(error.message) }
book = book.withSheet(sheet)
write(base.sheetCount())
write(book.sheets())
used: Range? := sheet.usedRange()
if used != null { write(used.address()) }
literal: Cell := Excel.fromString("=SUM(A1:A100)")
write(literal.kind())
book.save(` + strconv.Quote(output) + `)
loaded: Workbook := Excel.read(` + strconv.Quote(output) + `)
loadedSheet: Sheet := loaded.sheet("Students")
write(loadedSheet.cell(3, 1).string())
write(loadedSheet.cell(3, 2).int())
write(loadedSheet.cell(3, 2).real())
write(loadedSheet.cell(3, 3).kind())
write(loadedSheet.cell(3, 4).bool())
write(loadedSheet.cell(3, 5).formula())
loaded.save(` + strconv.Quote(filepath.Join(directory, "roundtrip.xlsx")) + `)
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Excel program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{
		"would lose its value\n", "Do not lose me\n", "requires 3 cells; received 2\n", "0\n",
		`["Students", "Statistics", "Configuration"]`, "A1:E3\n", "String\n", "Ali\n", "91\n", "91.0\n", "Real\n", "true\n", "=SUM(B3:C3)\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("native Excel output omitted %q:\n%s", want, stdout)
		}
	}
	data, err := os.ReadFile(output)
	if err != nil || !bytes.HasPrefix(data, []byte("PK")) {
		t.Fatalf("output is not XLSX: size=%d err=%v", len(data), err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	required := map[string]bool{"[Content_Types].xml": false, "_rels/.rels": false, "xl/workbook.xml": false, "xl/_rels/workbook.xml.rels": false, "xl/styles.xml": false, "xl/worksheets/sheet1.xml": false}
	for _, file := range reader.File {
		if _, known := required[file.Name]; !known {
			continue
		}
		required[file.Name] = true
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(opened)
		for {
			_, err = decoder.Token()
			if err != nil {
				break
			}
		}
		_ = opened.Close()
		if !errors.Is(err, io.EOF) {
			t.Fatalf("%s is not well-formed XML: %v", file.Name, err)
		}
	}
	for name, present := range required {
		if !present {
			t.Fatalf("XLSX omitted %s", name)
		}
	}
}

// TestExcelStyleFontSizeIntArgumentWidensThroughEveryStage is the regression
// for the v0.1.19 QA release blocker where CellStyle.fontSize(0) (an Int
// literal into a Real parameter) passed semantic analysis under AhdCode's
// normal safe Int -> Real widening rule, but the native Go backend emitted
// int64(0) into a call expecting float64, so `ahdcode build` failed with a
// BCK005 code-generation defect instead of producing a program that raises
// the intended runtime ExcelError. It covers the full pipeline: semantic
// analysis, the Go backend, and a real native build and run.
func TestExcelStyleFontSizeIntArgumentWidensThroughEveryStage(t *testing.T) {
	t.Run("Int and Real literals widen identically", func(t *testing.T) {
		directory := t.TempDir()
		output := filepath.Join(directory, "fontsize.xlsx")
		source := `bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring CellStyle

book: Workbook := Excel.new().addSheet("S")
sheet: Sheet := book.sheet("S")
sheet = sheet.setCell(1, 1, Excel.fromString("a"))
sheet = sheet.setCell(2, 1, Excel.fromString("b"))
intStyle: CellStyle := Excel.style().fontSize(12)
realStyle: CellStyle := Excel.style().fontSize(12.0)
sheet = sheet.style(sheet.range(1, 1, 1, 1), intStyle)
sheet = sheet.style(sheet.range(2, 1, 2, 1), realStyle)
book = book.withSheet(sheet)
book.save(` + strconv.Quote(output) + `)
write("styled ok")
`
		entry := filepath.Join(directory, "main.ahd")
		if err := os.WriteFile(entry, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := buildAndRun(t, entry, "")
		if code != 0 || stderr != "" {
			t.Fatalf("fontSize(Int) program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		if stdout != "styled ok\n" {
			t.Fatalf("stdout = %q", stdout)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatal(err)
		}
		var stylesXML []byte
		for _, file := range reader.File {
			if file.Name != "xl/styles.xml" {
				continue
			}
			opened, openErr := file.Open()
			if openErr != nil {
				t.Fatal(openErr)
			}
			stylesXML, err = io.ReadAll(opened)
			_ = opened.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
		if stylesXML == nil {
			t.Fatal("xl/styles.xml is missing")
		}
		// fontSize(12) (Int, widened to Real) and fontSize(12.0) (Real) must
		// produce byte-identical CellStyle representations, so the style
		// catalog collapses them into the same single custom <font>: the
		// base Calibri font plus exactly one "12" font, never two.
		if count := bytes.Count(stylesXML, []byte("<font>")); count != 2 {
			t.Fatalf("expected exactly 2 <font> elements (base + one deduplicated custom font); got %d:\n%s", count, stylesXML)
		}
		if !bytes.Contains(stylesXML, []byte(`<sz val="12"/>`)) {
			t.Fatalf("styles.xml missing the widened font size:\n%s", stylesXML)
		}
	})

	for _, invalid := range []string{"0", "-5"} {
		t.Run("fontSize("+invalid+") builds natively and fails at runtime", func(t *testing.T) {
			directory := t.TempDir()
			source := `bring Excel
from Excel bring CellStyle

style: CellStyle := Excel.style()
style = style.fontSize(` + invalid + `)
write("unreachable")
`
			entry := filepath.Join(directory, "main.ahd")
			if err := os.WriteFile(entry, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := buildAndRun(t, entry, "")
			if code == 0 {
				t.Fatalf("expected a nonzero exit for fontSize(%s); stdout=%q stderr=%q", invalid, stdout, stderr)
			}
			for _, forbidden := range []string{"BCK005", "code generation defect", "panic", "goroutine "} {
				if strings.Contains(stdout+stderr, forbidden) {
					t.Fatalf("fontSize(%s) surfaced a compiler/runtime internal (%q) instead of a controlled ExcelError:\nstdout=%s\nstderr=%s", invalid, forbidden, stdout, stderr)
				}
			}
			if !strings.Contains(stderr, "ExcelError") || !strings.Contains(stderr, "fontSize") {
				t.Fatalf("fontSize(%s) did not raise the intended ExcelError: stdout=%q stderr=%q", invalid, stdout, stderr)
			}
		})
	}
}

// TestNumericVectorScaleIntArgumentStillWidens is a non-Excel regression for
// an existing compiler-supplied TypeOperation with a Real parameter
// (Vector.scale), guarding against the fontSize fix regressing the older
// single-expected-type lowering path every other TypeOperation module still
// uses.
func TestNumericVectorScaleIntArgumentStillWidens(t *testing.T) {
	source := `bring Numeric
from Numeric bring Vector

v: Vector := Numeric.vector([1, 2, 3])
scaled: Vector := v.scale(2)
write(scaled.values())
`
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Vector.scale(Int) program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stdout != "[2.0, 4.0, 6.0]\n" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestExcelNativeExecutableRelocatesWithoutSourceOrRuntimeBundle(t *testing.T) {
	output := filepath.Join(t.TempDir(), "relocated.xlsx")
	sourceDirectory := t.TempDir()
	entry := filepath.Join(sourceDirectory, "main.ahd")
	source := `bring Excel
from Excel bring Workbook
from Excel bring Sheet
book: Workbook := Excel.new().addSheet("Data")
sheet: Sheet := book.sheet("Data").setCell(1, 1, Excel.fromString("relocated"))
book = book.withSheet(sheet)
book.save(` + strconv.Quote(output) + `)
write(Excel.read(` + strconv.Quote(output) + `).sheet("Data").cell(1, 1).string())
`
	if err := os.WriteFile(entry, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	stdout := buildRelocateAndRun(t, entry, "excel-relocated")
	if stdout != "relocated\n" {
		t.Fatalf("relocated output = %q", stdout)
	}
	data, err := os.ReadFile(output)
	if err != nil || !bytes.HasPrefix(data, []byte("PK")) {
		t.Fatalf("relocated XLSX invalid: size=%d err=%v", len(data), err)
	}
}
