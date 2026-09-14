package evaluator

import (
	"net"
	"testing"
)

// The evaluator drives the same PostgreSQL runtime a compiled program does, so
// values and messages match internal/build's native expectations.
func TestPostgreSQLThroughEvaluator(t *testing.T) {
	session := newLatexTestSession()
	flag := session.postgresqlBuiltin("fromBool", []any{true})
	if kind := session.postgresqlOperation("PostgreSQLValue.kind", flag, nil); kind != "Bool" {
		t.Fatalf("kind = %#v", kind)
	}
	if value := session.postgresqlOperation("PostgreSQLValue.bool", flag, nil); value != true {
		t.Fatalf("bool = %#v", value)
	}
	count := session.postgresqlBuiltin("fromInt", []any{int64(7)})
	if value := session.postgresqlOperation("PostgreSQLValue.real", count, nil); value != 7.0 {
		t.Fatalf("real = %#v", value)
	}
	if value := session.postgresqlOperation("PostgreSQLValue.isNull", session.postgresqlBuiltin("nullValue", nil), nil); value != true {
		t.Fatalf("isNull = %#v", value)
	}
	message := evaluatorRaisedMessage(t, "PostgreSQLError", func() {
		session.postgresqlOperation("PostgreSQLValue.string", count, nil)
	})
	if message != "string() requires kind String; this PostgreSQLValue has kind Int (check kind() first)" {
		t.Fatalf("wrong kind message %q", message)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	message = evaluatorRaisedMessage(t, "PostgreSQLError", func() {
		session.postgresqlBuiltin("connect", []any{"127.0.0.1", "app", "super-secret", int64(port), nil, "none", int64(2)})
	})
	if message != "PostgreSQL connection failed" {
		t.Fatalf("refused connection message %q", message)
	}
	message = evaluatorRaisedMessage(t, "PostgreSQLError", func() {
		session.postgresqlBuiltin("connect", []any{"", "app", "pw"})
	})
	if message != "PostgreSQL host must not be empty" {
		t.Fatalf("validation message %q", message)
	}
}
