package semantic

import (
	"strings"
	"testing"
)

const jsonPreamble = "bring JSON\nfrom JSON bring JSONValue\nfrom JSON bring JSONError\n\n"

func TestJSONModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, jsonPreamble+`value: JSONValue := JSON.nullValue()
value = JSON.fromBool(true)
value = JSON.fromInt(91)
value = JSON.fromReal(3.14)
value = JSON.fromString("hello")
value = JSON.array([JSON.fromInt(1), JSON.fromInt(2)])
value = JSON.object({"a": JSON.fromInt(1)})
parsed: JSONValue := JSON.parse(r"{}")
read: JSONValue := JSON.read("data.json")
text: String := JSON.stringify(value)
text = JSON.stringify(value, true)
JSON.write(value, "out.json")
JSON.write(value, "out.json", true)

kind: String := value.kind()
isNull: Bool := value.isNull()
flag: Bool := value.bool()
whole: Int := value.int()
realValue: Real := value.real()
stringValue: String := value.string()
items: List<JSONValue> := value.array()
pair: Pair<String, JSONValue> := value.object()
found: JSONValue? := value.get("a")
element: JSONValue := value.at(0)
`)
	requireSemanticClean(t, result)
}

func TestJSONOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`value.kind(1)`,
		`value.isNull(1)`,
		`value.bool(1)`,
		`value.int(1)`,
		`value.real(1)`,
		`value.string(1)`,
		`value.array(1)`,
		`value.object(1)`,
		`value.get()`,
		`value.get(1)`,
		`value.get("a", "b")`,
		`value.at()`,
		`value.at("a")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, jsonPreamble+"value: JSONValue := JSON.nullValue()\n"+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestJSONFunctionsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`JSON.parse(1)`,
		`JSON.parse()`,
		`JSON.fromBool(1)`,
		`JSON.fromInt(1.0)`,
		`JSON.fromReal("x")`,
		`JSON.fromString(1)`,
		`JSON.array(1)`,
		`JSON.object(1)`,
		`JSON.stringify(1)`,
		`JSON.write(JSON.nullValue(), 1)`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, jsonPreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestJSONFromRealAcceptsIntWidening(t *testing.T) {
	// AhdCode already defines safe Int -> Real widening, so a JSON.fromReal
	// call may be given an Int literal, matching real()'s documented
	// widening behavior.
	result := analyzeWithStandardModules(t, jsonPreamble+`value: JSONValue := JSON.fromReal(1)
`)
	requireSemanticClean(t, result)
}

func TestJSONValueGetIsNullableWithoutGuard(t *testing.T) {
	result := analyzeWithStandardModules(t, jsonPreamble+`value: JSONValue := JSON.nullValue()
found: JSONValue := value.get("a")
`)
	requireSemanticFailure(t, result)
}

func TestJSONValueOperationsArePositionalOnly(t *testing.T) {
	result := analyzeWithStandardModules(t, jsonPreamble+`value: JSONValue := JSON.nullValue()
value.get(key: "a")
`)
	requireSemanticCode(t, result, codeCallArguments)
}

func TestJSONValueConstructionHintNamesFactories(t *testing.T) {
	result := analyzeWithStandardModules(t, jsonPreamble+`value: JSONValue := JSONValue()
`)
	requireSemanticCode(t, result, codeCallArguments)
	found := false
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(diagnostic.Hint, "JSON.parse(source)") && strings.Contains(diagnostic.Hint, "JSON.nullValue()") {
			found = true
		}
	}
	if !found {
		t.Fatalf("JSONValue construction diagnostic omitted the JSON factories: %+v", result.Diagnostics)
	}
}

func TestJSONHiddenStorageAndUnknownMembersAreRejected(t *testing.T) {
	for _, member := range []string{"text", "data", "describe"} {
		result := analyzeWithStandardModules(t, jsonPreamble+
			"value: JSONValue := JSON.nullValue()\nwrite(value."+member+")\n")
		requireSemanticFailure(t, result)
	}
}

func TestJSONErrorCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, jsonPreamble+`attempt {
    JSON.parse("not json")
} except JSONError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}
