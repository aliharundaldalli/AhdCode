package ahdruntime

import (
	"encoding/json"
	"html"
	"strings"
	"unicode/utf8"
)

// The HTML standard module: a small safe structured HTML builder. Nodes are
// immutable values encoded as private JSON. There is no DOM, parser, CSS
// engine, or raw-node escape hatch. Dynamic text and attribute values are
// escaped with html.EscapeString.

var ahdHTMLVoid = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

type ahdHTMLNode struct {
	Kind     string        `json:"k"`
	Text     string        `json:"t,omitempty"`
	Name     string        `json:"n,omitempty"`
	AttrKeys []string      `json:"ak,omitempty"`
	AttrVals []string      `json:"av,omitempty"`
	Children []ahdHTMLNode `json:"c,omitempty"`
}

func AhdHTMLText(value string) string {
	encoded, _ := json.Marshal(ahdHTMLNode{Kind: "text", Text: value})
	return string(encoded)
}

func AhdHTMLElement(class *AhdClass, name string, attrKeys, attrVals, children []string) string {
	if !ahdHTMLTagNameOK(name) {
		AhdRaiseClass(class, "HTML element name "+ahdHTMLQuote(name)+" is not a valid tag name")
	}
	if len(attrKeys) != len(attrVals) {
		AhdRaiseClass(class, "HTML element attributes are malformed")
	}
	seen := make(map[string]bool, len(attrKeys))
	for _, key := range attrKeys {
		if !ahdHTMLAttributeNameOK(key) {
			AhdRaiseClass(class, "HTML attribute name "+ahdHTMLQuote(key)+" is not a valid attribute name")
		}
		if seen[key] {
			AhdRaiseClass(class, "HTML element has a duplicate attribute "+ahdHTMLQuote(key))
		}
		seen[key] = true
	}
	nodes := make([]ahdHTMLNode, len(children))
	for index, child := range children {
		nodes[index] = ahdHTMLDecode(class, child)
	}
	if ahdHTMLVoid[strings.ToLower(name)] && len(nodes) != 0 {
		AhdRaiseClass(class, "HTML void element "+name+" cannot have child content")
	}
	encoded, _ := json.Marshal(ahdHTMLNode{
		Kind: "element", Name: name,
		AttrKeys: append([]string(nil), attrKeys...),
		AttrVals: append([]string(nil), attrVals...),
		Children: nodes,
	})
	return string(encoded)
}

func AhdHTMLRender(class *AhdClass, data string) string {
	return ahdHTMLRenderNode(ahdHTMLDecode(class, data))
}

func AhdHTMLDocument(class *AhdClass, title string, body []string) string {
	var builder strings.Builder
	builder.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>")
	builder.WriteString(html.EscapeString(title))
	builder.WriteString("</title></head><body>")
	for _, child := range body {
		builder.WriteString(ahdHTMLRenderNode(ahdHTMLDecode(class, child)))
	}
	builder.WriteString("</body></html>")
	return builder.String()
}

func ahdHTMLRenderNode(node ahdHTMLNode) string {
	if node.Kind == "text" {
		return html.EscapeString(node.Text)
	}
	var builder strings.Builder
	builder.WriteByte('<')
	builder.WriteString(node.Name)
	for index, key := range node.AttrKeys {
		builder.WriteByte(' ')
		builder.WriteString(key)
		builder.WriteString(`="`)
		builder.WriteString(html.EscapeString(node.AttrVals[index]))
		builder.WriteByte('"')
	}
	if ahdHTMLVoid[strings.ToLower(node.Name)] {
		builder.WriteByte('>')
		return builder.String()
	}
	builder.WriteByte('>')
	for _, child := range node.Children {
		builder.WriteString(ahdHTMLRenderNode(child))
	}
	builder.WriteString("</")
	builder.WriteString(node.Name)
	builder.WriteByte('>')
	return builder.String()
}

func ahdHTMLTagNameOK(name string) bool {
	return ahdHTMLNameOK(name, false)
}

func ahdHTMLAttributeNameOK(name string) bool {
	return ahdHTMLNameOK(name, true)
}

