package ahdruntime

// The PDF standard module. It shares the same low-level offline renderer as
// Latex (ahdLatexCompile, ahdLatexVerifyPDF, ahdLatexPublish, ahdLatexRuntime,
// all in ahdruntime.go) but owns its own small document model and LaTeX-body
// construction, so PDF users never see or write raw TeX: every String a
// caller supplies is escaped text, never renderer source. This file is also
// emitted verbatim into native programs (with only its package clause
// rewritten), so it intentionally depends on the Go standard library and the
// sibling AhdCode runtime only.

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ahdPDFRaise(message string) { AhdRaiseClass(AhdClassPDFError, message) }

// ahdPDFBlock is the private content-block shape, mirroring Word's
// ahdWordBlock. Kind selects which of the remaining fields are meaningful.
type ahdPDFBlock struct {
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

// AhdPDFDocument is the runtime interchange shape the generated backend reads
// and writes through the PDFDocument Class's one hidden field.
type AhdPDFDocument struct {
	Blocks []string
}

// ahdPDFAppend returns a new document with one more block, always copying the
// existing block slice first so two documents derived from the same base
// never share a backing array.
func ahdPDFAppend(doc AhdPDFDocument, block ahdPDFBlock) AhdPDFDocument {
	encoded, _ := json.Marshal(block)
	blocks := append(append([]string(nil), doc.Blocks...), string(encoded))
	return AhdPDFDocument{Blocks: blocks}
}

func ahdPDFDecodeBlocks(doc AhdPDFDocument) []ahdPDFBlock {
	blocks := make([]ahdPDFBlock, len(doc.Blocks))
	for index, raw := range doc.Blocks {
		var block ahdPDFBlock
		if err := json.Unmarshal([]byte(raw), &block); err != nil {
			ahdPDFRaise("PDFDocument storage is corrupted")
		}
		blocks[index] = block
	}
	return blocks
}

// AhdPDFNew starts an empty PDFDocument.
func AhdPDFNew() AhdPDFDocument { return AhdPDFDocument{} }

var ahdPDFParagraphAlignments = map[string]bool{"left": true, "center": true, "right": true, "justify": true}
var ahdPDFTableAlignments = map[string]bool{"left": true, "center": true, "right": true}

func AhdPDFHeading(doc AhdPDFDocument, text string, level int64) AhdPDFDocument {
	if level < 1 || level > 6 {
		ahdPDFRaise("heading level must be between 1 and 6")
	}
	return ahdPDFAppend(doc, ahdPDFBlock{Kind: "heading", Text: text, Level: int(level)})
}

func AhdPDFParagraph(doc AhdPDFDocument, text, align string, bold, italic, underline bool) AhdPDFDocument {
	if !ahdPDFParagraphAlignments[align] {
		ahdPDFRaise("paragraph align must be left, center, right, or justify")
	}
	return ahdPDFAppend(doc, ahdPDFBlock{
		Kind: "paragraph", Text: text, Align: align, Bold: bold, Italic: italic, Underline: underline,
	})
}

// AhdPDFTable validates shape before ever building LaTeX source, so a
// malformed table is always a clean PDFError, never a corrupt or misleading
// render. No padding, truncation, or repair: a ragged row is rejected.
func AhdPDFTable(doc AhdPDFDocument, headers *AhdList[string], rows *AhdList[*AhdList[string]], align string) AhdPDFDocument {
	if !ahdPDFTableAlignments[align] {
		ahdPDFRaise("table align must be left, center, or right")
	}
	headerValues := headers.Snapshot()
	if len(headerValues) == 0 {
		ahdPDFRaise("table requires at least one column")
	}
	rowValues := rows.Snapshot()
	grid := make([][]string, len(rowValues))
	for index, row := range rowValues {
		nonNullRow := AhdNonNull(row)
		cells := nonNullRow.Snapshot()
		if len(cells) != len(headerValues) {
			ahdPDFRaise("table row column count does not match headers")
		}
		grid[index] = cells
	}
	return ahdPDFAppend(doc, ahdPDFBlock{Kind: "table", Headers: headerValues, Rows: grid, Align: align})
}

// AhdPDFImage reads and embeds the image bytes immediately, so the produced
// PDFDocument (and the PDF it eventually saves to) never depends on the
// source file surviving or on the working directory staying the same: moving
// or deleting the source file afterward changes nothing, and save() never
// silently relies on the repository working directory.
func AhdPDFImage(doc AhdPDFDocument, path string, size *AhdPair[string, float64]) AhdPDFDocument {
	if path == "" {
		ahdPDFRaise("image path must not be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		ahdPDFRaise("could not read image: " + err.Error())
	}
	format, naturalWidth, naturalHeight := ahdPDFDecodeImage(data)
	widthCM, heightCM := ahdPDFImageExtent(size, naturalWidth, naturalHeight)
	return ahdPDFAppend(doc, ahdPDFBlock{
		Kind: "image", Media: data, MediaExt: format, WidthCM: widthCM, HeightCM: heightCM,
	})
}

func ahdPDFDecodeImage(data []byte) (format string, width, height int) {
	config, formatName, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		ahdPDFRaise("unsupported image format: PDF supports PNG and JPEG")
	}
	switch formatName {
	case "png":
		return "png", config.Width, config.Height
	case "jpeg":
		return "jpeg", config.Width, config.Height
	default:
		ahdPDFRaise("unsupported image format: PDF supports PNG and JPEG")
		return "", 0, 0
	}
}

// ahdPDFImageExtent resolves the four size.md-documented cases. An empty
// Pair returns (0, 0), meaning "let the renderer use the image's own natural
// size" -- save() omits width/height options entirely in that case, exactly
// like Latex.image(path) with no size argument today.
func ahdPDFImageExtent(size *AhdPair[string, float64], naturalWidth, naturalHeight int) (widthCM, heightCM float64) {
	size.require()
	var width, height float64
	var hasWidth, hasHeight bool
	for _, key := range size.keys {
		switch key {
		case "width":
			hasWidth = true
			width = size.values[key]
		case "height":
			hasHeight = true
			height = size.values[key]
		default:
			ahdPDFRaise("image size supports only width and height")
		}
	}
	if hasWidth && width <= 0 {
		ahdPDFRaise("image width must be positive")
	}
	if hasHeight && height <= 0 {
		ahdPDFRaise("image height must be positive")
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

func AhdPDFPageBreak(doc AhdPDFDocument) AhdPDFDocument {
	return ahdPDFAppend(doc, ahdPDFBlock{Kind: "pageBreak"})
}

// ahdPDFHeadingCommand maps a 1..6 heading level to LaTeX source. Standard
// article-class sectioning has five named depths (section..subparagraph);
// level 6 is rendered as a bold, unnumbered run rather than inventing a
// sixth sectioning command.
func ahdPDFHeadingCommand(block ahdPDFBlock) string {
	escaped := AhdLatexEscape(block.Text)
	switch block.Level {
	case 1:
		return `\section{` + escaped + "}\n"
	case 2:
		return `\subsection{` + escaped + "}\n"
	case 3:
		return `\subsubsection{` + escaped + "}\n"
	case 4:
		return `\paragraph{` + escaped + "}\n"
	case 5:
		return `\subparagraph{` + escaped + "}\n"
	default:
		return `\textbf{` + escaped + "}\\par\n"
	}
}

func ahdPDFParagraphBody(block ahdPDFBlock) string {
	text := AhdLatexEscape(block.Text)
	if block.Bold {
		text = `\textbf{` + text + `}`
	}
	if block.Italic {
		text = `\textit{` + text + `}`
	}
	if block.Underline {
		text = `\underline{` + text + `}`
	}
	text += "\n"
	switch block.Align {
	case "center":
		return AhdLatexCenter(text)
	case "left":
		return "\\begin{flushleft}\n" + text + "\\end{flushleft}\n"
	case "right":
		return "\\begin{flushright}\n" + text + "\\end{flushright}\n"
	default: // "justify": LaTeX's own default paragraph behavior.
		return text + "\n"
	}
}

func ahdPDFTableBody(block ahdPDFBlock) string {
	column := map[string]string{"left": "l", "center": "c", "right": "r"}[block.Align]
	if column == "" {
		column = "l"
	}
	var result strings.Builder
	result.WriteString("\\begin{tabular}{")
	result.WriteString(strings.Repeat(column, len(block.Headers)))
	result.WriteString("}\n\\toprule\n")
	for index, value := range block.Headers {
		if index != 0 {
			result.WriteString(" & ")
		}
		result.WriteString(AhdLatexEscape(value))
	}
	result.WriteString(" \\\\\n\\midrule\n")
	for _, row := range block.Rows {
		for index, value := range row {
			if index != 0 {
				result.WriteString(" & ")
			}
			result.WriteString(AhdLatexEscape(value))
		}
		result.WriteString(" \\\\\n")
	}
	result.WriteString("\\bottomrule\n\\end{tabular}\n")
	return result.String()
}

// ahdPDFImageBody stages the block's embedded bytes as a file in the same
// directory the document is compiled from (name reused as the fixed pattern
// below so a repeated save of the same document produces byte-identical
// LaTeX source), then references it with a relative path.
func ahdPDFImageBody(block ahdPDFBlock, index int, directory string) (string, error) {
	name := "ahdpdf-image-" + strconv.Itoa(index) + "." + block.MediaExt
	if err := os.WriteFile(filepath.Join(directory, name), block.Media, 0o600); err != nil {
		return "", err
	}
	options := ""
	if block.WidthCM > 0 || block.HeightCM > 0 {
		options = "[width=" + ahdFormatReal(block.WidthCM) + "cm,height=" + ahdFormatReal(block.HeightCM) + "cm]"
	}
	return "\\includegraphics" + options + "{" + name + "}\n", nil
}

const ahdPDFMargin = 2.54

// ahdPDFPreamble is a small, fixed preamble: A4, portrait, the same
// proven-offline font/package closure Latex.document() already uses
// successfully against the staged --only-cached Tectonic bundle. PDF
// deliberately does not expose page-size, orientation, or margin
// configuration in v0.1.20.
func ahdPDFPreamble() string {
	var result strings.Builder
	result.WriteString("\\documentclass[a4paper]{article}\n")
	result.WriteString("\\usepackage{fontspec}\n")
	result.WriteString("\\setmainfont{lmroman10-regular.otf}[BoldFont=lmroman10-bold.otf,ItalicFont=lmroman10-italic.otf,BoldItalicFont=lmroman10-bolditalic.otf]\n")
	result.WriteString("\\usepackage{geometry,graphicx,booktabs,array}\n")
	result.WriteString("\\geometry{a4paper,margin=" + ahdFormatReal(ahdPDFMargin) + "cm}\n")
	return result.String()
}

// AhdPDFSave builds the document's LaTeX body in a secure temporary
// directory, then compiles and publishes it through the exact same low-level
// renderer Latex.pdf uses (ahdLatexCompile/ahdLatexVerifyPDF/ahdLatexPublish),
// passing PDFError as the catchable class instead of LatexError. No .tex
// sidecar is ever produced by the PDF module.
func AhdPDFSave(doc AhdPDFDocument, path string) {
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		ahdPDFRaise("PDFDocument.save destination must use the .pdf extension")
	}
	blocks := ahdPDFDecodeBlocks(doc)

	directory, err := os.MkdirTemp("", "ahdcode-pdf-source-*")
	if err != nil {
		ahdPDFRaise("could not create a secure temporary directory: " + err.Error())
	}
	defer os.RemoveAll(directory)

	var body strings.Builder
	imageIndex := 0
	for _, block := range blocks {
		switch block.Kind {
		case "heading":
			body.WriteString(ahdPDFHeadingCommand(block))
		case "paragraph":
			body.WriteString(ahdPDFParagraphBody(block))
		case "table":
			body.WriteString(ahdPDFTableBody(block))
		case "image":
			rendered, writeErr := ahdPDFImageBody(block, imageIndex, directory)
			if writeErr != nil {
				ahdPDFRaise("could not stage image: " + writeErr.Error())
			}
			imageIndex++
			body.WriteString(rendered)
		case "pageBreak":
			body.WriteString(AhdLatexPageBreak())
		default:
			ahdPDFRaise("PDFDocument storage is corrupted")
		}
	}

	var source strings.Builder
	source.WriteString(ahdPDFPreamble())
	source.WriteString("\\begin{document}\n")
	source.WriteString(body.String())
	source.WriteString("\\end{document}\n")

	input := filepath.Join(directory, "document.tex")
	if err := os.WriteFile(input, []byte(source.String()), 0o600); err != nil {
		ahdPDFRaise("could not write temporary LaTeX source: " + err.Error())
	}
	ahdLatexCompile(AhdClassPDFError, input, directory, path)
}

// AhdPDFFromWord converts a Word Document's own semantic blocks directly into
// PDF blocks -- no DOCX round trip, no Office/LibreOffice dependency. It
// preserves headings, paragraph text/align/bold/italic/underline, table
// content, page breaks, and images (converting Word's EMU dimensions back to
// centimeters); a table's merge geometry has no PDF equivalent attempted (see
// docs/PDF.md), so it is dropped, matching the "semantic conversion, not
// pixel-perfect printing" contract. The source Document is read-only
// throughout: only its already-decoded blocks are read, never mutated.
func AhdPDFFromWord(document AhdWordDocument) AhdPDFDocument {
	wordBlocks := ahdWordDecodeBlocks(document)
	result := AhdPDFDocument{}
	for _, block := range wordBlocks {
		switch block.Kind {
		case "heading":
			result = ahdPDFAppend(result, ahdPDFBlock{Kind: "heading", Text: block.Text, Level: block.Level})
		case "paragraph":
			result = ahdPDFAppend(result, ahdPDFBlock{
				Kind: "paragraph", Text: block.Text, Align: block.Align,
				Bold: block.Bold, Italic: block.Italic, Underline: block.Underline,
			})
		case "table":
			result = ahdPDFAppend(result, ahdPDFBlock{Kind: "table", Headers: block.Headers, Rows: block.Rows, Align: block.Align})
		case "image":
			result = ahdPDFAppend(result, ahdPDFBlock{
				Kind: "image", Media: block.Media, MediaExt: block.MediaExt,
				WidthCM:  float64(block.WidthEMU) / ahdWordEMUPerCentimeter,
				HeightCM: float64(block.HeightEMU) / ahdWordEMUPerCentimeter,
			})
		case "pageBreak":
			result = ahdPDFAppend(result, ahdPDFBlock{Kind: "pageBreak"})
		}
	}
	return result
}

// ahdPDFMaxExcelColumns bounds PDF.fromExcel's per-Sheet table width. AhdCode
// never silently drops columns to fit a page: a Sheet whose used range is
// wider than this raises PDFError instead of truncating or attempting a
// best-effort multi-page column-wrapping layout.
const ahdPDFMaxExcelColumns = 10

// AhdPDFFromExcel is a semantic tabular export of a Workbook, not Excel
// page-layout emulation: every Sheet becomes a heading (the Sheet name)
// followed by a table over its used range, in Workbook order. The used
// range's first row becomes the table header and the remaining rows become
// the table body (a purely presentational choice -- Excel workbooks have no
// formal header-row concept, and no cell is ever dropped either way). Formula
// cells display their formula source text, never a fabricated cached result,
// because AhdCode does not evaluate Excel formulas. Merged cells cannot hide
// a value: Excel's own model already guarantees a merge's non-anchor cells
// are Blank, so rendering the plain grid loses nothing -- no multi-column
// cell spanning is attempted in the output table.
func AhdPDFFromExcel(workbookText string) AhdPDFDocument {
	workbook, err := excelWorkbook(workbookText)
	if err != nil {
		ahdPDFRaise("Workbook storage is corrupted")
	}
	if len(workbook.Sheets) == 0 {
		ahdPDFRaise("PDF.fromExcel requires a Workbook with at least one Sheet")
	}
	result := AhdPDFDocument{}
	for _, sheet := range workbook.Sheets {
		result = ahdPDFAppend(result, ahdPDFBlock{Kind: "heading", Text: sheet.Name, Level: 1})
		used, set := excelUsedRange(sheet)
		if !set {
			continue
		}
		columnCount := int(used.EndColumn - used.StartColumn + 1)
		if columnCount > ahdPDFMaxExcelColumns {
			ahdPDFRaise("PDF.fromExcel: Sheet " + strconv.Quote(sheet.Name) + " has " + strconv.Itoa(columnCount) +
				" used columns, which exceeds the supported limit of " + strconv.Itoa(ahdPDFMaxExcelColumns))
		}
		var headers []string
		var rows [][]string
		for row := used.StartRow; row <= used.EndRow; row++ {
			line := make([]string, 0, columnCount)
			for column := used.StartColumn; column <= used.EndColumn; column++ {
				line = append(line, ahdPDFExcelCellText(excelCellAt(sheet, row, column)))
			}
			if headers == nil {
				headers = line
				continue
			}
			rows = append(rows, line)
		}
		result = ahdPDFAppend(result, ahdPDFBlock{Kind: "table", Headers: headers, Rows: rows, Align: "left"})
	}
	return result
}

// ahdPDFExcelCellText renders one Excel Cell deterministically for tabular
// display. A Formula Cell shows its formula source, matching Excel's own
// Cell.formula() contract, never a fabricated cached value.
func ahdPDFExcelCellText(cell excelCellData) string {
	switch cell.Kind {
	case "Blank":
		return ""
	case "String":
		return cell.Text
	case "Int":
		return strconv.FormatInt(cell.Int, 10)
	case "Real":
		value := strconv.FormatFloat(cell.Real, 'g', -1, 64)
		if !strings.ContainsAny(value, ".eE") {
			value += ".0"
		}
		return value
	case "Bool":
		if cell.Bool {
			return "true"
		}
		return "false"
	case "Formula":
		return cell.Text
	default:
		return ""
	}
}
