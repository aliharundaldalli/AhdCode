package ahdruntime

// The SQLite standard module's runtime client. This file is also emitted
// verbatim into native programs (with only its package clause rewritten), so
// it intentionally depends on the Go standard library and the sibling AhdCode
// runtime only. The SQLite engine itself lives in the bundled ahdsqlite
// helper, which this client drives over a long-lived JSON session, the same
// way the Numeric and Plot modules delegate to ahdnumeric and ahdplot.
//
// Every AhdCode Database is one logical SQLite connection inside that helper,
// addressed by a handle. A SQLiteValue is one storage-class value encoded as
// canonical text: a kind byte ('N', 'I', 'R', or 'S') followed by the payload.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// AhdSQLiteRuntimeHint is the installation directory recorded at build time
// for programs that use SQLite, mirroring AhdNumericRuntimeHint.
var AhdSQLiteRuntimeHint string

// The wire types intentionally mirror internal/sqliteproto. This runtime is
// copied into dependency-free generated workspaces, so it cannot import that
// package directly.
type ahdSQLiteValue struct {
	Kind   string  `json:"kind"`
	Int    int64   `json:"int,omitempty"`
	Real   float64 `json:"real,omitempty"`
	String string  `json:"string,omitempty"`
}

type ahdSQLiteRequest struct {
	Operation  string           `json:"operation"`
	Database   int64            `json:"database,omitempty"`
	Path       string           `json:"path,omitempty"`
	SQL        string           `json:"sql,omitempty"`
	Parameters []ahdSQLiteValue `json:"parameters,omitempty"`
}

type ahdSQLiteResponse struct {
	Error    string             `json:"error,omitempty"`
	Database int64              `json:"database,omitempty"`
	Changed  int64              `json:"changed,omitempty"`
	RowID    int64              `json:"rowId,omitempty"`
	Columns  []string           `json:"columns,omitempty"`
	Rows     [][]ahdSQLiteValue `json:"rows,omitempty"`
}

const (
	ahdSQLiteKindNull   = "Null"
	ahdSQLiteKindInt    = "Int"
	ahdSQLiteKindReal   = "Real"
	ahdSQLiteKindString = "String"
)

// ahdSQLiteHelper is the one helper process of this program. It starts on the
// first SQLite.open and exits when this process closes its input.
type ahdSQLiteHelper struct {
	mutex   sync.Mutex
	command *exec.Cmd
	stdin   io.WriteCloser
	encoder *json.Encoder
	decoder *json.Decoder
	failure error
}

var ahdSQLiteSession ahdSQLiteHelper

func ahdSQLiteDiscoverRuntime() (string, error) {
	name := "ahdsqlite"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var candidates []string
	if custom := os.Getenv("AHDCODE_SQLITE_RUNTIME"); custom != "" {
		candidates = append(candidates, custom, filepath.Join(custom, name))
	}
	if AhdSQLiteRuntimeHint != "" {
		candidates = append(candidates, filepath.Join(AhdSQLiteRuntimeHint, name))
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(bin, name),
			filepath.Join(bin, "..", "libexec", "ahdcode", name),
		)
	}
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", errors.New("the SQLite helper (ahdsqlite) was not found; set AHDCODE_SQLITE_RUNTIME or reinstall AhdCode with the bundled SQLite helper")
}

func (helper *ahdSQLiteHelper) start() error {
	path, err := ahdSQLiteDiscoverRuntime()
	if err != nil {
		return err
	}
	command := exec.Command(path)
	command.Stderr = os.Stderr
	stdin, err := command.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not start the SQLite helper: %v", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not start the SQLite helper: %v", err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("could not start the SQLite helper: %v", err)
	}
	helper.command = command
	helper.stdin = stdin
	helper.encoder = json.NewEncoder(stdin)
	helper.encoder.SetEscapeHTML(false)
	helper.decoder = json.NewDecoder(bufio.NewReader(stdout))
	return nil
}