func ahdHTMLNameOK(name string, attribute bool) bool {
	if name == "" {
		return false
	}
	for index, r := range name {
		if index == 0 {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				return false
			}
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		if attribute && r == '_' {
			continue
		}
		return false
	}
	return true
}

func ahdHTMLDecode(class *AhdClass, data string) ahdHTMLNode {
	var node ahdHTMLNode
	if err := json.Unmarshal([]byte(data), &node); err != nil || (node.Kind != "text" && node.Kind != "element") {
		AhdRaiseClass(class, "HTML node storage is corrupted")
	}
	return node
}

func ahdHTMLQuote(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range value {
		if r == '"' || r == '\\' {
			builder.WriteByte('\\')
		}
		builder.WriteRune(r)
	}
	builder.WriteByte('"')
	return builder.String()
}

// The HTML parsing half of the HTML standard module: HTML.parse and the
// HTMLDocument/HTMLElement query surface. This is a distinct concept from
// the builder's HTMLNode (html.go) - a parsed HTMLElement is never silently
// convertible to or from a builder HTMLNode.
//
// A hand-written tokenizer and tree builder is used rather than a
// regex-based scanner, giving browser-like recovery for ordinary malformed
// markup (unclosed tags are force-closed at end of input or by a later
// mismatched end tag; void elements never accept children; script/style
// content is captured as raw, undecoded text per the HTML5 "raw text
// element" rule). It is not a validator: syntactically scannable HTML never
// fails to parse, it only produces the most reasonable tree recoverable from
// the input.
//
// HTML.parse never fetches a network resource, never resolves a URL, and
// never executes script content - it only tokenizes the given String and
// builds a tree from it. A <script> or <img src="..."> is ordinary,
// unreachable markup exactly like any other element.
//
// Like XMLNode/XMLDocument, a parsed HTMLDocument/HTMLElement is at rest one
// hidden String field holding a private, opaque JSON encoding of the node's
// own subtree (never published, never valid HTML on its own). Because the
// encoding is self-contained, an HTMLElement obtained from a HTMLDocument
// stays valid independent of that HTMLDocument's own lifetime, with no
// shared mutable registry or handle to manage.
const (
	ahdHTMLParseMaxInputBytes = 8 * 1024 * 1024
	ahdHTMLParseMaxDepth      = 256
)

// ahdHTMLParsedAttr is one attribute as parsed: Name is always lowercased
// (HTML attribute names are matched case-insensitively), Value is exactly
// the decoded attribute value with no case normalization.
type ahdHTMLParsedAttr struct {
	Name  string `json:"n"`
	Value string `json:"v"`
}

// ahdHTMLParsedNode is the parsed-tree representation for HTMLDocument
// ("document") and HTMLElement ("element"/"text" children), and is also
// exactly what an HTMLDocument's/HTMLElement's hidden field encodes. Tag is
// always the normalized (lowercased) tag name; there is no way to recover
// the source's original tag capitalization, by design (tag() semantics).
type ahdHTMLParsedNode struct {
	Kind     string               `json:"k"`
	Tag      string               `json:"t,omitempty"`
	Attrs    []ahdHTMLParsedAttr  `json:"a,omitempty"`
	Text     string               `json:"x,omitempty"`
	Children []*ahdHTMLParsedNode `json:"c,omitempty"`
}

var ahdHTMLRawTextElements = map[string]bool{"script": true, "style": true}

func ahdHTMLEncode(node *ahdHTMLParsedNode) string {
	encoded, _ := json.Marshal(node)
	return string(encoded)
}

func ahdHTMLParseDecode(class *AhdClass, data string) *ahdHTMLParsedNode {
	var node ahdHTMLParsedNode
	if err := json.Unmarshal([]byte(data), &node); err != nil {
		AhdRaiseClass(class, "HTML node storage is corrupted")
	}
	return &node
}

// ---------------------------------------------------------------------------
// Tokenizing and tree building
// ---------------------------------------------------------------------------

