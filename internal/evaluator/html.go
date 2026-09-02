package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

const evaluatorHTMLNodeClass = ir.ClassID("builtin:HTML::class::HTMLNode")

var evaluatorHTMLNodeField = ir.FieldID("builtin:HTML::class::HTMLNode::field::data")

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
	}
	session.raise("Error", "unsupported HTML function "+name)
	return nil
}
