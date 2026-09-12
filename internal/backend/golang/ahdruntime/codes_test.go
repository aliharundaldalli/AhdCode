package ahdruntime

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// codesRaised runs fn, requires it to raise through AhdRaiseClass, and returns
// the message.
func codesRaised(t *testing.T, fn func()) (message string) {
	t.Helper()
	defer func() {
		signal, ok := recover().(*AhdSignal)
		if !ok {
			t.Fatalf("expected a raised AhdCode error, got %#v", signal)
		}
		message = signal.Message
	}()
	fn()
	return ""
}

func patternText(pattern []bool) string {
	var result strings.Builder
	for _, dark := range pattern {
		if dark {
			result.WriteByte('1')
		} else {
			result.WriteByte('0')
		}
	}
	return result.String()
}

// ean13Reference is an independent EAN-13 encoder written from the GS1
// General Specifications tables, so the vendored encoder is checked against
// the standard rather than against itself.
func ean13Reference(digits string) string {
	odd := []string{"0001101", "0011001", "0010011", "0111101", "0100011", "0110001", "0101111", "0111011", "0110111", "0001011"}
	even := []string{"0100111", "0110011", "0011011", "0100001", "0011101", "0111001", "0000101", "0010001", "0001001", "0010111"}
	right := []string{"1110010", "1100110", "1101100", "1000010", "1011100", "1001110", "1010000", "1000100", "1001000", "1110100"}
	parity := []string{"LLLLLL", "LLGLGG", "LLGGLG", "LLGGGL", "LGLLGG", "LGGLLG", "LGGGLL", "LGLGLG", "LGLGGL", "LGGLGL"}
	var result strings.Builder
	result.WriteString("101")
	first := digits[0] - '0'
	for index := 1; index <= 6; index++ {
		digit := digits[index] - '0'
		if parity[first][index-1] == 'L' {
			result.WriteString(odd[digit])
		} else {
			result.WriteString(even[digit])
		}
	}
	result.WriteString("01010")
	for index := 7; index <= 12; index++ {
		result.WriteString(right[digits[index]-'0'])
	}
	result.WriteString("101")
	return result.String()
}

func TestBarcodeCheckDigitsMatchPublishedExamples(t *testing.T) {
	cases := []struct{ kind, value, want string }{
		{"EAN13", "590123412345", "5901234123457"},
		{"EAN13", "400638133393", "4006381333931"},
		{"EAN13", "978030640615", "9780306406157"},
		{"EAN13", "9780306406157", "9780306406157"},
		{"UPCA", "03600029145", "036000291452"},
		{"UPCA", "01234567890", "012345678905"},
		{"UPCA", "012345678905", "012345678905"},
	}
	for _, test := range cases {
		_, full, problem := AhdBarcodePatternText("test", test.kind, test.value)
		if problem != "" || full != test.want {
			t.Fatalf("%s %s = %q (%s), want %q", test.kind, test.value, full, problem, test.want)
		}
	}
}

func TestEAN13AndUPCAPatternsMatchGS1Reference(t *testing.T) {
	for _, value := range []string{"5901234123457", "4006381333931", "9780306406157", "0000000000000"} {
		pattern, _, problem := AhdBarcodePatternText("test", "EAN13", value)
		if problem != "" {
			t.Fatal(problem)
		}
		if got, want := patternText(pattern), ean13Reference(value); got != want {
			t.Fatalf("EAN-13 %s pattern\n got %s\nwant %s", value, got, want)
		}
	}
	// UPC-A is the EAN-13 symbol with a leading zero: identical bars.
	pattern, full, problem := AhdBarcodePatternText("test", "UPCA", "03600029145")
	if problem != "" || full != "036000291452" {
		t.Fatalf("UPC-A = %q %q", full, problem)
	}
	if got, want := patternText(pattern), ean13Reference("0"+full); got != want {
		t.Fatalf("UPC-A pattern\n got %s\nwant %s", got, want)
	}
}

func TestCode128PatternFraming(t *testing.T) {
	for _, value := range []string{"AHD-ASSET-00042", "1234567890", "Mixed 123 text/ok", "a"} {
		pattern, full, problem := AhdBarcodePatternText("test", "Code128", value)
		if problem != "" || full != value {
			t.Fatalf("Code128 %q: %q %s", value, full, problem)
		}
		text := patternText(pattern)
		starts := []string{"11010000100", "11010010000", "11010011100"}
		if !strings.HasPrefix(text, starts[0]) && !strings.HasPrefix(text, starts[1]) && !strings.HasPrefix(text, starts[2]) {
			t.Fatalf("Code128 %q does not begin with a start code: %s", value, text[:11])
		}
		if !strings.HasSuffix(text, "1100011101011") || (len(text)-13)%11 != 0 {
			t.Fatalf("Code128 %q does not end with the stop pattern on an 11-module grid: %s", value, text)
		}
	}
}

