// Command ahdsqlite is the bundled pure-Go SQLite helper behind the SQLite
// standard module. It links the SQLite engine (github.com/ncruces/go-sqlite3,
// CGO-free) so that generated AhdCode programs stay stdlib-only, exactly the
// way ahdnumeric and ahdplot isolate their own third-party dependencies.
//
// The helper is a long-lived session: it reads one JSON request per line from
// standard input and writes one JSON response per line to standard output.
// Every open Database is exactly one SQLite connection owned by this process,
// so transaction state, :memory: contents, and last_insert_rowid all belong
// to that single logical connection.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"ahdcode/internal/sqliteproto"
	"github.com/ncruces/go-sqlite3"
)

// busyTimeout is how long a statement waits for another process's lock before
// SQLite reports "database is locked". Five seconds matches the common default
// of other SQLite bindings.
const busyTimeout = 5 * time.Second

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ahdsqlite (reads JSON requests from standard input)")
		os.Exit(2)
	}
	serve(os.Stdin, os.Stdout)
}

// database is one open (or already closed) logical connection. A closed
// handle keeps its entry so later use reports "closed" instead of "unknown".
type database struct {
	conn *sqlite3.Conn
}

type server struct {
	databases map[int64]*database
	next      int64
}

func serve(input io.Reader, output io.Writer) {
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	engine := &server{databases: make(map[int64]*database), next: 1}
	defer engine.closeAll()
	for {
		var request sqliteproto.Request
		if err := decoder.Decode(&request); err != nil {
			if !errors.Is(err, io.EOF) {
				_ = encoder.Encode(sqliteproto.Response{Error: "malformed SQLite helper request: " + err.Error()})
			}
			return
		}
		if err := encoder.Encode(engine.handle(request)); err != nil {
			return
		}
	}
}

func (engine *server) closeAll() {
	for _, entry := range engine.databases {
		if entry.conn != nil {
			_ = entry.conn.Close()
			entry.conn = nil
		}
	}
}

func (engine *server) handle(request sqliteproto.Request) (response sqliteproto.Response) {
	defer func() {
		if recovered := recover(); recovered != nil {
			response = sqliteproto.Response{Error: fmt.Sprintf("SQLite helper failure: %v", recovered)}
		}
	}()
	if request.Operation == sqliteproto.OperationOpen {
		return engine.open(request.Path)
	}
	entry, known := engine.databases[request.Database]
	if !known {
		return failure(errors.New("unknown Database handle"))
	}
	if entry.conn == nil && request.Operation == sqliteproto.OperationClose {
		// close() is idempotent: closing an already closed Database succeeds.
		return sqliteproto.Response{}
	}
	if entry.conn == nil {
		return failure(errors.New("the Database is closed"))
	}
	switch request.Operation {
	case sqliteproto.OperationExecute:
		return entry.execute(request.SQL, request.Parameters)
	case sqliteproto.OperationQuery:
		return entry.query(request.SQL, request.Parameters)
	case sqliteproto.OperationLastInsertID:
		return sqliteproto.Response{RowID: entry.conn.LastInsertRowID()}
	case sqliteproto.OperationBegin:
		if !entry.conn.GetAutocommit() {
			return failure(errors.New("a transaction is already active; call commit() or rollback() before begin()"))
		}
		return failure(entry.conn.Exec("BEGIN"))
	case sqliteproto.OperationCommit:
		if entry.conn.GetAutocommit() {
			return failure(errors.New("no transaction is active; call begin() before commit()"))
		}
		return failure(entry.conn.Exec("COMMIT"))
	case sqliteproto.OperationRollback:
		if entry.conn.GetAutocommit() {
			return failure(errors.New("no transaction is active; there is nothing to roll back"))
		}
		return failure(entry.conn.Exec("ROLLBACK"))
	case sqliteproto.OperationClose:
		if !entry.conn.GetAutocommit() {
			return failure(errors.New("the Database still has an active transaction; call commit() or rollback() before close()"))
		}
		if err := entry.conn.Close(); err != nil {
			return failure(err)
		}
		entry.conn = nil
		return sqliteproto.Response{}
	}
	return failure(fmt.Errorf("unknown SQLite helper operation %q", request.Operation))
}

func (engine *server) open(path string) sqliteproto.Response {
	if path == "" {
		return failure(errors.New("the database path is empty; use \":memory:\" or a file path"))
	}
	// OPEN_URI is deliberately absent: the String is a filesystem path (or
	// the :memory: marker), never a URI, so no query-string syntax is implied.
	conn, err := sqlite3.OpenFlags(path, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE)
	if err != nil {
		return failure(err)
	}
	if err := conn.BusyTimeout(busyTimeout); err != nil {
		_ = conn.Close()
		return failure(err)
	}
	handle := engine.next
	engine.next++
	engine.databases[handle] = &database{conn: conn}
	return sqliteproto.Response{Database: handle}
}

