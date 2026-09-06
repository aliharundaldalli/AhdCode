package ahdruntime

import (
	"errors"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// A refused connection tells the user nothing useful unless the server's own
// answer reaches them. The server's typed error cannot contain this client's
// DSN, so it is safe to show for the connect stage exactly as it already is
// after a query.
func TestConnectStageReportsTheServersOwnError(t *testing.T) {
	denied := &mysqldriver.MySQLError{Number: 1045, Message: "Access denied for user 'root'@'localhost' (using password: YES)"}
	message := ahdMySQLStageMessage(denied, "connect")
	if !strings.Contains(message, "Access denied for user 'root'@'localhost'") {
		t.Fatalf("connect stage message = %q, want the server's own text", message)
	}
	if !strings.Contains(message, "1045") {
		t.Errorf("connect stage message omits the server error code: %q", message)
	}
}

// A failure with no server answer -- nothing listening, a dropped network --
// must stay generic, because the driver's raw text can carry the DSN.
func TestConnectStageWithoutAServerAnswerStaysGeneric(t *testing.T) {
	message := ahdMySQLStageMessage(errors.New("dial tcp 127.0.0.1:3306: connect: connection refused"), "connect")
	if message != "MySQL connection failed" {
		t.Fatalf("connect stage message = %q, want the generic form", message)
	}
	if strings.Contains(message, "127.0.0.1") {
		t.Error("the driver's raw text reached the message")
	}
}

// Whatever the stage, the password must never survive into the message.
func TestConnectErrorsAreSanitizedAgainstThePassword(t *testing.T) {
	password := "s3cr3t-pw"
	raw := errors.New("root:" + password + "@tcp(127.0.0.1:3306)/: handshake failure")
	mapped := ahdMySQLMapError(raw, "connect", password)
	if strings.Contains(mapped.Error(), password) {
		t.Fatalf("the password survived into %q", mapped.Error())
	}
	denied := &mysqldriver.MySQLError{Number: 1045, Message: "Access denied for user 'root'@'localhost'"}
	mapped = ahdMySQLMapError(denied, "connect", password)
	if strings.Contains(mapped.Error(), password) {
		t.Fatalf("the password survived into %q", mapped.Error())
	}
	if !strings.Contains(mapped.Error(), "Access denied") {
		t.Fatalf("the server's answer was lost: %q", mapped.Error())
	}
}
