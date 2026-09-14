package ahdruntime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The PostgreSQL integration suite runs only against a server named through
// AHDCODE_TEST_POSTGRESQL_HOST, _PORT, _USERNAME, _PASSWORD, _DATABASE, and
// _SECURITY; without a host it skips. Every test works inside its own
// throwaway ahd_test_<random> schema, dropped when the test ends.

type postgresqlTestTarget struct {
	host, username, password, security string
	port                               int64
	database                           *string
}

func postgresqlIntegrationTarget(t *testing.T) postgresqlTestTarget {
	t.Helper()
	target := postgresqlTestTarget{
		host:     os.Getenv("AHDCODE_TEST_POSTGRESQL_HOST"),
		username: os.Getenv("AHDCODE_TEST_POSTGRESQL_USERNAME"),
		password: os.Getenv("AHDCODE_TEST_POSTGRESQL_PASSWORD"),
		security: "tls",
		port:     5432,
	}
	if target.host == "" {
		t.Skip("set AHDCODE_TEST_POSTGRESQL_HOST (with _PORT, _USERNAME, _PASSWORD, _DATABASE, _SECURITY) to run the PostgreSQL integration suite")
	}
	if target.username == "" {
		t.Fatal("AHDCODE_TEST_POSTGRESQL_USERNAME must be set together with AHDCODE_TEST_POSTGRESQL_HOST")
	}
	if text := os.Getenv("AHDCODE_TEST_POSTGRESQL_PORT"); text != "" {
		port, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			t.Fatalf("AHDCODE_TEST_POSTGRESQL_PORT=%q is not a port number", text)
		}
		target.port = port
	}
	if name := os.Getenv("AHDCODE_TEST_POSTGRESQL_DATABASE"); name != "" {
		target.database = &name
	}
	if security := os.Getenv("AHDCODE_TEST_POSTGRESQL_SECURITY"); security != "" {
		target.security = security
	}
	return target
}

func (target postgresqlTestTarget) connect(t *testing.T, timeoutSeconds int64) string {
	t.Helper()
	handle, err := PostgreSQLConnect(target.host, target.username, target.password, target.port, target.database, target.security, timeoutSeconds)
	if err != nil {
		t.Fatalf("connecting to the integration server: %v", err)
	}
	t.Cleanup(func() { _ = PostgreSQLClose(handle) })
	return handle
}

func postgresqlRandomSuffix(t *testing.T) string {
	t.Helper()
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(random)
}

// postgresqlIntegration returns the target, a connection the test owns, and a
// fresh schema. The schema is dropped through a separate connection, so a test
// may close its own handle; cleanups run in reverse, closing that handle (and
// rolling back anything it left open) before the drop.
func postgresqlIntegration(t *testing.T) (postgresqlTestTarget, string, string) {
	t.Helper()
	target := postgresqlIntegrationTarget(t)
	admin, err := PostgreSQLConnect(target.host, target.username, target.password, target.port, target.database, target.security, 10)
	if err != nil {
		t.Fatalf("connecting to the integration server: %v", err)
	}
	schema := "ahd_test_" + postgresqlRandomSuffix(t)
	if _, err := PostgreSQLExecute(admin, "CREATE SCHEMA "+schema, nil); err != nil {
		_ = PostgreSQLClose(admin)
		t.Fatalf("creating %s: %v", schema, err)
	}
	t.Cleanup(func() {
		if _, err := PostgreSQLExecute(admin, "DROP SCHEMA "+schema+" CASCADE", nil); err != nil {
			t.Errorf("dropping %s: %v", schema, err)
		}
		_ = PostgreSQLClose(admin)
	})
	return target, target.connect(t, 10), schema
}

// postgresqlShow renders one encoded PostgreSQLValue as "Kind value".
func postgresqlShow(t *testing.T, encoded string) string {
	t.Helper()
	value, err := postgresqlDecodeValue(encoded)
	if err != nil {
		t.Fatalf("decoding %q: %v", encoded, err)
	}
	switch value.Kind {
	case ahdPostgreSQLKindNull:
		return "Null"
	case ahdPostgreSQLKindBool:
		return "Bool " + strconv.FormatBool(value.Bool)
	case ahdPostgreSQLKindInt:
		return "Int " + strconv.FormatInt(value.Int, 10)
	case ahdPostgreSQLKindReal:
		return "Real " + strconv.FormatFloat(value.Real, 'g', -1, 64)
	case ahdPostgreSQLKindString:
		return "String " + strconv.Quote(value.String)
	case ahdPostgreSQLKindBinary:
		return "Binary " + hex.EncodeToString(value.Binary)
	}
	t.Fatalf("unknown kind in %q", encoded)
	return ""
}

