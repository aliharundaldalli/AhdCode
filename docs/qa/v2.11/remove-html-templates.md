# AhdCode v2.11 Templates removal QA

The project owner approved removal of the public `Templates` builtin before Phase 6. This is an intentional source-compatibility break. Applications that use `Templates`, `TemplateSet`, `TemplateData`, or `TemplateError` must migrate. The upgrade notes explain how to compose HTMLNode values in `.ahd` pages, layouts, and components. [English upgrade note](../../UPGRADE_2.11.md) · [Turkish upgrade note](../../UPGRADE_2.11_TR.md).

The worktree stays on `v2.11-dev`, based on `5239c32e2996463b51d0f8288b3a433741767cca`. The rebuilt compiler reports `AhdCode v2.10.0`. Phase 0 and Phase 5 changes remain in the working tree. The web packaging direction is the compiled AhdCode application and its static assets. Phase 6 remains pending.

## Implementation

- Removed the builtin registration, opaque types and operations, lowering, evaluator dispatch, Go code generation, generated template runtime, template engine and its fixtures.
- Moved the credential-specific no-follow reader and platform helpers into the development HTTPS runtime. The Windows junction replacement regression now exercises the credential reader directly. The certificate validation and file-size bound stay in effect.
- Joined the forced-shutdown fixture's HTTP dispatcher before the next test can replace its test-only reload identity. The race detector observed an overlap between the late response writer and that replacement. The production shutdown path is unchanged.
- The dev watcher follows the compiler's source graph. Atomic replacement of `.ahd` files and restoration of missing imports still trigger rebuilds. Untracked HTML no longer triggers a rebuild.
- Basic generates Turkish pages and custom 404/500 responses through a shared `.ahd` layout and `Web.UI` components. The generated HTML keeps its Turkish language attribute, navigation and routes. Bootstrap, the project stylesheet and script, and the logo use the existing static mount. The other starters retain their generation sources. The embedded scaffolding directory retains the shared, Admin, and vendor files.
- Updated English and Turkish current docs and synchronized the English starter bundle. The v2.10 release text remains historical. Its Turkish Templates reference now points to the published tag.

## Verification commands

Windows amd64 and Go 1.26.7. The temporary QA builds used `GOFLAGS=-buildvcs=false`. The full attempts used:

```powershell
$packages = @(
    './cmd/ahdcode', './internal/semantic', './internal/lowering', './internal/evaluator',
    './internal/backend/golang', './internal/backend/golang/ahdruntime', './internal/build',
    './internal/initweb', './internal/framework', './internal/migrate'
)
go test -json -p 3 -count=1 @packages
go test -json -p 3 -race -count=1 @packages
```

The targeted runs used the freshly rebuilt CLI as `AHDCODE_BIN` so the MVC and CRUD compilation fixture ran:

```powershell
$env:GOMAXPROCS = '4'
$focus = '^Test(DevHTTPS|HTTPSConfiguration|HTTPSIsRefused|ProductionConfiguration|DevWatcher|DevReload|HTTP(Endpoint|Shutdown|Publication|Registry|Wildcard|Nested|SameServer|BindFailure|CallbackRetirement|ErrorPage|Session|NativeAndEvaluator)|WebSocket|PersistentSession|MemorySession|Migrate(DoesNotDisturb|ExampleApp|CLI)|WebUI|WebAssets|HTML|WizardBasic|HTTPEvaluator|Evaluator(HTTP|Persistent|CrossServer|Yield|NestedServer|Unstarted)|MVCAndCRUDApplicationsCompile|BasicStarter|WebConfig|GeneratedProgramCarriesRuntime|DocumentationBundle|AllStartersReceiveDocumentationBundle)'
$skip = '^Test(HTTPSessionNativeRelocatesWithoutToolchain|HTMLRenderNativeProgram|HTMLParseHTTPScrapeNativeProgram|HTMLParseMakesNoResourceRequests|HTMLParseInvalidSelectorIsCatchableHTMLError|WebUI(EscapesEveryTextEntryPoint|ComposesTheSameMarkupAsTheHTMLModule|DoesNotMutateCallerAttributes|BuildsTablesAndForms))$'
go test -json -p 3 -count=1 -run $focus -skip $skip @packages
go test -json -p 3 -race -count=1 -run $focus -skip $skip @packages
```

