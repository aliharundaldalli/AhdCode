package ahdruntime

// The MySQL standard module's runtime client. Unlike SQLite, this needs no
// external helper process: github.com/go-sql-driver/mysql is a pure-Go
// implementation of the MySQL wire protocol, so a MySQLDatabase talks
// directly to the server over TCP from inside this process.
//
// Every MySQLDatabase is one connection pool (database/sql already pools and
// synchronizes concurrent use), addressed by a handle. Every
// MySQLTransaction pins one independent *sql.Tx, so it stays safe under
// concurrent web requests: two goroutines opening their own transactions on
// the same MySQLDatabase never observe each other's uncommitted state. A
// MySQLValue is one storage-class value encoded as canonical text: a kind
// byte ('N', 'I', 'R', 'S', or 'B') followed by the payload, the same
// encoding convention SQLiteValue uses, with 'B' (binary) added and its
// payload base64-encoded so arbitrary bytes stay representable as text.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// ahdMySQLMaxTimeoutSeconds is the largest whole-second timeout that still
// fits in a time.Duration after conversion to nanoseconds, the same bound
// SMTP and HTTP use.
const ahdMySQLMaxTimeoutSeconds = 9223372036

const (
	ahdMySQLKindNull   = "Null"
	ahdMySQLKindInt    = "Int"
	ahdMySQLKindReal   = "Real"
	ahdMySQLKindString = "String"
	ahdMySQLKindBinary = "Binary"
)

type ahdMySQLValue struct {
	Kind   string
	Int    int64
	Real   float64
	String string
	Binary []byte
}

// ahdMySQLExecer is satisfied by both *sql.DB and *sql.Tx, so
// MySQLDatabase and MySQLTransaction share one execute/query implementation.
type ahdMySQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type ahdMySQLDatabaseState struct {
	db       *sql.DB
	password string
	closed   atomic.Bool
}

type ahdMySQLTransactionState struct {
	mu       sync.Mutex
	tx       *sql.Tx
	password string
	done     bool
}

var (
	ahdMySQLDatabases   = map[string]*ahdMySQLDatabaseState{}
	ahdMySQLDatabasesMu sync.Mutex
	ahdMySQLNextDBID    atomic.Int64

	ahdMySQLTransactions   = map[string]*ahdMySQLTransactionState{}
	ahdMySQLTransactionsMu sync.Mutex
	ahdMySQLNextTxID       atomic.Int64
)

// ---------------------------------------------------------------------------
// MySQLValue encoding
// ---------------------------------------------------------------------------

// MySQLNullValue encodes SQL NULL. It is a MySQLValue of kind Null, never an
// AhdCode null, so query rows stay structurally Pair<String, MySQLValue>.
func MySQLNullValue() string { return "N" }

func MySQLFromInt(value int64) string { return "I" + strconv.FormatInt(value, 10) }

// MySQLFromReal rejects NaN and infinities: an AhdCode Real is always finite.
func MySQLFromReal(value float64) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", errors.New("MySQL Real value must be finite")
	}
	return "R" + strconv.FormatFloat(value, 'g', -1, 64), nil
}

func MySQLFromString(value string) string { return "S" + value }

func mysqlEncodeValue(value ahdMySQLValue) (string, error) {
	switch value.Kind {
	case ahdMySQLKindNull:
		return MySQLNullValue(), nil
	case ahdMySQLKindInt:
		return MySQLFromInt(value.Int), nil
	case ahdMySQLKindReal:
		return MySQLFromReal(value.Real)
	case ahdMySQLKindString:
		return MySQLFromString(value.String), nil
	case ahdMySQLKindBinary:
		return "B" + base64.StdEncoding.EncodeToString(value.Binary), nil
	}
	return "", fmt.Errorf("the MySQL runtime produced an unsupported value kind %q", value.Kind)
}

