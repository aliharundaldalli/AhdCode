package ahdruntime

// Durable HTTP sessions. HTTP.sessions stays in memory. HTTP.persistentSessions
// stores the same SessionStore contract in a SQLite file the caller names.
//
// The cookie still carries only the random identifier. The database stores
// SHA-256 of that identifier, the string map, and an absolute expiry. A
// restarted process that opens the same file and cookie name sees the same
// live sessions and the same revocations.
//
// open, commit, rotate, and destroy each run in one transaction. One file has
// one shared connection. The connection mutex covers the whole transaction, and
// the store mutex is always taken first, so two stores on one file cannot
// interleave statements. The wait for a SQLite lock is the helper's busy
// timeout (5 seconds). These calls use BEGIN IMMEDIATE, so a reader waits on a
// writer the same way a writer waits. WAL does not make open skip that lock.
// Session I/O does not take the evaluator lock beyond the handler that is
// already running. Callbacks stay serialized.
//
// Consistency: commit replaces the whole value map. It does not merge keys.
// A commit of an identifier that was revoked or that has expired does not
// write a row and raises HTTPError. rotate of a stored identifier revokes it
// only when that row is still live, in the same transaction. A missing,
// expired, revoked, or already-rotated identifier does not become a new
// authenticated session. There is no silent new login.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ahdHTTPSessionSchemaVersion = "1"
	ahdHTTPSessionKeyMaxBytes   = 128
	ahdHTTPSessionValueMaxBytes = 65536
	ahdHTTPSessionJSONMaxBytes  = 262144
	ahdHTTPSessionIDLength      = 43
)

var (
	errHTTPSessionCorrupt  = errors.New("HTTP session data is corrupted")
	errHTTPSessionStale    = errors.New("HTTP session is no longer valid")
	errHTTPSessionTooLarge = errors.New("HTTP session data is too large")
)

// ahdHTTPSessionSchemaError is raised as its own text so the diagnostic stays
// actionable and is not wrapped as a generic storage failure.
type ahdHTTPSessionSchemaError struct {
	message string
}

func (err *ahdHTTPSessionSchemaError) Error() string { return err.message }

// ahdHTTPSessionFault is a test-only injected storage failure. A nil pointer
// means the database is used for real.
type ahdHTTPSessionFault struct {
	message string
}

var ahdHTTPSessionStorageFault atomic.Pointer[ahdHTTPSessionFault]

// ahdHTTPSessionDatabase is the one SQLite connection for one session file.
// Several stores (different cookie names, or a restarted open of the same
// file) share it. opens counts in-flight and completed PersistentSessions
// calls so a failed open cannot drop the shared object while another open
// still uses it. It does not decide when the file handle is closed. The
// handle is closed when no HTTP server is actually running, no request
// callback is still inside user code, and no evaluator execution is still
// in progress, or when release is called directly. The next operation opens
// the same file again. Committed rows are not deleted.
type ahdHTTPSessionDatabase struct {
	mutex  sync.Mutex
	path   string
	handle string
	opens  int
	// unscoped, scopes, and servers are protected exclusively by ahdHTTPLifetimeMu.
	unscoped bool
	scopes   map[uint64]int
	servers  map[*ahdHTTPServerState]struct{}
}

var (
	ahdHTTPSessionDatabases   = map[string]*ahdHTTPSessionDatabase{}
	ahdHTTPSessionDatabasesMu sync.Mutex
)

func ahdHTTPSessionDatabaseFor(path string) *ahdHTTPSessionDatabase {
	ahdHTTPSessionDatabasesMu.Lock()
	defer ahdHTTPSessionDatabasesMu.Unlock()
	database := ahdHTTPSessionDatabases[path]
	if database == nil {
		database = &ahdHTTPSessionDatabase{path: path}
		ahdHTTPSessionDatabases[path] = database
	}
	database.opens++
	return database
}

func ahdHTTPSessionDatabaseAbandon(database *ahdHTTPSessionDatabase) {
	if database == nil {
		return
	}
	ahdHTTPSessionDatabasesMu.Lock()
	defer ahdHTTPSessionDatabasesMu.Unlock()
	if database.opens > 0 {
		database.opens--
	}
	if database.opens == 0 && ahdHTTPSessionDatabases[database.path] == database {
		delete(ahdHTTPSessionDatabases, database.path)
	}
}

