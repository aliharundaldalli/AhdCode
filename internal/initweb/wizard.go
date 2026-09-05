package initweb

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func resolveOptions(root string, options Options) (Options, error) {
	if options.Input == nil {
		options.Input = strings.NewReader("")
	}
	if _, ok := options.Input.(*bufio.Reader); !ok {
		options.Input = bufio.NewReader(options.Input)
	}
	if options.Output == nil {
		options.Output = io.Discard
	}

	if options.Starter == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("ahdcode init web needs a starter when input is not a terminal.\nUse: ahdcode init web empty|basic|admin")
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

	if options.Starter != StarterAdmin {
		return options, nil
	}

	return resolveAdminOptions(options)
}

func resolveAdminOptions(options Options) (Options, error) {
	if options.Database == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("Admin initialization requires an interactive terminal")
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
			return options, fmt.Errorf("Admin initialization requires an interactive terminal")
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
			return options, fmt.Errorf("Admin initialization requires an interactive terminal")
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
			return options, fmt.Errorf("Admin initialization requires an interactive terminal")
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
	if options.MySQLSecurity == "" {
		options.MySQLSecurity = "tls"
	}
	if err := validateMySQLSecurity(options.MySQLSecurity); err != nil {
		return options, err
	}

	if strings.TrimSpace(options.MySQLHost) == "" {
		if options.IsTTY {
			host, err := promptLine(options, "Host [127.0.0.1]:")
			if err != nil {
				return options, err
			}
			if strings.TrimSpace(host) == "" {
				options.MySQLHost = "127.0.0.1"
			} else {
				options.MySQLHost = host
			}
		} else {
			options.MySQLHost = "127.0.0.1"
		}
	}
	if err := validateMySQLHost(options.MySQLHost); err != nil {
		return options, err
	}

	if options.MySQLPort == 0 {
		if options.IsTTY {
			raw, err := promptLine(options, "Port [3306]:")
			if err != nil {
				return options, err
			}
			if strings.TrimSpace(raw) == "" {
				options.MySQLPort = 3306
			} else {
				port, convErr := strconv.Atoi(strings.TrimSpace(raw))
				if convErr != nil {
					return options, fmt.Errorf("MySQL port must be a whole number between 1 and 65535")
				}
				options.MySQLPort = port
			}
		} else {
			options.MySQLPort = 3306
		}
	}
	if err := validateMySQLPort(options.MySQLPort); err != nil {
		return options, err
	}

	if strings.TrimSpace(options.MySQLUser) == "" {
		if !options.IsTTY {
			return options, fmt.Errorf("Admin initialization requires an interactive terminal")
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

func promptStarter(options Options) (string, error) {
	fmt.Fprint(options.Output, "AhdCode Web Application Setup\n\nChoose a starter:\n\n  1. Empty\n  2. Basic\n  3. Admin\n\n> ")
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
	return readSecret(options.Input, options.Output)
}

func normalizeStarter(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", StarterEmpty:
		return StarterEmpty, nil
	case "2", StarterBasic:
		return StarterBasic, nil
	case "3", StarterAdmin:
		return StarterAdmin, nil
	default:
		return "", fmt.Errorf("unknown starter %q; choose Empty, Basic, or Admin", value)
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
	for _, r := range host {
		if r < 32 || r == 127 || r == '/' || r == ' ' || r == '\t' {
			return fmt.Errorf("MySQL host is not valid")
		}
	}
	return nil
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