func mysqlDecodeValue(text string) (ahdMySQLValue, error) {
	if text == "" {
		return ahdMySQLValue{}, errors.New("MySQLValue storage is corrupted")
	}
	payload := text[1:]
	switch text[0] {
	case 'N':
		return ahdMySQLValue{Kind: ahdMySQLKindNull}, nil
	case 'I':
		value, err := strconv.ParseInt(payload, 10, 64)
		if err != nil {
			return ahdMySQLValue{}, errors.New("MySQLValue storage is corrupted")
		}
		return ahdMySQLValue{Kind: ahdMySQLKindInt, Int: value}, nil
	case 'R':
		value, err := strconv.ParseFloat(payload, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return ahdMySQLValue{}, errors.New("MySQLValue storage is corrupted")
		}
		return ahdMySQLValue{Kind: ahdMySQLKindReal, Real: value}, nil
	case 'S':
		return ahdMySQLValue{Kind: ahdMySQLKindString, String: payload}, nil
	case 'B':
		raw, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return ahdMySQLValue{}, errors.New("MySQLValue storage is corrupted")
		}
		return ahdMySQLValue{Kind: ahdMySQLKindBinary, Binary: raw}, nil
	}
	return ahdMySQLValue{}, errors.New("MySQLValue storage is corrupted")
}

func MySQLValueKind(text string) (string, error) {
	value, err := mysqlDecodeValue(text)
	return value.Kind, err
}

func MySQLValueIsNull(text string) (bool, error) {
	value, err := mysqlDecodeValue(text)
	return value.Kind == ahdMySQLKindNull, err
}

// MySQLValueInt requires kind Int. A String is never parsed and a Real is
// never truncated: the programmer converts explicitly if that is the intent.
func MySQLValueInt(text string) (int64, error) {
	value, err := mysqlDecodeValue(text)
	if err != nil {
		return 0, err
	}
	if value.Kind != ahdMySQLKindInt {
		return 0, mysqlWrongKind("int()", ahdMySQLKindInt, value.Kind)
	}
	return value.Int, nil
}

// MySQLValueReal accepts kind Real, and kind Int widened exactly the way an
// AhdCode `Real := Int` assignment already widens. Strings (including exact
// DECIMAL text) are never parsed: the programmer converts explicitly.
func MySQLValueReal(text string) (float64, error) {
	value, err := mysqlDecodeValue(text)
	if err != nil {
		return 0, err
	}
	switch value.Kind {
	case ahdMySQLKindReal:
		return value.Real, nil
	case ahdMySQLKindInt:
		return float64(value.Int), nil
	}
	return 0, mysqlWrongKind("real()", "Real or Int", value.Kind)
}

// MySQLValueString requires kind String. Numbers are never stringified.
func MySQLValueString(text string) (string, error) {
	value, err := mysqlDecodeValue(text)
	if err != nil {
		return "", err
	}
	if value.Kind != ahdMySQLKindString {
		return "", mysqlWrongKind("string()", ahdMySQLKindString, value.Kind)
	}
	return value.String, nil
}

func MySQLValueIsBinary(text string) (bool, error) {
	value, err := mysqlDecodeValue(text)
	return value.Kind == ahdMySQLKindBinary, err
}

func MySQLValueBinarySize(text string) (int64, error) {
	value, err := mysqlDecodeValue(text)
	if err != nil {
		return 0, err
	}
	if value.Kind != ahdMySQLKindBinary {
		return 0, mysqlWrongKind("binarySize()", ahdMySQLKindBinary, value.Kind)
	}
	return int64(len(value.Binary)), nil
}

func MySQLValueBinaryBase64(text string) (string, error) {
	value, err := mysqlDecodeValue(text)
	if err != nil {
		return "", err
	}
	if value.Kind != ahdMySQLKindBinary {
		return "", mysqlWrongKind("binaryBase64()", ahdMySQLKindBinary, value.Kind)
	}
	return base64.StdEncoding.EncodeToString(value.Binary), nil
}

func mysqlWrongKind(accessor, expected, actual string) error {
	return fmt.Errorf("%s requires kind %s; this MySQLValue has kind %s (check kind() first)", accessor, expected, actual)
}

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

func ahdMySQLHostError(host string) string {
	if host == "" {
		return "MySQL host must not be empty"
	}
	if strings.Contains(host, "://") {
		return "MySQL host must not be a URL"
	}
	if strings.TrimSpace(host) != host {
		return "MySQL host is not valid"
	}
	for _, r := range host {
		if r < 32 || r == 127 || r == '/' || unicode.IsSpace(r) {
			return "MySQL host is not valid"
		}
	}
	return ""
}

