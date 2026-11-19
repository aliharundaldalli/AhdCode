// Package migrate is the versioned migration workflow behind `ahdcode migrate`.
// It applies author-written SQL files to a SQLite database and records each
// version in ahd_schema_migrations. It is not a database framework and it
// does not change the SQLite or MySQL language modules.
package migrate

import (
	"errors"
	"fmt"
	"time"

	"github.com/ncruces/go-sqlite3"
)

// Options selects the database file and the migration directory.
// Driver is sqlite when empty. Any other driver is rejected before a file
// is opened.
type Options struct {
	Database  string
	Directory string
	Driver    string
}

// Status is the applied history followed by the pending files.
type Status struct {
	DatabaseExists bool
	Applied        []appliedRow
	Pending        []File
}

// Result is one migration the command applied or rolled back.
type Result struct {
	Label   string
	Version int64
	Name    string
}

// Up applies every pending migration in version order. Each file and its
// history row commit together. A later failure leaves earlier versions
// applied. Re-running skips versions that are already recorded.
func Up(options Options) ([]Result, error) {
	if err := checkDriver(options.Driver); err != nil {
		return nil, err
	}
	files, err := Scan(options.Directory)
	if err != nil {
		return nil, err
	}
	conn, err := openSQLite(options.Database, openCreate)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var applied []Result
	for {
		did, result, err := applyNext(conn, files)
		if err != nil {
			return applied, err
		}
		if !did {
			return applied, nil
		}
		applied = append(applied, result)
	}
}

// Down rolls back the last applied migrations, one version at a time, and
// only when that version still has the down script that was recorded when
// it was applied. It never runs as part of a failed up.
func Down(options Options, steps int) ([]Result, error) {
	if steps < 1 {
		return nil, errors.New("steps must be at least 1")
	}
	if err := checkDriver(options.Driver); err != nil {
		return nil, err
	}
	files, err := Scan(options.Directory)
	if err != nil {
		return nil, err
	}
	conn, err := openSQLite(options.Database, openReadWrite)
	if err != nil {
		if errors.Is(err, errDatabaseMissing) {
			return nil, errors.New("database file does not exist; there is nothing to roll back")
		}
		return nil, err
	}
	defer conn.Close()
	rows, err := loadHistory(conn)
	if err != nil {
		return nil, err
	}
	if _, err := checkConsistent(files, rows); err != nil {
		return nil, err
	}
	if steps > len(rows) {
		return nil, fmt.Errorf("cannot roll back %d migration(s); only %d are applied", steps, len(rows))
	}
	byVersion := map[int64]File{}
	for _, file := range files {
		byVersion[file.Version] = file
	}
	for _, row := range rows[len(rows)-steps:] {
		file := byVersion[row.Version]
		if file.DownChecksum == "" || file.DownPath == "" {
			return nil, fmt.Errorf("migration %s has no down script; it was applied without one and will not be rolled back", file.Label)
		}
		if trimSpace(file.DownSQL) == "" {
			return nil, fmt.Errorf("migration %s has an empty down script", file.Label)
		}
	}
	var rolled []Result
	for count := 0; count < steps; count++ {
		result, err := revertTip(conn, files)
		if err != nil {
			return rolled, err
		}
		rolled = append(rolled, result)
	}
	return rolled, nil
}

// Inspect reports applied and pending migrations without changing the database.
// A missing database file is an empty history, not a created file.
func Inspect(options Options) (Status, error) {
	if err := checkDriver(options.Driver); err != nil {
		return Status{}, err
	}
	files, err := Scan(options.Directory)
	if err != nil {
		return Status{}, err
	}
	conn, err := openSQLite(options.Database, openReadOnly)
	if err != nil {
		if errors.Is(err, errDatabaseMissing) {
			return Status{Pending: files}, nil
		}
		return Status{}, err
	}
	defer conn.Close()
	rows, err := loadHistory(conn)
	if err != nil {
		return Status{}, err
	}
	pending, err := checkConsistent(files, rows)
	if err != nil {
		return Status{}, err
	}
	return Status{DatabaseExists: true, Applied: rows, Pending: pending}, nil
}

