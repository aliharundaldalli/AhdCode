# AhdCode v2.12.0 — Phase 5 Cross-Platform Release QA & Closure

## 1. Repository & Branch State
- **Branch:** `v2.12-dev` (tracking `origin/v2.12-dev`)
- **Verified code commit:** `5e3713248bf249a37ce9672c286fb3ca210ad238` (`Run native candidate smoke tests on Linux ARM64 and macOS Intel`)
- **Version identity:** `AhdCode v2.12.0` (`internal/ahdversion/version.go`)
- **Preceding Phase 5 commits on `v2.12-dev`:**
  - `9086c12` — `Prepare AhdCode v2.12 release candidate and fix Windows verification` (run `45075766097`, failed 3 jobs; resolved in `d934e58`)
  - `d934e58` — `Correct candidate tests for native path and registry semantics` (run `45076356656`, **12/12 jobs passed**)
  - `5e37132` — `Run native candidate smoke tests on Linux ARM64 and macOS Intel` (run `45076426514`, **14/14 jobs passed**)
- No release candidate branch, merge to `main`, release tag, or production deployment was performed.

## 2. GitHub Actions CI Results (`45076426514` on `5e37132`)

All **14/14** jobs succeeded on [CI run 45076426514](https://github.com/aliharundaldalli/AhdCode/actions/runs/45076426514):

| Job | Runner / Platform | Duration | Status | Job ID |
| --- | --- | ---: | --- | --- |
| Core tests and static checks | Linux x64 (`ubuntu-latest`) | 6m41s | PASS | `137374548862` |
| Native build suite | Linux x64 (`ubuntu-latest`) | 22m9s | PASS | `137374548986` |
| Targeted race coverage | Linux x64 (`ubuntu-latest`) | 8m37s | PASS | `137374549124` |
| Native Linux ARM64 acceptance | Linux ARM64 | 1m38s | PASS | `137374549143` |
| Windows x64 verification and packaging | Windows x64 (`windows-latest`) | 4m54s | PASS | `137374549148` |
| macOS ARM64 verification and packaging | macOS ARM64 (`macos-15`) | 6m44s | PASS | `137374549140` |
| Native macOS x64 acceptance | macOS x64 (Intel) | 3m39s | PASS | `137374549074` |
| MySQL 8.4 integration | Linux x64 + MySQL 8.4 service | 2m13s | PASS | `137374549234` |
| PostgreSQL 17 integration | Linux x64 + PostgreSQL 17 service | 2m13s | PASS | `137374549258` |
| PostgreSQL 18 integration | Linux x64 + PostgreSQL 18 service | 2m19s | PASS | `137374549278` |
| Nested helper modules (`cmd/ahdgraphics`) | Linux x64 | 20s | PASS | `137374549117` |
| Nested helper modules (`cmd/ahdidentity`) | Linux x64 | 18s | PASS | `137374549146` |
| Nested helper modules (`cmd/ahdplotview`) | Linux x64 | 36s | PASS | `137374549193` |
| Nested helper modules (`cmd/ahdgui`) | Linux x64 | 25s | PASS | `137374549393` |

## 3. Native Cross-Platform & Packaging Verification
- **Windows x64 (`137374549148`):**
  - Built native `ahdcode.exe` and `ahdsqlite.exe`, passed Web/HTTPS/PWA/packaging/migration/LSP/analysis tests and `-race` tests (`cmd/ahdcode`, `ahdruntime`, `internal/packaging`).
  - Built and verified `AhdCode-2.12.0-windows-x64.exe` (SHA-256 `36cef020440ac3aa8c2d4c3425e9948d90408e73f418c3c752453f9470a8ce7c`, 210 files, 202.0 MiB, leak-gate PASS against 33 tracked AhdDataStudio files, `ahdcode/tooling/distribution/winsetup` PASS, installed CLI reported `AhdCode v2.12.0`).
- **Linux x64 (`137374548862`, `137374548986`, `137374549124`):**
  - Passed full workspace tests (excluding `internal/build`), `go vet ./...`, `gofmt -l`, `git diff --check`, 4/4 Node PWA worker tests, full `internal/build` native suite (`22m9s`), and targeted `-race` suite (`8m37s`).
  - Built and verified `AhdCode-2.12.0-linux-amd64.tar.gz`, checked `SHA256SUMS.txt`, installed into isolated prefix, ran native SQLite smoke (`linux-ok`), and uninstalled cleanly.
- **Linux ARM64 (`137374549143`):**
  - Native architecture verified, built CLI and SQLite helper (`AhdCode v2.12.0`), passed compiler, editor, runtime, build, and packaging regressions natively (`1m38s`).
- **macOS ARM64 (`137374549140`):**
  - Built native CLI and SQLite helper (`AhdCode v2.12.0`), passed Web/HTTPS/PWA/deployment ordinary and `-race` tests.
  - Built and verified all 6 unsigned macOS distribution artifacts (`tar.gz`, `pkg`, `dmg` for both `arm64` and `x64`), passed `leak_gate.py` (211–218 files, 33 AhdDataStudio files verified) and `test_macos_dist.py` (9/9 tests in `19.026s`), verified `.pkg` install/uninstall, and retained CI artifact `macos-unsigned-qa-5e3713248bf249a37ce9672c286fb3ca210ad238`.
- **macOS x64 (`137374549074`):**
  - Native Intel x86_64 verified, built CLI and SQLite helper (`AhdCode v2.12.0`), passed `cmd/ahdcode`, `internal/analysis`, `internal/lsp`, `ahdruntime`, `internal/build`, and `internal/packaging` regressions natively (`3m39s`).

## 4. Live Database Integration Results
- **MySQL 8.4 (`137374549234`):** Zero skipped MySQL tests; `TestMySQLIntegrationCRUDAndTypes`, `TestMySQLIntegrationTransactionsConcurrencyAndRedaction`, and `TestMySQLIntegrationNativeAndEvaluatorAgree` passed in ordinary mode and `ahdruntime` passed under `-race` (`1.074s`).
- **PostgreSQL 17 (`137374549258`) & PostgreSQL 18 (`137374549278`):** Full `ahdruntime`, `evaluator` (`TestPostgreSQLThroughEvaluator`), and `internal/build` (`TestPostgreSQLAcceptanceProgramCompiles`, `TestPostgreSQLIntegrationNativeAndEvaluatorAgree`, `TestPostgreSQLNativeProgramWithoutAServer`) suites passed in ordinary mode, plus `ahdruntime` under `-race` (`2.522s` / `2.564s`).

## 5. Phase 1–4 Regressions, Security & Skips Review
- **Phase 1 (Platform, LSP bounds, Private-File Authorization, SQLite Backup):** Verified across Linux, macOS, and Windows CI; `HTTP.file`/`HTTP.download` `no-store` defaults, public mount root/file identity checks, and bounded LSP workspace discovery all pass.
- **Phase 2 (`SQLite.openWithOptions` & Live MySQL):** Verified across native/evaluator parity, Windows/Linux/macOS, and live MySQL 8.4 CI.
- **Phase 3 (`ahdcode check` & Language Conformance):** Verified in `cmd/ahdcode` and core/native suites across all 5 native CI platforms.
- **Phase 4 (Security, Windows DACL Studio `.env`, LSP References Optimization):** Verified in Windows x64 and Linux/macOS CI; Studio `.env` exclusive creation and Windows DACL protections pass.
- **Documentation & Release Alignment:** `CHANGELOG.md`, `docs/RELEASE_2.12.0*.md`, `docs/UPGRADE_2.12*.md`, `docs/INSTALLATION*.md`, `docs/CLI*.md`, `docs/SQLITE*.md`, `docs/WEB*.md`, `README*.md`, and `internal/initweb/docbundle/*` are synchronized and pass `TestDocumentationBundle*`.

## 6. Remaining Known Limitations
- **macOS signing & notarization:** CI produces unsigned macOS QA artifacts (`macos-unsigned-qa-5e3713248bf249a37ce9672c286fb3ca210ad238`); Apple Developer ID signing and notarization remain a separate release-time step per build.
- **Windows ARM64:** Supported as a packager target, but native execution on Windows ARM64 hardware is not covered by GitHub Actions runners.
- **Mobile OS launchers:** Mobile Home Screen installation and launcher behavior remain unverified (outside v2.12 desktop/web scope).
- **CI runner annotation (non-blocking):** `actions/upload-artifact@v4` emitted a GitHub Node.js 20 deprecation notice (automatically run on Node.js 24).
- **Windows symlink privilege:** On unprivileged Windows hosts, file-symlink fixtures skip cleanly while Windows junction fixtures execute.

## 7. Findings Classification
- **BLOCKER:** 0
- **MAJOR:** 0
- **MINOR:** 1 — `actions/upload-artifact@v4` Node.js 20 deprecation warning in `.github/workflows/ci.yml` (non-blocking, runs on Node 24).
- **OBSERVATION:** 1 — Windows ARM64 native execution and Apple signing/notarization remain release-time / out-of-CI-runner items as documented in `docs/RELEASE_2.12.0.md`.

## 8. Phase 5 Verdict
**AhdCode v2.12.0 Phase 5 — PASS**

All 14 CI jobs (including Windows x64, Linux x64/ARM64, macOS ARM64/x64, MySQL 8.4, PostgreSQL 17/18, full `internal/build`, race coverage, installer/package verification, and nested helpers) passed on `5e37132`. Ready for your authorization to cut the v2.12.0 release candidate / release workflow.