func ahdHTMLParseDocument(class *AhdClass, source string) *ahdHTMLParsedNode {
	if len(source) > ahdHTMLParseMaxInputBytes {
		AhdRaiseClass(class, "HTML input is larger than the supported limit")
	}
	if !utf8.ValidString(source) {
		AhdRaiseClass(class, "HTML input is not valid UTF-8")
	}
	root := &ahdHTMLParsedNode{Kind: "document"}
	stack := []*ahdHTMLParsedNode{root}
	pos := 0
	n := len(source)
	for pos < n {
		lt := strings.IndexByte(source[pos:], '<')
		if lt < 0 {
			ahdHTMLAppendText(stack[len(stack)-1], html.UnescapeString(source[pos:]))
			break
		}
		if lt > 0 {
			ahdHTMLAppendText(stack[len(stack)-1], html.UnescapeString(source[pos:pos+lt]))
			pos += lt
		}
		rest := source[pos:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			if end := strings.Index(rest, "-->"); end < 0 {
				pos = n
			} else {
				pos += end + 3
			}
		case len(rest) >= 2 && rest[1] == '!':
			if end := strings.IndexByte(rest, '>'); end < 0 {
				pos = n
			} else {
				pos += end + 1
			}
		case len(rest) >= 2 && rest[1] == '/':
			name, consumed := ahdHTMLReadEndTagName(rest)
			pos += consumed
			if name != "" {
				ahdHTMLCloseTag(&stack, name)
			}
		case len(rest) >= 2 && ahdHTMLIsNameStart(rest[1]):
			tag, attrs, selfClosing, consumed := ahdHTMLReadStartTag(rest)
			pos += consumed
			if len(stack) >= ahdHTMLParseMaxDepth {
				AhdRaiseClass(class, "HTML input exceeds the maximum supported nesting depth")
			}
			element := &ahdHTMLParsedNode{Kind: "element", Tag: tag, Attrs: attrs}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, element)
			void := ahdHTMLVoid[tag]
			if !void && !selfClosing {
				stack = append(stack, element)
			}
			if !void && ahdHTMLRawTextElements[tag] {
				text, consumed2 := ahdHTMLReadRawText(source[pos:], tag)
				pos += consumed2
				if text != "" {
					element.Children = append(element.Children, &ahdHTMLParsedNode{Kind: "text", Text: text})
				}
				stack = stack[:len(stack)-1]
			}
		default:
			ahdHTMLAppendText(stack[len(stack)-1], "<")
			pos++
		}
	}
	return root
}

func ahdHTMLAppendText(parent *ahdHTMLParsedNode, text string) {
	if text == "" {
		return
	}
	parent.Children = append(parent.Children, &ahdHTMLParsedNode{Kind: "text", Text: text})
}

// ahdHTMLCloseTag pops the open-element stack up to and including the
// nearest open element named name. A close tag with no matching open
// element is a stray tag and is ignored, matching typical lenient recovery.
func ahdHTMLCloseTag(stack *[]*ahdHTMLParsedNode, name string) {
	s := *stack
	for i := len(s) - 1; i >= 1; i-- {
		if s[i].Tag == name {
			*stack = s[:i]
			return
		}
	}
}

func ahdHTMLIsNameStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func ahdHTMLIsNameChar(b byte) bool {
	return ahdHTMLIsNameStart(b) || (b >= '0' && b <= '9') || b == '-'
}

func ahdHTMLIsSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// ahdHTMLReadEndTagName parses "</name ... >" starting at s[0]=='<', s[1]=='/'.
func ahdHTMLReadEndTagName(s string) (name string, consumed int) {
	i := 2
	start := i
	for i < len(s) && ahdHTMLIsNameChar(s[i]) {
		i++
	}
	name = strings.ToLower(s[start:i])
	end := strings.IndexByte(s[i:], '>')
	if end < 0 {
		return name, len(s)
	}
	return name, i + end + 1
}

