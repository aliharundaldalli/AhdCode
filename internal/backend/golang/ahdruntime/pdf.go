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
	// The v1.3.0 fields are omitted when empty, so a document built only from
	// v1.2.0 operations stores exactly the blocks it did before.
	TransformKeys   []string       `json:"transformKeys,omitempty"`
	TransformValues []float64      `json:"transformValues,omitempty"`
	Values          []string       `json:"values,omitempty"`
	URL             string         `json:"url,omitempty"`
	Total           bool           `json:"total,omitempty"`
	Layout          *AhdPageLayout `json:"layout,omitempty"`
	Author          string         `json:"author,omitempty"`
	Subject         string         `json:"subject,omitempty"`
	Creator         string         `json:"creator,omitempty"`
}

// ---------------------------------------------------------------------------
// v1.3.0 block builders
//
// Each builder validates its arguments and returns the encoded block or a
// problem. The native entry points raise the problem as PDFError; the
// interactive evaluator calls the same builders, so both store identical
// blocks and report identical messages.
// ---------------------------------------------------------------------------

func ahdPDFBlockText(block ahdPDFBlock) string {
	encoded, _ := json.Marshal(block)
	return string(encoded)
}

// ahdPDFAppendText appends a block a builder already encoded, or raises the
// builder's problem.
func ahdPDFAppendText(doc AhdPDFDocument, text, problem string) AhdPDFDocument {
	if problem != "" {
		ahdPDFRaise(problem)
	}
	return AhdPDFDocument{Blocks: append(append([]string(nil), doc.Blocks...), text)}
}

// ahdPDFBlockAlign validates the placement of a block that is not a
// paragraph: a QR symbol, barcode, link, or page number sits left, center,
// or right.
func ahdPDFBlockAlign(operation, align string) string {
	if align != "left" && align != "center" && align != "right" {
		return operation + " align must be left, center, or right; received " + strconv.Quote(align)
	}
	return ""
}

// AhdPDFLayoutBlock is PDFDocument.layout. A document without a layout keeps
// the A4 page with 2.54 cm margins; when layout is called more than once, the
// last call wins.
func AhdPDFLayoutBlock(paper string, landscape bool, sizeKeys []string, sizeValues []float64, marginKeys []string, marginValues []float64) (string, string) {
	layout, problem := AhdPageLayoutText("PDFDocument.layout", "A4", paper, landscape, ahdPDFMargin, sizeKeys, sizeValues, marginKeys, marginValues)
	if problem != "" {
		return "", problem
	}
	return ahdPDFBlockText(ahdPDFBlock{Kind: "layout", Layout: &layout}), ""
}

// AhdPDFRunningBlock is PDFDocument.header (part "header") or
// PDFDocument.footer (part "footer"): plain text for the left, center, and
// right of every page. A footer replaces the page number shown by default;
// pageNumbers places it in a free footer region. The last call wins.
func AhdPDFRunningBlock(part, left, center, right string) (string, string) {
	for _, value := range []string{left, center, right} {
		if strings.ContainsAny(value, "\r\n") {
			return "", "PDFDocument." + part + " text must be a single line"
		}
	}
	return ahdPDFBlockText(ahdPDFBlock{Kind: part, Values: []string{left, center, right}}), ""
}

// AhdPDFPageNumbersBlock is PDFDocument.pageNumbers: "3", or with total
// "3 / 12", in one footer region.
func AhdPDFPageNumbersBlock(align string, total bool) (string, string) {
	if problem := ahdPDFBlockAlign("PDFDocument.pageNumbers", align); problem != "" {
		return "", problem
	}
	return ahdPDFBlockText(ahdPDFBlock{Kind: "pageNumbers", Align: align, Total: total}), ""
}