func postgresqlShowRows(t *testing.T, columns []string, rows [][]string) []map[string]string {
	t.Helper()
	shown := make([]map[string]string, len(rows))
	for index, row := range rows {
		shown[index] = make(map[string]string, len(columns))
		for column, encoded := range row {
			shown[index][columns[column]] = postgresqlShow(t, encoded)
		}
	}
	return shown
}

func mustPostgreSQLQuery(t *testing.T, handle, sqlText string, parameters ...string) []map[string]string {
	t.Helper()
	columns, rows, err := PostgreSQLQuery(handle, sqlText, parameters)
	if err != nil {
		t.Fatalf("query %q: %v", sqlText, err)
	}
	return postgresqlShowRows(t, columns, rows)
}

func mustPostgreSQLExecute(t *testing.T, handle, sqlText string, parameters ...string) int64 {
	t.Helper()
	data, err := PostgreSQLExecute(handle, sqlText, parameters)
	if err != nil {
		t.Fatalf("execute %q: %v", sqlText, err)
	}
	affected, err := PostgreSQLResultAffectedRows(data)
	if err != nil {
		t.Fatal(err)
	}
	return affected
}

func mustPostgreSQLTransactionExecute(t *testing.T, transaction, sqlText string, parameters ...string) {
	t.Helper()
	if _, err := PostgreSQLTransactionExecute(transaction, sqlText, parameters); err != nil {
		t.Fatalf("transaction execute %q: %v", sqlText, err)
	}
}

func mustPostgreSQLBegin(t *testing.T, handle string) string {
	t.Helper()
	transaction, err := PostgreSQLBegin(handle)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	return transaction
}

func requirePostgreSQLError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || err.Error() != want {
		t.Fatalf("error %v, want %q", err, want)
	}
}

func requirePostgreSQLRow(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("row has %d columns, want %d: %v", len(got), len(want), got)
	}
	for column, value := range want {
		if got[column] != value {
			t.Fatalf("column %s = %s, want %s", column, got[column], value)
		}
	}
}

