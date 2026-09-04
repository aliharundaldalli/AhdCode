package ahdruntime

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestMySQLValidateConnect(t *testing.T) {
	cases := []struct {
		name           string
		host, security string
		port, timeout  int64
		wantSubstring  string
	}{
		{"empty host", "", "tls", 3306, 10, "host"},
		{"url host", "mysql://localhost", "tls", 3306, 10, "URL"},
		{"port zero", "localhost", "tls", 0, 10, "port"},
		{"port too big", "localhost", "tls", 65536, 10, "port"},
		{"bad security", "localhost", "ssl", 3306, 10, "security"},
		{"security preferred rejected", "localhost", "preferred", 3306, 10, "security"},
		{"timeout zero", "localhost", "tls", 3306, 0, "timeoutSeconds"},
		{"timeout too big", "localhost", "tls", 3306, ahdMySQLMaxTimeoutSeconds + 1, "timeoutSeconds"},
		{"timeout max ok", "localhost", "tls", 3306, ahdMySQLMaxTimeoutSeconds, ""},
		{"none security ok", "localhost", "none", 3306, 10, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			message := ahdMySQLValidateConnect(c.host, "user", c.port, c.security, c.timeout)
			if c.wantSubstring == "" {
				if message != "" {
					t.Fatalf("expected no error, got %q", message)
				}
				return
			}
			if !strings.Contains(message, c.wantSubstring) {
				t.Fatalf("message %q does not contain %q", message, c.wantSubstring)
			}
		})
	}
}

func TestMySQLValidateConnectEmptyUsername(t *testing.T) {
	if message := ahdMySQLValidateConnect("localhost", "", 3306, "tls", 10); !strings.Contains(message, "username") {
		t.Fatalf("expected username error, got %q", message)
	}
}

func TestMySQLValueRoundTrip(t *testing.T) {
	null := MySQLNullValue()
	if kind, err := MySQLValueKind(null); err != nil || kind != ahdMySQLKindNull {
		t.Fatalf("null kind = %q, %v", kind, err)
	}
	if isNull, err := MySQLValueIsNull(null); err != nil || !isNull {
		t.Fatalf("null isNull = %v, %v", isNull, err)
	}

	i := MySQLFromInt(-42)
	value, err := MySQLValueInt(i)
	if err != nil || value != -42 {
		t.Fatalf("int round trip = %d, %v", value, err)
	}
	if real, err := MySQLValueReal(i); err != nil || real != -42 {
		t.Fatalf("int widened to real = %v, %v", real, err)
	}

	r, err := MySQLFromReal(3.5)
	if err != nil {
		t.Fatalf("fromReal: %v", err)
	}
	if real, err := MySQLValueReal(r); err != nil || real != 3.5 {
		t.Fatalf("real round trip = %v, %v", real, err)
	}

	s := MySQLFromString("hello")
	if value, err := MySQLValueString(s); err != nil || value != "hello" {
		t.Fatalf("string round trip = %q, %v", value, err)
	}

	binaryText, err := mysqlEncodeValue(ahdMySQLValue{Kind: ahdMySQLKindBinary, Binary: []byte{0x00, 0xFF, 0x10}})
	if err != nil {
		t.Fatalf("encode binary: %v", err)
	}
	if isBinary, err := MySQLValueIsBinary(binaryText); err != nil || !isBinary {
		t.Fatalf("isBinary = %v, %v", isBinary, err)
	}
	if size, err := MySQLValueBinarySize(binaryText); err != nil || size != 3 {
		t.Fatalf("binarySize = %d, %v", size, err)
	}
	if b64, err := MySQLValueBinaryBase64(binaryText); err != nil || b64 != "AP8Q" {
		t.Fatalf("binaryBase64 = %q, %v", b64, err)
	}
}

func TestMySQLValueRejectsNonFiniteReal(t *testing.T) {
	if _, err := MySQLFromReal(mathNaN()); err == nil {
		t.Fatal("expected error for NaN")
	}
	if _, err := MySQLFromReal(mathInf()); err == nil {
		t.Fatal("expected error for +Inf")
	}
}

func TestMySQLValueWrongKindIsControlledError(t *testing.T) {
	s := MySQLFromString("text")
	if _, err := MySQLValueInt(s); err == nil {
		t.Fatal("expected wrong-kind error calling int() on a String")
	}
	i := MySQLFromInt(1)
	if _, err := MySQLValueString(i); err == nil {
		t.Fatal("expected wrong-kind error calling string() on an Int")
	}
	if _, err := MySQLValueBinarySize(s); err == nil {
		t.Fatal("expected wrong-kind error calling binarySize() on a String")
	}
}

