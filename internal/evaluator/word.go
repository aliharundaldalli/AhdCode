package evaluator

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"ahdcode/internal/ir"
)

// The Word standard module's REPL implementation. It mirrors the native
// backend's ahdruntime.go logic function-for-function - the same block
// encoding, the same WordprocessingML generation, the same reading limits -
// but operates on the evaluator's own List/Pair/Instance representation
// instead of AhdList/AhdPair, exactly as latex.go independently reimplements
// the Latex runtime rather than importing ahdruntime (which must stay a
// single, stdlib-only file that a generated native program embeds whole).

const wordDocumentClassID = ir.ClassID("builtin:Word::class::Document")

var wordBlocksField = ir.FieldID(string(wordDocumentClassID) + "::field::blocks")

// wordBlock is the private content-block shape, identical in spirit to the
// native runtime's ahdWordBlock: Kind selects which remaining fields matter.
type wordBlock struct {
	Kind      string     `json:"kind"`
	Text      string     `json:"text,omitempty"`
	Level     int        `json:"level,omitempty"`
	Align     string     `json:"align,omitempty"`
	Bold      bool       `json:"bold,omitempty"`
	Italic    bool       `json:"italic,omitempty"`
	Underline bool       `json:"underline,omitempty"`
	Headers   []string   `json:"headers,omitempty"`
	Rows      [][]string `json:"rows,omitempty"`
	Merges    [][4]int   `json:"merges,omitempty"`
	Media     []byte     `json:"media,omitempty"`
	MediaExt  string     `json:"mediaExt,omitempty"`
	WidthEMU  int64      `json:"widthEMU,omitempty"`
	HeightEMU int64      `json:"heightEMU,omitempty"`
}

// documentBlocks reads the one hidden storage field of a Document instance.
// The stored List is never handed out, so a caller cannot reach it to mutate.
func (s *Session) documentBlocks(value any) []wordBlock {
	instance := s.requireInstance(value)
	stored, ok := instance.Fields[wordBlocksField].(*List)
	if !ok {
		s.raise("WordError", "value is not a Document")
	}
	blocks := make([]wordBlock, len(stored.Items))
	for index, raw := range stored.Items {
		var block wordBlock
		if err := json.Unmarshal([]byte(raw.(string)), &block); err != nil {
			s.raise("WordError", "document storage is corrupted")
		}
		blocks[index] = block
	}
	return blocks
}

// documentValue materializes a block list as a new Document instance.
func (s *Session) documentValue(blocks []wordBlock) *Instance {
	items := make([]any, len(blocks))
	for index, block := range blocks {
		encoded, _ := json.Marshal(block)
		items[index] = string(encoded)
	}
	return &Instance{Class: wordDocumentClassID, Fields: map[ir.FieldID]any{
		wordBlocksField: &List{Items: items},
	}}
}

func (s *Session) wordAppend(value any, block wordBlock) *Instance {
	blocks := s.documentBlocks(value)
	next := make([]wordBlock, len(blocks)+1)
	copy(next, blocks)
	next[len(blocks)] = block
	return s.documentValue(next)
}

func (s *Session) wordBuiltin(name string, args []any) any {
	switch name {
	case "new":
		return s.documentValue(nil)
	case "read":
		return s.documentValue(s.wordReadBlocks(args[0].(string)))
	}
	s.raise("Error", "unsupported Word function "+name)
	return nil
}

var wordParagraphAlignments = map[string]bool{"left": true, "center": true, "right": true, "justify": true}
var wordTableAlignments = map[string]bool{"left": true, "center": true, "right": true}

func (s *Session) wordOperation(name string, receiver any, args []any) any {
	arg := func(index int, fallback any) any {
		if index < len(args) && args[index] != nil {
			return args[index]
		}
		return fallback
	}
	switch name {
	case "Document.heading":
		level := arg(1, int64(0)).(int64)
		if level < 1 || level > 6 {
			s.raise("WordError", "heading level must be between 1 and 6")
		}
		return s.wordAppend(receiver, wordBlock{Kind: "heading", Text: arg(0, "").(string), Level: int(level)})
	case "Document.paragraph":
		align := arg(1, "left").(string)
		if !wordParagraphAlignments[align] {
			s.raise("WordError", "paragraph align must be left, center, right, or justify")
		}
		return s.wordAppend(receiver, wordBlock{
			Kind: "paragraph", Text: arg(0, "").(string), Align: align,
			Bold: arg(2, false).(bool), Italic: arg(3, false).(bool), Underline: arg(4, false).(bool),
		})
	case "Document.table":
		return s.wordTable(receiver, args)
	case "Document.image":
		return s.wordImage(receiver, args)
	case "Document.pageBreak":
		return s.wordAppend(receiver, wordBlock{Kind: "pageBreak"})
	case "Document.save":
		s.wordSave(receiver, arg(0, "").(string))
		return Nothing
	case "Document.text":
		var lines []string
		for _, block := range s.documentBlocks(receiver) {
			if block.Kind == "heading" || block.Kind == "paragraph" {
				lines = append(lines, block.Text)
			}
		}
		return strings.Join(lines, "\n")
	case "Document.paragraphs":
		var texts []any
		for _, block := range s.documentBlocks(receiver) {
			if block.Kind == "paragraph" {
				texts = append(texts, block.Text)
			}
		}
		return &List{Items: texts}
	case "Document.headings":
		var texts []any
		for _, block := range s.documentBlocks(receiver) {
			if block.Kind == "heading" {
				texts = append(texts, block.Text)
			}
		}
		return &List{Items: texts}
	case "Document.tables":
		var tables []any
		for _, block := range s.documentBlocks(receiver) {
			if block.Kind != "table" {
				continue
			}
			rows := make([]any, 0, 1+len(block.Rows))
			rows = append(rows, wordStringListValue(block.Headers))
			for _, row := range block.Rows {
				rows = append(rows, wordStringListValue(row))
			}
			tables = append(tables, &List{Items: rows})
		}
		return &List{Items: tables}
	}
	s.raise("Error", "unsupported Document operation "+name)
	return nil
}

