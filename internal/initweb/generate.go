package initweb

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func managedFor(options Options) []fileSpec {
	files := []fileSpec{
		{relPath: "app.ahd", perm: 0o644, content: []byte(renderApp(options))},
		{relPath: ".env", perm: 0o600, content: []byte(renderEnv(options, false))},
		{relPath: ".env.example", perm: 0o644, content: []byte(renderEnv(options, true))},
		{relPath: ".gitignore", perm: 0o644, mergeGI: true, content: []byte(renderGitignore(options))},
		{relPath: "Config/App.ahd", perm: 0o644, embedPath: "templates/shared/Config/App.ahd"},
		{relPath: "Components/Navbar.ahd", perm: 0o644, content: []byte(renderNavbar(options))},
		{relPath: "Components/Footer.ahd", perm: 0o644, embedPath: "templates/shared/Components/Footer.ahd"},
		{relPath: "Layouts/Main.ahd", perm: 0o644, content: []byte(renderMainLayout(options))},
		{relPath: "Pages/Home.ahd", perm: 0o644, content: []byte(renderHome(options))},
		{relPath: "public/style.css", perm: 0o644, embedPath: "templates/shared/public/style.css"},
		{relPath: "public/main.js", perm: 0o644, embedPath: "templates/shared/public/main.js"},
		{relPath: "public/ahdcode-logo.png", perm: 0o644, embedPath: "templates/shared/public/ahdcode-logo.png"},
		{relPath: "public/vendor/bootstrap/bootstrap.min.css", perm: 0o644, embedPath: "templates/vendor/bootstrap/bootstrap.min.css"},
		{relPath: "public/vendor/bootstrap/bootstrap.bundle.min.js", perm: 0o644, embedPath: "templates/vendor/bootstrap/bootstrap.bundle.min.js"},
		{relPath: "public/vendor/bootstrap/LICENSE", perm: 0o644, embedPath: "templates/vendor/bootstrap/LICENSE"},
	}
	if options.Starter == StarterBasic || options.Starter == StarterAdmin {
		files = append(files, fileSpec{
			relPath:   "Config/Mail.ahd",
			perm:      0o644,
			embedPath: "templates/shared/Config/Mail.ahd",
		})
	}
	if options.Starter == StarterAdmin {
		files = append(files,
			fileSpec{relPath: "Layouts/Guest.ahd", perm: 0o644, embedPath: "templates/admin/Layouts/Guest.ahd"},
			fileSpec{relPath: "Pages/Login.ahd", perm: 0o644, embedPath: "templates/admin/Pages/Login.ahd"},
			fileSpec{relPath: "Pages/Dashboard.ahd", perm: 0o644, embedPath: "templates/admin/Pages/Dashboard.ahd"},
			fileSpec{relPath: "Services/Auth.ahd", perm: 0o644, embedPath: "templates/admin/Services/Auth.ahd"},
			fileSpec{relPath: "database/schema.sql", perm: 0o644, content: []byte(schemaSQL(options))},
			fileSpec{relPath: "Config/Database.ahd", perm: 0o644, content: []byte(renderDatabaseConfig(options))},
			fileSpec{relPath: "Repositories/Users.ahd", perm: 0o644, content: []byte(renderUsersRepo(options))},
		)
	}
	return files
}

func requiredDirsFor(options Options) []string {
	dirs := []string{
		"Config",
		"Components",
		"Layouts",
		"Pages",
		"public",
		"public/vendor",
		"public/vendor/bootstrap",
	}
	if options.Starter == StarterAdmin {
		dirs = append(dirs, "Repositories", "Services", "database")
	}
	return dirs
}

func sqliteRelPath(options Options) string {
	return filepath.ToSlash(filepath.Join("database", options.SQLiteFile))
}

func renderGitignore(options Options) string {
	text := ".env\n*.dev\n*.run\n"
	if options.isAdmin() {
		text += "database/*.db\n"
	}
	return text
}

func renderEnv(options Options, example bool) string {
	var b strings.Builder
	b.WriteString(envLine("APP_NAME", options.AppName) + "\n")
	b.WriteString("APP_ENV=development\n")
	b.WriteString("APP_HOST=localhost\n")
	b.WriteString("APP_PROTOCOL=http\n")
	b.WriteString("SERVER_HOST=127.0.0.1\n")
	b.WriteString("SERVER_PORT=8080\n")
	if options.Starter == StarterBasic || options.Starter == StarterAdmin {
		b.WriteString("\n")
		b.WriteString("MAIL_HOST=\n")
		b.WriteString("MAIL_PORT=587\n")
		b.WriteString("MAIL_USERNAME=\n")
		b.WriteString("MAIL_PASSWORD=\n")
		b.WriteString("MAIL_FROM_ADDRESS=\n")
		b.WriteString(envLine("MAIL_FROM_NAME", options.AppName) + "\n")
		b.WriteString("MAIL_SECURITY=starttls\n")
	}
	if options.isSQLite() {
		b.WriteString("\n")
		b.WriteString("DB_DRIVER=sqlite\n")
		b.WriteString(envLine("DB_DATABASE", sqliteRelPath(options)) + "\n")
	}
	if options.isMySQL() {
		b.WriteString("\n")
		b.WriteString("DB_DRIVER=mysql\n")
		b.WriteString(envLine("DB_HOST", options.MySQLHost) + "\n")
		b.WriteString(fmt.Sprintf("DB_PORT=%d\n", options.MySQLPort))
		b.WriteString(envLine("DB_DATABASE", options.DatabaseName) + "\n")
		if example {
			b.WriteString("DB_USERNAME=\n")
			b.WriteString("DB_PASSWORD=\n")
		} else {
			b.WriteString(envLine("DB_USERNAME", options.MySQLUser) + "\n")
			b.WriteString(envLine("DB_PASSWORD", options.MySQLPassword) + "\n")
		}
		b.WriteString("DB_SECURITY=tls\n")
	}
	return b.String()
}

func envLine(key, value string) string {
	return key + "=" + quoteEnvValue(value)
}

func quoteEnvValue(value string) string {
	if value == "" {
		return ""
	}
	if !strings.ContainsAny(value, "\t#\"'") && !strings.Contains(value, "\\") && strings.TrimSpace(value) == value {
		return value
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func loadSpecContent(spec fileSpec) ([]byte, error) {
	if len(spec.content) > 0 {
		return normalizeNewlines(spec.content), nil
	}
	content, err := fs.ReadFile(templates, spec.embedPath)
	if err != nil {
		return nil, err
	}
	return normalizeNewlines(content), nil
}

func writePlanned(planned []plannedFile) error {
	for _, item := range planned {
		if err := os.MkdirAll(filepath.Dir(item.abs), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(item.abs, item.content, item.perm); err != nil {
			return err
		}
	}
	return nil
}