func ahdMySQLValidateConnect(host, username string, port int64, security string, timeoutSeconds int64) string {
	if err := ahdMySQLHostError(host); err != "" {
		return err
	}
	if strings.TrimSpace(username) == "" {
		return "MySQL username must not be empty"
	}
	if port < 1 || port > 65535 {
		return "MySQL port must be in 1..65535"
	}
	switch security {
	case "tls", "none":
	default:
		return "MySQL security must be tls or none"
	}
	if timeoutSeconds < 1 || timeoutSeconds > ahdMySQLMaxTimeoutSeconds {
		return "MySQL timeoutSeconds must be between 1 and 9223372036"
	}
	return ""
}

// MySQLConnect opens a connection pool and verifies it is actually usable
// with a bounded ping before returning. database == nil (or, canonicalized
// the same way, an empty/whitespace-only String) connects without selecting
// a default database, so a caller can run SHOW DATABASES immediately.
func MySQLConnect(host, username, password string, port int64, database *string, security string, timeoutSeconds int64) (string, error) {
	if message := ahdMySQLValidateConnect(host, username, port, security, timeoutSeconds); message != "" {
		return "", errors.New(message)
	}

	cfg := mysqldriver.NewConfig()
	cfg.User = username
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, strconv.FormatInt(port, 10))
	if database != nil && strings.TrimSpace(*database) != "" {
		cfg.DBName = *database
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	cfg.Timeout = timeout
	cfg.ReadTimeout = timeout
	cfg.WriteTimeout = timeout
	if security == "tls" {
		tlsConfig, err := ahdMySQLTLSConfig(host)
		if err != nil {
			return "", err
		}
		cfg.TLS = tlsConfig
	}

	connector, err := mysqldriver.NewConnector(cfg)
	if err != nil {
		return "", ahdMySQLMapError(err, "connect", password)
	}
	db := sql.OpenDB(connector)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return "", ahdMySQLMapError(err, "connect", password)
	}

	id := strconv.FormatInt(ahdMySQLNextDBID.Add(1), 10)
	ahdMySQLDatabasesMu.Lock()
	ahdMySQLDatabases[id] = &ahdMySQLDatabaseState{db: db, password: password}
	ahdMySQLDatabasesMu.Unlock()
	return id, nil
}

// ahdMySQLTLSConfig mirrors SMTP's TLS configuration: system trust roots,
// hostname verification, TLS 1.2 minimum, and no public insecure-skip. An
// operator may extend trust the same standard way any TLS client on the
// platform can, via SSL_CERT_FILE -- that is honoring an existing system
// convention, not an AhdCode-specific bypass.
func ahdMySQLTLSConfig(host string) (*tls.Config, error) {
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if path := strings.TrimSpace(os.Getenv("SSL_CERT_FILE")); path != "" {
		pem, readErr := os.ReadFile(path)
		if readErr != nil || !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("MySQL TLS verification failed")
		}
	}
	serverName := host
	if strings.HasPrefix(serverName, "[") && strings.HasSuffix(serverName, "]") {
		serverName = strings.TrimSuffix(strings.TrimPrefix(serverName, "["), "]")
	}
	return &tls.Config{RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS12}, nil
}

func ahdMySQLLookupDatabase(handle string) (*ahdMySQLDatabaseState, error) {
	ahdMySQLDatabasesMu.Lock()
	state, ok := ahdMySQLDatabases[handle]
	ahdMySQLDatabasesMu.Unlock()
	if !ok {
		return nil, errors.New("MySQLDatabase storage is corrupted")
	}
	if state.closed.Load() {
		return nil, errors.New("this MySQLDatabase is closed")
	}
	return state, nil
}

func MySQLPing(handle string) error {
	state, err := ahdMySQLLookupDatabase(handle)
	if err != nil {
		return err
	}
	if err := state.db.PingContext(context.Background()); err != nil {
		return ahdMySQLMapError(err, "connect", state.password)
	}
	return nil
}

// MySQLClose releases pool resources. Closing twice safely no-ops, matching
// SQLite's Database.close convention; every operation after close raises.
func MySQLClose(handle string) error {
	ahdMySQLDatabasesMu.Lock()
	state, ok := ahdMySQLDatabases[handle]
	ahdMySQLDatabasesMu.Unlock()
	if !ok {
		return errors.New("MySQLDatabase storage is corrupted")
	}
	if state.closed.CompareAndSwap(false, true) {
		_ = state.db.Close()
	}
	return nil
}

