# AhdCode v2.12.0-rc1 — Independent macOS QA, Distribution & Signing Report

**Verdict:** `PASS`  
**Release Candidate Branch:** `release/v2.12.0-rc1`  
**Verified Commit:** `249c887cde9f4e4510e6ad8cadbb0fa7239e994c`  
**Upstream Repository:** `https://github.com/aliharundaldalli/AhdCode`  
**GitHub Actions Run:** `45079705719` (`14/14` jobs `completed` / `success`)  
**Scope:** Independent Release Candidate validation and local Developer ID signing / Apple notarization only (**not** final publication: `main` untouched, no `v2.12.0` tag, no GitHub Release, production sites untouched).

---

## 1. Environment & Toolchain

| Component | Observed Value |
| --- | --- |
| Operating System | macOS `27.2.1` (build `26B131`), Darwin Kernel `27.2.0` |
| Hardware | Apple M3, `8` CPU cores, `8 GiB` RAM (`8589934592` bytes) |
| Architecture | `arm64` (native) |
| Go Toolchain | `go version go1.27.0 darwin/arm64` |
| Python / Node | `Python 3.14.7`, Node `v25.9.0` |
| Apple CLI Tools | `xcrun` version `72`, `codesign`, `productbuild`, `pkgbuild`, `pkgutil`, `notarytool`, `stapler`, `spctl`, `hdiutil`, `lipo` |
| Gatekeeper Status | `spctl --status` → `assessments enabled` (kept enabled throughout) |
| Google Chrome | `Google Chrome 157.0.8103.79` (`V8 15.7.112.43`, WebKit `537.36`) |
| Signing Identities | `Developer ID Application: Ali Harun DALDALLI (HB46M6625T)` (`BCF38A1FEF3414DE222BBE7D32CD48C8AE1BA218`)<br>`Developer ID Installer: Ali Harun DALDALLI (HB46M6625T)` (`F4A02CF0E4EDA255965CB4231DCCF6994EF6A10D`) |
| Notary Profile | Keychain profile `ahdcode-notary` (`47` prior submissions verified; latest prior `AhdCode-2.11.0-macos-x64.dmg` `Accepted`) |
| Existing Installation | Preserved `/Users/ahd/Library/AhdCode` (`current -> versions/2.11.0-rc1`, `AhdCode v2.11.0`, SHA-256 `114a71d245daa1136cda67d4ddda90c7de81ee4276ac8f8ca4c185fd0ee4372f`) |

---

## 2. Repository, Commit Traceability & Documentation Review

- Isolated RC1 clone checked out at `/Users/ahd/.gemini/antigravity/brain/d4eb0842-bee3-4617-9709-1bf1e3faa3a0/scratch/v2.12.0-rc1-src` on branch `release/v2.12.0-rc1` (`HEAD=249c887cde9f4e4510e6ad8cadbb0fa7239e994c`).
- Commit delta from previously verified code commit `5e3713248bf249a37ce9672c286fb3ca210ad238` to `249c887cde9f4e4510e6ad8cadbb0fa7239e994c`: single documentation commit (`Document AhdCode v2.12 Phase 5 cross-platform CI and release QA`, adding `docs/qa/v2.12/phase-5.md`).
- Reviewed `docs/qa/v2.12/phase-1.md` through `phase-5.md`, `CHANGELOG.md`, `docs/RELEASE_2.12.0.md`, `docs/UPGRADE_2.12.md`, `docs/plans/v2.12-roadmap.md`, `docs/SQLITE.md`, `docs/PWA.md`, `docs/WEB.md`, `docs/CLI.md`, and `tooling/distribution/{build.py,build_pkg.py,test_macos_dist.py,install.sh}`.
- Packaging metadata inside both extracted ARM64 and x64 archives (`payload/BUILD.json`) traces directly to `249c887cde9f4e4510e6ad8cadbb0fa7239e994c`, version `2.12.0`, `bundled_runtimes: false`, `signed_app: true`, `signed_installer: true`, `notarized: true`. `tooling/distribution/build.py` was executed against a strictly clean Git working tree (`--allow-dirty-tree` unset) **before** writing this QA report into the scratch clone.

