package semantic

import (
	"strings"
	"testing"
)

const postgresqlPreamble = `bring PostgreSQL
from PostgreSQL bring (PostgreSQLDatabase, PostgreSQLTransaction, PostgreSQLResult, PostgreSQLValue, PostgreSQLError)

db: PostgreSQLDatabase := PostgreSQL.connect("127.0.0.1", "app", "secret")

`

func TestPostgreSQLModuleRegistered(t *testing.T) {
	module, ok := StandardModuleInterfaces()["PostgreSQL"]
	if !ok || module.ModuleID != "builtin:PostgreSQL" {
		t.Fatalf("PostgreSQL is not registered: %#v", module)
	}
	want := []string{"PostgreSQLDatabase", "PostgreSQLError", "PostgreSQLResult", "PostgreSQLTransaction", "PostgreSQLValue",
		"connect", "fromBool", "fromInt", "fromReal", "fromString", "nullValue"}
	if strings.Join(module.ExportNames, ",") != strings.Join(want, ",") {
		t.Fatalf("PostgreSQL exports %v, want %v", module.ExportNames, want)
	}
	// The frozen v1.4.0 surface: no binary parameters, arrays, LISTEN/NOTIFY, or COPY.
	for _, name := range []string{"fromBinary", "fromArray", "listen", "copy", "fromJSON", "open"} {
		if module.Exports[name] != nil {
			t.Fatalf("PostgreSQL must not export %q", name)
		}
	}
}

func TestPostgreSQLModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, postgresqlPreamble+`admin: PostgreSQLDatabase := PostgreSQL.connect("127.0.0.1", "app", "secret", 5432, null, "tls", 10)
named: PostgreSQLDatabase := PostgreSQL.connect("db.internal", "app", "secret", 6543, "attendance", "none", 30)
db.ping()
outcome: PostgreSQLResult := db.execute("INSERT INTO notes (title, done) VALUES ($1, $2)", [PostgreSQL.fromString("x"), PostgreSQL.fromBool(false)])
changed: Int := outcome.affectedRows()
rows: List<Pair<String, PostgreSQLValue>> := db.query("SELECT id FROM notes WHERE score > $1 AND n = $2 AND m IS $3", [PostgreSQL.fromReal(1.5), PostgreSQL.fromInt(2), PostgreSQL.nullValue()])
plain := db.query("SELECT 1 AS one")
cell: PostgreSQLValue := rows[0]["id"]
kind: String := cell.kind()
flag: Bool := cell.isNull() or cell.bool()
number: Int := cell.int()
measure: Real := cell.real()
label: String := cell.string()
blob: Bool := cell.isBinary()
size: Int := cell.binarySize()
encoded: String := cell.binaryBase64()
transaction: PostgreSQLTransaction := db.begin()
transaction.execute("UPDATE notes SET done = $1", [PostgreSQL.fromBool(true)])
more := transaction.query("SELECT id FROM notes")
transaction.commit()
transaction.rollback()
db.close()
attempt {
    db.ping()
} except PostgreSQLError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

func TestPostgreSQLRejectsStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`PostgreSQL.connect()`,
		`PostgreSQL.connect("h")`,
		`PostgreSQL.connect(1, "u", "p")`,
		`PostgreSQL.connect("h", "u", "p", "5432")`,
		`PostgreSQL.fromBool("true")`,
		`PostgreSQL.fromInt(1.5)`,
		`db.execute()`,
		`db.execute(1)`,
		`db.execute("x", [1])`,
		`db.execute("x").lastInsertId()`,
		`db.query("x")[0]["id"].json()`,
		`count: String := db.execute("x").affectedRows()`,
		`db.begin(1)`,
		`db.commit()`,
		"maybe: String? := null\nPostgreSQL.fromString(maybe)",
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, postgresqlPreamble+source+"\n"))
	}
}

func TestPostgreSQLValuesAreNotConstructedDirectly(t *testing.T) {
	for source, hint := range map[string]string{
		`PostgreSQLDatabase()`:    "connect a PostgreSQLDatabase",
		`PostgreSQLValue("Sx")`:   "create a PostgreSQLValue",
		`PostgreSQLTransaction()`: "open a PostgreSQLTransaction",
		`PostgreSQLResult("1")`:   "a PostgreSQLResult is produced",
	} {
		result := analyzeWithStandardModules(t, postgresqlPreamble+source+"\n")
		requireSemanticFailure(t, result)
		if !strings.Contains(semanticHintsOf(result), hint) {
			t.Fatalf("%s has no construction hint: %q", source, semanticHintsOf(result))
		}
	}
}