// MySQLBegin opens one independent *sql.Tx pinned to its own handle, so a
// MySQLTransaction never shares mutable state with its MySQLDatabase or with
// any other transaction, including under concurrent web requests.
func MySQLBegin(handle string) (string, error) {
	state, err := ahdMySQLLookupDatabase(handle)
	if err != nil {
		return "", err
	}
	tx, err := state.db.BeginTx(context.Background(), nil)
	if err != nil {
		return "", ahdMySQLMapError(err, "transaction", state.password)
	}
	id := strconv.FormatInt(ahdMySQLNextTxID.Add(1), 10)
	ahdMySQLTransactionsMu.Lock()
	ahdMySQLTransactions[id] = &ahdMySQLTransactionState{tx: tx, password: state.password}
	ahdMySQLTransactionsMu.Unlock()
	return id, nil
}

func ahdMySQLLookupTransaction(handle string) (*ahdMySQLTransactionState, error) {
	ahdMySQLTransactionsMu.Lock()
	state, ok := ahdMySQLTransactions[handle]
	ahdMySQLTransactionsMu.Unlock()
	if !ok {
		return nil, errors.New("MySQLTransaction storage is corrupted")
	}
	return state, nil
}

func ahdMySQLRequireOpenTransaction(state *ahdMySQLTransactionState) error {
	state.mu.Lock()
	done := state.done
	state.mu.Unlock()
	if done {
		return errors.New("this MySQLTransaction is already committed or rolled back")
	}
	return nil
}

// ahdMySQLCloseTransaction atomically checks-and-marks the transaction done,
// so two concurrent commit/rollback calls can never both proceed to close
// the same underlying *sql.Tx.
func ahdMySQLCloseTransaction(state *ahdMySQLTransactionState) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.done {
		return errors.New("this MySQLTransaction is already committed or rolled back")
	}
	state.done = true
	return nil
}

func MySQLTransactionCommit(handle string) error {
	state, err := ahdMySQLLookupTransaction(handle)
	if err != nil {
		return err
	}
	if err := ahdMySQLCloseTransaction(state); err != nil {
		return err
	}
	if err := state.tx.Commit(); err != nil {
		return ahdMySQLMapError(err, "transaction", state.password)
	}
	return nil
}

