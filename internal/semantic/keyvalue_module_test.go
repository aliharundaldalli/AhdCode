package semantic

import "testing"

const keyValuePreamble = "bring KeyValue\nfrom KeyValue bring KeyValueError\n\n"

const keyValueClassPreamble = keyValuePreamble + `Student: Class<> := {
    structure: Attributes := (
        name: String
    )
}

`

// TestKeyValueOperationsPreserveExactGenericTypes writes every result with its
// full generic shape, so an erased or widened key/value type would fail to
// compile.
func TestKeyValueOperationsPreserveExactGenericTypes(t *testing.T) {
	result := analyzeWithStandardModules(t, keyValueClassPreamble+`byName: Pair<String, Int> := {"a": 1, "b": 2}
byNumber: Pair<Int, String> := {1: "a"}
byFlag: Pair<Bool, Int> := {true: 1}
byStudent: Pair<String, Student> := {"ali": Student(name: "Ali")}

names: List<String> := KeyValue.keys(byName)
scores: List<Int> := KeyValue.values(byName)
numbers: List<Int> := KeyValue.keys(byNumber)
texts: List<String> := KeyValue.values(byNumber)
flags: List<Bool> := KeyValue.keys(byFlag)
students: List<Student> := KeyValue.values(byStudent)

built: Pair<String, Int> := KeyValue.combine(["a"], [1])
builtByInt: Pair<Int, String> := KeyValue.combine([1], ["a"])
builtByBool: Pair<Bool, Student> := KeyValue.combine([true], [Student(name: "Ali")])

extended: Pair<String, Int> := KeyValue.with(byName, "c", 3)
reduced: Pair<String, Int> := KeyValue.without(byName, "a")
picked: Pair<String, Int> := KeyValue.select(byName, ["b", "a"])
dropped: Pair<String, Int> := KeyValue.drop(byName, ["a"])
renamed: Pair<String, Int> := KeyValue.rename(byName, "a", "z")
merged: Pair<String, Int> := KeyValue.merge(byName, {"c": 3})
overlaid: Pair<String, Int> := KeyValue.overlay(byName, {"b": 9})

textValues: Pair<String, String> := KeyValue.mapValues(byName, lambda (value: Int) -> str(value))
intValues: Pair<Int, Int> := KeyValue.mapValues(byNumber, lambda (value: String) -> len(value))
listValues: Pair<String, List<Int>> := KeyValue.mapValues(byName, lambda (value: Int) -> [value])
studentValues: Pair<String, Student> := KeyValue.mapValues(byName, lambda (value: Int) -> Student(name: str(value)))
`)
	requireSemanticClean(t, result)
}

// TestKeyValuePreservesValueNullability proves a Pair's structural value
// nullability travels through every projection and transformation.
func TestKeyValuePreservesValueNullability(t *testing.T) {
	result := analyzeWithStandardModules(t, keyValuePreamble+`maybe: Function := (value: Int) -> String? {
    return null
}

scores: Pair<String, Int?> := {"a": 1, "b": null}
values: List<Int?> := KeyValue.values(scores)
keys: List<String> := KeyValue.keys(scores)
built: Pair<String, Int?> := KeyValue.combine(["a"], values)
extended: Pair<String, Int?> := KeyValue.with(scores, "c", null)
picked: Pair<String, Int?> := KeyValue.select(scores, ["b"])
merged: Pair<String, Int?> := KeyValue.merge(scores, {"c": null})

plain: Pair<String, Int> := {"a": 1}
mapped: Pair<String, String?> := KeyValue.mapValues(plain, maybe)
`)
	requireSemanticClean(t, result)
}

// TestKeyValueRejectsGenericWidening pins Pair invariance across every
// operation that takes or returns a second collection.
func TestKeyValueRejectsGenericWidening(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"with does not widen the value type",
			"record: Pair<String, Int> := {\"a\": 1}\nbad: Pair<String, Real> := KeyValue.with(record, \"b\", 2)"},
		{"merge rejects a differently typed operand",
			"left: Pair<String, Int> := {\"a\": 1}\nright: Pair<String, Real> := {\"b\": 2.5}\nwrite(KeyValue.merge(left, right))"},
		{"overlay rejects a differently typed operand",
			"base: Pair<String, Int> := {\"a\": 1}\nchanges: Pair<String, Real> := {\"b\": 2.5}\nwrite(KeyValue.overlay(base, changes))"},
		{"merge rejects a differently nullable operand",
			"left: Pair<String, Int> := {\"a\": 1}\nright: Pair<String, Int?> := {\"b\": null}\nwrite(KeyValue.merge(left, right))"},
		{"select rejects a key List of another type",
			"record: Pair<String, Int> := {\"a\": 1}\nwrite(KeyValue.select(record, [1]))"},
		{"drop rejects a nullable key List",
			"record: Pair<String, Int> := {\"a\": 1}\nkeys: List<String?> := [null]\nwrite(KeyValue.drop(record, keys))"},
		{"keys does not widen its element",
			"record: Pair<Int, String> := {1: \"a\"}\nbad: List<Real> := KeyValue.keys(record)"},
		{"values does not erase value nullability",
			"record: Pair<String, Int?> := {\"a\": null}\nbad: List<Int> := KeyValue.values(record)"},
		{"combine does not erase value nullability",
			"values: List<Int?> := [null]\nbad: Pair<String, Int> := KeyValue.combine([\"a\"], values)"},
		{"mapValues does not erase callback nullability",
			"maybe: Function := (value: Int) -> String? {\n    return null\n}\nrecord: Pair<String, Int> := {\"a\": 1}\nbad: Pair<String, String> := KeyValue.mapValues(record, maybe)"},
		{"with does not accept null for a non-null value slot",
			"record: Pair<String, Int> := {\"a\": 1}\nwrite(KeyValue.with(record, \"b\", null))"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, keyValuePreamble+test.text+"\n"))
		})
	}
}

