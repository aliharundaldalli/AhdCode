package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

const (
	evaluatorTemplateDataClass = ir.ClassID("builtin:Templates::class::TemplateData")
	evaluatorTemplateSetClass  = ir.ClassID("builtin:Templates::class::TemplateSet")
)

var (
	evaluatorTemplateDataField = ir.FieldID("builtin:Templates::class::TemplateData::field::text")
	evaluatorTemplateSetField  = ir.FieldID("builtin:Templates::class::TemplateSet::field::text")
)

func (session *Session) templateDataFrom(text string) *Instance {
	return &Instance{Class: evaluatorTemplateDataClass, Fields: map[ir.FieldID]any{evaluatorTemplateDataField: text}}
}

func (session *Session) templateSetFrom(text string) *Instance {
	return &Instance{Class: evaluatorTemplateSetClass, Fields: map[ir.FieldID]any{evaluatorTemplateSetField: text}}
}

func (session *Session) templateText(value any, class ir.ClassID, field ir.FieldID) string {
	instance := session.requireInstance(value)
	text, ok := instance.Fields[field].(string)
	if !ok || instance.Class != class {
		session.raise("TemplateError", "template storage is corrupted")
	}
	return text
}

func (session *Session) templatesBuiltin(name string, args []any) any {
	defer session.httpRecover("TemplateError")
	class := ahdruntime.AhdClassTemplateError
	switch name {
	case "nullValue":
		return session.templateDataFrom(ahdruntime.AhdTemplatesNull())
	case "fromBool":
		return session.templateDataFrom(ahdruntime.AhdTemplatesFromBool(args[0].(bool)))
	case "fromInt":
		return session.templateDataFrom(ahdruntime.AhdTemplatesFromInt(args[0].(int64)))
	case "fromReal":
		return session.templateDataFrom(ahdruntime.AhdTemplatesFromReal(class, args[0].(float64)))
	case "fromString":
		return session.templateDataFrom(ahdruntime.AhdTemplatesFromString(args[0].(string)))
	case "trusted":
		return session.templateDataFrom(ahdruntime.AhdTemplatesTrusted(args[0].(string)))
	case "list":
		items := session.requireList(args[0])
		texts := make([]string, len(items.Items))
		for index, item := range items.Items {
			texts[index] = session.templateText(item, evaluatorTemplateDataClass, evaluatorTemplateDataField)
		}
		return session.templateDataFrom(ahdruntime.AhdTemplatesList(texts))
	case "object":
		pair := session.requirePair(args[0])
		keys := make([]string, len(pair.Keys))
		texts := make([]string, len(pair.Keys))
		for index, key := range pair.Keys {
			keys[index] = key.(string)
			texts[index] = session.templateText(pair.Values[key], evaluatorTemplateDataClass, evaluatorTemplateDataField)
		}
		return session.templateDataFrom(ahdruntime.AhdTemplatesObject(class, keys, texts))
	case "parse":
		return session.templateSetFrom(ahdruntime.AhdTemplatesParse(class, args[0].(string), args[1].(string)))
	case "sources":
		names := session.requireList(args[0])
		bodies := session.requireList(args[1])
		nameText := make([]string, len(names.Items))
		bodyText := make([]string, len(bodies.Items))
		for index, item := range names.Items {
			nameText[index] = item.(string)
		}
		for index, item := range bodies.Items {
			bodyText[index] = item.(string)
		}
		return session.templateSetFrom(ahdruntime.AhdTemplatesSources(class, nameText, bodyText))
	case "files":
		return session.templateSetFrom(ahdruntime.AhdTemplatesFiles(class, args[0].(string)))
	default:
		session.raise("TemplateError", "unknown Templates function")
		return nil
	}
}

func (session *Session) templatesOperation(name string, receiver any, args []any) any {
	defer session.httpRecover("TemplateError")
	class := ahdruntime.AhdClassTemplateError
	handle := session.templateText(receiver, evaluatorTemplateSetClass, evaluatorTemplateSetField)
	switch name {
	case "TemplateSet.render":
		data := session.templateText(args[1], evaluatorTemplateDataClass, evaluatorTemplateDataField)
		return ahdruntime.AhdTemplatesRender(class, handle, args[0].(string), data)
	case "TemplateSet.layout":
		data := session.templateText(args[2], evaluatorTemplateDataClass, evaluatorTemplateDataField)
		return ahdruntime.AhdTemplatesLayout(class, handle, args[0].(string), args[1].(string), data)
	default:
		session.raise("TemplateError", "unknown template operation")
		return nil
	}
}
