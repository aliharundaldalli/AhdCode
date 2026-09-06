package initweb

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func resolveOptions(root string, options Options) (Options, error) {
	if options.Input == nil {
		options.Input = strings.NewReader("")
	}
	if options.SecretFile == nil {
		if file, ok := options.Input.(*os.File); ok {
			options.SecretFile = file
		}
	}
	if _, ok := options.Input.(*bufio.Reader); !ok {
		options.Input = bufio.NewReader(options.Input)
	}
	if options.Output == nil {
		options.Output = io.Discard
	}

	if options.Starter == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("ahdcode init web needs a starter when input is not a terminal.\nUse: ahdcode init web empty|basic|admin|mvc|crud")
		}
		starter, err := promptStarter(options)
		if err != nil {
			return options, err
		}
		options.Starter = starter
	} else {
		normalized, err := normalizeStarter(options.Starter)
		if err != nil {
			return options, err
		}
		options.Starter = normalized
	}

	if strings.TrimSpace(options.AppName) == "" {
		if options.IsTTY {
			name, err := promptLine(options, fmt.Sprintf("Application name [%s]:", defaultAppName(root)))
			if err != nil {
				return options, err
			}
			if strings.TrimSpace(name) == "" {
				options.AppName = defaultAppName(root)
			} else {
				options.AppName = name
			}
		} else {
			options.AppName = defaultAppName(root)
		}
	}
	if err := validateAppName(options.AppName); err != nil {
		return options, err
	}

	if !options.usesDatabase() {
		return options, nil
	}

	resolved, err := resolveAdminOptions(options)
	if err != nil {
		return options, err
	}
	if resolved.isAppStarter() {
		return resolveMailOptions(resolved)
	}
	return resolved, nil
}

func resolveAdminOptions(options Options) (Options, error) {
	if options.Database == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("%s initialization requires an interactive terminal", options.starterTitle())
		}
		driver, err := promptDatabase(options)
		if err != nil {
			return options, err
		}
		options.Database = driver
	} else {
		normalized, err := normalizeDatabase(options.Database)
		if err != nil {
			return options, err
		}
		options.Database = normalized
	}

	if strings.TrimSpace(options.DatabaseName) == "" {
		suggested := defaultDatabaseName(options.AppName)
		if options.IsTTY {
			name, err := promptLine(options, fmt.Sprintf("Database name [%s]:", suggested))
			if err != nil {
				return options, err
			}
			if strings.TrimSpace(name) == "" {
				options.DatabaseName = suggested
			} else {
				options.DatabaseName = name
			}
		} else {
			options.DatabaseName = suggested
		}
	}
	if err := validateDatabaseName(options.DatabaseName); err != nil {
		return options, err
	}
	options.SQLiteFile = sqliteFileName(options.DatabaseName)

	if options.Database == DriverMySQL {
		resolved, err := resolveMySQLOptions(options)
		if err != nil {
			return options, err
		}
		options = resolved
	}

	if strings.TrimSpace(options.AdminName) == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("%s initialization requires an interactive terminal", options.starterTitle())
		}
		name, err := promptLine(options, "Admin name:")
		if err != nil {
			return options, err
		}
		options.AdminName = name
	}
	if err := validateAdminName(options.AdminName); err != nil {
		return options, err
	}

	if strings.TrimSpace(options.AdminEmail) == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("%s initialization requires an interactive terminal", options.starterTitle())
		}
		email, err := promptLine(options, "Admin email:")
		if err != nil {
			return options, err
		}
		options.AdminEmail = email
	}
	if err := validateEmail(options.AdminEmail); err != nil {
		return options, err
	}

	if options.AdminPassword == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("%s initialization requires an interactive terminal", options.starterTitle())
		}
		password, err := promptSecret(options, "Admin password:")
		if err != nil {
			return options, err
		}
		confirm, err := promptSecret(options, "Confirm password:")
		if err != nil {
			return options, err
		}
		options.AdminPassword = password
		if err := validateAdminPassword(password, confirm); err != nil {
			return options, err
		}
	} else if err := validateAdminPassword(options.AdminPassword, options.AdminPassword); err != nil {
		return options, err
	}

	return options, nil
}

