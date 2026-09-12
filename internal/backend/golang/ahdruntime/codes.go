package ahdruntime

// Machine-readable codes: QR symbols and the Code 128, EAN-13 and UPC-A
// barcodes. One encoder serves every consumer -- the QR and Barcode modules,
// Latex.qr and Latex.barcode, and PDFDocument.qr and PDFDocument.barcode -- so
// a symbol is encoded exactly once, by the pinned github.com/boombuler/barcode
// source vendored in ahdruntime/codesvendor. Everything here works on the
// logical symbol: a QR module matrix or a barcode bar pattern. PNG, SVG, and
// Latex vector output are three renderings of that same value.
//
// Like the MySQL runtime, this file imports vendored third-party code, so a
// generated program receives it only when the program actually uses codes.

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/qr"
)

// The interactive evaluator raises QRError and BarcodeError through this
// runtime, so the classes need constructors in the compiler process too. A
// generated program raises through its own generated descriptors instead.
func init() {
	for _, class := range []*AhdClass{AhdClassQRError, AhdClassBarcodeError} {
		class := class
		AhdRegisterError(class, func(message string) AhdInstance {
			instance := &ahdModuleError{message: message}
			instance.AhdSetClass(class)
			return instance
		})
	}
}

const (
	// ahdQRQuietZone is the light margin, in modules, a QR symbol needs on
	// every side to scan reliably. Every rendering includes it.
	ahdQRQuietZone = 4
	// ahdCodesMaximumPixels bounds a generated PNG side, so a mistaken
	// argument cannot allocate an unreasonable image.
	ahdCodesMaximumPixels = 10000
	// ahdCodesMaximumCentimeters bounds a vector rendering's physical size.
	ahdCodesMaximumCentimeters = 1000.0
	// ahdCode128MaximumCharacters is the longest value the Code 128 encoder
	// accepts; longer barcodes are impractical to print and scan.
	ahdCode128MaximumCharacters = 80
)

var ahdQRLevelValues = map[string]qr.ErrorCorrectionLevel{"L": qr.L, "M": qr.M, "Q": qr.Q, "H": qr.H}

// AhdBarcodeKinds is the closed set of barcode symbologies, in documentation
// order. The same names are BarcodeCode.kind() results and Latex.barcode kinds.
var AhdBarcodeKinds = []string{"Code128", "EAN13", "UPCA"}

// ahdBarcodeQuietZones is the light margin, in modules, before and after the
// bars of each symbology, from the symbology specifications.
var ahdBarcodeQuietZones = map[string][2]int{"Code128": {10, 10}, "EAN13": {11, 7}, "UPCA": {9, 9}}

// ---------------------------------------------------------------------------
// QR
// ---------------------------------------------------------------------------

// AhdQRMatrixText encodes value at level and returns the symbol's module
// matrix: rows from the top, columns from the left, true for a dark module,
// without the quiet zone. It returns a problem message instead of raising, so
// the evaluator, Latex, and PDF share it unchanged.
func AhdQRMatrixText(operation, value, level string) ([][]bool, string) {
	correction, known := ahdQRLevelValues[level]
	if !known {
		return nil, operation + " level must be L, M, Q, or H; received " + strconv.Quote(level)
	}
	if value == "" {
		return nil, operation + " value must not be empty"
	}
	if !utf8.ValidString(value) {
		return nil, operation + " value must be valid UTF-8"
	}
	code, err := qr.Encode(value, correction, qr.Auto)
	if err != nil {
		return nil, operation + " value of " + strconv.Itoa(len(value)) +
			" bytes does not fit in the largest QR symbol at level " + level
	}
	bounds := code.Bounds()
	matrix := make([][]bool, bounds.Dy())
	for row := range matrix {
		matrix[row] = make([]bool, bounds.Dx())
		for column := range matrix[row] {
			matrix[row][column] = ahdCodesDark(code.At(bounds.Min.X+column, bounds.Min.Y+row))
		}
	}
	return matrix, ""
}

