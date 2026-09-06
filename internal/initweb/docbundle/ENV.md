# Env standard module

[English] · [Türkçe](ENV_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [File and Path](FILESYSTEM.md)

`Env` is the compiler-registered `builtin:Env` module. It is explicit and a
sibling `Env.ahd` cannot shadow it:

```ahd
bring Env
from Env bring EnvError
```

Env stays small on purpose: process environment variables and a bounded
`.env` file format, both as plain `String`s, with no numeric/boolean
inference, no shell interpolation, and no command execution anywhere in it.

## Surface

```text
Env.get(name: String)    -> String?
Env.getOr(name: String, fallback: String) -> String
Env.exists(name: String) -> Bool

Env.set(name: String, value: String) -> Nothing
Env.unset(name: String)  -> Nothing

Env.read(path: String)   -> Pair<String, String>
Env.load(path: String, override: Bool = false) -> Nothing
```

(`Env.exists`, not `Env.has`: `has` is a reserved keyword — the `x has y`
protocol operator — and cannot appear as a member name after `.`; `exists`
matches the existing `File.exists` naming.)

## get, getOr, exists

`Env.get(name)` returns `String?`: `null` means the variable is absent.
`Env.exists(name)` distinguishes absence from an explicitly present but
empty value — both are real, different states:

```ahd
found: String? := Env.get("PORT")
if found != null {
    write(found)
}
write(Env.exists("PORT"))
```

`Env.getOr(name, fallback)` returns `fallback` only when the variable is
absent; an explicitly present empty `String` is returned as `""`, not
replaced by the fallback. There is no automatic numeric or boolean
conversion — convert explicitly:

```ahd
port: Int := int(Env.getOr("PORT", "8080"))
```

## set and unset

`Env.set`/`Env.unset` change the current AhdCode process's own environment;
later `Env.get`/`Env.exists` calls, and child processes launched
afterward, see the change where OS semantics allow it. A name is validated
before anything is changed: it must be non-empty and must not contain a NUL
byte or `=`. Values are never logged in error messages.

## The `.env` format

```text
KEY=value
KEY="value"
KEY='value'

# full-line comment

EMPTY=
```

- A key matches `[A-Za-z_][A-Za-z0-9_]*`.
- Leading whitespace on a line is ignored; a line whose first non-whitespace
  character is `#` is a full-line comment; a blank line is ignored.
- An unquoted value is everything after `=` to end of line, with leading
  and trailing whitespace trimmed.
- A double-quoted value supports exactly `\\`, `\"`, `\n`, `\r`, and `\t`;
  any other escape is rejected. A single-quoted value is literal — nothing
  after the opening quote is treated specially until the matching closing
  quote.
- Nothing may follow a quoted value's closing quote except trailing
  whitespace.

There is deliberately no `$(...)`, `` `...` ``, `${...}`, `$NAME`, or any
other shell-style expansion: a `.env` value is read as literal text, never
evaluated, and never spawns a process.

```ahd
Env.load(".env")
databasePath: String := Env.getOr("DATABASE_PATH", "app.db")
```

## read vs. load

`Env.read(path)` parses a file and returns it as an insertion-ordered
`Pair<String, String>`, without touching the process environment at all.

`Env.load(path, override = false)` parses the entire file, validates it
completely, and only then applies it — a malformed later line can never
leave the process half-updated from the same call. With `override = false`
(the default), an already-present variable — checked the same
absent-vs-empty-aware way `exists()` is — is left untouched: the existing
process/OS environment wins over the file. With `override = true`, the
`.env` value always replaces whatever was there.

```ahd
Env.load(".env")            -- existing environment wins
Env.load(".env", true)      -- .env always wins
```

Duplicate keys within one `.env` file are rejected with `EnvError` rather
than silently letting the last one win.

## Errors

`EnvError` derives directly from `Error` and covers: a missing/unreadable
`.env` file, a malformed assignment, an invalid key, an unterminated quoted
value, a duplicate key, an invalid escape sequence, and an OS-level
`set`/`unset` failure. Error messages never include the variable's value.

```ahd
attempt {
    Env.load("missing.env")
} except EnvError as error {
    write(error.message)
}
```

## Variables the AhdCode toolchain itself reads

These are read by the `ahdcode` command line, not by the `Env` module. They
are listed here because this is where people look for them.

### Application configuration

`ahdcode dev` reads `APP_NAME`, `APP_ENV`, `APP_HOST`, `APP_PROTOCOL`,
`SERVER_HOST`, and `SERVER_PORT` with the application's own precedence — the
process environment first, then the app-root `.env` — and only ever to decide
what to print and whether the session may run. It never exports one and never
passes one to the child. See [Web](WEB.md) for what each means.

`APP_HOST` additionally determines the local `.test` name a development
session is routed at: `ahdakademi.com` develops as `ahdakademi.test`. See
[CLI](CLI.md#local-development-test-names-and-the-router).

### Local development

| Variable | Read by | Meaning |
| --- | --- | --- |
| `AHDCODE_LOCAL_HOME` | CLI | Overrides the per-user directory holding the route and database registries. Must be absolute. Ordinary use never sets it. |
| `AHDCODE_LOCAL_ROUTER_PORT` | CLI | The local router's fallback port when port 80 cannot be bound. Defaults to `7357`. A value outside 1–65535 is ignored rather than fatal. |
| `AHDCODE_ROOT` | CLI | Developer override: use `tools/AhdDataStudio` from this source checkout instead of the bundled exact-version Studio. Ordinary installed use leaves it unset. |
| `AHDCODE_STUDIO_CACHE` | CLI | Absolute directory for the materialized Studio cache. Tests and packaging use this; ordinary use leaves it unset. |
| `AHDCODE_SQLITE_RUNTIME` | runtime | Path to the bundled `ahdsqlite` helper, when it is not installed alongside `ahdcode`. |

### AhdDataStudio

| Variable | Meaning |
| --- | --- |
| `AHD_DATA_REGISTRY` | Path to the AhdCode database registry. Set automatically by `ahdcode databases`; Studio reads registered SQLite databases from it. |
| `AHD_DATA_SQLITE_PATHS` | Comma-separated explicit SQLite files. Still supported, and no longer something you have to edit by hand — see `ahdcode databases add`. |
| `AHD_DATA_PROJECT_ROOT` | A folder whose **immediate** `.db`/`.sqlite`/`.sqlite3` children are listed. Not recursive, and never walked outside that root. |
| `AHD_DATA_MYSQL_HOST` | MySQL server address only (`127.0.0.1`), never `host:port`. |
| `AHD_DATA_MYSQL_PORT` | MySQL port. |
| `AHD_DATA_MYSQL_USER`, `AHD_DATA_MYSQL_PASSWORD` | MySQL credentials. These live in Studio's `.env` and are deliberately **not** in any AhdCode registry. |
| `AHD_DATA_MYSQL_SECURITY` | `tls` or `none`. |

Studio combines the three SQLite sources — registry, `AHD_DATA_SQLITE_PATHS`,
`AHD_DATA_PROJECT_ROOT` — deduplicated in that fixed order. Nothing recursive
and nothing machine-wide is ever scanned.

### Web runtime limits

`Web.app` applies these after the HTTP server is created. Unset uses the
default. A malformed value fails the process; it is never silently ignored.

| Variable | Default | Meaning |
| --- | --- | --- |
| `AHD_WEB_MAX_BODY_SIZE` | `16MB` | Maximum request body. Units are 1024-based (`B`, `KB`, `MB`, `GB`). |
| `AHD_WEB_MAX_UPLOAD_SIZE` | `8MB` | Maximum size of one uploaded file. Must not exceed `AHD_WEB_MAX_BODY_SIZE`. |
| `AHD_WEB_MAX_UPLOAD_FILES` | `10` | Maximum number of uploaded files in one request. |
| `AHD_WEB_READ_TIMEOUT` | `30s` | Server read timeout (`ms`, `s`, `m`). |
| `AHD_WEB_WRITE_TIMEOUT` | `30s` | Server write timeout. |
| `AHD_WEB_IDLE_TIMEOUT` | `60s` | Server idle timeout. |

A raw `HTTP.server` keeps its historical 1MiB body default and 15s/15s/60s
timeouts until `applyWebLimits()` is called.
