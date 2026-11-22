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
)

// The MySQL integration suite runs only against a disposable server named via
// AHDCODE_TEST_MYSQL_HOST, _PORT, _USERNAME, _PASSWORD, _DATABASE, and
// _SECURITY; without a host it skips. Every test creates its own throwaway
// table prefix and drops it when the test finishes.

type mysqlTestTarget struct {
	host, username, password, security string
	port                               int64
	database                           *string
}

func mysqlIntegrationTarget(t *testing.T) mysqlTestTarget {
	t.Helper()
	target := mysqlTestTarget{
		host:     os.Getenv("AHDCODE_TEST_MYSQL_HOST"),
		username: os.Getenv("AHDCODE_TEST_MYSQL_USERNAME"),
		password: os.Getenv("AHDCODE_TEST_MYSQL_PASSWORD"),
		security: "none",
		port:     3306,
	}
	if target.host == "" {
		t.Skip("set AHDCODE_TEST_MYSQL_HOST (with _PORT, _USERNAME, _PASSWORD, _DATABASE, _SECURITY) to run the MySQL integration suite")
	}
	if target.username == "" {
		t.Fatal("AHDCODE_TEST_MYSQL_USERNAME must be set together with AHDCODE_TEST_MYSQL_HOST")
	}
	if text := os.Getenv("AHDCODE_TEST_MYSQL_PORT"); text != "" {
		port, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			t.Fatalf("AHDCODE_TEST_MYSQL_PORT=%q is not a port number", text)
		}
		target.port = port
	}
	if name := os.Getenv("AHDCODE_TEST_MYSQL_DATABASE"); name != "" {
		target.database = &name
	}
	if security := os.Getenv("AHDCODE_TEST_MYSQL_SECURITY"); security != "" {
		target.security = security
	}
	return target
}

func (target mysqlTestTarget) connect(t *testing.T, timeoutSeconds int64) string {
	t.Helper()
	handle, err := MySQLConnect(target.host, target.username, target.password, target.port, target.database, target.security, timeoutSeconds)
	if err != nil {
		t.Fatalf("connecting to MySQL integration server: %v", err)
	}
	t.Cleanup(func() { _ = MySQLClose(handle) })
	return handle
}

func mysqlRandomTable(t *testing.T, prefix string) string {
	t.Helper()
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	return prefix + "_" + hex.EncodeToString(random)
}

func TestMySQLIntegrationCRUDAndTypes(t *testing.T) {
	target := mysqlIntegrationTarget(t)
	db := target.connect(t, 10)
	if err := MySQLPing(db); err != nil {
		t.Fatalf("ping: %v", err)
	}

	table := mysqlRandomTable(t, "ahd_items")
	t.Cleanup(func() {
		_, _ = MySQLExecute(db, "DROP TABLE IF EXISTS "+table, nil)
	})
	if _, err := MySQLExecute(db, fmt.Sprintf(
		"CREATE TABLE %s (id BIGINT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(120) NOT NULL, score DOUBLE, note VARCHAR(120) NULL, raw_bytes VARBINARY(32))",
		table,
	), nil); err != nil {
		t.Fatalf("create table: %v", err)
	}

	realVal, err := MySQLFromReal(91.5)
	if err != nil {
		t.Fatal(err)
	}
	res1, err := MySQLExecute(db, fmt.Sprintf("INSERT INTO %s (name, score, note, raw_bytes) VALUES (?, ?, ?, UNHEX('DEADBEEF'))", table),
		[]string{MySQLFromString("Ayşe"), realVal, MySQLNullValue()})
	if err != nil {
		t.Fatalf("insert 1: %v", err)
	}
	aff, err := MySQLResultAffectedRows(res1)
	if err != nil || aff != 1 {
		t.Fatalf("affectedRows = %d, err = %v", aff, err)
	}
	id1, err := MySQLResultLastInsertID(res1)
	if err != nil || id1 == nil || *id1 <= 0 {
		t.Fatalf("lastInsertId = %v, err = %v", id1, err)
	}

	resUpd, err := MySQLExecute(db, fmt.Sprintf("UPDATE %s SET note = ? WHERE id = ?", table),
		[]string{MySQLFromString("updated"), MySQLFromInt(*id1)})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if affUpd, err := MySQLResultAffectedRows(resUpd); err != nil || affUpd != 1 {
		t.Fatalf("update affected = %d, err = %v", affUpd, err)
	}

	cols, rows, err := MySQLQuery(db, fmt.Sprintf("SELECT id, name, score, note, raw_bytes FROM %s WHERE id = ?", table),
		[]string{MySQLFromInt(*id1)})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(cols) != 5 || len(rows) != 1 {
		t.Fatalf("unexpected cols=%v rows=%v", cols, rows)
	}
	if name, err := MySQLValueString(rows[0][1]); err != nil || name != "Ayşe" {
		t.Fatalf("name = %q, err = %v", name, err)
	}
	if score, err := MySQLValueReal(rows[0][2]); err != nil || score != 91.5 {
		t.Fatalf("score = %v, err = %v", score, err)
	}
	if note, err := MySQLValueString(rows[0][3]); err != nil || note != "updated" {
		t.Fatalf("note = %q, err = %v", note, err)
	}
	if isBin, err := MySQLValueIsBinary(rows[0][4]); err != nil || !isBin {
		t.Fatalf("isBinary = %v, err = %v", isBin, err)
	}
	if sz, err := MySQLValueBinarySize(rows[0][4]); err != nil || sz != 4 {
		t.Fatalf("binarySize = %d, err = %v", sz, err)
	}

	// Duplicate column labels must be rejected.
	if _, _, err := MySQLQuery(db, fmt.Sprintf("SELECT id AS dup, name AS dup FROM %s", table), nil); err == nil || !strings.Contains(err.Error(), "duplicate column") {
		t.Fatalf("duplicate column error = %v", err)
	}

	resDel, err := MySQLExecute(db, fmt.Sprintf("DELETE FROM %s WHERE id = ?", table), []string{MySQLFromInt(*id1)})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if affDel, err := MySQLResultAffectedRows(resDel); err != nil || affDel != 1 {
		t.Fatalf("delete affected = %d, err = %v", affDel, err)
	}
}