// AhdHTTPReleasePersistentSessionStorage closes every persistent-session
// connection. The next session operation opens the same file again. It does
// not delete committed rows, and it does not close ordinary SQLite.open
// databases. It runs when the last actually running HTTP server stops and
// its handlers have returned, or when the last evaluator execution finishes,
// and no owner remains. An explicit test may call it directly.
func AhdHTTPReleasePersistentSessionStorage() {
	if err := AhdHTTPReleasePersistentSessionStorageErr(); err != nil {
		fmt.Fprintf(os.Stderr, "ahdcode: closing persistent session storage: %v\n", err)
	}
}

// AhdHTTPReleasePersistentSessionStorageErr closes every persistent-session
// connection and returns the first close error, if any.
func AhdHTTPReleasePersistentSessionStorageErr() error {
	return ahdHTTPClosePersistentSessionStorage()
}

func ahdHTTPTestClosePersistentStores() error {
	return ahdHTTPClosePersistentSessionStorage()
}

func ahdHTTPCloseDatabase(database *ahdHTTPSessionDatabase) error {
	if database == nil {
		return nil
	}
	database.mutex.Lock()
	defer database.mutex.Unlock()
	if database.handle == "" {
		return nil
	}
	err := SQLiteClose(database.handle)
	if err == nil || ahdSQLiteHelperFailed() {
		database.handle = ""
	}
	return err
}

