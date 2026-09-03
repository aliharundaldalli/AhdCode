package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

const (
	evaluatorHTMLNodeClass     = ir.ClassID("builtin:HTML::class::HTMLNode")
	evaluatorHTMLDocumentClass = ir.ClassID("builtin:HTML::class::HTMLDocument")
	evaluatorHTMLElementClass  = ir.ClassID("builtin:HTML::class::HTMLElement")
)

var (
	evaluatorHTMLNodeField     = ir.FieldID("builtin:HTML::class::HTMLNode::field::data")
	evaluatorHTMLDocumentField = ir.FieldID("builtin:HTML::class::HTMLDocument::field::data")
	evaluatorHTMLElementField  = ir.FieldID("builtin:HTML::class::HTMLElement::field::data")
)

func (session *Session) htmlNodeFrom(data string) *Instance {
	return &Instance{Class: evaluatorHTMLNodeClass, Fields: map[ir.FieldID]any{evaluatorHTMLNodeField: data}}
}

func (session *Session) htmlNodeData(value any) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[evaluatorHTMLNodeField].(string)
	if !ok || instance.Class != evaluatorHTMLNodeClass {
		session.raise("HTMLError", "HTMLNode storage is corrupted")
	}
	return data
}

func (session *Session) htmlDocumentFrom(data string) *Instance {
	return &Instance{Class: evaluatorHTMLDocumentClass, Fields: map[ir.FieldID]any{evaluatorHTMLDocumentField: data}}
}

func (session *Session) htmlElementFrom(data string) *Instance {
	return &Instance{Class: evaluatorHTMLElementClass, Fields: map[ir.FieldID]any{evaluatorHTMLElementField: data}}
}

func (session *Session) htmlDocumentData(value any) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[evaluatorHTMLDocumentField].(string)
	if !ok || instance.Class != evaluatorHTMLDocumentClass {
		session.raise("HTMLError", "HTMLDocument storage is corrupted")
	}
	return data
}

func (session *Session) htmlElementData(value any) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[evaluatorHTMLElementField].(string)
	if !ok || instance.Class != evaluatorHTMLElementClass {
		session.raise("HTMLError", "HTMLElement storage is corrupted")
	}
	return data
}

func (session *Session) htmlOptionalElement(value *string) any {
	if value == nil {
		return nil
	}
	return session.htmlElementFrom(*value)
}

func (session *Session) htmlElementList(items []string) *List {
	result := make([]any, len(items))
	for index, data := range items {
		result[index] = session.htmlElementFrom(data)
	}
	return &List{Items: result}
}

func (session *Session) htmlBuiltin(name string, args []any) any {
	defer session.httpRecover("HTMLError")
	class := ahdruntime.AhdClassHTMLError
	switch name {
	case "text":
		return session.htmlNodeFrom(ahdruntime.AhdHTMLText(args[0].(string)))
	case "element":
		attributes := session.requirePair(args[1])
		children := session.requireList(args[2])
		keys := make([]string, len(attributes.Keys))
		vals := make([]string, len(attributes.Keys))
		for index, key := range attributes.Keys {
			keys[index] = key.(string)
			vals[index] = attributes.Values[key].(string)
		}
		nodes := make([]string, len(children.Items))
		for index, item := range children.Items {
			nodes[index] = session.htmlNodeData(item)
		}
		return session.htmlNodeFrom(ahdruntime.AhdHTMLElement(class, args[0].(string), keys, vals, nodes))
	case "render":
		return ahdruntime.AhdHTMLRender(class, session.htmlNodeData(args[0]))
	case "document":
		body := session.requireList(args[1])
		nodes := make([]string, len(body.Items))
		for index, item := range body.Items {
			nodes[index] = session.htmlNodeData(item)
		}
		return ahdruntime.AhdHTMLDocument(class, args[0].(string), nodes)
	case "parse":
		return session.htmlDocumentFrom(ahdruntime.AhdHTMLParse(class, args[0].(string)))
	}
	session.raise("Error", "unsupported HTML function "+name)
	return nil
}

func (session *Session) htmlOperation(name string, receiver any, args []any) any {
	defer session.httpRecover("HTMLError")
	class := ahdruntime.AhdClassHTMLError
	arg := func(index int) string { return args[index].(string) }
	switch name {
	case "HTMLDocument.select":
		return session.htmlElementList(ahdruntime.AhdHTMLDocumentSelect(class, session.htmlDocumentData(receiver), arg(0)))
	case "HTMLDocument.first":
		return session.htmlOptionalElement(ahdruntime.AhdHTMLDocumentFirst(class, session.htmlDocumentData(receiver), arg(0)))
	case "HTMLElement.tag":
		return ahdruntime.AhdHTMLElementTag(class, session.htmlElementData(receiver))
	case "HTMLElement.text":
		return ahdruntime.AhdHTMLElementText(class, session.htmlElementData(receiver))
	case "HTMLElement.attr":
		return session.htmlOptionalString(ahdruntime.AhdHTMLElementAttr(class, session.htmlElementData(receiver), arg(0)))
	case "HTMLElement.hasAttr":
		return ahdruntime.AhdHTMLElementHasAttr(class, session.htmlElementData(receiver), arg(0))
	case "HTMLElement.select":
		return session.htmlElementList(ahdruntime.AhdHTMLElementSelect(class, session.htmlElementData(receiver), arg(0)))
	case "HTMLElement.first":
		return session.htmlOptionalElement(ahdruntime.AhdHTMLElementFirst(class, session.htmlElementData(receiver), arg(0)))
	}
	session.raise("Error", "unsupported HTMLElement operation "+name)
	return nil
}

func (session *Session) htmlOptionalString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
