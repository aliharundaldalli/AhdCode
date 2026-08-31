package evaluator

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"ahdcode/internal/ir"
)

// The JSON standard module's REPL implementation. It mirrors the native
// backend's json.go logic function-for-function - the same canonical-text
// representation, the same parser grammar, the same limits - but operates on
// the evaluator's own List/Pair/Instance representation instead of
// AhdList/AhdPair/AhdInstance, exactly as word.go independently reimplements
// the Word runtime rather than importing ahdruntime.
//
// A JSONValue instance stores its entire content as one hidden field: its
// own canonical, compact JSON text. Every accessor decodes that text, and
// any accessor that itself produces a JSONValue (array(), get(), at(), ...)
// hands back a fresh instance built from a fresh canonical text - the same
// "hidden String field, reparsed by helpers" pattern Document uses for its
// block list.

const jsonValueClassID = ir.ClassID("builtin:JSON::class::JSONValue")

var jsonValueTextField = ir.FieldID(string(jsonValueClassID) + "::field::text")

const (
	jsonMaxInputBytes = 8 * 1024 * 1024
	jsonMaxDepth      = 256
)

// jsonValueText reads the one hidden storage field of a JSONValue instance.
func (s *Session) jsonValueText(value any) string {
	instance := s.requireInstance(value)
	text, ok := instance.Fields[jsonValueTextField].(string)
	if !ok {
		s.raise("JSONError", "value is not a JSONValue")
	}
	return text
}

// jsonValueFrom materializes one canonical text as a new JSONValue instance.
func (s *Session) jsonValueFrom(text string) *Instance {
	return &Instance{Class: jsonValueClassID, Fields: map[ir.FieldID]any{jsonValueTextField: text}}
}

func (s *Session) jsonBuiltin(name string, args []any) any {
	switch name {
	case "parse":
		return s.jsonValueFrom(jsonCanonicalText(s.jsonParseDocument(args[0].(string))))
	case "read":
		return s.jsonValueFrom(jsonCanonicalText(s.jsonParseDocument(s.jsonReadFile(args[0].(string)))))
	case "nullValue":
		return s.jsonValueFrom("null")
	case "fromBool":
		if args[0].(bool) {
			return s.jsonValueFrom("true")
		}
		return s.jsonValueFrom("false")
	case "fromInt":
		return s.jsonValueFrom(strconv.FormatInt(args[0].(int64), 10))
	case "fromReal":
		value := args[0].(float64)
		if math.IsNaN(value) || math.IsInf(value, 0) {
			s.raise("JSONError", "JSON Real value must be finite")
		}
		return s.jsonValueFrom(jsonFormatReal(value))
	case "fromString":
		return s.jsonValueFrom(jsonEncodeString(args[0].(string)))
	case "array":
		list := s.requireList(args[0])
		texts := make([]string, len(list.Items))
		for index, item := range list.Items {
			texts[index] = s.jsonValueText(item)
		}
		return s.jsonValueFrom("[" + strings.Join(texts, ",") + "]")
	case "object":
		pair := s.requirePair(args[0])
		var builder strings.Builder
		builder.WriteByte('{')
		for index, key := range pair.Keys {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(jsonEncodeString(key.(string)))
			builder.WriteByte(':')
			builder.WriteString(s.jsonValueText(pair.Values[key]))
		}
		builder.WriteByte('}')
		return s.jsonValueFrom(builder.String())
	case "stringify":
		text := s.jsonValueText(args[0])
		pretty := len(args) > 1 && args[1] != nil && args[1].(bool)
		return s.jsonStringify(text, pretty)
	case "write":
		text := s.jsonValueText(args[0])
		pretty := len(args) > 2 && args[2] != nil && args[2].(bool)
		content := s.jsonStringify(text, pretty)
		if err := jsonPublish([]byte(content), s.sessionPath(args[1].(string))); err != nil {
			s.raise("JSONError", "could not write the JSON file: "+err.Error())
		}
		return Nothing
	}
	s.raise("Error", "unsupported JSON function "+name)
	return nil
}

func (s *Session) jsonReadFile(path string) string {
	content, err := os.ReadFile(s.sessionPath(path))
	if err != nil {
		s.raise("JSONError", "could not read the JSON file: "+err.Error())
	}
	return string(content)
}

func (s *Session) jsonStringify(text string, pretty bool) string {
	if !pretty {
		return text
	}
	node := s.jsonParseDocument(text)
	return jsonStringifyNode(node, true, 0)
}