func ahdHTTPClosePersistentSessionStorage() error {
	ahdHTTPSessionDatabasesMu.Lock()
	databases := make([]*ahdHTTPSessionDatabase, 0, len(ahdHTTPSessionDatabases))
	for _, database := range ahdHTTPSessionDatabases {
		databases = append(databases, database)
	}
	ahdHTTPSessionDatabasesMu.Unlock()
	var first error
	for _, database := range databases {
		if err := ahdHTTPCloseDatabase(database); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func ahdHTTPAttachDatabaseScope(scopeID uint64, database *ahdHTTPSessionDatabase) {
	if database == nil {
		return
	}
	ahdHTTPLifetimeMu.Lock()
	defer ahdHTTPLifetimeMu.Unlock()
	ahdHTTPAttachDatabaseScopeWhileLocked(scopeID, database)
}

func ahdHTTPAttachDatabaseScopeWhileLocked(scopeID uint64, database *ahdHTTPSessionDatabase) {
	if database == nil {
		return
	}
	if scopeID == 0 {
		database.unscoped = true
		return
	}
	scope := ahdHTTPEvaluatorScopes[scopeID]
	if scope == nil {
		database.unscoped = true
		return
	}
	if database.scopes == nil {
		database.scopes = make(map[uint64]int)
	}
	database.scopes[scopeID]++
	if scope.databases == nil {
		scope.databases = make(map[*ahdHTTPSessionDatabase]int)
	}
	scope.databases[database]++
}

func ahdHTTPDetachDatabaseScope(scopeID uint64, database *ahdHTTPSessionDatabase) {
	if database == nil || scopeID == 0 {
		return
	}
	ahdHTTPLifetimeMu.Lock()
	defer ahdHTTPLifetimeMu.Unlock()
	if database.scopes != nil {
		if count := database.scopes[scopeID]; count <= 1 {
			delete(database.scopes, scopeID)
		} else {
			database.scopes[scopeID] = count - 1
		}
	}
	if scope := ahdHTTPEvaluatorScopes[scopeID]; scope != nil && scope.databases != nil {
		if count := scope.databases[database]; count <= 1 {
			delete(scope.databases, database)
		} else {
			scope.databases[database] = count - 1
		}
	}
}

func ahdHTTPSessionDatabasesForStores(storeHandles []string) []*ahdHTTPSessionDatabase {
	if len(storeHandles) == 0 {
		return nil
	}
	ahdHTTPSessionStoresMu.Lock()
	defer ahdHTTPSessionStoresMu.Unlock()
	seen := make(map[*ahdHTTPSessionDatabase]struct{}, len(storeHandles))
	databases := make([]*ahdHTTPSessionDatabase, 0, len(storeHandles))
	for _, id := range storeHandles {
		store := ahdHTTPSessionStores[id]
		if store == nil || store.durable == nil || store.durable.db == nil {
			continue
		}
		db := store.durable.db
		if _, ok := seen[db]; ok {
			continue
		}
		seen[db] = struct{}{}
		databases = append(databases, db)
	}
	return databases
}

func ahdHTTPCloseIdleScopedDatabasesWhileLocked() {
	ahdHTTPSessionDatabasesMu.Lock()
	candidates := make([]*ahdHTTPSessionDatabase, 0, len(ahdHTTPSessionDatabases))
	for _, database := range ahdHTTPSessionDatabases {
		if !database.unscoped && len(database.scopes) == 0 && len(database.servers) == 0 {
			candidates = append(candidates, database)
		}
	}
	ahdHTTPSessionDatabasesMu.Unlock()
	for _, database := range candidates {
		if err := ahdHTTPCloseDatabase(database); err != nil {
			fmt.Fprintf(os.Stderr, "ahdcode: closing scoped persistent session storage: %v\n", err)
		}
	}
}

func ahdHTTPTestSetSessionStorageFault(message string) {
	if message == "" {
		ahdHTTPSessionStorageFault.Store(nil)
		return
	}
	ahdHTTPSessionStorageFault.Store(&ahdHTTPSessionFault{message: message})
}

// ahdHTTPDurableSessions is one cookie namespace on a shared session file.
type ahdHTTPDurableSessions struct {
	namespace     string
	db            *ahdHTTPSessionDatabase
	maxAgeSeconds int64
}

func nowFromStore(*ahdHTTPSessionStoreState) time.Time { return ahdHTTPTimeNow() }

func ahdHTTPSessionRaise(class *AhdClass, err error) {
	if err == nil {
		return
	}
	var schema *ahdHTTPSessionSchemaError
	if errors.As(err, &schema) || errors.Is(err, errHTTPSessionCorrupt) || errors.Is(err, errHTTPSessionStale) || errors.Is(err, errHTTPSessionTooLarge) {
		AhdRaiseClass(class, err.Error())
	}
	AhdRaiseClass(class, "HTTP session storage failed: "+err.Error())
}

func AhdHTTPPersistentSessions(class *AhdClass, path, cookieName string, maxAgeSeconds int64, secure bool, sameSite string) string {
	return AhdHTTPPersistentSessionsScoped(class, 0, path, cookieName, maxAgeSeconds, secure, sameSite)
}

// AhdHTTPPersistentSessionsScoped opens a persistent SessionStore associated
// with an evaluator execution scope so ending that scope can close the file
// independently of unrelated executions or servers.
func AhdHTTPPersistentSessionsScoped(class *AhdClass, scopeID uint64, path, cookieName string, maxAgeSeconds int64, secure bool, sameSite string) string {
	ahdHTTPRequireCookieName(class, cookieName)
	if maxAgeSeconds <= 0 {
		AhdRaiseClass(class, "HTTP session maxAgeSeconds must be greater than 0")
	}
	ahdHTTPRequireSameSite(class, sameSite, secure)
	ahdHTTPRequireSessionCookiePrefix(class, cookieName, secure)
	absolute, err := ahdHTTPSessionDatabasePath(path)
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	durable, err := ahdHTTPOpenDurableSessionsScoped(scopeID, absolute, cookieName, maxAgeSeconds)
	if err != nil {
		ahdHTTPSessionRaise(class, err)
	}
	id := fmt.Sprintf("%d", ahdHTTPSessionNextID.Add(1))
	ahdHTTPSessionStoresMu.Lock()
	ahdHTTPSessionStores[id] = &ahdHTTPSessionStoreState{
		cookieName: cookieName, maxAgeSeconds: maxAgeSeconds, secure: secure, sameSite: sameSite,
		durable: durable,
	}
	ahdHTTPSessionStoresMu.Unlock()
	return id
}

func ahdHTTPSessionDatabasePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("HTTP persistent session path must not be empty")
	}
	if strings.ContainsRune(path, 0) {
		return "", errors.New("HTTP persistent session path contains a NUL byte")
	}
	if path == ":memory:" {
		return "", errors.New(`HTTP persistent session path must be a file path, not ":memory:"`)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("HTTP persistent session path could not be resolved: %v", err)
	}
	if info, statErr := os.Stat(absolute); statErr == nil && info.IsDir() {
		return "", errors.New("HTTP persistent session path is a directory: " + absolute)
	}
	parent := filepath.Dir(absolute)
	parentInfo, err := os.Stat(parent)
	if err != nil || !parentInfo.IsDir() {
		return "", errors.New("HTTP persistent session directory does not exist: " + parent)
	}
	return absolute, nil
}

