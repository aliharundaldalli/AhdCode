# AhdCode v2.12 Phase 1 reliability and private-file QA

## Baseline and synchronization

The comparison point is the official `v2.11.0` commit
`e2505f8153c04b319f0dccafa3db4b0713597b50`. The annotated release tag
still points to that commit. The starting worktree held only the untracked
Phase 0 roadmap. `v2.12-dev` fast-forwarded through `d528f96` (canonical
and bundled docs) and `f0a17ee` (Studio Secure cookies, styled error
handlers and loopback port configuration). The roadmap was committed
unchanged in `c20bd77`. The existing local `main`, release candidate,
release tag and remote branch references were left in place. All validation
used disposable fixtures and loopback services.

## Reproduced defects and fixes

| Reproduction | Cause and correction |
| --- | --- |
| `TestReferencesExcludesDeclarationByDefault` timed out after 15 seconds on the tag. | The root guard matched Unix `/`. Windows normalized an implicit root to a volume path such as `C:\`, which fell through to `filepath.WalkDir`. The candidate limit also kept the walk running after enough files were found. |
| Workspace references and edits could use wrong ranges in files outside the active graph. | LSP converted those offsets using the active graph's text, which can be absent or stale. The results now carry the source snapshot that produced the span. Workspace queries refresh cached open-document snapshots after overlays change or close. |
| Both tagged SQLite backup tests failed with an invalid `/C:` path. | The helper prefixed a native drive path with a slash to create a file URI. SQLite's VFS received a path the Windows filesystem could not open. The backup API now receives the already-validated absolute native path, keeping spaces, Unicode, `#`, and `%` literal. The temporary snapshot, integrity check, transaction restrictions, atomic no-overwrite publication, and failure cleanup remain in effect. |
| The tagged SQLite test could leave its helper locked during cleanup. | Closing the last database does not stop the runtime's helper. The test fixture now closes its own helper's input and waits for process exit before removing the temporary executable. Runtime lifecycle is unchanged. |
| Two CLI build tests and native relocation tests used extensionless Windows paths. | Fixtures now use the executable naming policy. The Go toolchain's `GOROOT` fallback also uses that policy, fixing discovery when PATH is empty. The vendor composition fixture compares logical lines across LF and CRLF. |
| GUI resource-link and legacy packaging-link fixtures failed during symbolic-link creation. | This Windows host lacks the symbolic-link privilege. The build test did not reach product validation in those cases. New Windows junction fixtures exercise the workspace and HTTP boundaries without that privilege. Unix symbolic-link fixtures remain in scope on Unix. |

### Workspace traversal contract

The store keeps the entry directory and initialized workspace folders as
discovery roots, canonicalizes them, and excludes whole volumes, including
UNC share roots. Open overlays remain in the search even if a root is
excluded or does not exist. Discovery sorts each directory's entries and
deduplicates overlapping roots. It skips directory symlinks and junctions
found beneath a root, source symlink entries, and the existing ignored
directories. Opens use `os.Root` and verify directory identity.

One recursive discovery request may visit 512 directories, inspect 8,192
entries, and retain 1,024 candidate source files across its roots. An
oversized directory is refused after reading at most one sentinel entry
beyond the remaining allowance. This keeps selection deterministic without
allocating a listing of the entire directory. When an I/O error or budget
prevents complete discovery, references can return available locations and
rename returns the existing empty-result response without edits. UTF-16
conversion uses the exact source text for each range, including closed
files with non-BMP characters and CRLF. Compiler module resolution remains
separate from workspace enumeration.

## Private-file authorization audit

### Framework and application responsibilities

| Boundary | Verified behavior |
| --- | --- |
| Default exposure | The HTTP server serves registered routes and mounts. There is no implicit project web root or directory listing. `static` makes the declared directory tree public. `managed` narrows that exposure to declared assets under its mount. Place private data outside both kinds of root. |
| Public filesystem containment | Requests reject traversal and hidden segments. Files now open through the registered root, and the directory identity is checked before serving. Hidden alias targets are rejected. An ordinary link to a nonhidden file within the root can still serve it. Replacing the whole root requires registering the replacement, normally by restarting. Applications and deployments control which directories and links they publish. |
| Explicit file responses | `HTTP.file` and `HTTP.download` serve the trusted storage path chosen by the handler. They do not make authorization decisions. They default to `Cache-Control: no-store` on success and on file errors, including invalid ranges. Explicit cache policy remains available for deliberate public use. File-open failures emit generic client responses, and runtime error logs no longer print the stored path for those failures. |
| Identity and ownership | Cookies identify a server-side session. The app authenticates the user, rotates the session on sign-in, checks ownership before constructing a file response, and destroys the session at logout. The fixture passes fixed stored paths from a server-side mapping. URLs, query parameters, filenames, headers, hidden UI and JavaScript have no role in authorizing the requested record. |
| Caching, redirects and alternate paths | The authorized route and its HEAD registration share the same ownership check. Denial responses are generic and uncacheable. The fixture sends no redirect or private bytes on denial. The default PWA worker caches only explicitly pinned public assets and leaves private avatar routes to the network. It cannot compensate for an application adding private data to a public mount or overriding the cache policy. |

