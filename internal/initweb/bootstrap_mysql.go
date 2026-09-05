package initweb

import (
	"fmt"
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
)

// mysqlOperator is the testable sequencing surface for Admin/MySQL
// initialization. The live implementation uses the released MySQL runtime.
type mysqlOperator interface {
	Connect(host, username, password string, port int, security string) error
	DatabaseExists(name string) (bool, error)
	CreateDatabase(name string) error
	UseDatabase(name string) error
	Exec(sqlText string, parameters []string) error
	Close()
}

type mysqlSession struct {
	host     string
	username string
	password string
	port     int
	security string
	handle   string
}

func newMySQLSession(options Options) *mysqlSession {
	return &mysqlSession{
		host:     options.MySQLHost,
		username: options.MySQLUser,
		password: options.MySQLPassword,
		port:     options.MySQLPort,
		security: options.MySQLSecurity,
	}
}

func (session *mysqlSession) Connect(host, username, password string, port int, security string) error {
	session.host = host
	session.username = username
	session.password = password
	session.port = port
	session.security = security
	return session.connect(nil)
}

func (session *mysqlSession) connect(database *string) error {
	session.Close()
	handle, err := ahdruntime.MySQLConnect(session.host, session.username, session.password, int64(session.port), database, session.security, 10)
	if err != nil {
		return fmt.Errorf("cannot connect:\n%v", sanitizeSecret(err.Error(), session.password))
	}
	session.handle = handle
	return nil
}

func (session *mysqlSession) DatabaseExists(name string) (bool, error) {
	_, rows, err := ahdruntime.MySQLQuery(
		session.handle,
		"SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?",
		[]string{ahdruntime.MySQLFromString(name)},
	)
	if err != nil {
		return false, fmt.Errorf("cannot connect:\n%v", sanitizeSecret(err.Error(), session.password))
	}
	return len(rows) > 0, nil
}

func (session *mysqlSession) CreateDatabase(name string) error {
	_, err := ahdruntime.MySQLExecute(session.handle, "CREATE DATABASE `"+name+"`", nil)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "1044") || strings.Contains(lower, "access denied") {
			return fmt.Errorf("Unable to create database %q.\nThe MySQL account does not have CREATE DATABASE permission.", name)
		}
		return fmt.Errorf("Unable to create database %q.\n%v", name, sanitizeSecret(err.Error(), session.password))
	}
	return nil
}

func (session *mysqlSession) UseDatabase(name string) error {
	return session.connect(&name)
}

func (session *mysqlSession) Exec(sqlText string, parameters []string) error {
	_, err := ahdruntime.MySQLExecute(session.handle, sqlText, parameters)
	return err
}

func (session *mysqlSession) Close() {
	if session.handle == "" {
		return
	}
	_ = ahdruntime.MySQLClose(session.handle)
	session.handle = ""
}

func bootstrapMySQL(options Options, operator mysqlOperator) (created bool, err error) {
	if operator == nil {
		operator = newMySQLSession(options)
	}
	if err := operator.Connect(options.MySQLHost, options.MySQLUser, options.MySQLPassword, options.MySQLPort, options.MySQLSecurity); err != nil {
		return false, err
	}
	defer operator.Close()

	exists, err := operator.DatabaseExists(options.DatabaseName)
	if err != nil {
		return false, err
	}
	if exists {
		return false, fmt.Errorf("database %q already exists.\nAhdCode will not alter or reuse an existing MySQL database.", options.DatabaseName)
	}
	if err := operator.CreateDatabase(options.DatabaseName); err != nil {
		return false, err
	}
	created = true
	if err := operator.UseDatabase(options.DatabaseName); err != nil {
		return true, fmt.Errorf("schema initialization failed:\n%v", err)
	}
	if err := operator.Exec(strings.TrimSpace(mysqlSchemaSQL), nil); err != nil {
		return true, fmt.Errorf("schema initialization failed:\n%v", sanitizeSecret(err.Error(), options.MySQLPassword))
	}
	hash, err := hashPassword(options.AdminPassword)
	if err != nil {
		return true, fmt.Errorf("administrator creation failed")
	}
	if err := operator.Exec(
		"INSERT INTO users (name, email, password_hash, is_admin, created_at, updated_at) VALUES (?, ?, ?, 1, NOW(), NOW())",
		[]string{
			ahdruntime.MySQLFromString(options.AdminName),
			ahdruntime.MySQLFromString(options.AdminEmail),
			ahdruntime.MySQLFromString(hash),
		},
	); err != nil {
		return true, fmt.Errorf("administrator creation failed")
	}
	return true, nil
}

func sanitizeSecret(message, secret string) string {
	if secret == "" {
		return message
	}
	return strings.ReplaceAll(message, secret, "")
}

func leftoverMySQLMessage(name string) string {
	return fmt.Sprintf("MySQL database %q was created but initialization did not finish.\nThe database was not dropped.", name)
}
