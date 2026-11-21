# AhdCode v2.11.0 — Web Deployment & Progressive Web Apps

Türkçe

The toolchain release is source-only. From the v2.11.0 source, build with
the Go version specified in `go.mod`:

```sh
go build -o ahdcode ./cmd/ahdcode
go build -o ahdsqlite ./cmd/ahdsqlite
./ahdcode version
```

On Windows, use `ahdcode.exe` and `ahdsqlite.exe`. The version command
reports `AhdCode v2.11.0`. Keep the matching SQLite helper beside the CLI
for applications that need it. Existing published signed installers remain
v2.7.0 until a final signed release is published. The Web application
package built by the toolchain is a separate artifact.

## macOS distribution packaging (.pkg, .dmg, .tar.gz)

`tooling/distribution/build.py --platform darwin` produces macOS ARM64 (and
optional cross-compiled `x86_64`) toolchain packages without altering
Windows or Linux release behavior:

- `AhdCode-<version>-macos-<arch>[-unsigned].tar.gz`: portable toolchain +
  `install.sh`, installing into `~/Library/AhdCode/versions/<version>`.
- `AhdCode-<version>-macos-<arch>[-unsigned].pkg`: native home-directory
  `.pkg` installer (`auth="none"`, no admin password), preserving previously
  installed versions such as `2.9.0`, updating `current` and
  `~/Library/AhdCode/bin/ahdcode`, idempotently updating shell PATH profiles,
  and bundling `install.sh` for `--uninstall`.
- `AhdCode-<version>-macos-<arch>[-unsigned].dmg`: UDZO disk image containing
  the `.pkg`, `INSTALL.txt`, and `VERSION`.
- `SHA256SUMS.txt`: SHA-256 digests for every generated artifact.

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