func ahdHTTPOpenDurableSessionsScoped(scopeID uint64, path, namespace string, maxAgeSeconds int64) (*ahdHTTPDurableSessions, error) {
	if fault := ahdHTTPSessionStorageFault.Load(); fault != nil {
		return nil, errors.New(fault.message)
	}
	database := ahdHTTPSessionDatabaseFor(path)
	ahdHTTPAttachDatabaseScope(scopeID, database)
	store := &ahdHTTPDurableSessions{namespace: namespace, db: database, maxAgeSeconds: maxAgeSeconds}
	database.mutex.Lock()
	err := database.ensureHandleLocked(store)
	database.mutex.Unlock()
	if err != nil {
		ahdHTTPDetachDatabaseScope(scopeID, database)
		ahdHTTPSessionDatabaseAbandon(database)
		return nil, err
	}
	return store, nil
}

func (database *ahdHTTPSessionDatabase) ensureHandleLocked(store *ahdHTTPDurableSessions) error {
	if database.handle != "" {
		return nil
	}
	if fault := ahdHTTPSessionStorageFault.Load(); fault != nil {
		return errors.New(fault.message)
	}
	handle, err := SQLiteOpen(database.path)
	if err != nil {
		return err
	}
	database.handle = handle
	if err := store.prepareLocked(); err != nil {
		_ = SQLiteClose(handle)
		database.handle = ""
		return err
	}
	return nil
}

func (store *ahdHTTPDurableSessions) prepareLocked() error {
	if _, err := SQLiteExecute(store.db.handle, "PRAGMA busy_timeout=5000", nil); err != nil {
		return err
	}
	// journal_mode returns a row, so it is a query. WAL is the file format.
	// Session calls still take BEGIN IMMEDIATE and wait on a writer.
	if _, _, err := SQLiteQuery(store.db.handle, "PRAGMA journal_mode=WAL", nil); err != nil {
		return err
	}
	if _, err := SQLiteExecute(store.db.handle, "PRAGMA synchronous=NORMAL", nil); err != nil {
		return err
	}
	err := store.transactLocked(func() error {
		for _, statement := range ahdHTTPSessionSchemaDDL {
			if _, err := SQLiteExecute(store.db.handle, statement, nil); err != nil {
				return err
			}
		}
		version, ok, err := store.schemaVersion()
		if err != nil {
			return err
		}
		if !ok {
			_, err = SQLiteExecute(store.db.handle,
				`INSERT INTO ahd_session_meta (key, value) VALUES (?, ?)`,
				[]string{ahdHTTPSessionText("schema"), ahdHTTPSessionText(ahdHTTPSessionSchemaVersion)})
			return err
		}
		if version != ahdHTTPSessionSchemaVersion {
			return &ahdHTTPSessionSchemaError{message: fmt.Sprintf(
				"HTTP session database schema version %s is not supported (supported version: %s). Automatic migration is not performed. Use a new database file or a runtime that understands this schema.",
				version, ahdHTTPSessionSchemaVersion)}
		}
		return nil
	})
	if err == nil {
		return nil
	}
	if ahdHTTPSessionUnique(err) {
		version, ok, readErr := store.schemaVersion()
		if readErr != nil {
			return readErr
		}
		if ok && version == ahdHTTPSessionSchemaVersion {
			return nil
		}
		if ok {
			return &ahdHTTPSessionSchemaError{message: fmt.Sprintf(
				"HTTP session database schema version %s is not supported (supported version: %s). Automatic migration is not performed. Use a new database file or a runtime that understands this schema.",
				version, ahdHTTPSessionSchemaVersion)}
		}
	}
	return err
}