The new native fixture uses two synthetic users and different SVG files in a
temporary private directory. Two cookie jars, an anonymous client, and a
login switch cover owner access, cross-user requests, altered selectors and
headers, forged and logged-out cookies, HEAD, Range and conditional behavior,
direct URLs, alternate paths, traversal, and a directory link outside the
public mount. The client checks that denials disclose neither fixture
content nor storage paths. The cache-policy test also confirms that an
application's explicit public cache policy still works.

Two new tests ran unchanged against an extraction of `v2.11.0`. The old
mount served a replacement directory under a previously registered path.
The old explicit file response sent no cache policy. Both regression tests
passed after the fix. The Node worker suite tests bypass for both avatar
routes online and offline and inspects its simulated Cache Storage.
A separate Chrome 157.0.8103.79 run used a fresh profile and a generated
service worker controlling the actual test application. Alice could fetch
only her avatar. After logout and Bob login the same browser denied Alice's
avatar and returned Bob's. Both successful responses carried `no-store`.
With the browser offline the private request failed. Cache Storage contained
only the two pinned public icons before login and after the session switch.

## Ordinary and race verification

Commands ran on Windows x64 using Go 1.26.7 and Node 24.22.3.
Set `GOFLAGS=-buildvcs=false` for a linked worktree build and use fresh
local `ahdcode` and `ahdsqlite` builds for fixtures that consume
`AHDCODE_BIN` and `AHDCODE_SQLITE_RUNTIME`.

~~~powershell
$env:GOFLAGS='-buildvcs=false'
$env:GOMAXPROCS='4'
go test -json -count=1 -timeout=15s -run '^TestReferencesExcludesDeclarationByDefault$' ./internal/lsp
go test -json -count=1 -run '^TestSQLiteBackup(IsAConsistentLiveSnapshot|FailuresLeaveNoBackup)$' ./internal/backend/golang/ahdruntime
$language = './internal/analysis ./internal/lsp ./internal/module ./internal/semantic ./internal/formatter ./internal/evaluator ./internal/migrate ./internal/initweb ./internal/studio ./cmd/ahdsqlite' -split ' '
go test -json -p 3 -count=1 -timeout 8m @language
go test -race -json -p 3 -count=1 -timeout 8m @language
$core = './internal/build ./internal/backend/golang/ahdruntime ./internal/packaging ./cmd/ahdcode' -split ' '
$focus = '^Test(PrivateAvatar|HTTP(File|ExplicitFile|PublicMount|Static|Managed|Shutdown|Session|NativeAndEvaluator|ErrorPage)|PersistentSession|WebSocket|BasicStarter|PWA|WebPackage|WebAssets|WebPWA|DevHTTPS|DevWatcher|BuildProduces|RunAndBuild|.*Relocat|SQLiteBackup|SQLiteRuntime|CodesAndMySQLCompose|Studio)'
go test -json -p 3 -count=1 -timeout 7m -run $focus @core
go test -race -json -p 3 -count=1 -timeout 9m -run $focus @core
node --test internal/initweb/pwa_worker_test.cjs
~~~

The complete ten-package language and tools selection passed **2,082**
tests and subtests in each mode, with nine skips per mode. The skipped cases
were three opt-in starter exporters, three module link fixtures, two
migration file-link fixtures, and the starter symlink-conflict fixture.
The complete LSP suite passed, including the former timeout and closed-file
UTF-16 coverage.