Both targeted runs passed. Each recorded 268 passing tests and subtests,
with two skips for Windows file symlink privileges:
`TestDevHTTPSInvalidCredentials/linked_key` and
`TestDevHTTPSWindowsCredentialOpenRejectsReplacement/file_symlink`.
The Windows junction case ran and passed. Nine packages passed, and
`internal/framework` has no test files. The race run reported no races.

The generated Basic test runs the native starter and checks the Turkish home and about pages, both custom status pages, the absence of internal exception details, and byte-for-byte delivery of Bootstrap, CSS, JavaScript, and the logo.

`ahdcode run legacy-template.ahd` reported `SEM028` for an old `bring Templates` import.

The forced-shutdown fixture and subsequent reload test also passed six
times in one race-detector invocation:

```powershell
go test -race -count=6 -run '^Test(DevHTTPSForcedHTTPShutdownReleasesSQLite|DevReloadPreservesTurkishCapitalI)$' ./internal/backend/golang/ahdruntime
```

## Broader Windows results

Both unfiltered attempts reported failures. The `internal/build` package hit
Go's 10-minute timeout in both runs. The ordinary run reached a Data sort
fixture. The race run reached a forward-constant fixture. The complete
`internal/initweb` package passed in both modes after the updated canonical
docs and bundle were synchronized with matching Windows line endings.

| Windows observation | Evidence |
| --- | --- |
| Executable paths | The native build helpers expect paths such as `program` or `hello` while the Windows compiler emits `.exe`. Relocation fixtures also try to run or move an extensionless file. The baseline reproduced the CLI, Archive, HTML, and HTTP session examples. The two final filter commands exclude nine affected fixtures, including four Web.UI tests that use the same extensionless executable helper. |
| Permissions and symlinks | `TestStudioEnvBootstrapPreservesExisting` expects Unix-style mode bits. `TestResourceSymlinkStaysInsideTheCanonicalRoot` and `TestResourceSymlinkCannotLeaveTheRoot` require a Windows symlink privilege this host lacks. All three reproduced on the baseline. |
| Network fixture assertions | `TestHTTPDownloadDestinationRules` and `TestTLSInspectConnectionFailuresAreErrors` failed identically on the baseline. The development HTTPS listener, WSS, session, and callback regressions use separate fixtures. |
| Other broad failures reproduced on the baseline | `TestSQLiteBackupIsAConsistentLiveSnapshot` and `TestSQLiteBackupFailuresLeaveNoBackup` report an invalid Windows drive path. `TestDiskInspectReportsConsistentSizes` reports a filesystem-information error. `TestCodesNativeExecutableRelocates` moves an extensionless binary. `TestCodesAndMySQLComposeOneVendoredWorkspace` reports a vendored module manifest mismatch. |
| Scheduler timing in the broad race run | `TestCronStopWakesAWaitingSchedulerWithoutLeakingGoroutines` missed its deadline once. A three-run isolated race check on the baseline passed. |

## Browser acceptance

The browser started from a clean `ahdcode init web basic` project and used `ahdcode dev --https app.ahd`. Firefox verified the server with its public CA imported only into a disposable Firefox profile. The Windows trust store was untouched. The browser reported a secure context and loaded Bootstrap JavaScript and local CSS and JS. The browser observed status 200, 404, and 500 with the expected Turkish page headings on the three secondary routes. It loaded the same-origin reload script and endpoint. Editing the heading in `Pages/Home.ahd` advanced the reload marker from generation 1 to 2 and the already-open browser showed the new heading without a navigation command from the test. The QA script observed no mixed-content errors. Some repeat attempts timed out in Firefox's automation navigation calls. The retained passing run launched Firefox directly on the HTTPS URL. [Firefox automation](https://firefox-source-docs.mozilla.org/remote/Security.html) · [NSS certificate import implementation](https://raw.githubusercontent.com/nss-dev/nss/master/cmd/certutil/certutil.c).

## Proposed independent Gemini recheck

1. Check that `bring Templates` fails resolution and that native output retains HTML and development TLS support. Inspect the diff for changes to the HTML builder, Web.UI, or scaffold assets.
2. Generate Basic in an empty directory. Inspect its `.ahd` source graph and exercise `/`, `/hakkinda`, an unknown URL, and `/ornek-hata`. Confirm that both status pages keep the shared shell and hide the original handler error.
3. Run that generated Basic project over HTTPS in a profile with the test CA installed. Check Bootstrap, CSS, JS, the browser's secure context and network log. Change a required `.ahd` page and observe the open tab reload.
4. Re-run the Phase 0 and Phase 5 lifecycle, WSS, session, and native/evaluator tests under the race detector. Check the Windows credential junction case and ordinary HTTP dev.
5. Generate Empty, Admin, MVC, and CRUD from the same compiler. Check their existing source layout, then run the Linux CI verification when its pending integration is approved.

