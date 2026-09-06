package initweb

import (
	"io"
	"os"
)

const (
	StarterEmpty = "empty"
	StarterBasic = "basic"
	StarterAdmin = "admin"
	StarterMVC   = "mvc"
	StarterCRUD  = "crud"

	DriverSQLite = "sqlite"
	DriverMySQL  = "mysql"

	minAdminPasswordLen = 8
	maxAppNameLen       = 80
	maxMySQLIdentLen    = 64
)

// Options is the fully resolved init-web request. Tests and the CLI both
// build this before any filesystem or database side effect.
type Options struct {
	Starter       string
	AppName       string
	Database      string
	DatabaseName  string
	SQLiteFile    string
	MySQLHost     string
	MySQLPort     int
	MySQLUser     string
	MySQLPassword string
	MySQLSecurity string
	AdminName     string
	AdminEmail    string
	AdminPassword string
	MailHost      string
	MailPort      string
	MailUsername  string
	MailPassword  string
	MailFromAddr  string
	MailFromName  string
	MailSecurity  string
	Input         io.Reader
	Output        io.Writer
	IsTTY         bool
	// SecretFile is the original terminal, when Input is an *os.File.
	// resolveOptions wraps Input in a bufio.Reader, which would otherwise
	// hide the handle from echo suppression.
	SecretFile *os.File
}

func (options Options) isAdmin() bool { return options.Starter == StarterAdmin }
func (options Options) isAppStarter() bool {
	return options.Starter == StarterMVC || options.Starter == StarterCRUD
}
func (options Options) usesDatabase() bool {
	return options.isAdmin() || options.isAppStarter()
}
func (options Options) isMySQL() bool {
	return options.usesDatabase() && options.Database == DriverMySQL
}
func (options Options) isSQLite() bool {
	return options.usesDatabase() && options.Database == DriverSQLite
}
func (options Options) starterTitle() string {
	switch options.Starter {
	case StarterBasic:
		return "Basic"
	case StarterAdmin:
		return "Admin"
	case StarterMVC:
		return "MVC"
	case StarterCRUD:
		return "CRUD"
	default:
		return "Empty"
	}
}
func (options Options) footerCredit() string {
	switch options.Starter {
	case StarterMVC:
		return "Built with AhdCode · MVC Starter"
	case StarterCRUD:
		return "Built with AhdCode · CRUD Starter"
	default:
		return "Built with AhdCode"
	}
}
