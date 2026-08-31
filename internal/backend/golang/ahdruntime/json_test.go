package ahdruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONScalarsRoundTripWithCorrectKind(t *testing.T) {
	cases := []struct {
		source string
		kind   string
		text   string
	}{
		{"null", "Null", "null"},
		{"true", "Bool", "true"},
		{"false", "Bool", "false"},
		{"0", "Int", "0"},
		{"-1", "Int", "-1"},
		{"91", "Int", "91"},
		{"3.14", "Real", "3.14"},
		{"1e3", "Real", "1000.0"},
		{"1.0", "Real", "1.0"},
		{`"hello"`, "String", `"hello"`},
		{`"Ali çayı"`, "String", "\"Ali çayı\""},
	}
	for _, testCase := range cases {
		text := AhdJSONParse(AhdClassJSONError, testCase.source)
		if got := AhdJSONKind(AhdClassJSONError, text); got != testCase.kind {
			t.Fatalf("parse(%q) kind = %q, want %q", testCase.source, got, testCase.kind)
		}
		if text != testCase.text {
			t.Fatalf("parse(%q) canonical text = %q, want %q", testCase.source, text, testCase.text)
		}
	}
}

func TestJSONUnicodeAndEscapedStrings(t *testing.T) {
	text := AhdJSONParse(AhdClassJSONError, `"line1\nline2\ttab\"quote\\backslash"`)
	if got := AhdJSONString(AhdClassJSONError, text); got != "line1\nline2\ttab\"quote\\backslash" {
		t.Fatalf("escaped string = %q", got)
	}
	turkish := AhdJSONParse(AhdClassJSONError, `"Öğrenci ğüşıöç"`)
	if got := AhdJSONString(AhdClassJSONError, turkish); got != "Öğrenci ğüşıöç" {
		t.Fatalf("unicode string = %q", got)
	}
}

func TestJSONCollectionsAndOrdering(t *testing.T) {
	array := AhdJSONParse(AhdClassJSONError, "[]")
	if got := AhdJSONArrayElements(AhdClassJSONError, array); len(got) != 0 {
		t.Fatalf("empty array elements = %v", got)
	}
	nested := AhdJSONParse(AhdClassJSONError, "[[1,2],[3]]")
	outer := AhdJSONArrayElements(AhdClassJSONError, nested)
	if len(outer) != 2 {
		t.Fatalf("nested array outer length = %d", len(outer))
	}
	inner := AhdJSONArrayElements(AhdClassJSONError, outer[0])
	if len(inner) != 2 || AhdJSONInt(AhdClassJSONError, inner[0]) != 1 || AhdJSONInt(AhdClassJSONError, inner[1]) != 2 {
		t.Fatalf("nested array inner = %v", inner)
	}

	object := AhdJSONParse(AhdClassJSONError, `{}`)
	if got := AhdJSONObjectKeys(AhdClassJSONError, object); len(got) != 0 {
		t.Fatalf("empty object keys = %v", got)
	}
	ordered := AhdJSONParse(AhdClassJSONError, `{"z":1,"a":2,"m":3}`)
	keys := AhdJSONObjectKeys(AhdClassJSONError, ordered)
	if strings.Join(keys, ",") != "z,a,m" {
		t.Fatalf("object insertion order not preserved: %v", keys)
	}
	nestedObject := AhdJSONParse(AhdClassJSONError, `{"outer":{"inner":true}}`)
	inside := AhdJSONGet(AhdClassJSONError, nestedObject, "outer")
	if inside == nil {
		t.Fatal("expected outer key to be present")
	}
	deepInside := AhdJSONGet(AhdClassJSONError, *inside, "inner")
	if deepInside == nil || AhdJSONBool(AhdClassJSONError, *deepInside) != true {
		t.Fatalf("nested object get = %v", deepInside)
	}
}