func wordStringListValue(values []string) *List {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return &List{Items: items}
}

func (s *Session) wordTable(receiver any, args []any) any {
	headers := s.requireList(args[0])
	rows := s.requireList(args[1])
	align := "left"
	if len(args) > 3 && args[3] != nil {
		align = args[3].(string)
	}
	if !wordTableAlignments[align] {
		s.raise("WordError", "table align must be left, center, or right")
	}
	headerValues := make([]string, len(headers.Items))
	for index, value := range headers.Items {
		headerValues[index] = value.(string)
	}
	if len(headerValues) == 0 {
		s.raise("WordError", "table requires at least one column")
	}
	grid := make([][]string, len(rows.Items))
	for index, item := range rows.Items {
		row := s.requireList(item)
		if len(row.Items) != len(headerValues) {
			s.raise("WordError", "table row column count does not match headers")
		}
		cells := make([]string, len(row.Items))
		for position, value := range row.Items {
			cells[position] = value.(string)
		}
		grid[index] = cells
	}
	var merges *List
	if len(args) > 2 && args[2] != nil {
		merges = s.requireList(args[2])
	} else {
		merges = &List{}
	}
	mergeValues := s.wordValidateMerges(merges, 1+len(grid), len(headerValues))
	return s.wordAppend(receiver, wordBlock{Kind: "table", Headers: headerValues, Rows: grid, Merges: mergeValues, Align: align})
}

func (s *Session) wordValidateMerges(merges *List, rowCount, columnCount int) [][4]int {
	result := make([][4]int, 0, len(merges.Items))
	covered := make(map[[2]int]bool)
	for _, item := range merges.Items {
		entry := s.requireList(item)
		if len(entry.Items) != 4 {
			s.raise("WordError", "a table merge descriptor must have exactly four Int values: row, column, rowSpan, columnSpan")
		}
		values := make([]int64, 4)
		for index, value := range entry.Items {
			values[index] = value.(int64)
		}
		row, column, rowSpan, columnSpan := values[0], values[1], values[2], values[3]
		if row < 0 || column < 0 {
			s.raise("WordError", "a table merge row and column must not be negative")
		}
		if rowSpan < 1 || columnSpan < 1 {
			s.raise("WordError", "a table merge rowSpan and columnSpan must be at least 1")
		}
		if rowSpan == 1 && columnSpan == 1 {
			s.raise("WordError", "a 1x1 table merge is meaningless")
		}
		if row+rowSpan > int64(rowCount) || column+columnSpan > int64(columnCount) {
			s.raise("WordError", "a table merge extends outside the table")
		}
		for r := row; r < row+rowSpan; r++ {
			for c := column; c < column+columnSpan; c++ {
				key := [2]int{int(r), int(c)}
				if covered[key] {
					s.raise("WordError", "table merge regions overlap")
				}
				covered[key] = true
			}
		}
		result = append(result, [4]int{int(row), int(column), int(rowSpan), int(columnSpan)})
	}
	return result
}

const (
	wordEMUPerCentimeter = 360000
	wordEMUPerPixel96DPI = 9525
)

func (s *Session) wordImage(receiver any, args []any) any {
	path := args[0].(string)
	if path == "" {
		s.raise("WordError", "image path must not be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.raise("WordError", "could not read image: "+err.Error())
	}
	format, naturalWidth, naturalHeight := wordDecodeImage(s, data)
	size := &Pair{Values: map[any]any{}}
	if len(args) > 1 && args[1] != nil {
		size = s.requirePair(args[1])
	}
	widthEMU, heightEMU := s.wordImageExtent(size, naturalWidth, naturalHeight)
	return s.wordAppend(receiver, wordBlock{Kind: "image", Media: data, MediaExt: format, WidthEMU: widthEMU, HeightEMU: heightEMU})
}

func wordDecodeImage(s *Session, data []byte) (string, int, int) {
	config, formatName, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		s.raise("WordError", "unsupported image format: Word supports PNG and JPEG")
	}
	switch formatName {
	case "png":
		return "png", config.Width, config.Height
	case "jpeg":
		return "jpeg", config.Width, config.Height
	default:
		s.raise("WordError", "unsupported image format: Word supports PNG and JPEG")
		return "", 0, 0
	}
}

func (s *Session) wordImageExtent(size *Pair, naturalWidth, naturalHeight int) (int64, int64) {
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
			s.raise("WordError", "image size supports only width and height")
		}
	}
	if hasWidth && width <= 0 {
		s.raise("WordError", "image width must be positive")
	}
	if hasHeight && height <= 0 {
		s.raise("WordError", "image height must be positive")
	}
	if naturalWidth <= 0 || naturalHeight <= 0 {
		naturalWidth, naturalHeight = 1, 1
	}
	aspect := float64(naturalHeight) / float64(naturalWidth)
	switch {
	case hasWidth && hasHeight:
		return int64(width * wordEMUPerCentimeter), int64(height * wordEMUPerCentimeter)
	case hasWidth:
		return int64(width * wordEMUPerCentimeter), int64(width * aspect * wordEMUPerCentimeter)
	case hasHeight:
		return int64(height / aspect * wordEMUPerCentimeter), int64(height * wordEMUPerCentimeter)
	default:
		return int64(naturalWidth) * wordEMUPerPixel96DPI, int64(naturalHeight) * wordEMUPerPixel96DPI
	}
}
