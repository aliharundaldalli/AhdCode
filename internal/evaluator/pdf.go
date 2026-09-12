package evaluator

import (
	"encoding/json"
	"strconv"
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The PDF standard module's REPL implementation. heading, paragraph, table,
// and pageBreak mirror the native ahdruntime/pdf.go block encoding on the
// evaluator's own List/Pair/Instance representation; image and every v1.3.0
// operation call the shared runtime block builders, so the evaluator stores
// the same blocks and reports the same messages a compiled program does.
// save() cannot invoke the offline Tectonic renderer interactively, so it
// raises PDFError just like Latex.pdf/pdfFile already do in this same
// evaluator.

const pdfDocumentClassID = ir.ClassID("builtin:PDF::class::PDFDocument")

var pdfBlocksField = ir.FieldID(string(pdfDocumentClassID) + "::field::blocks")

// pdfBlock is the private content-block shape of the v1.2.0 blocks the
// evaluator still encodes itself, identical to the native ahdPDFBlock's
// fields of the same names.
type pdfBlock struct {
	Kind      string     `json:"kind"`
	Text      string     `json:"text,omitempty"`
	Level     int        `json:"level,omitempty"`
	Align     string     `json:"align,omitempty"`
	Bold      bool       `json:"bold,omitempty"`
	Italic    bool       `json:"italic,omitempty"`
	Underline bool       `json:"underline,omitempty"`
	Headers   []string   `json:"headers,omitempty"`
	Rows      [][]string `json:"rows,omitempty"`
	Media     []byte     `json:"media,omitempty"`
	MediaExt  string     `json:"mediaExt,omitempty"`
	WidthCM   float64    `json:"widthCM,omitempty"`
	HeightCM  float64    `json:"heightCM,omitempty"`
}

// pdfDocumentValue materializes a block list as a new PDFDocument instance.
func (s *Session) pdfDocumentValue(blocks []pdfBlock) *Instance {
	items := make([]any, len(blocks))
	for index, block := range blocks {
		encoded, _ := json.Marshal(block)
		items[index] = string(encoded)
	}
	return &Instance{Class: pdfDocumentClassID, Fields: map[ir.FieldID]any{
		pdfBlocksField: &List{Items: items},
	}}
}

func (s *Session) pdfAppend(value any, block pdfBlock) *Instance {
	encoded, _ := json.Marshal(block)
	return s.pdfAppendText(value, string(encoded), "")
}

// pdfAppendText appends one encoded block, or raises the problem the shared
// runtime builder reported. Stored blocks are copied as they are, so fields
// this evaluator's pdfBlock does not name survive every later append.
func (s *Session) pdfAppendText(value any, text, problem string) *Instance {
	instance := s.requireInstance(value)
	stored, ok := instance.Fields[pdfBlocksField].(*List)
	if !ok {
		s.raise("PDFError", "value is not a PDFDocument")
	}
	if problem != "" {
		s.raise("PDFError", problem)
	}
	items := append(append([]any(nil), stored.Items...), text)
	return &Instance{Class: pdfDocumentClassID, Fields: map[ir.FieldID]any{pdfBlocksField: &List{Items: items}}}
}

// pdfAppendTo returns an appender for one runtime builder's (block, problem)
// result.
func (s *Session) pdfAppendTo(value any) func(text, problem string) *Instance {
	return func(text, problem string) *Instance { return s.pdfAppendText(value, text, problem) }
}

// pdfStrings reads a List<String> argument, or an empty list when omitted.
func (s *Session) pdfStrings(args []any, index int) []string {
	if index >= len(args) || args[index] == nil {
		return []string{}
	}
	list := s.requireList(args[index])
	values := make([]string, len(list.Items))
	for position, item := range list.Items {
		values[position] = item.(string)
	}
	return values
}

func (s *Session) pdfBuiltin(name string, args []any) any {
	switch name {
	case "new":
		return s.pdfDocumentValue(nil)
	case "fromWord":
		return s.pdfDocumentValue(s.pdfBlocksFromWord(args[0]))
	case "fromExcel":
		return s.pdfDocumentValue(s.pdfBlocksFromExcel(args[0]))
	}
	s.raise("Error", "unsupported PDF function "+name)
	return nil
}

var pdfParagraphAlignments = map[string]bool{"left": true, "center": true, "right": true, "justify": true}
var pdfTableAlignments = map[string]bool{"left": true, "center": true, "right": true}

func (s *Session) pdfOperation(name string, receiver any, args []any) any {
	arg := func(index int, fallback any) any {
		if index < len(args) && args[index] != nil {
			return args[index]
		}
		return fallback
	}
	switch name {
	case "PDFDocument.heading":
		level := arg(1, int64(0)).(int64)
		if level < 1 || level > 6 {
			s.raise("PDFError", "heading level must be between 1 and 6")
		}
		return s.pdfAppend(receiver, pdfBlock{Kind: "heading", Text: arg(0, "").(string), Level: int(level)})
	case "PDFDocument.paragraph":
		align := arg(1, "left").(string)
		if !pdfParagraphAlignments[align] {
			s.raise("PDFError", "paragraph align must be left, center, right, or justify")
		}
		return s.pdfAppend(receiver, pdfBlock{
			Kind: "paragraph", Text: arg(0, "").(string), Align: align,
			Bold: arg(2, false).(bool), Italic: arg(3, false).(bool), Underline: arg(4, false).(bool),
		})
	case "PDFDocument.table":
		return s.pdfTable(receiver, args)
	case "PDFDocument.image":
		sizeKeys, sizeValues := s.latexRealEntries(pairArg(args, 1))
		transformKeys, transformValues := s.latexRealEntries(pairArg(args, 2))
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFImageBlock(arg(0, "").(string), sizeKeys, sizeValues, transformKeys, transformValues))
	case "PDFDocument.layout":
		sizeKeys, sizeValues := s.latexRealEntries(pairArg(args, 2))
		marginKeys, marginValues := s.latexRealEntries(pairArg(args, 3))
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFLayoutBlock(arg(0, "").(string), arg(1, false).(bool),
			sizeKeys, sizeValues, marginKeys, marginValues))
	case "PDFDocument.header", "PDFDocument.footer":
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFRunningBlock(strings.TrimPrefix(name, "PDFDocument."),
			arg(0, "").(string), arg(1, "").(string), arg(2, "").(string)))
	case "PDFDocument.pageNumbers":
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFPageNumbersBlock(arg(0, "center").(string), arg(1, false).(bool)))
	case "PDFDocument.qr":
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFQRBlock(arg(0, "").(string), latexReal(args, 1, 3.0),
			arg(2, "M").(string), arg(3, "center").(string)))
	case "PDFDocument.barcode":
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFBarcodeBlock(arg(0, "").(string), arg(1, "").(string),
			latexReal(args, 2, 8.0), latexReal(args, 3, 2.0), arg(4, "center").(string)))
	case "PDFDocument.link":
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFLinkBlock(arg(0, "").(string), arg(1, "").(string), arg(2, "left").(string)))
	case "PDFDocument.bookmark":
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFBookmarkBlock(arg(0, "").(string), arg(1, int64(1)).(int64)))
	case "PDFDocument.metadata":
		return s.pdfAppendTo(receiver)(ahdruntime.AhdPDFMetadataBlock(arg(0, "").(string), arg(1, "").(string),
			arg(2, "").(string), s.pdfStrings(args, 3), arg(4, "").(string)))
	case "PDFDocument.pageBreak":
		return s.pdfAppend(receiver, pdfBlock{Kind: "pageBreak"})
	case "PDFDocument.save":
		s.raise("PDFError", "PDF compilation is not available in the interactive evaluator")
		return Nothing
	}
	s.raise("Error", "unsupported PDFDocument operation "+name)
	return nil
}