func ahdCodesDark(value color.Color) bool {
	return color.GrayModel.Convert(value).(color.Gray).Y < 128
}

// AhdQRCreate validates value and level by encoding them once, and returns
// the QRCode storage: the level letter followed by the value.
func AhdQRCreate(errorClass *AhdClass, value, level string) string {
	if _, problem := AhdQRMatrixText("QR.create", value, level); problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	return level + value
}

func ahdQRParts(errorClass *AhdClass, data string) (value, level string) {
	if len(data) < 2 {
		AhdRaiseClass(errorClass, "QRCode storage is corrupted")
	}
	if _, known := ahdQRLevelValues[data[:1]]; !known {
		AhdRaiseClass(errorClass, "QRCode storage is corrupted")
	}
	return data[1:], data[:1]
}

func ahdQRStoredMatrix(errorClass *AhdClass, data string) [][]bool {
	value, level := ahdQRParts(errorClass, data)
	matrix, problem := AhdQRMatrixText("QRCode", value, level)
	if problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	return matrix
}

func AhdQRValue(errorClass *AhdClass, data string) string {
	value, _ := ahdQRParts(errorClass, data)
	return value
}

func AhdQRLevel(errorClass *AhdClass, data string) string {
	_, level := ahdQRParts(errorClass, data)
	return level
}

func AhdQRSize(errorClass *AhdClass, data string) int64 {
	return int64(len(ahdQRStoredMatrix(errorClass, data)))
}

// AhdQRMatrix returns a fresh List<List<Bool>>; a caller mutating it cannot
// change the QRCode, which re-encodes from its value on every call.
func AhdQRMatrix(errorClass *AhdClass, data string) *AhdList[*AhdList[bool]] {
	matrix := ahdQRStoredMatrix(errorClass, data)
	rows := make([]*AhdList[bool], len(matrix))
	for index, row := range matrix {
		rows[index] = AhdNewList(row...)
	}
	return AhdNewList(rows...)
}

// AhdQRPNGBytes renders matrix, with its quiet zone, as a black-on-white PNG
// of exactly pixels by pixels. Every module is a whole number of pixels, so
// module edges stay crisp; pixels left over after the largest whole module
// size widen the quiet zone evenly.
func AhdQRPNGBytes(operation string, matrix [][]bool, pixels int64) ([]byte, string) {
	total := len(matrix) + 2*ahdQRQuietZone
	if pixels < int64(total) || pixels > ahdCodesMaximumPixels {
		return nil, operation + " pixels must be between " + strconv.Itoa(total) + " and " +
			strconv.Itoa(ahdCodesMaximumPixels) + " for this symbol; received " + strconv.FormatInt(pixels, 10)
	}
	side := int(pixels)
	module := side / total
	offset := (side-module*total)/2 + ahdQRQuietZone*module
	picture := ahdCodesBlankImage(side, side)
	for row, cells := range matrix {
		for _, run := range ahdCodesRuns(cells) {
			ahdCodesFill(picture, offset+run[0]*module, offset+row*module, run[1]*module, module)
		}
	}
	return ahdCodesEncodePNG(picture), ""
}

// AhdQRSVGBytes renders matrix, with its quiet zone, as a square vector SVG
// of size centimeters: a white background and one path of dark module runs.
func AhdQRSVGBytes(operation string, matrix [][]bool, size float64) ([]byte, string) {
	if problem := ahdCodesCentimeters(operation, "size", size); problem != "" {
		return nil, problem
	}
	total := strconv.Itoa(len(matrix) + 2*ahdQRQuietZone)
	var result strings.Builder
	result.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	result.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="` + ahdFormatReal(size) + `cm" height="` +
		ahdFormatReal(size) + `cm" viewBox="0 0 ` + total + " " + total + `" shape-rendering="crispEdges">` + "\n")
	result.WriteString(`<rect width="` + total + `" height="` + total + `" fill="#FFFFFF"/>` + "\n")
	result.WriteString(`<path fill="#000000" d="`)
	for row, cells := range matrix {
		for _, run := range ahdCodesRuns(cells) {
			result.WriteString("M" + strconv.Itoa(run[0]+ahdQRQuietZone) + " " + strconv.Itoa(row+ahdQRQuietZone) +
				"h" + strconv.Itoa(run[1]) + "v1h-" + strconv.Itoa(run[1]) + "z")
		}
	}
	result.WriteString(`"/>` + "\n</svg>\n")
	return []byte(result.String()), ""
}

