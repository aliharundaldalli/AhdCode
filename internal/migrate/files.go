package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// migrationFileName is the only accepted name inside a migration directory.
// The version is the integer prefix, so 0002 and 2 are the same version.
var migrationFileName = regexp.MustCompile(`^([0-9]+)_([A-Za-z][A-Za-z0-9_]*)\.(up|down)\.sql$`)

const maxMigrationNameRunes = 64

// File is one version's up script and its optional down script.
type File struct {
	Version      int64
	Name         string
	Label        string
	UpPath       string
	DownPath     string
	UpSQL        string
	DownSQL      string
	UpChecksum   string
	DownChecksum string
}

// Scan reads the migration directory. It does not open a database.
// Entries are the directory's own files: subdirectories are ignored and
// symbolic links are rejected, so a link cannot pull a script in from
// outside this directory. Versions must be the contiguous sequence 1..N.
func Scan(directory string) ([]File, error) {
	directory, err := cleanDir(directory)
	if err != nil {
		return nil, err
	}
	if err := ensurePathContained(directory); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("migration directory does not exist: %s", directory)
		}
		return nil, fmt.Errorf("migration directory cannot be read: %s", directory)
	}
	type pair struct {
		up, down string
		name     string
	}
	found := map[int64]*pair{}
	var versions []int64
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}
		path := filepath.Join(directory, name)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("migration file cannot be read: %s", name)
		}
		reparse, err := componentReparse(path)
		if err != nil {
			return nil, fmt.Errorf("migration file cannot be read: %s", name)
		}
		if reparse {
			return nil, fmt.Errorf("migration file %s is a symbolic link or junction; migrate reads only regular files inside %s", name, directory)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("migration file %s is not a regular file", name)
		}
		match := migrationFileName.FindStringSubmatch(name)
		if match == nil {
			return nil, fmt.Errorf("invalid migration file name %s; expected <version>_<name>.up.sql or <version>_<name>.down.sql", name)
		}
		version, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil || version < 1 {
			return nil, fmt.Errorf("invalid migration version in %s; versions start at 1", name)
		}
		kind := match[3]
		current := found[version]
		if current == nil {
			current = &pair{}
			found[version] = current
			versions = append(versions, version)
		}
		if kind == "up" {
			if current.up != "" {
				return nil, fmt.Errorf("duplicate migration version %d (%s and %s)", version, filepath.Base(current.up), name)
			}
			if current.down != "" && current.name != match[2] {
				return nil, fmt.Errorf("migration version %d has conflicting names %s and %s", version, current.name, match[2])
			}
			current.up = path
			current.name = match[2]
		} else {
			if current.down != "" {
				return nil, fmt.Errorf("duplicate down migration for version %d (%s and %s)", version, filepath.Base(current.down), name)
			}
			if current.up != "" && current.name != match[2] {
				return nil, fmt.Errorf("migration version %d has conflicting names %s and %s", version, current.name, match[2])
			}
			current.down = path
			if current.name == "" {
				current.name = match[2]
			}
		}
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	if len(versions) > 0 {
		for index, version := range versions {
			if version != int64(index+1) {
				return nil, fmt.Errorf("migration versions skip %d; versions must be 1, 2, 3, ... without gaps", index+1)
			}
		}
	}
	files := make([]File, 0, len(versions))
	for _, version := range versions {
		current := found[version]
		if current.up == "" {
			return nil, fmt.Errorf("migration %d (%s) has a down file but no up file", version, current.name)
		}
		upSQL, upChecksum, err := readScript(current.up)
		if err != nil {
			return nil, err
		}
		file := File{
			Version: version, Name: current.name, Label: label(version, current.name),
			UpPath: current.up, UpSQL: upSQL, UpChecksum: upChecksum,
		}
		if current.down != "" {
			downSQL, downChecksum, err := readScript(current.down)
			if err != nil {
				return nil, err
			}
			file.DownPath = current.down
			file.DownSQL = downSQL
			file.DownChecksum = downChecksum
		}
		files = append(files, file)
	}
	return files, nil
}

