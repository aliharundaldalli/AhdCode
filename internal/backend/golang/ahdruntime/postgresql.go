package ahdruntime

// The PostgreSQL standard module's runtime client (v1.4.0).
// github.com/jackc/pgx/v5 is a pure-Go implementation of the PostgreSQL wire
// protocol, so a PostgreSQLDatabase talks directly to the server from inside
// this process: no libpq, no CGO, no helper process.
//
// A PostgreSQLDatabase is one pgxpool.Pool addressed by a handle; a
// PostgreSQLTransaction pins one pool connection. Every statement runs through
// pgconn directly: the SQL is parsed as the unnamed prepared statement -- which
// PostgreSQL refuses when the text holds more than one statement -- and then
// executed with every parameter sent as text for the server to convert in
// context, and with the result format chosen per column. pgx's own Exec and
// Query are not used, because a call without parameters would switch them to
// the simple protocol, which runs every statement in the text.
//
// A PostgreSQLValue is canonical text: a kind byte ('N' Null, 'T' or 'F' Bool,
// 'I' Int, 'R' Real, 'S' String, 'Y' Binary with a base64 payload) followed by
// the payload -- the convention MySQLValue and SQLiteValue already use.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ahdPostgreSQLMaxTimeoutSeconds is the largest whole-second timeout that still
// fits in a time.Duration, the same bound MySQL, SMTP, and HTTP use.
const ahdPostgreSQLMaxTimeoutSeconds = 9223372036

const (
	ahdPostgreSQLKindNull   = "Null"
	ahdPostgreSQLKindBool   = "Bool"
	ahdPostgreSQLKindInt    = "Int"
	ahdPostgreSQLKindReal   = "Real"
	ahdPostgreSQLKindString = "String"
	ahdPostgreSQLKindBinary = "Binary"
)

type ahdPostgreSQLValue struct {
	Kind   string
	Bool   bool
	Int    int64
	Real   float64
	String string
	Binary []byte
}

type ahdPostgreSQLDatabaseState struct {
	pool         *pgxpool.Pool
	password     string
	timeout      time.Duration
	closed       atomic.Bool
	mu           sync.Mutex
	categories   map[uint32]byte
	transactions map[*ahdPostgreSQLTransactionState]bool
}

// ahdPostgreSQLTransactionState guards its pgx.Tx with a mutex: a pinned
// connection serves one statement at a time.
type ahdPostgreSQLTransactionState struct {
	mu           sync.Mutex
	tx           pgx.Tx
	database     *ahdPostgreSQLDatabaseState
	done         bool
	commitFailed bool
}

var (
	ahdPostgreSQLDatabases   = map[string]*ahdPostgreSQLDatabaseState{}
	ahdPostgreSQLDatabasesMu sync.Mutex
	ahdPostgreSQLNextDBID    atomic.Int64

	ahdPostgreSQLTransactions   = map[string]*ahdPostgreSQLTransactionState{}
	ahdPostgreSQLTransactionsMu sync.Mutex
	ahdPostgreSQLNextTxID       atomic.Int64

	// ahdPostgreSQLTypes identifies the built-in types, array types included.
	// It is only read after construction.
	ahdPostgreSQLTypes = pgtype.NewMap()
)

// ---------------------------------------------------------------------------
// PostgreSQLValue encoding
// ---------------------------------------------------------------------------

func PostgreSQLNullValue() string { return "N" }

func PostgreSQLFromInt(value int64) string { return "I" + strconv.FormatInt(value, 10) }

// PostgreSQLFromReal rejects NaN and infinities: an AhdCode Real is always finite.
func PostgreSQLFromReal(value float64) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", errors.New("PostgreSQL Real value must be finite")
	}
	return "R" + strconv.FormatFloat(value, 'g', -1, 64), nil
}

func PostgreSQLFromString(value string) string { return "S" + value }

func PostgreSQLFromBool(value bool) string {
	if value {
		return "T"
	}
	return "F"
}

func postgresqlEncodeValue(value ahdPostgreSQLValue) string {
	switch value.Kind {
	case ahdPostgreSQLKindBool:
		return PostgreSQLFromBool(value.Bool)
	case ahdPostgreSQLKindInt:
		return PostgreSQLFromInt(value.Int)
	case ahdPostgreSQLKindReal:
		return "R" + strconv.FormatFloat(value.Real, 'g', -1, 64)
	case ahdPostgreSQLKindString:
		return PostgreSQLFromString(value.String)
	case ahdPostgreSQLKindBinary:
		return "Y" + base64.StdEncoding.EncodeToString(value.Binary)
	}
	return PostgreSQLNullValue()
}

func postgresqlDecodeValue(text string) (ahdPostgreSQLValue, error) {
	corrupted := errors.New("PostgreSQLValue storage is corrupted")
	if text == "" {
		return ahdPostgreSQLValue{}, corrupted
	}
	payload := text[1:]
	switch text[0] {
	case 'N':
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindNull}, nil
	case 'T', 'F':
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindBool, Bool: text[0] == 'T'}, nil
	case 'I':
		value, err := strconv.ParseInt(payload, 10, 64)
		if err != nil {
			return ahdPostgreSQLValue{}, corrupted
		}
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindInt, Int: value}, nil
	case 'R':
		value, err := strconv.ParseFloat(payload, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return ahdPostgreSQLValue{}, corrupted
		}
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindReal, Real: value}, nil
	case 'S':
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindString, String: payload}, nil
	case 'Y':
		raw, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return ahdPostgreSQLValue{}, corrupted
		}
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindBinary, Binary: raw}, nil
	}
	return ahdPostgreSQLValue{}, corrupted
}

