# AhdCode v2.12.0 Phase 4 QA

## Starting state

The `v2.12-dev` worktree was clean at `e420023ac54cfee0e01cbce380021fa3d42aca44`. Review used the Phase 0 roadmap and the Phase 1–3 QA baselines. Phase 4 prioritized security, data integrity, and Windows reliability before measuring the editor change.

## Filesystem and private-file authorization

| Boundary | Evidence and responsibilities |
| --- | --- |
| Default exposure | The server dispatches only registered routes and mounts. The native two-user fixture keeps both avatars and its session database outside public roots. Direct requests for either resource returned 404. |
| Static and managed content | A static mount intentionally publishes its directory. A managed mount publishes only asset paths declared through the asset APIs. Both reject hidden segments and outside-root links, pin the registered root identity, and check the opened leaf. Windows junction and Unix symlink fixtures are selected by platform. The app must keep private files outside public roots and control who can change those roots. |
| Record authorization | The app obtains the owner from a server-side session and maps an allowed record to a fixed storage path before calling `HTTP.file`. The native fixture exercises both directions of cross-user access, altered URL and query identifiers, request headers, an unauthenticated client and a revoked login. Denials are generic, with no redirect, private bytes or storage paths. |
| Methods and validators | Authorized GET, HEAD, byte ranges and a matching ETag produced 200, 206 or 304 as appropriate. HEAD and 304 carried empty bodies. A cross-user or anonymous request carrying the same validator received the route denial. Invalid ranges preserved the file cache policy. |
| Cache behavior | `HTTP.file` and `HTTP.download` default to `no-store`. The fixture explicitly sets `no-store` on auth and ownership denials, and keeps the file default on success. General response cache policy and any deliberate file-cache override are app responsibilities. |
| PWA | The shipped worker pins explicit public paths and integrity hashes, installs with credentials omitted, refuses private or session-varying responses, and leaves avatar/API requests to the network. Four Node worker tests inspected simulated Cache Storage for public and private responses, including offline and session switching. Phase 1 records real Chrome acceptance. Phase 4 re-ran the worker harness and native HTTP application. |

Public mount errors and private file-open errors stay generic. Explicit file routes accept a trusted filesystem path from the application. Session identity, ownership, and the storage-path mapping belong in the server-side handler.

The native avatar and public-mount junction cases reject access outside the mount and to hidden targets. A separate Windows junction control could be opened through the OS with ordinary file reads, while this Go toolchain's `filepath.EvalSymlinks` returned a path error on the same junction. Probing an in-root HTTP alias consequently returned 404. This host verified fail-closed behavior. Successful Windows serving through an in-root junction and file-symlink coverage need a separate runner.

## Sessions and authentication

The expanded native avatar fixture uses temporary SQLite persistence and two separate cookie jars. It verifies session rotation at sign-in, HttpOnly and SameSite=Lax on the emitted cookie, reload of both users after a new process starts, denial of unauthenticated and cross-user access after restart, and revocation at sign-out. Its loopback HTTP transport deliberately uses a non-Secure cookie. The runtime suite separately checks Secure cookie settings, the SameSite=None requirement, and consistent attributes on deletion.

The fixture uses the bundled Web request context for per-session CSRF tokens and validates the token before sign-in and sign-out. A missing token and a token from the other user's session receive a generic 403. An invalid credential receives a generic 401. Server-side token verification precedes session mutation. Applications apply the CSRF check to their own state-changing routes.

The durable store and tests cover hashed session IDs at rest, expiry, rotation and revocation, rejection of stale commits, concurrent updates, fail-closed storage errors, and release after server/evaluator shutdown. The fixture and runtime suites used synthetic identities and temporary databases. Production data was outside the exercise.

## Database integrity

SQLite backup, restore and `integrity_check` coverage passed, including live WAL snapshots with an outstanding write and existing-destination refusal. Phase 2 opening options retained read-only non-creation, write rejection, immediate versus bounded lock waits, and the helper-version check. The native and evaluator parity fixture reopened the backup and observed the original rows.