// AhdPDFLinkBlock is PDFDocument.link: a clickable line of text.
func AhdPDFLinkBlock(text, url, align string) (string, string) {
	if _, problem := AhdLatexLinkText("PDFDocument.link", text, url); problem != "" {
		return "", problem
	}
	if problem := ahdPDFBlockAlign("PDFDocument.link", align); problem != "" {
		return "", problem
	}
	return ahdPDFBlockText(ahdPDFBlock{Kind: "link", Text: text, URL: url, Align: align}), ""
}

// AhdPDFBookmarkBlock is PDFDocument.bookmark: an outline entry for the
// position where it is added.
func AhdPDFBookmarkBlock(title string, level int64) (string, string) {
	if _, problem := AhdLatexBookmarkText("PDFDocument.bookmark", title, level); problem != "" {
		return "", problem
	}
	return ahdPDFBlockText(ahdPDFBlock{Kind: "bookmark", Text: title, Level: int(level)}), ""
}

// AhdPDFMetadataBlock is PDFDocument.metadata: the PDF document properties.
// The last call wins.
func AhdPDFMetadataBlock(title, author, subject string, keywords []string, creator string) (string, string) {
	if problem := ahdDocumentKeywordsProblem("PDFDocument.metadata", keywords); problem != "" {
		return "", problem
	}
	for _, value := range append([]string{title, author, subject, creator}, keywords...) {
		if strings.ContainsAny(value, "\r\n") {
			return "", "PDFDocument.metadata values must be single lines"
		}
	}
	return ahdPDFBlockText(ahdPDFBlock{
		Kind: "metadata", Text: title, Author: author, Subject: subject, Values: keywords, Creator: creator,
	}), ""
}

// AhdPDFImageBlock is PDFDocument.image. It reads the image immediately, so
// the saved PDF never depends on the source file afterward. PNG and JPEG
// bytes are embedded as they are; an SVG is converted to vector PGF here, so
// an unsupported SVG is rejected when it is added rather than when the
// document saves.
func AhdPDFImageBlock(path string, sizeKeys []string, sizeValues []float64, transformKeys []string, transformValues []float64) (string, string) {
	if path == "" {
		return "", "image path must not be empty"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "could not read image: " + err.Error()
	}
	block := ahdPDFBlock{Kind: "image"}
	var naturalWidth, naturalHeight float64
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		picture, problem := AhdSVGConvert("PDFDocument.image", data)
		if problem != "" {
			return "", problem
		}
		block.MediaExt, block.Text = "svg", picture.Source
		naturalWidth, naturalHeight = picture.Width*2.54/72, picture.Height*2.54/72
	} else {
		config, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (format != "png" && format != "jpeg") {
			return "", "unsupported image format: PDF supports PNG, JPEG, and SVG"
		}
		block.Media, block.MediaExt = data, format
		naturalWidth, naturalHeight = float64(config.Width), float64(config.Height)
	}
	width, height, problem := ahdPDFImageExtentText(sizeKeys, sizeValues, naturalWidth, naturalHeight)
	if problem != "" {
		return "", problem
	}
	transform, problem := AhdImageTransformText("PDFDocument.image", transformKeys, transformValues)
	if problem != "" {
		return "", problem
	}
	trimWidth, trimHeight := width, height
	if block.MediaExt == "svg" && width == 0 && height == 0 {
		// An SVG's natural size is known in centimeters; a raster image's
		// depends on its resolution, so only a sized raster image is checked.
		trimWidth, trimHeight = naturalWidth, naturalHeight
	}
	if problem := transform.checkTrim("PDFDocument.image", trimWidth, trimHeight); problem != "" {
		return "", problem
	}
	block.WidthCM, block.HeightCM = width, height
	if len(transformKeys) > 0 {
		block.TransformKeys, block.TransformValues = transformKeys, transformValues
	}
	return ahdPDFBlockText(block), ""
}

