package ahdruntime

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgreSQLConnectValidation(t *testing.T) {
	for _, testCase := range []struct {
		host, username string
		port           int64
		security       string
		timeout        int64
		want           string
	}{
		{"", "app", 5432, "tls", 10, "PostgreSQL host must not be empty"},
		{"postgres://db", "app", 5432, "tls", 10, "PostgreSQL host must not be a URL"},
		{"/var/run/postgresql", "app", 5432, "tls", 10, "PostgreSQL host is not valid"},
		{"db1,db2", "app", 5432, "tls", 10, "PostgreSQL host is not valid"},
		{" db", "app", 5432, "tls", 10, "PostgreSQL host is not valid"},
		{"db", " ", 5432, "tls", 10, "PostgreSQL username must not be empty"},
		{"db", "app", 0, "tls", 10, "PostgreSQL port must be in 1..65535"},
		{"db", "app", 65536, "tls", 10, "PostgreSQL port must be in 1..65535"},
		{"db", "app", 5432, "prefer", 10, "PostgreSQL security must be tls or none"},
		{"db", "app", 5432, "TLS", 10, "PostgreSQL security must be tls or none"},
		{"db", "app", 5432, "tls", 0, "PostgreSQL timeoutSeconds must be between 1 and 9223372036"},
		{"db", "app", 5432, "tls", ahdPostgreSQLMaxTimeoutSeconds + 1, "PostgreSQL timeoutSeconds must be between 1 and 9223372036"},
		{"db", "app", 5432, "none", ahdPostgreSQLMaxTimeoutSeconds, ""},
		{"[::1]", "app", 5432, "tls", 10, ""},
	} {
		if got := ahdPostgreSQLValidateConnect(testCase.host, testCase.username, testCase.port, testCase.security, testCase.timeout); got != testCase.want {
			t.Fatalf("validate(%q, %q, %d, %q, %d) = %q, want %q", testCase.host, testCase.username, testCase.port, testCase.security, testCase.timeout, got, testCase.want)
		}
	}
}

func TestPostgreSQLValueEncoding(t *testing.T) {
	real, err := PostgreSQLFromReal(1.5)
	if err != nil {
		t.Fatal(err)
	}
	for text, kind := range map[string]string{
		PostgreSQLNullValue(): "Null", PostgreSQLFromBool(true): "Bool", PostgreSQLFromBool(false): "Bool",
		PostgreSQLFromInt(-7): "Int", real: "Real", PostgreSQLFromString("Ayşe"): "String", "YAQI=": "Binary",
	} {
		if got, err := PostgreSQLValueKind(text); err != nil || got != kind {
			t.Fatalf("kind(%q) = %q %v, want %q", text, got, err, kind)
		}
	}
	if value, _ := PostgreSQLValueBool(PostgreSQLFromBool(false)); value {
		t.Fatal("false bool read as true")
	}
	if value, _ := PostgreSQLValueReal(PostgreSQLFromInt(7)); value != 7 {
		t.Fatal("Int does not widen to Real")
	}
	if size, _ := PostgreSQLValueBinarySize("YAQI="); size != 2 {
		t.Fatalf("binary size = %d", size)
	}
	if _, err := PostgreSQLFromReal(math.NaN()); err == nil {
		t.Fatal("NaN accepted")
	}
	for accessor, call := range map[string]func() error{
		"bool() requires kind Bool; this PostgreSQLValue has kind Int (check kind() first)": func() error {
			_, err := PostgreSQLValueBool(PostgreSQLFromInt(1))
			return err
		},
		"int() requires kind Int; this PostgreSQLValue has kind String (check kind() first)": func() error {
			_, err := PostgreSQLValueInt(PostgreSQLFromString("1"))
			return err
		},
		"string() requires kind String; this PostgreSQLValue has kind Bool (check kind() first)": func() error {
			_, err := PostgreSQLValueString(PostgreSQLFromBool(true))
			return err
		},
		"binaryBase64() requires kind Binary; this PostgreSQLValue has kind Null (check kind() first)": func() error {
			_, err := PostgreSQLValueBinaryBase64(PostgreSQLNullValue())
			return err
		},
		"PostgreSQLValue storage is corrupted": func() error {
			_, err := PostgreSQLValueKind("Qnot-a-kind")
			return err
		},
	} {
		if err := call(); err == nil || err.Error() != accessor {
			t.Fatalf("error %v, want %q", err, accessor)
		}
	}
}