func resolveMySQLOptions(options Options) (Options, error) {
	if strings.TrimSpace(options.MySQLHost) == "" {
		host, err := resolveStudioMySQLHost()
		if err != nil {
			return options, err
		}
		options.MySQLHost = host
	}
	if err := validateMySQLHost(options.MySQLHost); err != nil {
		return options, err
	}

	if options.MySQLPort == 0 {
		port, err := resolveStudioMySQLPort()
		if err != nil {
			return options, err
		}
		options.MySQLPort = port
	}
	if err := validateMySQLPort(options.MySQLPort); err != nil {
		return options, err
	}

	if options.MySQLSecurity == "" {
		if raw, err := lookupStudioSetting(ahdDataMySQLSecurityKey); err == nil && raw != "" {
			options.MySQLSecurity = raw
		} else {
			options.MySQLSecurity = defaultMySQLSecurity(options.MySQLHost)
		}
	}
	normalized, err := normalizeMySQLSecurity(options.MySQLSecurity)
	if err != nil {
		return options, err
	}
	options.MySQLSecurity = normalized
	if err := validateMySQLSecurity(options.MySQLSecurity); err != nil {
		return options, err
	}

	if strings.TrimSpace(options.MySQLUser) == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("%s initialization requires an interactive terminal", options.starterTitle())
		}
		user, err := promptLine(options, "Username:")
		if err != nil {
			return options, err
		}
		options.MySQLUser = user
	}
	if strings.TrimSpace(options.MySQLUser) == "" {
		return options, fmt.Errorf("MySQL username is required")
	}

	if options.MySQLPassword == "" && options.IsTTY {
		password, err := promptSecret(options, "Password:")
		if err != nil {
			return options, err
		}
		options.MySQLPassword = password
	}

	return options, nil
}

func resolveMailOptions(options Options) (Options, error) {
	if options.MailPort == "" {
		options.MailPort = "587"
	}
	if options.MailSecurity == "" {
		options.MailSecurity = "starttls"
	}
	if options.MailFromName == "" {
		options.MailFromName = options.AppName
	}
	if options.MailHost != "" || !options.IsTTY {
		return options, nil
	}
	fmt.Fprint(options.Output, "Configure SMTP now? [y/N]\n> ")
	line, err := readLine(options.Input)
	if err != nil {
		return options, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return options, nil
	}
	host, err := promptLine(options, "MAIL_HOST:")
	if err != nil {
		return options, err
	}
	options.MailHost = strings.TrimSpace(host)
	port, err := promptLine(options, "MAIL_PORT [587]:")
	if err != nil {
		return options, err
	}
	if strings.TrimSpace(port) != "" {
		options.MailPort = strings.TrimSpace(port)
	}
	user, err := promptLine(options, "MAIL_USERNAME:")
	if err != nil {
		return options, err
	}
	options.MailUsername = strings.TrimSpace(user)
	password, err := promptSecret(options, "MAIL_PASSWORD:")
	if err != nil {
		return options, err
	}
	options.MailPassword = password
	from, err := promptLine(options, "MAIL_FROM_ADDRESS:")
	if err != nil {
		return options, err
	}
	options.MailFromAddr = strings.TrimSpace(from)
	name, err := promptLine(options, fmt.Sprintf("MAIL_FROM_NAME [%s]:", options.AppName))
	if err != nil {
		return options, err
	}
	if strings.TrimSpace(name) != "" {
		options.MailFromName = strings.TrimSpace(name)
	}
	security, err := promptLine(options, "MAIL_SECURITY [starttls]:")
	if err != nil {
		return options, err
	}
	if strings.TrimSpace(security) != "" {
		options.MailSecurity = strings.TrimSpace(security)
	}
	return options, nil
}