---

## 3. GitHub Actions CI Run `45079705719` (`14/14` Passing)

- **Workflow Run:** `https://github.com/aliharundaldalli/AhdCode/actions/runs/45079705719`
- **Branch / Commit:** `release/v2.12.0-rc1` @ `249c887cde9f4e4510e6ad8cadbb0fa7239e994c`
- **Created:** `2026-11-22T22:47:55Z` · **Updated:** `2026-11-22T23:16:26Z` · **Overall Status:** `completed` / `success`

| Job ID | Job Name | Started (UTC) | Completed (UTC) | Duration | Conclusion |
| --- | --- | --- | --- | --- | --- |
| `137384861851` | Windows x64 verification and packaging | `22:47:57Z` | `22:52:20Z` | `4m 23s` | `success` |
| `137384862023` | Targeted race coverage | `22:47:57Z` | `22:55:40Z` | `7m 43s` | `success` |
| `137384862024` | Nested helper modules (`cmd/ahdgraphics`) | `22:47:57Z` | `22:48:15Z` | `0m 18s` | `success` |
| `137384862064` | Core tests and static checks | `22:47:57Z` | `22:54:52Z` | `6m 55s` | `success` |
| `137384862095` | Native macOS x64 acceptance | `22:48:01Z` | `22:53:02Z` | `5m 01s` | `success` |
| `137384862108` | Native build suite | `22:47:57Z` | `23:16:25Z` | `28m 28s` | `success` |
| `137384862110` | MySQL 8.4 integration | `22:47:57Z` | `22:49:58Z` | `2m 01s` | `success` |
| `137384862115` | macOS ARM64 verification and packaging | `22:47:59Z` | `22:54:17Z` | `6m 18s` | `success` |
| `137384862128` | Nested helper modules (`cmd/ahdidentity`) | `22:47:57Z` | `22:48:14Z` | `0m 17s` | `success` |
| `137384862140` | Native Linux ARM64 acceptance | `22:48:00Z` | `22:49:44Z` | `1m 44s` | `success` |
| `137384862148` | PostgreSQL 17 integration | `22:47:57Z` | `22:50:07Z` | `2m 10s` | `success` |
| `137384862150` | Nested helper modules (`cmd/ahdplotview`) | `22:47:58Z` | `22:48:31Z` | `0m 33s` | `success` |
| `137384862156` | Nested helper modules (`cmd/ahdgui`) | `22:47:57Z` | `22:48:17Z` | `0m 20s` | `success` |
| `137384862196` | PostgreSQL 18 integration | `22:47:57Z` | `22:49:55Z` | `1m 58s` | `success` |

---

## 4. Native macOS ARM64 Test Suites, Static Checks & Benchmarks

All native Go suites passed with **0 failures** and **0 data races**:

- **Suite A — Compiler, Frontend, LSP, CLI, SQLite Protocol, InitWeb, Packaging (18 packages):**
  - Packages: `cmd/ahdcode`, `cmd/ahdsqlite`, `internal/{source,diagnostics,lexer,parser,module,semantic,analysis,lsp,formatter,lowering,evaluator,repl,sqliteproto,migrate,initweb,packaging}`
  - Ordinary (`go test`): `all=(pass=2546, fail=0, skip=6)` / `top=(pass=1198, fail=0, skip=6)`
  - Race (`go test -race`): `all=(pass=2546, fail=0, skip=6)` / `top=(pass=1198, fail=0, skip=6)`
  - Exact 6 skips (environment/platform-gated): `internal/initweb::TestWrite{CRUDHTTPProject,DogfoodAdminProject,MVCHTTPProject}`, `internal/migrate::TestMigrateRejects{Junctions,ReparseLockAndIncoming}`, `internal/packaging::TestWebAssetsRejectCaseCollisions`