func postgresqlWrongKind(accessor, expected, actual string) error {
	return fmt.Errorf("%s requires kind %s; this PostgreSQLValue has kind %s (check kind() first)", accessor, expected, actual)
}

func PostgreSQLValueKind(text string) (string, error) {
	value, err := postgresqlDecodeValue(text)
	return value.Kind, err
}

func PostgreSQLValueIsNull(text string) (bool, error) {
	value, err := postgresqlDecodeValue(text)
	return value.Kind == ahdPostgreSQLKindNull, err
}

func PostgreSQLValueBool(text string) (bool, error) {
	value, err := postgresqlDecodeValue(text)
	if err != nil {
		return false, err
	}
	if value.Kind != ahdPostgreSQLKindBool {
		return false, postgresqlWrongKind("bool()", ahdPostgreSQLKindBool, value.Kind)
	}
	return value.Bool, nil
}

// PostgreSQLValueInt requires kind Int. A String is never parsed and a Real is
// never truncated.
func PostgreSQLValueInt(text string) (int64, error) {
	value, err := postgresqlDecodeValue(text)
	if err != nil {
		return 0, err
	}
	if value.Kind != ahdPostgreSQLKindInt {
		return 0, postgresqlWrongKind("int()", ahdPostgreSQLKindInt, value.Kind)
	}
	return value.Int, nil
}

// PostgreSQLValueReal accepts kind Real, and kind Int widened the way an
// AhdCode `Real := Int` assignment widens. NUMERIC text is never parsed.
func PostgreSQLValueReal(text string) (float64, error) {
	value, err := postgresqlDecodeValue(text)
	if err != nil {
		return 0, err
	}
	switch value.Kind {
	case ahdPostgreSQLKindReal:
		return value.Real, nil
	case ahdPostgreSQLKindInt:
		return float64(value.Int), nil
	}
	return 0, postgresqlWrongKind("real()", "Real or Int", value.Kind)
}

func PostgreSQLValueString(text string) (string, error) {
	value, err := postgresqlDecodeValue(text)
	if err != nil {
		return "", err
	}
	if value.Kind != ahdPostgreSQLKindString {
		return "", postgresqlWrongKind("string()", ahdPostgreSQLKindString, value.Kind)
	}
	return value.String, nil
}

func PostgreSQLValueIsBinary(text string) (bool, error) {
	value, err := postgresqlDecodeValue(text)
	return value.Kind == ahdPostgreSQLKindBinary, err
}

func PostgreSQLValueBinarySize(text string) (int64, error) {
	value, err := postgresqlDecodeValue(text)
	if err != nil {
		return 0, err
	}
	if value.Kind != ahdPostgreSQLKindBinary {
		return 0, postgresqlWrongKind("binarySize()", ahdPostgreSQLKindBinary, value.Kind)
	}
	return int64(len(value.Binary)), nil
}

func PostgreSQLValueBinaryBase64(text string) (string, error) {
	value, err := postgresqlDecodeValue(text)
	if err != nil {
		return "", err
	}
	if value.Kind != ahdPostgreSQLKindBinary {
		return "", postgresqlWrongKind("binaryBase64()", ahdPostgreSQLKindBinary, value.Kind)
	}
	return base64.StdEncoding.EncodeToString(value.Binary), nil
}

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

func ahdPostgreSQLHostError(host string) string {
	if host == "" {
		return "PostgreSQL host must not be empty"
	}
	if strings.Contains(host, "://") {
		return "PostgreSQL host must not be a URL"
	}
	if strings.TrimSpace(host) != host || strings.ContainsAny(host, "/,") {
		return "PostgreSQL host is not valid"
	}
	for _, r := range host {
		if r < 32 || r == 127 || unicode.IsSpace(r) {
			return "PostgreSQL host is not valid"
		}
	}
	return ""
}

func ahdPostgreSQLValidateConnect(host, username string, port int64, security string, timeoutSeconds int64) string {
	if message := ahdPostgreSQLHostError(host); message != "" {
		return message
	}
	if strings.TrimSpace(username) == "" {
		return "PostgreSQL username must not be empty"
	}
	if port < 1 || port > 65535 {
		return "PostgreSQL port must be in 1..65535"
	}
	switch security {
	case "tls", "none":
	default:
		return "PostgreSQL security must be tls or none"
	}
	if timeoutSeconds < 1 || timeoutSeconds > ahdPostgreSQLMaxTimeoutSeconds {
		return "PostgreSQL timeoutSeconds must be between 1 and 9223372036"
	}
	return ""
}

// ahdPostgreSQLNeutralSettings gives every setting pgx would otherwise take
// from PG* environment variables, ~/.pgpass, ~/.postgresql, or a service file
// an explicit value, so none of them is read. It carries no secret: the real
// host, user, password, and database are assigned after parsing.
const ahdPostgreSQLNeutralSettings = "host=localhost port=5432 user=ahdcode dbname=ahdcode sslmode=disable " +
	"sslrootcert='' sslcert='' sslkey='' sslpassword='' sslsni='' sslnegotiation='' passfile='' " +
	"connect_timeout=10 target_session_attrs=any min_protocol_version='' max_protocol_version='' " +
	"channel_binding='' require_auth=''"