func TestPostgreSQLIntegrationResultTypes(t *testing.T) {
	_, handle, schema := postgresqlIntegration(t)
	rows := mustPostgreSQLQuery(t, handle, `SELECT true AS b, 7::smallint AS i2, (-2147483648)::integer AS i4,
		9223372036854775807::bigint AS i8, 1.5::real AS f4, 0.1::double precision AS f8,
		19.990::numeric(10,3) AS amount, 'A0EEBC99-9C0B-4EF8-BB6D-6BB9BD380A11'::uuid AS id,
		'{"a":1}'::jsonb AS doc, '{"a":1}'::json AS raw, DATE '2026-09-14' AS day,
		'0044-03-15 BC'::date AS ides, 'infinity'::date AS forever,
		TIMESTAMP '2026-09-14 07:30:00.120' AS stamp, TIMESTAMPTZ '2026-09-14 10:30:00+03' AS instant,
		'\x00ff'::bytea AS blob, 'Ayşe'::text AS name, 'c'::char(3) AS padded, TIME '07:30' AS clock,
		INTERVAL '1 day 2 hours' AS span, '192.168.0.1/24'::inet AS network, NULL::integer AS missing`)
	if len(rows) != 1 {
		t.Fatalf("got %d rows", len(rows))
	}
	requirePostgreSQLRow(t, rows[0], map[string]string{
		"b": "Bool true", "i2": "Int 7", "i4": "Int -2147483648", "i8": "Int 9223372036854775807",
		"f4": "Real 1.5", "f8": "Real 0.1", "amount": `String "19.990"`,
		"id":  `String "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"`,
		"doc": `String "{\"a\": 1}"`, "raw": `String "{\"a\":1}"`,
		"day": `String "2026-09-14"`, "ides": `String "0044-03-15 BC"`, "forever": `String "infinity"`,
		"stamp": `String "2026-09-14 07:30:00.12"`, "instant": `String "2026-09-14 07:30:00+00"`,
		"blob": "Binary 00ff", "name": `String "Ayşe"`, "padded": `String "c  "`, "clock": `String "07:30:00"`,
		"span": `String "1 day 02:00:00"`, "network": `String "192.168.0.1/24"`, "missing": "Null",
	})

	if empty := mustPostgreSQLQuery(t, handle, "SELECT 1 AS one WHERE false"); len(empty) != 0 {
		t.Fatalf("empty result has rows: %v", empty)
	}

	mustPostgreSQLExecute(t, handle, "CREATE TYPE "+schema+".mood AS ENUM ('calm', 'busy')")
	mustPostgreSQLExecute(t, handle, "CREATE TYPE "+schema+".point2 AS (x integer, y integer)")
	for range 2 { // the second pass reads the cached type categories
		if got := mustPostgreSQLQuery(t, handle, "SELECT 'busy'::"+schema+".mood AS mood")[0]["mood"]; got != `String "busy"` {
			t.Fatalf("enum = %s", got)
		}
		for sqlText, want := range map[string]error{
			"SELECT ARRAY[1, 2] AS numbers":                      ahdPostgreSQLArrayError("numbers"),
			"SELECT ARRAY['calm'::" + schema + ".mood] AS moods": ahdPostgreSQLArrayError("moods"),
			"SELECT ROW(1, 2) AS pair":                           ahdPostgreSQLCompositeError("pair"),
			"SELECT ROW(1, 2)::" + schema + ".point2 AS point":   ahdPostgreSQLCompositeError("point"),
			"SELECT 'NaN'::double precision AS broken":           fmt.Errorf(`PostgreSQL query failed: column "broken" holds NaN or an infinity, which an AhdCode Real cannot hold`),
			"SELECT 'Infinity'::real AS broken":                  fmt.Errorf(`PostgreSQL query failed: column "broken" holds NaN or an infinity, which an AhdCode Real cannot hold`),
			"SELECT 1 AS label, 2 AS label":                      fmt.Errorf(`query result has duplicate column "label"; alias it with AS`),
		} {
			_, _, err := PostgreSQLQuery(handle, sqlText, nil)
			requirePostgreSQLError(t, err, want.Error())
		}
	}

	// Session settings a program changes never change how results read, and
	// AhdCode never changes them itself.
	transaction := mustPostgreSQLBegin(t, handle)
	for _, setting := range []string{"SET LOCAL TimeZone = 'Asia/Tokyo'", "SET LOCAL DateStyle = 'SQL, DMY'", "SET LOCAL bytea_output = 'escape'"} {
		mustPostgreSQLTransactionExecute(t, transaction, setting)
	}
	columns, settled, err := PostgreSQLTransactionQuery(transaction, `SELECT TIMESTAMPTZ '2026-09-14 10:30:00+03' AS instant,
		DATE '2026-09-14' AS day, '\x00ff'::bytea AS blob, current_setting('TimeZone') AS zone,
		current_setting('transaction_isolation') AS isolation`, nil)
	if err != nil {
		t.Fatal(err)
	}
	requirePostgreSQLRow(t, postgresqlShowRows(t, columns, settled)[0], map[string]string{
		"instant": `String "2026-09-14 07:30:00+00"`, "day": `String "2026-09-14"`, "blob": "Binary 00ff",
		"zone": `String "Asia/Tokyo"`, "isolation": `String "read committed"`,
	})
	if err := PostgreSQLTransactionRollback(transaction); err != nil {
		t.Fatal(err)
	}
}

