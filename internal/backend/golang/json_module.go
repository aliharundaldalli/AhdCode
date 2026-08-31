package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const jsonModulePrefix = "builtin:JSON::"

var (
	jsonValueClass     = ir.ClassID("builtin:JSON::class::JSONValue")
	jsonValueTextField = ir.FieldID("builtin:JSON::class::JSONValue::field::text")
	jsonErrorClass     = ir.ClassID("builtin:JSON::class::JSONError")
)

// jsonCall lowers the JSON module's plain functions.
func (generator *generator) jsonCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), jsonModulePrefix)
	errorClass := generator.descriptorName(jsonErrorClass)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	boolean := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	integer := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	real := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.RealType}, false)
	}
	switch name {
	case "parse":
		return generator.jsonValueFrom("AhdJSONParse("+errorClass+", "+text(0, `""`)+")", meta)
	case "read":
		return generator.jsonValueFrom("AhdJSONRead("+errorClass+", "+text(0, `""`)+")", meta)
	case "nullValue":
		return generator.jsonValueFrom("AhdJSONNull()", meta)
	case "fromBool":
		return generator.jsonValueFrom("AhdJSONFromBool("+boolean(0, "false")+")", meta)
	case "fromInt":
		return generator.jsonValueFrom("AhdJSONFromInt("+integer(0, "int64(0)")+")", meta)
	case "fromReal":
		return generator.jsonValueFrom("AhdJSONFromReal("+errorClass+", "+real(0, "float64(0)")+")", meta)
	case "fromString":
		return generator.jsonValueFrom("AhdJSONFromString("+text(0, `""`)+")", meta)
	case "array":
		return generator.jsonValueFrom("AhdJSONArray("+generator.jsonValueTexts(value, 0)+")", meta)
	case "object":
		return generator.jsonValueFrom("AhdJSONObject("+errorClass+", "+generator.jsonValuePairEntries(value, 0)+")", meta)
	case "stringify":
		return "AhdJSONStringify(" + generator.jsonValueOf(value.Arguments[0].Value) + ", " + boolean(1, "false") + ")"
	case "write":
		return "AhdJSONWrite(" + errorClass + ", " + generator.jsonValueOf(value.Arguments[0].Value) + ", " +
			text(1, `""`) + ", " + boolean(2, "false") + ")"
	default:
		return generator.unsupported("JSON function "+name, meta.Span)
	}
}

// jsonValueFrom wraps one canonical JSON text expression into a JSONValue
// instance through the generated class helper, the same way Word's
// documentFrom wraps one AhdWordDocument reading into a Document instance.
func (generator *generator) jsonValueFrom(text string, meta ir.ExprBase) string {
	helper, ok := generator.jsonValueHelper()
	if !ok {
		return generator.unsupported("a JSONValue without its Class declaration", meta.Span)
	}
	return helper + "(" + text + ")"
}

// jsonValueOf evaluates one JSONValue expression exactly once and reads its
// one hidden text field.
func (generator *generator) jsonValueOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	text := "value." + generator.fieldName(jsonValueTextField) + "_get()"
	return "func(value " + generator.interfaceName(jsonValueClass) + ") string { return " + text + " }(" + rendered + ")"
}