// call sends one request and waits for its response. A transport failure is
// permanent: the helper owned every open connection, so no later request can
// succeed, and each one reports the same error instead of hanging.
func (helper *ahdSQLiteHelper) call(request ahdSQLiteRequest) (ahdSQLiteResponse, error) {
	helper.mutex.Lock()
	defer helper.mutex.Unlock()
	if helper.failure != nil {
		return ahdSQLiteResponse{}, helper.failure
	}
	if helper.command == nil {
		if err := helper.start(); err != nil {
			return ahdSQLiteResponse{}, err
		}
	}
	if err := helper.encoder.Encode(request); err != nil {
		helper.failure = errors.New("the SQLite helper (ahdsqlite) stopped accepting requests: " + err.Error())
		return ahdSQLiteResponse{}, helper.failure
	}
	var response ahdSQLiteResponse
	if err := helper.decoder.Decode(&response); err != nil {
		helper.failure = errors.New("the SQLite helper (ahdsqlite) exited unexpectedly; every Database it held is lost")
		return ahdSQLiteResponse{}, helper.failure
	}
	if response.Error != "" {
		return ahdSQLiteResponse{}, errors.New(response.Error)
	}
	return response, nil
}

// ---------------------------------------------------------------------------
// SQLiteValue encoding
// ---------------------------------------------------------------------------

// SQLiteNullValue encodes SQL NULL. It is a SQLiteValue of kind Null, never
// an AhdCode null, so query rows stay structurally Pair<String, SQLiteValue>.
func SQLiteNullValue() string { return "N" }

func SQLiteFromInt(value int64) string { return "I" + strconv.FormatInt(value, 10) }

// SQLiteFromReal rejects NaN and infinities: an AhdCode Real is always finite.
func SQLiteFromReal(value float64) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", errors.New("SQLite Real value must be finite")
	}
	return "R" + strconv.FormatFloat(value, 'g', -1, 64), nil
}

func SQLiteFromString(value string) string { return "S" + value }

func sqliteDecodeValue(text string) (ahdSQLiteValue, error) {
	if text == "" {
		return ahdSQLiteValue{}, errors.New("SQLiteValue storage is corrupted")
	}
	payload := text[1:]
	switch text[0] {
	case 'N':
		return ahdSQLiteValue{Kind: ahdSQLiteKindNull}, nil
	case 'I':
		value, err := strconv.ParseInt(payload, 10, 64)
		if err != nil {
			return ahdSQLiteValue{}, errors.New("SQLiteValue storage is corrupted")
		}
		return ahdSQLiteValue{Kind: ahdSQLiteKindInt, Int: value}, nil
	case 'R':
		value, err := strconv.ParseFloat(payload, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return ahdSQLiteValue{}, errors.New("SQLiteValue storage is corrupted")
		}
		return ahdSQLiteValue{Kind: ahdSQLiteKindReal, Real: value}, nil
	case 'S':
		return ahdSQLiteValue{Kind: ahdSQLiteKindString, String: payload}, nil
	}
	return ahdSQLiteValue{}, errors.New("SQLiteValue storage is corrupted")
}

func sqliteEncodeValue(value ahdSQLiteValue) (string, error) {
	switch value.Kind {
	case ahdSQLiteKindNull:
		return SQLiteNullValue(), nil
	case ahdSQLiteKindInt:
		return SQLiteFromInt(value.Int), nil
	case ahdSQLiteKindReal:
		return SQLiteFromReal(value.Real)
	case ahdSQLiteKindString:
		return SQLiteFromString(value.String), nil
	}
	return "", fmt.Errorf("the SQLite helper returned an unsupported value kind %q", value.Kind)
}

func SQLiteValueKind(text string) (string, error) {
	value, err := sqliteDecodeValue(text)
	return value.Kind, err
}

func SQLiteValueIsNull(text string) (bool, error) {
	value, err := sqliteDecodeValue(text)
	return value.Kind == ahdSQLiteKindNull, err
}

// SQLiteValueInt requires kind Int. A String is never parsed and a Real is
// never truncated: the programmer converts explicitly if that is the intent.
func SQLiteValueInt(text string) (int64, error) {
	value, err := sqliteDecodeValue(text)
	if err != nil {
		return 0, err
	}
	if value.Kind != ahdSQLiteKindInt {
		return 0, sqliteWrongKind("int()", ahdSQLiteKindInt, value.Kind)
	}
	return value.Int, nil
}