func MySQLTransactionRollback(handle string) error {
	state, err := ahdMySQLLookupTransaction(handle)
	if err != nil {
		return err
	}
	if err := ahdMySQLCloseTransaction(state); err != nil {
		return err
	}
	if err := state.tx.Rollback(); err != nil {
		return ahdMySQLMapError(err, "transaction", state.password)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Execute / query, shared between MySQLDatabase and MySQLTransaction
// ---------------------------------------------------------------------------

func mysqlParameters(encoded []string) ([]any, error) {
	args := make([]any, len(encoded))
	for index, text := range encoded {
		value, err := mysqlDecodeValue(text)
		if err != nil {
			return nil, fmt.Errorf("parameter %d: %v", index+1, err)
		}
		switch value.Kind {
		case ahdMySQLKindNull:
			args[index] = nil
		case ahdMySQLKindInt:
			args[index] = value.Int
		case ahdMySQLKindReal:
			args[index] = value.Real
		case ahdMySQLKindString:
			args[index] = value.String
		case ahdMySQLKindBinary:
			args[index] = value.Binary
		}
	}
	return args, nil
}

// mysqlExecuteOn runs one statement through execer (a MySQLDatabase's pool or
// one MySQLTransaction's pinned *sql.Tx) and encodes the result as
// "<affectedRows>|<lastInsertId or empty>". 0 is never a real AUTO_INCREMENT
// id, so an empty id segment (lastInsertId absent) and 0 are both read back
// as "no generated id" -- see MySQLResultLastInsertID.
func mysqlExecuteOn(execer ahdMySQLExecer, sqlText string, parameters []string, password string) (string, error) {
	args, err := mysqlParameters(parameters)
	if err != nil {
		return "", err
	}
	result, err := execer.ExecContext(context.Background(), sqlText, args...)
	if err != nil {
		return "", ahdMySQLMapError(err, "execute", password)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		affected = 0
	}
	lastID, idErr := result.LastInsertId()
	idText := ""
	if idErr == nil && lastID != 0 {
		idText = strconv.FormatInt(lastID, 10)
	}
	return strconv.FormatInt(affected, 10) + "|" + idText, nil
}

// mysqlQueryOn runs one statement and returns the result column labels in
// result order and every row's values, each encoded as MySQLValue text.
func mysqlQueryOn(execer ahdMySQLExecer, sqlText string, parameters []string, password string) ([]string, [][]string, error) {
	args, err := mysqlParameters(parameters)
	if err != nil {
		return nil, nil, err
	}
	rows, err := execer.QueryContext(context.Background(), sqlText, args...)
	if err != nil {
		return nil, nil, ahdMySQLMapError(err, "query", password)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, ahdMySQLMapError(err, "query", password)
	}
	if err := ahdMySQLCheckDuplicateColumns(columns); err != nil {
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, ahdMySQLMapError(err, "query", password)
	}

	var encodedRows [][]string
	raw := make([]sql.RawBytes, len(columns))
	dest := make([]any, len(columns))
	for index := range dest {
		dest[index] = &raw[index]
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return nil, nil, ahdMySQLMapError(err, "query", password)
		}
		encoded := make([]string, len(columns))
		for index, bytesValue := range raw {
			value := ahdMySQLClassifyValue(bytesValue, columnTypes[index].DatabaseTypeName())
			text, err := mysqlEncodeValue(value)
			if err != nil {
				return nil, nil, err
			}
			encoded[index] = text
		}
		encodedRows = append(encodedRows, encoded)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, ahdMySQLMapError(err, "query", password)
	}
	if encodedRows == nil {
		encodedRows = [][]string{}
	}
	return columns, encodedRows, nil
}

func ahdMySQLCheckDuplicateColumns(columns []string) error {
	seen := make(map[string]bool, len(columns))
	for _, name := range columns {
		if seen[name] {
			return fmt.Errorf("query result has duplicate column %q; alias it with AS", name)
		}
		seen[name] = true
	}
	return nil
}

// ahdMySQLClassifyValue maps one raw wire value to a MySQLValue kind using
// the column's server-declared type, never by guessing from content. NULL is
// nil regardless of declared type. DECIMAL stays String so its precision is
// never silently narrowed into a Real. A signed integer too large for AhdCode
// Int (BIGINT UNSIGNED near its ceiling) falls back to its exact String text
// rather than overflowing.
func ahdMySQLClassifyValue(raw sql.RawBytes, typeName string) ahdMySQLValue {
	if raw == nil {
		return ahdMySQLValue{Kind: ahdMySQLKindNull}
	}
	data := append([]byte(nil), raw...)
	switch typeName {
	case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "BIGINT", "YEAR",
		"UNSIGNED TINYINT", "UNSIGNED SMALLINT", "UNSIGNED MEDIUMINT", "UNSIGNED INT", "UNSIGNED BIGINT":
		if value, err := strconv.ParseInt(string(data), 10, 64); err == nil {
			return ahdMySQLValue{Kind: ahdMySQLKindInt, Int: value}
		}
		return ahdMySQLValue{Kind: ahdMySQLKindString, String: string(data)}
	case "FLOAT", "DOUBLE":
		if value, err := strconv.ParseFloat(string(data), 64); err == nil {
			return ahdMySQLValue{Kind: ahdMySQLKindReal, Real: value}
		}
		return ahdMySQLValue{Kind: ahdMySQLKindString, String: string(data)}
	case "DECIMAL":
		return ahdMySQLValue{Kind: ahdMySQLKindString, String: string(data)}
	case "DATE", "TIME", "DATETIME", "TIMESTAMP":
		return ahdMySQLValue{Kind: ahdMySQLKindString, String: string(data)}
	case "JSON":
		return ahdMySQLValue{Kind: ahdMySQLKindString, String: string(data)}
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "BINARY", "VARBINARY", "BIT", "GEOMETRY":
		return ahdMySQLValue{Kind: ahdMySQLKindBinary, Binary: data}
	default:
		// CHAR, VARCHAR, TEXT, TINYTEXT, MEDIUMTEXT, LONGTEXT, ENUM, SET, and
		// anything not explicitly classified above stay String -- the
		// conservative, never-corrupting default.
		return ahdMySQLValue{Kind: ahdMySQLKindString, String: string(data)}
	}
}

// ---------------------------------------------------------------------------
// MySQLDatabase / MySQLTransaction execute and query
// ---------------------------------------------------------------------------

func MySQLExecute(handle, sqlText string, parameters []string) (string, error) {
	state, err := ahdMySQLLookupDatabase(handle)
	if err != nil {
		return "", err
	}
	return mysqlExecuteOn(state.db, sqlText, parameters, state.password)
}

func MySQLQuery(handle, sqlText string, parameters []string) ([]string, [][]string, error) {
	state, err := ahdMySQLLookupDatabase(handle)
	if err != nil {
		return nil, nil, err
	}
	return mysqlQueryOn(state.db, sqlText, parameters, state.password)
}

func MySQLTransactionExecute(handle, sqlText string, parameters []string) (string, error) {
	state, err := ahdMySQLLookupTransaction(handle)
	if err != nil {
		return "", err
	}
	if err := ahdMySQLRequireOpenTransaction(state); err != nil {
		return "", err
	}
	return mysqlExecuteOn(state.tx, sqlText, parameters, state.password)
}

func MySQLTransactionQuery(handle, sqlText string, parameters []string) ([]string, [][]string, error) {
	state, err := ahdMySQLLookupTransaction(handle)
	if err != nil {
		return nil, nil, err
	}
	if err := ahdMySQLRequireOpenTransaction(state); err != nil {
		return nil, nil, err
	}
	return mysqlQueryOn(state.tx, sqlText, parameters, state.password)
}

// MySQLResultAffectedRows and MySQLResultLastInsertID decode the
// "<affectedRows>|<lastInsertId or empty>" encoding mysqlExecuteOn produces.
func MySQLResultAffectedRows(data string) (int64, error) {
	affected, _, ok := strings.Cut(data, "|")
	if !ok {
		return 0, errors.New("MySQLResult storage is corrupted")
	}
	value, err := strconv.ParseInt(affected, 10, 64)
	if err != nil {
		return 0, errors.New("MySQLResult storage is corrupted")
	}
	return value, nil
}

func MySQLResultLastInsertID(data string) (*int64, error) {
	_, idText, ok := strings.Cut(data, "|")
	if !ok {
		return nil, errors.New("MySQLResult storage is corrupted")
	}
	if idText == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(idText, 10, 64)
	if err != nil {
		return nil, errors.New("MySQLResult storage is corrupted")
	}
	return &value, nil
}

// ---------------------------------------------------------------------------
// Error mapping. Every path is sanitized against the owning connection's
// password, and connection-stage messages never include the driver's raw
// error text, so a leaked address or handshake detail can never carry it
// either.
// ---------------------------------------------------------------------------

func ahdMySQLMapError(err error, stage, password string) error {
	if err == nil {
		return nil
	}
	return errors.New(ahdMySQLSanitize(ahdMySQLStageMessage(err, stage), password))
}

func ahdMySQLStageMessage(err error, stage string) string {
	if ahdMySQLTimedOut(err) {
		return "MySQL connection timed out"
	}
	if ahdMySQLTLSFailed(err) || stage == "tls" {
		return "MySQL TLS verification failed"
	}
	switch stage {
	case "connect":
		return "MySQL connection failed"
	case "query":
		return "MySQL query failed" + ahdMySQLServerDetail(err)
	case "execute":
		return "MySQL execution failed" + ahdMySQLServerDetail(err)
	case "transaction":
		return "MySQL transaction failed" + ahdMySQLServerDetail(err)
	default:
		return "MySQL connection failed"
	}
}

// ahdMySQLServerDetail appends the server's own error code and message for a
// typed *mysql.MySQLError -- text that originates on the server, not from
// this client's connection configuration, so it can never carry a password.
func ahdMySQLServerDetail(err error) string {
	var serverErr *mysqldriver.MySQLError
	if errors.As(err, &serverErr) {
		return fmt.Sprintf(": (%d) %s", serverErr.Number, serverErr.Message)
	}
	return ""
}

func ahdMySQLTimedOut(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") || strings.Contains(text, "deadline exceeded") || strings.Contains(text, "i/o timeout")
}

func ahdMySQLTLSFailed(err error) bool {
	if err == nil {
		return false
	}
	var unknown x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	if errors.As(err, &unknown) || errors.As(err, &hostname) || errors.As(err, &invalid) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "certificate") || strings.Contains(text, "tls:") || strings.Contains(text, "x509:")
}