func AhdQRSavePNG(errorClass *AhdClass, data, path string, pixels int64) {
	ahdCodesRequirePath(errorClass, "QRCode.savePNG", path, ".png")
	encoded, problem := AhdQRPNGBytes("QRCode.savePNG", ahdQRStoredMatrix(errorClass, data), pixels)
	if problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	ahdCodesPublish(errorClass, "QRCode.savePNG", path, encoded)
}

func AhdQRSaveSVG(errorClass *AhdClass, data, path string, size float64) {
	ahdCodesRequirePath(errorClass, "QRCode.saveSVG", path, ".svg")
	encoded, problem := AhdQRSVGBytes("QRCode.saveSVG", ahdQRStoredMatrix(errorClass, data), size)
	if problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	ahdCodesPublish(errorClass, "QRCode.saveSVG", path, encoded)
}

// ---------------------------------------------------------------------------
// Barcode
// ---------------------------------------------------------------------------

// AhdBarcodePatternText encodes value as kind and returns the logical bar
// pattern (true for a dark module, without quiet zones) and the value the
// symbol carries: a Code 128 value unchanged, an EAN-13 or UPC-A value with
// its check digit. A problem message is returned instead of raising.
func AhdBarcodePatternText(operation, kind, value string) ([]bool, string, string) {
	switch kind {
	case "Code128":
		if value == "" {
			return nil, "", operation + " value must not be empty"
		}
		if count := utf8.RuneCountInString(value); count > ahdCode128MaximumCharacters {
			return nil, "", operation + " value must be at most " + strconv.Itoa(ahdCode128MaximumCharacters) +
				" characters; received " + strconv.Itoa(count)
		}
		for index, character := range []rune(value) {
			if character > 127 {
				return nil, "", operation + " value must contain only ASCII characters; received " +
					strconv.QuoteRune(character) + " at position " + strconv.Itoa(index+1)
			}
		}
		code, err := code128.Encode(value)
		if err != nil {
			return nil, "", operation + " could not encode " + strconv.Quote(value) + " as Code 128"
		}
		return ahdCodesPattern(code), value, ""
	case "EAN13", "UPCA":
		data := 12
		if kind == "UPCA" {
			data = 11
		}
		for _, character := range value {
			if character < '0' || character > '9' {
				return nil, "", operation + " value must contain only the digits 0-9; received " + strconv.Quote(value)
			}
		}
		full := value
		switch len(value) {
		case data:
			full = value + string(ahdBarcodeCheckDigit(value))
		case data + 1:
			if expected := ahdBarcodeCheckDigit(value[:data]); value[data] != expected {
				return nil, "", operation + " check digit " + value[data:] + " is wrong for " + value[:data] +
					"; the correct check digit is " + string(expected)
			}
		default:
			return nil, "", operation + " value must have " + strconv.Itoa(data) + " digits (the check digit is computed) or " +
				strconv.Itoa(data+1) + " digits (the check digit is verified); received " + strconv.Itoa(len(value))
		}
		symbol := full
		if kind == "UPCA" {
			// A UPC-A symbol is the EAN-13 symbol with a leading zero; the bars
			// are identical, which is why every EAN-13 scanner reads UPC-A.
			symbol = "0" + full
		}
		code, err := ean.Encode(symbol)
		if err != nil {
			return nil, "", operation + " could not encode " + strconv.Quote(full)
		}
		return ahdCodesPattern(code), full, ""
	default:
		return nil, "", operation + " kind must be Code128, EAN13, or UPCA; received " + strconv.Quote(kind)
	}
}

