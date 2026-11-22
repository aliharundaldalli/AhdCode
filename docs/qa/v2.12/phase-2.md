# AhdCode v2.12.0 — Phase 2: Database Reliability & Compatibility QA

## 1. Initial repository state
- **Branch:** `v2.12-dev`
- **Starting commit:** `a37a4f956f466d1d20d6eabc778d2c94cd389894` (`Stabilize AhdCode v2.12 Phase 1 platform and file boundaries`)
- **Working tree at start:** Clean. Phase 1 LSP, private-file authorization, SQLite backup path fixes, and performance baseline were preserved unchanged.

## 2. Changes implemented
- **SQLite opening options (`SQLite.openWithOptions`):**
  - Wire protocol (`internal/sqliteproto/sqliteproto.go`): added `OperationOpenWithOptions = "openWithOptions"` plus `ReadOnly` and `BusyTimeoutMilliseconds` fields on `Request`.
  - Helper (`cmd/ahdsqlite/main.go`): implemented `openWithOptions(path, readOnly, busyTimeoutMilliseconds)` using `sqlite3.OPEN_READONLY` when `readOnly` is true (never creating missing files, rejecting writes) and `sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE` otherwise; enforced `busyTimeoutMilliseconds` in `[0, 300000]` (with `0` meaning immediate lock failure) and rejected `readOnly` on `":memory:"`. Kept `OperationOpen` (`SQLite.open`) unchanged at `5000` ms.
  - Runtime (`internal/backend/golang/ahdruntime/sqlite.go`): added `SQLiteOpenWithOptions` and `AhdSQLiteOpenWithOptions`; translated older helper rejections (`unknown SQLite helper operation "openWithOptions"` or `unknown Database handle`) into an explicit helper-version compatibility error so old helpers never silently ignore options.
  - Compiler & evaluator (`internal/semantic/sqlite_module.go`, `internal/backend/golang/sqlite_module.go`, `internal/evaluator/sqlite.go`): exported `SQLite.openWithOptions(path: String, readOnly: Bool, busyTimeoutMilliseconds: Int) -> Database`, lowered native calls, and wired session-relative path resolution in the evaluator.
- **Tests:**
  - `internal/semantic/sqlite_module_test.go`: updated exact exported surface and signature assertions.
  - `internal/backend/golang/ahdruntime/sqlite_options_test.go`: read-only open/non-creation/write-rejection/backup, invalid option validation, lock-contention `0` ms vs bounded timeout, and old-helper compatibility error coverage.
  - `internal/build/sqlite_options_test.go`: end-to-end native and evaluator parity for `SQLite.openWithOptions` and `Database.backupTo` with WAL, spaces, Unicode, `#`, and `%`.
  - `internal/backend/golang/ahdruntime/mysql_integration_test.go` and `internal/build/mysql_integration_test.go`: service-gated (`AHDCODE_TEST_MYSQL_*`) live MySQL runtime, concurrency, error-redaction, and native/evaluator parity suites.
- **Documentation:**
  - Updated `docs/SQLITE.md`, `docs/SQLITE_TR.md`, and `internal/initweb/docbundle/SQLITE.md` for `SQLite.openWithOptions`, keeping `TestDocumentationBundle` in sync.

## 3. SQLite backup verification
- Verified `TestSQLiteBackupIsAConsistentLiveSnapshot`, `TestSQLiteBackupFailuresLeaveNoBackup`, `TestSQLiteOpenWithOptionsReadOnlyAndValidation`, and `TestSQLiteOpenWithOptionsAndBackupNativeEvaluatorParity` in both ordinary and `-race` modes.
- Covered: WAL live snapshot with concurrent uncommitted writer, restoration and `PRAGMA integrity_check` (`ok`), rollback-journal self-containment, Windows drive paths, spaces, Unicode (`ü`), `#`, `%`, existing-destination refusal, missing directory/empty/`:memory:`/in-transaction restrictions, temporary `.ahdbackup-*` cleanup, read-only handle backup, and native/evaluator parity.

## 4. SQLite opening-options tests
- Verified `SQLite.open(path)` preserves read-write/create and 5-second busy timeout.
- Verified `SQLite.openWithOptions(path, true, ...)` fails without creating a missing file, reads committed rows, allows `backupTo`, and rejects `INSERT`/`CREATE TABLE` with a read-only `SQLiteError`.
- Verified lock contention: under an active `BEGIN` write transaction, a second connection with `busyTimeoutMilliseconds = 0` fails immediately (`database is locked`), while a bounded timeout succeeds after commit.
- Verified invalid combinations (empty path, read-only `":memory:"`, `-1`, `300001`) raise `SQLiteError`.
- Verified old-helper protocol rejection (`unknown Database handle`) is translated into an explicit helper-upgrade error rather than falling back to read-write.

## 5. MySQL live acceptance evidence
- Executed against a disposable local `mysqld.exe` 8.4.3 instance (`127.0.0.1:33307`, throwaway `ahd_qa_db` and non-root `ahd_qa` user, `AHDCODE_TEST_MYSQL_SECURITY=none`), then shut down and deleted its temporary datadir (`MYSQL_CLEANUP_DONE=True`).
- `TestMySQLIntegrationCRUDAndTypes` and `TestMySQLIntegrationTransactionsConcurrencyAndRedaction` passed in both ordinary (`0.267s`) and `-race` (`1.254s`, 0 races) modes: verified ping, parameterized CRUD (`Null`, `Int`, `Real`, `String`, `Binary`), affected rows, `lastInsertId`, duplicate column rejection, commit/rollback, 4-goroutine concurrent transactions under `-race`, post-close errors, and password redaction on auth failure.
- `TestMySQLIntegrationNativeAndEvaluatorAgree` and `TestMySQLAcceptanceProgramCompiles` passed in both ordinary (`11.626s`) and `-race` (`15.072s`, 0 races) modes.