func ahdMySQLSanitize(message, password string) string {
	if password != "" {
		message = strings.ReplaceAll(message, password, "")
	}
	return message
}

// ---------------------------------------------------------------------------
// Generated-program entry points: every failure becomes a catchable
// MySQLError of the class the generated program passes in.
// ---------------------------------------------------------------------------

func ahdMySQLRaise(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, strings.TrimSpace(err.Error()))
	}
}

func AhdMySQLConnect(class *AhdClass, host, username, password string, port int64, database *string, security string, timeoutSeconds int64) string {
	handle, err := MySQLConnect(host, username, password, port, database, security, timeoutSeconds)
	ahdMySQLRaise(class, err)
	return handle
}

func AhdMySQLNullValue() string { return MySQLNullValue() }

func AhdMySQLFromInt(value int64) string { return MySQLFromInt(value) }

func AhdMySQLFromReal(class *AhdClass, value float64) string {
	text, err := MySQLFromReal(value)
	ahdMySQLRaise(class, err)
	return text
}

func AhdMySQLFromString(value string) string { return MySQLFromString(value) }

func AhdMySQLValueKind(class *AhdClass, text string) string {
	kind, err := MySQLValueKind(text)
	ahdMySQLRaise(class, err)
	return kind
}

func AhdMySQLValueIsNull(class *AhdClass, text string) bool {
	isNull, err := MySQLValueIsNull(text)
	ahdMySQLRaise(class, err)
	return isNull
}