func TestPostgreSQLParametersAreRenderedAsText(t *testing.T) {
	real, _ := PostgreSQLFromReal(1e21)
	values, err := ahdPostgreSQLParameterValues([]string{
		PostgreSQLNullValue(), PostgreSQLFromBool(true), PostgreSQLFromInt(-42), real,
		PostgreSQLFromString(""), PostgreSQLFromString("Robert'); DROP TABLE users;--"), "YAQI=",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"", "true", "-42", "1e+21", "", "Robert'); DROP TABLE users;--", `\x0102`}
	if values[0] != nil {
		t.Fatal("NULL was not a nil parameter")
	}
	if values[4] == nil {
		t.Fatal("an empty String became NULL")
	}
	for index := 1; index < len(want); index++ {
		if string(values[index]) != want[index] {
			t.Fatalf("parameter %d = %q, want %q", index+1, values[index], want[index])
		}
	}
	if _, err := ahdPostgreSQLParameterValues([]string{PostgreSQLFromInt(1), ""}); err == nil || err.Error() != "parameter 2: PostgreSQLValue storage is corrupted" {
		t.Fatalf("corrupted parameter error %v", err)
	}
}

func postgresqlDateBytes(year int, month time.Month, day int) []byte {
	days := (time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() - ahdPostgreSQLEpochSeconds) / 86400
	raw := make([]byte, 4)
	binary.BigEndian.PutUint32(raw, uint32(int32(days)))
	return raw
}

func postgresqlTimestampBytes(moment time.Time, micros int64) []byte {
	value := (moment.Unix()-ahdPostgreSQLEpochSeconds)*1000000 + micros
	raw := make([]byte, 8)
	binary.BigEndian.PutUint64(raw, uint64(value))
	return raw
}

func TestPostgreSQLDateAndTimestampFormatting(t *testing.T) {
	dates := map[string][]byte{
		"2000-01-01":    postgresqlDateBytes(2000, 1, 1),
		"1999-12-31":    postgresqlDateBytes(1999, 12, 31),
		"2026-09-14":    postgresqlDateBytes(2026, 9, 14),
		"0001-01-01":    postgresqlDateBytes(1, 1, 1),
		"0001-12-31 BC": postgresqlDateBytes(0, 12, 31),
		"0044-03-15 BC": postgresqlDateBytes(-43, 3, 15),
		"infinity":      {0x7f, 0xff, 0xff, 0xff},
		"-infinity":     {0x80, 0x00, 0x00, 0x00},
	}
	for want, raw := range dates {
		if got, ok := ahdPostgreSQLFormatDate(raw); !ok || got != want {
			t.Fatalf("date = %q %v, want %q", got, ok, want)
		}
	}
	base := time.Date(2026, 1, 15, 7, 30, 0, 0, time.UTC)
	timestamps := []struct {
		raw  []byte
		zone bool
		want string
	}{
		{postgresqlTimestampBytes(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), 0), false, "2000-01-01 00:00:00"},
		{postgresqlTimestampBytes(base, 500000), true, "2026-01-15 07:30:00.5+00"},
		{postgresqlTimestampBytes(base, 123456), false, "2026-01-15 07:30:00.123456"},
		{postgresqlTimestampBytes(base, 120), true, "2026-01-15 07:30:00.00012+00"},
		{postgresqlTimestampBytes(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), -1), false, "1999-12-31 23:59:59.999999"},
		{postgresqlTimestampBytes(time.Date(-43, 3, 15, 12, 0, 0, 0, time.UTC), 0), true, "0044-03-15 12:00:00+00 BC"},
		{postgresqlTimestampBytes(time.Date(294276, 12, 31, 23, 59, 59, 0, time.UTC), 999999), false, "294276-12-31 23:59:59.999999"},
		{[]byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, true, "infinity"},
		{[]byte{0x80, 0, 0, 0, 0, 0, 0, 0}, false, "-infinity"},
	}
	for _, testCase := range timestamps {
		if got, ok := ahdPostgreSQLFormatTimestamp(testCase.raw, testCase.zone); !ok || got != testCase.want {
			t.Fatalf("timestamp = %q %v, want %q", got, ok, testCase.want)
		}
	}
	if _, ok := ahdPostgreSQLFormatDate([]byte{1, 2, 3}); ok {
		t.Fatal("a short date was accepted")
	}
}

