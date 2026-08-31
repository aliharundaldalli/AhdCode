package evaluator

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"ahdcode/internal/ir"
)

// The XML standard module's REPL implementation. It mirrors the native
// backend's ahdruntime XML section function-for-function - the same hidden
// encoded-data representation, the same encoding/xml.Decoder token walking,
// the same limits - but operates on the evaluator's own List/Pair/Instance
// representation instead of AhdList/AhdPair/AhdInstance. See xml.go's
// package-level comment in ahdruntime for why encoding/xml.Decoder is both
// sufficient and safe here (namespace resolution, CDATA-as-text, and no XXE
// all come from its own stdlib defaults).

const (
	xmlNodeClassID     = ir.ClassID("builtin:XML::class::XMLNode")
	xmlDocumentClassID = ir.ClassID("builtin:XML::class::XMLDocument")
)

var (
	xmlNodeDataField     = ir.FieldID(string(xmlNodeClassID) + "::field::data")
	xmlDocumentDataField = ir.FieldID(string(xmlDocumentClassID) + "::field::data")
)

const (
	xmlMaxInputBytes = 8 * 1024 * 1024
	xmlMaxDepth      = 256
)

// xmlData is the parsed/interchange form of one XMLNode, and is also
// exactly what an XMLNode's/XMLDocument's hidden field encodes.
type xmlData struct {
	Kind      string    `json:"kind"`
	Name      string    `json:"name,omitempty"`
	Namespace string    `json:"namespace,omitempty"`
	Text      string    `json:"text,omitempty"`
	AttrKeys  []string  `json:"attrKeys,omitempty"`
	AttrVals  []string  `json:"attrVals,omitempty"`
	Children  []xmlData `json:"children,omitempty"`
}

func xmlEncode(node xmlData) string {
	encoded, _ := json.Marshal(node)
	return string(encoded)
}

func (s *Session) xmlDecode(data string) xmlData {
	var node xmlData
	if err := json.Unmarshal([]byte(data), &node); err != nil {
		s.raise("XMLError", "XML node storage is corrupted")
	}
	return node
}

func (s *Session) xmlWrongKind(operation, kind string) {
	s.raise("XMLError", operation+" cannot be called on a "+kind+" XMLNode")
}

func (s *Session) xmlNodeData(value any) string {
	instance := s.requireInstance(value)
	data, ok := instance.Fields[xmlNodeDataField].(string)
	if !ok {
		s.raise("XMLError", "value is not an XMLNode")
	}
	return data
}

func (s *Session) xmlNodeFrom(data string) *Instance {
	return &Instance{Class: xmlNodeClassID, Fields: map[ir.FieldID]any{xmlNodeDataField: data}}
}

func (s *Session) xmlDocumentData(value any) string {
	instance := s.requireInstance(value)
	data, ok := instance.Fields[xmlDocumentDataField].(string)
	if !ok {
		s.raise("XMLError", "value is not an XMLDocument")
	}
	return data
}

func (s *Session) xmlDocumentFrom(data string) *Instance {
	return &Instance{Class: xmlDocumentClassID, Fields: map[ir.FieldID]any{xmlDocumentDataField: data}}
}