func AhdMySQLValueInt(class *AhdClass, text string) int64 {
	value, err := MySQLValueInt(text)
	ahdMySQLRaise(class, err)
	return value
}

func AhdMySQLValueReal(class *AhdClass, text string) float64 {
	value, err := MySQLValueReal(text)
	ahdMySQLRaise(class, err)
	return value
}

func AhdMySQLValueString(class *AhdClass, text string) string {
	value, err := MySQLValueString(text)
	ahdMySQLRaise(class, err)
	return value
}

func AhdMySQLValueIsBinary(class *AhdClass, text string) bool {
	value, err := MySQLValueIsBinary(text)
	ahdMySQLRaise(class, err)
	return value
}

func AhdMySQLValueBinarySize(class *AhdClass, text string) int64 {
	value, err := MySQLValueBinarySize(text)
	ahdMySQLRaise(class, err)
	return value
}

func AhdMySQLValueBinaryBase64(class *AhdClass, text string) string {
	value, err := MySQLValueBinaryBase64(text)
	ahdMySQLRaise(class, err)
	return value
}

func AhdMySQLPing(class *AhdClass, handle string) { ahdMySQLRaise(class, MySQLPing(handle)) }

func AhdMySQLExecute(class *AhdClass, handle, sql string, parameters []string) string {
	data, err := MySQLExecute(handle, sql, parameters)
	ahdMySQLRaise(class, err)
	return data
}

func AhdMySQLQuery(class *AhdClass, handle, sql string, parameters []string) ([]string, [][]string) {
	columns, rows, err := MySQLQuery(handle, sql, parameters)
	ahdMySQLRaise(class, err)
	return columns, rows
}

func AhdMySQLBegin(class *AhdClass, handle string) string {
	transaction, err := MySQLBegin(handle)
	ahdMySQLRaise(class, err)
	return transaction
}

func AhdMySQLClose(class *AhdClass, handle string) { ahdMySQLRaise(class, MySQLClose(handle)) }

func AhdMySQLTransactionExecute(class *AhdClass, handle, sql string, parameters []string) string {
	data, err := MySQLTransactionExecute(handle, sql, parameters)
	ahdMySQLRaise(class, err)
	return data
}

func AhdMySQLTransactionQuery(class *AhdClass, handle, sql string, parameters []string) ([]string, [][]string) {
	columns, rows, err := MySQLTransactionQuery(handle, sql, parameters)
	ahdMySQLRaise(class, err)
	return columns, rows
}

func AhdMySQLTransactionCommit(class *AhdClass, handle string) {
	ahdMySQLRaise(class, MySQLTransactionCommit(handle))
}

func AhdMySQLTransactionRollback(class *AhdClass, handle string) {
	ahdMySQLRaise(class, MySQLTransactionRollback(handle))
}

func AhdMySQLResultAffectedRows(class *AhdClass, data string) int64 {
	value, err := MySQLResultAffectedRows(data)
	ahdMySQLRaise(class, err)
	return value
}

func AhdMySQLResultLastInsertID(class *AhdClass, data string) *int64 {
	value, err := MySQLResultLastInsertID(data)
	ahdMySQLRaise(class, err)
	return value
}