The first core selections each recorded **199 passes, 9 skips, and 3
failure records**. The link-privilege, case-sensitive-filesystem, and
unstaged LaTeX fixture skips were already conditional in the tests. The
failures were the CRLF-dependent `TestCodesAndMySQLComposeOneVendoredWorkspace`,
the missing Windows toolchain suffix exposed by
`TestPlotOnlyProgramFindsTheInstalledDialogHelperAfterRelocation`, and
`TestStudioEnvBootstrapPreservesExisting` comparing a Windows mode to
Unix `0600`. The first two were fixed. The final targeted normal and race
reruns each passed nine tests, including both relocated Plot and Plain
cases, both SQLite regressions, and the private-file audit. The Studio mode
assertion remains an inherited Windows fixture issue. A separate native
Studio smoke test passed the dashboard, configured loopback port, styled
404, `no-store`, and Secure session-cookie checks. The Node worker suite
passed all four tests. No data races were reported.

The final targeted filter was
`^Test(FindGoToolchainUsesNativeSuffixInGOROOT|PlotOnlyProgramFindsTheInstalledDialogHelperAfterRelocation|PlainProgramStillCarriesNoDialogHelperHintWhenInstalled|CodesAndMySQLComposeOneVendoredWorkspace|PrivateAvatarAuthorizationNative|HTTPPublicMountKeepsRegisteredRootAndBlocksHiddenTargets|HTTPExplicitFileCachePolicy|SQLiteBackup)`
for `./internal/build ./internal/backend/golang/ahdruntime`, with
`-p 2 -count=1 -timeout 4m` in both modes. The pre-fix platform filter
also explicitly ran `TestResourceSymlink` and
`TestResourceCopyRejectsSymlinks` on the tag, where they failed during
fixture setup. Native Windows junction tests passed on this branch.

The POSIX volume-root fixture passed in Node's WASI sandbox. An initial
WASI run reached the assertions and failed during temporary-directory
cleanup because the host does not implement directory enumeration for
that case. The root-rule fixture now uses lexical paths on POSIX and
passes. Linux amd64 and macOS ARM64 each compiled the analysis, LSP,
HTTP runtime, and SQLite helper test binaries. Native Linux/macOS
execution and the Unix symlink fixture remain unverified on this host.

## Performance baseline

Windows 11 x64 build 26300.9974, Intel Core i5-12500H, Go 1.26.7 and
`GOMAXPROCS=4`. The two synthetic fixtures have 2 and 25 physical
`.ahd` files respectively. Each required part adds 12 typed bindings.
The medium entry imports Web and renders a small HTMLNode. The SQLite
fixture uses a 1,000-row temporary database and a warmed helper process.
LSP timings include JSON-RPC framing and delivery of the entry diagnostics
or reference result. All results are per operation.

The warm run uses the existing Go compiler cache, an isolated Windows
AhdCode run cache, and 20 one-operation samples per metric. A run-cache
hit is primed before measurement. A miss bypasses AhdCode's executable
cache but retains the warm Go cache. The supplemental native builds use
a fresh `GOCACHE` per sample. Their smaller sample count makes the p95
an upper observation, not a stable estimate of the tail. Allocation counts
cover the benchmark process and exclude Go child processes and the SQLite
helper. The measurements provide a starting point for comparison on the
same host and workload.

~~~powershell
$env:GOFLAGS='-buildvcs=false'
$env:GOMAXPROCS='4'
# Point this at the ahdsqlite built from the current worktree.
$env:AHDCODE_SQLITE_RUNTIME='<absolute path to ahdsqlite.exe>'
# In this temporary benchmark shell, keep Go's existing cache and isolate
# AhdCode's per-user executable cache.
$env:GOCACHE = go env GOCACHE
$benchCache = Join-Path ([IO.Path]::GetTempPath()) ('ahd-v212-bench-' + [guid]::NewGuid().ToString('N'))
$env:LOCALAPPDATA = (New-Item -ItemType Directory $benchCache).FullName
go test -run '^$' -bench '^BenchmarkPhase1$' -benchmem -benchtime=1x -count=20 -timeout 20m ./internal/build
$env:AHDCODE_BENCH_COLD_GO='1'
go test -run '^$' -bench '^BenchmarkPhase1/(Small|Medium)/NativeBuild$' -benchmem -benchtime=1x -count=5 -timeout 20m ./internal/build
Remove-Item Env:AHDCODE_BENCH_COLD_GO
go test -run '^$' -bench '^BenchmarkPhase1/(Small|Medium)/LSPReferences$' -benchmem -benchtime=20x -count=1 -timeout 4m ./internal/build
~~~

Warm measurements in milliseconds and median in-process allocations per
operation. The p95 uses the nearest-rank sample.

