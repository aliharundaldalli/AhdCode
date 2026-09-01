package module

import "testing"

// TestBuiltinListsCannotBeShadowedByASiblingFile proves a user Lists.ahd in
// the entry's own directory never displaces the standard module.
func TestBuiltinListsCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd":  "bring Lists\nvalue := Lists.chunk([1, 2, 3], 2)",
		"/Lists.ahd": "chunk: Function := (values: List<Int>, size: Int) -> String {\n    return \"user\"\n}",
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/Lists.ahd").ID) != 0 {
		t.Fatal("the sibling Lists.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "Lists")
	if !module.Source.Builtin || module.ID != "builtin:Lists" {
		t.Fatalf("Lists did not keep its built-in identity: %#v", module)
	}
}

func TestBuiltinKeyValueCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd":     "bring KeyValue\nvalue := KeyValue.keys({\"a\": 1})",
		"/KeyValue.ahd": "keys: Function := (pair: Pair<String, Int>) -> String {\n    return \"user\"\n}",
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/KeyValue.ahd").ID) != 0 {
		t.Fatal("the sibling KeyValue.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "KeyValue")
	if !module.Source.Builtin || module.ID != "builtin:KeyValue" {
		t.Fatalf("KeyValue did not keep its built-in identity: %#v", module)
	}
}

// TestCollectionModulesAreExplicit proves neither module is implicitly in
// scope: both must be brought like every other standard module.
func TestCollectionModulesAreExplicit(t *testing.T) {
	for _, text := range []string{
		"value := Lists.chunk([1, 2], 2)",
		"value := KeyValue.keys({\"a\": 1})",
	} {
		t.Run(text, func(t *testing.T) {
			_, result := compileMemory(t, map[string]string{"/Main.ahd": text}, "/Main.ahd")
			requireCode(t, result, "SEM001")
		})
	}
}

// TestCollectionModulesCompose proves the two modules interoperate through
// ordinary typed values, with no coupling layer between them.
func TestCollectionModulesCompose(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `bring Lists
bring KeyValue

record: Pair<String, String> := KeyValue.combine(["name", "score"], ["Ali", "91"])
rows: List<List<String>> := [KeyValue.keys(record), KeyValue.values(record)]
turned: List<List<String>> := Lists.transpose(rows)
counts: Pair<String, Int> := Lists.valueCounts(KeyValue.keys(record))
`,
	}, "/Main.ahd")
	requireClean(t, result)
}