// ahdBarcodeCheckDigit is the GS1 modulo-10 check digit shared by EAN-13 and
// UPC-A: data digits are weighted 3, 1, 3, ... starting from the rightmost.
func ahdBarcodeCheckDigit(data string) byte {
	sum := 0
	for index := 0; index < len(data); index++ {
		digit := int(data[len(data)-1-index] - '0')
		if index%2 == 0 {
			sum += 3 * digit
		} else {
			sum += digit
		}
	}
	return byte('0' + (10-sum%10)%10)
}

func ahdCodesPattern(code barcode.Barcode) []bool {
	bounds := code.Bounds()
	pattern := make([]bool, bounds.Dx())
	for index := range pattern {
		pattern[index] = ahdCodesDark(code.At(bounds.Min.X+index, bounds.Min.Y))
	}
	return pattern
}

var ahdBarcodeOperations = map[string]string{"Code128": "Barcode.code128", "EAN13": "Barcode.ean13", "UPCA": "Barcode.upca"}

// AhdBarcodeCreate validates value for kind and returns the BarcodeCode
// storage: the kind, a colon, and the value the symbol carries.
func AhdBarcodeCreate(errorClass *AhdClass, kind, value string) string {
	_, full, problem := AhdBarcodePatternText(ahdBarcodeOperations[kind], kind, value)
	if problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	return kind + ":" + full
}

func ahdBarcodeParts(errorClass *AhdClass, data string) (kind, value string) {
	separator := strings.IndexByte(data, ':')
	if separator < 0 {
		AhdRaiseClass(errorClass, "BarcodeCode storage is corrupted")
	}
	kind = data[:separator]
	if _, known := ahdBarcodeQuietZones[kind]; !known {
		AhdRaiseClass(errorClass, "BarcodeCode storage is corrupted")
	}
	return kind, data[separator+1:]
}

func ahdBarcodeStoredPattern(errorClass *AhdClass, data string) (string, []bool) {
	kind, value := ahdBarcodeParts(errorClass, data)
	pattern, _, problem := AhdBarcodePatternText("BarcodeCode", kind, value)
	if problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	return kind, pattern
}

func AhdBarcodeKind(errorClass *AhdClass, data string) string {
	kind, _ := ahdBarcodeParts(errorClass, data)
	return kind
}

func AhdBarcodeValue(errorClass *AhdClass, data string) string {
	_, value := ahdBarcodeParts(errorClass, data)
	return value
}

func AhdBarcodePattern(errorClass *AhdClass, data string) *AhdList[bool] {
	_, pattern := ahdBarcodeStoredPattern(errorClass, data)
	return AhdNewList(pattern...)
}

// AhdBarcodePNGBytes renders pattern, with its quiet zones, as black bars on
// white in an image of exactly width by height pixels. Every module is a
// whole number of pixels wide; leftover pixels widen the quiet zones evenly.
func AhdBarcodePNGBytes(operation, kind string, pattern []bool, width, height int64) ([]byte, string) {
	quiet := ahdBarcodeQuietZones[kind]
	total := quiet[0] + len(pattern) + quiet[1]
	if width < int64(total) || width > ahdCodesMaximumPixels {
		return nil, operation + " width must be between " + strconv.Itoa(total) + " and " +
			strconv.Itoa(ahdCodesMaximumPixels) + " pixels for this barcode; received " + strconv.FormatInt(width, 10)
	}
	if height < 1 || height > ahdCodesMaximumPixels {
		return nil, operation + " height must be between 1 and " + strconv.Itoa(ahdCodesMaximumPixels) +
			" pixels; received " + strconv.FormatInt(height, 10)
	}
	module := int(width) / total
	offset := (int(width)-module*total)/2 + quiet[0]*module
	picture := ahdCodesBlankImage(int(width), int(height))
	for _, run := range ahdCodesRuns(pattern) {
		ahdCodesFill(picture, offset+run[0]*module, 0, run[1]*module, int(height))
	}
	return ahdCodesEncodePNG(picture), ""
}