// prepare compiles exactly one SQL statement and binds its parameters. Any
// second statement in the text is rejected: one call is one statement, so a
// parameter list can never silently apply to a different statement than the
// one the programmer meant.
func (entry *database) prepare(sql string, parameters []sqliteproto.Value) (*sqlite3.Stmt, error) {
	stmt, tail, err := entry.conn.Prepare(sql)
	if err != nil {
		return nil, err
	}
	if stmt == nil {
		return nil, errors.New("the SQL text contains no statement")
	}
	if strings.TrimSpace(tail) != "" {
		rest, _, restErr := entry.conn.Prepare(tail)
		if rest != nil {
			_ = rest.Close()
		}
		if restErr != nil || rest != nil {
			_ = stmt.Close()
			return nil, errors.New("execute and query run exactly one SQL statement; split the text into one call per statement")
		}
	}
	if placeholders := stmt.BindCount(); placeholders != len(parameters) {
		_ = stmt.Close()
		return nil, fmt.Errorf("the SQL statement has %d parameter placeholder(s); received %d value(s)", placeholders, len(parameters))
	}
	for index, parameter := range parameters {
		position := index + 1
		switch parameter.Kind {
		case sqliteproto.KindNull:
			err = stmt.BindNull(position)
		case sqliteproto.KindInt:
			err = stmt.BindInt64(position, parameter.Int)
		case sqliteproto.KindReal:
			if math.IsNaN(parameter.Real) || math.IsInf(parameter.Real, 0) {
				err = fmt.Errorf("parameter %d is not a finite Real", position)
			} else {
				err = stmt.BindFloat(position, parameter.Real)
			}
		case sqliteproto.KindString:
			err = stmt.BindText(position, parameter.String)
		default:
			err = fmt.Errorf("parameter %d has unsupported kind %q", position, parameter.Kind)
		}
		if err != nil {
			_ = stmt.Close()
			return nil, err
		}
	}
	return stmt, nil
}

// execute runs one statement and reports how many rows it inserted, updated,
// or deleted (including rows changed by its triggers). Statements that change
// no rows, such as CREATE TABLE, report 0.
func (entry *database) execute(sql string, parameters []sqliteproto.Value) sqliteproto.Response {
	stmt, err := entry.prepare(sql, parameters)
	if err != nil {
		return failure(err)
	}
	before := entry.conn.TotalChanges()
	err = stmt.Exec()
	if closeErr := stmt.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return failure(err)
	}
	return sqliteproto.Response{Changed: entry.conn.TotalChanges() - before}
}

// query runs one statement and materializes every result row using the
// runtime storage class of each value, never the declared column type.
func (entry *database) query(sql string, parameters []sqliteproto.Value) sqliteproto.Response {
	stmt, err := entry.prepare(sql, parameters)
	if err != nil {
		return failure(err)
	}
	defer stmt.Close()
	count := stmt.ColumnCount()
	columns := make([]string, count)
	seen := make(map[string]bool, count)
	for index := range columns {
		name := stmt.ColumnName(index)
		if seen[name] {
			return failure(fmt.Errorf("the result has the duplicate column label %q; give each column a distinct name with AS", name))
		}
		seen[name] = true
		columns[index] = name
	}
	rows := make([][]sqliteproto.Value, 0)
	for stmt.Step() {
		row := make([]sqliteproto.Value, count)
		for index := range row {
			switch stmt.ColumnType(index) {
			case sqlite3.NULL:
				row[index] = sqliteproto.Value{Kind: sqliteproto.KindNull}
			case sqlite3.INTEGER:
				row[index] = sqliteproto.Value{Kind: sqliteproto.KindInt, Int: stmt.ColumnInt64(index)}
			case sqlite3.FLOAT:
				value := stmt.ColumnFloat(index)
				if math.IsNaN(value) || math.IsInf(value, 0) {
					return failure(fmt.Errorf("column %q holds a non-finite REAL value that an AhdCode Real cannot represent", columns[index]))
				}
				row[index] = sqliteproto.Value{Kind: sqliteproto.KindReal, Real: value}
			case sqlite3.TEXT:
				row[index] = sqliteproto.Value{Kind: sqliteproto.KindString, String: stmt.ColumnText(index)}
			case sqlite3.BLOB:
				return failure(fmt.Errorf("column %q holds a BLOB value; BLOB results are not supported by AhdCode SQLite v0.3.0", columns[index]))
			default:
				return failure(fmt.Errorf("column %q holds an unsupported SQLite storage class", columns[index]))
			}
		}
		rows = append(rows, row)
	}
	if err := stmt.Err(); err != nil {
		return failure(err)
	}
	return sqliteproto.Response{Columns: columns, Rows: rows}
}

// failure turns a Go error into a response. The driver formats SQLite errors
// as "sqlite3: <result code text>: <message>"; the AhdCode SQLiteError keeps
// SQLite's own message ("no such table: notes", "NOT NULL constraint failed:
// notes.title") and falls back to the result code text ("unable to open
// database file") when SQLite supplied no further message.
func failure(err error) sqliteproto.Response {
	if err == nil {
		return sqliteproto.Response{}
	}
	text := strings.TrimPrefix(err.Error(), "sqlite3: ")
	var driverError *sqlite3.Error
	if errors.As(err, &driverError) && driverError.Unwrap() == nil {
		prefix := strings.TrimPrefix(driverError.Code().Error(), "sqlite3: ") + ": "
		if rest := strings.TrimPrefix(text, prefix); rest != text && strings.TrimSpace(rest) != "" {
			text = rest
		}
	}
	return sqliteproto.Response{Error: text}
}