func TestJSONConstructors(t *testing.T) {
	if AhdJSONNull() != "null" {
		t.Fatal("JSON.null()")
	}
	if AhdJSONFromBool(true) != "true" || AhdJSONFromBool(false) != "false" {
		t.Fatal("JSON.fromBool()")
	}
	if AhdJSONFromInt(91) != "91" {
		t.Fatal("JSON.fromInt()")
	}
	if AhdJSONFromReal(AhdClassJSONError, 5) != "5.0" {
		t.Fatalf("JSON.fromReal(5) = %q, want 5.0 (must not silently read back as Int)", AhdJSONFromReal(AhdClassJSONError, 5))
	}
	if AhdJSONFromString(`a"b`) != `"a\"b"` {
		t.Fatalf("JSON.fromString escaping = %q", AhdJSONFromString(`a"b`))
	}
	array := AhdJSONArray([]string{AhdJSONFromInt(1), AhdJSONFromInt(2)})
	if array != "[1,2]" {
		t.Fatalf("JSON.array() = %q", array)
	}
	object := AhdJSONObject(AhdClassJSONError, []AhdJSONEntry{
		{Key: "name", Text: AhdJSONFromString("Ali")},
		{Key: "score", Text: AhdJSONFromInt(91)},
	})
	if object != `{"name":"Ali","score":91}` {
		t.Fatalf("JSON.object() = %q", object)
	}
}

func TestJSONRealAcceptsIntWidening(t *testing.T) {
	intText := AhdJSONFromInt(7)
	if got := AhdJSONReal(AhdClassJSONError, intText); got != 7.0 {
		t.Fatalf("real() on an Int JSONValue = %v, want 7.0", got)
	}
}

func TestJSONWrongKindAccessorsRaiseJSONError(t *testing.T) {
	stringValue := AhdJSONFromString("x")
	expectRaise(t, AhdClassJSONError, func() { AhdJSONInt(AhdClassJSONError, stringValue) })
	arrayValue := AhdJSONArray(nil)
	expectRaise(t, AhdClassJSONError, func() { AhdJSONObjectKeys(AhdClassJSONError, arrayValue) })
	nullValue := AhdJSONNull()
	expectRaise(t, AhdClassJSONError, func() { AhdJSONString(AhdClassJSONError, nullValue) })
	expectRaise(t, AhdClassJSONError, func() { AhdJSONBool(AhdClassJSONError, nullValue) })
	expectRaise(t, AhdClassJSONError, func() { AhdJSONGet(AhdClassJSONError, arrayValue, "k") })
	expectRaise(t, AhdClassJSONError, func() { AhdJSONAt(AhdClassJSONError, nullValue, 0) })
}

func TestJSONGetMissingKeyReturnsNilNotError(t *testing.T) {
	object := AhdJSONParse(AhdClassJSONError, `{"a":1}`)
	if got := AhdJSONGet(AhdClassJSONError, object, "missing"); got != nil {
		t.Fatalf("get(missing) = %v, want nil", got)
	}
}

func TestJSONAtIndexBoundsAndNegative(t *testing.T) {
	array := AhdJSONParse(AhdClassJSONError, `[10,20,30]`)
	if got := AhdJSONInt(AhdClassJSONError, AhdJSONAt(AhdClassJSONError, array, 0)); got != 10 {
		t.Fatalf("at(0) = %v", got)
	}
	if got := AhdJSONInt(AhdClassJSONError, AhdJSONAt(AhdClassJSONError, array, -1)); got != 30 {
		t.Fatalf("at(-1) = %v", got)
	}
	expectRaise(t, AhdClassJSONError, func() { AhdJSONAt(AhdClassJSONError, array, 3) })
	expectRaise(t, AhdClassJSONError, func() { AhdJSONAt(AhdClassJSONError, array, -4) })
}