func jsonPublish(data []byte, output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-json-output-*")
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

func (s *Session) jsonOperation(name string, receiver any, arguments []any) any {
	text := s.jsonValueText(receiver)
	switch name {
	case "JSONValue.kind":
		return s.jsonDecode(text).kind
	case "JSONValue.isNull":
		return s.jsonDecode(text).kind == "Null"
	case "JSONValue.bool":
		node := s.jsonDecode(text)
		if node.kind != "Bool" {
			s.jsonWrongKind("bool()", node.kind)
		}
		return node.flag
	case "JSONValue.int":
		node := s.jsonDecode(text)
		if node.kind != "Int" {
			s.jsonWrongKind("int()", node.kind)
		}
		return node.number
	case "JSONValue.real":
		node := s.jsonDecode(text)
		switch node.kind {
		case "Real":
			return node.real
		case "Int":
			return float64(node.number)
		default:
			s.jsonWrongKind("real()", node.kind)
			return nil
		}
	case "JSONValue.string":
		node := s.jsonDecode(text)
		if node.kind != "String" {
			s.jsonWrongKind("string()", node.kind)
		}
		return node.text
	case "JSONValue.array":
		node := s.jsonDecode(text)
		if node.kind != "Array" {
			s.jsonWrongKind("array()", node.kind)
		}
		items := make([]any, len(node.items))
		for index, item := range node.items {
			items[index] = s.jsonValueFrom(jsonCanonicalText(item))
		}
		return &List{Items: items}
	case "JSONValue.object":
		node := s.jsonDecode(text)
		if node.kind != "Object" {
			s.jsonWrongKind("object()", node.kind)
		}
		pair := &Pair{Keys: make([]any, len(node.keys)), Values: make(map[any]any, len(node.keys))}
		for index, key := range node.keys {
			pair.Keys[index] = key
			pair.Values[key] = s.jsonValueFrom(jsonCanonicalText(node.values[key]))
		}
		return pair
	case "JSONValue.get":
		node := s.jsonDecode(text)
		if node.kind != "Object" {
			s.jsonWrongKind("get()", node.kind)
		}
		value, found := node.values[arguments[0].(string)]
		if !found {
			return nil
		}
		return s.jsonValueFrom(jsonCanonicalText(value))
	case "JSONValue.at":
		node := s.jsonDecode(text)
		if node.kind != "Array" {
			s.jsonWrongKind("at()", node.kind)
		}
		length := int64(len(node.items))
		index := arguments[0].(int64)
		if index < 0 {
			index += length
		}
		if index < 0 || index >= length {
			s.raise("JSONError", "JSONValue array index is out of range")
		}
		return s.jsonValueFrom(jsonCanonicalText(node.items[index]))
	}
	s.raise("Error", "unsupported JSONValue operation "+name)
	return nil
}

func (s *Session) jsonWrongKind(operation, kind string) {
	s.raise("JSONError", operation+" cannot be called on a "+kind+" JSONValue")
}

func (s *Session) jsonDecode(text string) jsonNode {
	return s.jsonParseDocument(text)
}

// ---------------------------------------------------------------------------
// Parser and serializer (duplicated from the native runtime's json.go by
// design; see the package comment above)
// ---------------------------------------------------------------------------

type jsonNode struct {
	kind   string
	flag   bool
	number int64
	real   float64
	text   string
	items  []jsonNode
	keys   []string
	values map[string]jsonNode
}

type jsonParser struct {
	session *Session
	source  string
	pos     int
}

func (s *Session) jsonParseDocument(source string) jsonNode {
	if len(source) > jsonMaxInputBytes {
		s.raise("JSONError", "JSON input is larger than the supported limit")
	}
	if !utf8.ValidString(source) {
		s.raise("JSONError", "JSON input is not valid UTF-8")
	}
	parser := &jsonParser{session: s, source: source}
	parser.skipWhitespace()
	if parser.pos >= len(parser.source) {
		s.raise("JSONError", "JSON input is empty")
	}
	node := parser.parseValue(0)
	parser.skipWhitespace()
	if parser.pos != len(parser.source) {
		s.raise("JSONError", "JSON input has trailing content after its value")
	}
	return node
}

func (parser *jsonParser) fail(message string) {
	parser.session.raise("JSONError", message)
}

func (parser *jsonParser) skipWhitespace() {
	for parser.pos < len(parser.source) {
		switch parser.source[parser.pos] {
		case ' ', '\t', '\n', '\r':
			parser.pos++
		default:
			return
		}
	}
}

func (parser *jsonParser) parseValue(depth int) jsonNode {
	if depth > jsonMaxDepth {
		parser.fail("JSON input exceeds the maximum supported nesting depth")
	}
	parser.skipWhitespace()
	if parser.pos >= len(parser.source) {
		parser.fail("JSON input ends where a value was expected")
	}
	switch character := parser.source[parser.pos]; {
	case character == '{':
		return parser.parseObject(depth)
	case character == '[':
		return parser.parseArray(depth)
	case character == '"':
		return jsonNode{kind: "String", text: parser.parseString()}
	case character == 't':
		parser.expectLiteral("true")
		return jsonNode{kind: "Bool", flag: true}
	case character == 'f':
		parser.expectLiteral("false")
		return jsonNode{kind: "Bool", flag: false}
	case character == 'n':
		parser.expectLiteral("null")
		return jsonNode{kind: "Null"}
	case character == '-' || (character >= '0' && character <= '9'):
		return parser.parseNumber()
	default:
		parser.fail("JSON input has an unexpected character")
		return jsonNode{}
	}
}

func (parser *jsonParser) expectLiteral(literal string) {
	if !strings.HasPrefix(parser.source[parser.pos:], literal) {
		parser.fail("JSON input has an invalid literal")
	}
	parser.pos += len(literal)
}

func (parser *jsonParser) parseObject(depth int) jsonNode {
	parser.pos++
	node := jsonNode{kind: "Object", values: make(map[string]jsonNode)}
	parser.skipWhitespace()
	if parser.pos < len(parser.source) && parser.source[parser.pos] == '}' {
		parser.pos++
		return node
	}
	for {
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) || parser.source[parser.pos] != '"' {
			parser.fail("JSON object key must be a String")
		}
		key := parser.parseString()
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) || parser.source[parser.pos] != ':' {
			parser.fail("JSON object is missing ':' after a key")
		}
		parser.pos++
		value := parser.parseValue(depth + 1)
		if _, duplicate := node.values[key]; duplicate {
			parser.fail("JSON object has a duplicate key")
		}
		node.keys = append(node.keys, key)
		node.values[key] = value
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) {
			parser.fail("JSON object is not closed")
		}
		switch parser.source[parser.pos] {
		case ',':
			parser.pos++
			continue
		case '}':
			parser.pos++
			return node
		default:
			parser.fail("JSON object is missing ',' or '}'")
		}
	}
}