The runtime and helper suites cover transactions and rollback, closing resources, and the persistent-session database lifecycle. MySQL and PostgreSQL interface checks and available non-live tests ran. Live tests require disposable service endpoints. None were configured for this phase, so the live cases were skipped. Phase 2's live MySQL report remains historical evidence.

## Windows permission fixes

- `TestStudioEnvBootstrapPreservesExisting` previously treated Windows file mode bits as a privacy guarantee. Studio now creates a missing `.env` with a protected Windows DACL for the current user, SYSTEM and Administrators. The test uses a parent with an inheritable broad read ACL and verifies the new file's private DACL. Unix retains the `0600` assertion. Exclusive creation also prevents another launcher from truncating a file that appeared after the existence check. An existing file and its permissions are preserved.
- `TestHTTPDownloadDestinationRules` previously used a Unix directory mode on Windows. This fixture now denies file creation with a Windows DACL and checks that the OS refuses creation before asserting that the HTTP client raises an error and leaves no partial file. Unix retains the permission-bit test for non-root callers. The runtime download implementation is unchanged.

The English and Turkish CLI guides document the Studio creation policy. The English starter bundle was regenerated with the repository synchronizer.

## Compiler and editor

`TestReferencesAndRenameUseTheRequiredFilesOwnSpans` reproduced a reference whose offset came from a CRLF source with a non-BMP character but whose path and text came from the entry file. References and rename now locate occurrences by their actual source file IDs. Declaration matching uses source paths already resolved during compilation, plus the declaration owner, name, and offsets. This removes filesystem lookups and repeated declaration resolution while scanning symbol occurrences.

Workspace traversal, volume-root rejection, ordering and budget limits, incomplete rename refusal, open overlays, closed-file UTF-16 positions and Phase 3 checker parity remain covered. Phase 3 recorded malformed file URIs on empty bundled-module diagnostics. This phase did not establish a user-visible effect or an incorrect source diagnostic, so the inherited URI behavior remains a tracked minor item.

## References latency

Windows amd64, Intel Core i5-12500H, Go 1.26.7, `GOMAXPROCS=4`. The Phase 1 benchmark sends real LSP requests and waits for their wire responses. Its 2-file and 25-file fixtures each used 20 one-request samples before and after. The same fixture generator was extended to 49 files for 10 samples before and after. All runs used the existing Go cache and an isolated local AhdCode cache directory. The p95 uses the nearest rank. Time is in milliseconds and allocation counts are per request.

| Fixture | Before median / p95 | After median / p95 | Before median allocations | After median allocations |
| --- | ---: | ---: | ---: | ---: |
| 2 files | 26.68 / 30.23 | 5.49 / 6.40 | 9,305.5 | 5,661 |
| 25 files | 1,042 / 2,756 | 75.04 / 97.26 | 273,851.5 | 121,183.5 |
| 49 files | 3,299 / 3,448 | 144.65 / 172.90 | 475,504.5 | 241,729 |

The medium and large baseline tails varied on this host. The allocation reductions and sustained median improvements justified keeping the small symbol-matching change. Workspace discovery and per-request compilation retain their existing scope and freshness rules.

~~~powershell
go test -run '^$' -bench '^BenchmarkPhase1/(Small|Medium)/LSPReferences$' -benchmem -benchtime=1x -count=20 -timeout 5m ./internal/build
go test -run '^$' -bench '^BenchmarkPhase4LargeReferences$' -benchmem -benchtime=1x -count=10 -timeout 5m ./internal/build
~~~

## Regression verification

Counts of named tests and subtests on Windows amd64, with `GOFLAGS=-buildvcs=false`, `GOMAXPROCS=4` and a fresh temporary `ahdsqlite.exe`. Affected Go suites ran with and without the race detector.

