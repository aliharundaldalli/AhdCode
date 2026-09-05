package localdev

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The database registry replaces hand-editing AHD_DATA_SQLITE_PATHS for the
// ordinary case: a database AhdCode itself created, or one the user pointed
// at explicitly, becomes visible in AhdDataStudio without any environment
// configuration.
//
// It records metadata and nothing else. There is no password, no token, no
// connection string, and no database content here, and there is no scan: an
// entry exists because a starter created that exact file or because someone
// named it on the command line. Removing an entry removes the entry -- the
// SQLite file itself is never touched.
const (
	databasesSchema  = "ahdcode.localdatabases"
	databasesVersion = 1

	// DriverSQLite is the only driver this registry accepts in v0.19.
	// MySQL stays where it already is: explicit AhdDataStudio configuration,
	// because a MySQL source is inseparable from credentials and credentials
	// have no place in a registry.
	DriverSQLite = "sqlite"
)

// Database is one registered local database.
type Database struct {
	Driver      string `json:"driver"`
	Path        string `json:"path"`
	ProjectRoot string `json:"projectRoot,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	AddedAt     string `json:"addedAt,omitempty"`
}

// Available reports whether the registered file is still present. A missing
// file is shown as unavailable rather than deleted from the registry: a
// database on an unmounted volume or a project not currently checked out is
// temporarily absent, not withdrawn, and quietly forgetting it would lose
// user state that only the user should be able to remove.
func (database Database) Available() bool {
	if database.Path == "" {
		return false
	}
	info, err := os.Stat(database.Path)
	return err == nil && info.Mode().IsRegular()
}

func (database Database) valid() bool {
	if database.Driver != DriverSQLite {
		return false
	}
	if database.Path == "" || !filepath.IsAbs(database.Path) {
		return false
	}
	if strings.ContainsRune(database.Path, 0) {
		return false
	}
	return database.Path == filepath.Clean(database.Path)
}

type databaseFile struct {
	Schema    string     `json:"schema"`
	Version   int        `json:"version"`
	Databases []Database `json:"databases"`
}

// CanonicalDatabasePath resolves a user-supplied path to the single form the
// registry stores, so the same file named two different ways deduplicates
// instead of appearing twice. Symlinks are resolved when they can be, which
// is what makes "the same file" mean the same file rather than the same
// spelling.
func CanonicalDatabasePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("a database path is required")
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		return filepath.Clean(resolved), nil
	}
	return absolute, nil
}

func loadDatabasesFrom(path string) ([]Database, error) {
	var file databaseFile
	present, err := readJSON(path, &file)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}
	if file.Schema != databasesSchema {
		return nil, fmt.Errorf("%s is not an AhdCode database registry", path)
	}
	if file.Version != databasesVersion {
		return nil, fmt.Errorf("unsupported AhdCode database registry version %d", file.Version)
	}
	kept := make([]Database, 0, len(file.Databases))
	seen := map[string]bool{}
	for _, database := range file.Databases {
		if !database.valid() || seen[database.Path] {
			continue
		}
		seen[database.Path] = true
		kept = append(kept, database)
	}
	return kept, nil
}

func saveDatabasesTo(path string, databases []Database) error {
	sort.Slice(databases, func(i, j int) bool { return databases[i].Path < databases[j].Path })
	return writeAtomic(path, databaseFile{Schema: databasesSchema, Version: databasesVersion, Databases: databases})
}

// LoadDatabases returns every registered database, in stable path order.
func LoadDatabases() ([]Database, error) {
	path, err := DatabasesPath()
	if err != nil {
		return nil, err
	}
	return loadDatabasesFrom(path)
}

// AddSQLite registers one existing SQLite file. It reports whether the entry
// was newly added; registering an already-registered file is a no-op rather
// than an error, so a starter re-run and a manual `databases add` of the same
// path agree.
//
// The file must already exist: the registry describes databases, it does not
// create them.
func AddSQLite(path, projectRoot string) (Database, bool, error) {
	canonical, err := CanonicalDatabasePath(path)
	if err != nil {
		return Database{}, false, err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		if os.IsNotExist(err) {
			return Database{}, false, fmt.Errorf("no such database file: %s", canonical)
		}
		return Database{}, false, err
	}
	if !info.Mode().IsRegular() {
		return Database{}, false, fmt.Errorf("not a database file: %s", canonical)
	}

	entry := Database{
		Driver:      DriverSQLite,
		Path:        canonical,
		DisplayName: filepath.Base(canonical),
		AddedAt:     nowStamp(),
	}
	if root := strings.TrimSpace(projectRoot); root != "" {
		if absolute, err := filepath.Abs(root); err == nil {
			entry.ProjectRoot = filepath.Clean(absolute)
		}
	}
	if !entry.valid() {
		return Database{}, false, fmt.Errorf("refusing to register %s", canonical)
	}

	registryPath, err := DatabasesPath()
	if err != nil {
		return Database{}, false, err
	}
	added := false
	err = withLock(registryPath, func() error {
		databases, err := loadDatabasesFrom(registryPath)
		if err != nil {
			return err
		}
		for _, existing := range databases {
			if existing.Path == entry.Path {
				entry = existing
				return nil
			}
		}
		added = true
		return saveDatabasesTo(registryPath, append(databases, entry))
	})
	if err != nil {
		return Database{}, false, err
	}
	return entry, added, nil
}

// RemoveSQLite drops exactly one registry entry and reports whether anything
// matched. The SQLite file is never opened, moved, or deleted -- forgetting a
// database and destroying one are different operations, and only the first
// belongs to this command.
func RemoveSQLite(path string) (Database, bool, error) {
	canonical, err := CanonicalDatabasePath(path)
	if err != nil {
		return Database{}, false, err
	}
	registryPath, err := DatabasesPath()
	if err != nil {
		return Database{}, false, err
	}
	var removed Database
	found := false
	err = withLock(registryPath, func() error {
		databases, err := loadDatabasesFrom(registryPath)
		if err != nil {
			return err
		}
		kept := make([]Database, 0, len(databases))
		for _, database := range databases {
			if database.Path == canonical {
				removed = database
				found = true
				continue
			}
			kept = append(kept, database)
		}
		if !found {
			return nil
		}
		return saveDatabasesTo(registryPath, kept)
	})
	if err != nil {
		return Database{}, false, err
	}
	return removed, found, nil
}