var ahdHTTPSessionSchemaDDL = []string{
	`CREATE TABLE IF NOT EXISTS ahd_session_meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS ahd_sessions (
		namespace TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		values_json TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		PRIMARY KEY (namespace, token_hash)
	)`,
	`CREATE INDEX IF NOT EXISTS ahd_sessions_expiry ON ahd_sessions (expires_at)`,
	`CREATE TABLE IF NOT EXISTS ahd_session_revoked (
		namespace TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		PRIMARY KEY (namespace, token_hash)
	)`,
	`CREATE INDEX IF NOT EXISTS ahd_session_revoked_expiry ON ahd_session_revoked (expires_at)`,
}

func (store *ahdHTTPDurableSessions) open(storeHandle, presented string, now time.Time) (ahdHTTPSessionData, error) {
	session := ahdHTTPSessionData{StoreID: storeHandle, Values: map[string]string{}, Fresh: true}
	err := store.transact(func() error {
		if err := store.cleanup(now); err != nil {
			return err
		}
		if !ahdHTTPSessionIDWellFormed(presented) {
			return nil
		}
		hash := ahdHTTPSessionTokenHash(presented)
		revoked, err := store.revoked(hash, now)
		if err != nil {
			return err
		}
		if revoked {
			return nil
		}
		raw, expires, ok, err := store.load(hash)
		if err != nil {
			return err
		}
		if !ok || expires <= now.Unix() {
			return nil
		}
		values, err := ahdHTTPDecodeSessionValues(raw)
		if err != nil {
			return err
		}
		session.ID = presented
		session.Values = values
		session.Fresh = false
		return nil
	})
	return session, err
}

func (store *ahdHTTPDurableSessions) save(session ahdHTTPSessionData, now time.Time) error {
	if session.Values == nil {
		session.Values = map[string]string{}
	}
	encoded, err := json.Marshal(session.Values)
	if err != nil {
		return err
	}
	if len(encoded) > ahdHTTPSessionJSONMaxBytes {
		return errHTTPSessionTooLarge
	}
	expires := now.Add(time.Duration(store.maxAgeSeconds) * time.Second).Unix()
	hash := ahdHTTPSessionTokenHash(session.ID)
	return store.transact(func() error {
		if err := store.cleanup(now); err != nil {
			return err
		}
		revoked, err := store.revoked(hash, now)
		if err != nil {
			return err
		}
		if revoked {
			return errHTTPSessionStale
		}
		updated, err := store.update(hash, string(encoded), expires, now.Unix())
		if err != nil {
			return err
		}
		if updated {
			return nil
		}
		if !session.Fresh {
			return errHTTPSessionStale
		}
		err = store.insert(hash, string(encoded), expires)
		if err == nil {
			return nil
		}
		if !ahdHTTPSessionUnique(err) {
			return err
		}
		updated, err = store.update(hash, string(encoded), expires, now.Unix())
		if err != nil {
			return err
		}
		if !updated {
			return errHTTPSessionStale
		}
		return nil
	})
}

func (store *ahdHTTPDurableSessions) revoke(id string, now time.Time) error {
	if id == "" || !ahdHTTPSessionIDWellFormed(id) {
		return nil
	}
	return store.revokeHash(ahdHTTPSessionTokenHash(id), now)
}