## 6. PostgreSQL regression status
- PostgreSQL runtime, semantic, lowering, backend, and compile checks passed with zero regressions.
- The 5 live `TestPostgreSQLIntegration*` tests skipped cleanly on this host because `AHDCODE_TEST_POSTGRESQL_HOST` is unset (no local PostgreSQL server installed; live PostgreSQL 17/18 coverage runs in GitHub CI).

## 7. Native/evaluator parity results
- `TestSQLiteOpenWithOptionsAndBackupNativeEvaluatorParity`: identical output in native and evaluator modes (ordinary and `-race`).
- `TestMySQLIntegrationNativeAndEvaluatorAgree`: identical output in native and evaluator modes against live MySQL 8.4.3 (ordinary and `-race`).

## 8. Windows, Linux and macOS verification
- **Windows x64 (host):** Full Phase 2 ordinary, `-race`, SQLite backup/options, and live MySQL 8.4.3 suites executed natively and passed.
- **Linux amd64 & macOS arm64:** Cross-compiled `./cmd/ahdsqlite` cleanly (`CGO_ENABLED=0`, `CROSS_LINUX_EXIT=0`, `CROSS_DARWIN_EXIT=0`). Native Linux/macOS execution remains unverified on this Windows host (WSL is not installed).

## 9. Security and compatibility findings
- Private-file authorization boundaries (`HTTP.file`/`HTTP.download` `no-store` default, `HTTPPublicMount`, no BLOB file storage) were preserved and verified via `TestPrivateAvatar*`, `TestHTTPExplicitFile*`, and `TestHTTPPublicMount*`.
- MySQL authentication failure errors never disclose the supplied password (`Dispos4ble_Wrong_Secret`).
- **Windows Studio permission-mode test (`cmd/ahdcode/databases_test.go:49`, `TestStudioEnvBootstrapPreservesExisting`):** Inherited Windows fixture defect — `os.WriteFile(..., 0600)` on Windows reports `0666` via `os.Stat().Mode().Perm()` because Windows NTFS ACLs do not map POSIX group/other bits. Left out of Phase 2 scope per instruction not to expand into Studio development.

## 10. Exact normal and race test counts
- **8-package full suite (`./cmd/ahdsqlite`, `./internal/sqliteproto`, `./internal/semantic`, `./internal/lowering`, `./internal/backend/golang`, `./internal/evaluator`, `./internal/migrate`, `./internal/initweb`):**
  - Ordinary: **1,557 pass, 0 fail, 7 skip** (`TestWriteDogfoodAdminProject`, `TestWriteMVCHTTPProject`, `TestWriteCRUDHTTPProject`, `TestMVCAndCRUDApplicationsCompile`, `TestWebSymlinkConflict`, `TestMigrateRejectsSymlinkLock`, `TestMigrateRejectsASymlinkOutsideTheDirectory`).
  - `-race`: **1,557 pass, 0 fail, 7 skip, 0 data races**.
- **Focused `ahdruntime` DB/session/private-file suite (`^Test(SQLite|MySQL|PostgreSQL|PersistentSession|PrivateAvatar|HTTPExplicitFile|HTTPPublicMount)` without live DB env vars):**
  - Ordinary: **88 pass, 0 fail, 7 skip** (2 MySQL + 5 PostgreSQL service-gated integration tests).
  - `-race`: **88 pass, 0 fail, 7 skip, 0 data races**.
- **Live MySQL 8.4.3 & SQLite parity suites (disposable local MySQL 8.4.3):**
  - `ahdruntime` MySQL integration: **2 pass, 0 fail** (ordinary & `-race`, 0 races).
  - `internal/build` MySQL & SQLite parity: **3 pass, 0 fail** (ordinary & `-race`, 0 races).
- **Static checks:** `go vet` (0 issues), `gofmt -l` (0 issues after formatting), `git diff --check` (0 issues).

## 11. Remaining known limitations
- Live PostgreSQL 17/18 tests require a PostgreSQL server (`AHDCODE_TEST_POSTGRESQL_HOST`, covered by CI).
- Native Linux and macOS execution were not run on this Windows host.
- SQLite BLOB values, structured DB error codes, and streaming result sets remain deferred to v2.13 per `docs/plans/v2.12-roadmap.md`.
- `TestStudioEnvBootstrapPreservesExisting` remains an inherited Windows POSIX-mode test defect.

## 12. BLOCKER / MAJOR / MINOR classification
- **BLOCKER:** 0
- **MAJOR:** 0
- **MINOR:** 2
  1. `TestStudioEnvBootstrapPreservesExisting` fails on Windows due to POSIX `0600` mode check (pre-existing, out of Phase 2 scope).
  2. Native Linux/macOS execution and live PostgreSQL 17/18 execution require non-Windows / PostgreSQL-equipped runners (CI).

## 13. Phase 3 readiness verdict
**AhdCode v2.12.0 Phase 2 — PASS. Ready to proceed to Phase 3 (Standalone `ahdcode check` & Targeted Language Conformance).**