func promptStarter(options Options) (string, error) {
	fmt.Fprint(options.Output, "AhdCode Web Application Setup\n\nChoose a starter:\n\n  1. Empty\n  2. Basic\n  3. Admin\n  4. MVC\n  5. CRUD\n\n> ")
	line, err := readLine(options.Input)
	if err != nil {
		return "", err
	}
	return normalizeStarter(line)
}

func promptDatabase(options Options) (string, error) {
	fmt.Fprint(options.Output, "Database:\n  1. SQLite\n  2. MySQL\n\n> ")
	line, err := readLine(options.Input)
	if err != nil {
		return "", err
	}
	return normalizeDatabase(line)
}

func promptLine(options Options, label string) (string, error) {
	fmt.Fprint(options.Output, label+"\n> ")
	return readLine(options.Input)
}

func promptSecret(options Options, label string) (string, error) {
	fmt.Fprint(options.Output, label+"\n> ")
	return readSecret(options.Input, options.Output, options.SecretFile)
}

func normalizeStarter(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", StarterEmpty:
		return StarterEmpty, nil
	case "2", StarterBasic:
		return StarterBasic, nil
	case "3", StarterAdmin:
		return StarterAdmin, nil
	case "4", StarterMVC:
		return StarterMVC, nil
	case "5", StarterCRUD:
		return StarterCRUD, nil
	default:
		return "", fmt.Errorf("unknown starter %q; choose Empty, Basic, Admin, MVC, or CRUD", value)
	}
}

func normalizeDatabase(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", DriverSQLite:
		return DriverSQLite, nil
	case "2", DriverMySQL:
		return DriverMySQL, nil
	default:
		return "", fmt.Errorf("unknown database %q; choose SQLite or MySQL", value)
	}
}

func validateAdminName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("admin name is required")
	}
	if len(trimmed) > maxAppNameLen {
		return fmt.Errorf("admin name must be at most %d characters", maxAppNameLen)
	}
	return nil
}

func validateMySQLHost(host string) error {
	if host == "" {
		return fmt.Errorf("MySQL host must not be empty")
	}
	if strings.Contains(host, "://") {
		return fmt.Errorf("MySQL host must not be a URL")
	}
	if strings.TrimSpace(host) != host {
		return fmt.Errorf("MySQL host is not valid")
	}
	if looksLikeHostPort(host) {
		return fmt.Errorf("MySQL host is the server address only (127.0.0.1), not host:port")
	}
	if looksLikePort(host) {
		return fmt.Errorf("MySQL host %q looks like a port. Use 127.0.0.1 for this machine.", host)
	}
	for _, r := range host {
		if r < 32 || r == 127 || r == '/' || r == ' ' || r == '\t' {
			return fmt.Errorf("MySQL host is not valid")
		}
	}
	return nil
}

func looksLikeHostPort(host string) bool {
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]:") {
		return true
	}
	if strings.HasPrefix(host, "[") {
		return false
	}
	return strings.Count(host, ":") == 1
}

func looksLikePort(host string) bool {
	if host == "" {
		return false
	}
	for _, r := range host {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isLoopbackMySQLHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "127.0.0.1", "localhost", "::1", "[::1]":
		return true
	default:
		return false
	}
}

func defaultMySQLSecurity(host string) string {
	if isLoopbackMySQLHost(host) {
		return "none"
	}
	return "tls"
}

func normalizeMySQLSecurity(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "tls", "1":
		return "tls", nil
	case "none", "2":
		return "none", nil
	default:
		return "", fmt.Errorf("MySQL security must be tls or none")
	}
}

func validateMySQLPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("MySQL port must be a whole number between 1 and 65535")
	}
	return nil
}

func validateMySQLSecurity(value string) error {
	if value == "tls" || value == "none" {
		return nil
	}
	return fmt.Errorf("MySQL security must be tls or none")
}