func TestJSONParseErrors(t *testing.T) {
	malformed := []string{
		"",
		"{",
		"[1,2",
		`{"a":1,}`,
		`{"a" 1}`,
		`{"a":1 "b":2}`,
		`"unterminated`,
		`"bad \x escape"`,
		"01",
		"1.",
		".5",
		"1e",
		"+1",
		"NaN",
		"Infinity",
		"undefined",
		"nul",
		`{"a":1} trailing`,
		`{"dup":1,"dup":2}`,
		"99999999999999999999",
		"1e400",
	}
	for _, source := range malformed {
		source := source
		expectRaise(t, AhdClassJSONError, func() { AhdJSONParse(AhdClassJSONError, source) })
	}
}

func TestJSONExcessiveDepthRejected(t *testing.T) {
	var builder strings.Builder
	for i := 0; i < ahdJSONMaxDepth+10; i++ {
		builder.WriteByte('[')
	}
	expectRaise(t, AhdClassJSONError, func() { AhdJSONParse(AhdClassJSONError, builder.String()) })
}

func TestJSONStringifyCompactAndPretty(t *testing.T) {
	value := AhdJSONParse(AhdClassJSONError, `{"a":1,"b":[1,2,3]}`)
	if got := AhdJSONStringify(value, false); got != `{"a":1,"b":[1,2,3]}` {
		t.Fatalf("compact stringify = %q", got)
	}
	pretty := AhdJSONStringify(value, true)
	want := "{\n  \"a\": 1,\n  \"b\": [\n    1,\n    2,\n    3\n  ]\n}"
	if pretty != want {
		t.Fatalf("pretty stringify = %q, want %q", pretty, want)
	}
	// Deterministic: repeated stringify of the same value produces the same
	// output.
	if AhdJSONStringify(value, true) != pretty {
		t.Fatal("pretty stringify is not deterministic")
	}
	emptyArray := AhdJSONParse(AhdClassJSONError, "[]")
	if AhdJSONStringify(emptyArray, true) != "[]" {
		t.Fatalf("pretty empty array = %q", AhdJSONStringify(emptyArray, true))
	}
	emptyObject := AhdJSONParse(AhdClassJSONError, "{}")
	if AhdJSONStringify(emptyObject, true) != "{}" {
		t.Fatalf("pretty empty object = %q", AhdJSONStringify(emptyObject, true))
	}
}

func TestJSONParseStringifyParseSemanticEquality(t *testing.T) {
	source := `{"name":"Ali","scores":[91,88.5,null,true],"nested":{"x":1}}`
	first := AhdJSONParse(AhdClassJSONError, source)
	stringified := AhdJSONStringify(first, false)
	second := AhdJSONParse(AhdClassJSONError, stringified)
	if first != second {
		t.Fatalf("parse -> stringify -> parse changed canonical text: %q vs %q", first, second)
	}
}

func TestJSONFileIO(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "a b", "réport.json")
	value := AhdJSONObject(AhdClassJSONError, []AhdJSONEntry{{Key: "ok", Text: AhdJSONFromBool(true)}})
	AhdJSONWrite(AhdClassJSONError, value, path, true)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"ok": true`) {
		t.Fatalf("written file content = %q", string(content))
	}
	read := AhdJSONRead(AhdClassJSONError, path)
	if AhdJSONBool(AhdClassJSONError, *AhdJSONGet(AhdClassJSONError, read, "ok")) != true {
		t.Fatal("AhdJSONRead did not round-trip the written file")
	}
	expectRaise(t, AhdClassJSONError, func() { AhdJSONRead(AhdClassJSONError, filepath.Join(directory, "missing.json")) })
}

func TestJSONFromRealRejectsNonFinite(t *testing.T) {
	nan := 0.0
	nan = nan / nan
	expectRaise(t, AhdClassJSONError, func() { AhdJSONFromReal(AhdClassJSONError, nan) })
	positiveInfinity := 1.0
	for i := 0; i < 400; i++ {
		positiveInfinity *= 10
	}
	expectRaise(t, AhdClassJSONError, func() { AhdJSONFromReal(AhdClassJSONError, positiveInfinity) })
}
