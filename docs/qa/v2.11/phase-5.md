# AhdCode v2.11 Phase 5 implementation and QA

The development worktree is on `v2.11-dev` at the published `v2.10.0` commit `5239c32e2996463b51d0f8288b3a433741767cca`. The rebuilt CLI reports `AhdCode v2.10.0`. The seven Phase 0 edits remain in the worktree. Phase 5 changes are unstaged and await independent Gemini QA.

## 1. CLI and application API

```text
ahdcode dev <entry.ahd>
ahdcode dev --https <entry.ahd>
ahdcode dev --https --cert <certificate.pem> --key <private-key.pem> <entry.ahd>
```

Place options before the entry file. HTTPS accepts a certificate and key together. Existing `HTTP.server`, `Web.app`, and start APIs serve through the selected listener. The Web configuration and Basic startup banner show the effective HTTPS address. Ordinary development defaults to HTTP.

## 2. Certificates and storage

The default store is `https/` under AhdCode's per-user local configuration directory. It contains `localhost.pem`, `localhost-key.pem`, and the public `root-ca.pem`. AhdCode uses cryptographically random ECDSA P-256 keys and serials, a 30-day server certificate, a 45-day restricted CA, and SANs for `localhost`, `127.0.0.1`, and `::1`. It never writes the CA signing key. The generated leaf's Authority Key Identifier matches the issuing CA's Subject Key Identifier.

The supervisor checks the pair and the generated store's issuing CA before replacing a working child. To rotate the default store, stop dev, move its `https` directory to a private location, restart dev, and explicitly replace the old public CA wherever you trusted it. An explicitly supplied pair can use mkcert or another local issuer. The private key and its immediate directory need restricted permissions. The key belongs outside the working directory and entry module directory.

## 3. Trust