// jsonValueTexts evaluates a List<JSONValue> argument exactly once and
// returns a []string of every element's own canonical text, in order.
func (generator *generator) jsonValueTexts(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(jsonValueClass)
	getter := generator.fieldName(jsonValueTextField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { " +
		"items := list.Snapshot(); result := make([]string, len(items)); " +
		"for index, item := range items { result[index] = item." + getter + " }; " +
		"return result }(" + rendered + ")"
}

// jsonValuePairEntries evaluates a Pair<String, JSONValue> argument exactly
// once and returns an []AhdJSONEntry of every key and its value's own
// canonical text, in insertion order.
func (generator *generator) jsonValuePairEntries(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(jsonValueClass)
	getter := generator.fieldName(jsonValueTextField) + "_get()"
	return "func(pair *AhdPair[string, " + element + "]) []AhdJSONEntry { " +
		"keys := pair.Keys(); entries := make([]AhdJSONEntry, len(keys)); " +
		"for index, key := range keys { entries[index] = AhdJSONEntry{Key: key, Text: pair.Get(key)." + getter + "} }; " +
		"return entries }(" + rendered + ")"
}

func (generator *generator) jsonValueHelper() (string, bool) {
	if generator.layouts[jsonValueClass] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[jsonValueClass]; known {
		return name, true
	}
	name := mangleNamed("jh_", generator.classDisplayName(jsonValueClass), string(jsonValueClass))
	generator.timeHelpers[jsonValueClass] = name
	return name, true
}

// emitJSONValueHelpers writes the JSONValue wrapper, turning one canonical
// text reading into a constructed AhdCode value.
func (generator *generator) emitJSONValueHelpers(writer *emitter) {
	name, known := generator.timeHelpers[jsonValueClass]
	if !known {
		return
	}
	layout := generator.layouts[jsonValueClass]
	if layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	writer.line("// JSONValue built from one runtime canonical-text reading.")
	writer.open("func " + name + "(text string) " + generator.interfaceName(jsonValueClass) + " {")
	writer.line("return " + generator.callableName(constructor) + "(text)")
	writer.close("}")
	writer.blank()
}

// jsonOperation lowers the built-in members of JSONValue. Every member
// reaches this through the ordinary type-operation path, matching Word's
// wordOperation.
func (generator *generator) jsonOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	receiver := generator.jsonValueOf(value.Callee)
	errorClass := generator.descriptorName(jsonErrorClass)
	switch name {
	case "JSONValue.kind":
		return "AhdJSONKind(" + errorClass + ", " + receiver + ")"
	case "JSONValue.isNull":
		return "AhdJSONIsNull(" + errorClass + ", " + receiver + ")"
	case "JSONValue.bool":
		return "AhdJSONBool(" + errorClass + ", " + receiver + ")"
	case "JSONValue.int":
		return "AhdJSONInt(" + errorClass + ", " + receiver + ")"
	case "JSONValue.real":
		return "AhdJSONReal(" + errorClass + ", " + receiver + ")"
	case "JSONValue.string":
		return "AhdJSONString(" + errorClass + ", " + receiver + ")"
	case "JSONValue.array":
		return generator.jsonValueArrayResult(receiver, meta)
	case "JSONValue.object":
		return generator.jsonValueObjectResult(receiver, meta)
	case "JSONValue.get":
		return generator.jsonValueGetResult(receiver, value, meta)
	case "JSONValue.at":
		return generator.jsonValueFrom("AhdJSONAt("+errorClass+", "+receiver+", "+
			generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.IntType}, false)+")", meta)
	default:
		return generator.unsupported("JSONValue operation "+name, meta.Span)
	}
}

// jsonValueArrayResult wraps every element of AhdJSONArrayElements back into
// a JSONValue, producing the List<JSONValue> array() itself returns.
func (generator *generator) jsonValueArrayResult(receiver string, meta ir.ExprBase) string {
	helper, ok := generator.jsonValueHelper()
	if !ok {
		return generator.unsupported("a JSONValue without its Class declaration", meta.Span)
	}
	errorClass := generator.descriptorName(jsonErrorClass)
	element := generator.interfaceName(jsonValueClass)
	return "func(texts []string) *AhdList[" + element + "] { " +
		"items := make([]" + element + ", len(texts)); " +
		"for index, text := range texts { items[index] = " + helper + "(text) }; " +
		"return AhdNewList(items...) }(AhdJSONArrayElements(" + errorClass + ", " + receiver + "))"
}

// jsonValueObjectResult wraps AhdJSONObjectKeys/AhdJSONObjectValueTexts back
// into a Pair<String, JSONValue>, producing what object() itself returns.
func (generator *generator) jsonValueObjectResult(receiver string, meta ir.ExprBase) string {
	helper, ok := generator.jsonValueHelper()
	if !ok {
		return generator.unsupported("a JSONValue without its Class declaration", meta.Span)
	}
	errorClass := generator.descriptorName(jsonErrorClass)
	element := generator.interfaceName(jsonValueClass)
	return "func(keys, texts []string) *AhdPair[string, " + element + "] { " +
		"values := make([]" + element + ", len(texts)); " +
		"for index, text := range texts { values[index] = " + helper + "(text) }; " +
		"return AhdBuildPair(keys, values) }(AhdJSONObjectKeys(" + errorClass + ", " + receiver +
		"), AhdJSONObjectValueTexts(" + errorClass + ", " + receiver + "))"
}

// jsonValueGetResult wraps AhdJSONGet's *string result into a nullable
// JSONValue: nil stays a genuine AhdCode null, a present key becomes one
// wrapped instance.
func (generator *generator) jsonValueGetResult(receiver string, value *ir.CallExpr, meta ir.ExprBase) string {
	helper, ok := generator.jsonValueHelper()
	if !ok {
		return generator.unsupported("a JSONValue without its Class declaration", meta.Span)
	}
	errorClass := generator.descriptorName(jsonErrorClass)
	element := generator.interfaceName(jsonValueClass)
	key := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
	return "func(text *string) " + element + " { " +
		"if text == nil { return nil }; return " + helper + "(*text) }(AhdJSONGet(" +
		errorClass + ", " + receiver + ", " + key + "))"
}