// ahdHTMLReadStartTag parses "<name attr attr="v" ... >" or its
// self-closing "/>" form, starting at s[0]=='<'.
func ahdHTMLReadStartTag(s string) (tag string, attrs []ahdHTMLParsedAttr, selfClosing bool, consumed int) {
	i := 1
	start := i
	for i < len(s) && ahdHTMLIsNameChar(s[i]) {
		i++
	}
	tag = strings.ToLower(s[start:i])
	seen := make(map[string]bool)
	for {
		for i < len(s) && ahdHTMLIsSpace(s[i]) {
			i++
		}
		if i >= len(s) {
			return tag, attrs, selfClosing, i
		}
		if s[i] == '/' {
			selfClosing = true
			i++
			continue
		}
		if s[i] == '>' {
			i++
			return tag, attrs, selfClosing, i
		}
		nameStart := i
		for i < len(s) && !ahdHTMLIsSpace(s[i]) && s[i] != '=' && s[i] != '>' && s[i] != '/' {
			i++
		}
		if i == nameStart {
			i++
			continue
		}
		attrName := strings.ToLower(s[nameStart:i])
		for i < len(s) && ahdHTMLIsSpace(s[i]) {
			i++
		}
		value := ""
		if i < len(s) && s[i] == '=' {
			i++
			for i < len(s) && ahdHTMLIsSpace(s[i]) {
				i++
			}
			if i < len(s) && (s[i] == '"' || s[i] == '\'') {
				quote := s[i]
				i++
				valueStart := i
				for i < len(s) && s[i] != quote {
					i++
				}
				value = html.UnescapeString(s[valueStart:i])
				if i < len(s) {
					i++
				}
			} else {
				valueStart := i
				for i < len(s) && !ahdHTMLIsSpace(s[i]) && s[i] != '>' {
					i++
				}
				value = html.UnescapeString(s[valueStart:i])
			}
		}
		if attrName != "" && !seen[attrName] {
			seen[attrName] = true
			attrs = append(attrs, ahdHTMLParsedAttr{Name: attrName, Value: value})
		}
	}
}

// ahdHTMLReadRawText reads the literal (non-decoded, non-tag-parsing)
// content of a <script> or <style> element, up to and including its
// matching end tag, per the HTML5 "raw text element" rule.
func ahdHTMLReadRawText(s string, tag string) (text string, consumed int) {
	closeTag := "</" + tag
	lower := strings.ToLower(s)
	searchFrom := 0
	for {
		idx := strings.Index(lower[searchFrom:], closeTag)
		if idx < 0 {
			return s, len(s)
		}
		idx += searchFrom
		after := idx + len(closeTag)
		if after >= len(s) || ahdHTMLIsSpace(s[after]) || s[after] == '>' || s[after] == '/' {
			text = s[:idx]
			end := strings.IndexByte(s[after:], '>')
			if end < 0 {
				return text, len(s)
			}
			return text, after + end + 1
		}
		searchFrom = idx + len(closeTag)
	}
}

// ---------------------------------------------------------------------------
// HTMLDocument / HTMLElement public accessors
// ---------------------------------------------------------------------------

func AhdHTMLParse(class *AhdClass, source string) string {
	return ahdHTMLEncode(ahdHTMLParseDocument(class, source))
}

func AhdHTMLDocumentSelect(class *AhdClass, documentData, selector string) []string {
	doc := ahdHTMLRequireKind(class, documentData, "document", "HTMLDocument")
	return ahdHTMLEncodeAll(ahdHTMLSelectAll(doc, ahdHTMLParseSelector(class, selector)))
}

func AhdHTMLDocumentFirst(class *AhdClass, documentData, selector string) *string {
	doc := ahdHTMLRequireKind(class, documentData, "document", "HTMLDocument")
	return ahdHTMLEncodeFirst(ahdHTMLSelectFirst(doc, ahdHTMLParseSelector(class, selector)))
}

func AhdHTMLElementSelect(class *AhdClass, elementData, selector string) []string {
	element := ahdHTMLRequireKind(class, elementData, "element", "HTMLElement")
	return ahdHTMLEncodeAll(ahdHTMLSelectAll(element, ahdHTMLParseSelector(class, selector)))
}

func AhdHTMLElementFirst(class *AhdClass, elementData, selector string) *string {
	element := ahdHTMLRequireKind(class, elementData, "element", "HTMLElement")
	return ahdHTMLEncodeFirst(ahdHTMLSelectFirst(element, ahdHTMLParseSelector(class, selector)))
}