// ahdPDFImageExtentText resolves an image size against the image's natural
// aspect ratio. Neither width nor height returns (0, 0): the renderer uses
// the image's natural size.
func ahdPDFImageExtentText(keys []string, values []float64, naturalWidth, naturalHeight float64) (float64, float64, string) {
	var width, height float64
	var hasWidth, hasHeight bool
	for index, key := range keys {
		switch key {
		case "width":
			hasWidth, width = true, values[index]
		case "height":
			hasHeight, height = true, values[index]
		default:
			return 0, 0, "image size supports only width and height"
		}
	}
	if hasWidth && width <= 0 {
		return 0, 0, "image width must be positive"
	}
	if hasHeight && height <= 0 {
		return 0, 0, "image height must be positive"
	}
	if !hasWidth && !hasHeight {
		return 0, 0, ""
	}
	if naturalWidth <= 0 || naturalHeight <= 0 {
		naturalWidth, naturalHeight = 1, 1
	}
	aspect := naturalHeight / naturalWidth
	switch {
	case hasWidth && hasHeight:
		return width, height, ""
	case hasWidth:
		return width, width * aspect, ""
	default:
		return height / aspect, height, ""
	}
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

// AhdPDFImage is the v1.2.0 entry point of PDFDocument.image, without a
// transform.
func AhdPDFImage(doc AhdPDFDocument, path string, size *AhdPair[string, float64]) AhdPDFDocument {
	return AhdPDFImageComplete(doc, path, size, nil)
}

// AhdPDFImageComplete is the native entry point of PDFDocument.image. The
// image bytes are read and embedded immediately (see AhdPDFImageBlock), so
// moving or deleting the source file afterward changes nothing, and save()
// never relies on the working directory.
func AhdPDFImageComplete(doc AhdPDFDocument, path string, size, transform *AhdPair[string, float64]) AhdPDFDocument {
	sizeKeys, sizeValues := ahdPairEntries(size)
	transformKeys, transformValues := ahdPairEntries(transform)
	text, problem := AhdPDFImageBlock(path, sizeKeys, sizeValues, transformKeys, transformValues)
	return ahdPDFAppendText(doc, text, problem)
}

// AhdPDFLayout is the native entry point of PDFDocument.layout.
func AhdPDFLayout(doc AhdPDFDocument, paper string, landscape bool, pageSize, margins *AhdPair[string, float64]) AhdPDFDocument {
	sizeKeys, sizeValues := ahdPairEntries(pageSize)
	marginKeys, marginValues := ahdPairEntries(margins)
	text, problem := AhdPDFLayoutBlock(paper, landscape, sizeKeys, sizeValues, marginKeys, marginValues)
	return ahdPDFAppendText(doc, text, problem)
}

// AhdPDFRunning is the native entry point of PDFDocument.header and
// PDFDocument.footer.
func AhdPDFRunning(doc AhdPDFDocument, part, left, center, right string) AhdPDFDocument {
	text, problem := AhdPDFRunningBlock(part, left, center, right)
	return ahdPDFAppendText(doc, text, problem)
}

// AhdPDFPageNumbers is the native entry point of PDFDocument.pageNumbers.
func AhdPDFPageNumbers(doc AhdPDFDocument, align string, total bool) AhdPDFDocument {
	text, problem := AhdPDFPageNumbersBlock(align, total)
	return ahdPDFAppendText(doc, text, problem)
}

// AhdPDFLink is the native entry point of PDFDocument.link.
func AhdPDFLink(doc AhdPDFDocument, text, url, align string) AhdPDFDocument {
	block, problem := AhdPDFLinkBlock(text, url, align)
	return ahdPDFAppendText(doc, block, problem)
}

// AhdPDFBookmark is the native entry point of PDFDocument.bookmark.
func AhdPDFBookmark(doc AhdPDFDocument, title string, level int64) AhdPDFDocument {
	text, problem := AhdPDFBookmarkBlock(title, level)
	return ahdPDFAppendText(doc, text, problem)
}

// AhdPDFMetadata is the native entry point of PDFDocument.metadata.
func AhdPDFMetadata(doc AhdPDFDocument, title, author, subject string, keywords *AhdList[string], creator string) AhdPDFDocument {
	var values []string
	if keywords != nil {
		values = keywords.Snapshot()
	}
	text, problem := AhdPDFMetadataBlock(title, author, subject, values, creator)
	return ahdPDFAppendText(doc, text, problem)
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
// LaTeX source), then references it with a relative path. An SVG block is
// staged as its converted PGF source and scaled as a box. A transform wraps
// the sized image; without one, a PNG or JPEG renders exactly as in v1.2.0.
func ahdPDFImageBody(block ahdPDFBlock, index int, directory string) (string, error) {
	transform, problem := AhdImageTransformText("PDFDocument.image", block.TransformKeys, block.TransformValues)
	if problem != "" {
		ahdPDFRaise("PDFDocument storage is corrupted")
	}
	marker, content := "", ""
	if block.MediaExt == "svg" {
		name := "ahdpdf-image-" + strconv.Itoa(index) + "-svg.tex"
		if err := os.WriteFile(filepath.Join(directory, name), []byte(block.Text), 0o600); err != nil {
			return "", err
		}
		marker = "%\n% AHDCODE_TIKZ\n"
		content = ahdLatexScaledBox(block.WidthCM, block.HeightCM, "\\input{"+name+"}")
	} else {
		name := "ahdpdf-image-" + strconv.Itoa(index) + "." + block.MediaExt
		if err := os.WriteFile(filepath.Join(directory, name), block.Media, 0o600); err != nil {
			return "", err
		}
		options := ""
		if block.WidthCM > 0 || block.HeightCM > 0 {
			options = "[width=" + ahdFormatReal(block.WidthCM) + "cm,height=" + ahdFormatReal(block.HeightCM) + "cm]"
		}
		content = "\\includegraphics" + options + "{" + name + "}"
	}
	if transform.NeedsPGF() {
		marker = "%\n% AHDCODE_TIKZ\n% AHDCODE_IMAGE\n"
	}
	return marker + transform.Wrap(content) + "\n", nil
}

// ahdPDFAligned places a block's content at the left, center, or right.
func ahdPDFAligned(align, content string) string {
	switch align {
	case "center":
		return AhdLatexCenter(content)
	case "right":
		return "\\begin{flushright}\n" + content + "\\end{flushright}\n"
	default:
		return "\\begin{flushleft}\n" + content + "\\end{flushleft}\n"
	}
}

// ahdPDFRunningText renders the page header and footer. A header alone keeps
// the default centered page number; a footer replaces it, and pageNumbers
// places the number in one free footer region.
func ahdPDFRunningText(header, footer []string, numbers *ahdPDFBlock) string {
	result := ""
	if len(header) == 3 {
		result += AhdLatexRunningText("head", AhdLatexEscape(header[0]), AhdLatexEscape(header[1]), AhdLatexEscape(header[2]))
	}
	if len(footer) != 3 && numbers == nil {
		return result
	}
	regions := []string{"", "", ""}
	if len(footer) == 3 {
		for index, text := range footer {
			regions[index] = AhdLatexEscape(text)
		}
	}
	if numbers != nil {
		index := map[string]int{"left": 0, "center": 1, "right": 2}[numbers.Align]
		if regions[index] != "" {
			ahdPDFRaise("PDFDocument.pageNumbers " + numbers.Align + " overlaps the footer's " + numbers.Align + " text")
		}
		regions[index] = AhdLatexPageNumberText()
		if numbers.Total {
			regions[index] += " / " + AhdLatexPageCountText()
		}
	}
	return result + AhdLatexRunningText("foot", regions[0], regions[1], regions[2])
}

const ahdPDFMargin = 2.54

// ahdPDFPreamble is a small preamble: by default A4, portrait, 2.54 cm
// margins, and the same proven-offline font/package closure Latex.document()
// uses against the staged --only-cached Tectonic bundle. A configured layout
// replaces only the geometry line.
func ahdPDFPreamble(layout *AhdPageLayout) string {
	geometry := "a4paper,margin=" + ahdFormatReal(ahdPDFMargin) + "cm"
	if layout != nil && layout.Configured {
		geometry = layout.Geometry()
	}
	var result strings.Builder
	result.WriteString("\\documentclass[a4paper]{article}\n")
	result.WriteString("\\usepackage{fontspec}\n")
	result.WriteString("\\setmainfont{lmroman10-regular.otf}[BoldFont=lmroman10-bold.otf,ItalicFont=lmroman10-italic.otf,BoldItalicFont=lmroman10-bolditalic.otf]\n")
	result.WriteString("\\usepackage{geometry,graphicx,booktabs,array}\n")
	result.WriteString("\\geometry{" + geometry + "}\n")
	return result.String()
}

// ahdPDFSource builds a document's complete LaTeX source, staging its images
// in directory. Content blocks render in order; for layout, header, footer,
// pageNumbers, and metadata, the last block of each kind wins. Packages for
// links, TikZ, headers, page counts, bookmarks, and image transforms load
// only when a block needs them, so a document built only from v1.2.0
// operations compiles from exactly the v1.2.0 source.
func ahdPDFSource(blocks []ahdPDFBlock, directory string) string {
	var layout *AhdPageLayout
	var header, footer []string
	var numbers, metadata *ahdPDFBlock
	hyperlinks := false
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
		case "vector":
			body.WriteString(ahdPDFAligned(block.Align, block.Text))
		case "link":
			text, problem := AhdLatexLinkText("PDFDocument.link", block.Text, block.URL)
			if problem != "" {
				ahdPDFRaise("PDFDocument storage is corrupted")
			}
			hyperlinks = true
			body.WriteString(ahdPDFAligned(block.Align, text+"\n"))
		case "bookmark":
			text, problem := AhdLatexBookmarkText("PDFDocument.bookmark", block.Text, int64(block.Level))
			if problem != "" {
				ahdPDFRaise("PDFDocument storage is corrupted")
			}
			hyperlinks = true
			body.WriteString(text)
		case "layout":
			layout = block.Layout
		case "header":
			header = block.Values
		case "footer":
			footer = block.Values
		case "pageNumbers":
			current := block
			numbers = &current
		case "metadata":
			current := block
			metadata = &current
		default:
			ahdPDFRaise("PDFDocument storage is corrupted")
		}
	}
	content := ahdPDFRunningText(header, footer, numbers) + body.String()
	properties := ""
	if metadata != nil {
		properties = ahdDocumentProperties(metadata.Text, metadata.Author, metadata.Subject, metadata.Values, metadata.Creator)
	}
	var source strings.Builder
	source.WriteString(ahdPDFPreamble(layout))
	if hyperlinks || properties != "" {
		// Headings never add outline entries on their own: a PDFDocument's
		// outline is exactly its bookmark() calls, whether or not a link or
		// metadata call happens to load hyperref.
		source.WriteString("\\usepackage{hyperref}\n\\hypersetup{hidelinks,bookmarksdepth=-2}\n")
		source.WriteString(properties)
	}
	tikz, problem := AhdLatexTikZPreamble(content)
	if problem != "" {
		ahdPDFRaise("PDFDocument storage is corrupted")
	}
	source.WriteString(tikz)
	source.WriteString(AhdLatexFeaturePreamble(content))
	source.WriteString("\\begin{document}\n")
	source.WriteString(content)
	source.WriteString("\\end{document}\n")
	return source.String()
}

// AhdPDFSave builds the document's LaTeX source in a secure temporary
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

	source := ahdPDFSource(blocks, directory)
	input := filepath.Join(directory, "document.tex")
	if err := os.WriteFile(input, []byte(source), 0o600); err != nil {
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