- **Suite B — Runtime (`./internal/backend/golang/ahdruntime`):**
  - Ordinary (`go test`): `all=(pass=747, fail=0, skip=8)` / `top=(pass=607, fail=0, skip=8)`
  - Race (`go test -race`): `all=(pass=747, fail=0, skip=8)` / `top=(pass=607, fail=0, skip=8)`
  - Exact 8 skips (external DB/systemd-gated): `2` `TestMySQLIntegration*`, `5` `TestPostgreSQLIntegration*`, `TestServiceStatusAgainstRealSystemd`
- **Suite C — Focused Codegen/Build (`./internal/build`):**
  - Ordinary (`go test`): `all=(pass=100, fail=0, skip=2)` / `top=(pass=38, fail=0, skip=2)`
  - Race (`go test -race`): `all=(pass=100, fail=0, skip=2)` / `top=(pass=38, fail=0, skip=2)`
  - Exact 2 skips (external DB-gated): `TestMySQLIntegrationNativeAndEvaluatorAgree`, `TestPostgreSQLIntegrationNativeAndEvaluatorAgree`
- **PWA Worker JS Unit Tests:** `node --test internal/initweb/pwa_worker_test.cjs` → `4/4` passed, `0` failed.
- **Static Checks:** `go vet ./...`, `gofmt -l .`, and `git diff --check` all returned clean with `0` issues.
- **Nested Helper Modules (`vet` + `go test -race`):** `cmd/ahdgraphics` (`2.056s`), `cmd/ahdgui` (`1.673s`), `cmd/ahdplotview` (`4.093s`), `cmd/ahdidentity` (`2.004s`) — all passed cleanly.
- **Apple M3 LSP Benchmarks (`BenchmarkPhase1/(Small|Medium)/LSPReferences-8`, `-benchtime=1x -count=5`):**
  - `Small`: `0.4089`–`0.5404` ms/op (`~5,337–5,510` allocs/op)
  - `Medium`: `8.284`–`9.079` ms/op (`~119,098–119,562` allocs/op)

---

## 5. 10-Point Security & Isolation Validation

1. **Private/Source/DB Exclusion on HTTP Routes:** Verified `private/`, `sessions.db`, and `app.ahd` return `404` in the 2-user native security app; verified `/.env`, `/app.ahd`, `/PWA/Integration.ahd`, `/sessions.db`, and `/app.sqlite` return `404` on both `ahdcode dev --https` and packaged `RC1QA.app`.
2. **Path Traversal Blocking:** Verified `../` and `%2e%2e` traversal probes return `404`.
3. **Unauthenticated Access Denial & Cache Policy:** Unauthenticated requests return `401` with `Cache-Control: no-store`.
4. **CSRF Enforcement:** State-changing requests fail with `403` when CSRF token is missing or belongs to another user (`Bob` → `Alice`).
5. **Session Fixation Protection:** Login invokes session rotation (`session.rotate()`), invalidating pre-login session cookies (`401`).
6. **Cross-User Authorization & `no-store` on Denials:** Cross-user request (`/avatar/bob?owner=alice&file=alice.bin`) returns `403` with `Cache-Control: no-store` across `GET`, `HEAD`, `Range`, and `If-None-Match`.
7. **Authenticated File Serving Semantics:** `Alice` receives `200` (`HEAD`/`GET`, `Cache-Control: no-store`), `206` (`Range: bytes=0-5`), and `304` (`If-None-Match`).
8. **Session Persistence & Revocation:** Persistent SQLite sessions survive process kill/restart; `POST /logout` permanently revokes the session (`401`).
9. **Local HTTPS Key/CA Symlink & Permission Guard:** `ahdcode dev --https` stores `root-ca.pem`, `localhost.pem`, and `localhost-key.pem` with `0600` permissions in a `0700` directory and explicitly refuses paths traversing symlinks (`/tmp` → `/private/tmp`).
10. **Packaged Bundle & Release Leak Gate:** Packaged `RC1QA.app` contains zero `.ahd`, `.env`, `.db`, `.sqlite*`, or `AHDCODE.md` files; `leak_gate.py --self-test` passed and all `6` signed macOS release archives passed leak-gate inspection (checked against `33` tracked `AhdDataStudio` files and forbidden developer-local values).

