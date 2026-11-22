# AhdCode v2.12.0 Phase 3 QA

## Starting state

- Development branch: `v2.12-dev`, clean at `ae3754cfef01a87809a0c3663e52b1f88de7304d` (Phase 2).
- Phase 1 ancestor: `a37a4f9`. Published v2.11.0 baseline commit: `e2505f8153c04b319f0dccafa3db4b0713597b50`.
- Reviewed the v2.12 roadmap, Phase 1 and Phase 2 reports, module frontend, diagnostic codes and spans, build and CLI dispatch, editor analysis, formatter, both language specifications, and the Web and library references.
- Baseline failures were replayed against an archive of `ae3754c` outside the development worktree.

## Checker and diagnostic contract

`ahdcode check <entry.ahd> [--format text|json]` reads saved sources via the existing module resolver and loader. It runs the lexer, parser and semantic analyzer, including `bring` and `require` resolution. The command ends at the frontend. Build and run retain lowering, IR validation, Go generation, native compilation and execution. The LSP also consumes editor overlays. `format --check` continues to validate canonical formatting.

Exit codes are **0** for no source errors, **1** for source or entry input errors, and **2** for invalid usage. Warnings alone satisfy the existing frontend's `HasErrors` contract. The diagnostic model supports warnings, but the current lexer, parser and semantic frontend have no warning producer. Warning serialization and warning-only classification were exercised with a synthetic diagnostic.

- Text is the default. Diagnostics go to stdout as `path:line:column: severity CODE: message` with any compiler hint underneath. Text coordinates are 1-based and columns count UTF-16 code units. A locationless diagnostic prints the entry filename without coordinates. A clean check emits no text.
- JSON stdout contains one UTF-8 document with `schemaVersion: 1` and a `diagnostics` array. Each item has `severity`, `code`, `message`, `path` and `range`. An existing hint is emitted as `hint`. The successful empty form is `{"schemaVersion":1,"diagnostics":[]}`. Usage errors and output-write failures go to stderr. Usage errors have no JSON payload.
- JSON `range` has inclusive `start` and exclusive `end`, each with a zero-based `line` and a zero-based UTF-16 `character`. An input issue without a source span uses `null`. The LSP and checker share the same byte-offset conversion for LF and CRLF source.
- Local paths are relative to the resolved entry directory and use `/`. A path on a different volume falls back to its filename. Bundled paths use `<ahdcode>/`. Ordering is by path, range (absent first), severity, code, message, then hint. The checker omits excerpts and removes compiler-generated absolute paths from filesystem errors and dependency-cycle messages. Authored literals can still appear in compiler messages.
- The real Windows CLI was run from a directory with spaces, with `PATH` and `GOROOT` pointed at an empty scratch directory and a missing SQLite helper configured. Valid AhdCode that would write a file and open a database if executed returned an empty JSON array and left both directories free of side effects or build artifacts. Invalid inputs delivered JSON and an invalid format returned status 2 with stderr usage.

## Language conformance and parity

| Rule | Evidence and disposition |
| --- | --- |
| Nullable narrowing | A nullable parameter guarded by an early return works with subsequent arithmetic in the evaluator and native program. An unguarded nullable use reports `SEM011` with matching ranges. Existing semantic tests cover control-flow branches and reassignment. |
| Numeric widening and overloads | An `Int` argument safely feeds a `Real` parameter. Exact typed overloads and contextual selection of an overloaded callable match the spec. An unmatched overload set reports `SEM026`. Existing semantic tests cover ambiguous and return-type-only overloads. |
| Defaults | An omitted parameter supplies the declared default. The compiler also prefers fewer defaults when argument conversion ranks tie, as shown in `overloadScore.betterThan` and `TestFewerDefaultsWins`. Both specifications omitted that tie-breaker. Section 16 now states the existing behavior and the integration fixture pins it in native and evaluator execution. |
| Inferred callable signatures | An imported overloaded function selected against an inferred callback signature chooses the same `Int` implementation at runtime. Existing semantic tests cover incompatible and unconstrained signatures. |
| Built-ins and methods | `len("ü😀")` returns 2 and `len(5)` reports `SEM016`. Passing a string to `List<Int>.add` or to an `Int` function parameter reports `SEM004`. An unknown named parameter reports `SEM016`. |
| Web and Web.UI | Rendering `Web.UI.h2("ü😀")` via `HTML.render` produces the same markup. `Web.UI.h2(7)` and the wrong type for the optional `Web.text` status report `SEM004`. Windows native Web/UI rendering, generated Basic starter and persistent sessions passed their regression tests. |
| Graph errors and positions | Missing `bring` is `SEM028`, missing `require` is `SEM046`, and a cyclic `bring` graph is `SEM032`. Checker coverage includes parser errors, required-file diagnostics, relative names with spaces, CRLF and non-BMP characters. |

The valid two-file CRLF fixture is accepted by `check`, the native compiler and the wire-level LSP. Evaluator and native executable both print the same six output lines, including the Turkish default and rendered heading. Ten invalid cases compare compiler codes, paths, ranges, severity, message and hint between checker, native frontend and LSP. Invalid programs stop before IR and have no applicable evaluator or native execution result. The evaluator and native runtime use the same checked IR for valid inputs.

