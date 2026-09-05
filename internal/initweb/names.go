package initweb

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

func defaultAppName(root string) string {
	base := filepath.Base(filepath.Clean(root))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "AhdCode Web"
	}
	if err := validateAppName(base); err != nil {
		return "AhdCode Web"
	}
	return base
}

func validateAppName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("application name is required")
	}
	if trimmed != name {
		return fmt.Errorf("application name must not start or end with spaces")
	}
	if len(trimmed) > maxAppNameLen {
		return fmt.Errorf("application name must be at most %d characters", maxAppNameLen)
	}
	if strings.ContainsAny(trimmed, "\n\r\t=") {
		return fmt.Errorf("application name must not contain newlines, tabs, or '='")
	}
	for _, r := range trimmed {
		if r < 32 || r == 127 {
			return fmt.Errorf("application name must not contain control characters")
		}
	}
	return nil
}

func defaultDatabaseName(appName string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.TrimSpace(appName) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
		case r == ' ' || r == '-' || r == '_':
			if b.Len() > 0 && !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	name := strings.Trim(b.String(), "_")
	if name == "" {
		name = "app"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "app_" + name
	}
	if len(name) > maxMySQLIdentLen {
		name = name[:maxMySQLIdentLen]
	}
	return name
}

func validateDatabaseName(name string) error {
	if name == "" {
		return fmt.Errorf("database name is required")
	}
	if len(name) > maxMySQLIdentLen {
		return fmt.Errorf("database name must be at most %d characters", maxMySQLIdentLen)
	}
	first := name[0]
	if (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') {
		return fmt.Errorf("database name must start with a letter")
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			continue
		}
		return fmt.Errorf("database name may contain only letters, digits, and underscores")
	}
	return nil
}

func validateEmail(value string) error {
	if !emailShaped(value) {
		return fmt.Errorf("admin email is not a valid email shape")
	}
	return nil
}

func emailShaped(value string) bool {
	if value == "" || strings.ContainsAny(value, " \t\n\r") {
		return false
	}
	parts := strings.Split(value, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	labels := strings.Split(parts[1], ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" {
			return false
		}
	}
	return true
}

func validateAdminPassword(password, confirm string) error {
	if password == "" {
		return fmt.Errorf("admin password is required")
	}
	if len(password) < minAdminPasswordLen {
		return fmt.Errorf("admin password must be at least %d characters", minAdminPasswordLen)
	}
	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func sqliteFileName(databaseName string) string {
	return databaseName + ".db"
}
