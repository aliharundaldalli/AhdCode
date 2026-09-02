package semantic

import (
	"strings"
	"testing"

	"ahdcode/internal/types"
)

const sqlitePreamble = "bring SQLite\nfrom SQLite bring Database\nfrom SQLite bring SQLiteValue\nfrom SQLite bring SQLiteError\n\n"

func TestSQLiteModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, sqlitePreamble+`db: Database := SQLite.open("notes.db")
memory: Database := SQLite.open(":memory:")
value: SQLiteValue := SQLite.nullValue()
value = SQLite.fromInt(42)
value = SQLite.fromReal(91.5)
value = SQLite.fromString("Ayşe")

created: Int := db.execute("CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)")
inserted: Int := db.execute("INSERT INTO notes (title) VALUES (?)", [SQLite.fromString("first")])
db.execute("DELETE FROM notes WHERE id = ?", [])
rowId: Int := db.lastInsertId()

rows: List<Pair<String, SQLiteValue>> := db.query("SELECT id, title FROM notes ORDER BY id")
filtered: List<Pair<String, SQLiteValue>> := db.query("SELECT id FROM notes WHERE id > ?", [SQLite.fromInt(1)])
for row in rows {
    kind: Local String := row["id"].kind()
    missing: Local Bool := row["id"].isNull()
    identifier: Local Int := row["id"].int()
    score: Local Real := row["id"].real()
    title: Local String := row["title"].string()
}

db.begin()
db.commit()
db.begin()
db.rollback()
db.close()
`)
	requireSemanticClean(t, result)
}

func TestSQLiteOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`db.execute()`,
		`db.execute(1)`,
		`db.execute("SELECT 1", 1)`,
		`db.execute("SELECT 1", [1])`,
		`db.execute("SELECT 1", ["text"])`,
		`db.execute("SELECT 1", [], 1)`,
		`db.query()`,
		`db.query(1)`,
		`db.query("SELECT 1", "x")`,
		`db.lastInsertId(1)`,
		`db.begin(1)`,
		`db.commit(1)`,
		`db.rollback(1)`,
		`db.close(1)`,
		`value.kind(1)`,
		`value.isNull(1)`,
		`value.int(1)`,
		`value.real(1)`,
		`value.string(1)`,
		`whole: Int := value.string()`,
		`text: String := value.int()`,
		`rows: List<SQLiteValue> := db.query("SELECT 1")`,
		`changed: String := db.execute("SELECT 1")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, sqlitePreamble+
				"db: Database := SQLite.open(\":memory:\")\nvalue: SQLiteValue := SQLite.nullValue()\n"+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestSQLiteFunctionsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`SQLite.open()`,
		`SQLite.open(1)`,
		`SQLite.open("a", "b")`,
		`SQLite.nullValue(1)`,
		`SQLite.fromInt("1")`,
		`SQLite.fromInt(1.5)`,
		`SQLite.fromReal("x")`,
		`SQLite.fromString(1)`,
		`SQLite.fromBool(true)`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, sqlitePreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestSQLiteFromRealAcceptsIntWidening(t *testing.T) {
	result := analyzeWithStandardModules(t, sqlitePreamble+`value: SQLiteValue := SQLite.fromReal(1)
`)
	requireSemanticClean(t, result)
}

func TestSQLiteOperationsArePositionalOnly(t *testing.T) {
	result := analyzeWithStandardModules(t, sqlitePreamble+`db: Database := SQLite.open(":memory:")
