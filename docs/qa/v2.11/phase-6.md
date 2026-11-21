# AhdCode v2.11 Phase 6 PWA QA handoff

Worktree `v2.11-dev` remains based on `5239c32e2996463b51d0f8288b3a433741767cca`.
The compiler reports `AhdCode v2.10.0`. Existing Phase 0, Phase 5, and
Templates-removal changes remain in the worktree. The Phase 6 work is
unstaged and awaits independent QA.

## 1. CLI and API

`ahdcode pwa init [--name NAME] [--short-name NAME]` creates the optional
`PWA/` files and preserves existing project sources. The entry point mounts
them with `ahdPwaMount(site)`. A shared layout adds the HTMLNode results
of `ahdPwaManifest()` and `ahdPwaRegistration()`. The generated module uses
existing Web and HTTP APIs. [Setup and cache policy](../../PWA.md).

## 2. Changed files

Phase 6 paths relative to the development worktree:

```text
M  cmd/ahdcode/main.go
A  cmd/ahdcode/pwa.go
M  docs/CLI.md
M  docs/CLI_TR.md
A  docs/PWA.md
A  docs/PWA_TR.md
M  docs/WEB.md
M  docs/WEB_TR.md
A  docs/qa/v2.11/phase-6.md
A  internal/build/pwa_test.go
M  internal/initweb/docbundle.go
M  internal/initweb/docbundle/CLI.md
A  internal/initweb/docbundle/PWA.md
M  internal/initweb/docbundle/WEB.md
A  internal/initweb/pwa.go
A  internal/initweb/pwa/Integration.ahd
A  internal/initweb/pwa/public/register.js
A  internal/initweb/pwa/worker.js
A  internal/initweb/pwa_test.go
A  internal/initweb/pwa_worker_test.cjs
```

## 3. Manifest and icons

The exact `/ahd-pwa.webmanifest` route serves editable JSON with
`application/manifest+json`. It contains name, short name, ID, start URL,
scope, display, theme and background colors, and local PNG icons. The
scaffold generates the 192 and 512 pixel versions from the existing
AhdCode logo. Real Firefox decoded both dimensions as expected.

## 4. Service worker

The registration script checks for a secure top-level context and
browser support. The worker lives at `/ahdcode-sw.js` with scope `/` and
uses revalidation for its script. Cache names incorporate the registration
scope and a digest of the worker version and asset descriptor list. An
update waits while the old version has clients, then prunes only its own
old caches on activation. Firefox exercised the waiting state and cleanup.

## 5. Offline and recovery

A query-free navigation to `/` uses the network. A transport failure on
that navigation produces a generic `503` offline page. A reachable
server's redirects and errors still pass through. The selected public
assets can use their cached copies on network failure. Firefox verified
the fallback after stopping the actual dev process and recovered after
restarting it at the same origin.

## 6. Cache protection

The worker writes only explicit public assets fetched with credentials
omitted, redirects disabled, integrity pinned, and media type validated.
It rejects private or non-cacheable responses and authentication-based
variation. It stores no runtime fetch or navigation response, bypasses
writes, queries, authenticated API traffic, and AhdCode's reload endpoints.
The browser test verified login, rotation, logout, and a second identity
on the same profile, with only the two icons in Cache Storage throughout.
It also verified that private and reload requests failed offline. The
existing persistent session fixture exercised two clients, logout,
revocation, and restarts. Private routes still need appropriate HTTP cache
headers for the browser's separate HTTP cache.

## 7. Generated Basic integration

A fresh `ahdcode init web basic` project used the supplied `.ahd` layout,
pages and components, Bootstrap, CSS, JavaScript, and custom 404/500
handlers. The CLI created PWA files independently. The browser fixture
then applied the documented entry and layout changes. Editing the home
page triggered AhdCode's existing live reload while the worker controlled
the tab. The underlying starter generators and Go runtime are unchanged.

## 8. Browser installation

Both installed Chromium-family browsers reported zero installability
errors on trusted loopback HTTP with the worker active and the expected
manifest. Separate Firefox runs verified real HTTPS behavior with its
temporary profile trusting the local CA. Installation evidence is
recorded separately from manifest discovery below.