// ahdPostgreSQLConfig builds the pool configuration from connect's arguments
// alone. pgx requires a configuration produced by its parser, so the neutral
// settings are parsed first and every field that decides where and how to
// connect is then overwritten. A PGSERVICE variable is the one input parsing
// cannot ignore -- pgx looks the service up whenever the key is present -- so
// an unreadable one fails closed with a message that says so.
func ahdPostgreSQLConfig(host, username, password string, port int64, database *string, security string, timeoutSeconds int64) (*pgxpool.Config, error) {
	if message := ahdPostgreSQLValidateConnect(host, username, port, security, timeoutSeconds); message != "" {
		return nil, errors.New(message)
	}
	config, err := pgxpool.ParseConfig(ahdPostgreSQLNeutralSettings)
	if err != nil {
		if os.Getenv("PGSERVICE") != "" {
			return nil, errors.New("PostgreSQL connection failed: PGSERVICE names a service that cannot be read; AhdCode never uses service files, so unset PGSERVICE")
		}
		return nil, errors.New("PostgreSQL connection failed")
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	serverName := strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")

	connection := config.ConnConfig
	connection.Host = serverName
	connection.Port = uint16(port)
	connection.User = username
	connection.Password = password
	connection.Database = ""
	if database != nil && strings.TrimSpace(*database) != "" {
		connection.Database = *database
	}
	connection.TLSConfig = nil
	if security == "tls" {
		tlsConfig, err := ahdPostgreSQLTLSConfig(serverName)
		if err != nil {
			return nil, err
		}
		connection.TLSConfig = tlsConfig
	}
	connection.Fallbacks = nil
	connection.SSLNegotiation = ""
	connection.KerberosSrvName = ""
	connection.KerberosSpn = ""
	connection.ValidateConnect = nil
	connection.AfterConnect = nil
	connection.RuntimeParams = map[string]string{"client_encoding": "UTF8"}
	connection.ConnectTimeout = timeout
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 5 * time.Minute}
	connection.DialFunc = dialer.DialContext
	connection.LookupFunc = net.DefaultResolver.LookupHost
	connection.MinProtocolVersion = "3.0"
	connection.MaxProtocolVersion = "3.0"
	connection.ChannelBinding = "prefer"
	connection.RequireAuth = ""
	// A statement that outlives its timeout is cancelled on the server first;
	// the connection itself is abandoned only if that does not end it.
	connection.BuildContextWatcherHandler = func(pgConn *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.CancelRequestContextWatcherHandler{Conn: pgConn, DeadlineDelay: 2 * time.Second}
	}
	connection.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec
	connection.StatementCacheCapacity = 0
	connection.DescriptionCacheCapacity = 0

	config.MaxConns = int32(max(4, runtime.NumCPU()))
	config.MinConns = 0
	config.MinIdleConns = 0
	config.MaxConnLifetime = time.Hour
	config.MaxConnLifetimeJitter = 0
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute
	return config, nil
}

// ahdPostgreSQLTLSConfig mirrors MySQL's: system trust roots (extended only by
// the standard SSL_CERT_FILE convention), hostname verification, TLS 1.2
// minimum, and no public insecure-skip.
func ahdPostgreSQLTLSConfig(serverName string) (*tls.Config, error) {
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if path := strings.TrimSpace(os.Getenv("SSL_CERT_FILE")); path != "" {
		pem, readErr := os.ReadFile(path)
		if readErr != nil || !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("PostgreSQL TLS verification failed")
		}
	}
	return &tls.Config{RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS12}, nil
}

// PostgreSQLConnect opens a pool and verifies it with a bounded ping before
// returning, so a PostgreSQLDatabase you hold is known to be reachable.
func PostgreSQLConnect(host, username, password string, port int64, database *string, security string, timeoutSeconds int64) (string, error) {
	config, err := ahdPostgreSQLConfig(host, username, password, port, database, security, timeoutSeconds)
	if err != nil {
		return "", err
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return "", ahdPostgreSQLMapError(err, "connect", password)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return "", ahdPostgreSQLMapError(err, "connect", password)
	}
	state := &ahdPostgreSQLDatabaseState{
		pool: pool, password: password, timeout: timeout,
		categories:   map[uint32]byte{},
		transactions: map[*ahdPostgreSQLTransactionState]bool{},
	}
	id := strconv.FormatInt(ahdPostgreSQLNextDBID.Add(1), 10)
	ahdPostgreSQLDatabasesMu.Lock()
	ahdPostgreSQLDatabases[id] = state
	ahdPostgreSQLDatabasesMu.Unlock()
	return id, nil
}

func ahdPostgreSQLLookupDatabase(handle string) (*ahdPostgreSQLDatabaseState, error) {
	ahdPostgreSQLDatabasesMu.Lock()
	state, ok := ahdPostgreSQLDatabases[handle]
	ahdPostgreSQLDatabasesMu.Unlock()
	if !ok {
		return nil, errors.New("PostgreSQLDatabase storage is corrupted")
	}
	if state.closed.Load() {
		return nil, errors.New("this PostgreSQLDatabase is closed")
	}
	return state, nil
}

func PostgreSQLPing(handle string) error {
	state, err := ahdPostgreSQLLookupDatabase(handle)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), state.timeout)
	defer cancel()
	if err := state.pool.Ping(ctx); err != nil {
		return ahdPostgreSQLMapError(err, "connect", state.password)
	}
	return nil
}