func TestMySQLIntegrationTransactionsConcurrencyAndRedaction(t *testing.T) {
	target := mysqlIntegrationTarget(t)
	db := target.connect(t, 10)

	// Authentication failure must never leak the password.
	badSecret := "Dispos4ble_Wrong_Secret_987654"
	if _, err := MySQLConnect(target.host, target.username, badSecret, target.port, target.database, target.security, 5); err == nil {
		t.Fatal("wrong password unexpectedly succeeded")
	} else if strings.Contains(err.Error(), badSecret) {
		t.Fatalf("password leaked in error: %v", err)
	}

	table := mysqlRandomTable(t, "ahd_tx")
	t.Cleanup(func() {
		_, _ = MySQLExecute(db, "DROP TABLE IF EXISTS "+table, nil)
	})
	if _, err := MySQLExecute(db, fmt.Sprintf("CREATE TABLE %s (id BIGINT AUTO_INCREMENT PRIMARY KEY, label VARCHAR(64) NOT NULL)", table), nil); err != nil {
		t.Fatal(err)
	}

	// Rollback discards uncommitted row.
	tx1, err := MySQLBegin(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MySQLTransactionExecute(tx1, fmt.Sprintf("INSERT INTO %s (label) VALUES (?)", table), []string{MySQLFromString("rolled-back")}); err != nil {
		t.Fatal(err)
	}
	if err := MySQLTransactionRollback(tx1); err != nil {
		t.Fatal(err)
	}

	// Commit persists row.
	tx2, err := MySQLBegin(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MySQLTransactionExecute(tx2, fmt.Sprintf("INSERT INTO %s (label) VALUES (?)", table), []string{MySQLFromString("committed")}); err != nil {
		t.Fatal(err)
	}
	if err := MySQLTransactionCommit(tx2); err != nil {
		t.Fatal(err)
	}

	// Concurrent pool queries + independent transactions under -race.
	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tx, err := MySQLBegin(db)
			if err != nil {
				errCh <- err
				return
			}
			if _, err := MySQLTransactionExecute(tx, fmt.Sprintf("INSERT INTO %s (label) VALUES (?)", table), []string{MySQLFromString(fmt.Sprintf("worker-%d", idx))}); err != nil {
				_ = MySQLTransactionRollback(tx)
				errCh <- err
				return
			}
			if err := MySQLTransactionCommit(tx); err != nil {
				errCh <- err
				return
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent transaction failed: %v", err)
	}

	_, rows, err := MySQLQuery(db, fmt.Sprintf("SELECT COUNT(*) AS n FROM %s", table), nil)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := MySQLValueInt(rows[0][0]); err != nil || n != 5 {
		t.Fatalf("final row count = %d, err = %v; want 5", n, err)
	}

	// Close prevents further queries.
	dedicated := target.connect(t, 5)
	if err := MySQLClose(dedicated); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, _, err := MySQLQuery(dedicated, "SELECT 1", nil); err == nil {
		t.Fatal("query succeeded on closed MySQL handle")
	}
}