func readScript(path string) (string, string, error) {
	if err := ensurePathContained(path); err != nil {
		return "", "", err
	}
	data, err := readFileNoFollow(path)
	if err != nil {
		return "", "", fmt.Errorf("migration file cannot be read: %s", filepath.Base(path))
	}
	if !utf8.Valid(data) {
		return "", "", fmt.Errorf("migration file %s is not valid UTF-8", filepath.Base(path))
	}
	sum := sha256.Sum256(data)
	return string(data), hex.EncodeToString(sum[:]), nil
}

func label(version int64, name string) string {
	return fmt.Sprintf("%04d_%s", version, name)
}

func cleanDir(directory string) (string, error) {
	if strings.TrimSpace(directory) == "" {
		return "", fmt.Errorf("migration directory is empty")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("migration directory cannot be resolved: %s", directory)
	}
	return filepath.Clean(absolute), nil
}

// createMu serializes creators in this process. The lock file serializes
// creators in other processes. The operating system releases the file lock
// when the process exits, including after a crash, so a dead process cannot
// leave a lock that later creators must delete.
var createMu sync.Mutex

const createLockName = ".ahdcode-migrate.lock"

// createStage is a test hook. before-down runs after the up body is staged
// and before the down file is staged. down-published runs after the down
// file has its final name and before the up file does.
var createStage func(stage string) error

// Create writes the next up/down pair. The down file is not run unless
// migrate down is asked for it. Its initial statement fails on purpose so a
// down that was never written cannot drop history.
func Create(directory, name string) (File, error) {
	if !validName(name) {
		return File{}, fmt.Errorf("migration name %q must start with a letter and contain only letters, digits, and underscores", name)
	}
	directory, err := cleanDir(directory)
	if err != nil {
		return File{}, err
	}
	createMu.Lock()
	defer createMu.Unlock()
	if err := prepareMigrationDir(directory); err != nil {
		return File{}, err
	}
	lock, err := openLockFile(filepath.Join(directory, createLockName))
	if err != nil {
		return File{}, fmt.Errorf("migration directory cannot be locked: %s", err.Error())
	}
	defer lock.Close()
	if err := lockOS(lock); err != nil {
		return File{}, err
	}
	defer unlockOS(lock)
	if err := recoverPartial(directory); err != nil {
		return File{}, err
	}
	existing, err := Scan(directory)
	if err != nil {
		return File{}, err
	}
	version := int64(1)
	if len(existing) > 0 {
		version = existing[len(existing)-1].Version + 1
	}
	file := File{Version: version, Name: name, Label: label(version, name)}
	file.UpPath = filepath.Join(directory, file.Label+".up.sql")
	file.DownPath = filepath.Join(directory, file.Label+".down.sql")
	up := "-- " + file.Label + "\n-- This file runs inside one SQLite transaction.\n\n"
	down := "-- " + file.Label + " down\n" +
		"-- Replace this statement before running ahdcode migrate down.\n" +
		"-- Down is never run automatically.\n" +
		"SELECT * FROM ahdcode_migrate_down_not_written;\n"
	upIncoming := file.UpPath + ".incoming"
	downIncoming := file.DownPath + ".incoming"
	if err := writeNew(upIncoming, up); err != nil {
		return File{}, err
	}
	if createStage != nil {
		if err := createStage("before-down"); err != nil {
			_ = removeOwned(upIncoming)
			return File{}, err
		}
	}
	if err := writeNew(downIncoming, down); err != nil {
		_ = removeOwned(upIncoming)
		return File{}, err
	}
	if err := os.Rename(downIncoming, file.DownPath); err != nil {
		_ = removeOwned(upIncoming)
		_ = removeOwned(downIncoming)
		return File{}, fmt.Errorf("migration file cannot be created: %s", filepath.Base(file.DownPath))
	}
	if createStage != nil {
		if err := createStage("down-published"); err != nil {
			_ = removeOwned(file.DownPath)
			_ = removeOwned(upIncoming)
			return File{}, err
		}
	}
	if err := os.Rename(upIncoming, file.UpPath); err != nil {
		_ = removeOwned(file.DownPath)
		_ = removeOwned(upIncoming)
		return File{}, fmt.Errorf("migration file cannot be created: %s", filepath.Base(file.UpPath))
	}
	return file, nil
}

