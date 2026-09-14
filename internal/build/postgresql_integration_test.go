package build

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

// postgresqlAcceptanceSource exercises the whole PostgreSQL module against a
// real server. {{CONNECT}} and {{SCHEMA}} are filled in per run.
const postgresqlAcceptanceSource = `bring PostgreSQL
from PostgreSQL bring (PostgreSQLDatabase, PostgreSQLTransaction, PostgreSQLResult, PostgreSQLError)

db: PostgreSQLDatabase := PostgreSQL.connect({{CONNECT}})
db.ping()
db.execute("CREATE TABLE {{SCHEMA}}.people (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name text NOT NULL UNIQUE, score numeric(5, 2), active boolean NOT NULL, joined timestamptz, photo bytea)")
created: PostgreSQLResult := db.execute("INSERT INTO {{SCHEMA}}.people (name, score, active, joined) VALUES ($1, $2::numeric, $3, $4::timestamptz), ($5, $6::numeric, $7, $8::timestamptz)", [PostgreSQL.fromString("Ayşe"), PostgreSQL.fromString("91.50"), PostgreSQL.fromBool(true), PostgreSQL.fromString("2026-09-14 10:30:00+03"), PostgreSQL.fromString("Mehmet"), PostgreSQL.nullValue(), PostgreSQL.fromBool(false), PostgreSQL.nullValue()])
write("inserted " + str(created.affectedRows()))
returned := db.query("INSERT INTO {{SCHEMA}}.people (name, active) VALUES ($1, $2) RETURNING id", [PostgreSQL.fromString("Zeynep"), PostgreSQL.fromBool(true)])
write("returned id " + str(returned[0]["id"].int()))
rows := db.query("SELECT id, name, score, active, joined, photo FROM {{SCHEMA}}.people ORDER BY id")
for row in rows {
    write(str(row["id"].int()) + " " + row["name"].string() + " " + row["score"].kind() + " " + str(row["active"].bool()) + " " + row["joined"].kind())
}
write(rows[0]["score"].string() + " " + rows[0]["joined"].string() + " " + str(rows[0]["photo"].isNull()))

transaction: PostgreSQLTransaction := db.begin()
transaction.execute("UPDATE {{SCHEMA}}.people SET active = $1 WHERE name = $2", [PostgreSQL.fromBool(false), PostgreSQL.fromString("Ayşe")])
inside := transaction.query("SELECT count(*) AS n FROM {{SCHEMA}}.people WHERE active")
outside := db.query("SELECT count(*) AS n FROM {{SCHEMA}}.people WHERE active")
write("inside " + str(inside[0]["n"].int()) + " outside " + str(outside[0]["n"].int()))
transaction.commit()
after := db.query("SELECT count(*) AS n FROM {{SCHEMA}}.people WHERE active")
write("after commit " + str(after[0]["n"].int()))

duplicate: PostgreSQLTransaction := db.begin()
attempt {
    duplicate.execute("INSERT INTO {{SCHEMA}}.people (name, active) VALUES ($1, $2)", [PostgreSQL.fromString("Zeynep"), PostgreSQL.fromBool(true)])
    duplicate.commit()
} except PostgreSQLError as error {
    write(error.message)
    duplicate.rollback()
}
aborted: PostgreSQLTransaction := db.begin()
attempt {
    aborted.query("SELECT 1 / 0 AS broken")
} except PostgreSQLError as error {
    write(error.message)
}
attempt {
    aborted.commit()
} except PostgreSQLError as error {
    write(error.message)
    aborted.rollback()
    write("rollback after a failed commit is quiet")
}

attempt {
    db.query("SELECT 1 AS one; SELECT 2 AS two")
} except PostgreSQLError as error {
    write(error.message)
}
attempt {
    db.query("SELECT id AS label, name AS label FROM {{SCHEMA}}.people")
} except PostgreSQLError as error {
    write(error.message)
}
attempt {
    db.query("SELECT ARRAY[1, 2] AS numbers")
} except PostgreSQLError as error {
    write(error.message)
}
attempt {
    db.query("SELECT $1::integer AS n")
} except PostgreSQLError as error {
    write(error.message)
}
attempt {
    write(str(rows[0]["name"].int()))
} except PostgreSQLError as error {
    write(error.message)
}
db.close()
db.close()
attempt {
    db.ping()
} except PostgreSQLError as error {
    write(error.message)
}
`

const postgresqlAcceptanceOutput = "inserted 2\n" +
	"returned id 3\n" +
	"1 Ayşe String true String\n" +
	"2 Mehmet Null false Null\n" +
	"3 Zeynep Null true Null\n" +
	"91.50 2026-09-14 07:30:00+00 true\n" +
	"inside 1 outside 2\n" +
	"after commit 1\n" +
	"PostgreSQL execution failed: (23505) duplicate key value violates unique constraint \"people_name_key\"\n" +
	"PostgreSQL query failed: (22012) division by zero\n" +
	"PostgreSQL transaction was rolled back because an earlier statement failed\n" +
	"rollback after a failed commit is quiet\n" +
	"PostgreSQL query failed: (42601) cannot insert multiple commands into a prepared statement\n" +
	"query result has duplicate column \"label\"; alias it with AS\n" +
	"PostgreSQL query failed: column \"numbers\" holds a PostgreSQL array, which AhdCode v1.4.0 does not read; convert it in SQL, for example array_to_json(numbers)::text\n" +
	"PostgreSQL query failed: the statement has 1 placeholder(s) but 0 parameter(s) were passed\n" +
	"int() requires kind Int; this PostgreSQLValue has kind String (check kind() first)\n" +
	"this PostgreSQLDatabase is closed\n"

type postgresqlAcceptanceServer struct {
	host, username, password, database, security string
	port                                         int64
}

