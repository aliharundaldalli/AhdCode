package ahdruntime

import (
	"encoding/json"
	"html"
	"strings"
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
