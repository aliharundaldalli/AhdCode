package lowering

import (
	"testing"

	"ahdcode/internal/ir"
)

func TestSQLiteModuleLowersItsClassesAndOperations(t *testing.T) {
	result := lowerSources(t, map[string]string{
		"/Main.ahd": "bring SQLite\nfrom SQLite bring Database\nfrom SQLite bring SQLiteValue\n" +
			"db: Database := SQLite.open(\":memory:\")\n" +
			"db.execute(\"CREATE TABLE t (x INTEGER)\")\n" +
			"db.execute(\"INSERT INTO t VALUES (?)\", [SQLite.fromInt(1)])\n" +
			"rows: List<Pair<String, SQLiteValue>> := db.query(\"SELECT x FROM t\")\n" +
			"write(rows[0][\"x\"].int())\ndb.close()\n",
	}, "/Main.ahd")
	var sqlite *ir.Module
	for _, module := range result.Compilation.Modules {
		if module != nil && module.ID == ir.ModuleID(SQLiteModuleID) {
			sqlite = module
		}
	}
	if sqlite == nil {
		t.Fatal("the SQLite builtin module was not lowered")
	}
	classes := make(map[ir.ClassID]*ir.Class)
	for _, class := range sqlite.Classes {
		classes[class.ID] = class
	}
	database := classes[sqliteDatabaseClassID]
	if database == nil || len(database.Fields) != 1 || database.Fields[0].ID != SQLiteDatabaseHandleFieldID || !database.Fields[0].Hidden {
		t.Fatalf("Database lowered without its single hidden handle field: %#v", database)
	}
	if len(database.Operations) != 7 {
		t.Fatalf("Database publishes %v", database.Operations)
	}
	value := classes[sqliteValueClassID]
	if value == nil || len(value.Fields) != 1 || value.Fields[0].ID != SQLiteValueDataFieldID || !value.Fields[0].Hidden {
		t.Fatalf("SQLiteValue lowered without its single hidden data field: %#v", value)
	}
	errorClass := classes[sqliteErrorClassID]
	if errorClass == nil || !errorClass.Builtin || errorClass.Parent != ir.ClassID("builtin:core::class::Error") {
		t.Fatalf("SQLiteError lowered incorrectly: %#v", errorClass)
	}
	functions := make(map[ir.CallableID]bool)
	for _, function := range sqlite.Functions {
		functions[function.ID] = true
	}
	for _, class := range []*ir.Class{database, value, errorClass} {
		if !functions[class.Constructor] {
			t.Fatalf("%s has no lowered constructor %s", class.Name, class.Constructor)
		}
	}
}
