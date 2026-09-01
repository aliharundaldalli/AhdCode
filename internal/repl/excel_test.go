package repl

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestExcelWorkbookInPersistentREPL(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "repl.xlsx")
	program := `bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring CellStyle
book: Workbook := Excel.new()
book = book.addSheet("Data")
sheet: Sheet := book.sheet("Data")
sheet = sheet.setCell(1, 1, Excel.fromString("Name"))
sheet = sheet.setCell(2, 1, Excel.fromString("Ali"))
sheet = sheet.setCell(2, 2, Excel.formula("=1+1"))
header: CellStyle := Excel.style().bold(true).fillColor("#1F4E79")
sheet = sheet.style(sheet.range(1, 1, 1, 2), header)
book = book.withSheet(sheet)
write(book.sheetCount())
write(sheet.range(1, 1, 2, 2).address())
write(sheet.cell(2, 2).formula())
book.save(` + strconv.Quote(outputPath) + `)
loaded: Workbook := Excel.read(` + strconv.Quote(outputPath) + `)
write(loaded.sheet("Data").cell(2, 1).string())
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(program), &output, &errorOutput, "AhdCode v0.1.19")
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL errors: %s", errorOutput.String())
	}
	for _, want := range []string{"1", "A1:B2", "=1+1", "Ali"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("REPL output omitted %q:\n%s", want, output.String())
		}
	}
}