// postgresqlAcceptanceTarget reads AHDCODE_TEST_POSTGRESQL_*; without a host
// the test skips.
func postgresqlAcceptanceTarget(t *testing.T) postgresqlAcceptanceServer {
	t.Helper()
	server := postgresqlAcceptanceServer{
		host:     os.Getenv("AHDCODE_TEST_POSTGRESQL_HOST"),
		username: os.Getenv("AHDCODE_TEST_POSTGRESQL_USERNAME"),
		password: os.Getenv("AHDCODE_TEST_POSTGRESQL_PASSWORD"),
		database: os.Getenv("AHDCODE_TEST_POSTGRESQL_DATABASE"),
		security: "tls",
		port:     5432,
	}
	if server.host == "" {
		t.Skip("set AHDCODE_TEST_POSTGRESQL_HOST (with _PORT, _USERNAME, _PASSWORD, _DATABASE, _SECURITY) to run the PostgreSQL integration suite")
	}
	if text := os.Getenv("AHDCODE_TEST_POSTGRESQL_PORT"); text != "" {
		port, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			t.Fatalf("AHDCODE_TEST_POSTGRESQL_PORT=%q is not a port number", text)
		}
		server.port = port
	}
	if security := os.Getenv("AHDCODE_TEST_POSTGRESQL_SECURITY"); security != "" {
		server.security = security
	}
	return server
}

func postgresqlAhdLiteral(t *testing.T, value string) string {
	t.Helper()
	if strings.ContainsAny(value, "\"\r\n") {
		t.Skip("AHDCODE_TEST_POSTGRESQL_* values with quotes or line breaks cannot be embedded in the acceptance program")
	}
	return `r"` + value + `"`
}

// schema creates a throwaway schema through the runtime directly and drops it
// when the test ends.
func (server postgresqlAcceptanceServer) schema(t *testing.T) string {
	t.Helper()
	var database *string
	if server.database != "" {
		database = &server.database
	}
	admin, err := ahdruntime.PostgreSQLConnect(server.host, server.username, server.password, server.port, database, server.security, 10)
	if err != nil {
		t.Fatalf("connecting to the integration server: %v", err)
	}
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	schema := "ahd_test_" + hex.EncodeToString(random)
	if _, err := ahdruntime.PostgreSQLExecute(admin, "CREATE SCHEMA "+schema, nil); err != nil {
		_ = ahdruntime.PostgreSQLClose(admin)
		t.Fatalf("creating %s: %v", schema, err)
	}
	t.Cleanup(func() {
		if _, err := ahdruntime.PostgreSQLExecute(admin, "DROP SCHEMA "+schema+" CASCADE", nil); err != nil {
			t.Errorf("dropping %s: %v", schema, err)
		}
		_ = ahdruntime.PostgreSQLClose(admin)
	})
	return schema
}

func (server postgresqlAcceptanceServer) source(t *testing.T, schema string) string {
	t.Helper()
	database := "null"
	if server.database != "" {
		database = postgresqlAhdLiteral(t, server.database)
	}
	connect := strings.Join([]string{
		postgresqlAhdLiteral(t, server.host), postgresqlAhdLiteral(t, server.username), postgresqlAhdLiteral(t, server.password),
		strconv.FormatInt(server.port, 10), database, postgresqlAhdLiteral(t, server.security), "10",
	}, ", ")
	return strings.NewReplacer("{{CONNECT}}", connect, "{{SCHEMA}}", schema).Replace(postgresqlAcceptanceSource)
}

// Without a server the acceptance program must still compile, so a mistake in
// it cannot hide behind the integration test's skip.
func TestPostgreSQLAcceptanceProgramCompiles(t *testing.T) {
	source := strings.NewReplacer(
		"{{CONNECT}}", `r"127.0.0.1", r"app", r"pw", 5432, null, r"none", 10`,
		"{{SCHEMA}}", "ahd_test_compile",
	).Replace(postgresqlAcceptanceSource)
	directory := writeSources(t, map[string]string{"main.ahd": source})
	compiled := Compile(filepath.Join(directory, "main.ahd"))
	if compiled.HasErrors() {
		t.Fatalf("acceptance program does not compile:\n%s", diagnosticText(compiled.Diagnostics))
	}
	if !compiled.Program.RequiresPostgreSQL {
		t.Fatal("acceptance program does not require the PostgreSQL tree")
	}
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": source})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	if lowered := lowering.LowerCompilation(frontend); lowered.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", lowered.Diagnostics)
	}
}

// The same program, run as a native executable and in the evaluator, each in
// its own schema, prints the same lines.
func TestPostgreSQLIntegrationNativeAndEvaluatorAgree(t *testing.T) {
	server := postgresqlAcceptanceTarget(t)

	directory := writeSources(t, map[string]string{"main.ahd": server.source(t, server.schema(t))})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("native program failed: code=%d stderr=%q\nstdout:\n%s", code, stderr, stdout)
	}
	if stdout != postgresqlAcceptanceOutput {
		t.Fatalf("native output mismatch:\n got: %q\nwant: %q", stdout, postgresqlAcceptanceOutput)
	}

	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": server.source(t, server.schema(t))})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	lowered := lowering.LowerCompilation(frontend)
	if lowered.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", lowered.Diagnostics)
	}
	var output bytes.Buffer
	session := evaluator.New(bufio.NewReader(strings.NewReader("")), &output, directory)
	if result := session.Execute(lowered.Compilation, 0); result.Failure != nil {
		t.Fatalf("evaluator raised %s: %s\noutput so far:\n%s", result.Failure.Name, result.Failure.Message, output.String())
	}
	if output.String() != stdout {
		t.Fatalf("evaluator output differs from native:\n evaluator: %q\n    native: %q", output.String(), stdout)
	}
}
