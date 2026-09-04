# AhdDataStudio

AhdDataStudio is a **local-only** database development application. It is
not a compiler builtin, not an ORM, and not a database abstraction layer.
It is a first-party AhdCode program that uses the released `HTTP`, `HTML`,
`MySQL`, `SQLite`, `Env`, `File`, `Path`, and `Security` modules.

## Start

```bash
cd tools/AhdDataStudio
cp .env.example .env   # then edit placeholders
ahdcode run app.ahd
```

Open:

[http://127.0.0.1:8081/AhdDataStudio](http://127.0.0.1:8081/AhdDataStudio)

The server binds **127.0.0.1:8081** only. It is a local development tool
and must not be exposed on `0.0.0.0` or the public internet.

There is no `ahdcode studio` command in this version.

## MySQL

Configure through Env (or `.env` in this directory):

| Variable | Default | Meaning |
|---|---|---|
| `AHD_DATA_MYSQL_HOST` | `127.0.0.1` | MySQL host |
| `AHD_DATA_MYSQL_PORT` | `3306` | MySQL port |
| `AHD_DATA_MYSQL_USER` | `root` | username |
| `AHD_DATA_MYSQL_PASSWORD` | empty | password — never committed |
| `AHD_DATA_MYSQL_SECURITY` | `none` | `none` or `tls` |

The Studio connects with `database: null`, then lists every schema those
credentials can see (`SHOW DATABASES` / `INFORMATION_SCHEMA`). MySQL
permissions are respected; the Studio does not bypass them.

Table metadata comes from ordinary SQL: `INFORMATION_SCHEMA.TABLES`,
`COLUMNS`, and `STATISTICS`. Browse uses `LIMIT` 50. `TABLE_ROWS` is shown
as an estimate when MySQL provides one, not as an exact `COUNT(*)`.

Studio-generated INSERT / UPDATE / DELETE bind values with `?`. Identifiers
are taken from discovered metadata and quoted. They are never taken from
untrusted form text and concatenated as raw SQL.

## SQLite

SQLite files appear only when:

- listed in `AHD_DATA_SQLITE_PATHS` (comma-separated), and/or
- they are immediate children of `AHD_DATA_PROJECT_ROOT` with extension
  `.db`, `.sqlite`, or `.sqlite3`

There is **no machine-wide scan**, no recursive walk, and no password
file search. A query/form path is accepted only if it matches that
allowlist. `..` components and directory targets are rejected.

SQLite in generated AhdCode programs uses the bundled `ahdsqlite` helper.
If the helper is missing, the first `SQLite.open` raises `SQLiteError`
explaining `AHDCODE_SQLITE_RUNTIME`. Install it next to the compiler:

```bash
go install ./cmd/ahdcode ./cmd/ahdsqlite
```

SQLite `BLOB` values are not a Studio value kind: querying a BLOB raises
`SQLiteError`, matching the released SQLite contract. Browse omits columns
whose declared type contains `blob`.

## SQL console

The SQL console runs the SQL you type. Destructive statements are **not**
extra-confirmed. That is intentional for a local admin tool.

Studio-generated forms (insert/update/delete/drop/truncate) are different:
they use POST, CSRF tokens (`Security.token` + `Security.secureEqual`),
and a confirmation step for DROP/TRUNCATE.

## Security posture

- Local bind only (`127.0.0.1`)
- No credential scanning
- Database cell values, names, and SQL errors render through `HTML.text`
- MySQL passwords are never written back into HTML
- Connection failure on MySQL does not disable SQLite, and the reverse

## Current limitations (why later Web work exists)

This first version is one `app.ahd` because local source `require(...)`
does not exist yet. Routes are exact paths; there is no Pages/Components
layer, no static-asset pipeline, and no `Web.UI`. CSS is a trusted raw
String inside the HTML builder. Path/File have no `realpath` / symlink
API, so SQLite discovery is name-based and non-recursive.

Those constraints are product evidence for a future Web facade, not
features of this release.

## Language

[English](README.md) · [Türkçe](README_TR.md)