func TestBarcodeRejectsInvalidInput(t *testing.T) {
	cases := []struct{ kind, value, message string }{
		{"EAN13", "59012341234", "test value must have 12 digits (the check digit is computed) or 13 digits (the check digit is verified); received 11"},
		{"EAN13", "5901234123458", "test check digit 8 is wrong for 590123412345; the correct check digit is 7"},
		{"EAN13", "59012341234A", `test value must contain only the digits 0-9; received "59012341234A"`},
		{"EAN13", "", "test value must have 12 digits (the check digit is computed) or 13 digits (the check digit is verified); received 0"},
		{"UPCA", "036000291453", "test check digit 3 is wrong for 03600029145; the correct check digit is 2"},
		{"UPCA", "0360002914", "test value must have 11 digits (the check digit is computed) or 12 digits (the check digit is verified); received 10"},
		{"UPCA", "٠٣٦٠٠٠٢٩١٤٥", `test value must contain only the digits 0-9; received "٠٣٦٠٠٠٢٩١٤٥"`},
		{"Code128", "", "test value must not be empty"},
		{"Code128", "Ayşe", `test value must contain only ASCII characters; received 'ş' at position 3`},
		{"Code128", strings.Repeat("A", 81), "test value must be at most 80 characters; received 81"},
		{"QR", "123", `test kind must be Code128, EAN13, or UPCA; received "QR"`},
	}
	for _, test := range cases {
		if _, _, problem := AhdBarcodePatternText("test", test.kind, test.value); problem != test.message {
			t.Fatalf("%s %q problem\n got %q\nwant %q", test.kind, test.value, problem, test.message)
		}
	}
	message := codesRaised(t, func() { AhdBarcodeCreate(AhdClassBarcodeError, "EAN13", "5901234123458") })
	if !strings.HasPrefix(message, "Barcode.ean13 check digit 8") {
		t.Fatalf("BarcodeCreate names its operation: %q", message)
	}
}

// A QR matrix is the exact symbol: square, with the three finder patterns,
// the timing patterns, and the always-dark module ISO/IEC 18004 places at
// (row 4*version+9, column 8).
func TestQRMatrixCarriesTheFixedStructure(t *testing.T) {
	for _, level := range []string{"L", "M", "Q", "H"} {
		matrix, problem := AhdQRMatrixText("test", "https://ahdcode.org/verify?id=42", level)
		if problem != "" {
			t.Fatal(problem)
		}
		size := len(matrix)
		if (size-17)%4 != 0 || size < 21 {
			t.Fatalf("level %s size %d is not a QR version size", level, size)
		}
		for _, row := range matrix {
			if len(row) != size {
				t.Fatalf("level %s matrix is not square", level)
			}
		}
		finder := func(top, left int) {
			for r := 0; r < 7; r++ {
				for c := 0; c < 7; c++ {
					ring := r == 0 || r == 6 || c == 0 || c == 6
					core := r >= 2 && r <= 4 && c >= 2 && c <= 4
					if matrix[top+r][left+c] != (ring || core) {
						t.Fatalf("level %s finder at (%d,%d) wrong at (%d,%d)", level, top, left, r, c)
					}
				}
			}
		}
		finder(0, 0)
		finder(0, size-7)
		finder(size-7, 0)
		for index := 8; index < size-8; index++ {
			if matrix[6][index] != (index%2 == 0) || matrix[index][6] != (index%2 == 0) {
				t.Fatalf("level %s timing pattern broken at %d", level, index)
			}
		}
		version := (size - 17) / 4
		if !matrix[4*version+9][8] {
			t.Fatalf("level %s dark module missing", level)
		}
	}
	small, _ := AhdQRMatrixText("test", "1234567890", "L")
	if len(small) != 21 {
		t.Fatalf("a ten-digit numeric value should fit version 1, got size %d", len(small))
	}
}