Existing LSP publications for clean bundled Web modules can use a virtual-looking `file://%3Cahdcode%3E/...` URI that its own URI parser rejects. The test excludes those empty bundled publications and compares all diagnostics for actual files. Standalone reports have an explicit bundled path spelling. LSP URI behavior was preserved.

## Windows verification

Go 1.26.7 on Windows amd64, with a local SQLite helper and `GOFLAGS=-buildvcs=false`, `GOMAXPROCS=4`. Counts below include named tests and subtests. Counts for focused reruns overlap the full selections. No race reports were emitted.

| Selection | Ordinary pass / fail / skip | Race pass / fail / skip |
| --- | ---: | ---: |
| Core compiler and tools, 17 packages, first run | 2,412 / 4 / 10 | 2,412 / 4 / 10 |
| Complete analysis and REPL packages after fixture corrections | 193 / 0 / 0 | 193 / 0 / 0 |
| Final complete CLI package | 122 / 1 / 14 | 122 / 1 / 14 |
| Final checker and parity subset | 22 / 0 / 0 | 22 / 0 / 0 |
| Focused native build, HTTP runtime, packaging | 366 / 1 / 16 | 366 / 1 / 16 |
| Private-file and SQLite repeat, with optional PostgreSQL parity | 4 / 0 / 1 | 4 / 0 / 1 |
| Generated documentation consistency and starter bundle | 10 / 0 / 0 | 10 / 0 / 0 |
| Generated MVC and CRUD native compile rerun | 3 / 0 / 0 | 3 / 0 / 0 |

Selected commands, with the same tests repeated using `go test -race`. Set `AHDCODE_SQLITE_RUNTIME` to a fresh build of `./cmd/ahdsqlite` in a temporary directory.

~~~powershell
$flags = '-json', '-p', '2', '-count=1', '-timeout', '12m'
$core = './internal/source ./internal/diagnostics ./internal/lexer ./internal/parser ./internal/module ./internal/semantic ./internal/analysis ./internal/lsp ./internal/formatter ./internal/lowering ./internal/evaluator ./internal/repl ./internal/backend/golang ./cmd/ahdsqlite ./internal/sqliteproto ./internal/migrate ./internal/initweb' -split ' '
go test @flags @core
go test @flags ./internal/analysis ./internal/repl
go test @flags ./cmd/ahdcode
go test @flags -run '^TestCheck' ./cmd/ahdcode
$focus = '^Test(PrivateAvatar|HTTP|PersistentSession|WebSocket|BasicStarter|PWA|Web|DevHTTPS|DevWatcher|BuildProduces|RunAndBuild|.*Relocat|SQLite|MySQL|PostgreSQL|NumericFundamentals|Function|NullableCallable|CodesAndMySQLCompose)'
go test @flags -run $focus ./internal/build ./internal/backend/golang/ahdruntime ./internal/packaging
$repeat = '^Test(PostgreSQLIntegrationNativeAndEvaluatorAgree|PrivateAvatarAuthorizationNative|HTTPExplicitFileCachePolicy|HTTPPublicMountKeepsRegisteredRootAndBlocksHiddenTargets|SQLiteOpenWithOptionsAndBackupNativeEvaluatorParity)$'
go test @flags -run $repeat ./internal/build ./internal/backend/golang/ahdruntime
go test @flags -run 'Bundle|Document' ./internal/initweb
# Set AHDCODE_BIN to the freshly built Phase 3 CLI for this follow-up.
go test @flags -run '^TestMVCAndCRUDApplicationsCompile$' ./internal/initweb
node --test internal/initweb/pwa_worker_test.cjs
go vet -p 2 ./cmd/ahdcode ./internal/source ./internal/lsp ./internal/analysis ./internal/repl ./internal/initweb ./internal/build ./internal/backend/golang/ahdruntime
~~~

