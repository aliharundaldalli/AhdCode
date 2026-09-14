package golang

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// The PostgreSQL runtime imports the vendored pgx graph, so it reaches a
// generated program only when the program uses PostgreSQL.
func TestPostgreSQLRuntimeOnlyJoinsProgramsThatUseIt(t *testing.T) {
	plain := generate(t, "bring MySQL\nfrom MySQL bring MySQLValue\nvalue: MySQLValue := MySQL.fromInt(1)\n")
	if plain.RequiresPostgreSQL {
		t.Fatal("a MySQL program requires the PostgreSQL tree")
	}
	for _, file := range plain.Files {
		if file.Name == postgresqlRuntimeFileName {
			t.Fatal("a program without PostgreSQL received its runtime")
		}
	}
	program := generate(t, "bring PostgreSQL\nfrom PostgreSQL bring PostgreSQLValue\nvalue: PostgreSQLValue := PostgreSQL.fromBool(true)\nwrite(value.kind())\n")
	if !program.RequiresPostgreSQL || program.RequiresMySQL {
		t.Fatalf("flags: postgresql=%v mysql=%v", program.RequiresPostgreSQL, program.RequiresMySQL)
	}
	seen := 0
	for _, file := range program.Files {
		if file.Name != postgresqlRuntimeFileName {
			continue
		}
		seen++
		parsed, err := parser.ParseFile(token.NewFileSet(), file.Name, file.Content, 0)
		if err != nil {
			t.Fatalf("PostgreSQL runtime is not valid Go: %v", err)
		}
		if parsed.Name.Name != "main" {
			t.Fatalf("PostgreSQL runtime package clause not rewritten: %s", parsed.Name.Name)
		}
		for _, spec := range parsed.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if strings.Contains(path, ".") && !strings.HasPrefix(path, "github.com/jackc/pgx/v5") {
				t.Fatalf("PostgreSQL runtime imports %s", path)
			}
			for _, forbidden := range []string{"os/exec", "syscall", "unsafe", "database/sql"} {
				if path == forbidden {
					t.Fatalf("PostgreSQL runtime imports %s", path)
				}
			}
		}
	}
	if seen != 1 {
		t.Fatalf("PostgreSQL runtime emitted %d times, want 1", seen)
	}
}