func (parser *jsonParser) parseArray(depth int) jsonNode {
	parser.pos++
	node := jsonNode{kind: "Array"}
	parser.skipWhitespace()
	if parser.pos < len(parser.source) && parser.source[parser.pos] == ']' {
		parser.pos++
		return node
	}
	for {
		node.items = append(node.items, parser.parseValue(depth+1))
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) {
			parser.fail("JSON array is not closed")
		}
		switch parser.source[parser.pos] {
		case ',':
			parser.pos++
			continue
		case ']':
			parser.pos++
			return node
		default:
			parser.fail("JSON array is missing ',' or ']'")
		}
	}
}

func (parser *jsonParser) parseString() string {
	parser.pos++
	var builder strings.Builder
	for {
		if parser.pos >= len(parser.source) {
			parser.fail("JSON String is not closed")
		}
		character := parser.source[parser.pos]
		switch {
		case character == '"':
			parser.pos++
			return builder.String()
		case character == '\\':
			parser.pos++
			if parser.pos >= len(parser.source) {
				parser.fail("JSON String has an incomplete escape sequence")
			}
			switch parser.source[parser.pos] {
			case '"':
				builder.WriteByte('"')
				parser.pos++
			case '\\':
				builder.WriteByte('\\')
				parser.pos++
			case '/':
				builder.WriteByte('/')
				parser.pos++
			case 'b':
				builder.WriteByte('\b')
				parser.pos++
			case 'f':
				builder.WriteByte('\f')
				parser.pos++
			case 'n':
				builder.WriteByte('\n')
				parser.pos++
			case 'r':
				builder.WriteByte('\r')
				parser.pos++
			case 't':
				builder.WriteByte('\t')
				parser.pos++
			case 'u':
				builder.WriteRune(parser.parseUnicodeEscape())
			default:
				parser.fail("JSON String has an invalid escape sequence")
			}
		case character < 0x20:
			parser.fail("JSON String contains an unescaped control character")
		default:
			_, width := utf8.DecodeRuneInString(parser.source[parser.pos:])
			builder.WriteString(parser.source[parser.pos : parser.pos+width])
			parser.pos += width
		}
	}
}

func (parser *jsonParser) parseUnicodeEscape() rune {
	high := parser.parseHex4()
	if utf16.IsSurrogate(rune(high)) {
		if strings.HasPrefix(parser.source[parser.pos:], `\u`) {
			mark := parser.pos
			parser.pos += 2
			low := parser.parseHex4()
			combined := utf16.DecodeRune(rune(high), rune(low))
			if combined != utf8.RuneError {
				return combined
			}
			parser.pos = mark
		}
		return utf8.RuneError
	}
	return rune(high)
}