func (s *Session) xmlBuiltin(name string, args []any) any {
	switch name {
	case "text":
		return s.xmlNodeFrom(xmlEncode(xmlData{Kind: "Text", Text: args[0].(string)}))
	case "element":
		attributes := s.requirePair(args[1])
		children := s.requireList(args[2])
		attrKeys := make([]string, len(attributes.Keys))
		attrVals := make([]string, len(attributes.Keys))
		for index, key := range attributes.Keys {
			attrKeys[index] = key.(string)
			attrVals[index] = attributes.Values[key].(string)
		}
		nodeChildren := make([]xmlData, len(children.Items))
		for index, item := range children.Items {
			nodeChildren[index] = s.xmlDecode(s.xmlNodeData(item))
		}
		return s.xmlNodeFrom(xmlEncode(xmlData{Kind: "Element", Name: args[0].(string), AttrKeys: attrKeys, AttrVals: attrVals, Children: nodeChildren}))
	case "document":
		data := s.xmlNodeData(args[0])
		root := s.xmlDecode(data)
		if root.Kind != "Element" {
			s.raise("XMLError", "an XMLDocument root must be an Element, not Text")
		}
		return s.xmlDocumentFrom(data)
	case "parse":
		return s.xmlDocumentFrom(xmlEncode(s.xmlParseDocument(args[0].(string))))
	case "read":
		return s.xmlDocumentFrom(xmlEncode(s.xmlParseDocument(s.xmlReadFile(args[0].(string)))))
	case "stringify":
		pretty := len(args) > 1 && args[1] != nil && args[1].(bool)
		return xmlStringifyNode(s.xmlDecode(s.xmlDocumentData(args[0])), pretty, 0)
	case "write":
		pretty := len(args) > 2 && args[2] != nil && args[2].(bool)
		content := xmlStringifyNode(s.xmlDecode(s.xmlDocumentData(args[0])), pretty, 0)
		if err := xmlPublish([]byte(content), s.sessionPath(args[1].(string))); err != nil {
			s.raise("XMLError", "could not write the XML file: "+err.Error())
		}
		return Nothing
	}
	s.raise("Error", "unsupported XML function "+name)
	return nil
}

func (s *Session) xmlReadFile(path string) string {
	content, err := os.ReadFile(s.sessionPath(path))
	if err != nil {
		s.raise("XMLError", "could not read the XML file: "+err.Error())
	}
	return string(content)
}

func xmlPublish(data []byte, output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-xml-output-*")
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

func (s *Session) xmlOperation(name string, receiver any, arguments []any) any {
	if name == "XMLDocument.root" {
		return s.xmlNodeFrom(s.xmlDocumentData(receiver))
	}
	data := s.xmlNodeData(receiver)
	node := s.xmlDecode(data)
	switch name {
	case "XMLNode.kind":
		return node.Kind
	case "XMLNode.name":
		if node.Kind != "Element" {
			s.xmlWrongKind("name()", node.Kind)
		}
		return node.Name
	case "XMLNode.namespace":
		if node.Kind != "Element" {
			s.xmlWrongKind("namespace()", node.Kind)
		}
		return node.Namespace
	case "XMLNode.text":
		if node.Kind == "Text" {
			return node.Text
		}
		var builder strings.Builder
		for _, child := range node.Children {
			if child.Kind == "Text" {
				builder.WriteString(child.Text)
			}
		}
		return builder.String()
	case "XMLNode.attribute":
		if node.Kind != "Element" {
			s.xmlWrongKind("attribute()", node.Kind)
		}
		key := arguments[0].(string)
		for index, candidate := range node.AttrKeys {
			if candidate == key {
				return node.AttrVals[index]
			}
		}
		return nil
	case "XMLNode.attributes":
		if node.Kind != "Element" {
			s.xmlWrongKind("attributes()", node.Kind)
		}
		pair := &Pair{Keys: make([]any, len(node.AttrKeys)), Values: make(map[any]any, len(node.AttrKeys))}
		for index, key := range node.AttrKeys {
			pair.Keys[index] = key
			pair.Values[key] = node.AttrVals[index]
		}
		return pair
	case "XMLNode.children":
		if node.Kind != "Element" {
			s.xmlWrongKind("children()", node.Kind)
		}
		items := make([]any, len(node.Children))
		for index, child := range node.Children {
			items[index] = s.xmlNodeFrom(xmlEncode(child))
		}
		return &List{Items: items}
	case "XMLNode.elements":
		if node.Kind != "Element" {
			s.xmlWrongKind("elements()", node.Kind)
		}
		var items []any
		for _, child := range node.Children {
			if child.Kind == "Element" {
				items = append(items, s.xmlNodeFrom(xmlEncode(child)))
			}
		}
		return &List{Items: items}
	}
	s.raise("Error", "unsupported XMLNode operation "+name)
	return nil
}

// ---------------------------------------------------------------------------
// Parsing (duplicated from the native runtime's ahdruntime XML section by
// design; see the package comment above)
// ---------------------------------------------------------------------------

func (s *Session) xmlParseDocument(source string) xmlData {
	if len(source) > xmlMaxInputBytes {
		s.raise("XMLError", "XML input is larger than the supported limit")
	}
	if !utf8.ValidString(source) {
		s.raise("XMLError", "XML input is not valid UTF-8")
	}
	decoder := xml.NewDecoder(strings.NewReader(source))
	var root *xmlData
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			s.raise("XMLError", "XML input does not parse: "+err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			if root != nil {
				s.raise("XMLError", "XML input has more than one root element")
			}
			node := s.xmlParseElement(decoder, element, 1)
			root = &node
		case xml.CharData:
			if root == nil && strings.TrimSpace(string(element)) != "" {
				s.raise("XMLError", "XML input has content before its root element")
			}
		}
	}
	if root == nil {
		s.raise("XMLError", "XML input has no root element")
	}
	return *root
}