// PostgreSQLClose rolls back every transaction still open on the database --
// nothing is committed implicitly -- and then releases the pool. Closing twice
// does nothing further; every operation after close raises.
func PostgreSQLClose(handle string) error {
	ahdPostgreSQLDatabasesMu.Lock()
	state, ok := ahdPostgreSQLDatabases[handle]
	ahdPostgreSQLDatabasesMu.Unlock()
	if !ok {
		return errors.New("PostgreSQLDatabase storage is corrupted")
	}
	if !state.closed.CompareAndSwap(false, true) {
		return nil
	}
	state.mu.Lock()
	open := make([]*ahdPostgreSQLTransactionState, 0, len(state.transactions))
	for transaction := range state.transactions {
		open = append(open, transaction)
	}
	state.transactions = map[*ahdPostgreSQLTransactionState]bool{}
	state.mu.Unlock()
	for _, transaction := range open {
		transaction.mu.Lock()
		if !transaction.done {
			transaction.done = true
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = transaction.tx.Rollback(ctx)
			cancel()
		}
		transaction.mu.Unlock()
	}
	state.pool.Close()
	return nil
}

// ---------------------------------------------------------------------------
// Transactions
// ---------------------------------------------------------------------------

// PostgreSQLBegin opens one transaction pinned to its own pool connection.
func PostgreSQLBegin(handle string) (string, error) {
	state, err := ahdPostgreSQLLookupDatabase(handle)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), state.timeout)
	defer cancel()
	tx, err := state.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return "", ahdPostgreSQLMapError(err, "transaction", state.password)
	}
	transaction := &ahdPostgreSQLTransactionState{tx: tx, database: state}
	state.mu.Lock()
	if state.closed.Load() {
		state.mu.Unlock()
		_ = tx.Rollback(ctx)
		return "", errors.New("this PostgreSQLDatabase is closed")
	}
	state.transactions[transaction] = true
	state.mu.Unlock()
	id := strconv.FormatInt(ahdPostgreSQLNextTxID.Add(1), 10)
	ahdPostgreSQLTransactionsMu.Lock()
	ahdPostgreSQLTransactions[id] = transaction
	ahdPostgreSQLTransactionsMu.Unlock()
	return id, nil
}

func ahdPostgreSQLLookupTransaction(handle string) (*ahdPostgreSQLTransactionState, error) {
	ahdPostgreSQLTransactionsMu.Lock()
	state, ok := ahdPostgreSQLTransactions[handle]
	ahdPostgreSQLTransactionsMu.Unlock()
	if !ok {
		return nil, errors.New("PostgreSQLTransaction storage is corrupted")
	}
	return state, nil
}

func (state *ahdPostgreSQLDatabaseState) forget(transaction *ahdPostgreSQLTransactionState) {
	state.mu.Lock()
	delete(state.transactions, transaction)
	state.mu.Unlock()
}

const ahdPostgreSQLTransactionFinished = "this PostgreSQLTransaction is already committed or rolled back"