func TestPostgreSQLIntegrationParametersAndStatements(t *testing.T) {
	_, handle, schema := postgresqlIntegration(t)
	table := schema + ".notes"
	mustPostgreSQLExecute(t, handle, "CREATE TABLE "+table+` (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		title text NOT NULL UNIQUE, score numeric(6,2), done boolean NOT NULL DEFAULT false,
		ratio double precision, seen timestamptz)`)

	hostile := "Robert'); DROP TABLE " + table + "; --"
	ratio, err := PostgreSQLFromReal(1e21)
	if err != nil {
		t.Fatal(err)
	}
	if affected := mustPostgreSQLExecute(t, handle,
		"INSERT INTO "+table+" (title, score, done, ratio, seen) VALUES ($1, $2::numeric, $3, $4, $5::timestamptz)",
		PostgreSQLFromString(hostile), PostgreSQLFromString("19.99"), PostgreSQLFromBool(true), ratio,
		PostgreSQLFromString("2026-09-14T10:30:00+03:00")); affected != 1 {
		t.Fatalf("insert affected %d rows", affected)
	}
	stored := mustPostgreSQLQuery(t, handle, "SELECT title, score, done, ratio, seen FROM "+table+" WHERE title = $1", PostgreSQLFromString(hostile))
	if len(stored) != 1 {
		t.Fatalf("stored rows: %v", stored)
	}
	requirePostgreSQLRow(t, stored[0], map[string]string{
		"title": "String " + strconv.Quote(hostile), "score": `String "19.99"`, "done": "Bool true",
		"ratio": "Real 1e+21", "seen": `String "2026-09-14 07:30:00+00"`,
	})

	returning := mustPostgreSQLQuery(t, handle, "INSERT INTO "+table+" (title) VALUES ($1) RETURNING id", PostgreSQLFromString("second"))
	if len(returning) != 1 || returning[0]["id"] != "Int 2" {
		t.Fatalf("RETURNING rows: %v", returning)
	}
	nulls := mustPostgreSQLQuery(t, handle, "SELECT $1::text IS NULL AS absent, $2::text IS NULL AS blank, length($2::text) AS size",
		PostgreSQLNullValue(), PostgreSQLFromString(""))
	requirePostgreSQLRow(t, nulls[0], map[string]string{"absent": "Bool true", "blank": "Bool false", "size": "Int 0"})
	if mark := mustPostgreSQLQuery(t, handle, "SELECT '?' AS mark")[0]["mark"]; mark != `String "?"` {
		t.Fatalf("a question mark was rewritten: %s", mark)
	}
	if affected := mustPostgreSQLExecute(t, handle, "UPDATE "+table+" SET done = $1", PostgreSQLFromBool(false)); affected != 2 {
		t.Fatalf("update affected %d rows", affected)
	}

	failures := map[string]func() error{
		"PostgreSQL query failed: the statement has 2 placeholder(s) but 1 parameter(s) were passed": func() error {
			_, _, err := PostgreSQLQuery(handle, "SELECT $1::int + $2::int AS total", []string{PostgreSQLFromInt(1)})
			return err
		},
		"PostgreSQL execution failed: the statement has 0 placeholder(s) but 1 parameter(s) were passed": func() error {
			_, err := PostgreSQLExecute(handle, "DELETE FROM "+table, []string{PostgreSQLFromInt(1)})
			return err
		},
		"PostgreSQL query failed: (42601) cannot insert multiple commands into a prepared statement": func() error {
			_, _, err := PostgreSQLQuery(handle, "SELECT 1 AS one; SELECT 2 AS two", nil)
			return err
		},
		"PostgreSQL execution failed: (42601) cannot insert multiple commands into a prepared statement": func() error {
			_, err := PostgreSQLExecute(handle, "UPDATE "+table+" SET done = true; DROP TABLE "+table, nil)
			return err
		},
		`PostgreSQL execution failed: (23505) duplicate key value violates unique constraint "notes_title_key"`: func() error {
			_, err := PostgreSQLExecute(handle, "INSERT INTO "+table+" (title) VALUES ($1)", []string{PostgreSQLFromString("second")})
			return err
		},
		`PostgreSQL query failed: (42P01) relation "` + schema + `.missing" does not exist`: func() error {
			_, _, err := PostgreSQLQuery(handle, "SELECT * FROM "+schema+".missing", nil)
			return err
		},
	}
	for want, run := range failures {
		requirePostgreSQLError(t, run(), want)
	}
	_, _, err = PostgreSQLQuery(handle, "SELECT $1::integer AS n", []string{PostgreSQLFromInt(1 << 40)})
	if err == nil || !strings.HasPrefix(err.Error(), "PostgreSQL query failed: (22003) ") {
		t.Fatalf("out-of-range error %v", err)
	}
	if count := mustPostgreSQLQuery(t, handle, "SELECT count(*) AS n FROM "+table)[0]["n"]; count != "Int 2" {
		t.Fatalf("a rejected multi-statement call changed the table: %s", count)
	}
}

