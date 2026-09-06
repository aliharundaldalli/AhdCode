package initweb

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ahdcode/internal/studio"
)

const (
	ahdDataMySQLHostKey     = "AHD_DATA_MYSQL_HOST"
	ahdDataMySQLPortKey     = "AHD_DATA_MYSQL_PORT"
	ahdDataMySQLSecurityKey = "AHD_DATA_MYSQL_SECURITY"
	ahdDataSQLitePathsKey   = "AHD_DATA_SQLITE_PATHS"
	AhdDataStudioHost       = "ahddatabasestudio.test"
	// v0.19 routes ahddatabasestudio.test through the local router, so the
	// canonical URL no longer carries Studio's port or its mount path. The
	// loopback URL below keeps working unchanged and is what `ahdcode
	// databases` falls back to when local routing is not available.
	ahdDataStudioURL     = "http://ahddatabasestudio.test/"
	ahdDataStudioBindURL = "http://127.0.0.1:8081/AhdDataStudio"

	// Init runs from a project directory, which has no AhdDataStudio .env in
	// scope. Falling back to the ordinary local MySQL socket is better than
	// failing the wizard over configuration the user has no reason to have
	// set yet; an explicit setting still wins.
	defaultStudioMySQLHost = "127.0.0.1"
	defaultStudioMySQLPort = 3306
)

// AhdDataStudioPublicURL is the .test identity the CLI prints and opens.
func AhdDataStudioPublicURL() string { return ahdDataStudioURL }

// AhdDataStudioLoopbackURL is the direct bind address. It stays supported and
// is used whenever the clean name is not resolvable on this machine.
func AhdDataStudioLoopbackURL() string { return ahdDataStudioBindURL }

// LocateAhdDataStudio resolves the exact-version AhdDataStudio that belongs
// to this CLI. AHDCODE_ROOT is an explicit developer override. The ordinary
// installed-product path materializes the Studio bundled in the toolchain.
func LocateAhdDataStudio() (string, error) {
	if root := strings.TrimSpace(os.Getenv("AHDCODE_ROOT")); root != "" {
		candidate := filepath.Join(root, "tools", "AhdDataStudio")
		if studioAppExists(candidate) {
			return candidate, nil
		}
		return "", fmt.Errorf("AHDCODE_ROOT is set but %s does not contain AhdDataStudio.", candidate)
	}
	dir, err := materializeBundledStudio()
	if err != nil {
		return "", fmt.Errorf("cannot start AhdDataStudio.\n%v", err)
	}
	if studioAppExists(dir) {
		return dir, nil
	}
	return "", fmt.Errorf("cannot start AhdDataStudio.\nThe bundled Studio for this AhdCode version is missing.")
}

func materializeBundledStudio() (string, error) {
	return studio.Materialize()
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

	if root := strings.TrimSpace(os.Getenv("AHDCODE_ROOT")); root != "" {
		add(filepath.Join(root, "tools", "AhdDataStudio", ".env"))
	}
	// Do not materialize Studio just to look for a leftover .env. Registry
	// operations and init must stay independent of Studio extraction.
	if dir, err := studio.CacheDir(); err == nil && studioAppExists(dir) {
		add(filepath.Join(dir, ".env"))
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