func AhdHTMLElementTag(class *AhdClass, elementData string) string {
	return ahdHTMLRequireKind(class, elementData, "element", "HTMLElement").Tag
}

func AhdHTMLElementText(class *AhdClass, elementData string) string {
	element := ahdHTMLRequireKind(class, elementData, "element", "HTMLElement")
	var builder strings.Builder
	ahdHTMLCollectText(element, &builder)
	return builder.String()
}

func AhdHTMLElementAttr(class *AhdClass, elementData, name string) *string {
	element := ahdHTMLRequireKind(class, elementData, "element", "HTMLElement")
	value, ok := element.attrValue(strings.ToLower(name))
	if !ok {
		return nil
	}
	return &value
}

func AhdHTMLElementHasAttr(class *AhdClass, elementData, name string) bool {
	element := ahdHTMLRequireKind(class, elementData, "element", "HTMLElement")
	_, ok := element.attrValue(strings.ToLower(name))
	return ok
}

func ahdHTMLRequireKind(class *AhdClass, data, kind, label string) *ahdHTMLParsedNode {
	node := ahdHTMLParseDecode(class, data)
	if node.Kind != kind {
		AhdRaiseClass(class, label+" storage is corrupted")
	}
	return node
}

func ahdHTMLEncodeAll(nodes []*ahdHTMLParsedNode) []string {
	result := make([]string, len(nodes))
	for index, node := range nodes {
		result[index] = ahdHTMLEncode(node)
	}
	return result
}

func ahdHTMLEncodeFirst(node *ahdHTMLParsedNode) *string {
	if node == nil {
		return nil
	}
	encoded := ahdHTMLEncode(node)
	return &encoded
}

func ahdHTMLCollectText(node *ahdHTMLParsedNode, builder *strings.Builder) {
	for _, child := range node.Children {
		switch child.Kind {
		case "text":
			builder.WriteString(child.Text)
		case "element":
			ahdHTMLCollectText(child, builder)
		}
	}
}

func (node *ahdHTMLParsedNode) attrValue(name string) (string, bool) {
	for _, attr := range node.Attrs {
		if attr.Name == name {
			return attr.Value, true
		}
	}
	return "", false
}

func (node *ahdHTMLParsedNode) classTokens() []string {
	value, ok := node.attrValue("class")
	if !ok {
		return nil
	}
	return strings.Fields(value)
}

// The v0.7 frozen CSS-like selector subset: universal (*), tag, #id, .class,
// compound selectors (concatenated with no combinator, e.g. "article.card"),
// attribute presence ([href]) and exact value ([rel="next"]) with quoted
// values, the descendant (space) and direct child (>) combinators, and
// comma-separated selector lists. Anything outside this subset - pseudo
// classes/elements, sibling combinators, other attribute operators, CSS
// escapes, XPath - is rejected with HTMLError rather than approximated.

type ahdHTMLAttrExact struct {
	Name  string
	Value string
}

// ahdHTMLCompound is one compound selector: every set field must match for
// the compound to match a node. An empty Tag means "no tag constraint" (as
// in bare "*" or a bare "[href]"), not "matches nothing".
type ahdHTMLCompound struct {
	Tag          string
	ID           string
	Classes      []string
	AttrPresence []string
	AttrExact    []ahdHTMLAttrExact
}

// ahdHTMLComplex is a chain of compounds joined by combinators.
// Combinators[i] describes the relationship between Compounds[i] and
// Compounds[i+1] ('>' for direct child, ' ' for descendant).
type ahdHTMLComplex struct {
	Compounds   []ahdHTMLCompound
	Combinators []byte
}

type ahdHTMLSelectorList struct {
	Complex []ahdHTMLComplex
}

type ahdHTMLSelectorParser struct {
	class *AhdClass
	runes []rune
	pos   int
}

func ahdHTMLParseSelector(class *AhdClass, source string) ahdHTMLSelectorList {
	parser := &ahdHTMLSelectorParser{class: class, runes: []rune(source)}
	list := parser.parseList()
	parser.skipSpace()
	if parser.pos != len(parser.runes) {
		parser.fail("unexpected trailing text")
	}
	return list
}