func TestQRRejectsInvalidInput(t *testing.T) {
	cases := []struct{ value, level, message string }{
		{"x", "m", `test level must be L, M, Q, or H; received "m"`},
		{"x", "", `test level must be L, M, Q, or H; received ""`},
		{"", "M", "test value must not be empty"},
		{strings.Repeat("é", 1500), "H", "test value of 3000 bytes does not fit in the largest QR symbol at level H"},
		{"\xff", "M", "test value must be valid UTF-8"},
	}
	for _, test := range cases {
		if _, problem := AhdQRMatrixText("test", test.value, test.level); problem != test.message {
			t.Fatalf("QR %q/%q problem\n got %q\nwant %q", test.value, test.level, problem, test.message)
		}
	}
	if message := codesRaised(t, func() { AhdQRCreate(AhdClassQRError, "x", "Z") }); !strings.HasPrefix(message, "QR.create level") {
		t.Fatalf("QRCreate names its operation: %q", message)
	}
}

func TestQRPNGIsCrispExactAndDeterministic(t *testing.T) {
	matrix, _ := AhdQRMatrixText("test", "AhdCode", "M")
	first, problem := AhdQRPNGBytes("test", matrix, 512)
	if problem != "" {
		t.Fatal(problem)
	}
	second, _ := AhdQRPNGBytes("test", matrix, 512)
	if !bytes.Equal(first, second) {
		t.Fatal("QR PNG output is not deterministic")
	}
	if !bytes.HasPrefix(first, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatal("QR PNG lacks the PNG signature")
	}
	decoded, err := png.Decode(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != image.Rect(0, 0, 512, 512) {
		t.Fatalf("QR PNG bounds %v, want 512x512", decoded.Bounds())
	}
	gray := decoded.(*image.Gray)
	for _, value := range gray.Pix {
		if value != 0 && value != 255 {
			t.Fatalf("QR PNG has an interpolated gray %d", value)
		}
	}
	// The quiet zone is light: the outer four modules of every edge.
	module := 512 / (len(matrix) + 8)
	for index := 0; index < 4*module; index++ {
		if gray.GrayAt(index, index).Y != 255 {
			t.Fatalf("quiet zone is not light at %d", index)
		}
	}
	total := len(matrix) + 8
	if _, problem := AhdQRPNGBytes("test", matrix, int64(total-1)); problem != "test pixels must be between 29 and 10000 for this symbol; received 28" {
		t.Fatalf("minimum pixels problem: %q", problem)
	}
	if _, problem := AhdQRPNGBytes("test", matrix, 0); problem == "" {
		t.Fatal("zero pixels accepted")
	}
	if _, problem := AhdQRPNGBytes("test", matrix, 10001); problem == "" {
		t.Fatal("oversized PNG accepted")
	}
}

func TestCodesSVGIsVectorXML(t *testing.T) {
	matrix, _ := AhdQRMatrixText("test", "AhdCode", "M")
	qrSVG, problem := AhdQRSVGBytes("test", matrix, 5)
	if problem != "" {
		t.Fatal(problem)
	}
	pattern, _, _ := AhdBarcodePatternText("test", "EAN13", "590123412345")
	barSVG, problem := AhdBarcodeSVGBytes("test", "EAN13", pattern, 8, 2.4)
	if problem != "" {
		t.Fatal(problem)
	}
	for name, svg := range map[string][]byte{"qr": qrSVG, "barcode": barSVG} {
		var root struct {
			XMLName xml.Name
			Width   string `xml:"width,attr"`
			ViewBox string `xml:"viewBox,attr"`
			Paths   []struct {
				D string `xml:"d,attr"`
			} `xml:"path"`
		}
		if err := xml.Unmarshal(svg, &root); err != nil {
			t.Fatalf("%s SVG is not XML: %v", name, err)
		}
		if root.XMLName.Local != "svg" || len(root.Paths) != 1 || root.Paths[0].D == "" {
			t.Fatalf("%s SVG shape wrong: %+v", name, root)
		}
		if strings.Contains(string(svg), "<image") || strings.Contains(string(svg), "base64") {
			t.Fatalf("%s SVG embeds a raster image", name)
		}
	}
	if !strings.Contains(string(qrSVG), `width="5.0cm" height="5.0cm" viewBox="0 0 29 29"`) {
		t.Fatalf("QR SVG dimensions: %s", qrSVG[:200])
	}
	if !strings.Contains(string(barSVG), `width="8.0cm" height="2.4cm" viewBox="0 0 113 1" preserveAspectRatio="none"`) {
		t.Fatalf("barcode SVG dimensions: %s", barSVG[:220])
	}
	if _, problem := AhdQRSVGBytes("test", matrix, 0); problem != "test size must be greater than 0 and at most 1000.0 centimeters; received 0.0" {
		t.Fatalf("zero SVG size problem: %q", problem)
	}
}

// A failed save never leaves a partial file and never replaces an existing
// destination; a successful save replaces it atomically.
func TestCodesSavePublishesAtomically(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "code.png")
	data := AhdQRCreate(AhdClassQRError, "AhdCode", "M")
	AhdQRSavePNG(AhdClassQRError, data, destination, 256)
	original, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if message := codesRaised(t, func() { AhdQRSavePNG(AhdClassQRError, data, destination, 3) }); !strings.Contains(message, "pixels must be between") {
		t.Fatalf("unexpected message %q", message)
	}
	if message := codesRaised(t, func() { AhdQRSavePNG(AhdClassQRError, data, filepath.Join(directory, "code.jpg"), 256) }); message != "QRCode.savePNG destination must use the .png extension" {
		t.Fatalf("unexpected message %q", message)
	}
	if message := codesRaised(t, func() {
		AhdQRSaveSVG(AhdClassQRError, data, filepath.Join(directory, "missing", "code.svg"), 5)
	}); !strings.HasPrefix(message, "QRCode.saveSVG could not write") {
		t.Fatalf("unexpected message %q", message)
	}
	after, _ := os.ReadFile(destination)
	if !bytes.Equal(original, after) {
		t.Fatal("a failed save changed the existing destination")
	}
	entries, _ := os.ReadDir(directory)
	for _, entry := range entries {
		if entry.Name() != "code.png" {
			t.Fatalf("a save left %s behind", entry.Name())
		}
	}
	bar := AhdBarcodeCreate(AhdClassBarcodeError, "Code128", "AHD-42")
	AhdBarcodeSaveSVG(AhdClassBarcodeError, bar, filepath.Join(directory, "bar.svg"), 8, 2.4)
	AhdBarcodeSavePNG(AhdClassBarcodeError, bar, filepath.Join(directory, "bar.png"), 800, 240)
	if message := codesRaised(t, func() { AhdBarcodeSavePNG(AhdClassBarcodeError, bar, filepath.Join(directory, "b.png"), 20, 240) }); !strings.HasPrefix(message, "BarcodeCode.savePNG width must be between") {
		t.Fatalf("unexpected message %q", message)
	}
}

