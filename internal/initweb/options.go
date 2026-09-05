package initweb

import "io"

const (
	StarterEmpty = "empty"
	StarterBasic = "basic"
	StarterAdmin = "admin"

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
	Input         io.Reader
	Output        io.Writer
	IsTTY         bool
}

func (options Options) isAdmin() bool { return options.Starter == StarterAdmin }
func (options Options) isMySQL() bool {
	return options.isAdmin() && options.Database == DriverMySQL
}
func (options Options) isSQLite() bool {
	return options.isAdmin() && options.Database == DriverSQLite
}