## Changed paths

Paths are relative to the development worktree and compare to its accepted Phase 5 state.

```text
M  README.md
M  README_TR.md
M  ROADMAP_3.0_TR.md
M  cmd/ahdcode/dev.go
M  cmd/ahdcode/devwatch.go
M  cmd/ahdcode/devwatch_test.go
M  docs/CLI.md
M  docs/CLI_TR.md
M  docs/GETTING_STARTED.md
M  docs/GETTING_STARTED_TR.md
M  docs/HTML.md
M  docs/HTML_TR.md
M  docs/HTTP.md
M  docs/HTTP_TR.md
M  docs/MODULES.md
M  docs/MODULES_TR.md
M  docs/RELEASE_2.10.0_TR.md
M  docs/STUDENT_GUIDE_EN.md
M  docs/STUDENT_GUIDE_TR.md
D  docs/TEMPLATES.md
D  docs/TEMPLATES_TR.md
A  docs/UPGRADE_2.11.md
A  docs/UPGRADE_2.11_TR.md
M  docs/WEB.md
M  docs/WEB_TR.md
A  docs/qa/v2.11/remove-html-templates.md
M  internal/backend/golang/ahdruntime/ahdruntime.go
M  internal/backend/golang/ahdruntime/embed.go
M  internal/backend/golang/ahdruntime/http_devhttps_test.go
M  internal/backend/golang/ahdruntime/http_devreload.go
A  internal/backend/golang/ahdruntime/http_devtls_other.go
A  internal/backend/golang/ahdruntime/http_devtls_unix.go
A  internal/backend/golang/ahdruntime/http_devtls_windows.go
A  internal/backend/golang/ahdruntime/http_devtls_windows_test.go
D  internal/backend/golang/ahdruntime/templates.go
D  internal/backend/golang/ahdruntime/templates_fs_other.go
D  internal/backend/golang/ahdruntime/templates_fs_unix.go
D  internal/backend/golang/ahdruntime/templates_fs_windows.go
D  internal/backend/golang/ahdruntime/templates_fs_windows_test.go
D  internal/backend/golang/ahdruntime/templates_qa_test.go
D  internal/backend/golang/ahdruntime/templates_test.go
M  internal/backend/golang/expr.go
M  internal/backend/golang/generate.go
M  internal/backend/golang/generate_test.go
D  internal/backend/golang/templates_module.go
A  internal/build/basic_starter_test.go
D  internal/build/template_object_once_test.go
D  internal/build/template_pages_test.go
M  internal/evaluator/call.go
D  internal/evaluator/templates.go
D  internal/evaluator/templates_test.go
M  internal/framework/sources/Web.ahd
M  internal/initweb/docbundle.go
M  internal/initweb/docbundle/CLI.md
M  internal/initweb/docbundle/GETTING_STARTED.md
M  internal/initweb/docbundle/HTML.md
M  internal/initweb/docbundle/HTTP.md
M  internal/initweb/docbundle/MODULES.md
M  internal/initweb/docbundle/README.md
M  internal/initweb/docbundle/STUDENT_GUIDE_EN.md
D  internal/initweb/docbundle/TEMPLATES.md
A  internal/initweb/docbundle/UPGRADE_2.11.md
M  internal/initweb/docbundle/WEB.md
M  internal/initweb/generate.go
M  internal/initweb/initweb_test.go
M  internal/initweb/sources.go
D  internal/initweb/templates/basic/templates/components/nav.html
D  internal/initweb/templates/basic/templates/layouts/site.html
D  internal/initweb/templates/basic/templates/pages/about.html
D  internal/initweb/templates/basic/templates/pages/home.html
D  internal/initweb/templates/basic/templates/pages/not-found.html
D  internal/initweb/templates/basic/templates/pages/server-error.html
M  internal/lowering/lower.go
D  internal/lowering/templates_builtin.go
M  internal/semantic/expression.go
M  internal/semantic/model.go
M  internal/semantic/standard_modules.go
D  internal/semantic/templates_module.go
D  internal/semantic/templates_module_test.go
```