| Operation | Small median / p95 (ms) | Small allocs | Medium median / p95 (ms) | Medium allocs |
| --- | ---: | ---: | ---: | ---: |
| Module resolution and semantic analysis | 1.745 / 1.989 | 4,746 | 33.305 / 37.760 | 128,799 |
| IR lowering and validation | 0.050 / 0.073 | 322 | 2.617 / 3.390 | 31,404 |
| Go generation | 70.496 / 77.163 | 414,362 | 86.684 / 90.818 | 527,895 |
| Prepare and close temporary workspace | 134.894 / 149.717 | 1,918 | 126.056 / 136.889 | 1,918.5 |
| Native build with warm Go cache | 1,199.289 / 1,362.329 | 408 | 1,222.162 / 1,313.249 | 407.5 |
| Run without AhdCode executable cache | 1,841.610 / 1,894.949 | 423,668 | 1,941.254 / 2,166.694 | 692,291 |
| Run with AhdCode executable cache hit | 100.067 / 110.569 | 421,428 | 152.253 / 174.173 | 690,062 |
| LSP edit to entry diagnostics | 4.733 / 4.978 | 5,560 | 45.666 / 52.499 | 131,856 |
| LSP references with an open entry | 24.037 / 25.323 | 9,290.5 | 815.898 / 1,012.550 | 273,484 |

| SQLite helper operation | Median / p95 (ms) | Median allocs | Median bytes |
| --- | ---: | ---: | ---: |
| Open and close an existing database | 0.912 / 1.651 | 14.5 | 2,144 |
| Bound primary-key query on the open database | 0.159 / 0.238 | 22 | 1,832 |

| Fresh Go cache native build | Samples | Median / p95 (ms) | Median allocs |
| --- | ---: | ---: | ---: |
| Small | 5 | 12,374.407 / 17,028.496 | 414 |
| Medium | 5 | 12,811.950 / 13,199.909 | 411 |

Twenty successive references requests in one initialized LSP session
per fixture measured **23.88 / 25.07 ms** median/p95 for Small and
**793.0 / 864.5 ms** for Medium. The Go benchmark reported an
average of 9,278 and 273,176 allocations per request respectively.

The medium fixture's references requests and native builds show the
largest elapsed costs in this sample. Cache hits avoid native build time
while retaining frontend and Go-generation allocations. The measurements
do not identify a code-level optimization. Phase 1 kept the existing cache
design and compiler pipeline.

## Files and Phase 2 handoff

Paths in the Phase 1 change set relative to the worktree. The two upstream commits and the Phase 0 roadmap have separate history.

~~~text
cmd/ahdcode/main_test.go
cmd/ahdsqlite/main.go
docs/HTTP.md
docs/HTTP_TR.md
docs/LSP.md
docs/LSP_TR.md
docs/qa/v2.12/phase-1.md
internal/analysis/definition.go
internal/analysis/discovery.go
internal/analysis/references.go
internal/analysis/rename.go
internal/analysis/store.go
internal/analysis/workspace.go
internal/analysis/workspace_test.go
internal/analysis/workspace_unix_test.go
internal/analysis/workspace_windows_test.go
internal/backend/golang/ahdruntime/http.go
internal/backend/golang/ahdruntime/http_private_test.go
internal/backend/golang/ahdruntime/sqlite_backup_test.go
internal/backend/golang/ahdruntime/sqlite_test.go
internal/build/codes_test.go
internal/build/collections_test.go
internal/build/executable_test.go
internal/build/http_client_test.go
internal/build/http_session_test.go
internal/build/phase1_bench_test.go
internal/build/pipeline_test.go
internal/build/plot_dialog_install_test.go
internal/build/private_avatar_test.go
internal/build/toolchain.go
internal/initweb/docbundle/HTTP.md
internal/initweb/docbundle/LSP.md
internal/initweb/pwa_worker_test.cjs
internal/lsp/references.go
internal/lsp/references_test.go
internal/lsp/rename.go
internal/lsp/uri_test.go
~~~

**Blocker, major, minor and readiness.** The volume-root timeout, partial rename, Windows backup, and public-mount and cache findings are resolved in the exercised configurations. There are zero open blocker or major findings in the covered scope. Open minor items are the Windows Studio mode assertion and fixture prerequisites listed above. Native Unix execution remains a validation limit. Phase 2 can begin on this branch with the recorded baseline and current release constraints. This commit retains the compiler version and contains no release or deployment changes.