// PostgreSQLTransactionCommit is one-shot. A transaction an earlier statement
// aborted cannot commit: PostgreSQL rolls it back instead, and that raises.
func PostgreSQLTransactionCommit(handle string) error {
	transaction, err := ahdPostgreSQLLookupTransaction(handle)
	if err != nil {
		return err
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if transaction.done {
		return errors.New(ahdPostgreSQLTransactionFinished)
	}
	transaction.done = true
	transaction.database.forget(transaction)
	ctx, cancel := context.WithTimeout(context.Background(), transaction.database.timeout)
	defer cancel()
	if err := transaction.tx.Commit(ctx); err != nil {
		transaction.commitFailed = true
		return ahdPostgreSQLMapError(err, "transaction", transaction.database.password)
	}
	return nil
}

// PostgreSQLTransactionRollback is one-shot too, except after a commit that
// failed: that transaction is already over, so the usual
// attempt { ... commit } except { rollback } pattern never raises twice.
func PostgreSQLTransactionRollback(handle string) error {
	transaction, err := ahdPostgreSQLLookupTransaction(handle)
	if err != nil {
		return err
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	if transaction.done {
		if transaction.commitFailed {
			return nil
		}
		return errors.New(ahdPostgreSQLTransactionFinished)
	}
	transaction.done = true
	transaction.database.forget(transaction)
	ctx, cancel := context.WithTimeout(context.Background(), transaction.database.timeout)
	defer cancel()
	if err := transaction.tx.Rollback(ctx); err != nil {
		return ahdPostgreSQLMapError(err, "transaction", transaction.database.password)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

// ahdPostgreSQLCountError reports a parameter list that does not match the
// statement's placeholders. It contains counts only.
type ahdPostgreSQLCountError struct{ placeholders, parameters int }

func (e ahdPostgreSQLCountError) Error() string {
	return fmt.Sprintf("the statement has %d placeholder(s) but %d parameter(s) were passed", e.placeholders, e.parameters)
}

// ahdPostgreSQLParameterValues renders each parameter as the text the server
// converts in the statement's context. A nil entry is SQL NULL.
func ahdPostgreSQLParameterValues(encoded []string) ([][]byte, error) {
	values := make([][]byte, len(encoded))
	for index, text := range encoded {
		value, err := postgresqlDecodeValue(text)
		if err != nil {
			return nil, fmt.Errorf("parameter %d: %v", index+1, err)
		}
		switch value.Kind {
		case ahdPostgreSQLKindNull:
			values[index] = nil
		case ahdPostgreSQLKindBool:
			values[index] = []byte(strconv.FormatBool(value.Bool))
		case ahdPostgreSQLKindInt:
			values[index] = []byte(strconv.FormatInt(value.Int, 10))
		case ahdPostgreSQLKindReal:
			values[index] = []byte(strconv.FormatFloat(value.Real, 'g', -1, 64))
		case ahdPostgreSQLKindString:
			values[index] = append([]byte{}, value.String...)
		case ahdPostgreSQLKindBinary:
			values[index] = []byte(`\x` + hex.EncodeToString(value.Binary))
		}
	}
	return values, nil
}

// ahdPostgreSQLBinaryResult names the result types read in binary form: those
// whose text form depends on DateStyle or bytea_output.
func ahdPostgreSQLBinaryResult(oid uint32) bool {
	switch oid {
	case pgtype.DateOID, pgtype.TimestampOID, pgtype.TimestamptzOID, pgtype.ByteaOID:
		return true
	}
	return false
}

// ahdPostgreSQLRun runs exactly one statement on conn as the unnamed prepared
// statement and reads its whole result.
func ahdPostgreSQLRun(ctx context.Context, conn *pgconn.PgConn, sqlText string, parameters []string) (*pgconn.Result, error) {
	values, err := ahdPostgreSQLParameterValues(parameters)
	if err != nil {
		return nil, err
	}
	description, err := conn.Prepare(ctx, "", sqlText, nil)
	if err != nil {
		return nil, err
	}
	if len(description.ParamOIDs) != len(values) {
		return nil, ahdPostgreSQLCountError{placeholders: len(description.ParamOIDs), parameters: len(values)}
	}
	resultFormats := make([]int16, len(description.Fields))
	for index, field := range description.Fields {
		if ahdPostgreSQLBinaryResult(field.DataTypeOID) {
			resultFormats[index] = 1
		}
	}
	result := conn.ExecPrepared(ctx, "", values, nil, resultFormats).Read()
	if result.Err != nil {
		return nil, result.Err
	}
	return result, nil
}

// onPool runs one statement on a pooled connection and releases it before the
// result is decoded.
func (state *ahdPostgreSQLDatabaseState) onPool(sqlText string, parameters []string) (*pgconn.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), state.timeout)
	defer cancel()
	conn, err := state.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return ahdPostgreSQLRun(ctx, conn.Conn().PgConn(), sqlText, parameters)
}

// onTransaction runs one statement on the transaction's pinned connection.
func (transaction *ahdPostgreSQLTransactionState) onTransaction(sqlText string, parameters []string) (*pgconn.Result, error) {
	if transaction.done {
		return nil, errors.New(ahdPostgreSQLTransactionFinished)
	}
	ctx, cancel := context.WithTimeout(context.Background(), transaction.database.timeout)
	defer cancel()
	return ahdPostgreSQLRun(ctx, transaction.tx.Conn().PgConn(), sqlText, parameters)
}

func ahdPostgreSQLAffected(result *pgconn.Result) string {
	return strconv.FormatInt(result.CommandTag.RowsAffected(), 10)
}

// ahdPostgreSQLColumnKind is how one result column is decoded.
type ahdPostgreSQLColumnKind int

const (
	ahdPostgreSQLColumnText ahdPostgreSQLColumnKind = iota
	ahdPostgreSQLColumnBool
	ahdPostgreSQLColumnInt
	ahdPostgreSQLColumnReal
	ahdPostgreSQLColumnDate
	ahdPostgreSQLColumnTimestamp
	ahdPostgreSQLColumnTimestamptz
	ahdPostgreSQLColumnBytea
)

// columnKind decides a column's decoding from its declared type, never from
// its content. Arrays and composite values are refused; any other type the
// runtime does not name is read as the server's text output.
func (state *ahdPostgreSQLDatabaseState) columnKind(field pgconn.FieldDescription) (ahdPostgreSQLColumnKind, error) {
	switch field.DataTypeOID {
	case pgtype.BoolOID:
		return ahdPostgreSQLColumnBool, nil
	case pgtype.Int2OID, pgtype.Int4OID, pgtype.Int8OID:
		return ahdPostgreSQLColumnInt, nil
	case pgtype.Float4OID, pgtype.Float8OID:
		return ahdPostgreSQLColumnReal, nil
	case pgtype.DateOID:
		return ahdPostgreSQLColumnDate, nil
	case pgtype.TimestampOID:
		return ahdPostgreSQLColumnTimestamp, nil
	case pgtype.TimestamptzOID:
		return ahdPostgreSQLColumnTimestamptz, nil
	case pgtype.ByteaOID:
		return ahdPostgreSQLColumnBytea, nil
	case pgtype.RecordOID, pgtype.RecordArrayOID:
		return 0, ahdPostgreSQLCompositeError(field.Name)
	}
	if dataType, known := ahdPostgreSQLTypes.TypeForOID(field.DataTypeOID); known {
		if _, array := dataType.Codec.(*pgtype.ArrayCodec); array {
			return 0, ahdPostgreSQLArrayError(field.Name)
		}
		return ahdPostgreSQLColumnText, nil
	}
	category, err := state.category(field.DataTypeOID)
	if err != nil {
		return 0, fmt.Errorf("PostgreSQL query failed: could not determine the type of column %q", field.Name)
	}
	switch category {
	case 'A':
		return 0, ahdPostgreSQLArrayError(field.Name)
	case 'C', 'P':
		return 0, ahdPostgreSQLCompositeError(field.Name)
	}
	return ahdPostgreSQLColumnText, nil
}

func ahdPostgreSQLArrayError(column string) error {
	return fmt.Errorf("PostgreSQL query failed: column %q holds a PostgreSQL array, which AhdCode v1.4.0 does not read; convert it in SQL, for example array_to_json(%s)::text", column, column)
}

func ahdPostgreSQLCompositeError(column string) error {
	return fmt.Errorf("PostgreSQL query failed: column %q holds a composite or record value, which AhdCode v1.4.0 does not read; convert it in SQL, for example row_to_json(...)::text", column)
}

// category looks up a user-defined type's pg_type category once per database,
// on its own pooled connection so it can never disturb a transaction.
func (state *ahdPostgreSQLDatabaseState) category(oid uint32) (byte, error) {
	state.mu.Lock()
	category, known := state.categories[oid]
	state.mu.Unlock()
	if known {
		return category, nil
	}
	result, err := state.onPool("SELECT typcategory::text FROM pg_catalog.pg_type WHERE oid = $1", []string{PostgreSQLFromInt(int64(oid))})
	if err != nil {
		return 0, err
	}
	if len(result.Rows) != 1 || len(result.Rows[0]) != 1 || len(result.Rows[0][0]) != 1 {
		return 0, errors.New("unknown type")
	}
	category = result.Rows[0][0][0]
	state.mu.Lock()
	state.categories[oid] = category
	state.mu.Unlock()
	return category, nil
}

func (state *ahdPostgreSQLDatabaseState) decodeResult(result *pgconn.Result) ([]string, [][]string, error) {
	columns := make([]string, len(result.FieldDescriptions))
	kinds := make([]ahdPostgreSQLColumnKind, len(result.FieldDescriptions))
	seen := make(map[string]bool, len(columns))
	for index, field := range result.FieldDescriptions {
		if seen[field.Name] {
			return nil, nil, fmt.Errorf("query result has duplicate column %q; alias it with AS", field.Name)
		}
		seen[field.Name] = true
		columns[index] = field.Name
		kind, err := state.columnKind(field)
		if err != nil {
			return nil, nil, err
		}
		kinds[index] = kind
	}
	rows := make([][]string, len(result.Rows))
	for rowIndex, raw := range result.Rows {
		encoded := make([]string, len(raw))
		for column, bytesValue := range raw {
			value, err := ahdPostgreSQLDecodeColumn(kinds[column], columns[column], bytesValue)
			if err != nil {
				return nil, nil, err
			}
			encoded[column] = postgresqlEncodeValue(value)
		}
		rows[rowIndex] = encoded
	}
	return columns, rows, nil
}

// ahdPostgreSQLDecodeColumn converts one wire value. Text columns arrive in
// PostgreSQL's text format; dates, timestamps, and bytea arrive binary.
func ahdPostgreSQLDecodeColumn(kind ahdPostgreSQLColumnKind, column string, raw []byte) (ahdPostgreSQLValue, error) {
	if raw == nil {
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindNull}, nil
	}
	unexpected := fmt.Errorf("PostgreSQL query failed: column %q returned a value AhdCode could not read", column)
	switch kind {
	case ahdPostgreSQLColumnBool:
		switch string(raw) {
		case "t":
			return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindBool, Bool: true}, nil
		case "f":
			return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindBool}, nil
		}
		return ahdPostgreSQLValue{}, unexpected
	case ahdPostgreSQLColumnInt:
		value, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil {
			return ahdPostgreSQLValue{}, unexpected
		}
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindInt, Int: value}, nil
	case ahdPostgreSQLColumnReal:
		value, err := strconv.ParseFloat(string(raw), 64)
		if err != nil && !math.IsInf(value, 0) {
			return ahdPostgreSQLValue{}, unexpected
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ahdPostgreSQLValue{}, fmt.Errorf("PostgreSQL query failed: column %q holds NaN or an infinity, which an AhdCode Real cannot hold", column)
		}
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindReal, Real: value}, nil
	case ahdPostgreSQLColumnDate:
		text, ok := ahdPostgreSQLFormatDate(raw)
		if !ok {
			return ahdPostgreSQLValue{}, unexpected
		}
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindString, String: text}, nil
	case ahdPostgreSQLColumnTimestamp, ahdPostgreSQLColumnTimestamptz:
		text, ok := ahdPostgreSQLFormatTimestamp(raw, kind == ahdPostgreSQLColumnTimestamptz)
		if !ok {
			return ahdPostgreSQLValue{}, unexpected
		}
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindString, String: text}, nil
	case ahdPostgreSQLColumnBytea:
		return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindBinary, Binary: append([]byte(nil), raw...)}, nil
	}
	if !utf8.Valid(raw) {
		return ahdPostgreSQLValue{}, unexpected
	}
	return ahdPostgreSQLValue{Kind: ahdPostgreSQLKindString, String: string(raw)}, nil
}