func TestQRCodeAccessorsReencodeFromStorage(t *testing.T) {
	data := AhdQRCreate(AhdClassQRError, "Ayşe 😊", "Q")
	if AhdQRValue(AhdClassQRError, data) != "Ayşe 😊" || AhdQRLevel(AhdClassQRError, data) != "Q" {
		t.Fatalf("accessors wrong for %q", data)
	}
	rows := AhdQRMatrix(AhdClassQRError, data).Snapshot()
	if int64(len(rows)) != AhdQRSize(AhdClassQRError, data) {
		t.Fatal("matrix row count differs from size")
	}
	rows[0].Snapshot()[0] = false
	if !AhdQRMatrix(AhdClassQRError, data).Snapshot()[0].Snapshot()[0] {
		t.Fatal("mutating a returned matrix changed the QRCode")
	}
	if message := codesRaised(t, func() { AhdQRValue(AhdClassQRError, "Zbroken") }); message != "QRCode storage is corrupted" {
		t.Fatalf("corrupted storage message %q", message)
	}
	bar := AhdBarcodeCreate(AhdClassBarcodeError, "UPCA", "03600029145")
	if AhdBarcodeKind(AhdClassBarcodeError, bar) != "UPCA" || AhdBarcodeValue(AhdClassBarcodeError, bar) != "036000291452" ||
		AhdBarcodePattern(AhdClassBarcodeError, bar).Len() != 95 {
		t.Fatalf("barcode accessors wrong for %q", bar)
	}
}

// Latex and PDF draw the same logical symbol as vector TikZ: no raster image.
func TestLatexCodeFragmentsAreVector(t *testing.T) {
	text, problem := AhdLatexQRText("Latex.qr", "https://ahdcode.org", 3, "M")
	if problem != "" {
		t.Fatal(problem)
	}
	if !strings.HasPrefix(text, "%\n% AHDCODE_TIKZ\n\\begin{tikzpicture}\n\\begin{scope}") || strings.Contains(text, "includegraphics") {
		t.Fatalf("QR fragment is not the vector TikZ form:\n%s", text[:120])
	}
	matrix, _ := AhdQRMatrixText("test", "https://ahdcode.org", "M")
	runs := 0
	for _, row := range matrix {
		runs += len(ahdCodesRuns(row))
	}
	if strings.Count(text, "rectangle") != runs+1 {
		t.Fatalf("QR fragment draws %d rectangles, want %d runs plus the background", strings.Count(text, "rectangle"), runs)
	}
	bar, problem := AhdLatexBarcodeText("Latex.barcode", "EAN13", "590123412345", 8, 2)
	if problem != "" || !strings.Contains(bar, "\\fill[black] (11,0) rectangle (12,1)") {
		t.Fatalf("barcode fragment %q (%s)", bar, problem)
	}
	if _, problem := AhdLatexBarcodeText("Latex.barcode", "Code39", "X", 8, 2); problem != `Latex.barcode kind must be Code128, EAN13, or UPCA; received "Code39"` {
		t.Fatalf("unknown kind problem %q", problem)
	}
	if _, problem := AhdLatexQRText("Latex.qr", "x", -1, "M"); !strings.HasPrefix(problem, "Latex.qr size must be greater than 0") {
		t.Fatalf("negative size problem %q", problem)
	}
}