// AhdBarcodeSVGBytes renders pattern, with its quiet zones, as a vector SVG
// width by height centimeters. The viewBox is one unit per module and one
// unit tall, stretched to the requested size.
func AhdBarcodeSVGBytes(operation, kind string, pattern []bool, width, height float64) ([]byte, string) {
	if problem := ahdCodesCentimeters(operation, "width", width); problem != "" {
		return nil, problem
	}
	if problem := ahdCodesCentimeters(operation, "height", height); problem != "" {
		return nil, problem
	}
	quiet := ahdBarcodeQuietZones[kind]
	total := strconv.Itoa(quiet[0] + len(pattern) + quiet[1])
	var result strings.Builder
	result.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	result.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="` + ahdFormatReal(width) + `cm" height="` +
		ahdFormatReal(height) + `cm" viewBox="0 0 ` + total + ` 1" preserveAspectRatio="none" shape-rendering="crispEdges">` + "\n")
	result.WriteString(`<rect width="` + total + `" height="1" fill="#FFFFFF"/>` + "\n")
	result.WriteString(`<path fill="#000000" d="`)
	for _, run := range ahdCodesRuns(pattern) {
		result.WriteString("M" + strconv.Itoa(run[0]+quiet[0]) + " 0h" + strconv.Itoa(run[1]) + "v1h-" + strconv.Itoa(run[1]) + "z")
	}
	result.WriteString(`"/>` + "\n</svg>\n")
	return []byte(result.String()), ""
}

func AhdBarcodeSavePNG(errorClass *AhdClass, data, path string, width, height int64) {
	ahdCodesRequirePath(errorClass, "BarcodeCode.savePNG", path, ".png")
	kind, pattern := ahdBarcodeStoredPattern(errorClass, data)
	encoded, problem := AhdBarcodePNGBytes("BarcodeCode.savePNG", kind, pattern, width, height)
	if problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	ahdCodesPublish(errorClass, "BarcodeCode.savePNG", path, encoded)
}

func AhdBarcodeSaveSVG(errorClass *AhdClass, data, path string, width, height float64) {
	ahdCodesRequirePath(errorClass, "BarcodeCode.saveSVG", path, ".svg")
	kind, pattern := ahdBarcodeStoredPattern(errorClass, data)
	encoded, problem := AhdBarcodeSVGBytes("BarcodeCode.saveSVG", kind, pattern, width, height)
	if problem != "" {
		AhdRaiseClass(errorClass, problem)
	}
	ahdCodesPublish(errorClass, "BarcodeCode.saveSVG", path, encoded)
}

// ---------------------------------------------------------------------------
// Vector renderings for Latex and PDF
// ---------------------------------------------------------------------------

// AhdLatexQRText builds a vector QR fragment size centimeters square, quiet
// zone included, from the shared encoder: a white square and one filled path
// of dark module runs drawn in TikZ. Nothing is rasterized. The leading empty
// comment line keeps the TikZ marker at the start of a line even when the
// fragment is appended to text.
func AhdLatexQRText(operation, value string, size float64, level string) (string, string) {
	if problem := ahdCodesCentimeters(operation, "size", size); problem != "" {
		return "", problem
	}
	matrix, problem := AhdQRMatrixText(operation, value, level)
	if problem != "" {
		return "", problem
	}
	total := len(matrix) + 2*ahdQRQuietZone
	var source strings.Builder
	unit := ahdCodesUnit(size / float64(total))
	source.WriteString("\\begin{scope}[x=" + unit + "cm,y=" + unit + "cm]\n")
	source.WriteString("\\fill[white] (0,0) rectangle (" + strconv.Itoa(total) + "," + strconv.Itoa(total) + ");\n")
	source.WriteString("\\fill[black]")
	for row, cells := range matrix {
		top := total - ahdQRQuietZone - row
		for _, run := range ahdCodesRuns(cells) {
			left := run[0] + ahdQRQuietZone
			source.WriteString(" (" + strconv.Itoa(left) + "," + strconv.Itoa(top-1) + ") rectangle (" +
				strconv.Itoa(left+run[1]) + "," + strconv.Itoa(top) + ")")
		}
	}
	source.WriteString(";\n\\end{scope}\n")
	text, problem := AhdLatexTikZText(operation, source.String(), nil, false)
	if problem != "" {
		return "", problem
	}
	return "%\n" + text, ""
}