// PostgreSQL's binary date and timestamp values count from 2000-01-01.
const ahdPostgreSQLEpochSeconds = 946684800

func ahdPostgreSQLEraYear(year int) (int, bool) {
	if year <= 0 {
		return 1 - year, true
	}
	return year, false
}

// ahdPostgreSQLFormatDate renders a binary date as YYYY-MM-DD, with PostgreSQL's
// own infinity and BC spellings.
func ahdPostgreSQLFormatDate(raw []byte) (string, bool) {
	if len(raw) != 4 {
		return "", false
	}
	days := int32(binary.BigEndian.Uint32(raw))
	switch days {
	case math.MaxInt32:
		return "infinity", true
	case math.MinInt32:
		return "-infinity", true
	}
	civil := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(days))
	year, bc := ahdPostgreSQLEraYear(civil.Year())
	text := fmt.Sprintf("%04d-%02d-%02d", year, int(civil.Month()), civil.Day())
	if bc {
		text += " BC"
	}
	return text, true
}

// ahdPostgreSQLFormatTimestamp renders a binary timestamp as
// YYYY-MM-DD HH:MM:SS[.ffffff], trailing zeros of the fraction removed. A
// timestamptz is an instant and is always rendered in UTC with +00, whatever
// the session TimeZone is.
func ahdPostgreSQLFormatTimestamp(raw []byte, zone bool) (string, bool) {
	if len(raw) != 8 {
		return "", false
	}
	micros := int64(binary.BigEndian.Uint64(raw))
	switch micros {
	case math.MaxInt64:
		return "infinity", true
	case math.MinInt64:
		return "-infinity", true
	}
	seconds, remainder := micros/1000000, micros%1000000
	if remainder < 0 {
		remainder += 1000000
		seconds--
	}
	moment := time.Unix(ahdPostgreSQLEpochSeconds+seconds, remainder*1000).UTC()
	year, bc := ahdPostgreSQLEraYear(moment.Year())
	text := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", year, int(moment.Month()), moment.Day(), moment.Hour(), moment.Minute(), moment.Second())
	if remainder != 0 {
		text += "." + strings.TrimRight(fmt.Sprintf("%06d", remainder), "0")
	}
	if zone {
		text += "+00"
	}
	if bc {
		text += " BC"
	}
	return text, true
}