db.execute(sql: "SELECT 1")
`)
	requireSemanticCode(t, result, codeCallArguments)
}

func TestSQLiteOperationsRejectNullableReceiversAndArguments(t *testing.T) {
	tests := []string{
		"db: Database? := null\ndb.close()",
		"db: Database := SQLite.open(\":memory:\")\nsql: String? := null\ndb.execute(sql)",
		"db: Database := SQLite.open(\":memory:\")\nvalues: List<SQLiteValue>? := null\ndb.execute(\"SELECT 1\", values)",
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, sqlitePreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestSQLiteValuesAreNotConstructedDirectly(t *testing.T) {
	for _, source := range []string{`db: Database := Database("x")`, `value: SQLiteValue := SQLiteValue("x")`} {
		result := analyzeWithStandardModules(t, sqlitePreamble+source+"\n")
		requireSemanticCode(t, result, codeCallArguments)
		found := false
		for _, diagnostic := range result.Diagnostics {
			if strings.Contains(diagnostic.Hint, "SQLite.open(path)") || strings.Contains(diagnostic.Hint, "SQLite.nullValue()") {
				found = true
			}
		}
		if !found {
			t.Fatalf("construction diagnostic omitted the SQLite factories: %+v", result.Diagnostics)
		}
	}
}

func TestSQLiteHiddenStorageAndUnknownMembersAreRejected(t *testing.T) {
	for _, member := range []string{"handle", "data", "conn", "rows"} {
		result := analyzeWithStandardModules(t, sqlitePreamble+
			"db: Database := SQLite.open(\":memory:\")\nwrite(db."+member+")\n")
		requireSemanticFailure(t, result)
		result = analyzeWithStandardModules(t, sqlitePreamble+
			"value: SQLiteValue := SQLite.nullValue()\nwrite(value."+member+")\n")
		requireSemanticFailure(t, result)
	}
}

func TestSQLiteErrorCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, sqlitePreamble+`db: Database := SQLite.open(":memory:")
db.begin()
attempt {
    db.execute("UPDATE accounts SET balance = balance - ? WHERE id = ?", [SQLite.fromReal(10.0), SQLite.fromInt(1)])
    db.commit()
} except SQLiteError as error {
    db.rollback()
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

func TestSQLiteModuleInterfaceExportsExactSurface(t *testing.T) {
	module := StandardModuleInterfaces()["SQLite"]
	if module == nil || module.ModuleID != "builtin:SQLite" {
		t.Fatalf("SQLite is not a registered builtin module: %#v", module)
	}
	wantExports := []string{"Database", "SQLiteError", "SQLiteValue", "fromInt", "fromReal", "fromString", "nullValue", "open"}
	if strings.Join(module.ExportNames, ",") != strings.Join(wantExports, ",") {
		t.Fatalf("SQLite exports %v; want %v", module.ExportNames, wantExports)
	}
	signatures := map[string]string{
		"open":       "(path: String) -> Database",
		"nullValue":  "() -> SQLiteValue",
		"fromInt":    "(value: Int) -> SQLiteValue",
		"fromReal":   "(value: Real) -> SQLiteValue",
		"fromString": "(value: String) -> SQLiteValue",
	}
	for name, want := range signatures {
		symbol := module.Exports[name]
		if symbol == nil || symbol.Callable == nil {
			t.Fatalf("SQLite.%s is not an exported function", name)
		}
		if have := FormatSignature(symbol.Callable.Signature); have != want {
			t.Fatalf("SQLite.%s signature %q; want %q", name, have, want)
		}
	}
	for _, class := range []string{"Database", "SQLiteError", "SQLiteValue"} {
		symbol := module.Exports[class]
		if symbol == nil || symbol.Kind != ClassSymbol || symbol.Class == nil || symbol.Class.ModuleID != "builtin:SQLite" {
			t.Fatalf("SQLite.%s is not an exported builtin Class: %#v", class, symbol)
		}
	}
	errorSymbol := module.Exports["SQLiteError"]
	if errorSymbol.Class.Parent == nil || errorSymbol.Class.Parent.Name != "Error" {
		t.Fatalf("SQLiteError does not derive from Error: %#v", errorSymbol.Class)
	}
}

func TestSQLiteOperationShapesMatchFrozenAPI(t *testing.T) {
	shapes := sqliteOperationShapes()
	row := types.Pair{Key: types.String, Value: sqliteValueType()}
	parameters := types.List{Element: sqliteValueType()}
	expected := map[TypeOperation]struct {
		parameters []types.Type
		optional   int
		result     types.Type
	}{
		SQLiteDatabaseExecute:      {[]types.Type{types.String, parameters}, 1, types.Int},
		SQLiteDatabaseQuery:        {[]types.Type{types.String, parameters}, 1, types.List{Element: row}},
		SQLiteDatabaseLastInsertID: {nil, 0, types.Int},
		SQLiteDatabaseBegin:        {nil, 0, types.Nothing},
		SQLiteDatabaseCommit:       {nil, 0, types.Nothing},
		SQLiteDatabaseRollback:     {nil, 0, types.Nothing},
		SQLiteDatabaseClose:        {nil, 0, types.Nothing},
		SQLiteValueKind:            {nil, 0, types.String},
		SQLiteValueIsNull:          {nil, 0, types.Bool},
		SQLiteValueInt:             {nil, 0, types.Int},
		SQLiteValueReal:            {nil, 0, types.Real},
		SQLiteValueString:          {nil, 0, types.String},
	}
	if len(shapes) != len(expected) {
		t.Fatalf("SQLite publishes %d operations; want %d", len(shapes), len(expected))
	}
	for operation, want := range expected {
		shape, known := shapes[operation]
		if !known {
			t.Fatalf("operation %s is missing", operation)
		}
		if len(shape.parameters) != len(want.parameters) || shape.optional != want.optional || !types.Equal(shape.result, want.result) {
			t.Fatalf("operation %s has shape %+v; want %+v", operation, shape, want)
		}
		for index, parameter := range want.parameters {
			if !types.Equal(shape.parameters[index], parameter) {
				t.Fatalf("operation %s parameter %d is %s; want %s", operation, index, types.Display(shape.parameters[index]), types.Display(parameter))
			}
		}
	}
	if len(SQLiteDatabaseOperations) != 7 || len(SQLiteValueOperations) != 5 {
		t.Fatalf("published operation names drifted: %v %v", SQLiteDatabaseOperations, SQLiteValueOperations)
	}
}
