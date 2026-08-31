package evaluator

// DOCX package generation and reading for the REPL evaluator. This mirrors
// internal/backend/golang/ahdruntime/ahdruntime.go's Word section function
// for function - same block shape, same WordprocessingML, same bounded
// reading limits - so a Document behaves identically whether it was built
// through a native program or through the persistent REPL.

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const wordNamespaces = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`

func wordEscapeXML(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\'':
			b.WriteString("&apos;")
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteRune(r)
		case r < 0x20:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func wordAlignValue(align string) string {
	if align == "justify" {
		return "both"
	}
	return align
}

func wordHeadingXML(block wordBlock) string {
	style := "Heading" + strconv.Itoa(block.Level)
	return `<w:p><w:pPr><w:pStyle w:val="` + style + `"/></w:pPr><w:r><w:t xml:space="preserve">` +
		wordEscapeXML(block.Text) + `</w:t></w:r></w:p>`
}

func wordParagraphXML(block wordBlock) string {
	var run strings.Builder
	run.WriteString(`<w:r>`)
	var properties strings.Builder
	if block.Bold {
		properties.WriteString(`<w:b/>`)
	}
	if block.Italic {
		properties.WriteString(`<w:i/>`)
	}
	if block.Underline {
		properties.WriteString(`<w:u w:val="single"/>`)
	}
	if properties.Len() > 0 {
		run.WriteString(`<w:rPr>` + properties.String() + `</w:rPr>`)
	}
	run.WriteString(`<w:t xml:space="preserve">` + wordEscapeXML(block.Text) + `</w:t></w:r>`)
	return `<w:p><w:pPr><w:jc w:val="` + wordAlignValue(block.Align) + `"/></w:pPr>` + run.String() + `</w:p>`
}

func wordPageBreakXML() string {
	return `<w:p><w:r><w:br w:type="page"/></w:r></w:p>`
}

func wordSectionPropertiesXML() string {
	return `<w:sectPr><w:pgSz w:w="12240" w:h="15840"/>` +
		`<w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/>` +
		`</w:sectPr>`
}

func wordColumnWidth(columnCount int) int {
	const contentWidth = 9350
	if columnCount <= 0 {
		return contentWidth
	}
	return contentWidth / columnCount
}

func wordTableXML(block wordBlock) string {
	columnCount := len(block.Headers)
	rowCount := 1 + len(block.Rows)
	anchorMerge := make([][]int, rowCount)
	consumed := make([][]bool, rowCount)
	for r := 0; r < rowCount; r++ {
		anchorMerge[r] = make([]int, columnCount)
		consumed[r] = make([]bool, columnCount)
		for c := range anchorMerge[r] {
			anchorMerge[r][c] = -1
		}
	}
	for mergeIndex, merge := range block.Merges {
		row, column, rowSpan, columnSpan := merge[0], merge[1], merge[2], merge[3]
		for r := row; r < row+rowSpan; r++ {
			anchorMerge[r][column] = mergeIndex
			for c := column + 1; c < column+columnSpan; c++ {
				consumed[r][c] = true
			}
		}
	}
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:jc w:val="` + block.Align + `"/><w:tblBorders>`)
	for _, edge := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		b.WriteString(`<w:` + edge + ` w:val="single" w:sz="4" w:space="0" w:color="auto"/>`)
	}
	b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
	columnWidth := wordColumnWidth(columnCount)
	for i := 0; i < columnCount; i++ {
		b.WriteString(`<w:gridCol w:w="` + strconv.Itoa(columnWidth) + `"/>`)
	}
	b.WriteString(`</w:tblGrid>`)
	for r := 0; r < rowCount; r++ {
		var rowText []string
		if r == 0 {
			rowText = block.Headers
		} else {
			rowText = block.Rows[r-1]
		}
		b.WriteString(`<w:tr>`)
		for c := 0; c < columnCount; c++ {
			if consumed[r][c] {
				continue
			}
			mergeIndex := anchorMerge[r][c]
			columnSpan, rowSpan, mergeRow := 1, 1, r
			if mergeIndex >= 0 {
				merge := block.Merges[mergeIndex]
				mergeRow, rowSpan, columnSpan = merge[0], merge[2], merge[3]
			}
			var properties strings.Builder
			properties.WriteString(`<w:tcW w:w="0" w:type="auto"/>`)
			if columnSpan > 1 {
				properties.WriteString(`<w:gridSpan w:val="` + strconv.Itoa(columnSpan) + `"/>`)
			}
			if rowSpan > 1 {
				if r == mergeRow {
					properties.WriteString(`<w:vMerge w:val="restart"/>`)
				} else {
					properties.WriteString(`<w:vMerge/>`)
				}
			}
			paragraph := `<w:p>`
			if mergeIndex < 0 || r == mergeRow {
				cellText := ""
				if c < len(rowText) {
					cellText = rowText[c]
				}
				paragraph += `<w:r><w:t xml:space="preserve">` + wordEscapeXML(cellText) + `</w:t></w:r>`
			}
			paragraph += `</w:p>`
			b.WriteString(`<w:tc><w:tcPr>` + properties.String() + `</w:tcPr>` + paragraph + `</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}

func wordImageXML(block wordBlock, id int, relID string) string {
	name := wordEscapeXML("Picture " + strconv.Itoa(id))
	cx, cy := strconv.FormatInt(block.WidthEMU, 10), strconv.FormatInt(block.HeightEMU, 10)
	idText := strconv.Itoa(id)
	return `<w:p><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">` +
		`<wp:extent cx="` + cx + `" cy="` + cy + `"/>` +
		`<wp:effectExtent l="0" t="0" r="0" b="0"/>` +
		`<wp:docPr id="` + idText + `" name="` + name + `"/>` +
		`<wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic><pic:nvPicPr><pic:cNvPr id="` + idText + `" name="` + name + `"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="` + relID + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="` + cx + `" cy="` + cy + `"/></a:xfrm>` +
		`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic>` +
		`</a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`
}

func wordDocumentXML(blocks []wordBlock, imageRelIDs []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document ` + wordNamespaces + `><w:body>`)
	imageIndex := 0
	for _, block := range blocks {
		switch block.Kind {
		case "heading":
			b.WriteString(wordHeadingXML(block))
		case "paragraph":
			b.WriteString(wordParagraphXML(block))
		case "table":
			b.WriteString(wordTableXML(block))
		case "image":
			b.WriteString(wordImageXML(block, imageIndex+1, imageRelIDs[imageIndex]))
			imageIndex++
		case "pageBreak":
			b.WriteString(wordPageBreakXML())
		}
	}
	b.WriteString(wordSectionPropertiesXML())
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func wordStylesXML() string {
	sizes := [6]int{32, 28, 26, 24, 22, 22}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:docDefaults><w:rPrDefault><w:rPr><w:sz w:val="22"/><w:szCs w:val="22"/></w:rPr></w:rPrDefault></w:docDefaults>`)
	b.WriteString(`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>`)
	for level := 1; level <= 6; level++ {
		id := "Heading" + strconv.Itoa(level)
		b.WriteString(`<w:style w:type="paragraph" w:styleId="` + id + `"><w:name w:val="heading ` + strconv.Itoa(level) + `"/>` +
			`<w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>` +
			`<w:pPr><w:spacing w:before="240" w:after="120"/><w:outlineLvl w:val="` + strconv.Itoa(level-1) + `"/></w:pPr>` +
			`<w:rPr><w:b/><w:sz w:val="` + strconv.Itoa(sizes[level-1]) + `"/></w:rPr></w:style>`)
	}
	b.WriteString(`</w:styles>`)
	return b.String()
}

func wordContentTypesXML(hasPNG, hasJPEG bool) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	if hasPNG {
		b.WriteString(`<Default Extension="png" ContentType="image/png"/>`)
	}
	if hasJPEG {
		b.WriteString(`<Default Extension="jpeg" ContentType="image/jpeg"/>`)
	}
	b.WriteString(`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`)
	b.WriteString(`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`)
	b.WriteString(`</Types>`)
	return b.String()
}

const wordPackageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`</Relationships>`

func wordDocumentRelsXML(extensions []string) (string, []string) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	b.WriteString(`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`)
	relIDs := make([]string, len(extensions))
	for index, extension := range extensions {
		relID := "rId" + strconv.Itoa(2+index)
		relIDs[index] = relID
		b.WriteString(`<Relationship Id="` + relID + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" ` +
			`Target="media/image` + strconv.Itoa(index+1) + `.` + extension + `"/>`)
	}
	b.WriteString(`</Relationships>`)
	return b.String(), relIDs
}

func wordBuildPackage(blocks []wordBlock) ([]byte, error) {
	var images []wordBlock
	for _, block := range blocks {
		if block.Kind == "image" {
			images = append(images, block)
		}
	}
	extensions := make([]string, len(images))
	hasPNG, hasJPEG := false, false
	for index, block := range images {
		extensions[index] = block.MediaExt
		hasPNG = hasPNG || block.MediaExt == "png"
		hasJPEG = hasJPEG || block.MediaExt == "jpeg"
	}
	documentRelsXML, relIDs := wordDocumentRelsXML(extensions)
	documentXML := wordDocumentXML(blocks, relIDs)

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if err := wordWriteEntry(writer, "[Content_Types].xml", []byte(wordContentTypesXML(hasPNG, hasJPEG))); err != nil {
		return nil, err
	}
	if err := wordWriteEntry(writer, "_rels/.rels", []byte(wordPackageRelsXML)); err != nil {
		return nil, err
	}
	if err := wordWriteEntry(writer, "word/document.xml", []byte(documentXML)); err != nil {
		return nil, err
	}
	if err := wordWriteEntry(writer, "word/_rels/document.xml.rels", []byte(documentRelsXML)); err != nil {
		return nil, err
	}
	if err := wordWriteEntry(writer, "word/styles.xml", []byte(wordStylesXML())); err != nil {
		return nil, err
	}
	for index, block := range images {
		name := "word/media/image" + strconv.Itoa(index+1) + "." + block.MediaExt
		if err := wordWriteEntry(writer, name, block.Media); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func wordWriteEntry(writer *zip.Writer, name string, content []byte) error {
	part, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = part.Write(content)
	return err
}

func wordValidatePackage(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("package is empty")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("not a valid ZIP package: %w", err)
	}
	required := map[string]bool{"[Content_Types].xml": false, "_rels/.rels": false, "word/document.xml": false}
	var documentXML []byte
	for _, file := range reader.File {
		if _, known := required[file.Name]; known {
			required[file.Name] = true
		}
		if file.Name != "word/document.xml" {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			return fmt.Errorf("could not read word/document.xml: %w", err)
		}
		documentXML, err = io.ReadAll(opened)
		closeErr := opened.Close()
		if err != nil {
			return fmt.Errorf("could not read word/document.xml: %w", err)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	for name, present := range required {
		if !present {
			return fmt.Errorf("package is missing %s", name)
		}
	}
	if len(documentXML) == 0 {
		return fmt.Errorf("word/document.xml is empty")
	}
	decoder := xml.NewDecoder(bytes.NewReader(documentXML))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("word/document.xml does not parse: %w", err)
		}
	}
	return nil
}

func wordPublish(data []byte, output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-word-output-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	_, writeError := temporary.Write(data)
	syncError := temporary.Sync()
	closeError := temporary.Close()
	for _, candidate := range []error{writeError, syncError, closeError} {
		if candidate != nil {
			return candidate
		}
	}
	return os.Rename(temporaryPath, absolute)
}

func (s *Session) wordSave(receiver any, path string) {
	if path == "" {
		s.raise("WordError", "save path must not be empty")
	}
	if !strings.EqualFold(filepath.Ext(path), ".docx") {
		s.raise("WordError", "Word.save only supports a .docx destination")
	}
	blocks := s.documentBlocks(receiver)
	if message := wordRaggedTableMessage(blocks); message != "" {
		s.raise("WordError", message)
	}
	packageBytes, err := wordBuildPackage(blocks)
	if err != nil {
		s.raise("WordError", "could not assemble the DOCX package: "+err.Error())
	}
	if err := wordValidatePackage(packageBytes); err != nil {
		s.raise("WordError", "failed to produce a valid DOCX: "+err.Error())
	}
	if err := wordPublish(packageBytes, path); err != nil {
		s.raise("WordError", "could not write the destination file: "+err.Error())
	}
}

// ---------------------------------------------------------------------------
// Reading
// ---------------------------------------------------------------------------

const (
	wordMaxArchiveSize       = 64 * 1024 * 1024
	wordMaxEntryUncompressed = 32 * 1024 * 1024
	wordMaxTotalUncompressed = 128 * 1024 * 1024
	wordMaxCompressionRatio  = 200
	wordMaxEntries           = 2000
)

func (s *Session) wordReadBlocks(path string) []wordBlock {
	if path == "" {
		s.raise("WordError", "read path must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		s.raise("WordError", "could not open the DOCX file: "+err.Error())
	}
	if !info.Mode().IsRegular() {
		s.raise("WordError", "the DOCX path is not a regular file")
	}
	if info.Size() > wordMaxArchiveSize {
		s.raise("WordError", "the DOCX file is larger than the supported limit")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.raise("WordError", "could not read the DOCX file: "+err.Error())
	}
	documentXML := s.wordExtractDocumentXML(data)
	return s.wordParseDocumentXML(documentXML)
}

func (s *Session) wordExtractDocumentXML(data []byte) []byte {
	if len(data) == 0 {
		s.raise("WordError", "the DOCX file is empty")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		s.raise("WordError", "not a valid DOCX package: "+err.Error())
	}
	if len(reader.File) == 0 {
		s.raise("WordError", "the DOCX package has no members")
	}
	if len(reader.File) > wordMaxEntries {
		s.raise("WordError", "the DOCX package has too many members")
	}
	seen := make(map[string]bool, len(reader.File))
	var documentEntry *zip.File
	var totalUncompressed uint64
	for _, file := range reader.File {
		name := file.Name
		if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(name) || strings.Contains(name, "..") {
			s.raise("WordError", "the DOCX package contains an unsafe member path")
		}
		if seen[name] {
			s.raise("WordError", "the DOCX package has a duplicate member")
		}
		seen[name] = true
		if file.UncompressedSize64 > wordMaxEntryUncompressed {
			s.raise("WordError", "the DOCX package has a member that is too large")
		}
		if file.CompressedSize64 > 0 && file.UncompressedSize64/file.CompressedSize64 > wordMaxCompressionRatio {
			s.raise("WordError", "the DOCX package has a member with an unreasonable compression ratio")
		}
		totalUncompressed += file.UncompressedSize64
		if totalUncompressed > wordMaxTotalUncompressed {
			s.raise("WordError", "the DOCX package is larger than the supported limit once decompressed")
		}
		if name == "word/document.xml" {
			documentEntry = file
		}
	}
	if documentEntry == nil {
		s.raise("WordError", "the DOCX package has no word/document.xml")
	}
	opened, err := documentEntry.Open()
	if err != nil {
		s.raise("WordError", "could not read word/document.xml: "+err.Error())
	}
	defer opened.Close()
	content, err := io.ReadAll(io.LimitReader(opened, wordMaxEntryUncompressed+1))
	if err != nil {
		s.raise("WordError", "could not read word/document.xml: "+err.Error())
	}
	if int64(len(content)) > wordMaxEntryUncompressed {
		s.raise("WordError", "word/document.xml is larger than the supported limit")
	}
	if len(content) == 0 {
		s.raise("WordError", "word/document.xml is empty")
	}
	return content
}

var wordHeadingStylePattern = regexp.MustCompile(`(?i)^heading\s*([1-6])$`)

func wordHeadingLevel(styleID string) int {
	match := wordHeadingStylePattern.FindStringSubmatch(strings.TrimSpace(styleID))
	if match == nil {
		return 0
	}
	level, _ := strconv.Atoi(match[1])
	return level
}

func wordAttr(element xml.StartElement, local string) string {
	for _, attr := range element.Attr {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func (s *Session) wordParseDocumentXML(data []byte) []wordBlock {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var blocks []wordBlock
	inBody := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			s.raise("WordError", "word/document.xml does not parse: "+err.Error())
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "body":
			inBody = true
		case "p":
			if inBody {
				block := s.wordParseParagraphElement(decoder)
				if block.Kind != "" {
					blocks = append(blocks, block)
				}
			}
		case "tbl":
			if inBody {
				blocks = append(blocks, s.wordParseTableElement(decoder))
			}
		}
	}
	return blocks
}

func (s *Session) wordParseParagraphElement(decoder *xml.Decoder) wordBlock {
	styleID := ""
	var text strings.Builder
	var stack []string
	hasContent := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			s.raise("WordError", "word/document.xml does not parse: "+err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			stack = append(stack, element.Name.Local)
			if element.Name.Local == "pStyle" {
				styleID = wordAttr(element, "val")
			}
			if element.Name.Local == "t" {
				hasContent = true
			}
			if element.Name.Local == "tab" {
				hasContent = true
				text.WriteByte('\t')
			}
			if element.Name.Local == "br" && wordAttr(element, "type") != "page" {
				hasContent = true
				text.WriteByte('\n')
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return wordFinishParagraph(styleID, text.String(), hasContent)
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 && stack[len(stack)-1] == "t" {
				text.Write(element)
			}
		}
	}
	return wordFinishParagraph(styleID, text.String(), hasContent)
}

func wordFinishParagraph(styleID, text string, hasContent bool) wordBlock {
	if !hasContent {
		return wordBlock{}
	}
	if level := wordHeadingLevel(styleID); level > 0 {
		return wordBlock{Kind: "heading", Text: text, Level: level}
	}
	return wordBlock{Kind: "paragraph", Text: text, Align: "left"}
}

// wordParseTableElement consumes tokens up to and including the matching
// </w:tbl>, collecting one logical column per <w:tc> encountered, grouped by
// <w:tr>. A <w:tc> carrying <w:gridSpan w:val="N"/> expands to N logical
// columns at the position where the span occurred - the cell's own text
// followed by N-1 empty columns - so a merged header still lines up with the
// unmerged data columns beneath it. wordFinishTable then widens any row that
// is still short (a defensive fallback for markup whose spans do not fully
// account for the table's true width) so every row stays rectangular and no
// cell text is ever dropped.
func (s *Session) wordParseTableElement(decoder *xml.Decoder) wordBlock {
	var rows [][]string
	var currentRow []string
	var cellText *strings.Builder
	cellSpan := 1
	var stack []string
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			s.raise("WordError", "word/document.xml does not parse: "+err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			stack = append(stack, element.Name.Local)
			switch element.Name.Local {
			case "tr":
				currentRow = nil
			case "tc":
				cellText = &strings.Builder{}
				cellSpan = 1
			case "gridSpan":
				if cellText != nil {
					if value, err := strconv.Atoi(wordAttr(element, "val")); err == nil && value > 1 {
						cellSpan = value
					}
				}
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return wordFinishTable(rows)
			}
			closed := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if closed == "tc" && cellText != nil {
				currentRow = append(currentRow, cellText.String())
				for extra := 1; extra < cellSpan; extra++ {
					currentRow = append(currentRow, "")
				}
				cellText = nil
				cellSpan = 1
			}
			if closed == "tr" {
				rows = append(rows, currentRow)
			}
		case xml.CharData:
			if cellText != nil && len(stack) > 0 && stack[len(stack)-1] == "t" {
				cellText.Write(element)
			}
		}
	}
	return wordFinishTable(rows)
}

func wordFinishTable(rows [][]string) wordBlock {
	if len(rows) == 0 {
		return wordBlock{Kind: "table", Headers: []string{}, Align: "left"}
	}
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	widened := make([][]string, len(rows))
	for index, row := range rows {
		if len(row) == width {
			widened[index] = row
			continue
		}
		padded := make([]string, width)
		copy(padded, row)
		widened[index] = padded
	}
	return wordBlock{Kind: "table", Headers: widened[0], Rows: widened[1:], Align: "left"}
}

// wordRaggedTableMessage reports the first table block whose rows are not
// all the same logical width, or "" if every table block is rectangular.
// wordFinishTable always produces rectangular blocks and Document.table
// rejects mismatched row widths at construction time, so this exists purely
// as a save-time defensive invariant: it must never be possible to silently
// truncate a ragged table into a shorter DOCX row.
func wordRaggedTableMessage(blocks []wordBlock) string {
	for _, block := range blocks {
		if block.Kind != "table" {
			continue
		}
		width := len(block.Headers)
		for _, row := range block.Rows {
			if len(row) != width {
				return "a table has rows with different widths and cannot be saved"
			}
		}
	}
	return ""
}