func TestPostgreSQLIntegrationTransactions(t *testing.T) {
	target, handle, schema := postgresqlIntegration(t)
	table := schema + ".ledger"
	mustPostgreSQLExecute(t, handle, "CREATE TABLE "+table+" (id integer PRIMARY KEY, note text NOT NULL)")
	count := func() string {
		t.Helper()
		return mustPostgreSQLQuery(t, handle, "SELECT count(*) AS n FROM "+table)[0]["n"]
	}
	insert := "INSERT INTO " + table + " (id, note) VALUES ($1, $2)"

	committed := mustPostgreSQLBegin(t, handle)
	mustPostgreSQLTransactionExecute(t, committed, insert, PostgreSQLFromInt(1), PostgreSQLFromString("one"))
	columns, inside, err := PostgreSQLTransactionQuery(committed, "SELECT count(*) AS n FROM "+table, nil)
	if err != nil || postgresqlShowRows(t, columns, inside)[0]["n"] != "Int 1" {
		t.Fatalf("transaction does not see its own row: %v %v", inside, err)
	}
	if outside := count(); outside != "Int 0" {
		t.Fatalf("an uncommitted row is visible outside: %s", outside)
	}
	if err := PostgreSQLTransactionCommit(committed); err != nil {
		t.Fatal(err)
	}
	if after := count(); after != "Int 1" {
		t.Fatalf("committed row missing: %s", after)
	}
	requirePostgreSQLError(t, PostgreSQLTransactionCommit(committed), ahdPostgreSQLTransactionFinished)
	requirePostgreSQLError(t, PostgreSQLTransactionRollback(committed), ahdPostgreSQLTransactionFinished)
	_, err = PostgreSQLTransactionExecute(committed, insert, []string{PostgreSQLFromInt(9), PostgreSQLFromString("late")})
	requirePostgreSQLError(t, err, ahdPostgreSQLTransactionFinished)

	rolled := mustPostgreSQLBegin(t, handle)
	mustPostgreSQLTransactionExecute(t, rolled, insert, PostgreSQLFromInt(2), PostgreSQLFromString("two"))
	if err := PostgreSQLTransactionRollback(rolled); err != nil {
		t.Fatal(err)
	}
	requirePostgreSQLError(t, PostgreSQLTransactionRollback(rolled), ahdPostgreSQLTransactionFinished)
	if after := count(); after != "Int 1" {
		t.Fatalf("rolled-back row persisted: %s", after)
	}

	aborted := mustPostgreSQLBegin(t, handle)
	mustPostgreSQLTransactionExecute(t, aborted, insert, PostgreSQLFromInt(3), PostgreSQLFromString("three"))
	_, err = PostgreSQLTransactionExecute(aborted, insert, []string{PostgreSQLFromInt(1), PostgreSQLFromString("again")})
	requirePostgreSQLError(t, err, `PostgreSQL execution failed: (23505) duplicate key value violates unique constraint "ledger_pkey"`)
	_, _, err = PostgreSQLTransactionQuery(aborted, "SELECT 1 AS one", nil)
	requirePostgreSQLError(t, err, "PostgreSQL query failed: (25P02) current transaction is aborted, commands ignored until end of transaction block")
	requirePostgreSQLError(t, PostgreSQLTransactionCommit(aborted), "PostgreSQL transaction was rolled back because an earlier statement failed")
	if err := PostgreSQLTransactionRollback(aborted); err != nil {
		t.Fatalf("rollback after a failed commit raised: %v", err)
	}
	requirePostgreSQLError(t, PostgreSQLTransactionCommit(aborted), ahdPostgreSQLTransactionFinished)
	if after := count(); after != "Int 1" {
		t.Fatalf("an aborted transaction persisted rows: %s", after)
	}

	other := target.connect(t, 10)
	open := mustPostgreSQLBegin(t, other)
	mustPostgreSQLTransactionExecute(t, open, insert, PostgreSQLFromInt(4), PostgreSQLFromString("four"))
	if err := PostgreSQLClose(other); err != nil {
		t.Fatal(err)
	}
	if err := PostgreSQLClose(other); err != nil {
		t.Fatalf("second close raised: %v", err)
	}
	if after := count(); after != "Int 1" {
		t.Fatalf("close committed an open transaction: %s", after)
	}
	requirePostgreSQLError(t, PostgreSQLPing(other), "this PostgreSQLDatabase is closed")
	_, err = PostgreSQLExecute(other, "SELECT 1", nil)
	requirePostgreSQLError(t, err, "this PostgreSQLDatabase is closed")
	_, err = PostgreSQLBegin(other)
	requirePostgreSQLError(t, err, "this PostgreSQLDatabase is closed")
	requirePostgreSQLError(t, PostgreSQLTransactionCommit(open), ahdPostgreSQLTransactionFinished)
}