func validName(name string) bool {
	if name == "" || utf8.RuneCountInString(name) > maxMigrationNameRunes {
		return false
	}
	for index, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case index > 0 && (r >= '0' && r <= '9' || r == '_'):
		default:
			return false
		}
	}
	return true
}

func writeNew(path, text string) error {
	if _, err := os.Lstat(path); err == nil {
		reparse, reparseErr := componentReparse(path)
		if reparseErr == nil && reparse {
			return fmt.Errorf("refusing to write through symbolic link or junction %s", filepath.Base(path))
		}
		return fmt.Errorf("migration file already exists: %s", filepath.Base(path))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("migration file cannot be created: %s", filepath.Base(path))
	}
	handle, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("migration file cannot be created: %s", filepath.Base(path))
	}
	if _, err := handle.WriteString(text); err != nil {
		handle.Close()
		_ = removeOwned(path)
		return fmt.Errorf("migration file cannot be written: %s", filepath.Base(path))
	}
	if err := handle.Sync(); err != nil {
		handle.Close()
		_ = removeOwned(path)
		return fmt.Errorf("migration file cannot be written: %s", filepath.Base(path))
	}
	if err := handle.Close(); err != nil {
		_ = removeOwned(path)
		return fmt.Errorf("migration file cannot be written: %s", filepath.Base(path))
	}
	return nil
}

func prepareMigrationDir(directory string) error {
	if err := ensurePathContained(directory); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	switch {
	case os.IsNotExist(err):
		parent := filepath.Dir(directory)
		if err := ensurePathContained(parent); err != nil {
			return err
		}
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("migration directory cannot be created: %s", directory)
		}
		info, err = os.Lstat(directory)
		if err != nil {
			return fmt.Errorf("migration directory cannot be read: %s", directory)
		}
	case err != nil:
		return fmt.Errorf("migration directory cannot be read: %s", directory)
	}
	reparse, err := componentReparse(directory)
	if err != nil {
		return fmt.Errorf("migration directory cannot be read: %s", directory)
	}
	if reparse {
		return fmt.Errorf("migration directory is a symbolic link or junction: %s", directory)
	}
	if !info.IsDir() {
		return fmt.Errorf("migration directory is not a directory: %s", directory)
	}
	return nil
}

