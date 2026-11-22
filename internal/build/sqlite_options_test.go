package build

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

var (
	sqliteOptionsHelperOnce sync.Once
	sqliteOptionsHelperPath string
	sqliteOptionsHelperErr  error
)

func ensureSQLiteOptionsHelper(t *testing.T) {
	t.Helper()
	if existing := os.Getenv("AHDCODE_SQLITE_RUNTIME"); existing != "" {
		if info, err := os.Stat(existing); err == nil && info.Mode().IsRegular() {
			return
		}
	}
	sqliteOptionsHelperOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ahdcode-sqlite-helper-*")
		if err != nil {
			sqliteOptionsHelperErr = err
			return
		}
		helper := filepath.Join(dir, "ahdsqlite")
		if runtime.GOOS == "windows" {
			helper += ".exe"
		}
		root, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			sqliteOptionsHelperErr = err
			return
		}
		cmd := exec.Command("go", "build", "-o", helper, "./cmd/ahdsqlite")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			sqliteOptionsHelperErr = &os.PathError{Op: "build", Path: helper, Err: os.ErrInvalid}
			_ = out
			return
		}
		sqliteOptionsHelperPath = helper
	})
	if sqliteOptionsHelperErr != nil {
		t.Fatalf("build ahdsqlite helper: %v", sqliteOptionsHelperErr)
	}
	t.Setenv("AHDCODE_SQLITE_RUNTIME", sqliteOptionsHelperPath)
}

const sqliteOptionsParitySource = `bring SQLite
from SQLite bring (Database, SQLiteError, SQLiteValue)

dbPath: String := {{DB_PATH}}
missingPath: String := {{MISSING_PATH}}
backupPath: String := {{BACKUP_PATH}}

attempt {
    bad: Local Database := SQLite.openWithOptions(missingPath, true, 0)
    bad.close()
    write("unexpected-missing-open")
} except SQLiteError as err {
    write("missing-ro-failed")
}

attempt {
    badMem: Local Database := SQLite.openWithOptions(":memory:", true, 0)
    badMem.close()
    write("unexpected-mem-ro")
} except SQLiteError as err {
    write("mem-ro-failed")
}

attempt {
    badTimeout: Local Database := SQLite.openWithOptions(":memory:", false, -5)
    badTimeout.close()
    write("unexpected-neg-timeout")
} except SQLiteError as err {
    write("neg-timeout-failed")
}

rw: Database := SQLite.openWithOptions(dbPath, false, 1500)
rw.execute("PRAGMA journal_mode=WAL")
rw.execute("CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL)")
rw.begin()
rw.execute("INSERT INTO notes (body) VALUES (?)", [SQLite.fromString("rolled-back")])
rw.rollback()
rw.begin()
rw.execute("INSERT INTO notes (body) VALUES (?)", [SQLite.fromString("saved-ü-#%")])
rw.commit()
rw.backupTo(backupPath)
rw.close()

ro: Database := SQLite.openWithOptions(backupPath, true, 500)
rows: List<Pair<String, SQLiteValue>> := ro.query("SELECT body FROM notes")
write("row=" + rows[0]["body"].string())
attempt {
    ro.execute("INSERT INTO notes (body) VALUES (?)", [SQLite.fromString("no")])
    write("unexpected-ro-write")
} except SQLiteError as err {
    write("ro-write-blocked")
}
ro.close()
`

func TestSQLiteOpenWithOptionsAndBackupNativeEvaluatorParity(t *testing.T) {
	ensureSQLiteOptionsHelper(t)

	runMode := func(t *testing.T, mode string) string {
		t.Helper()
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "app ü #1%.db")
		missingPath := filepath.Join(dir, "missing.db")
		backupPath := filepath.Join(dir, "backup ü #2%.db")

		rawLit := func(p string) string {
			return `r"` + filepath.ToSlash(p) + `"`
		}
		src := strings.NewReplacer(
			"{{DB_PATH}}", rawLit(dbPath),
			"{{MISSING_PATH}}", rawLit(missingPath),
			"{{BACKUP_PATH}}", rawLit(backupPath),
		).Replace(sqliteOptionsParitySource)

		if mode == "native" {
			srcDir := writeSources(t, map[string]string{"main.ahd": src})
			stdout, stderr, code := buildAndRun(t, filepath.Join(srcDir, "main.ahd"), "")
			if code != 0 || stderr != "" {
				t.Fatalf("native run failed: code=%d stderr=%q\nstdout:\n%s", code, stderr, stdout)
			}
			if _, statErr := os.Stat(missingPath); !os.IsNotExist(statErr) {
				t.Fatalf("native read-only open created %s", missingPath)
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
		session := evaluator.New(bufio.NewReader(strings.NewReader("")), &output, dir)
		if result := session.Execute(lowered.Compilation, 0); result.Failure != nil {
			t.Fatalf("evaluator raised %s: %s\noutput:\n%s", result.Failure.Name, result.Failure.Message, output.String())
		}
		if _, statErr := os.Stat(missingPath); !os.IsNotExist(statErr) {
			t.Fatalf("evaluator read-only open created %s", missingPath)
		}
		return strings.ReplaceAll(output.String(), "\r\n", "\n")
	}

	nativeOut := runMode(t, "native")
	evalOut := runMode(t, "evaluator")
	want := strings.Join([]string{
		"missing-ro-failed",
		"mem-ro-failed",
		"neg-timeout-failed",
		"row=saved-ü-#%",
		"ro-write-blocked",
	}, "\n") + "\n"

	if nativeOut != want {
		t.Fatalf("native output:\n%s\nwant:\n%s", nativeOut, want)
	}
	if evalOut != want {
		t.Fatalf("evaluator output:\n%s\nwant:\n%s", evalOut, want)
	}
}