func checkDriver(driver string) error {
	switch driver {
	case "", "sqlite":
		return nil
	case "mysql":
		return errors.New("MySQL migrations are not supported. The MySQL module is a program runtime, and MySQL DDL cannot share one transaction with the migration history. No migration was applied")
	default:
		return fmt.Errorf("unknown database driver %q; the supported driver is sqlite", driver)
	}
}

// applyNext locks the database, applies at most one pending migration, and
// reports whether it did. A version committed by another process is skipped.
func applyNext(conn *sqlite3.Conn, files []File) (bool, Result, error) {
	if err := beginImmediate(conn); err != nil {
		return false, Result{}, err
	}
	committed := false
	defer func() {
		if !committed {
			rollback(conn)
		}
	}()
	rows, err := loadHistory(conn)
	if err != nil {
		return false, Result{}, err
	}
	pending, err := checkConsistent(files, rows)
	if err != nil {
		return false, Result{}, err
	}
	if len(pending) == 0 {
		if err := conn.Exec("COMMIT"); err != nil {
			return false, Result{}, err
		}
		committed = true
		return false, Result{}, nil
	}
	file := pending[0]
	freshSQL, freshChecksum, err := readScript(file.UpPath)
	if err != nil {
		return false, Result{}, err
	}
	if freshChecksum != file.UpChecksum {
		return false, Result{}, fmt.Errorf("migration %s changed while it was being applied", file.Label)
	}
	file.UpSQL = freshSQL
	if err := ensureHistory(conn); err != nil {
		return false, Result{}, err
	}
	if err := execScript(conn, file.UpSQL, file.Label); err != nil {
		return false, Result{}, err
	}
	if interruptAfterScript != nil {
		if err := interruptAfterScript(conn); err != nil {
			return false, Result{}, err
		}
	}
	if err := insertHistory(conn, file, time.Now().Unix()); err != nil {
		return false, Result{}, err
	}
	if err := conn.Exec("COMMIT"); err != nil {
		return false, Result{}, fmt.Errorf("could not commit migration %s: %s", file.Label, err.Error())
	}
	committed = true
	return true, Result{Label: file.Label, Version: file.Version, Name: file.Name}, nil
}

func revertTip(conn *sqlite3.Conn, files []File) (Result, error) {
	if err := beginImmediate(conn); err != nil {
		return Result{}, err
	}
	committed := false
	defer func() {
		if !committed {
			rollback(conn)
		}
	}()
	rows, err := loadHistory(conn)
	if err != nil {
		return Result{}, err
	}
	if _, err := checkConsistent(files, rows); err != nil {
		return Result{}, err
	}
	if len(rows) == 0 {
		return Result{}, errors.New("there is no applied migration to roll back")
	}
	var file File
	for _, candidate := range files {
		if candidate.Version == rows[len(rows)-1].Version {
			file = candidate
			break
		}
	}
	if file.Version == 0 {
		return Result{}, fmt.Errorf("applied migration %d is not in the migration directory", rows[len(rows)-1].Version)
	}
	if err := rows[len(rows)-1].matches(file); err != nil {
		return Result{}, err
	}
	if file.DownChecksum == "" || trimSpace(file.DownSQL) == "" {
		return Result{}, fmt.Errorf("migration %s has no down script; it was applied without one and will not be rolled back", file.Label)
	}
	freshSQL, freshChecksum, err := readScript(file.DownPath)
	if err != nil {
		return Result{}, err
	}
	if freshChecksum != file.DownChecksum {
		return Result{}, fmt.Errorf("the down script for applied migration %s changed after it was applied", file.Label)
	}
	if err := execScript(conn, freshSQL, file.Label+" down"); err != nil {
		return Result{}, err
	}
	if err := deleteHistory(conn, file.Version); err != nil {
		return Result{}, err
	}
	if err := conn.Exec("COMMIT"); err != nil {
		return Result{}, fmt.Errorf("could not commit rollback of %s: %s", file.Label, err.Error())
	}
	committed = true
	return Result{Label: file.Label, Version: file.Version, Name: file.Name}, nil
}

func trimSpace(value string) string {
	start, end := 0, len(value)
	for start < end {
		switch value[start] {
		case ' ', '\t', '\n', '\r':
			start++
		default:
			goto tail
		}
	}
tail:
	for end > start {
		switch value[end-1] {
		case ' ', '\t', '\n', '\r':
			end--
		default:
			return value[start:end]
		}
	}
	return ""
}
