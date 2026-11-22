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

const mysqlAcceptanceSource = `bring MySQL
from MySQL bring (MySQLDatabase, MySQLTransaction, MySQLResult, MySQLError, MySQLValue)

db: MySQLDatabase := MySQL.connect({{CONNECT}})
db.ping()
db.execute("CREATE TABLE {{TABLE}} (id BIGINT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(120) NOT NULL, score DOUBLE)")
r1: MySQLResult := db.execute("INSERT INTO {{TABLE}} (name, score) VALUES (?, ?)", [MySQL.fromString("Ayşe"), MySQL.fromReal(91.5)])
write("affected=" + str(r1.affectedRows()))
id1: Int? := r1.lastInsertId()
if id1 != null {
    write("has-id=true")
}

tx: MySQLTransaction := db.begin()
tx.execute("INSERT INTO {{TABLE}} (name, score) VALUES (?, ?)", [MySQL.fromString("RollbackMe"), MySQL.fromReal(1.0)])
tx.rollback()

tx2: MySQLTransaction := db.begin()
tx2.execute("UPDATE {{TABLE}} SET score = ? WHERE name = ?", [MySQL.fromReal(95.0), MySQL.fromString("Ayşe")])
tx2.commit()

rows: List<Pair<String, MySQLValue>> := db.query("SELECT name, score FROM {{TABLE}} ORDER BY id")
write("count=" + str(len(rows)))
write("row0=" + rows[0]["name"].string() + ":" + str(rows[0]["score"].real()))

attempt {
    db.query("SELECT name AS dup, score AS dup FROM {{TABLE}}")
    write("unexpected-dup")
} except MySQLError as err {
    write("dup-rejected")
}

attempt {
    bad: Local MySQLDatabase := MySQL.connect({{BAD_CONNECT}})
    bad.close()
    write("unexpected-bad-auth")
} except MySQLError as err {
    if err.message.contains("Dispos4ble_Wrong_Secret") {
        write("secret-leaked")
    } else {
        write("auth-redacted")
    }
}

db.close()
`

func mysqlAhdLiteral(t *testing.T, value string) string {
	t.Helper()
	if strings.ContainsAny(value, "\"\r\n") {
		t.Skip("AHDCODE_TEST_MYSQL_* values with quotes or line breaks cannot be embedded in the acceptance program")
	}
	return `r"` + value + `"`
}

func TestMySQLAcceptanceProgramCompiles(t *testing.T) {
	source := strings.NewReplacer(
		"{{CONNECT}}", `r"127.0.0.1", r"app", r"pw", 3306, null, r"none", 10`,
		"{{BAD_CONNECT}}", `r"127.0.0.1", r"app", r"Dispos4ble_Wrong_Secret", 3306, null, r"none", 5`,
		"{{TABLE}}", "ahd_test_compile",
	).Replace(mysqlAcceptanceSource)
	directory := writeSources(t, map[string]string{"main.ahd": source})
	compiled := Compile(filepath.Join(directory, "main.ahd"))
	if compiled.HasErrors() {
		t.Fatalf("acceptance program does not compile:\n%s", diagnosticText(compiled.Diagnostics))
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

func TestMySQLIntegrationNativeAndEvaluatorAgree(t *testing.T) {
	host := os.Getenv("AHDCODE_TEST_MYSQL_HOST")
	if host == "" {
		t.Skip("set AHDCODE_TEST_MYSQL_HOST (with _PORT, _USERNAME, _PASSWORD, _DATABASE, _SECURITY) to run live MySQL native/evaluator parity tests")
	}
	user := os.Getenv("AHDCODE_TEST_MYSQL_USERNAME")
	pass := os.Getenv("AHDCODE_TEST_MYSQL_PASSWORD")
	dbName := os.Getenv("AHDCODE_TEST_MYSQL_DATABASE")
	security := os.Getenv("AHDCODE_TEST_MYSQL_SECURITY")
	if security == "" {
		security = "none"
	}
	port := int64(3306)
	if text := os.Getenv("AHDCODE_TEST_MYSQL_PORT"); text != "" {
		p, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		port = p
	}

	var dbPtr *string
	if dbName != "" {
		dbPtr = &dbName
	}
	admin, err := ahdruntime.MySQLConnect(host, user, pass, port, dbPtr, security, 10)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	defer ahdruntime.MySQLClose(admin)

	dbLiteral := "null"
	if dbName != "" {
		dbLiteral = mysqlAhdLiteral(t, dbName)
	}
	connectArgs := strings.Join([]string{
		mysqlAhdLiteral(t, host), mysqlAhdLiteral(t, user), mysqlAhdLiteral(t, pass),
		strconv.FormatInt(port, 10), dbLiteral, mysqlAhdLiteral(t, security), "10",
	}, ", ")
	badConnectArgs := strings.Join([]string{
		mysqlAhdLiteral(t, host), mysqlAhdLiteral(t, user), `r"Dispos4ble_Wrong_Secret"`,
		strconv.FormatInt(port, 10), dbLiteral, mysqlAhdLiteral(t, security), "5",
	}, ", ")

	runMode := func(t *testing.T, mode string) string {
		t.Helper()
		randBytes := make([]byte, 4)
		if _, err := rand.Read(randBytes); err != nil {
			t.Fatal(err)
		}
		table := "ahd_parity_" + mode + "_" + hex.EncodeToString(randBytes)
		t.Cleanup(func() {
			_, _ = ahdruntime.MySQLExecute(admin, "DROP TABLE IF EXISTS "+table, nil)
		})

		src := strings.NewReplacer(
			"{{CONNECT}}", connectArgs,
			"{{BAD_CONNECT}}", badConnectArgs,
			"{{TABLE}}", table,
		).Replace(mysqlAcceptanceSource)

		if mode == "native" {
			dir := writeSources(t, map[string]string{"main.ahd": src})
			stdout, stderr, code := buildAndRun(t, filepath.Join(dir, "main.ahd"), "")
			if code != 0 || stderr != "" {
				t.Fatalf("native run failed: code=%d stderr=%q\nstdout:\n%s", code, stderr, stdout)
			}
			return strings.ReplaceAll(stdout, "\r\n", "\n")
		}

		workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": src})
		frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
		if frontend.HasErrors() {
			t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
		}
		lowered := lowering.LowerCompilation(frontend)
		if lowered.HasErrors() {
			t.Fatalf("lowering diagnostics: %+v", lowered.Diagnostics)
		}
		var output bytes.Buffer
		session := evaluator.New(bufio.NewReader(strings.NewReader("")), &output, t.TempDir())
		if result := session.Execute(lowered.Compilation, 0); result.Failure != nil {
			t.Fatalf("evaluator raised %s: %s\noutput:\n%s", result.Failure.Name, result.Failure.Message, output.String())
		}
		return strings.ReplaceAll(output.String(), "\r\n", "\n")
	}

	nativeOut := runMode(t, "native")
	evalOut := runMode(t, "evaluator")
	want := strings.Join([]string{
		"affected=1",
		"has-id=true",
		"count=1",
		"row0=Ayşe:95.0",
		"dup-rejected",
		"auth-redacted",
	}, "\n") + "\n"
	if nativeOut != want {
		t.Fatalf("native output:\n%s\nwant:\n%s", nativeOut, want)
	}
	if evalOut != want {
		t.Fatalf("evaluator output:\n%s\nwant:\n%s", evalOut, want)
	}
}