func (s *Session) xmlParseElement(decoder *xml.Decoder, start xml.StartElement, depth int) xmlData {
	if depth > xmlMaxDepth {
		s.raise("XMLError", "XML input exceeds the maximum supported nesting depth")
	}
	node := xmlData{Kind: "Element", Name: start.Name.Local, Namespace: start.Name.Space}
	seen := make(map[string]bool, len(start.Attr))
	for _, attr := range start.Attr {
		if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
			continue
		}
		key := attr.Name.Local
		if seen[key] {
			s.raise("XMLError", "XML element has a duplicate attribute")
		}
		seen[key] = true
		node.AttrKeys = append(node.AttrKeys, key)
		node.AttrVals = append(node.AttrVals, attr.Value)
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				s.raise("XMLError", "XML input ends before an element is closed")
			}
			s.raise("XMLError", "XML input does not parse: "+err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			node.Children = append(node.Children, s.xmlParseElement(decoder, element, depth+1))
		case xml.EndElement:
			return node
		case xml.CharData:
			if len(element) > 0 {
				node.Children = append(node.Children, xmlData{Kind: "Text", Text: string(element)})
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func xmlEscapeText(value string) string {
	var builder strings.Builder
	for _, character := range value {
		switch character {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		default:
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func xmlEscapeAttr(value string) string {
	var builder strings.Builder
	for _, character := range value {
		switch character {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '"':
			builder.WriteString("&quot;")
		case '\n':
			builder.WriteString("&#10;")
		case '\r':
			builder.WriteString("&#13;")
		case '\t':
			builder.WriteString("&#9;")
		default:
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func xmlStringifyNode(node xmlData, pretty bool, depth int) string {
	if node.Kind == "Text" {
		return xmlEscapeText(node.Text)
	}
	var builder strings.Builder
	builder.WriteByte('<')
	builder.WriteString(node.Name)
	for index, key := range node.AttrKeys {
		builder.WriteByte(' ')
		builder.WriteString(key)
		builder.WriteString(`="`)
		builder.WriteString(xmlEscapeAttr(node.AttrVals[index]))
		builder.WriteByte('"')
	}
	if len(node.Children) == 0 {
		builder.WriteString("/>")
		return builder.String()
	}
	builder.WriteByte('>')
	hasText := false
	for _, child := range node.Children {
		if child.Kind == "Text" {
			hasText = true
			break
		}
	}
	indent := func(level int) string {
		if !pretty || hasText {
			return ""
		}
		return "\n" + strings.Repeat("  ", level)
	}
	for _, child := range node.Children {
		builder.WriteString(indent(depth + 1))
		builder.WriteString(xmlStringifyNode(child, pretty, depth+1))
	}
	builder.WriteString(indent(depth))
	builder.WriteString("</")
	builder.WriteString(node.Name)
	builder.WriteByte('>')
	return builder.String()
}
