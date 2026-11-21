# AhdCode v2.11.0 Phase 7 packaging and release QA

## 1. Starting point

The existing `v2.11-dev` worktree started at
`5239c32e2996463b51d0f8288b3a433741767cca` with the version set to
`AhdCode v2.10.0`. Its index was clean. The worktree contained 54 modified
tracked paths, 24 tracked deletions, and 31 untracked paths. These included the
approved Phase 0 lifecycle, Phase 5 HTTPS, public Templates API removal, and
Phase 6 PWA changes. Phase 6 independent QA recorded 332 passes and four skips
in each Go mode and four worker passes. It left HEAD support and native Caddy
execution for follow-up.

Before changing packaging, the baseline filter below ran across
`./internal/packaging ./internal/initweb ./internal/build` in ordinary and
race modes. Both recorded 17 test/subtest passes and one macOS-specific skip.
The existing `TestResourceCopyRejectsSymlinks` failed at file creation in all
four subtests because the Windows account could not create symlinks.
The four JavaScript worker tests passed before the change.

## 2. Deployment design and commands

```sh
ahdcode build app.ahd -o Site
ahdcode package app.ahd --web --name Site --output releases/1
ahdcode package app.ahd --web --pwa --name Site --output releases/pwa-1
```

Use the third command after wiring the generated `ahdcode pwa init` module
into the app. `--pwa` without `--web` is a usage error with exit code 2.
The existing packaging command compiles the `.ahd` source graph and selects
its required platform helpers. Web mode adds the reviewed `public/` tree.
PWA mode also adds `PWA/app.webmanifest`, `PWA/worker.js`, and
`PWA/public/`. The integration module is compiled. Web code remains
`.ahd`, Web.UI, and HTMLNode.

