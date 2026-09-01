package semantic

import "testing"

const excelPreamble = `bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell
from Excel bring Range
from Excel bring CellStyle
from Excel bring ExcelError

`

func TestExcelModuleValidTypedSurface(t *testing.T) {
	result := analyzeWithStandardModules(t, excelPreamble+`book: Workbook := Excel.new()
book = book.addSheet("Data")
sheet: Sheet := book.sheet("Data")
blank: Cell := Excel.blank()
text: Cell := Excel.fromString("Name")
integer: Cell := Excel.fromInt(91)
realCell: Cell := Excel.fromReal(91.5)
boolean: Cell := Excel.fromBool(true)
formula: Cell := Excel.formula("=SUM(A1:A3)")
sheet = sheet.setCell(1, 1, text)
sheet = sheet.setRow(2, 1, [integer, realCell, boolean, formula])
area: Range := sheet.range(1, 1, 3, 4)
sheet = sheet.setRange(area, [[blank, blank, blank, blank], [blank, blank, blank, blank], [blank, blank, blank, blank]])
grid: List<List<Cell>> := sheet.cells(area)
used: Range? := sheet.usedRange()
sheet = sheet.merge(sheet.range(1, 1, 1, 2))
merges: List<Range> := sheet.merges()
style: CellStyle := Excel.style().bold(true).italic(false).underline(true).fontSize(12.0)
style = style.textColor("#FFFFFF").fillColor("#1F4E79").horizontal("center").vertical("top")
style = style.wrap(true).numberFormat("0.00").border("thin", "#000000")
sheet = sheet.style(area, style).columnWidth(1, 20.0).rowHeight(1, 24.0)
book = book.withSheet(sheet)
names: List<String> := book.sheets()
count: Int := book.sheetCount()
book.save("book.xlsx")
loaded: Workbook := Excel.read("book.xlsx")
write(area.startRow())
write(area.startColumn())
write(area.endRow())
write(area.endColumn())
write(area.rowCount())
write(area.columnCount())
write(area.address())
write(text.kind())
write(text.isBlank())
write(text.string())
write(integer.int())
write(integer.real())
write(realCell.real())
write(boolean.bool())
write(formula.formula())
`)
	requireSemanticClean(t, result)
}

func TestExcelRejectsScalarCellsAndWrongCollectionShapesStatically(t *testing.T) {
	for _, source := range []string{
		`sheet = sheet.setCell(1, 1, 91)`,
		`sheet = sheet.setRow(1, 1, [1, 2])`,
		`sheet = sheet.setRange(sheet.range(1, 1, 1, 1), [["x"]])`,
		`sheet = sheet.style(sheet.range(1, 1, 1, 1), "bold")`,
		`sheet.cell(1)`,
		`Excel.fromInt("91")`,
	} {
		result := analyzeWithStandardModules(t, excelPreamble+`book: Workbook := Excel.new().addSheet("Data")
sheet: Sheet := book.sheet("Data")
`+source+"\n")
		requireSemanticFailure(t, result)
	}
}

func TestExcelMembersArePositionalOnly(t *testing.T) {
	result := analyzeWithStandardModules(t, excelPreamble+`book: Workbook := Excel.new().addSheet("Data")
sheet: Sheet := book.sheet("Data")
sheet.setCell(row: 1, column: 1, value: Excel.blank())
`)
	requireSemanticCode(t, result, codeCallArguments)
}
