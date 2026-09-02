package module

import "testing"

func TestBuiltinSQLiteCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd":   "bring SQLite\ndb := SQLite.open(\":memory:\")",
		"/SQLite.ahd": `open: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/SQLite.ahd").ID) != 0 {
		t.Fatal("the sibling SQLite.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "SQLite")
	if !module.Source.Builtin || module.ID != "builtin:SQLite" {
		t.Fatalf("SQLite did not keep its built-in identity: %#v", module)
	}
}

func TestSQLiteIsExplicit(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `db := SQLite.open(":memory:")`,
	}, "/Main.ahd")
	requireCode(t, result, "SEM001")
}

func TestSQLiteTypesRequireExplicitFromBring(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring SQLite\ndb: Database := SQLite.open(\":memory:\")",
	}, "/Main.ahd")
	if !result.HasErrors() {
		t.Fatal("Database was usable as a type without `from SQLite bring Database`")
	}
	_, result = compileMemory(t, map[string]string{
		"/Main.ahd": "bring SQLite\nfrom SQLite bring Database\nfrom SQLite bring SQLiteValue\nfrom SQLite bring SQLiteError\n" +
			"db: Database := SQLite.open(\":memory:\")\nrows: List<Pair<String, SQLiteValue>> := db.query(\"SELECT 1 AS one\")\n" +
			"attempt {\n    db.close()\n} except SQLiteError as error {\n    write(error.message)\n}\n",
	}, "/Main.ahd")
	requireClean(t, result)
}