---

## 6. SQLite Reliability, Concurrency & Integrity

- **Live WAL Backup (`backupTo("sub/snap #%.db")`):** Succeeded while a second uncommitted writer transaction (`db2`) was open; duplicate destination overwrite was refused; uncommitted writes rolled back (`ROLLBACK`) were absent from the backup; reopened backup passed `PRAGMA integrity_check = ok` with `COUNT=1;FIRST=committed-1`.
- **Read-Only Opening (`SQLite.openWithOptions(..., readOnly=true, ...)`):** Refuses write statements with `SQLiteError` and never creates missing database files.
- **Cross-Process Lock Contention:** Two-process contention check confirmed `busyTimeout=0` fails immediately with `database is locked`, whereas `busyTimeout=5000` waits and completes cleanly.
- **Packaged Helper Discovery:** Verified compiled binary using `SQLite.openWithOptions` with `AHDCODE_SQLITE_RUNTIME` unset discovers sibling packaged `ahdsqlite` cleanly (`UPGRADE_SQLITE=ok-pkg-sqlite`).

---

## 7. Compiler & Standalone Checker Validation (No Go on `PATH`)

- `ahdcode check` (valid `.ahd`, `PATH` stripped of Go):
  - Default text output: exit `0`, empty stdout.
  - `--format json`: exit `0`, `{"schemaVersion":1,"diagnostics":[]}`.
  - Left zero build artifacts in the project directory.
- Invalid `.ahd` (`x: Int := "😀"`): exits `1` with UTF-16 surrogate pair coordinates `{"start":{"line":1,"character":10},"end":{"line":1,"character":14}}` and relative `path: "bad.ahd"`.
- Invalid CLI flag (`--format xml`): exits `2` on stderr with empty stdout.
- Native build (`ahdcode build`): invalid input exits `1`; valid build inside `"proj with spaces/main.ahd"` succeeds and prints `"Merhaba ü😀 AhdCode"`.

---

## 8. Local HTTPS, Desktop `--web --pwa` Packaging & Chrome 157 Verification

- **`ahdcode dev --https app.ahd` (canonical `HOME=/private/tmp/rc1-v212-qa/dev-home`, `SERVER_PORT=19211`):**
  - Generated CA/cert material (`root-ca.pem` `713` B, `localhost.pem` `648` B, `localhost-key.pem` `241` B).
  - Verified `curl --cacert` over TLS against both `https://127.0.0.1:19211/` and `https://localhost:19211/` (`TLS_LOCALHOST_SAN_OK`).
  - Verified `/.env` → `404` and `/app.ahd` → `404`.
  - Edited `Pages/Home.ahd` and confirmed live rebuild/reload updated HTTPS responses automatically (`LIVE_RELOAD_HTTPS_OK`).
  - Stopped cleanly with `SIGINT` (`DEV_STOP_OK`).
- **Desktop PWA Packaging & Relocated Execution:**
  - `ahdcode package app.ahd --web --pwa --name RC1QA --output .../pwa-dist` produced `RC1QA.app` (`11,716,786` B binary + `PWA/` + `public/` + `Resources/{AppIcon.icns,ahdcode-app.json,app-icon.png}`) with zero leaked `.ahd`/`.env`/`.db` files.
  - Renamed source directory to `pwa-src-moved` and started relocated `RC1QA` on `127.0.0.1:19212`:
    - `GET /ahd-pwa.webmanifest` → `200` (`display: "standalone"`, `192x192` & `512x512` PNG icons)
    - `HEAD /ahdcode-sw.js` → `200`, `Cache-Control: no-cache`, `X-Content-Type-Options: nosniff`, `Content-Length: 4327`
    - `GET /private-check` → `200`, `Cache-Control: no-store`, `Set-Cookie: ahdsession=...; Path=/; Max-Age=86400; HttpOnly; SameSite=Lax`, body `private-no-store-qa`
    - `GET /assets/style.css` → `200`; `/app.ahd`, `/.env`, `/PWA/Integration.ahd`, `/sessions.db`, `/app.sqlite` → `404`