// SQLiteValueReal accepts kind Real, and kind Int widened exactly the way an
// AhdCode `Real := Int` assignment already widens. Strings are never parsed.
func SQLiteValueReal(text string) (float64, error) {
	value, err := sqliteDecodeValue(text)
	if err != nil {
		return 0, err
	}
	switch value.Kind {
	case ahdSQLiteKindReal:
		return value.Real, nil
	case ahdSQLiteKindInt:
		return float64(value.Int), nil
	}
	return 0, sqliteWrongKind("real()", "Real or Int", value.Kind)
}

// SQLiteValueString requires kind String. Numbers are never stringified.
func SQLiteValueString(text string) (string, error) {
	value, err := sqliteDecodeValue(text)
	if err != nil {
		return "", err
	}
	if value.Kind != ahdSQLiteKindString {
		return "", sqliteWrongKind("string()", ahdSQLiteKindString, value.Kind)
	}
	return value.String, nil
}

func sqliteWrongKind(accessor, expected, actual string) error {
	return fmt.Errorf("%s requires kind %s; this SQLiteValue has kind %s (check kind() first)", accessor, expected, actual)
}

// ---------------------------------------------------------------------------
// Database operations
// ---------------------------------------------------------------------------

func sqliteHandle(handle string) (int64, error) {
	value, err := strconv.ParseInt(handle, 10, 64)
	if err != nil || value <= 0 {
		return 0, errors.New("Database storage is corrupted")
	}
	return value, nil
}

func sqliteParameters(encoded []string) ([]ahdSQLiteValue, error) {
	parameters := make([]ahdSQLiteValue, len(encoded))
	for index, text := range encoded {
		value, err := sqliteDecodeValue(text)
		if err != nil {
			return nil, fmt.Errorf("parameter %d: %v", index+1, err)
		}
		parameters[index] = value
	}
	return parameters, nil
}

// SQLiteOpen opens (creating when absent) the database file at path, or a
// private in-memory database for ":memory:". A relative path is resolved
// against the current working directory once, here, so the helper never
// depends on its own working directory. Parent directories are not created.
func SQLiteOpen(path string) (string, error) {
	if path == "" {
		return "", errors.New("the database path is empty; use \":memory:\" or a file path")
	}
	if path != ":memory:" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("could not resolve the database path %q: %v", path, err)
		}
		path = absolute
	}
	response, err := ahdSQLiteSession.call(ahdSQLiteRequest{Operation: "open", Path: path})
	if err != nil {
		return "", err
	}
	if response.Database <= 0 {
		return "", errors.New("the SQLite helper returned an invalid Database handle")
	}
	return strconv.FormatInt(response.Database, 10), nil
}

func sqliteSimple(operation, handle string) (ahdSQLiteResponse, error) {
	database, err := sqliteHandle(handle)
	if err != nil {
		return ahdSQLiteResponse{}, err
	}
	return ahdSQLiteSession.call(ahdSQLiteRequest{Operation: operation, Database: database})
}

func sqliteStatement(operation, handle, sql string, parameters []string) (ahdSQLiteResponse, error) {
	database, err := sqliteHandle(handle)
	if err != nil {
		return ahdSQLiteResponse{}, err
	}
	values, err := sqliteParameters(parameters)
	if err != nil {
		return ahdSQLiteResponse{}, err
	}
	return ahdSQLiteSession.call(ahdSQLiteRequest{Operation: operation, Database: database, SQL: sql, Parameters: values})
}

// SQLiteExecute runs one statement with real SQLite parameter binding and
// reports how many rows it changed. The SQL text is passed to SQLite
// unchanged; parameters travel separately and are never spliced into it.
func SQLiteExecute(handle, sql string, parameters []string) (int64, error) {
	response, err := sqliteStatement("execute", handle, sql, parameters)
	if err != nil {
		return 0, err
	}
	return response.Changed, nil
}

// SQLiteQuery runs one statement and returns the result column labels in
// result order and every row's values, each encoded as SQLiteValue text.
func SQLiteQuery(handle, sql string, parameters []string) ([]string, [][]string, error) {
	response, err := sqliteStatement("query", handle, sql, parameters)
	if err != nil {
		return nil, nil, err
	}
	columns := response.Columns
	if columns == nil {
		columns = []string{}
	}
	rows := make([][]string, len(response.Rows))
	for rowIndex, row := range response.Rows {
		if len(row) != len(columns) {
			return nil, nil, errors.New("the SQLite helper returned a row that does not match its columns")
		}
		encoded := make([]string, len(row))
		for index, value := range row {
			text, err := sqliteEncodeValue(value)
			if err != nil {
				return nil, nil, err
			}
			encoded[index] = text
		}
		rows[rowIndex] = encoded
	}
	return columns, rows, nil
}