// revokeLive revokes id only when its row still exists, is unexpired, and is
// not already revoked. The check and the revocation commit together. A second
// process waiting on BEGIN IMMEDIATE sees the revoked row and fails. Failure
// leaves the caller's session identifier unchanged.
func (store *ahdHTTPDurableSessions) revokeLive(id string, now time.Time) error {
	if id == "" || !ahdHTTPSessionIDWellFormed(id) {
		return errHTTPSessionStale
	}
	hash := ahdHTTPSessionTokenHash(id)
	return store.transact(func() error {
		if err := store.cleanup(now); err != nil {
			return err
		}
		revoked, err := store.revoked(hash, now)
		if err != nil {
			return err
		}
		if revoked {
			return errHTTPSessionStale
		}
		_, expiresAt, ok, err := store.load(hash)
		if err != nil {
			return err
		}
		if !ok || expiresAt <= now.Unix() {
			return errHTTPSessionStale
		}
		return store.revokeHashLocked(hash, now)
	})
}

func (store *ahdHTTPDurableSessions) revokeHash(hash string, now time.Time) error {
	return store.transact(func() error {
		if err := store.cleanup(now); err != nil {
			return err
		}
		return store.revokeHashLocked(hash, now)
	})
}

func (store *ahdHTTPDurableSessions) revokeHashLocked(hash string, now time.Time) error {
	expires := now.Add(time.Duration(store.maxAgeSeconds) * time.Second).Unix()
	if _, err := SQLiteExecute(store.db.handle, `
		INSERT INTO ahd_session_revoked (namespace, token_hash, expires_at)
		VALUES (?, ?, ?)
		ON CONFLICT(namespace, token_hash) DO UPDATE SET expires_at = excluded.expires_at`,
		[]string{ahdHTTPSessionText(store.namespace), ahdHTTPSessionText(hash), ahdHTTPSessionInt(expires)}); err != nil {
		return err
	}
	_, err := SQLiteExecute(store.db.handle,
		`DELETE FROM ahd_sessions WHERE namespace = ? AND token_hash = ?`,
		[]string{ahdHTTPSessionText(store.namespace), ahdHTTPSessionText(hash)})
	return err
}

func (store *ahdHTTPDurableSessions) forget(id string, now time.Time) error {
	return store.revoke(id, now)
}

