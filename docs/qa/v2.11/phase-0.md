# AhdCode v2.11 Phase 0 implementation and QA report

Phase 0 addresses HTTP and WebSocket callback retirement, failed startup
ownership, and a WebSocket clock test race. The branch is `v2.11-dev` at the
published `v2.10.0` commit `5239c32e2996463b51d0f8288b3a433741767cca`, with
unstaged Phase 0 changes. The compiler reports `AhdCode v2.10.0`.

Development took place in a separate Git worktree. The original Desktop
checkout had existing tracked modifications and three untracked executables.
The remote `main` also contained a post-release CI commit. The new branch
starts at the verified release tag, as requested. Phase 5 awaits independent
QA. No commit, push, tag, or release was made.

## Root causes and fixes

| Finding | Root cause and fix |
| --- | --- |
| 0.1 Callback admission | The callback checked `closing` before taking the lifetime lock. Retirement could therefore see zero callbacks and release scoped SQLite ownership just before an apparently accepted handler entered. `tryEnterCallback` now tests `closing` and increments the server and process callback counts under the same lock used by shutdown and retirement. HTTP routes, installed 404/500 handlers, and WebSocket callbacks use this gate. |
| 0.2 Bind failure | A failing `net.Listen` unlocked the lifecycle mutex while start ownership was still visible. Resetting route flags before releasing ownership also let a retry overlap the prior attempt's cleanup. The failed bind drops ownership before releasing lifecycle serialization. Routes stay frozen through ownership cleanup, then the handle is reusable. A pre-listen shutdown prevents the pending start from reaching bind. |
| 0.3 Clock fixture | HTTP 101 can reach the client before the server snapshots its connection clock. The test now exchanges an echo on each connection. Delivery through the read loop establishes that the snapshot has happened before the test changes the default clock. Close events and the short-clock observation interval verify that the first connection times out and the second stays live. The original clock durations and negative-observation window were preserved. |

## Lock order and database lifetime

- Route configuration and user calls keep the per-server request mutex. Start
  and callback admission acquire the lifetime mutex under it. Setting and
  clearing the callback goroutine identifier stays within the request mutex.
  User code can acquire the evaluator session lock as before.
- The lifecycle mutex serializes bind, publication, and shutdown claim.
  Failed bind and shutdown claim take the lifetime mutex under lifecycle.
  Retirement updates `closing` and ownership under lifetime, releases it,
  and then takes lifecycle to detach any remaining listener. The lifetime
  mutex never waits for either per-server mutex. The registry lock is used
  for lookup, removal, and the idle check. Registry operations release it
  before shutdown or retirement proceeds.
- Shutdown claims the listener and closes callback admission without taking
  the request mutex. Listener close and the graceful or forced HTTP shutdown
  run outside lifecycle and lifetime. Multiple shutdown calls still have a
  single winner.
- `AhdHTTPEvaluatorScopeEnd` takes the same lifetime mutex when deciding
  whether to hand off databases to a started server or an admitted callback.
  If the callback wins, its activity retains ownership through retirement,
  including forced shutdown. If retirement wins, the callback is refused.
  `finishCallback` runs after the request mutex is released and detaches a
  retired server's database ownership after its last callback. Other scope
  and server owners continue to protect a shared database.

The pre-listen and failed-bind tests use the private start and serve stages
to control ordering. The latter holds the route mutex while the real bind
fails on an occupied port, so it verifies that shutdown sees cleared
ownership even during rollback. It then repeats the bind failure, reuses the
same handle, and checks listener and file release. A channel-driven test
covers retirement on both sides of admission for an HTTP route, an installed
error page, and the WebSocket callback gate. It checks scoped ownership
during the held callback and deletes the database after completion.

## Changed files

| File | Change |
| --- | --- |
| `ROADMAP_3.0_TR.md` | Reordered the v2.11 PWA roadmap and recorded future dependencies and QA gates. |
| `internal/backend/golang/ahdruntime/http.go` | Serialized callback admission and startup/shutdown ownership. |
| `internal/backend/golang/ahdruntime/websocket_conn.go` | Used the common callback admission gate. |
| `internal/backend/golang/ahdruntime/http_lifecycle_test.go` | Exercised the real startup transition in the pre-listen shutdown test. |
| `internal/backend/golang/ahdruntime/http_phase0_test.go` | Added controlled bind rollback, retry, callback admission, and SQLite release tests. |
| `internal/backend/golang/ahdruntime/websocket_test.go` | Added readiness synchronization to the connection clock test. |
| `docs/qa/v2.11/phase-0.md` | Recorded the implementation and verification evidence. |

## Windows verification

Windows 11 Home 64-bit build 26300, Go 1.26.7 windows/amd64, and
MSYS2 UCRT64 GCC 16.2.0.

| Run | Pass | Fail | Skip |
| --- | ---: | ---: | ---: |
| Four critical tests, 12 repetitions, ordinary | 84 | 0 | 0 |
| Four critical tests, 12 repetitions, race | 84 | 0 | 0 |
| Focused ordinary | 174 | 1 | 1 |
| Focused race | 174 | 1 | 1 |
| Complete migration package, ordinary | 118 | 0 | 2 |
| Complete migration package, race | 118 | 0 | 2 |

The race logs contain zero data-race reports. The new tests completed
without deadlock and verified that their SQLite files could be removed.
A host-wide process query after the suites found zero running
`ahdsqlite.exe` processes. The CLI version smoke returned `AhdCode v2.10.0`.