- **Google Chrome `157.0.8103.79` (Fresh Headless Profile via CDP):**
  - Service Worker `/ahdcode-sw.js` reached `activated` and controlled `http://localhost:19212/`.
  - `Page.getAppManifest` parsed `/ahd-pwa.webmanifest` with `errors: []`.
  - `Page.getInstallabilityErrors` returned `[]` (no installability blockers).
  - `/private-check` (both `fetch` and top-level navigation) returned `200` / `no-store`; Cache Storage (`ahdcode:pwa:http://localhost:19212/:53b9aabe...`) contained **strictly** two entries: `/ahd-pwa/icon-192.png` and `/ahd-pwa/icon-512.png`.
  - Attaching CDP offline emulation (`Network.emulateNetworkConditions({ offline: true })`) across both Page and Service Worker CDP targets confirmed `/ahdcode-sw.js` served its `503` offline fallback (`title: "Offline"`, `h1: "You are offline"`) for `/` while `/private-check` failed (`TypeError: Failed to fetch`). Disabling offline emulation recovered live HTML (`title: "Ana sayfa"`, `h1: "Ana sayfa"`).
  - *Scope note:* Browser verification used automated headless Chrome CDP; clicking the interactive desktop "Install" prompt and launching an OS-installed Chrome PWA window were not executed.

---

## 9. macOS Packaging, Upgrade, Rollback & Uninstall Verification

- **`tooling/distribution/test_macos_dist.py` against signed artifacts:** `Ran 9 tests in 21.985s` — **`OK`** (`leak_gate` self-test, postinstall traversal/symlink rejection, source-only policy, `.dmg` mount + `.pkg` install, home-directory `.pkg` metadata/install/execute/uninstall, `SHA256SUMS.txt` & manifests, signatures/notarization/Gatekeeper, `.tar.gz` upgrade/rollback/uninstall, `x64` artifact layout).
- **Standalone Tarball Lifecycle (`install.sh`, isolated `HOME=/private/tmp/rc1-v212-qa/upgrade-home`):**
  - Pre-existing unowned `Library/AhdCode` (without `.ahdcode-install` marker) was safely refused (`Refusing to change an existing directory not owned by this installer.`).
  - Clean `--setup-path` install wrote `.ahdcode-install` and configured `.bash_profile`, `.zprofile`, and `.zshrc`.
  - Simulated `v2.11.0` → `v2.12.0` upgrade flipped `current` atomically from `versions/2.11.0` to `versions/2.12.0` (`AhdCode v2.12.0`).
  - Rollback `ln -sfn versions/2.11.0 current` restored `AhdCode v2.11.0`; restoring `versions/2.12.0` returned `AhdCode v2.12.0`.
  - `--uninstall` removed `Library/AhdCode` and removed the `.zshrc` PATH block while preserving `Library/Application Support/ahdcode/sentinel.txt` (`preserve-me`).

---

## 10. Apple Developer ID Code Signing Evidence