func (store *ahdHTTPDurableSessions) count() (int, error) {
	store.db.mutex.Lock()
	defer store.db.mutex.Unlock()
	if store.db.handle == "" {
		return 0, errors.New("HTTP session storage is closed")
	}
	_, rows, err := SQLiteQuery(store.db.handle,
		`SELECT COUNT(*) AS total FROM ahd_sessions WHERE namespace = ?`,
		[]string{ahdHTTPSessionText(store.namespace)})
	if err != nil {
		return 0, err
	}
	if len(rows) != 1 || len(rows[0]) != 1 {
		return 0, errHTTPSessionCorrupt
	}
	value, err := SQLiteValueInt(rows[0][0])
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

func (store *ahdHTTPDurableSessions) transact(body func() error) error {
	store.db.mutex.Lock()
	defer store.db.mutex.Unlock()
	return store.transactLocked(body)
}

func (store *ahdHTTPDurableSessions) transactLocked(body func() error) error {
	if fault := ahdHTTPSessionStorageFault.Load(); fault != nil {
		return errors.New(fault.message)
	}
	if err := store.db.ensureHandleLocked(store); err != nil {
		return err
	}
	if _, err := SQLiteExecute(store.db.handle, "BEGIN IMMEDIATE", nil); err != nil {
		return err
	}
	if err := body(); err != nil {
		_, _ = SQLiteExecute(store.db.handle, "ROLLBACK", nil)
		return err
	}
	if _, err := SQLiteExecute(store.db.handle, "COMMIT", nil); err != nil {
		_, _ = SQLiteExecute(store.db.handle, "ROLLBACK", nil)
		return err
	}
	return nil
}

func (store *ahdHTTPDurableSessions) cleanup(now time.Time) error {
	stamp := ahdHTTPSessionInt(now.Unix())
	namespace := ahdHTTPSessionText(store.namespace)
	if _, err := SQLiteExecute(store.db.handle,
		`DELETE FROM ahd_sessions WHERE namespace = ? AND expires_at <= ?`,
		[]string{namespace, stamp}); err != nil {
		return err
	}
	_, err := SQLiteExecute(store.db.handle,
		`DELETE FROM ahd_session_revoked WHERE namespace = ? AND expires_at <= ?`,
		[]string{namespace, stamp})
	return err
}

func (store *ahdHTTPDurableSessions) schemaVersion() (string, bool, error) {
	_, rows, err := SQLiteQuery(store.db.handle,
		`SELECT value AS version FROM ahd_session_meta WHERE key = ?`,
		[]string{ahdHTTPSessionText("schema")})
	if err != nil {
		return "", false, err
	}
	if len(rows) == 0 {
		return "", false, nil
	}
	if len(rows) != 1 || len(rows[0]) != 1 {
		return "", false, errHTTPSessionCorrupt
	}
	value, err := SQLiteValueString(rows[0][0])
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (store *ahdHTTPDurableSessions) load(hash string) (string, int64, bool, error) {
	_, rows, err := SQLiteQuery(store.db.handle,
		`SELECT values_json AS values_json, expires_at AS expires_at
		 FROM ahd_sessions WHERE namespace = ? AND token_hash = ?`,
		[]string{ahdHTTPSessionText(store.namespace), ahdHTTPSessionText(hash)})
	if err != nil {
		return "", 0, false, err
	}
	if len(rows) == 0 {
		return "", 0, false, nil
	}
	if len(rows) != 1 || len(rows[0]) != 2 {
		return "", 0, false, errHTTPSessionCorrupt
	}
	raw, err := SQLiteValueString(rows[0][0])
	if err != nil {
		return "", 0, false, errHTTPSessionCorrupt
	}
	expires, err := SQLiteValueInt(rows[0][1])
	if err != nil {
		return "", 0, false, errHTTPSessionCorrupt
	}
	return raw, expires, true, nil
}

func (store *ahdHTTPDurableSessions) revoked(hash string, now time.Time) (bool, error) {
	_, rows, err := SQLiteQuery(store.db.handle,
		`SELECT token_hash AS token_hash FROM ahd_session_revoked
		 WHERE namespace = ? AND token_hash = ? AND expires_at > ?`,
		[]string{ahdHTTPSessionText(store.namespace), ahdHTTPSessionText(hash), ahdHTTPSessionInt(now.Unix())})
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (store *ahdHTTPDurableSessions) update(hash, values string, expires, now int64) (bool, error) {
	_, rows, err := SQLiteQuery(store.db.handle, `
		UPDATE ahd_sessions
		SET values_json = ?, expires_at = ?
		WHERE namespace = ? AND token_hash = ? AND expires_at > ?
		RETURNING token_hash AS token_hash`,
		[]string{
			ahdHTTPSessionText(values), ahdHTTPSessionInt(expires),
			ahdHTTPSessionText(store.namespace), ahdHTTPSessionText(hash), ahdHTTPSessionInt(now),
		})
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (store *ahdHTTPDurableSessions) insert(hash, values string, expires int64) error {
	_, err := SQLiteExecute(store.db.handle,
		`INSERT INTO ahd_sessions (namespace, token_hash, values_json, expires_at) VALUES (?, ?, ?, ?)`,
		[]string{
			ahdHTTPSessionText(store.namespace), ahdHTTPSessionText(hash),
			ahdHTTPSessionText(values), ahdHTTPSessionInt(expires),
		})
	return err
}

func ahdHTTPSessionValuesSize(values map[string]string) error {
	if values == nil {
		return nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return err
	}
	if len(encoded) > ahdHTTPSessionJSONMaxBytes {
		return errHTTPSessionTooLarge
	}
	return nil
}

func ahdHTTPDecodeSessionValues(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errHTTPSessionCorrupt
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, errHTTPSessionCorrupt
	}
	if values == nil {
		return nil, errHTTPSessionCorrupt
	}
	return values, nil
}

func ahdHTTPSessionTokenHash(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func ahdHTTPSessionIDWellFormed(id string) bool {
	if len(id) != ahdHTTPSessionIDLength {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

func ahdHTTPSessionUnique(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "UNIQUE") || strings.Contains(text, "constraint failed")
}

func ahdHTTPSessionText(value string) string { return SQLiteFromString(value) }

func ahdHTTPSessionInt(value int64) string { return SQLiteFromInt(value) }
