package build

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	backend "ahdcode/internal/backend/golang"
	"ahdcode/internal/backend/golang/ahdruntime/bcryptvendor"
	"ahdcode/internal/backend/golang/ahdruntime/codesvendor"
	"ahdcode/internal/backend/golang/ahdruntime/mysqlvendor"
	"ahdcode/internal/backend/golang/ahdruntime/postgresqlvendor"
	"ahdcode/internal/backend/golang/ahdruntime/websocketvendor"
)

type embeddedTree struct {
	name     string
	files    fs.ReadFileFS
	requires []string
	sum      string
}

func embeddedTrees() []embeddedTree {
	return []embeddedTree{
		{"mysql", mysqlvendor.Vendor, mysqlvendor.Requires, mysqlvendor.GoSum},
		{"codes", codesvendor.Vendor, codesvendor.Requires, codesvendor.GoSum},
		{"websocket", websocketvendor.Vendor, websocketvendor.Requires, websocketvendor.GoSum},
		{"postgresql", postgresqlvendor.Vendor, postgresqlvendor.Requires, postgresqlvendor.GoSum},
		{"bcrypt", bcryptvendor.Vendor, bcryptvendor.Requires, bcryptvendor.GoSum},
	}
}

// Every embedded tree is self-consistent -- each module in modules.txt has a
// require line and a go.sum entry -- and no module appears in two trees, so
// composing any combination into one workspace can never list a module twice.
func TestEmbeddedVendorTreesAreConsistentAndDisjoint(t *testing.T) {
	owners := map[string]string{}
	for _, tree := range embeddedTrees() {
		modules, err := tree.files.ReadFile("vendor/modules.txt")
		if err != nil {
			t.Fatalf("%s tree has no modules.txt: %v", tree.name, err)
		}
		for _, line := range strings.Split(string(modules), "\n") {
			if !strings.HasPrefix(line, "# ") {
				continue
			}
			fields := strings.Fields(strings.TrimPrefix(line, "# "))
			if len(fields) != 2 {
				t.Fatalf("%s tree has a malformed module line %q", tree.name, line)
			}
			path, version := fields[0], fields[1]
			if owner, seen := owners[path]; seen {
				t.Fatalf("module %s is vendored by both the %s and %s trees", path, owner, tree.name)
			}
			owners[path] = tree.name
			required := false
			for _, requirement := range tree.requires {
				if strings.HasPrefix(requirement, path+" "+version) {
					required = true
				}
			}
			if !required {
				t.Fatalf("%s tree vendors %s %s without a require line", tree.name, path, version)
			}
			if !strings.Contains(tree.sum, path+" "+version+" h1:") || !strings.Contains(tree.sum, path+" "+version+"/go.mod h1:") {
				t.Fatalf("%s tree lacks go.sum entries for %s %s", tree.name, path, version)
			}
		}
		if len(tree.requires) == 0 {
			t.Fatalf("%s tree has no require lines", tree.name)
		}
	}
}

// A program needing all four trees composes one vendored workspace that builds
// with the network disabled and an empty module cache.
func TestAllVendorTreesComposeAndBuildOffline(t *testing.T) {
	program := &backend.GeneratedProgram{
		Files: []backend.GeneratedFile{{Name: "main.go", Content: `package main

import (
	"fmt"

	_ "github.com/boombuler/barcode/qr"
	_ "github.com/coder/websocket"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/pgxpool"
	_ "golang.org/x/crypto/bcrypt"
)

func main() { fmt.Println("offline") }
`}},
		RequiresMySQL: true, RequiresCodes: true, RequiresWebSocket: true, RequiresPostgreSQL: true, RequiresBcrypt: true,
	}
	workspace, diagnostics := NewWorkspace(program)
	if len(diagnostics) != 0 {
		t.Fatalf("workspace failed: %s", diagnosticText(diagnostics))
	}
	defer workspace.Close()
	goMod, _ := os.ReadFile(filepath.Join(workspace.Directory, "go.mod"))
	for _, tree := range embeddedTrees() {
		for _, requirement := range tree.requires {
			if !strings.Contains(string(goMod), requirement) {
				t.Fatalf("composed go.mod lacks %q:\n%s", requirement, goMod)
			}
		}
	}
	t.Setenv("GOMODCACHE", t.TempDir())
	output := filepath.Join(t.TempDir(), "offline")
	if diagnostics := workspace.BuildExecutable(output); len(diagnostics) != 0 {
		t.Fatalf("five-tree vendored build failed: %s", diagnosticText(diagnostics))
	}
	result, err := exec.Command(output).CombinedOutput()
	if err != nil || string(result) != "offline\n" {
		t.Fatalf("five-tree program failed: %v %q", err, result)
	}
}

// Only the flags a program actually sets bring their trees: a WebSocket-only
// program never receives the PostgreSQL graph, and the reverse.
func TestVendorTreesFollowProgramRequirements(t *testing.T) {
	cases := []struct {
		program  *backend.GeneratedProgram
		present  []string
		excluded []string
	}{
		{&backend.GeneratedProgram{RequiresWebSocket: true}, []string{"github.com/coder/websocket"}, []string{"github.com/jackc/pgx/v5", "github.com/go-sql-driver/mysql"}},
		{&backend.GeneratedProgram{RequiresPostgreSQL: true}, []string{"github.com/jackc/pgx/v5", "golang.org/x/text"}, []string{"github.com/coder/websocket", "github.com/boombuler/barcode"}},
		{&backend.GeneratedProgram{RequiresBcrypt: true}, []string{"golang.org/x/crypto"}, []string{"github.com/jackc/pgx/v5", "golang.org/x/text"}},
		{&backend.GeneratedProgram{}, nil, []string{"github.com/coder/websocket", "github.com/jackc/pgx/v5", "golang.org/x/crypto"}},
	}
	for _, testCase := range cases {
		workspace, diagnostics := NewWorkspace(testCase.program)
		if len(diagnostics) != 0 {
			t.Fatalf("workspace failed: %s", diagnosticText(diagnostics))
		}
		goMod, _ := os.ReadFile(filepath.Join(workspace.Directory, "go.mod"))
		for _, module := range testCase.present {
			if !strings.Contains(string(goMod), module+" ") {
				t.Fatalf("go.mod lacks %s:\n%s", module, goMod)
			}
		}
		for _, module := range testCase.excluded {
			if strings.Contains(string(goMod), module+" ") {
				t.Fatalf("go.mod unexpectedly has %s:\n%s", module, goMod)
			}
		}
		workspace.Close()
	}
}