Run the executable from its directory with deployment-provided
`APP_*`, `SERVER_*`, and applicable `DB_*` settings. Keep production
configuration and SQLite files in private storage. On macOS the working
directory is `Name.app/Contents/MacOS`. Use a fresh output for each Web
package. The CLI rejects existing deployment or archive destinations and
rolls back its staging changes if publication to the local directory fails.
Development HTTPS remains loopback-only. For production Caddy can terminate
HTTPS and proxy the application, assets, and worker through one origin.
[Packaging](../../PACKAGING.md) · [PWA](../../PWA.md) · [HTTPS](../../WEB.md#14-local-https)

The portable asset checks reject traversal, reserved device and stream
names, case-folded path collisions, unsupported types, hidden paths,
private directory names, links, junctions, and recognizable PEM keys.
Asset opening is rooted inside the approved directory and checks file
identity after opening. Manifest validation checks JSON, a display name,
and existing local nonempty image assets. Packaging preserves app routes
and uses the existing sorted archive order and fixed timestamps. A browser
still validates the actual PWA manifest, script, and image content. Custom
static roots require explicit deployment. Review public asset contents for
credentials before packaging.

## 3. Phase 7 file list

Paths are relative to the worktree. Several files already had earlier phase
edits. The CI and migration-test updates carry the reviewed patch from
`main` commit `3577cbf` into the working tree without changing history.
The two existing path tests now use illustrative user paths while retaining
coverage for spaces, Windows drive letters, and escaping.

```text
.github/workflows/ci.yml
README.md
README_TR.md
cmd/ahdcode/main.go
cmd/ahdcode/package_web_test.go
docs/CLI.md
docs/CLI_TR.md
docs/PACKAGING.md
docs/PACKAGING_TR.md
docs/PWA.md
docs/PWA_TR.md
docs/RELEASE_2.11.0.md
docs/RELEASE_2.11.0_TR.md
docs/qa/v2.11/phase-7.md
internal/ahdversion/version.go
internal/backend/golang/ahdruntime/latex_bundle_test.go
internal/build/executable_test.go
internal/initweb/docbundle.go
internal/initweb/docbundle/CLI.md
internal/initweb/docbundle/PACKAGING.md
internal/initweb/docbundle/PWA.md
internal/initweb/docbundle/README.md
internal/initweb/docbundle/RELEASE_2.11.0.md
internal/initweb/pwa/Integration.ahd
internal/migrate/migrate_qa_test.go
internal/packaging/packaging.go
internal/packaging/web.go
internal/packaging/web_test.go
```

## 4. Relocated application and browser acceptance

The Windows checks used generated projects with spaces and Unicode characters
in their original paths. Each production executable was launched from a
freshly extracted ZIP after moving the source directory. The source directory
and compiler were outside the packaged program's working directory.

| Fixture | Evidence |
| --- | --- |
| Basic and Empty without PWA | Home responded. The packaged stylesheet responded on its mounted route where configured. Worker and private configuration URLs returned 404. Basic rejected an invalid `APP_PROTOCOL` at startup. |
| Admin, MVC, and CRUD without PWA | Each archive included its required SQLite helper. Each app used a separately provisioned SQLite file with a space in the path. Dashboard access redirected before login and after logout. A valid CSRF-backed login opened the dashboard. |
| Basic PWA on Chrome 157.0.8103.79 | Real Chrome installed the app from its Windows package on a secure loopback context, opened the configured start URL, launched it in standalone display mode, and removed the test installation. |
| Basic PWA on Firefox 160.0.1 via Caddy 2.11.9 | A clean browser profile trusted the QA CA. The browser decoded both generated icons, registered the root-scoped worker, executed Bootstrap, and rendered Basic's Turkish routes and custom 404/500 pages. Package HEAD requests returned 200, `Cache-Control: no-cache`, and an empty body. |
| Caddy offline, session, and update | Signed in, rotated the session, signed out, and signed in as a different user. Only the two pinned icons entered the worker's cache. Stopping the actual proxy produced the offline page, and private and reload fetches failed. Restoring the proxy recovered the page. After updating the worker and closing the original tab, the old cache was removed and an unrelated cache survived. |
| Direct development HTTPS | A separate generated Basic project under an active worker reloaded in Firefox after an edit to `Pages/Home.ahd`. The browser observed generation 2. It accepted the QA CA without certificate-error overrides. |

The first Caddy test run navigated its old tab away and timed out waiting
for cache deletion. The distinct retry explicitly closed the old controlled
tab and passed. Browser profiles, Caddy state, and certificates were confined
to test directories. No system trust-store change was required.

The Go packaging regression also sends an encoded traversal request and
checks that `.env` and PWA source URLs are unavailable. Malformed JSON,
missing files, external or traversal icon references, non-image references,
and empty icons are rejected. An actual Windows junction was tested.
The permission-denied symlink creation cases are listed separately below.

## 5. Automated verification

Host Windows amd64, Go 1.26.7, Node 24.22.3. The commands use PowerShell.
`AHDCODE_BIN` and `AHDCODE_SQLITE_RUNTIME` point to fresh local builds of
`./cmd/ahdcode` and `./cmd/ahdsqlite` when required by a fixture.
Counts include subtests, as printed by the Go JSON stream.

```powershell
$env:GOFLAGS = '-buildvcs=false'
$env:GOMAXPROCS = '4'
$baseline = '^Test(PackageHoldsExactly|PackagingRejects|LeakGate|PackageCopiesImage|ResourceCopyRejects|PWA)'
$focus = '^Test(WebPackage|WebAssets|WebPWA|PWA|DevHTTPS|DevWatcher|DevReload|HTTP(Endpoint|Shutdown|Publication|Registry|Wildcard|Nested|SameServer|BindFailure|CallbackRetirement|ErrorPage|Session|NativeAndEvaluator)|WebSocket|PersistentSession|MemorySession|HTML(Text|Compose|Document|Preserves)|HTTPEvaluator|Evaluator(HTTP|Persistent|CrossServer|Yield|NestedServer|Unstarted)|BasicStarter|WizardBasic|MVCAndCRUDApplicationsCompile|AdminSQLite|MVCSQLite|DocumentationBundle|AllStartersReceiveDocumentationBundle|Migrate)'
$skip = '^TestHTTPSessionNativeRelocatesWithoutToolchain$'
$packages = './cmd/ahdcode ./internal/backend/golang/ahdruntime ./internal/build ./internal/evaluator ./internal/initweb ./internal/migrate ./internal/packaging' -split ' '
go test -json -count=1 -run $baseline ./internal/packaging ./internal/initweb ./internal/build
go test -race -json -count=1 -run $baseline ./internal/packaging ./internal/initweb ./internal/build
go test -json -p 3 -count=1 -run $focus -skip $skip @packages
go test -race -json -p 3 -count=1 -run $focus -skip $skip @packages
go test -json -count=1 ./internal/packaging ./cmd/ahdsqlite
go test -race -json -count=1 ./internal/packaging ./cmd/ahdsqlite
go test -race -json -count=1 -skip '^TestResourceCopyRejectsSymlinks$' ./internal/packaging ./cmd/ahdsqlite
node --test internal/initweb/pwa_worker_test.cjs
go test -json -count=1 -run '^Test(WebUIEscapesEveryTextEntryPoint|WebNotesNativeXSSAndSQLStayData|HTMLEvaluator|HTMLInlineAssetsAreDistinctFromEscapedText|LatexBundleURL|ExecutablePathKeepsSpacesAndDotsIntact)' ./internal/backend/golang/ahdruntime ./internal/build ./internal/evaluator
# Repeat the preceding command with -race for the race result.

$rootPackages = go list -e ./... | Where-Object { $_ -notin @('ahdcode/internal/build', 'ahdcode/tooling/distribution/windows') }
go test -json -p 3 -count=1 -timeout 8m @rootPackages
$vetPackages = go list -e ./... | Where-Object { $_ -ne 'ahdcode/tooling/distribution/windows' }
go vet @vetPackages
git -c core.safecrlf=false diff --check
```

The final focused ordinary and race commands passed all seven packages.
Each recorded 354 passing test/subtest results and five skips. The final
race run reported zero races. Both used the Windows CLI and SQLite helper
built from the candidate source ZIP. The five skipped tests are the four
permission-gated tests and the case-collision test listed below. The
extensionless relocation fixture was explicitly excluded with `-skip`.

The unqualified packaging and SQLite-helper runs recorded 45 passing
test/subtest results, one macOS skip, and five inherited symlink failure
records in each mode. The final packaging/SQLite race rerun, with the
symlink-creation fixture explicitly excluded, passed 44 and skipped two
platform-dependent tests. The supplementary escaping and fixture selection
recorded 11 passes and the same extensionless-output failure in ordinary
and race modes. All four worker tests passed again.

The broader ordinary run recorded 3,287 passes and 54 test/subtest skips.
Of its 20 failure records, 19 matched the v2.10.0 comparison, and one was
the corrected documentation line-ending mismatch. `go vet` passed
with the legacy installer wrapper excluded. `gofmt -l` over Phase 7 Go
files and `git diff --check` produced no findings after the final edits.

When C: ran out of space, temporary extraction copies were removed and the
acceptance runs used an isolated cache and temporary directories on D:.
The interrupted logs are excluded from the successful-run counts.

The focused selection covers HTTP and HTTPS lifecycle and shutdown, WSS
and origin policy, persistent sessions and migrations, custom errors,
native/evaluator consistency, live reload, and packaged HTTP. For starter
database setup the process checks used `TestWriteDogfoodAdminProject`,
`TestWriteMVCHTTPProject`, and `TestWriteCRUDHTTPProject` with temporary
destinations. Runtime environment settings were passed to the executable
and no private state was copied into either application archive.

## 6. Windows failures, skips, and coverage limits

- The known `TestHTTPSessionNativeRelocatesWithoutToolchain` fixture
  executes an extensionless relocated binary on Windows. Phase 5 reproduced
  it before and after the HTTPS change. It was explicitly excluded here.
  The relocated Windows package checks executed the actual `.exe` files.
- `TestResourceCopyRejectsSymlinks` failed before Phase 7 and in the
  unfiltered packaging checks in both modes. Its four subtests are
  `link_to_an_outside_file`, `link_to_an_outside_folder`,
  `link_that_stays_inside`, and `resources_folder_is_a_link`. The Win32
  privilege error occurs before the packager is invoked.
- The existing `TestDevHTTPSInvalidCredentials/linked_key`,
  `TestDevHTTPSWindowsCredentialOpenRejectsReplacement/file_symlink`,
  `TestMigrateRejectsSymlinkLock`, and
  `TestMigrateRejectsASymlinkOutsideTheDirectory` skipped for the same
  system privilege. The Windows junction regression ran.
- `TestWebAssetsRejectCaseCollisions` requires a case-sensitive filesystem.
  The default Windows volume cannot create the fixture's two distinct paths.
  The macOS package fixture `TestPackageHoldsExactlyWhatTheProgramNeeds`
  is platform-specific.
- An unqualified root `go list ./...` and `go vet ./...` hit the legacy
  Windows installer wrapper's `go:embed payload.zip`. That generated file
  is absent from this source-only worktree. The broader commands above
  explicitly omit that wrapper. The native compiler suite has known Windows
  fixture failures and long runtimes. This pass ran its focused cases.

The broader ordinary run also reported the failures below. Each of the
19 test/subtest failures listed in this table repeated on a temporary
extraction of committed v2.10.0 using `git archive HEAD`. The comparison
ran only the listed cases with `go test -json -p 3 -count=1 -run <pattern>`
on the affected packages. The doc bundle mismatch from the first final run
had a separate cause. It was corrected by matching the canonical and
embedded files' Windows line endings, and the source candidate already
contained identical normalized content.

| Package | Test and observed reason |
| --- | --- |
| `cmd/ahdcode` | `TestBuildProducesAnExecutable` and `TestRunAndBuildAcceptUnicodePathsWithSpaces` look for extensionless output. `TestStudioEnvBootstrapPreservesExisting` expects Unix permission bits. |
| `internal/backend/golang/ahdruntime` | `TestResourceSymlinkStaysInsideTheCanonicalRoot` and `TestResourceSymlinkCannotLeaveTheRoot` fail to create symlinks. `TestHTTPDownloadDestinationRules` and `TestTLSInspectConnectionFailuresAreErrors` fail their platform-specific error expectations. `TestSQLiteBackupIsAConsistentLiveSnapshot` and `TestSQLiteBackupFailuresLeaveNoBackup` fail on a Windows drive path. `TestDiskInspectReportsConsistentSizes` receives unavailable filesystem information. |
| `internal/localdev` | `TestDatabaseRegistryShape` differs on path serialization and file-mode expectations. |
| `internal/repl` | `TestPlotRenderAndSaveParityInThePersistentREPL`, `TestSQLiteDatabaseSurvivesSuccessiveREPLEntries`, and `TestSQLiteFileDatabaseIsReadBackInAFreshREPL` fail the REPL output assertions. |
| `internal/packaging` | `TestResourceCopyRejectsSymlinks` and its four subtests fail at Windows symlink creation. |

The supplementary native escaping fixture
`TestWebUIEscapesEveryTextEntryPoint` also uses an extensionless output
assertion and reports the actual `.exe` in both modes. The same eleven
inputs were compiled and evaluated in a separate Windows `.exe` smoke
check using the archive-built compiler.

## 7. Platform and artifact validation

| Target | Verification |
| --- | --- |
| Windows x64 | Source archive extracted, toolchain built, CLIs started, and all five application starters ran. Browser QA ran on Windows. |
| Linux x64 | CLI and SQLite helper cross-compiled from the extracted source. A Basic Web archive was inspected for an ELF executable, public stylesheet, and Bootstrap JavaScript. Linux execution was unavailable on the host. |
| Windows ARM64, Linux ARM64, macOS x64 and ARM64 | Existing packager target support. Phase 7 native execution and installs were unverified. |
| Mobile | Native installation and launcher behavior were unverified. |

The release policy in `tooling/distribution/build.py` marks v2.11.x as
source-only and retains the v2.7.0 signed installers. The current candidate
is a local source snapshot that includes the reviewed working-tree sources.
It omits this post-build report to avoid a recursive checksum. The official
source publication should be produced from the approved repository state.
The source ZIP can require a short extraction path on Windows because
vendored paths are deep.

The ZIP CRCs, archive formats and modes, and all 2,117 file hashes agree
between ZIP, TAR.GZ, and the source inventory. A repeat run reproduced
both archives byte for byte. The source secret-pattern audit passed after
reviewing the distribution scanner's own matcher and self-test literals.
The raw binary-targeted scanner flags those literals when scanned as source.
Application ZIP/TAR checks verified the source modules and private
configuration were outside the deployable archives.

The local release candidate is in the Phase 7 evidence directory.
Its `SHA256SUMS` contains these hashes.

```text
1a0107a95b8910b564e84118988fe9e5c9f911ecc667e3f5eed7c656318e0d00  AhdCode-2.11.0-source.tar.gz
6f9c7c7751c5afbdf7554921d79bffda08b6817d5445cb9ede760791121da1ed  AhdCode-2.11.0-source.zip
```

`source-inventory.json` contains the path and SHA-256 for every source
member. The repository scanner passed both application archives and four
validation binaries, including both CLIs' 33 expected AhdDataStudio files.
The following files in `validation/` are local QA inputs.

```text
3472a34d7a20900344752409e9bf84aa16fb826bffbcb72d8d04e47b2953c553  Basic-Web-linux-x64-QA.tar.gz
78ba03ef26182f903fe55ebe3289fd39dbb19983010db639443ad00be1e8f4bd  Packaged-Basic-PWA-windows-x64-QA.zip
745a9a5b27e2b416d0da027a5b990030cd91ae62084065f258f422dfd20c505d  linux-x64/bin/ahdcode
3e24687c0550cc7f7818aa57933aabe13e7b4514839c9876d3894f6704d6c3b7  linux-x64/bin/ahdsqlite
1e9e97a4572efbb7d2a51ed787e3d40596b12c8172b5ef7b971364d66aa08b2e  windows-x64/bin/ahdcode.exe
5645a8ae2abaa2c1bf47fd76536a3b827cf5d4f478398f62f4b1e3cfff496d1b  windows-x64/bin/ahdsqlite.exe
```

The binaries were built with `CGO_ENABLED=0 go build -trimpath` for their
recorded GOOS and GOARCH. The Windows CLI reported `AhdCode v2.11.0`.
The Windows SQLite helper accepted an empty input stream, and the relocated
starter checks exercised actual SQLite work. Toolchain installer packages
and signed binaries follow the older release policy.

## 8. Final audit verdict

The final audit used fresh projects and browser profiles, moved the projects
away from their source paths, and inspected both application archives and
the two source formats. It also replayed Windows failures against v2.10.0.

| Severity | Findings |
| --- | --- |
| BLOCKER | 0 new findings |
| MAJOR | 0 new findings |
| MINOR | Windows fixture and privilege limitations. The case-collision fixture still needs a case-sensitive runner. |
| OBSERVATION | Additional OS and mobile browser coverage and the Linux CI run remain pending. Public assets require review, and local source extraction needs a Windows path short enough for the vendored tree. |

- **Is Phase 7 implemented?** Yes. Windows Web and PWA packages ran outside
  their original source paths. The compiler and documentation identify
  v2.11.0.
- **Did independent final QA pass?** The separate local audit passed within
  the verified scope. A new Gemini review has not run.
- **Is AhdCode v2.11.0 ready for release?** The source-only candidate is
  ready for release review. Publication readiness remains conditional on
  review of the final Git revision and CI on that revision.
- **What remains unverified?** Native Linux and macOS execution, other
  architecture execution, mobile installation, the privilege-gated links,
  and the full native compiler suite on Linux.
- **Is the local working tree clean?** No. All earlier work and Phase 7
  changes are present and unstaged.

Gemini recheck proposal

1. Rebuild from the candidate ZIP, verify both source checksums, and compare
   the archive inventory with the approved release commit.
2. On Linux, run the root CI jobs and the case-sensitive asset-collision and
   symlink cases. Test refusal to overwrite deployments with privately
   provisioned state.
3. Extract and run a fresh Basic PWA package behind Caddy. Confirm GET and
   HEAD, real installation and launch, offline fallback, authenticated-cache
   bypass, worker update after closing old clients, and unrelated-cache
   preservation.
4. Recheck persistent session shutdown and WSS, and try a privileged Windows
   symlink host or a native macOS runner for the platform-specific fixtures.

Final Git state is `v2.11-dev` at
`5239c32e2996463b51d0f8288b3a433741767cca`. The index is clean.
There are 63 modified and 24 deleted tracked paths and 38 untracked paths
with this report included. Earlier phase work remains in place. No commits,
tags, pushes, or remote release actions were performed.