func PostgreSQLExecute(handle, sqlText string, parameters []string) (string, error) {
	state, err := ahdPostgreSQLLookupDatabase(handle)
	if err != nil {
		return "", err
	}
	result, err := state.onPool(sqlText, parameters)
	if err != nil {
		return "", ahdPostgreSQLMapError(err, "execute", state.password)
	}
	return ahdPostgreSQLAffected(result), nil
}

func PostgreSQLQuery(handle, sqlText string, parameters []string) ([]string, [][]string, error) {
	state, err := ahdPostgreSQLLookupDatabase(handle)
	if err != nil {
		return nil, nil, err
	}
	result, err := state.onPool(sqlText, parameters)
	if err != nil {
		return nil, nil, ahdPostgreSQLMapError(err, "query", state.password)
	}
	return state.decodeResult(result)
}

func PostgreSQLTransactionExecute(handle, sqlText string, parameters []string) (string, error) {
	transaction, err := ahdPostgreSQLLookupTransaction(handle)
	if err != nil {
		return "", err
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	result, err := transaction.onTransaction(sqlText, parameters)
	if err != nil {
		if err.Error() == ahdPostgreSQLTransactionFinished {
			return "", err
		}
		return "", ahdPostgreSQLMapError(err, "execute", transaction.database.password)
	}
	return ahdPostgreSQLAffected(result), nil
}

func PostgreSQLTransactionQuery(handle, sqlText string, parameters []string) ([]string, [][]string, error) {
	transaction, err := ahdPostgreSQLLookupTransaction(handle)
	if err != nil {
		return nil, nil, err
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	result, err := transaction.onTransaction(sqlText, parameters)
	if err != nil {
		if err.Error() == ahdPostgreSQLTransactionFinished {
			return nil, nil, err
		}
		return nil, nil, ahdPostgreSQLMapError(err, "query", transaction.database.password)
	}
	return transaction.database.decodeResult(result)
}

func PostgreSQLResultAffectedRows(data string) (int64, error) {
	value, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, errors.New("PostgreSQLResult storage is corrupted")
	}
	return value, nil
}

// ---------------------------------------------------------------------------
// Error mapping. Server errors contribute their SQLSTATE and primary message
// only -- never DETAIL, HINT, or WHERE, which can echo row values -- and
// connection-stage failures without a server answer never include pgx's raw
// text. Every message is sanitized against the connection's password.
// ---------------------------------------------------------------------------

func ahdPostgreSQLMapError(err error, stage, password string) error {
	if err == nil {
		return nil
	}
	return errors.New(ahdPostgreSQLSanitize(ahdPostgreSQLStageMessage(err, stage), password))
}

func ahdPostgreSQLStageMessage(err error, stage string) string {
	if errors.Is(err, pgx.ErrTxCommitRollback) {
		return "PostgreSQL transaction was rolled back because an earlier statement failed"
	}
	var serverErr *pgconn.PgError
	server := errors.As(err, &serverErr)
	if (server && serverErr.Code == "57014") || (!server && ahdPostgreSQLTimedOut(err)) {
		return "PostgreSQL connection timed out"
	}
	if !server && ahdPostgreSQLTLSFailed(err) {
		return "PostgreSQL TLS verification failed"
	}
	detail := ""
	if server {
		detail = ": (" + serverErr.Code + ") " + serverErr.Message
	}
	var count ahdPostgreSQLCountError
	if errors.As(err, &count) {
		detail = ": " + count.Error()
	}
	switch stage {
	case "connect":
		return "PostgreSQL connection failed" + detail
	case "query":
		return "PostgreSQL query failed" + detail
	case "execute":
		return "PostgreSQL execution failed" + detail
	case "transaction":
		return "PostgreSQL transaction failed" + detail
	}
	return "PostgreSQL connection failed"
}

func ahdPostgreSQLTimedOut(err error) bool {
	if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") || strings.Contains(text, "deadline exceeded")
}

func ahdPostgreSQLTLSFailed(err error) bool {
	var unknown x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	if errors.As(err, &unknown) || errors.As(err, &hostname) || errors.As(err, &invalid) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "certificate") || strings.Contains(text, "tls:") ||
		strings.Contains(text, "x509:") || strings.Contains(text, "refused tls")
}