func (s *Session) pdfTable(receiver any, args []any) any {
	headers := s.requireList(args[0])
	rows := s.requireList(args[1])
	align := "left"
	if len(args) > 2 && args[2] != nil {
		align = args[2].(string)
	}
	if !pdfTableAlignments[align] {
		s.raise("PDFError", "table align must be left, center, or right")
	}
	headerValues := make([]string, len(headers.Items))
	for index, value := range headers.Items {
		headerValues[index] = value.(string)
	}
	if len(headerValues) == 0 {
		s.raise("PDFError", "table requires at least one column")
	}
	grid := make([][]string, len(rows.Items))
	for index, item := range rows.Items {
		row := s.requireList(item)
		if len(row.Items) != len(headerValues) {
			s.raise("PDFError", "table row column count does not match headers")
		}
		cells := make([]string, len(row.Items))
		for position, value := range row.Items {
			cells[position] = value.(string)
		}
		grid[index] = cells
	}
	return s.pdfAppend(receiver, pdfBlock{Kind: "table", Headers: headerValues, Rows: grid, Align: align})
}

// pdfBlocksFromWord converts a Word Document's own blocks directly into PDF
// blocks, reusing Word's existing evaluator block reader. See
// ahdruntime.AhdPDFFromWord for the exact same mapping used natively.
func (s *Session) pdfBlocksFromWord(document any) []pdfBlock {
	wordBlocks := s.documentBlocks(document)
	blocks := make([]pdfBlock, 0, len(wordBlocks))
	for _, block := range wordBlocks {
		switch block.Kind {
		case "heading":
			blocks = append(blocks, pdfBlock{Kind: "heading", Text: block.Text, Level: block.Level})
		case "paragraph":
			blocks = append(blocks, pdfBlock{
				Kind: "paragraph", Text: block.Text, Align: block.Align,
				Bold: block.Bold, Italic: block.Italic, Underline: block.Underline,
			})
		case "table":
			blocks = append(blocks, pdfBlock{Kind: "table", Headers: block.Headers, Rows: block.Rows, Align: block.Align})
		case "image":
			blocks = append(blocks, pdfBlock{
				Kind: "image", Media: block.Media, MediaExt: block.MediaExt,
				WidthCM:  float64(block.WidthEMU) / wordEMUPerCentimeter,
				HeightCM: float64(block.HeightEMU) / wordEMUPerCentimeter,
			})
		case "pageBreak":
			blocks = append(blocks, pdfBlock{Kind: "pageBreak"})
		}
	}
	return blocks
}

