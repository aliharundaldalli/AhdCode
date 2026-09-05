package initweb

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ahdDataMySQLHostKey     = "AHD_DATA_MYSQL_HOST"
	ahdDataMySQLPortKey     = "AHD_DATA_MYSQL_PORT"
	ahdDataMySQLSecurityKey = "AHD_DATA_MYSQL_SECURITY"
	ahdDataSQLitePathsKey   = "AHD_DATA_SQLITE_PATHS"
	AhdDataStudioHost       = "ahddatabasestudio.test"
	ahdDataStudioURL        = "http://ahddatabasestudio.test:8081/AhdDataStudio"
	ahdDataStudioBindURL    = "http://127.0.0.1:8081/AhdDataStudio"
	defaultStudioMySQLHost  = "127.0.0.1"
	defaultStudioMySQLPort  = 3306
)

// AhdDataStudioPublicURL is the .test identity the CLI prints and opens.
func AhdDataStudioPublicURL() string { return ahdDataStudioURL }

// AhdDataStudioLoopbackURL is the bind address, used when .test is not in hosts.
func AhdDataStudioLoopbackURL() string { return ahdDataStudioBindURL }

// LocateAhdDataStudio finds tools/AhdDataStudio/app.ahd from AHDCODE_ROOT
// or by walking up from the current directory.
func LocateAhdDataStudio() (string, error) {
	if root := strings.TrimSpace(os.Getenv("AHDCODE_ROOT")); root != "" {
		candidate := filepath.Join(root, "tools", "AhdDataStudio")
		if studioAppExists(candidate) {
			return candidate, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot find AhdDataStudio.\n%v", err)
	}
	dir := cwd
	for {
		nested := filepath.Join(dir, "tools", "AhdDataStudio")
		if studioAppExists(nested) {
			return nested, nil
		}
		if filepath.Base(dir) == "AhdDataStudio" && studioAppExists(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find AhdDataStudio.\nRun ahdcode databases from the AhdCode repository, or set AHDCODE_ROOT.")
}

func studioAppExists(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "app.ahd"))
	return err == nil && !info.IsDir()
}

func resolveStudioMySQLHost() (string, error) {
	host, err := lookupStudioSetting(ahdDataMySQLHostKey)
	if err != nil {
		return defaultStudioMySQLHost, nil
	}
	return host, nil
}

func lookupStudioSetting(key string) (string, error) {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		return raw, nil
	}
	for _, path := range ahdDataStudioEnvFiles() {
		raw, ok, err := readEnvKey(path, key)
		if err != nil || !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		return strings.TrimSpace(raw), nil
	}
	return "", fmt.Errorf("%s is not set", key)
}

func resolveStudioMySQLPort() (int, error) {
	raw, err := lookupStudioSetting(ahdDataMySQLPortKey)
	if err != nil {
		return defaultStudioMySQLPort, nil
	}
	return parseRequiredMySQLPort(raw)
}

func parseRequiredMySQLPort(raw string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s must be a whole number between 1 and 65535", ahdDataMySQLPortKey)
	}
	if err := validateMySQLPort(port); err != nil {
		return 0, fmt.Errorf("%s must be a whole number between 1 and 65535", ahdDataMySQLPortKey)
	}
	return port, nil
}

func registerSQLiteWithStudio(dbAbs string) bool {
	if !filepath.IsAbs(dbAbs) || strings.TrimSpace(dbAbs) != dbAbs || strings.ContainsAny(dbAbs, ",\n\r\"'\x00") || strings.Contains(dbAbs, " #") {
		return false
	}
	dbAbs = filepath.Clean(dbAbs)
	files := ahdDataStudioEnvFiles()
	if len(files) == 0 {
		return false
	}
	return appendStudioListKey(files[0], ahdDataSQLitePathsKey, dbAbs) == nil
}

func appendStudioListKey(path, key, value string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	updated := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		name, current, ok := strings.Cut(trim, "=")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		merged := mergeCommaList(unquoteEnvFileValue(current), value)
		lines[i] = key + "=" + merged
		updated = true
		break
	}
	if !updated {
		if len(lines) == 1 && lines[0] == "" {
			lines = []string{key + "=" + value}
		} else {
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines[len(lines)-1] = key + "=" + value
				lines = append(lines, "")
			} else {
				lines = append(lines, key+"="+value)
			}
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
}

func mergeCommaList(existing, value string) string {
	var parts []string
	seen := map[string]bool{}
	for _, part := range strings.Split(existing, ",") {
		item := strings.TrimSpace(part)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		parts = append(parts, item)
	}
	if !seen[value] {
		parts = append(parts, value)
	}
	return strings.Join(parts, ",")
}

func ahdDataStudioEnvFiles() []string {
	var files []string
	seen := map[string]bool{}
	add := func(path string) {
		clean := filepath.Clean(path)
		if seen[clean] {
			return
		}
		info, err := os.Lstat(clean)
		if err != nil || !info.Mode().IsRegular() {
			return
		}
		seen[clean] = true
		files = append(files, clean)
	}

	// Explicit source root takes precedence over nearest cwd/upward discovery.
	if root := strings.TrimSpace(os.Getenv("AHDCODE_ROOT")); root != "" {
		add(filepath.Join(root, "tools", "AhdDataStudio", ".env"))
	}
	if dir, err := LocateAhdDataStudio(); err == nil {
		add(filepath.Join(dir, ".env"))
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			add(filepath.Join(dir, "tools", "AhdDataStudio", ".env"))
			if filepath.Base(dir) == "AhdDataStudio" {
				add(filepath.Join(dir, ".env"))
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return files
}

func readEnvKey(path, key string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		return unquoteEnvFileValue(value), true, nil
	}
	return "", false, scanner.Err()
}

func unquoteEnvFileValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 {
		if trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
			return strings.Trim(trimmed, `"`)
		}
		if trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'' {
			return strings.Trim(trimmed, `'`)
		}
	}
	if i := strings.Index(trimmed, " #"); i >= 0 {
		trimmed = strings.TrimSpace(trimmed[:i])
	}
	return trimmed
}