func TestPostgreSQLColumnDecoding(t *testing.T) {
	decode := func(kind ahdPostgreSQLColumnKind, raw []byte) (ahdPostgreSQLValue, error) {
		return ahdPostgreSQLDecodeColumn(kind, "c", raw)
	}
	if value, _ := decode(ahdPostgreSQLColumnInt, nil); value.Kind != ahdPostgreSQLKindNull {
		t.Fatal("NULL is not kind Null")
	}
	if value, _ := decode(ahdPostgreSQLColumnBool, []byte("t")); !value.Bool || value.Kind != ahdPostgreSQLKindBool {
		t.Fatal("t is not true")
	}
	if value, _ := decode(ahdPostgreSQLColumnInt, []byte("-9223372036854775808")); value.Int != math.MinInt64 {
		t.Fatal("bigint minimum lost")
	}
	if value, _ := decode(ahdPostgreSQLColumnReal, []byte("0.1")); value.Real != 0.1 {
		t.Fatal("real text lost")
	}
	if value, _ := decode(ahdPostgreSQLColumnText, []byte("19.990")); value.Kind != ahdPostgreSQLKindString || value.String != "19.990" {
		t.Fatal("numeric text changed")
	}
	if value, _ := decode(ahdPostgreSQLColumnBytea, []byte{0, 0xff}); value.Kind != ahdPostgreSQLKindBinary || len(value.Binary) != 2 {
		t.Fatal("bytea lost")
	}
	for _, raw := range []string{"NaN", "Infinity", "-Infinity"} {
		if _, err := decode(ahdPostgreSQLColumnReal, []byte(raw)); err == nil || err.Error() != `PostgreSQL query failed: column "c" holds NaN or an infinity, which an AhdCode Real cannot hold` {
			t.Fatalf("%s error %v", raw, err)
		}
	}
	for kind, raw := range map[ahdPostgreSQLColumnKind][]byte{
		ahdPostgreSQLColumnBool: []byte("yes"), ahdPostgreSQLColumnInt: []byte("1.5"),
		ahdPostgreSQLColumnText: {0xff}, ahdPostgreSQLColumnDate: {1},
	} {
		if _, err := decode(kind, raw); err == nil || err.Error() != `PostgreSQL query failed: column "c" returned a value AhdCode could not read` {
			t.Fatalf("kind %d error %v", kind, err)
		}
	}
}

func TestPostgreSQLErrorMapping(t *testing.T) {
	duplicate := &pgconn.PgError{Severity: "ERROR", Code: "23505",
		Message: `duplicate key value violates unique constraint "users_email_key"`,
		Detail:  "Key (email)=(ayse@example.com) already exists.", Hint: "a hint", Where: "a context"}
	message := ahdPostgreSQLMapError(duplicate, "execute", "pw").Error()
	if message != `PostgreSQL execution failed: (23505) duplicate key value violates unique constraint "users_email_key"` {
		t.Fatalf("server error message %q", message)
	}
	if strings.Contains(message, "ayse@example.com") || strings.Contains(message, "hint") {
		t.Fatalf("DETAIL or HINT leaked: %q", message)
	}
	password := "hunter2-secret"
	cases := map[string]error{
		"PostgreSQL connection timed out":     &pgconn.PgError{Code: "57014", Message: "canceling statement due to user request"},
		"PostgreSQL connection timed out ":    fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
		"PostgreSQL connection failed":        errors.New("failed to connect to `user=app database=`: 127.0.0.1:1 (127.0.0.1): dial error: connect: connection refused " + password),
		"PostgreSQL TLS verification failed":  fmt.Errorf("tls error: %w", x509.UnknownAuthorityError{}),
		"PostgreSQL TLS verification failed ": errors.New("server refused TLS connection"),
		`PostgreSQL connection failed: (28P01) password authentication failed for user "app"`: &pgconn.PgError{Code: "28P01", Message: `password authentication failed for user "app"`},
		"PostgreSQL transaction was rolled back because an earlier statement failed":          fmt.Errorf("commit: %w", pgx.ErrTxCommitRollback),
	}
	for want, err := range cases {
		stage := "connect"
		if strings.Contains(want, "rolled back") {
			stage = "transaction"
		}
		if got := ahdPostgreSQLMapError(err, stage, password).Error(); got != strings.TrimSpace(want) {
			t.Fatalf("mapped %v to %q, want %q", err, got, strings.TrimSpace(want))
		}
	}
	count := ahdPostgreSQLMapError(ahdPostgreSQLCountError{placeholders: 2, parameters: 1}, "query", "").Error()
	if count != "PostgreSQL query failed: the statement has 2 placeholder(s) but 1 parameter(s) were passed" {
		t.Fatalf("count message %q", count)
	}
	leaking := &pgconn.PgError{Code: "22P02", Message: "invalid input syntax for type integer: \"" + password + "\""}
	if got := ahdPostgreSQLMapError(leaking, "query", password).Error(); strings.Contains(got, password) {
		t.Fatalf("password survived: %q", got)
	}
}