- The initial core failures were `TestSQLiteNamespaceMembersCarryRealSignatures`, `TestPlotRenderAndSaveParityInThePersistentREPL`, `TestSQLiteDatabaseSurvivesSuccessiveREPLEntries` and `TestSQLiteFileDatabaseIsReadBackInAFreshREPL`. The first used the pre-Phase 2 SQLite namespace expectation. The other three tried to launch test helpers without the Windows executable suffix. All four passed after revising the fixtures. There were no compiler or runtime changes for those fixes.
- Two inherited Windows assertions still fail in both modes. `TestStudioEnvBootstrapPreservesExisting` expects POSIX mode `0600`. `TestHTTPDownloadDestinationRules` assumes that mode `0500` makes a directory unwritable on Windows. Both reproduced in the archived Phase 2 source. The failures are Windows fixture limitations and remain visible.
- Core skips were three module symlink tests (`TestFileResolverCanonicalizesAliases`, `TestPhysicalModuleAliasIsAnalyzedOnce`, `TestRequireSymlinkEscapeRejected`), two migration symlink tests, the starter symlink-conflict test, three opt-in starter exports, and the MVC/CRUD native compile test gated on `AHDCODE_BIN`. The latter passed in both follow-up runs with the Phase 3 CLI.
- CLI skips were three Windows HTTPS credential symlink cases and eleven run-control tests whose existing fixtures start `/bin/sleep`. Other Windows HTTPS and CLI tests completed.
- Native skips were four symlink cases (`TestDevHTTPSWindowsCredentialOpenRejectsReplacement/file_symlink`, `TestHTTPStaticLeafSymlinkEscapeRejected`, `TestHTTPStaticDirectorySymlinkEscapeRejected`, `TestHTTPStaticSymlinkStayingInsideRootStillWorks`), the case-sensitive `TestWebAssetsRejectCaseCollisions`, nine live MySQL/PostgreSQL cases, `TestPDFNativeExecutableRelocatesWithoutSourceOrRuntimeBundle` (staged TeX required), and `TestWebV016ScratchDogfoodCompile` (external project required).
- Phase 1's `TestReferencesExcludesDeclarationByDefault`, `TestPrivateAvatarAuthorizationNative`, `TestHTTPPublicMountKeepsRegisteredRootAndBlocksHiddenTargets` and `TestHTTPExplicitFileCachePolicy` passed. Phase 2's SQLite backup and open-option runtime suite and `TestSQLiteOpenWithOptionsAndBackupNativeEvaluatorParity` passed. Node 24.22.3 passed all four PWA worker tests, including Cache Storage isolation for user and session responses.
- Linux amd64 and macOS arm64 CLI cross-builds passed with `CGO_ENABLED=0`. Native Linux and macOS execution was unavailable on this Windows host. WSL is not installed. Live databases were not provisioned in Phase 3. Phase 2's live MySQL result remains historical evidence. No browser session was required for the checker and no new browser validation is claimed.
- Ran the repository documentation synchronizer (72 bundled English documents), `gofmt -l` on the changed Go files, `go vet` on the affected packages above, and `git diff --check`.

## Performance observation

Windows amd64, Intel Core i5-12500H, Go 1.26.7 and `GOMAXPROCS=4`. Used a two-file CRLF fixture (3 lines, 105 bytes). `Main.ahd` contains `from Library bring twice` and `write(twice(21))`. `Library.ahd` contains `twice: Function := (value: Int) -> Int { return value * 2 }`. Time wall-clock process startup through exit with both streams captured. Warm up each command once, preserve the existing Go cache, then alternate `ahdcode check Main.ahd --format json` and `ahdcode build Main.ahd -o built.exe` 15 times. The native timing ends when compilation returns. The p95 is the nearest-rank value.

| Command | Iterations | Median | p95 |
| --- | ---: | ---: | ---: |
| Checker | 15 | 24.3 ms | 31.3 ms |
| Native build | 15 | 590.4 ms | 721.5 ms |

This local comparison measures wall-clock latency for two different operations under normal host load. The Phase 1 baselines track compiler stage timings and allocation counts separately.

## Changed files

| Files | Purpose |
| --- | --- |
| `cmd/ahdcode/main.go`, `cmd/ahdcode/check.go` | Register the checker and implement its frontend invocation, output and exit contract. |
| `cmd/ahdcode/check_test.go`, `cmd/ahdcode/main_test_setup_test.go` | Exercise CLI, JSON and parity, including an isolated no-Go subprocess. |
| `internal/source/utf16.go`, `internal/lsp/position.go` | Share the existing editor position conversion with the checker. |
| `internal/analysis/sqlite_test.go`, `internal/repl/repl_test.go`, `internal/repl/sqlite_test.go` | Pin Phase 2 SQLite completion and repair native helper filenames in test fixtures. |
| `AHDCODE_LANGUAGE_SPEC_v0.1.md`, `AHDCODE_LANGUAGE_SPEC_v0.1_TR.md`, `docs/CLI.md`, `docs/CLI_TR.md` | Document the existing overload rule and the checker. |
| `internal/initweb/docbundle.go`, `internal/initweb/docbundle/AHDCODE_LANGUAGE_SPEC_v0.1.md`, `internal/initweb/docbundle/CLI.md` | Synchronize the generated bundle and its manifest order. |
| `docs/qa/v2.12/phase-3.md` | Record scope, exact checks and the Phase 4 handoff. |

## Findings and Phase 4 readiness

- **BLOCKER: 0. MAJOR: 0** in the exercised Phase 3 scope.
- **MINOR:** Two inherited Windows fixture failures remain reproducible. The inherited LSP virtual-URI issue remains, and the shared editor index recognizes LF and CRLF. The lexer also accepts legacy bare CR; editor coordinates for that line ending follow existing LSP behavior.
- **Verification limits:** Native Unix runners and live MySQL/PostgreSQL coverage remain for cross-platform QA. Lowering, linking, execution, and availability of runtime services require separate validation.

**Ready for Phase 4 on the recorded evidence.** Preserve the Phase 1 measurement method when selecting performance work. Windows baseline failures and unexecuted platform or service tests remain open in QA.
