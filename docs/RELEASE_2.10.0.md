# AhdCode v2.10.0 — Web Development & Application Reliability

[English] · [Türkçe](https://github.com/aliharundaldalli/AhdCode/blob/v2.10.0/docs/RELEASE_2.10.0_TR.md)

v2.10.0 is a stable source-only release. GitHub provides ZIP and tar.gz
source archives. The latest signed installers are v2.7.0. Build and verify
v2.10.0 from the tagged checkout with the Go version specified in go.mod.

```sh
go build -o ahdcode ./cmd/ahdcode
./ahdcode version
```

On Windows, use `ahdcode.exe` as the output name. The version command prints
`AhdCode v2.10.0`. Persistent sessions and other SQLite-backed native programs
need the matching `ahdsqlite` helper. Build it with `go build -o ahdsqlite
./cmd/ahdsqlite` and keep it beside the compiler or built application. Use the
`.exe` suffix on Windows.

## Web and HTTP

- `ahdcode dev` reloads eligible HTML pages for loopback development clients
  after successful rebuilds. It watches HTML under `templates/` for creates,
  edits, deletions, restorations, and renames.
- HTTP, WebSocket, and evaluator synchronization improvements cover handler
  lifecycle, server state, shutdown, and concurrent requests.
- `HTTP.persistentSessions` and `Web.persistentSessions` store committed
  session values and revocations in SQLite across process restarts. Session
  rotation invalidates the old identifier. Existing in-memory session
  stores remain available.
- `server.notFound` and `server.internalError` customize 404 and 500 responses.
  Their status stays fixed. If a custom error handler fails, the runtime
  returns the default error response without exposing the failure.
- `ahdcode init web basic` generates a Turkish site with pages, a reusable
  navigation component, a shared layout, and custom error pages.

## SQLite migrations

Run these commands from a project directory. `new` creates an up/down SQL
pair in `migrations/`. Edit both files before applying the migration.

```sh
ahdcode migrate new create_notes
ahdcode migrate status --db notes.db
ahdcode migrate up --db notes.db
ahdcode migrate down --db notes.db --steps 1
```

Commit both SQL files with the app. Back up an existing database before
applying or reversing migrations. Use a separate SQLite file for persistent
sessions.

## Templates and layouts

A layout such as `templates/layouts/site.html` can contain a block.

```html
<main>{{block "content" .}}{{end}}</main>
```

The selected page, `templates/pages/home.html`, supplies that block.

```html
{{define "content"}}<h1>{{.title}}</h1>{{end}}
```

```ahd
bring Templates
from Templates bring TemplateSet

pages: TemplateSet := Templates.files("templates")
markup: String := pages.layout("layouts/site.html", "pages/home.html", Templates.object({
    "title": Templates.fromString("Ana sayfa")
}))
```

Ordinary data is escaped. Layouts require a shared layout name and a selected
page under `pages/`. Each template owns its filename, and shared files cannot
define names under `pages/`. Rename any conflicting definitions before
upgrading a development build.

## Known limitations

- The migration CLI supports SQLite. MySQL and PostgreSQL migrations are
  outside this release.
- Full-package tests on Windows still have known executable-path, permissions,
  symlink, and platform-fixture failures. Windows may skip symlink fixtures
  when the process lacks permission to create links.
- Template leaf opens reject reparse points on Windows. Ancestor directory
  checks are path-based, leaving a time-of-check/time-of-use window if an
  ancestor is replaced.
- `ahdcode package` does not automatically bundle external `templates/` or
  `public/` directories. Deploy them with the application and set its
  working directory appropriately.
- The final independent release QA passed with three non-blocking minor
  findings: a narrow shutdown TOCTOU window between the HTTP/WebSocket `closing`
  check and callback entry (no data race, SQLite corruption, or permanent
  handle leak; deferred for hardening), a narrow pre-listen shutdown race on a
  concurrent `net.Listen` bind failure before ownership is cleared, and a
  pre-existing test-only timing flake in
  `TestWebSocketClockIsImmutablePerConnection` (production code never mutates
  the WebSocket clock).
- Local HTTPS, PWA functionality, broader runtime consolidation, and future
  language specification work are deferred.

[CLI](https://github.com/aliharundaldalli/AhdCode/blob/v2.10.0/docs/CLI.md) · [HTTP](https://github.com/aliharundaldalli/AhdCode/blob/v2.10.0/docs/HTTP.md) · [Templates](https://github.com/aliharundaldalli/AhdCode/blob/v2.10.0/docs/TEMPLATES.md) · [Web](https://github.com/aliharundaldalli/AhdCode/blob/v2.10.0/docs/WEB.md)
