# AhdCode v2.12.0 — Standalone Checker, SQLite Options & Platform Reliability

Türkçe · [Installation](INSTALLATION.md) · [Upgrade](UPGRADE_2.12.md)

- `ahdcode check app.ahd --format json` runs module resolution, parsing and semantic analysis. It emits schema version 1 diagnostics with UTF-16 ranges and works without an installed Go toolchain.
- `SQLite.openWithOptions(path, readOnly, busyTimeoutMilliseconds)` adds read-only opens and an explicit lock wait. Existing `SQLite.open` defaults remain available. Backups use a consistent snapshot and reject an existing destination.
- LSP workspace discovery is bounded. Rename refuses incomplete discovery and applies edits to the source that owns each span, including `require(...)` files.
- HTTP public mounts check root and file identity. Explicit file responses default to `Cache-Control: no-store`. Applications supply server-side ownership checks and mark authorization denials uncacheable.
- Studio creates a missing `.env` with exclusive creation and private permissions. Windows GUI resources and HTTP public mounts resolve directory junctions through the opened handle, and disk inspection accepts existing file paths.
- Native packaging keeps the v2.11 toolchain layout. macOS `.pkg` and `.dmg` installers are Apple Developer ID signed, Apple-notarized, ticket-stapled, and Gatekeeper-verified; `.tar.gz` archives carry signed Mach-O binaries, and Windows installers and Linux archives are distributed with `SHA256SUMS.txt` verification manifests.

The release pipeline targets Windows x64, Linux amd64/arm64 and macOS arm64/x64. A cross-built artifact still requires native validation on its target. Toolchain-only packages need the Go version in `go.mod` for build and packaging commands. Optional GUI/graphics helpers need their platform display environment.
