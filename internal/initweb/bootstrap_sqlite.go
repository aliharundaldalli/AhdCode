package initweb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func sqliteStage(options Options) (string, error) {
	temp, err := os.CreateTemp("", "ahdcode-v018-sqlite-*.db")
	if err != nil {
		return "", fmt.Errorf("schema initialization failed:\n%v", err)
	}
	path := temp.Name()
	_ = temp.Close()

	if err := sqliteInitializeFile(path, options); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func sqliteInitializeFile(path string, options Options) error {
	conn, err := sqlite3.OpenFlags(path, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE)
	if err != nil {
		return fmt.Errorf("schema initialization failed:\n%v", err)
	}
	defer conn.Close()

	if err := conn.Exec(schemaSQL(options)); err != nil {
		return fmt.Errorf("schema initialization failed:\n%v", err)
	}

	hash, err := hashPassword(options.AdminPassword)
	if err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if options.isAppStarter() {
		return sqliteInsertAppUsers(conn, options, hash, now)
	}
	stmt, tail, err := conn.Prepare(
		"INSERT INTO users (name, email, password_hash, is_admin, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)",
	)
	if err != nil || stmt == nil || strings.TrimSpace(tail) != "" {
		return fmt.Errorf("administrator creation failed")
	}
	defer stmt.Close()
	if err := stmt.BindText(1, options.AdminName); err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	if err := stmt.BindText(2, options.AdminEmail); err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	if err := stmt.BindText(3, hash); err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	if err := stmt.BindText(4, now); err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	if err := stmt.BindText(5, now); err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	if err := stmt.Exec(); err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	return nil
}

func installSQLiteFile(root string, options Options, staged string) error {
	dest, err := resolveManaged(root, sqliteRelPath(options))
	if err != nil {
		return fmt.Errorf("cannot initialize Web project:\n%v", err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("cannot initialize Web project:\n%v", err)
	}
	if err := os.Rename(staged, dest); err != nil {
		data, readErr := os.ReadFile(staged)
		if readErr != nil {
			return fmt.Errorf("cannot initialize Web project:\n%v", err)
		}
		if writeErr := os.WriteFile(dest, data, 0o600); writeErr != nil {
			return fmt.Errorf("cannot initialize Web project:\n%v", writeErr)
		}
		_ = os.Remove(staged)
	}
	return nil
}

func inspectSQLiteAdminHash(path string) (string, error) {
	conn, err := sqlite3.OpenFlags(path, sqlite3.OPEN_READONLY)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	stmt, tail, err := conn.Prepare("SELECT password_hash FROM users LIMIT 1")
	if err != nil || stmt == nil || strings.TrimSpace(tail) != "" {
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("administrator row is missing")
	}
	defer stmt.Close()
	if !stmt.Step() {
		return "", fmt.Errorf("administrator row is missing")
	}
	return stmt.ColumnText(0), nil
}

func sqliteInsertAppUsers(conn *sqlite3.Conn, options Options, adminHash, now string) error {
	adminID, err := newPublicID()
	if err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	if err := sqliteInsertUser(conn, adminID, options.AdminName, options.AdminEmail, adminHash, "administrator", now); err != nil {
		return err
	}
	memberPassword, err := newOpaquePassword()
	if err != nil {
		return fmt.Errorf("member creation failed")
	}
	memberHash, err := hashPassword(memberPassword)
	if err != nil {
		return fmt.Errorf("member creation failed")
	}
	memberID, err := newPublicID()
	if err != nil {
		return fmt.Errorf("member creation failed")
	}
	return sqliteInsertUser(conn, memberID, "Ayşe Yılmaz", "ayse@example.com", memberHash, "member", now)
}

func sqliteInsertUser(conn *sqlite3.Conn, publicID, name, email, hash, role, now string) error {
	stmt, tail, err := conn.Prepare(
		"INSERT INTO users (public_id, name, email, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil || stmt == nil || strings.TrimSpace(tail) != "" {
		return fmt.Errorf("administrator creation failed")
	}
	defer stmt.Close()
	values := []string{publicID, name, email, hash, role, now, now}
	for index, value := range values {
		if err := stmt.BindText(index+1, value); err != nil {
			return fmt.Errorf("administrator creation failed")
		}
	}
	if err := stmt.Exec(); err != nil {
		return fmt.Errorf("administrator creation failed")
	}
	return nil
}