The test event counts include subtests. The HTTP runtime, evaluator, build,
and documentation results above are filtered. The two migration runs cover
the entire migration package. Coverage includes graceful and forced
shutdown, scoped and shared persistent sessions, HTTP/WebSocket callbacks,
evaluator sessions, native HTTP parity, template rendering, the Basic
starter, and the executable migration example. The Windows runs do not
establish macOS, Linux, or browser behavior.

- The only focused failure in either mode was
  `TestHTTPSessionNativeRelocatesWithoutToolchain`. Its Windows fixture
  writes the relocated binary without an `.exe` suffix and fails at
  `exec.Command.Start`. Both prior v2.10 broad QA logs record the same failure
  at `internal/build/http_session_test.go:336`. The fixture was left as-is.
- Windows denied symlink creation for
  `TestTemplateWindowsLeafOpenRejectsReplacement/file_symlink`,
  `TestMigrateRejectsSymlinkLock`, and
  `TestMigrateRejectsASymlinkOutsideTheDirectory`. The Windows junction
  replacement case passed.
- The original `main` worktree still has its pre-existing modifications and
  binaries. Its post-release CI commit `3577cbf` was outside the requested
  branch base.

### Commands

```powershell
$env:GOFLAGS = '-buildvcs=false'
$critical = '^(TestHTTPBindFailureClearsOwnershipBeforeShutdownAndRetry|TestHTTPCallbackRetirementPreservesOnlyAdmittedOwnership|TestHTTPShutdownClaimsStartedServerBeforeListen|TestWebSocketClockIsImmutablePerConnection)$'
go test -json -count=12 -timeout 8m -run $critical ./internal/backend/golang/ahdruntime
go test -race -json -count=12 -timeout 8m -run $critical ./internal/backend/golang/ahdruntime
$focus = '^Test(HTTP(Endpoint|Shutdown|Publication|Registry|Wildcard|Nested|SameServer|BindFailure|CallbackRetirement|ErrorPage|Session|NativeAndEvaluator)|WebSocket|PersistentSession|MemorySession|Migrate(DoesNotDisturb|ExampleApp)|Templates|Template(Windows|Program)|HTTPEvaluator|Evaluator(HTTP|Persistent|CrossServer|Yield|NestedServer|Unstarted)|BasicStarter|DocumentationBundle)'
$packages = @('./internal/backend/golang/ahdruntime', './internal/evaluator', './internal/build', './internal/initweb')
go test -json -p 2 -count=1 -timeout 12m -run $focus @packages
go test -race -json -p 2 -count=1 -timeout 12m -run $focus @packages
go test -json -count=1 -timeout 8m ./internal/migrate
go test -race -json -count=1 -timeout 8m ./internal/migrate
go run ./cmd/ahdcode version
```

## Dependencies and compatibility for later phases

- **HTTPS.** The current `dev` orchestration starts a child HTTP server.
  `APP_PROTOCOL=https` describes a production public URL when a reverse
  proxy terminates TLS, and the current plaintext dev path refuses it.
  Phase 5 must decide whether TLS belongs at the development router or the
  child listener and validate the existing startup/shutdown contracts at
  that boundary. Assess Go's `ServeTLS` alongside a proven local certificate
  manager. Plan explicit browser trust, key permissions, SANs for the chosen
  DNS and IP names, renewal, and validation errors. Keep the public origin
  coherent for cookies, same-origin WebSocket checks, `wss://`, and the
  reload endpoints. Test ordinary HTTP and proxy termination as compatibility
  cases. [Go HTTP server](https://pkg.go.dev/net/http#Server.ServeTLS)
- **PWA.** Service workers require a secure context and registration on the
  correct origin and URL scope. The loopback or `localhost` development
  allowance does not imply the same treatment for an HTTP `.test` alias.
  Decide the manifest ID and scope under both root and subpath deployments.
  Verify worker MIME type, update activation, and cache key/version
  transitions. Restrict caching to public allowlisted assets and a static
  offline page. Keep authenticated HTML, private APIs, redirects containing
  user state, administrative pages, and writes on the network. Test install
  and standalone behavior on supported Chromium and WebKit targets because
  browser install UI and criteria differ. [Secure contexts](https://w3c.github.io/webappsec-secure-contexts/)
  · [Service Worker specification](https://w3c.github.io/ServiceWorker/spec/service_worker/)
  · [Chromium install guidance](https://developer.chrome.com/docs/modern-web-guidance/guides/pwa/install-web-app)
  · [WebKit install behavior](https://webkit.org/blog/17333/webkit-features-in-safari-26-0/)
- **Deployment.** Finalize worker URL and asset scope before deciding on a
  distribution mode. External `templates/` and `public/` require explicit
  relocation and working-directory tests. Treat the asset list as an
  allowlist to keep environment files and private keys out of deployment.

## Remaining risks and independent QA cases

1. Race shutdown against route body materialization, an installed error
   page, and each actual WebSocket callback stage. Verify zero new user calls
   after shutdown wins and successful persistence by a callback already
   admitted. Repeat with a callback that raises and with the same SQLite
   file held by another evaluator or server.
2. Interleave two addresses, a bind loser, cancellation before the
   listener appears, and a retry immediately after cleanup. Verify the
   shutdown boolean, registry membership, port reuse, and database cleanup
   for each ordering.
3. Alternate connection clocks across successive upgrades under the race
   detector. Keep the echo barrier and verify that a live second connection
   still follows its own interval. Go's graceful HTTP shutdown does not
   itself close hijacked WebSocket connections. Check explicit connection
   teardown and callback suppression separately. [Go HTTP shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
4. Repeat the ownership cases on another available OS and check the existing
   Windows relocation fixture separately. The SQLite deletion assertions
   cover the test files, while the post-run process check covers the named
   test helpers.