// TestKeyValueCombineRestrictsPairKeys proves combine enforces the existing
// Pair key rules on its key List rather than converting keys to fit.
func TestKeyValueCombineRestrictsPairKeys(t *testing.T) {
	tests := []string{
		"keys: List<Real> := [1.5]\nwrite(KeyValue.combine(keys, [1]))",
		"keys: List<String?> := [null]\nwrite(KeyValue.combine(keys, [1]))",
		"keys: List<List<Int>> := [[1]]\nwrite(KeyValue.combine(keys, [1]))",
	}
	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			requireSemanticCode(t, analyzeWithStandardModules(t, keyValuePreamble+text+"\n"), codeInvalidPairKey)
		})
	}
}

// TestKeyValueMapValuesCallbackContract pins the transform Function's typing.
func TestKeyValueMapValuesCallbackContract(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"matching parameter accepted",
			"record: Pair<String, String> := {\"a\": \"1\"}\nresult: Pair<String, Int> := KeyValue.mapValues(record, lambda (value: String) -> int(value))", true},
		{"mismatched parameter rejected",
			"record: Pair<String, String> := {\"a\": \"1\"}\nwrite(KeyValue.mapValues(record, lambda (value: Int) -> value))", false},
		{"Nothing result rejected",
			"noop: Function := (value: Int) -> Nothing {\n    return\n}\nrecord: Pair<String, Int> := {\"a\": 1}\nwrite(KeyValue.mapValues(record, noop))", false},
		{"nullable value with matching nullable parameter accepted",
			"record: Pair<String, Int?> := {\"a\": null}\nresult: Pair<String, String> := KeyValue.mapValues(record, lambda (value: Int?) -> str(value))", true},
		{"nullable value with a non-null parameter rejected",
			"record: Pair<String, Int?> := {\"a\": null}\nwrite(KeyValue.mapValues(record, lambda (value: Int) -> value))", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, keyValuePreamble+test.text+"\n")
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			requireSemanticFailure(t, result)
		})
	}
}

func TestKeyValueRejectsWrongArityAndShapes(t *testing.T) {
	tests := []string{
		"write(KeyValue.keys())",
		"write(KeyValue.keys({\"a\": 1}, 2))",
		"write(KeyValue.keys([1, 2]))",
		"write(KeyValue.values(\"not a Pair\"))",
		"write(KeyValue.combine([\"a\"]))",
		"write(KeyValue.combine([\"a\"], [1], [2]))",
		"write(KeyValue.with({\"a\": 1}, \"b\"))",
		"write(KeyValue.with({\"a\": 1}, 2, 3))",
		"write(KeyValue.without({\"a\": 1}))",
		"write(KeyValue.select({\"a\": 1}, \"a\"))",
		"write(KeyValue.rename({\"a\": 1}, \"a\"))",
		"write(KeyValue.mapValues({\"a\": 1}))",
		"write(KeyValue.merge({\"a\": 1}))",
		"write(KeyValue.overlay({\"a\": 1}))",
		"write(KeyValue.with(pair: {\"a\": 1}, key: \"b\", value: 2))",
		"maybe: Pair<String, Int>? := null\nwrite(KeyValue.keys(maybe))",
	}
	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, keyValuePreamble+text+"\n"))
		})
	}
}

func TestKeyValueOperationHasNoFunctionValue(t *testing.T) {
	t.Run("namespace member", func(t *testing.T) {
		result := analyzeWithStandardModules(t, keyValuePreamble+"stored := KeyValue.keys\n")
		requireSemanticCode(t, result, codeInvalidType)
	})
	t.Run("direct import", func(t *testing.T) {
		result := analyzeWithStandardModules(t, "bring KeyValue\nfrom KeyValue bring keys\n\nstored := keys\n")
		requireSemanticCode(t, result, codeInvalidType)
	})
}

func TestKeyValueDirectImportCall(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring KeyValue
from KeyValue bring combine
from KeyValue bring overlay

record: Pair<String, Int> := combine(["a"], [1])
updated: Pair<String, Int> := overlay(record, {"a": 2})
`)
	requireSemanticClean(t, result)
}

func TestKeyValueErrorIsCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, keyValuePreamble+`attempt {
    write(KeyValue.combine(["a", "b"], [1]))
} except KeyValueError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

// TestKeyValueMissingKeyUsesKeyError documents that a genuinely missing Pair
// key keeps the language's existing KeyError rather than gaining a
// module-specific class.
func TestKeyValueMissingKeyUsesKeyError(t *testing.T) {
	result := analyzeWithStandardModules(t, keyValuePreamble+`attempt {
    write(KeyValue.without({"a": 1}, "z"))
} except KeyError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}