// codesFixture is one symbol the independent decoder in tooling/codesqa reads
// back. The PNG files in testdata/codes are regenerated with
// AHDCODE_UPDATE_CODES_TESTDATA=1; otherwise this test proves they still show
// exactly the modules the current encoder draws.
type codesFixture struct {
	File    string `json:"file"`
	Format  string `json:"format"`
	Payload string `json:"payload"`
	kind    string
	value   string
	level   string
}

func codesFixtures() []codesFixture {
	return []codesFixture{
		{File: "qr-url-M.png", Format: "QR_CODE", Payload: "https://ahdcode.org/verify?id=AHD-2026-0042&lang=tr#top", kind: "QR", level: "M"},
		{File: "qr-turkish-emoji-H.png", Format: "QR_CODE", Payload: "Doğrulama: Ayşe Yılmaz — İstanbul 😊", kind: "QR", level: "H"},
		{File: "qr-long-url-L.png", Format: "QR_CODE", Payload: "https://ahdcode.org/certificates/verify?number=AHD-2026-000042&recipient=Ay%C5%9Fe%20Y%C4%B1lmaz&course=Numerical%20Methods&issued=2026-09-12&signature=" + strings.Repeat("0123456789abcdef", 8), kind: "QR", level: "L"},
		{File: "ean13-computed.png", Format: "EAN_13", Payload: "5901234123457", kind: "EAN13", value: "590123412345"},
		{File: "upca-computed.png", Format: "UPC_A", Payload: "036000291452", kind: "UPCA", value: "03600029145"},
		{File: "code128-mixed.png", Format: "CODE_128", Payload: "AHD-Asset 00042/b", kind: "Code128", value: "AHD-Asset 00042/b"},
	}
}

func (fixture codesFixture) render(t *testing.T) []byte {
	t.Helper()
	if fixture.kind == "QR" {
		matrix, problem := AhdQRMatrixText("test", fixture.Payload, fixture.level)
		if problem != "" {
			t.Fatal(problem)
		}
		encoded, problem := AhdQRPNGBytes("test", matrix, 600)
		if problem != "" {
			t.Fatal(problem)
		}
		return encoded
	}
	pattern, _, problem := AhdBarcodePatternText("test", fixture.kind, fixture.value)
	if problem != "" {
		t.Fatal(problem)
	}
	encoded, problem := AhdBarcodePNGBytes("test", fixture.kind, pattern, 800, 240)
	if problem != "" {
		t.Fatal(problem)
	}
	return encoded
}

func TestCodesDecoderFixturesAreCurrent(t *testing.T) {
	directory := filepath.Join("testdata", "codes")
	update := os.Getenv("AHDCODE_UPDATE_CODES_TESTDATA") == "1"
	if update {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest, _ := json.MarshalIndent(codesFixtures(), "", "  ")
		if err := os.WriteFile(filepath.Join(directory, "manifest.json"), append(manifest, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range codesFixtures() {
		rendered := fixture.render(t)
		path := filepath.Join(directory, fixture.File)
		if update {
			if err := os.WriteFile(path, rendered, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		stored, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (regenerate with AHDCODE_UPDATE_CODES_TESTDATA=1)", err)
		}
		want, _ := png.Decode(bytes.NewReader(rendered))
		got, err := png.Decode(bytes.NewReader(stored))
		if err != nil {
			t.Fatal(err)
		}
		if got.Bounds() != want.Bounds() || !bytes.Equal(got.(*image.Gray).Pix, want.(*image.Gray).Pix) {
			t.Fatalf("%s no longer shows the symbol the encoder draws; regenerate with AHDCODE_UPDATE_CODES_TESTDATA=1", fixture.File)
		}
	}
}
