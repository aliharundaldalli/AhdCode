package build

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A PostgreSQL program compiles from the vendored pgx graph with the network
// and module cache unavailable, and its value operations and connection
// failures behave natively exactly as documented -- no server required.
func TestPostgreSQLNativeProgramWithoutAServer(t *testing.T) {
	port := freeLoopbackPort(t)
	source := `bring PostgreSQL
from PostgreSQL bring (PostgreSQLDatabase, PostgreSQLValue, PostgreSQLError)

flag: PostgreSQLValue := PostgreSQL.fromBool(true)
write(flag.kind() + " " + str(flag.bool()))
count: PostgreSQLValue := PostgreSQL.fromInt(7)
write(str(count.real() == 7.0) + " " + str(PostgreSQL.nullValue().isNull()))
attempt {
    write(count.string())
} except PostgreSQLError as error {
    write(error.message)
}
attempt {
    db: Local PostgreSQLDatabase := PostgreSQL.connect("127.0.0.1", "app", "super-secret", ` + strconv.Itoa(port) + `, null, "none", 2)
    write("connected")
} except PostgreSQLError as error {
    write(error.message)
}
attempt {
    PostgreSQL.connect("127.0.0.1", "app", "x", 70000)
} except PostgreSQLError as error {
    write(error.message)
}
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	entry := filepath.Join(directory, "main.ahd")
	compiled := Compile(entry)
	if compiled.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(compiled.Diagnostics))
	}
	if !compiled.Program.RequiresPostgreSQL || compiled.Program.RequiresMySQL || compiled.Program.RequiresWebSocket {
		t.Fatalf("flags: postgresql=%v mysql=%v websocket=%v", compiled.Program.RequiresPostgreSQL, compiled.Program.RequiresMySQL, compiled.Program.RequiresWebSocket)
	}
	workspace, diagnostics := NewWorkspace(compiled.Program)
	if len(diagnostics) != 0 {
		t.Fatalf("workspace failed: %s", diagnosticText(diagnostics))
	}
	goMod, _ := os.ReadFile(filepath.Join(workspace.Directory, "go.mod"))
	workspace.Close()
	if !strings.Contains(string(goMod), "github.com/jackc/pgx/v5 v5.11.0") {
		t.Fatalf("workspace go.mod lacks pgx:\n%s", goMod)
	}
	t.Setenv("GOMODCACHE", t.TempDir())
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("PostgreSQL program failed: code=%d stderr=%q", code, stderr)
	}
	want := "Bool true\n" +
		"true true\n" +
		"string() requires kind String; this PostgreSQLValue has kind Int (check kind() first)\n" +
		"PostgreSQL connection failed\n" +
		"PostgreSQL port must be in 1..65535\n"
	if stdout != want {
		t.Fatalf("PostgreSQL native output mismatch:\n got: %q\nwant: %q", stdout, want)
	}
	if strings.Contains(stdout, "super-secret") {
		t.Fatal("the password reached program output")
	}
}