func TestPostgreSQLIntegrationTimeoutAndConcurrency(t *testing.T) {
	target, handle, _ := postgresqlIntegration(t)

	quick := target.connect(t, 1)
	started := time.Now()
	_, _, err := PostgreSQLQuery(quick, "SELECT pg_sleep(5)::text AS slept", nil)
	requirePostgreSQLError(t, err, "PostgreSQL connection timed out")
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("the timed-out statement held the caller for %s", elapsed)
	}
	if one := mustPostgreSQLQuery(t, quick, "SELECT 1 AS one")[0]["one"]; one != "Int 1" {
		t.Fatalf("pool unusable after a timeout: %s", one)
	}

	var group sync.WaitGroup
	failures := make(chan string, 16)
	for worker := range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			for round := range 25 {
				want := PostgreSQLFromInt(int64(worker*100 + round))
				_, rows, err := PostgreSQLQuery(handle, "SELECT $1::bigint AS n", []string{want})
				if err != nil || len(rows) != 1 || rows[0][0] != want {
					failures <- fmt.Sprintf("worker %d round %d: rows %v error %v", worker, round, rows, err)
					return
				}
			}
		}()
	}
	group.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}

func TestPostgreSQLIntegrationConnectionBoundaries(t *testing.T) {
	target := postgresqlIntegrationTarget(t)
	for key, value := range map[string]string{
		"PGHOST": "192.0.2.1", "PGPORT": "1", "PGDATABASE": "ahd_hostile", "PGUSER": "ahd_hostile",
		"PGPASSWORD": "ahd_hostile", "PGOPTIONS": "-c search_path=ahd_hostile", "PGAPPNAME": "ahd_hostile",
		"PGTZ": "Asia/Tokyo", "PGSSLMODE": "verify-full", "PGCONNECT_TIMEOUT": "1",
	} {
		t.Setenv(key, value)
	}
	handle := target.connect(t, 10)
	settings := mustPostgreSQLQuery(t, handle, `SELECT current_setting('search_path') AS path,
		current_setting('application_name') AS app, current_setting('TimeZone') AS zone, current_user::text AS who`)[0]
	if strings.Contains(settings["path"], "ahd_hostile") || settings["app"] != `String ""` ||
		settings["zone"] == `String "Asia/Tokyo"` || settings["who"] != "String "+strconv.Quote(target.username) {
		t.Fatalf("PG* variables reached the session: %v", settings)
	}

	t.Run("missing database", func(t *testing.T) {
		name := "ahd_missing_" + postgresqlRandomSuffix(t)
		_, err := PostgreSQLConnect(target.host, target.username, target.password, target.port, &name, target.security, 10)
		requirePostgreSQLError(t, err, fmt.Sprintf("PostgreSQL connection failed: (3D000) database %q does not exist", name))
	})
	t.Run("wrong password", func(t *testing.T) {
		wrong := target.password + "-ahd-wrong-password"
		handle, err := PostgreSQLConnect(target.host, target.username, wrong, target.port, target.database, target.security, 10)
		if err == nil {
			_ = PostgreSQLClose(handle)
			t.Skip("the integration server does not check passwords")
		}
		if !strings.HasPrefix(err.Error(), "PostgreSQL connection failed") || strings.Contains(err.Error(), wrong) {
			t.Fatalf("wrong password error %q", err)
		}
	})
	t.Run("tls required against a plaintext server", func(t *testing.T) {
		if target.security != "none" {
			t.Skip("the integration server is reached over TLS")
		}
		_, err := PostgreSQLConnect(target.host, target.username, target.password, target.port, target.database, "tls", 10)
		requirePostgreSQLError(t, err, "PostgreSQL TLS verification failed")
	})
}