Chrome 157 installed the app via its local DevTools pipe in a disposable
profile. Opening an installed app with the headless default preference
returned its page in browser display mode. After setting the isolated
profile's app preference to standalone, Chrome launched the configured
start URL, rendered the Basic home, and returned true for
`matchMedia("(display-mode: standalone)")`. The test uninstalled the app.
Edge passed its installability check. Its remote debugging connection
did not expose the install command.
[Chrome DevTools PWA commands](https://chromedevtools.github.io/devtools-protocol/tot/PWA/).

## 9. HTTPS and Caddy

Firefox passed the Basic PWA smoke over direct `ahdcode dev --https` and
separately through verified Caddy `v2.11.9` with a plaintext production-mode
AhdCode upstream and `APP_PROTOCOL=https`. The proxy forwarded the app
paths without rewriting them. The real browser verified HTTPS context,
registration, icons, asset loading, sessions, 404/500, and no mixed
content diagnostics. Both used a disposable Firefox trust profile with
certificate verification enabled. The OS trust store stayed untouched.
[Caddy release](https://github.com/caddyserver/caddy/releases/tag/v2.11.9).

## 10. Regression commands and results

Windows amd64, Go 1.26.7, temporary build flags `GOFLAGS=-buildvcs=false`
and `GOMAXPROCS=4`. `AHDCODE_BIN` pointed at the temporary rebuilt CLI
to enable MVC/CRUD compile checks.

```powershell
$packages = @('./cmd/ahdcode','./internal/backend/golang/ahdruntime',
    './internal/build','./internal/evaluator','./internal/initweb','./internal/migrate')
$focus = '^Test(PWA|DevHTTPS|DevWatcher|DevReload|HTTP(Endpoint|Shutdown|Publication|Registry|Wildcard|Nested|SameServer|BindFailure|CallbackRetirement|ErrorPage|Session|NativeAndEvaluator)|WebSocket|PersistentSession|MemorySession|HTML(Text|Compose|Document|Preserves)|HTTPEvaluator|Evaluator(HTTP|Persistent|CrossServer|Yield|NestedServer|Unstarted)|BasicStarter|WizardBasic|MVCAndCRUDApplicationsCompile|AdminSQLite|MVCSQLite|DocumentationBundle|AllStartersReceiveDocumentationBundle|Migrate)'
$skip = '^TestHTTPSessionNativeRelocatesWithoutToolchain$'
go test -p 3 -json -count=1 -run $focus -skip $skip @packages
go test -p 3 -json -race -count=1 -run $focus -skip $skip @packages
node --test internal/initweb/pwa_worker_test.cjs
go test -count=1 -run '^TestDocumentationBundle' ./internal/initweb
go test -race -count=1 -run '^TestDocumentationBundle' ./internal/initweb
```

The focused ordinary and race runs each recorded 332 passing tests and
subtests, with four privilege-related skips and no failures. The race
run reported no races. All four JavaScript tests passed. The bundle
checks passed. The existing Web.UI escaping fixture was also executed
directly through the Windows CLI and escaped all 11 text entry points.
Coverage includes HTTP/HTTPS shutdown, WSS, cookies and
persistent sessions, evaluator parity, Basic and Web assets, Admin
initialization, MVC/CRUD compile checks, and the SQLite migration CLI.

## 11. Existing Windows issues

- The extensionless native session relocation fixture was excluded.
  This issue reproduced on the prior baseline.
- File symlink privileges are unavailable. The skips were
  `TestDevHTTPSInvalidCredentials/linked_key`,
  `TestDevHTTPSWindowsCredentialOpenRejectsReplacement/file_symlink`,
  `TestMigrateRejectsSymlinkLock`, and
  `TestMigrateRejectsASymlinkOutsideTheDirectory`. The Windows
  junction case ran.
- Phase 6 did not run unfiltered `internal/build`. The prior baseline
  report records its ten-minute timeout and other Windows fixtures.
  [Templates-removal baseline](remove-html-templates.md).

## 12. Remaining limitations

The scaffold is for a dedicated root-scoped app. The worker does not
provide authenticated offline access, queued writes, or database
synchronization. Keep the generated PWA files available at runtime.
Platform install UI varies, and mobile Home Screen behavior and OS
launcher icon presentation remain unverified. Linux and macOS execution
remain for independent QA.

## 13. Proposed Gemini recheck

1. Generate Basic and confirm it works before and after adding the three
   documented wiring calls. Verify the manifest MIME, configured name and
   both icon dimensions.
2. Install in a supporting browser, launch standalone, and inspect
   the platform app name and icon.
3. Sign in with two distinct identities. Rotate one session and log out of
   it. Inspect Cache Storage online and offline, including direct requests
   to a private route and the dev reload endpoints.
4. Stop and restore the HTTPS server. Verify the generic fallback and
   recovery. Change a required `.ahd` file while the worker is active.
5. Change a correctly pinned public asset and worker version. Verify the
   update waits, removes only its old cache, and leaves unrelated caches.
6. Repeat with Caddy terminating TLS and AhdCode serving HTTP internally.
   Run the focused regressions and compare Windows failures separately.