// Connection parameters come only from PostgreSQL.connect: PG* environment
// variables, ~/.pgpass, ~/.postgresql, and a service file never change them.
func TestPostgreSQLConfigIgnoresTheEnvironment(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".postgresql"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		".pgpass":                    "*:*:*:*:from-pgpass\n",
		".postgresql/root.crt":       "not a certificate",
		".postgresql/postgresql.crt": "not a certificate",
		".postgresql/postgresql.key": "not a key",
		"service.conf":               "[hostile]\nhost=evil.example\nport=1\nuser=evil\ndbname=evil\nsslmode=verify-full\noptions=-c search_path=evil\ndefault_query_exec_mode=simple_protocol\npool_max_conns=1\n",
	} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	for key, value := range map[string]string{
		"PGHOST": "evil.example", "PGPORT": "1", "PGDATABASE": "evil", "PGUSER": "evil",
		"PGPASSWORD": "evil-password", "PGPASSFILE": filepath.Join(home, ".pgpass"), "PGAPPNAME": "evil",
		"PGCONNECT_TIMEOUT": "not-a-number", "PGSSLMODE": "verify-everything", "PGSSLKEY": "/nonexistent",
		"PGSSLCERT": "/nonexistent", "PGSSLROOTCERT": "/nonexistent", "PGSSLSNI": "2",
		"PGSSLNEGOTIATION": "sideways", "PGSSLPASSWORD": "x", "PGTARGETSESSIONATTRS": "sometimes",
		"PGTZ": "Mars/Base", "PGOPTIONS": "-c search_path=evil", "PGMINPROTOCOLVERSION": "9.9",
		"PGMAXPROTOCOLVERSION": "0.1", "PGCHANNELBINDING": "always", "PGREQUIREAUTH": "magic",
		"PGSERVICE": "hostile", "PGSERVICEFILE": filepath.Join(home, "service.conf"),
	} {
		t.Setenv(key, value)
	}
	database := "attendance"
	config, err := ahdPostgreSQLConfig("db.internal", "app", "", 6543, &database, "none", 7)
	if err != nil {
		t.Fatalf("hostile environment broke configuration: %v", err)
	}
	connection := config.ConnConfig
	if connection.Host != "db.internal" || connection.Port != 6543 || connection.User != "app" ||
		connection.Password != "" || connection.Database != "attendance" {
		t.Fatalf("connection target = %s:%d user=%q password=%q database=%q", connection.Host, connection.Port, connection.User, connection.Password, connection.Database)
	}
	if connection.TLSConfig != nil || len(connection.Fallbacks) != 0 {
		t.Fatalf("security none has TLS %v and fallbacks %v", connection.TLSConfig, connection.Fallbacks)
	}
	if len(connection.RuntimeParams) != 1 || connection.RuntimeParams["client_encoding"] != "UTF8" {
		t.Fatalf("runtime params = %v", connection.RuntimeParams)
	}
	if connection.ConnectTimeout != 7*time.Second || connection.DefaultQueryExecMode != pgx.QueryExecModeDescribeExec ||
		connection.KerberosSrvName != "" || connection.ValidateConnect != nil || connection.MinProtocolVersion != "3.0" {
		t.Fatalf("connection settings were influenced: %+v", connection)
	}
	if config.MaxConns < 4 {
		t.Fatalf("pool size taken from a service file: %d", config.MaxConns)
	}

	config, err = ahdPostgreSQLConfig("[::1]", "app", "pw", 5432, nil, "tls", 5)
	if err != nil {
		t.Fatal(err)
	}
	connection = config.ConnConfig
	if connection.Host != "::1" || connection.Database != "" {
		t.Fatalf("host %q database %q", connection.Host, connection.Database)
	}
	if connection.TLSConfig == nil || connection.TLSConfig.ServerName != "::1" || connection.TLSConfig.InsecureSkipVerify ||
		connection.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("tls config %+v", connection.TLSConfig)
	}
}

// A PGSERVICE naming a service that cannot be read is the one input pgx's
// parser cannot ignore; it fails closed with a message that says so.
func TestPostgreSQLConfigFailsClosedOnAnUnreadableService(t *testing.T) {
	t.Setenv("PGSERVICE", "missing")
	t.Setenv("PGSERVICEFILE", filepath.Join(t.TempDir(), "absent.conf"))
	_, err := ahdPostgreSQLConfig("db", "app", "pw", 5432, nil, "none", 5)
	if err == nil || err.Error() != "PostgreSQL connection failed: PGSERVICE names a service that cannot be read; AhdCode never uses service files, so unset PGSERVICE" {
		t.Fatalf("error %v", err)
	}
}

func TestPostgreSQLConnectFailureIsCategorized(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	_, err = PostgreSQLConnect("127.0.0.1", "app", "super-secret", int64(port), nil, "none", 2)
	if err == nil || err.Error() != "PostgreSQL connection failed" {
		t.Fatalf("refused connection error %v", err)
	}
	if _, err := PostgreSQLExecute("missing", "SELECT 1", nil); err == nil || err.Error() != "PostgreSQLDatabase storage is corrupted" {
		t.Fatalf("unknown database error %v", err)
	}
	if err := PostgreSQLTransactionCommit("missing"); err == nil || err.Error() != "PostgreSQLTransaction storage is corrupted" {
		t.Fatalf("unknown transaction error %v", err)
	}
}