// AhdLatexBarcodeText builds a vector barcode fragment width by height
// centimeters, quiet zones included, from the shared encoder.
func AhdLatexBarcodeText(operation, kind, value string, width, height float64) (string, string) {
	if problem := ahdCodesCentimeters(operation, "width", width); problem != "" {
		return "", problem
	}
	if problem := ahdCodesCentimeters(operation, "height", height); problem != "" {
		return "", problem
	}
	pattern, _, problem := AhdBarcodePatternText(operation, kind, value)
	if problem != "" {
		return "", problem
	}
	quiet := ahdBarcodeQuietZones[kind]
	total := quiet[0] + len(pattern) + quiet[1]
	var source strings.Builder
	source.WriteString("\\begin{scope}[x=" + ahdCodesUnit(width/float64(total)) + "cm,y=" + ahdCodesUnit(height) + "cm]\n")
	source.WriteString("\\fill[white] (0,0) rectangle (" + strconv.Itoa(total) + ",1);\n")
	source.WriteString("\\fill[black]")
	for _, run := range ahdCodesRuns(pattern) {
		left := run[0] + quiet[0]
		source.WriteString(" (" + strconv.Itoa(left) + ",0) rectangle (" + strconv.Itoa(left+run[1]) + ",1)")
	}
	source.WriteString(";\n\\end{scope}\n")
	text, problem := AhdLatexTikZText(operation, source.String(), nil, false)
	if problem != "" {
		return "", problem
	}
	return "%\n" + text, ""
}

// AhdLatexQR is the native entry point of Latex.qr. Latex reports invalid
// input as ValueError, like every other Latex helper.
func AhdLatexQR(value string, size float64, level string) string {
	text, problem := AhdLatexQRText("Latex.qr", value, size, level)
	if problem != "" {
		AhdRaiseClass(AhdClassValueError, problem)
	}
	return text
}

// AhdLatexBarcode is the native entry point of Latex.barcode.
func AhdLatexBarcode(kind, value string, width, height float64) string {
	text, problem := AhdLatexBarcodeText("Latex.barcode", kind, value, width, height)
	if problem != "" {
		AhdRaiseClass(AhdClassValueError, problem)
	}
	return text
}

// AhdPDFQRBlock is PDFDocument.qr: a vector QR block built by the shared
// encoder. The block stores generated vector source, never caller text.
func AhdPDFQRBlock(value string, size float64, level, align string) (string, string) {
	if problem := ahdPDFBlockAlign("PDFDocument.qr", align); problem != "" {
		return "", problem
	}
	text, problem := AhdLatexQRText("PDFDocument.qr", value, size, level)
	if problem != "" {
		return "", problem
	}
	return ahdPDFBlockText(ahdPDFBlock{Kind: "vector", Text: text, Align: align}), ""
}

// AhdPDFBarcodeBlock is PDFDocument.barcode: a vector barcode block built by
// the shared encoder.
func AhdPDFBarcodeBlock(kind, value string, width, height float64, align string) (string, string) {
	if problem := ahdPDFBlockAlign("PDFDocument.barcode", align); problem != "" {
		return "", problem
	}
	text, problem := AhdLatexBarcodeText("PDFDocument.barcode", kind, value, width, height)
	if problem != "" {
		return "", problem
	}
	return ahdPDFBlockText(ahdPDFBlock{Kind: "vector", Text: text, Align: align}), ""
}