func ahdPostgreSQLSanitize(message, password string) string {
	if password != "" {
		message = strings.ReplaceAll(message, password, "")
	}
	return message
}

// ---------------------------------------------------------------------------
// Generated-program entry points: every failure becomes a catchable
// PostgreSQLError of the class the generated program passes in.
// ---------------------------------------------------------------------------

func ahdPostgreSQLRaise(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, strings.TrimSpace(err.Error()))
	}
}

func AhdPostgreSQLConnect(class *AhdClass, host, username, password string, port int64, database *string, security string, timeoutSeconds int64) string {
	handle, err := PostgreSQLConnect(host, username, password, port, database, security, timeoutSeconds)
	ahdPostgreSQLRaise(class, err)
	return handle
}

func AhdPostgreSQLNullValue() string { return PostgreSQLNullValue() }

func AhdPostgreSQLFromInt(value int64) string { return PostgreSQLFromInt(value) }

func AhdPostgreSQLFromReal(class *AhdClass, value float64) string {
	text, err := PostgreSQLFromReal(value)
	ahdPostgreSQLRaise(class, err)
	return text
}

func AhdPostgreSQLFromString(value string) string { return PostgreSQLFromString(value) }

func AhdPostgreSQLFromBool(value bool) string { return PostgreSQLFromBool(value) }

func AhdPostgreSQLValueKind(class *AhdClass, text string) string {
	value, err := PostgreSQLValueKind(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueIsNull(class *AhdClass, text string) bool {
	value, err := PostgreSQLValueIsNull(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueBool(class *AhdClass, text string) bool {
	value, err := PostgreSQLValueBool(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueInt(class *AhdClass, text string) int64 {
	value, err := PostgreSQLValueInt(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueReal(class *AhdClass, text string) float64 {
	value, err := PostgreSQLValueReal(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueString(class *AhdClass, text string) string {
	value, err := PostgreSQLValueString(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueIsBinary(class *AhdClass, text string) bool {
	value, err := PostgreSQLValueIsBinary(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueBinarySize(class *AhdClass, text string) int64 {
	value, err := PostgreSQLValueBinarySize(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLValueBinaryBase64(class *AhdClass, text string) string {
	value, err := PostgreSQLValueBinaryBase64(text)
	ahdPostgreSQLRaise(class, err)
	return value
}

func AhdPostgreSQLPing(class *AhdClass, handle string) {
	ahdPostgreSQLRaise(class, PostgreSQLPing(handle))
}

func AhdPostgreSQLExecute(class *AhdClass, handle, sql string, parameters []string) string {
	data, err := PostgreSQLExecute(handle, sql, parameters)
	ahdPostgreSQLRaise(class, err)
	return data
}

func AhdPostgreSQLQuery(class *AhdClass, handle, sql string, parameters []string) ([]string, [][]string) {
	columns, rows, err := PostgreSQLQuery(handle, sql, parameters)
	ahdPostgreSQLRaise(class, err)
	return columns, rows
}

func AhdPostgreSQLBegin(class *AhdClass, handle string) string {
	transaction, err := PostgreSQLBegin(handle)
	ahdPostgreSQLRaise(class, err)
	return transaction
}

func AhdPostgreSQLClose(class *AhdClass, handle string) {
	ahdPostgreSQLRaise(class, PostgreSQLClose(handle))
}

func AhdPostgreSQLTransactionExecute(class *AhdClass, handle, sql string, parameters []string) string {
	data, err := PostgreSQLTransactionExecute(handle, sql, parameters)
	ahdPostgreSQLRaise(class, err)
	return data
}

func AhdPostgreSQLTransactionQuery(class *AhdClass, handle, sql string, parameters []string) ([]string, [][]string) {
	columns, rows, err := PostgreSQLTransactionQuery(handle, sql, parameters)
	ahdPostgreSQLRaise(class, err)
	return columns, rows
}

func AhdPostgreSQLTransactionCommit(class *AhdClass, handle string) {
	ahdPostgreSQLRaise(class, PostgreSQLTransactionCommit(handle))
}

func AhdPostgreSQLTransactionRollback(class *AhdClass, handle string) {
	ahdPostgreSQLRaise(class, PostgreSQLTransactionRollback(handle))
}

func AhdPostgreSQLResultAffectedRows(class *AhdClass, data string) int64 {
	value, err := PostgreSQLResultAffectedRows(data)
	ahdPostgreSQLRaise(class, err)
	return value
}