func TestMySQLValueCorruptTextIsControlledError(t *testing.T) {
	for _, text := range []string{"", "Znotakind", "Inot-a-number"} {
		if _, err := MySQLValueKind(text); err == nil {
			t.Fatalf("expected corrupted-storage error for %q", text)
		}
	}
}

func TestMySQLClassifyValueNull(t *testing.T) {
	value := ahdMySQLClassifyValue(nil, "INT")
	if value.Kind != ahdMySQLKindNull {
		t.Fatalf("nil raw bytes must classify as Null regardless of declared type, got %s", value.Kind)
	}
}

func TestMySQLClassifyValueByDeclaredType(t *testing.T) {
	cases := []struct {
		typeName string
		raw      string
		wantKind string
	}{
		{"INT", "42", ahdMySQLKindInt},
		{"BIGINT", "-9000", ahdMySQLKindInt},
		{"UNSIGNED BIGINT", "1", ahdMySQLKindInt},
		{"UNSIGNED BIGINT", "18446744073709551615", ahdMySQLKindString}, // overflows int64: exact String
		{"YEAR", "2026", ahdMySQLKindInt},
		{"FLOAT", "3.5", ahdMySQLKindReal},
		{"DOUBLE", "3.5", ahdMySQLKindReal},
		{"DECIMAL", "19.99", ahdMySQLKindString}, // never coerced to Real
		{"DATE", "2026-01-15", ahdMySQLKindString},
		{"DATETIME", "2026-01-15 10:30:00", ahdMySQLKindString},
		{"JSON", `{"a":1}`, ahdMySQLKindString},
		{"VARCHAR", "hello", ahdMySQLKindString},
		{"BLOB", "\x00\xff", ahdMySQLKindBinary},
		{"BINARY", "\x00\xff", ahdMySQLKindBinary},
		{"BIT", "\x01", ahdMySQLKindBinary},
	}
	for _, c := range cases {
		t.Run(c.typeName, func(t *testing.T) {
			value := ahdMySQLClassifyValue(sql.RawBytes(c.raw), c.typeName)
			if value.Kind != c.wantKind {
				t.Fatalf("%s %q classified as %s; want %s", c.typeName, c.raw, value.Kind, c.wantKind)
			}
		})
	}
}

func TestMySQLCheckDuplicateColumns(t *testing.T) {
	if err := ahdMySQLCheckDuplicateColumns([]string{"id", "name"}); err != nil {
		t.Fatalf("unique columns must not error: %v", err)
	}
	if err := ahdMySQLCheckDuplicateColumns([]string{"id", "id"}); err == nil {
		t.Fatal("expected an error for duplicate result-column labels")
	}
}

func TestMySQLResultEncodingRoundTrip(t *testing.T) {
	withID := strconv1(5) + "|" + strconv1(42)
	affected, err := MySQLResultAffectedRows(withID)
	if err != nil || affected != 5 {
		t.Fatalf("affectedRows = %d, %v", affected, err)
	}
	id, err := MySQLResultLastInsertID(withID)
	if err != nil || id == nil || *id != 42 {
		t.Fatalf("lastInsertId = %v, %v", id, err)
	}

	noID := strconv1(1) + "|"
	id, err = MySQLResultLastInsertID(noID)
	if err != nil || id != nil {
		t.Fatalf("expected nil lastInsertId for a non-generating statement, got %v, %v", id, err)
	}
}

func TestMySQLSanitizeRemovesPassword(t *testing.T) {
	message := ahdMySQLSanitize("connection failed: password=hunter2 rejected", "hunter2")
	if strings.Contains(message, "hunter2") {
		t.Fatalf("password leaked into sanitized message: %q", message)
	}
}

func TestMySQLErrorMappingNeverIncludesRawConnectText(t *testing.T) {
	err := errors.New("dial tcp 10.0.0.1:3306: password=hunter2 connect: connection refused")
	mapped := ahdMySQLMapError(err, "connect", "hunter2")
	if mapped == nil {
		t.Fatal("expected a mapped error")
	}
	if strings.Contains(mapped.Error(), "hunter2") || strings.Contains(mapped.Error(), "10.0.0.1") {
		t.Fatalf("connect-stage message must never include raw driver text, got %q", mapped.Error())
	}
}

func mathNaN() float64        { var zero float64; return zero / zero }
func mathInf() float64        { var zero float64; return 1 / zero }
func strconv1(n int64) string { return strconv.FormatInt(n, 10) }