func (parser *jsonParser) parseHex4() uint16 {
	parser.pos++
	if parser.pos+4 > len(parser.source) {
		parser.fail("JSON String has an incomplete \\u escape")
	}
	digits := parser.source[parser.pos : parser.pos+4]
	value, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		parser.fail("JSON String has an invalid \\u escape")
	}
	parser.pos += 4
	return uint16(value)
}

func (parser *jsonParser) parseNumber() jsonNode {
	start := parser.pos
	if parser.pos < len(parser.source) && parser.source[parser.pos] == '-' {
		parser.pos++
	}
	if parser.pos >= len(parser.source) || parser.source[parser.pos] < '0' || parser.source[parser.pos] > '9' {
		parser.fail("JSON input has a malformed number")
	}
	if parser.source[parser.pos] == '0' {
		parser.pos++
	} else {
		for parser.pos < len(parser.source) && parser.source[parser.pos] >= '0' && parser.source[parser.pos] <= '9' {
			parser.pos++
		}
	}
	isReal := false
	if parser.pos < len(parser.source) && parser.source[parser.pos] == '.' {
		isReal = true
		parser.pos++
		digitStart := parser.pos
		for parser.pos < len(parser.source) && parser.source[parser.pos] >= '0' && parser.source[parser.pos] <= '9' {
			parser.pos++
		}
		if parser.pos == digitStart {
			parser.fail("JSON number has a malformed fraction")
		}
	}
	if parser.pos < len(parser.source) && (parser.source[parser.pos] == 'e' || parser.source[parser.pos] == 'E') {
		isReal = true
		parser.pos++
		if parser.pos < len(parser.source) && (parser.source[parser.pos] == '+' || parser.source[parser.pos] == '-') {
			parser.pos++
		}
		digitStart := parser.pos
		for parser.pos < len(parser.source) && parser.source[parser.pos] >= '0' && parser.source[parser.pos] <= '9' {
			parser.pos++
		}
		if parser.pos == digitStart {
			parser.fail("JSON number has a malformed exponent")
		}
	}
	lexeme := parser.source[start:parser.pos]
	if !isReal {
		value, err := strconv.ParseInt(lexeme, 10, 64)
		if err != nil {
			parser.fail("JSON integer literal " + lexeme + " does not fit AhdCode's Int range")
		}
		return jsonNode{kind: "Int", number: value}
	}
	value, err := strconv.ParseFloat(lexeme, 64)
	if err != nil || math.IsInf(value, 0) {
		parser.fail("JSON real literal " + lexeme + " is out of range")
	}
	return jsonNode{kind: "Real", real: value}
}

func jsonFormatReal(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}

func jsonEncodeString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, character := range value {
		switch {
		case character == '"':
			builder.WriteString(`\"`)
		case character == '\\':
			builder.WriteString(`\\`)
		case character == '\n':
			builder.WriteString(`\n`)
		case character == '\r':
			builder.WriteString(`\r`)
		case character == '\t':
			builder.WriteString(`\t`)
		case character == '\b':
			builder.WriteString(`\b`)
		case character == '\f':
			builder.WriteString(`\f`)
		case character < 0x20:
			builder.WriteString(`\u`)
			builder.WriteString(strconv.FormatInt(int64(character), 16))
		default:
			builder.WriteRune(character)
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func jsonStringifyNode(node jsonNode, pretty bool, depth int) string {
	indent := func(level int) string {
		if !pretty {
			return ""
		}
		return "\n" + strings.Repeat("  ", level)
	}
	switch node.kind {
	case "Null":
		return "null"
	case "Bool":
		if node.flag {
			return "true"
		}
		return "false"
	case "Int":
		return strconv.FormatInt(node.number, 10)
	case "Real":
		return jsonFormatReal(node.real)
	case "String":
		return jsonEncodeString(node.text)
	case "Array":
		if len(node.items) == 0 {
			return "[]"
		}
		var builder strings.Builder
		builder.WriteByte('[')
		for index, item := range node.items {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(indent(depth + 1))
			builder.WriteString(jsonStringifyNode(item, pretty, depth+1))
		}
		builder.WriteString(indent(depth))
		builder.WriteByte(']')
		return builder.String()
	case "Object":
		if len(node.keys) == 0 {
			return "{}"
		}
		var builder strings.Builder
		builder.WriteByte('{')
		for index, key := range node.keys {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(indent(depth + 1))
			builder.WriteString(jsonEncodeString(key))
			builder.WriteByte(':')
			if pretty {
				builder.WriteByte(' ')
			}
			builder.WriteString(jsonStringifyNode(node.values[key], pretty, depth+1))
		}
		builder.WriteString(indent(depth))
		builder.WriteByte('}')
		return builder.String()
	}
	return "null"
}

func jsonCanonicalText(node jsonNode) string {
	return jsonStringifyNode(node, false, 0)
}
