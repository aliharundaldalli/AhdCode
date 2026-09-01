package evaluator

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The PDF standard module's REPL implementation. It mirrors the native
// backend's ahdruntime/pdf.go logic function-for-function -- the same block
// encoding, the same LaTeX-body construction rules -- but operates on the
// evaluator's own List/Pair/Instance representation instead of
// AhdList/AhdPair, exactly as word.go independently reimplements the Word
// runtime rather than importing ahdruntime for its own document model.
// save() is the one exception: it cannot invoke the offline Tectonic
// renderer interactively, so it raises PDFError just like Latex.pdf/pdfFile
// already do in this same evaluator.

const pdfDocumentClassID = ir.ClassID("builtin:PDF::class::PDFDocument")

var pdfBlocksField = ir.FieldID(string(pdfDocumentClassID) + "::field::blocks")

// pdfBlock is the private content-block shape, identical in spirit to the
// native runtime's ahdPDFBlock.
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

// pdfDocumentBlocks reads the one hidden storage field of a PDFDocument
// instance. The stored List is never handed out, so a caller cannot reach it
// to mutate.
func (s *Session) pdfDocumentBlocks(value any) []pdfBlock {
	instance := s.requireInstance(value)
	stored, ok := instance.Fields[pdfBlocksField].(*List)
	if !ok {
		s.raise("PDFError", "value is not a PDFDocument")
	}
	blocks := make([]pdfBlock, len(stored.Items))
	for index, raw := range stored.Items {
		var block pdfBlock
		if err := json.Unmarshal([]byte(raw.(string)), &block); err != nil {
			s.raise("PDFError", "PDFDocument storage is corrupted")
		}
		blocks[index] = block
	}
	return blocks
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
	blocks := s.pdfDocumentBlocks(value)
	next := make([]pdfBlock, len(blocks)+1)
	copy(next, blocks)
	next[len(blocks)] = block
	return s.pdfDocumentValue(next)
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
		return s.pdfImage(receiver, args)
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

func (s *Session) pdfImage(receiver any, args []any) any {
	path := args[0].(string)
	if path == "" {
		s.raise("PDFError", "image path must not be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.raise("PDFError", "could not read image: "+err.Error())
	}
	format, naturalWidth, naturalHeight := pdfDecodeImage(s, data)
	size := &Pair{Values: map[any]any{}}
	if len(args) > 1 && args[1] != nil {
		size = s.requirePair(args[1])
	}
	widthCM, heightCM := s.pdfImageExtent(size, naturalWidth, naturalHeight)
	return s.pdfAppend(receiver, pdfBlock{Kind: "image", Media: data, MediaExt: format, WidthCM: widthCM, HeightCM: heightCM})
}

func pdfDecodeImage(s *Session, data []byte) (string, int, int) {
	config, formatName, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		s.raise("PDFError", "unsupported image format: PDF supports PNG and JPEG")
	}
	switch formatName {
	case "png":
		return "png", config.Width, config.Height
	case "jpeg":
		return "jpeg", config.Width, config.Height
	default:
		s.raise("PDFError", "unsupported image format: PDF supports PNG and JPEG")
		return "", 0, 0
	}
}

func (s *Session) pdfImageExtent(size *Pair, naturalWidth, naturalHeight int) (float64, float64) {
	var width, height float64
	var hasWidth, hasHeight bool
	for _, key := range size.Keys {
		k := key.(string)
		switch k {
		case "width":
			hasWidth = true
			width = size.Values[key].(float64)
		case "height":
			hasHeight = true
			height = size.Values[key].(float64)
		default:
			s.raise("PDFError", "image size supports only width and height")
		}
	}
	if hasWidth && width <= 0 {
		s.raise("PDFError", "image width must be positive")
	}
	if hasHeight && height <= 0 {
		s.raise("PDFError", "image height must be positive")
	}
	if !hasWidth && !hasHeight {
		return 0, 0
	}
	if naturalWidth <= 0 || naturalHeight <= 0 {
		naturalWidth, naturalHeight = 1, 1
	}
	aspect := float64(naturalHeight) / float64(naturalWidth)
	switch {
	case hasWidth && hasHeight:
		return width, height
	case hasWidth:
		return width, width * aspect
	default:
		return height / aspect, height
	}
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
