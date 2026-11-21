# AhdCode v2.11.0 — Web Deployment & Progressive Web Apps

[Türkçe](RELEASE_2.11.0_TR.md)

AhdCode `v2.11.0` ships toolchain distribution packages for macOS, Windows,
and Linux, as well as source builds from the `v2.11.0` tag with the Go version
specified in `go.mod`:

```sh
go build -o ahdcode ./cmd/ahdcode
go build -o ahdsqlite ./cmd/ahdsqlite
./ahdcode version
```

On Windows, use `ahdcode.exe` and `ahdsqlite.exe`. The version command
reports `AhdCode v2.11.0`. Keep the matching helpers beside the CLI when
running from source. Toolchain packages require the Go version specified in
`go.mod` on `PATH` for native `ahdcode build` and `ahdcode package`; full
self-contained bundles with private Go, offline Tectonic LaTeX, and VSIX remain
available on `v2.7.0` and return at `v3.0`. The Web application package built
by `ahdcode package app.ahd --web` is a separate deployment artifact.

## Multi-platform distribution packaging

`tooling/distribution/build.py` produces native toolchain packages for macOS,
Windows, and Linux:

- **macOS (`arm64` and cross-compiled `x64`):**
  - `AhdCode-2.11.0-macos-<arch>.pkg`: home-directory `.pkg` installer
    (`auth="none"`, no admin password), preserving prior versions such as
    `2.9.0`, updating `current` and `~/Library/AhdCode/bin/ahdcode`,
    idempotently updating shell PATH profiles, and bundling `install.sh` for
    `--uninstall`. Signed with Apple Developer ID Installer and notarized by
    Apple with a stapled ticket.
  - `AhdCode-2.11.0-macos-<arch>.dmg`: UDZO disk image containing the `.pkg`,
    `INSTALL.txt`, and `VERSION`. Signed with Apple Developer ID Application
    and notarized/stapled.
  - `AhdCode-2.11.0-macos-<arch>.tar.gz`: portable toolchain + `install.sh`
    with Developer ID Application signed Mach-O binaries (hardened runtime and
    secure timestamp).
- **Windows (`x64`):** `AhdCode-2.11.0-windows-x64.exe` per-user installer and
  stable launcher (`--silent` supported). It is not Authenticode-signed; verify
  its SHA-256 digest before running.
- **Linux (`amd64` and `arm64`):** `AhdCode-2.11.0-linux-amd64.tar.gz` and
  `AhdCode-2.11.0-linux-arm64.tar.gz` with `install.sh` (`--setup-path`,
  `--prefix`, `--uninstall`).
- **`SHA256SUMS.txt`:** SHA-256 digests for every published artifact.

When `--sign-app`, `--sign-installer`, and `--notary-profile` are supplied,
artifacts enforce Apple's standard `codesign --verify --deep --strict`,
`pkgutil --check-signature`, `xcrun notarytool submit --wait`,
`xcrun stapler staple`/`validate`, and Gatekeeper `spctl` assessment:
`.tar.gz` carries Developer ID signed Mach-O binaries with hardened runtime and
timestamp, while `.pkg` and `.dmg` artifacts are signed, Apple-notarized, and
stapled. Without notarization credentials, builds are clearly labeled
`-unsigned` and never bypass Gatekeeper. Uninstall with:

```sh
sh "$HOME/Library/AhdCode/current/install.sh" --uninstall
```

## Changes since v2.10.0

- Added loopback development HTTPS with an optional certificate and key.
  Trust the local CA separately in the browser when needed. Production
  deployments can continue to terminate HTTPS at Caddy.
- Hardened callback ownership during HTTP and WebSocket shutdown. Scoped
  SQLite resources stay open until admitted callbacks complete.
- Added optional browser installation and a limited offline fallback with
  `ahdcode pwa init`. The generated worker caches only explicitly selected,
  integrity-pinned public assets. It leaves AhdCode live-reload requests alone.
- Added `ahdcode package app.ahd --web` and optional `--pwa` to deploy a
  compiled application alongside its reviewed public assets. Existing
  Web destinations are protected against replacement. PWA manifest and
  worker routes support both GET and HEAD.
- Removed the public Templates builtin. Basic now uses `.ahd`, Web.UI,
  and HTMLNode for pages, layouts, and components. Applications using the
  former Templates API need the documented migration.

Browser installation and offline behavior vary by platform. Keep private
configuration, credentials, and databases in deployment-managed storage.
See [packaging and deployment](PACKAGING.md), [PWA setup](PWA.md), and
[the Templates migration guide](UPGRADE_2.11.md).