// AhdPDFQR is the native entry point of PDFDocument.qr.
func AhdPDFQR(doc AhdPDFDocument, value string, size float64, level, align string) AhdPDFDocument {
	text, problem := AhdPDFQRBlock(value, size, level, align)
	return ahdPDFAppendText(doc, text, problem)
}

// AhdPDFBarcode is the native entry point of PDFDocument.barcode.
func AhdPDFBarcode(doc AhdPDFDocument, kind, value string, width, height float64, align string) AhdPDFDocument {
	text, problem := AhdPDFBarcodeBlock(kind, value, width, height, align)
	return ahdPDFAppendText(doc, text, problem)
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// ahdCodesRuns returns each maximal run of dark modules as (start, length).
func ahdCodesRuns(cells []bool) [][2]int {
	var runs [][2]int
	for index := 0; index < len(cells); {
		if !cells[index] {
			index++
			continue
		}
		start := index
		for index < len(cells) && cells[index] {
			index++
		}
		runs = append(runs, [2]int{start, index - start})
	}
	return runs
}

// ahdCodesUnit formats one module's size in centimeters with micrometer
// precision, so generated source stays short and deterministic.
func ahdCodesUnit(value float64) string {
	return strconv.FormatFloat(math.Round(value*1e6)/1e6, 'f', -1, 64)
}

func ahdCodesCentimeters(operation, name string, value float64) string {
	if math.IsNaN(value) || value <= 0 || value > ahdCodesMaximumCentimeters {
		return operation + " " + name + " must be greater than 0 and at most " +
			ahdFormatReal(ahdCodesMaximumCentimeters) + " centimeters; received " + ahdFormatReal(value)
	}
	return ""
}

func ahdCodesBlankImage(width, height int) *image.Gray {
	picture := image.NewGray(image.Rect(0, 0, width, height))
	for index := range picture.Pix {
		picture.Pix[index] = 0xFF
	}
	return picture
}

func ahdCodesFill(picture *image.Gray, left, top, width, height int) {
	for y := top; y < top+height; y++ {
		row := picture.Pix[y*picture.Stride : (y+1)*picture.Stride]
		for x := left; x < left+width; x++ {
			row[x] = 0
		}
	}
}

func ahdCodesEncodePNG(picture image.Image) []byte {
	var buffer bytes.Buffer
	// Encoding an in-memory Gray image cannot fail.
	_ = png.Encode(&buffer, picture)
	return buffer.Bytes()
}

func ahdCodesRequirePath(errorClass *AhdClass, operation, path, extension string) {
	if path == "" {
		AhdRaiseClass(errorClass, operation+" path must not be empty")
	}
	if !strings.EqualFold(filepath.Ext(path), extension) {
		AhdRaiseClass(errorClass, operation+" destination must use the "+extension+" extension")
	}
}

// ahdCodesPublish verifies the encoded bytes are the format the destination
// names, then publishes them with the same temporary-file-and-rename step
// Latex output uses, so a failure never leaves a partial file or replaces an
// existing destination.
func ahdCodesPublish(errorClass *AhdClass, operation, path string, encoded []byte) {
	if strings.EqualFold(filepath.Ext(path), ".png") {
		if _, err := png.DecodeConfig(bytes.NewReader(encoded)); err != nil {
			AhdRaiseClass(errorClass, operation+" produced an invalid PNG: "+err.Error())
		}
	} else {
		decoder := xml.NewDecoder(bytes.NewReader(encoded))
		for {
			if _, err := decoder.Token(); err != nil {
				if err == io.EOF {
					break
				}
				AhdRaiseClass(errorClass, operation+" produced invalid SVG: "+err.Error())
			}
		}
	}
	if err := ahdLatexPublishBytes(encoded, path); err != nil {
		AhdRaiseClass(errorClass, operation+" could not write "+path+": "+err.Error())
	}
}