// SQLiteLastInsertID is SQLite's connection-local last_insert_rowid().
func SQLiteLastInsertID(handle string) (int64, error) {
	response, err := sqliteSimple("lastInsertId", handle)
	if err != nil {
		return 0, err
	}
	return response.RowID, nil
}

func SQLiteBegin(handle string) error {
	_, err := sqliteSimple("begin", handle)
	return err
}

func SQLiteCommit(handle string) error {
	_, err := sqliteSimple("commit", handle)
	return err
}

func SQLiteRollback(handle string) error {
	_, err := sqliteSimple("rollback", handle)
	return err
}

// SQLiteClose releases the connection. Closing twice succeeds; closing while
// a transaction is active fails so nothing is ever committed or discarded
// implicitly.
func SQLiteClose(handle string) error {
	_, err := sqliteSimple("close", handle)
	return err
}

// ---------------------------------------------------------------------------
// Generated-program entry points: every failure becomes a catchable
// SQLiteError of the class the generated program passes in.
// ---------------------------------------------------------------------------

func ahdSQLiteRaise(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, strings.TrimSpace(err.Error()))
	}
}

func AhdSQLiteOpen(class *AhdClass, path string) string {
	handle, err := SQLiteOpen(path)
	ahdSQLiteRaise(class, err)
	return handle
}

func AhdSQLiteNullValue() string { return SQLiteNullValue() }

func AhdSQLiteFromInt(value int64) string { return SQLiteFromInt(value) }

func AhdSQLiteFromReal(class *AhdClass, value float64) string {
	text, err := SQLiteFromReal(value)
	ahdSQLiteRaise(class, err)
	return text
}

func AhdSQLiteFromString(value string) string { return SQLiteFromString(value) }

func AhdSQLiteValueKind(class *AhdClass, text string) string {
	kind, err := SQLiteValueKind(text)
	ahdSQLiteRaise(class, err)
	return kind
}

func AhdSQLiteValueIsNull(class *AhdClass, text string) bool {
	isNull, err := SQLiteValueIsNull(text)
	ahdSQLiteRaise(class, err)
	return isNull
}

func AhdSQLiteValueInt(class *AhdClass, text string) int64 {
	value, err := SQLiteValueInt(text)
	ahdSQLiteRaise(class, err)
	return value
}

func AhdSQLiteValueReal(class *AhdClass, text string) float64 {
	value, err := SQLiteValueReal(text)
	ahdSQLiteRaise(class, err)
	return value
}

func AhdSQLiteValueString(class *AhdClass, text string) string {
	value, err := SQLiteValueString(text)
	ahdSQLiteRaise(class, err)
	return value
}

func AhdSQLiteExecute(class *AhdClass, handle, sql string, parameters []string) int64 {
	changed, err := SQLiteExecute(handle, sql, parameters)
	ahdSQLiteRaise(class, err)
	return changed
}

func AhdSQLiteQuery(class *AhdClass, handle, sql string, parameters []string) ([]string, [][]string) {
	columns, rows, err := SQLiteQuery(handle, sql, parameters)
	ahdSQLiteRaise(class, err)
	return columns, rows
}

func AhdSQLiteLastInsertID(class *AhdClass, handle string) int64 {
	value, err := SQLiteLastInsertID(handle)
	ahdSQLiteRaise(class, err)
	return value
}

func AhdSQLiteBegin(class *AhdClass, handle string)    { ahdSQLiteRaise(class, SQLiteBegin(handle)) }
func AhdSQLiteCommit(class *AhdClass, handle string)   { ahdSQLiteRaise(class, SQLiteCommit(handle)) }
func AhdSQLiteRollback(class *AhdClass, handle string) { ahdSQLiteRaise(class, SQLiteRollback(handle)) }
func AhdSQLiteClose(class *AhdClass, handle string)    { ahdSQLiteRaise(class, SQLiteClose(handle)) }