const pdfMaxExcelColumns = 10

// pdfBlocksFromExcel is a semantic tabular export of a Workbook, reading it
// through the same public Excel runtime accessors Excel's own evaluator
// operations already use (session.excelData plus the exported
// ahdruntime.Excel* functions), rather than duplicating Excel's internal
// storage format. See ahdruntime.AhdPDFFromExcel for the exact same policy
// (used-range table per Sheet, formula source text, no fabricated merge
// spanning, a documented column-count limit instead of silent truncation).
func (s *Session) pdfBlocksFromExcel(workbook any) []pdfBlock {
	workbookText := s.excelData(workbook, evaluatorExcelWorkbookClass)
	sheetNames, err := ahdruntime.ExcelWorkbookSheets(workbookText)
	if err != nil {
		s.raise("PDFError", err.Error())
	}
	if len(sheetNames) == 0 {
		s.raise("PDFError", "PDF.fromExcel requires a Workbook with at least one Sheet")
	}
	var blocks []pdfBlock
	for _, name := range sheetNames {
		blocks = append(blocks, pdfBlock{Kind: "heading", Text: name, Level: 1})
		sheetText, err := ahdruntime.ExcelWorkbookSheet(workbookText, name)
		if err != nil {
			s.raise("PDFError", err.Error())
		}
		usedRangeText, err := ahdruntime.ExcelSheetUsedRange(sheetText)
		if err != nil {
			s.raise("PDFError", err.Error())
		}
		if usedRangeText == nil {
			continue
		}
		startRow, _ := ahdruntime.ExcelRangeInt(*usedRangeText, "startRow")
		startColumn, _ := ahdruntime.ExcelRangeInt(*usedRangeText, "startColumn")
		endRow, _ := ahdruntime.ExcelRangeInt(*usedRangeText, "endRow")
		endColumn, _ := ahdruntime.ExcelRangeInt(*usedRangeText, "endColumn")
		columnCount := int(endColumn - startColumn + 1)
		if columnCount > pdfMaxExcelColumns {
			s.raise("PDFError", "PDF.fromExcel: Sheet "+strconv.Quote(name)+" has "+strconv.Itoa(columnCount)+
				" used columns, which exceeds the supported limit of "+strconv.Itoa(pdfMaxExcelColumns))
		}
		var headers []string
		var rows [][]string
		for row := startRow; row <= endRow; row++ {
			line := make([]string, 0, columnCount)
			for column := startColumn; column <= endColumn; column++ {
				cellText, err := ahdruntime.ExcelSheetCell(sheetText, row, column)
				if err != nil {
					s.raise("PDFError", err.Error())
				}
				line = append(line, pdfExcelCellText(s, cellText))
			}
			if headers == nil {
				headers = line
				continue
			}
			rows = append(rows, line)
		}
		blocks = append(blocks, pdfBlock{Kind: "table", Headers: headers, Rows: rows, Align: "left"})
	}
	return blocks
}

func pdfExcelCellText(s *Session, cellText string) string {
	kind, err := ahdruntime.ExcelCellKind(cellText)
	if err != nil {
		s.raise("PDFError", err.Error())
	}
	switch kind {
	case "Blank":
		return ""
	case "String":
		value, err := ahdruntime.ExcelCellString(cellText)
		if err != nil {
			s.raise("PDFError", err.Error())
		}
		return value
	case "Int":
		value, err := ahdruntime.ExcelCellInt(cellText)
		if err != nil {
			s.raise("PDFError", err.Error())
		}
		return strconv.FormatInt(value, 10)
	case "Real":
		value, err := ahdruntime.ExcelCellReal(cellText)
		if err != nil {
			s.raise("PDFError", err.Error())
		}
		text := strconv.FormatFloat(value, 'g', -1, 64)
		if !strings.ContainsAny(text, ".eE") {
			text += ".0"
		}
		return text
	case "Bool":
		value, err := ahdruntime.ExcelCellBool(cellText)
		if err != nil {
			s.raise("PDFError", err.Error())
		}
		if value {
			return "true"
		}
		return "false"
	case "Formula":
		value, err := ahdruntime.ExcelCellFormula(cellText)
		if err != nil {
			s.raise("PDFError", err.Error())
		}
		return value
	default:
		return ""
	}
}