The launcher prints the path to the public CA. Trust enrollment is a separate user decision. The test client trusted only the generated public CA for the request and verified the hostname. A fresh Edge profile refused the untrusted CA with `net::ERR_CERT_AUTHORITY_INVALID`. Verification stayed enabled, and the OS trust store and hosts file were untouched. See the [mkcert documentation](https://github.com/FiloSottile/mkcert) for an alternative trust setup.

## 4. Listener, lifecycle, and production boundary

The existing lifecycle binds the socket and wraps it using Go's `tls.NewListener`. It accepts TLS 1.2 and later and advertises HTTP/1.1 for WebSocket upgrades. Only `localhost`, `127.0.0.1`, and `::1` are accepted. The listener resolves literal `localhost` directly to IPv4 loopback. Port conflicts and invalid credentials roll back start ownership. The child enables transport from startup-only dev identity and certificate paths.

Direct HTTPS skips the local HTTP router and its `.test` domain setup. The Web CLI rejects production mode. Production HTTPS termination at a reverse proxy remains supported with `APP_PROTOCOL=https` for external URLs and secure cookies. See [Go's TLS listener](https://pkg.go.dev/crypto/tls#NewListener).

## 5. WebSocket

An HTTPS page uses WSS and the default browser origin rule checks for a matching HTTPS origin. Plain HTTP continues to support WS. Shutdown detaches and closes upgraded sockets outside the lifetime lock. An admitted callback keeps its scoped resources until it returns.

## 6. Browser reload

The existing reload script and polling endpoint use relative URLs. HTTPS uses the same session and generation protocol, polling cadence, and in-flight request guard as HTTP. In a real Basic CLI session, template and source edits advanced generations and changed the verified HTTPS response. A corrupt leaf and a corrupt default root each preserved the previous child. Restoring each certificate and editing the source allowed the next generation. The project's `.env` remained byte-for-byte identical.

## 7. Persistent sessions

The runtime preserves the application's cookie policy. The Basic starter derives `Secure` from the configuration's HTTPS protocol. Tests verified Secure, HttpOnly, SameSite, login rotation, previous-token invalidation, and scoped SQLite ownership through forced HTTP and WSS shutdown. Credentials and test token values are excluded from logs.

## 8. Native and evaluator coverage

Native verification launched the rebuilt CLI and a generated Turkish Basic project on IPv4 and IPv6. Strict OpenSSL verification passed with the default store and with its certificate and key supplied explicitly on IPv6. A separate evaluator subprocess exercised the same runtime startup transport over verified HTTPS, including a plain HTTP control without the development identity. The public `dev` command builds a native child, so there is no separate evaluator-facing `dev --https` flag. Standalone HTTP and Web programs also exercised the `.env` isolation test.

## 9. Security tests

| Area | Verification |
| --- | --- |
| Certificate lifecycle | Fresh and reused store, expiry, not-yet-valid dates, missing certificate and key, wrong key, malformed PEM, corrupt default CA, and Authority Key Identifier |
| Identity and transport | Loopback SAN verification, wrong hostname, untrusted CA, plaintext refused on the TLS port, correct HTTPS URL, and rejection of non-loopback and production Web configuration |
| Filesystem | Private key refused inside a project, Windows junction and broad ACL refused, certificate PEM accepts certificates only, and a malformed private-key marker stays out of diagnostics |
| Resource ownership | Port conflict and failed credentials release startup ownership, repeated lifecycle races stay clean, and active HTTP and WSS callbacks retain scoped SQLite ownership until completion |
| HTTPS application | WSS echo and origin rejection, secure session rotation, Turkish Basic HTML and 404/500 pages, relative reload assets, and no private-key response from either tested public path |

## 10. Ordinary and race-detector commands

Windows amd64, Go 1.26.7, with `GOFLAGS=-buildvcs=false` for temporary QA builds. The commands below use PowerShell. The JSON test streams record test and subtest names with their results.

```powershell
$packages = './cmd/ahdcode ./internal/backend/golang/ahdruntime ./internal/backend/golang ./internal/evaluator ./internal/build ./internal/initweb' -split ' '
$focus = '^Test(DevHTTPS|HTTPSConfiguration|HTTPSIsRefused|ProductionConfiguration|DevWatcher|DevReload|HTTP(Endpoint|Shutdown|Publication|Registry|Wildcard|Nested|SameServer|BindFailure|CallbackRetirement|ErrorPage|Session|NativeAndEvaluator)|WebSocket|PersistentSession|MemorySession|Migrate(DoesNotDisturb|ExampleApp|CLI)|Templates|Template(Windows|Program)|HTTPEvaluator|Evaluator(HTTP|Persistent|CrossServer|Yield|NestedServer|Unstarted)|BasicStarter|WebConfig|GeneratedProgramCarriesRuntime|DocumentationBundle|AllStartersReceiveDocumentationBundle)'
$repeat = '^Test(DevHTTPS(StartupRollsBack|WebSocketShutdown|ForcedHTTPShutdown)|HTTPBindFailureClears|HTTPCallbackRetirement|HTTPShutdownClaimsStarted|WebSocketClockIsImmutable)'
go test -json -count=1 -run $focus @packages
go test -json -race -count=1 -run $focus @packages
go test -json -count=1 -run $focus -skip '^TestHTTPSessionNativeRelocatesWithoutToolchain$' @packages
go test -json -race -count=1 -run $focus -skip '^TestHTTPSessionNativeRelocatesWithoutToolchain$' @packages
go test -json -count=6 -run $repeat ./internal/backend/golang/ahdruntime
go test -json -race -count=6 -run $repeat ./internal/backend/golang/ahdruntime
go test -json -count=1 ./internal/migrate
go test -json -race -count=1 ./internal/migrate
```

The original regression filter reported one test failure, the Windows relocation fixture, in each mode. With that fixture explicitly excluded, the six-package filter passed 236 tests and subtests and recorded two privilege-related skips per mode. The repeated lifecycle filter passed 60 tests and subtests per mode. The full migration package passed 118 tests and subtests and recorded two privilege-related skips per mode.

## 11. Windows failures and skips

- `TestHTTPSessionNativeRelocatesWithoutToolchain` fails when the fixture executes a relocated, extensionless binary on Windows. The same failure was reproduced on the baseline in ordinary and race modes. The fixture is unchanged.
- The host lacks file symlink creation privilege. `TestDevHTTPSInvalidCredentials/linked_key`, `TestTemplateWindowsLeafOpenRejectsReplacement/file_symlink`, `TestMigrateRejectsSymlinkLock`, and `TestMigrateRejectsASymlinkOutsideTheDirectory` skipped. The Windows HTTPS junction and template leaf-junction cases ran.

## 12. Changed files and worktree

Paths are relative to the development worktree.

- CLI and certificate management — `cmd/ahdcode/{dev.go,main.go,webdev.go,devhttps.go,devhttps_unix.go,devhttps_windows.go,devhttps_test.go,devhttps_windows_test.go,devhttps_evaluator_test.go}`.
- Listeners and sockets — `internal/backend/golang/ahdruntime/{http.go,http_devreload.go,http_devhttps_test.go,websocket.go,websocket_conn.go}`. The HTTP and WebSocket runtime files also carry Phase 0 changes.
- Application and build coverage — `internal/framework/sources/WebConfig.ahd`, `internal/initweb/sources.go`, and `internal/build/devreload_isolation_test.go`.
- Documentation — `README.md`, `README_TR.md`, `docs/{CLI.md,CLI_TR.md,WEB.md,WEB_TR.md}`, `internal/initweb/docbundle/{CLI.md,README.md,WEB.md}`, and this report.

Phase 0-only edits remain in `ROADMAP_3.0_TR.md`, `docs/qa/v2.11/phase-0.md`, and `internal/backend/golang/ahdruntime/{http_lifecycle_test.go,http_phase0_test.go,websocket_test.go}`. The original Desktop checkout, published release, and compiler version remain as found.

## 13. Linux CI commit

`3577cbf` is pending. It separates `internal/build` into its own workflow run and restricts a Windows junction fixture in `internal/migrate` to Windows. After review and authorization to commit, preserve the approved local changes in commits on `v2.11-dev`, then merge `main` or cherry-pick `3577cbf` if that is the only wanted change. Review both changed CI paths and run the Linux workflow. No integration operation was performed during Phase 5.

## 14. Limits of verification

- Browser coverage verified the expected untrusted-CA interstitial. Trusted browser rendering, browser execution of the reload script, and mixed-content behavior in that trusted browser remain for independent QA. A CA-pinned client verified the delivered relative reload assets and state transitions.
- Direct TLS is loopback-only and uses HTTP/1.1. The Windows smoke sessions exercised IPv4 and IPv6 with verified clients. Linux and macOS execution and the external mkcert issuance workflow remain for additional QA.
- Certificate changes are checked at startup or a source-triggered rebuild. The listener continues to serve its loaded certificate until replaced. Parent path checks are path-based and the private store or key directory must remain controlled by its owner.
- The runtime shutdown tests exercised real active callbacks. The existing Windows CLI process supervisor uses process termination when stopping a child.

## 15. Suggested Gemini checks

- Launch Basic with both certificate modes. With an explicitly trusted test CA in an isolated environment, confirm browser secure context, the reload script in DevTools, no mixed-content errors, and WS and WSS behavior.
- Compare HTTP and HTTPS sessions and custom 404/500 pages. Rotate a persistent session, try the old token, and stop a server with a live WSS callback.
- Validate default CA expiry and regeneration, explicit PEM mismatch, restricted Windows ACLs, and Unix path containment. Test `localhost`, IPv4, IPv6, and a port conflict.
- Repeat Phase 0 lifecycle cases with the race detector, run the Linux CI coverage after approval, and verify the compiler still reports `AhdCode v2.10.0`.