// ensurePathContained rejects a symbolic link or junction on any existing
// component of path. filepath.Abs does not resolve those links. A component
// that does not exist yet ends the walk. Fully closing the gap between this
// check and a later open is not portable. The final component of a migration
// file, the lock file, and a recovery rename are opened or classified without
// following a reparse point. A parent directory replaced by a link in that
// gap is still the remaining race.
func ensurePathContained(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("path cannot be resolved: %s", path)
	}
	absolute = filepath.Clean(absolute)
	volume := filepath.VolumeName(absolute)
	rest := absolute
	if len(volume) > 0 && len(absolute) >= len(volume) {
		rest = absolute[len(volume):]
	}
	current := volume
	if len(volume) == 2 && volume[1] == ':' {
		current = volume + `\`
	}
	for _, part := range strings.FieldsFunc(rest, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == "" || part == "." {
			continue
		}
		if current == "" || strings.HasSuffix(current, `\`) || strings.HasSuffix(current, `/`) {
			current += part
		} else {
			current += string(os.PathSeparator) + part
		}
		reparse, err := componentReparse(current)
		if isMissing(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("path cannot be read: %s", current)
		}
		if reparse {
			return fmt.Errorf("%s is a symbolic link or junction; migrate will not follow it", current)
		}
	}
	return nil
}

var incomingFileName = regexp.MustCompile(`^([0-9]+_[A-Za-z][A-Za-z0-9_]*)\.(up|down)\.sql\.incoming$`)

// recoverPartial finishes or discards files left by an interrupted Create.
// It only renames or removes *.incoming files, plus a down file this process
// published in the same Create call. A user migration without a down script
// has no incoming sibling and is left in place.
func recoverPartial(directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("migration directory cannot be read: %s", directory)
	}
	type staged struct {
		up, down bool
	}
	labels := map[string]*staged{}
	for _, entry := range entries {
		match := incomingFileName.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		if _, _, err := classifyLeaf(filepath.Join(directory, entry.Name())); err != nil {
			return fmt.Errorf("interrupted migration %s could not be recovered: %s", match[1], err.Error())
		}
		current := labels[match[1]]
		if current == nil {
			current = &staged{}
			labels[match[1]] = current
		}
		if match[2] == "up" {
			current.up = true
		} else {
			current.down = true
		}
	}
	for label, staged := range labels {
		upPath := filepath.Join(directory, label+".up.sql")
		downPath := filepath.Join(directory, label+".down.sql")
		upIncoming := filepath.Join(directory, label+".up.sql.incoming")
		downIncoming := filepath.Join(directory, label+".down.sql.incoming")
		_, upMissing, err := classifyLeaf(upPath)
		if err != nil {
			return fmt.Errorf("interrupted migration %s could not be recovered: %s", label, err.Error())
		}
		_, downMissing, err := classifyLeaf(downPath)
		if err != nil {
			return fmt.Errorf("interrupted migration %s could not be recovered: %s", label, err.Error())
		}
		upExists := !upMissing
		downExists := !downMissing
		switch {
		case downExists && !upExists && staged.up:
			if err := renameOwned(upIncoming, upPath); err != nil {
				return fmt.Errorf("interrupted migration %s could not be recovered: %s", label, err.Error())
			}
		case upExists && !downExists && staged.down:
			if err := renameOwned(downIncoming, downPath); err != nil {
				return fmt.Errorf("interrupted migration %s could not be recovered: %s", label, err.Error())
			}
		case !upExists && !downExists:
			if staged.up {
				if err := removeOwned(upIncoming); err != nil {
					return err
				}
			}
			if staged.down {
				if err := removeOwned(downIncoming); err != nil {
					return err
				}
			}
		}
		if upExists && downExists || (downExists && staged.up) || (upExists && staged.down) {
			_ = removeOwned(upIncoming)
			_ = removeOwned(downIncoming)
		}
	}
	return nil
}

func classifyLeaf(path string) (regular bool, missing bool, err error) {
	info, err := os.Lstat(path)
	if isMissing(err) {
		return false, true, nil
	}
	if err != nil {
		return false, false, err
	}
	reparse, err := componentReparse(path)
	if err != nil {
		return false, false, err
	}
	if reparse || !info.Mode().IsRegular() {
		return false, false, fmt.Errorf("%s is a symbolic link, junction, or non-regular file", filepath.Base(path))
	}
	return true, false, nil
}

func renameOwned(from, to string) error {
	if _, _, err := classifyLeaf(from); err != nil {
		return err
	}
	regular, missing, err := classifyLeaf(to)
	if err != nil {
		return err
	}
	if regular || !missing {
		return fmt.Errorf("refusing to replace %s", filepath.Base(to))
	}
	return os.Rename(from, to)
}

func removeOwned(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	reparse, err := componentReparse(path)
	if err != nil {
		return err
	}
	if reparse || !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to remove %s", filepath.Base(path))
	}
	return os.Remove(path)
}
