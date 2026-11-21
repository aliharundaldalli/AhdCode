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
for applications that need it. Existing signed installers remain v2.7.0.
The Web application package built by the toolchain is a separate artifact.

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