func (p *ahdHTMLSelectorParser) fail(message string) {
	AhdRaiseClass(p.class, "HTML selector is invalid: "+message)
}

func (p *ahdHTMLSelectorParser) peek() (rune, bool) {
	if p.pos >= len(p.runes) {
		return 0, false
	}
	return p.runes[p.pos], true
}

func (p *ahdHTMLSelectorParser) skipSpace() bool {
	start := p.pos
	for p.pos < len(p.runes) && isHTMLSelectorSpace(p.runes[p.pos]) {
		p.pos++
	}
	return p.pos > start
}

func isHTMLSelectorSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
}

func (p *ahdHTMLSelectorParser) parseList() ahdHTMLSelectorList {
	var list ahdHTMLSelectorList
	p.skipSpace()
	list.Complex = append(list.Complex, p.parseComplex())
	for {
		p.skipSpace()
		r, ok := p.peek()
		if !ok || r != ',' {
			break
		}
		p.pos++
		p.skipSpace()
		list.Complex = append(list.Complex, p.parseComplex())
	}
	return list
}

func (p *ahdHTMLSelectorParser) parseComplex() ahdHTMLComplex {
	var complex ahdHTMLComplex
	first, ok := p.parseCompound()
	if !ok {
		p.fail("expected a selector")
	}
	complex.Compounds = append(complex.Compounds, first)
	for {
		hadSpace := p.skipSpace()
		r, ok := p.peek()
		if !ok || r == ',' {
			break
		}
		if r == '>' {
			p.pos++
			p.skipSpace()
			next, ok := p.parseCompound()
			if !ok {
				p.fail("expected a selector after '>'")
			}
			complex.Combinators = append(complex.Combinators, '>')
			complex.Compounds = append(complex.Compounds, next)
			continue
		}
		if hadSpace {
			next, ok := p.parseCompound()
			if !ok {
				p.fail("expected a selector")
			}
			complex.Combinators = append(complex.Combinators, ' ')
			complex.Compounds = append(complex.Compounds, next)
			continue
		}
		p.fail("unexpected character")
	}
	return complex
}

func (p *ahdHTMLSelectorParser) parseCompound() (ahdHTMLCompound, bool) {
	var compound ahdHTMLCompound
	found := false
	for {
		r, ok := p.peek()
		if !ok {
			break
		}
		switch {
		case r == '*':
			p.pos++
			found = true
		case isHTMLIdentStart(r):
			compound.Tag = strings.ToLower(p.readIdent())
			found = true
		case r == '#':
			p.pos++
			ident := p.readIdent()
			if ident == "" {
				p.fail("expected an id after '#'")
			}
			compound.ID = ident
			found = true
		case r == '.':
			p.pos++
			ident := p.readIdent()
			if ident == "" {
				p.fail("expected a class name after '.'")
			}
			compound.Classes = append(compound.Classes, ident)
			found = true
		case r == '[':
			p.pos++
			p.parseAttrSelector(&compound)
			found = true
		default:
			return compound, found
		}
	}
	return compound, found
}

func isHTMLIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isHTMLIdentChar(r rune) bool {
	return isHTMLIdentStart(r) || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

func (p *ahdHTMLSelectorParser) readIdent() string {
	start := p.pos
	for p.pos < len(p.runes) && isHTMLIdentChar(p.runes[p.pos]) {
		p.pos++
	}
	return string(p.runes[start:p.pos])
}

func (p *ahdHTMLSelectorParser) parseAttrSelector(compound *ahdHTMLCompound) {
	name := p.readIdent()
	if name == "" {
		p.fail("expected an attribute name after '['")
	}
	name = strings.ToLower(name)
	if r, ok := p.peek(); ok && r == '=' {
		p.pos++
		r, ok := p.peek()
		if !ok || (r != '"' && r != '\'') {
			p.fail("expected a quoted value after '='")
		}
		quote := r
		p.pos++
		start := p.pos
		for p.pos < len(p.runes) && p.runes[p.pos] != quote {
			p.pos++
		}
		if p.pos >= len(p.runes) {
			p.fail("unterminated attribute value")
		}
		value := string(p.runes[start:p.pos])
		p.pos++
		compound.AttrExact = append(compound.AttrExact, ahdHTMLAttrExact{Name: name, Value: value})
	} else {
		compound.AttrPresence = append(compound.AttrPresence, name)
	}
	r, ok := p.peek()
	if !ok || r != ']' {
		p.fail("expected ']'")
	}
	p.pos++
}

// ---------------------------------------------------------------------------
// Matching
// ---------------------------------------------------------------------------

func ahdHTMLCompoundMatches(node *ahdHTMLParsedNode, compound ahdHTMLCompound) bool {
	if node.Kind != "element" {
		return false
	}
	if compound.Tag != "" && node.Tag != compound.Tag {
		return false
	}
	if compound.ID != "" {
		id, ok := node.attrValue("id")
		if !ok || id != compound.ID {
			return false
		}
	}
	if len(compound.Classes) > 0 {
		tokens := node.classTokens()
		for _, want := range compound.Classes {
			matched := false
			for _, token := range tokens {
				if token == want {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	for _, name := range compound.AttrPresence {
		if _, ok := node.attrValue(name); !ok {
			return false
		}
	}
	for _, exact := range compound.AttrExact {
		value, ok := node.attrValue(exact.Name)
		if !ok || value != exact.Value {
			return false
		}
	}
	return true
}

func ahdHTMLComplexMatchesAt(complex ahdHTMLComplex, node *ahdHTMLParsedNode, ancestors []*ahdHTMLParsedNode) bool {
	last := len(complex.Compounds) - 1
	if !ahdHTMLCompoundMatches(node, complex.Compounds[last]) {
		return false
	}
	return ahdHTMLChainSatisfied(complex, last-1, ancestors)
}

func ahdHTMLChainSatisfied(complex ahdHTMLComplex, index int, ancestors []*ahdHTMLParsedNode) bool {
	if index < 0 {
		return true
	}
	combinator := complex.Combinators[index]
	if len(ancestors) == 0 {
		return false
	}
	parent := ancestors[len(ancestors)-1]
	if combinator == '>' {
		if !ahdHTMLCompoundMatches(parent, complex.Compounds[index]) {
			return false
		}
		return ahdHTMLChainSatisfied(complex, index-1, ancestors[:len(ancestors)-1])
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		if ahdHTMLCompoundMatches(ancestors[i], complex.Compounds[index]) {
			if ahdHTMLChainSatisfied(complex, index-1, ancestors[:i]) {
				return true
			}
		}
	}
	return false
}

// ahdHTMLSelectAll returns every descendant of root matching any complex
// selector in list, in document order, de-duplicated (an element matching
// more than one branch of a selector list is reported once, at its first
// document-order position).
func ahdHTMLSelectAll(root *ahdHTMLParsedNode, list ahdHTMLSelectorList) []*ahdHTMLParsedNode {
	var results []*ahdHTMLParsedNode
	var walk func(node *ahdHTMLParsedNode, ancestors []*ahdHTMLParsedNode)
	walk = func(node *ahdHTMLParsedNode, ancestors []*ahdHTMLParsedNode) {
		for _, child := range node.Children {
			if child.Kind != "element" {
				continue
			}
			for _, complex := range list.Complex {
				if ahdHTMLComplexMatchesAt(complex, child, ancestors) {
					results = append(results, child)
					break
				}
			}
			walk(child, append(ancestors, child))
		}
	}
	walk(root, nil)
	return results
}

// ahdHTMLSelectFirst is the first element document-order would report from
// ahdHTMLSelectAll, or nil. It is defined directly in terms of
// ahdHTMLSelectAll so the two can never disagree on which elements match.
func ahdHTMLSelectFirst(root *ahdHTMLParsedNode, list ahdHTMLSelectorList) *ahdHTMLParsedNode {
	all := ahdHTMLSelectAll(root, list)
	if len(all) == 0 {
		return nil
	}
	return all[0]
}