| Selection | Ordinary pass / fail / skip | Race pass / fail / skip |
| --- | ---: | ---: |
| CLI, SQLite helper, analysis, LSP, module, semantic, evaluator, REPL, sqliteproto, migrate, initweb, packaging | 1,993 / 5 / 26 | 1,993 / 5 / 26 |
| Complete runtime | 726 / 4 / 24 | 726 / 4 / 24 |
| HTTP, sessions, TLS development, WebSockets, database and authentication runtime subset | 230 / 0 / 11 | 230 / 0 / 11 |
| Focused native build integration | 58 / 0 / 3 | 58 / 0 / 3 |
| Final native avatar rerun with session database URL assertions | 1 / 0 / 0 | 1 / 0 / 0 |

~~~powershell
$core = './cmd/ahdcode ./cmd/ahdsqlite ./internal/analysis ./internal/lsp ./internal/module ./internal/semantic ./internal/evaluator ./internal/repl ./internal/sqliteproto ./internal/migrate ./internal/initweb ./internal/packaging' -split ' '
$native = '^Test(PrivateAvatar|HTTP|PersistentSession|WebSocket|BasicStarter|PWA|WebUI|WebV01[67]|WebDocument|SQLite|MySQL|PostgreSQL)'
$runtime = '^Test(HTTP|PersistentSession|MemorySession|SessionDeletion|WebSocket|DevHTTPS|SQLite|MySQL|PostgreSQL|Security|Identity|Password|PrivateAvatarAuthorizationNative|StudioEnvBootstrapPreservesExisting|ReferencesAndRenameUseTheRequiredFilesOwnSpans)'
go test -json -p 2 -count=1 -timeout 12m @core
go test -json -p 2 -count=1 -timeout 10m ./internal/backend/golang/ahdruntime
go test -json -p 2 -count=1 -timeout 8m -run $runtime ./internal/backend/golang/ahdruntime
go test -json -p 2 -count=1 -timeout 12m -run $native ./internal/build
go test -json -count=1 -timeout 5m -run '^TestPrivateAvatarAuthorizationNative$' ./internal/build
node --test internal/initweb/pwa_worker_test.cjs
~~~

The Go commands were repeated with `-race`. No race reports were emitted. The build test harness was instrumented in those runs and its generated applications used their ordinary native builds. Node passed 4/4. Verified changed Go formatting, affected-package `go vet` and `git diff --check`.

An isolated replay of `e420023` reproduced all nine failure records from the broader selections. Packaging's `TestResourceCopyRejectsSymlinks` failed during Windows link creation in four subtests, plus its aggregate failure record. The runtime failures were `TestResourceSymlinkStaysInsideTheCanonicalRoot` and `TestResourceSymlinkCannotLeaveTheRoot` (Windows link-creation privilege); `TestTLSInspectConnectionFailuresAreErrors` (generic handshake failure for a plain peer versus the specific expected wording); and `TestDiskInspectReportsConsistentSizes` (the Windows volume query rejects a regular-file path).

Core skips were existing link/privilege, Unix process, optional starter, macOS packaging and case-sensitive-filesystem fixtures. The focused runtime skips were the seven live database tests and four file or directory link cases without the required Windows privilege. The full runtime also skipped pre-existing optional helper and platform-dependent fixtures. Native skips were the two live DB parity fixtures and the external Web dogfood app.

## Cross-platform coverage and readiness

Linux amd64 and macOS arm64 test binaries for the CLI, analysis, LSP and runtime cross-compiled with `CGO_ENABLED=0`. Native Linux/macOS execution was unavailable on this Windows host. Existing live database and Windows privilege-dependent tests are reported as skipped where applicable.

**BLOCKER: 0. MAJOR: 0 open in the exercised Phase 4 scope.** The required-file editor bug and both requested Windows fixture failures are resolved. **MINOR:** The inherited Windows resource/packaging fixtures, Windows disk file-path defect, TLS fixture wording mismatch, and virtual URI behavior remain recorded. Phase 5 can begin with these baseline issues and native Unix, privilege-dependent link, and live database tests still on the release QA matrix. The readiness verdict applies to the exercised configurations.