- **Mach-O Binaries (14/14 verified after extraction of both `.tar.gz` archives):**
  - Every binary (`bin/ahdcode`, `libexec/ahdcode/{ahdsqlite,ahdnumeric,ahdplot,ahdgraphics,ahdgui,ahdplotview}`) passed `codesign --verify --strict --verbose=2` (`valid on disk`, `satisfies its Designated Requirement`), signed with `Authority=Developer ID Application: Ali Harun DALDALLI (HB46M6625T)`, `TeamIdentifier=HB46M6625T`, hardened runtime `flags=0x10000(runtime)`, trusted timestamps (`2026-11-23 02:13:48–54` ARM64; `02:17:14–20` x64 local UTC+3).
  - `lipo -archs`: ARM64 tarball binaries all `arm64`; x64 tarball binaries all `x86_64`.
- **Installer Packages (`.pkg`):**
  - `pkgutil --check-signature` on both `.pkg` installers: `signed by a developer certificate issued by Apple for distribution`, `Notarization: trusted by the Apple notary service`, chain `Developer ID Installer: Ali Harun DALDALLI (HB46M6625T)` → `Developer ID Certification Authority` → `Apple Root CA`, signed with trusted timestamp (`2026-11-22 23:14:08 +0000` observed for ARM64 `.pkg`).
- **Disk Images (`.dmg`):**
  - Signed with Developer ID Application identity; `codesign --verify` reported `valid on disk` and `satisfies its Designated Requirement`.

---

## 11. Apple Notarization, Stapling & Gatekeeper Verification

| Artifact | Notary Submission ID | Notary Result | `stapler validate` | `spctl -a -vv` |
| --- | --- | --- | --- | --- |
| `AhdCode-2.12.0-macos-arm64.pkg` | `f06f6a3b-2561-4f36-80d3-5cd7fb9830aa` | `Accepted` | `The validate action worked!` | `accepted` (`source=Notarized Developer ID`, `origin=Developer ID Installer: Ali Harun DALDALLI (HB46M6625T)`) |
| `AhdCode-2.12.0-macos-arm64.dmg` | `a7e3b5da-d9c4-429f-9523-0cc6d09dce4b` | `Accepted` | `The validate action worked!` | `accepted` (`source=Notarized Developer ID`, `origin=Developer ID Application: Ali Harun DALDALLI (HB46M6625T)`) |
| `AhdCode-2.12.0-macos-x64.pkg` | `1a4e6cb2-d45c-4cf2-912f-83b514f2ed76` | `Accepted` | `The validate action worked!` | `accepted` (`source=Notarized Developer ID`, `origin=Developer ID Installer: Ali Harun DALDALLI (HB46M6625T)`) |
| `AhdCode-2.12.0-macos-x64.dmg` | `b99d7715-718b-43b9-ae29-2395da196749` | `Accepted` | `The validate action worked!` | `accepted` (`source=Notarized Developer ID`, `origin=Developer ID Application: Ali Harun DALDALLI (HB46M6625T)`) |

---

## 12. Preserved Signed Release Candidate Artifact Inventory

Preserved locally (not published) in `/Users/ahd/.gemini/antigravity/brain/d4eb0842-bee3-4617-9709-1bf1e3faa3a0/scratch/v2.12.0-rc1-macos-signed` alongside `SHA256SUMS.txt` (`6/6` `shasum -a 256 -c` `OK`) and `manifest-darwin.json`:

| Filename | Architecture | Format | Byte Size | Signed | Notarized | SHA-256 |
| --- | --- | --- | ---: | --- | --- | --- |
| `AhdCode-2.12.0-macos-arm64.tar.gz` | `arm64` | `tar.gz` | `68,621,883` | Yes (inner Mach-O) | No | `16786d2125c847225cc11659e6e7a29cb178a9cb6a8e88dfab66c89efbdc9c96` |
| `AhdCode-2.12.0-macos-arm64.pkg` | `arm64` | `pkg` | `68,789,633` | Yes | Yes | `a7d63801905e4cf6fd36b977536b6dea9d0d7e466455d958765e12455441c2ba` |
| `AhdCode-2.12.0-macos-arm64.dmg` | `arm64` | `dmg` | `69,853,374` | Yes | Yes | `0d33e4bd39305ba965a3e24e4562cd95ca61bd4c679de28d3e1e50cfea56eb81` |
| `AhdCode-2.12.0-macos-x64.tar.gz` | `amd64` (`x64`) | `tar.gz` | `74,644,702` | Yes (inner Mach-O) | No | `6b6d155b74f7a543b564d0815e76a8da69ec6def57a3d85945b79d03be2e6d04` |
| `AhdCode-2.12.0-macos-x64.pkg` | `amd64` (`x64`) | `pkg` | `74,876,232` | Yes | Yes | `d8e96fd1d34ffdd9c6a8ecc0c52f5ac037b770af3ce206d6140fc0f0d6d46f86` |
| `AhdCode-2.12.0-macos-x64.dmg` | `amd64` (`x64`) | `dmg` | `75,929,249` | Yes | Yes | `e4eaca4e7cccbd8d7c76452b68f276c61b5d8bcb3df08f263d1428c5fc9858a2` |

---

## 13. Cross-Platform Coverage & Honest Scope Limitations

- **macOS `x64` Runtime Execution:** Built, signed, Apple-notarized, Gatekeeper-verified, `lipo`-checked (`x86_64`), and inspected locally on Apple M3 via cross-compilation; native Intel macOS execution is covered by GitHub Actions job `Native macOS x64 acceptance` (`137384862095` on `macos-15-intel`, `success`), not executed locally on Apple Silicon without Rosetta testing.
- **Windows / Linux / External Databases / `systemd`:** Not executed natively on this Mac (reflected in the 6 Windows/NTFS/case skips in Suite A, 8 DB/`systemd` skips in Suite B, and 2 DB skips in Suite C); verified via GitHub Actions Run `45079705719` (`14/14` jobs `success`).
- **Desktop PWA Interactive OS Install UI:** Verified automated Chrome `157.0.8103.79` headless CDP installability, manifest, Service Worker, Cache Storage policy, and offline `503` fallback; did not perform a manual mouse-click desktop OS PWA installation.
- **Release Governance Boundaries Maintained:** `release/v2.12.0-rc1` was not merged into `main`, no `v2.12.0` tag or GitHub Release was created, and `ahdcode.org` / `panel.ahdcode.org` / `/Users/ahd/Library/AhdCode` (`v2.11.0`) remain untouched.

---

## 14. Findings & Defect Classification

- **BLOCKER:** `0`
- **MAJOR:** `0`
- **MINOR / Informational (`2`):**
  1. **macOS `27.2.1` `hdiutil create` Deprecation Warning:** During `.dmg` creation, macOS emitted `hdiutil: WARNING: 'hdiutil create -volname -size -format ...' is deprecated. Please use 'diskutil image create from/blank --volumeName -size --format ...' instead.` Both `.dmg` artifacts built, signed, notarized, stapled, and passed `spctl`; track migrating `tooling/distribution/build.py` in a future maintenance pass.
  2. **Symlink Traversal Check with `/tmp` in `ahdcode dev --https`:** On macOS, `/tmp` is a symlink to `/private/tmp`. Running `ahdcode dev --https` with `HOME=/tmp/...` exits `1` with `ahdcode dev: local HTTPS credential paths must be readable and must not use symbolic links or junctions`. Passing canonical paths (`/Users/...` or `/private/tmp/...`) succeeds cleanly; behavior reflects intentional security hardening.

---

## 15. Final Verdict

**`PASS`**

AhdCode `v2.12.0-rc1` (`release/v2.12.0-rc1` @ `249c887cde9f4e4510e6ad8cadbb0fa7239e994c`) passed every local native macOS ARM64 unit/race/security/SQLite/checker/HTTPS/Chrome-PWA/packaging/signing/notarization gate as well as all `14/14` jobs in GitHub Actions run `45079705719`, with zero BLOCKER or MAJOR defects. Signed and notarized macOS distribution artifacts are ready in the preserved scratch directory for the separate final release and publication workflow upon explicit release approval.
