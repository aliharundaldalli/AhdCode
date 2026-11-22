# CLI

[English] · Türkçe

[Back to README](README.md) · [Formatter](FORMATTER.md) · [REPL](REPL.md) · [Language server](LSP.md)

The current command surface is:

```text
ahdcode
ahdcode init web [empty|basic|admin|mvc|crud]
ahdcode pwa init [--name NAME] [--short-name NAME]
ahdcode databases
ahdcode databases list
ahdcode databases add <file.db>
ahdcode databases remove <file.db>
ahdcode migrate new <name> [--dir <directory>]
ahdcode migrate status --db <file> [--dir <directory>] [--driver sqlite]
ahdcode migrate up --db <file> [--dir <directory>] [--driver sqlite]
ahdcode migrate down --db <file> [--dir <directory>] [--steps <n>] [--driver sqlite]
ahdcode build <entry.ahd> [-o <output>]
ahdcode check <entry.ahd> [--format text|json]
ahdcode package <entry.ahd> [--name <name>] [--output <folder>] [--icon <icon.png>]
                [--target <os-arch>] [--helpers <folder>] [--console] [--web [--pwa]]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode dev [--https [--cert FILE --key FILE]] <entry.ahd>
ahdcode stop <app.dev|app.run>
ahdcode kill [--force] <app.dev|app.run>
ahdcode local status
ahdcode local hosts [apply|remove]
ahdcode format [--check] <file.ahd|folder>...
ahdcode lsp
ahdcode --help
ahdcode --version
```


`ahdcode init web` writes a Web starter into the **current directory**. On a
TTY it asks Empty, Basic, Admin, MVC, or CRUD. Non-interactive use must pass
`empty`, `basic`, `admin`, `mvc`, or `crud`. Starter sources, Bootstrap 5.3.3, and
the AhdCode logo are embedded: offline, no package manager, no overwrite.
Every starter also receives `AHDCODE.md` and an English `Documents/AhdCode/`
snapshot for this version. Those files are not runtime. Next:
`ahdcode dev app.ahd`.

`package` (v2.0) makes a self-contained desktop application from an entry
module: a macOS `.app`, or a Windows or Linux folder and archive, holding the
compiled program and only the AhdCode helpers it uses, so it runs without
AhdCode installed. See [Packaging](PACKAGING.md).

`run` compiles through the normal frontend and Go backend, then executes the
native result. Arguments after the entry (optionally after `--`) are forwarded
to the generated process, although v0.1 publishes no language-level argument
API yet.

While `run` is running, it keeps a small `app.run` descriptor beside the
entry module (`app.ahd` produces `app.run` in the same directory) and removes
it when the run ends. `kill` uses that descriptor to stop the application:

```bash
ahdcode run app.ahd
ahdcode kill app.run
```

That replaces looking the process up by port with `lsof -i :8080` and then
`kill <pid>`. `kill` requests a graceful stop; `ahdcode kill --force app.run`
stops the application immediately.

**`kill` never signals the process id written in the run file.** A process id
in a file proves nothing: anyone who can write the file can name an unrelated
process, and operating systems reuse ids, so a stale descriptor can come to
name something else entirely. Instead, a live `ahdcode run` listens on a
loopback-only control port and holds a 256-bit random token, and the
descriptor records how to reach it. `kill` connects to `127.0.0.1` on that
port, presents the token, and the running supervisor terminates the child
process it started and owns.

The consequences are the point:

- a forged descriptor naming an unrelated live process stops nothing, because
  no supervisor answers for it;
- a recycled process id is harmless for the same reason;
- a wrong token is refused and nothing is stopped;
- a descriptor with no live supervisor is reported as stale and removed, with
  no process signalled;
- a file that is not a well-formed AhdCode run descriptor — including a bare
  pid — is refused outright.

`--force` changes only how the supervisor terminates its own child; it never
restores direct signalling from the file. Starting a second `run` while a
descriptor's supervisor still answers fails with the pid and the `kill`
command to use, instead of silently colliding on the port; a descriptor whose
supervisor is gone is cleared so the new run can proceed.

The descriptor is internal CLI metadata, not a language-level format: nothing
in the standard library reads or writes it, and it is written `0600` because
it carries a control capability.

`build` prints the produced executable path. Without `-o`, the compiler uses
the entry module's base name in the current working directory.

### `dev`: watch, rebuild, restart

`dev` runs `build` and `run` in a foreground watch loop, the same way a
MAMP/Vite-style dev server works, entirely as orchestration around the
existing build pipeline — it is not a second compiler:

```bash
ahdcode dev app.ahd
```

It compiles the entry module, starts the result, and then watches it. On
every save it rebuilds:

- if the rebuild **succeeds**, the previously running process is stopped and
  the new one takes its place;
- if the rebuild **fails**, the diagnostics print in place and the
  previously running (last-good) process is left running untouched — a
  broken save never takes down a working session, including the very first
  build;
- if the running process exits on its own after a successful build (a
  runtime crash, for instance), `dev` reports it and goes back to waiting
  for the next save; it does not loop retrying the same broken binary.

A process started by `dev` receives `AHDCODE_DEV_SESSION` and
`AHDCODE_DEV_GENERATION` before it starts. The HTTP runtime reads that pair
once, at process start. `Env.load`, `Web.configure`, and `Web.start` run
later, so a `.env` file cannot turn the hook on and cannot change a session
that `dev` already set. `ahdcode run` does not add the pair.

Production warning: a standalone process activates the hook when its
startup environment already contains a valid session and generation. Do not
export `AHDCODE_DEV_SESSION` and `AHDCODE_DEV_GENERATION` into a production
shell, service, or container. The binary will inject the development script
for loopback clients. Application configuration cannot do this after the
process has started; an environment that is present at startup can.

For a program that `bring Web`, the first successful start prints a
`Browser refresh:` section in the Web banner and does not print
`✓ Browser will refresh` after the first `✓ Running`. A later successful
rebuild prints:

```
→ Restarting...
✓ Running
✓ Browser will refresh
```

For a loopback client, and only when the application did not register any
method on the path, the process answers `GET /__ahdcode/dev/reload` with `session
generation` and `GET /__ahdcode/dev/reload.js` with a same-origin script.
Eligible HTML gains `<script src="/__ahdcode/dev/reload.js" ...>` rather than
an inline script, so `script-src 'self'` can run it. The tag carries the
session and generation that wrote the page. One poll runs at a time. A
different pair reloads the page once. A failed rebuild keeps the same
process, so the pair does not change and the last page stays up. Stopping
`ahdcode dev` and starting it again creates a new session, so a tab left
open reloads instead of treating generation `1` as the page it already has.

The script is inserted before the last `</body>`, matched without changing
UTF-8 offsets, or appended when the response is an HTML fragment. It is not
added to a 1xx or 3xx response, a `Content-Encoding` other than `identity`,
an attachment, or a charset other than UTF-8, US-ASCII, ISO-8859-1,
ISO-8859-15, Windows-1252, or Windows-1254. A missing charset is treated as
compatible. The final `Content-Type`, after `withHeader`, decides. A page
that merely mentions `/__ahdcode/dev/reload` is still eligible. `HTTP.file`
and `server.static` are not rewritten. A non-loopback client receives the
ordinary HTML unchanged: no script, no session identifier, and no
development cache header. It also does not receive the probe or the script
file. If the application registered any method on one of those paths, that
registration is left in place for every client and the development handler
does not answer the path.

The refresh is a full page load, not hot module replacement, and it does
not preserve form input or scroll position. Static files are outside the
watch set, so editing one does not rebuild and does not refresh the browser.

If the new process cannot bind its port, it exits with the HTTP server's
own bind error and `dev` waits for the next save. Nothing is left listening
from the process that just failed. Ctrl+C and `ahdcode stop` both stop the
child and the watcher. A second `dev` on the same entry is refused while
the first session's descriptor is live.

Saves are debounced (~150-300ms) so a burst of writes from an editor
produces one rebuild, not several, and only one build ever runs at a time.

Like `run`, a live `dev` session keeps a small descriptor beside the entry
module — `app.ahd` produces `app.dev` — over its own authenticated loopback
control channel, published as soon as the session starts (even before the
first build finishes), so it is always stoppable and a second `dev` against
the same source is always detected rather than silently racing the first.
Press Ctrl+C, or run `ahdcode stop app.dev` from elsewhere, to end it
cleanly.

#### Dev watch scope

`dev` watches the entry file plus the compiler's resolved
[`require(...)`](REQUIRE.md) graph plus any `require(...)` target the
latest build attempt named but could not find yet — never a recursive
project-wide scan. The watch set is recomputed after every build attempt,
success or failure, so:

- editing any required file (however deeply nested) rebuilds and restarts,
  the same as editing the entry;
- creating a required file that was previously missing rebuilds
  automatically, with no further edit to the file that requires it needed;
- a file dropped from the `require(...)` graph (its `require(...)` line
  removed) stops being watched.

Static assets served through [`server.static`](HTTP.md#static-files) are
never part of this graph: editing one never triggers a rebuild, since
static files are read straight from disk on every request. See
[`require(...)`](REQUIRE.md) for the composition rules this graph follows.

A program started without `ahdcode dev` does not inject the reload script.

Bundled first-party modules are never watched either. `bring Web` compiles
from source embedded in the compiler, so there is no file on disk to change.

#### Dev and Web applications

When the compiled module graph contains the first-party [`Web`](WEB.md)
framework, `dev` adds a banner naming the application, the socket it bound,
and the local name this machine now routes to it:

```
AhdCode Web
  Ahd Akademi (development)

  Open:
  http://127.0.0.1:8080

  Browser refresh:
  Open pages reload after a successful rebuild.
  A failed build keeps the last page.

  Local identity:
  http://ahdakademi.test/
  Bind: 127.0.0.1:8080
```

The address under `Open:` is built from `SERVER_HOST` and `SERVER_PORT` — the
socket the application actually binds — and always works. It follows the
configured host rather than assuming loopback; a wildcard bind (`0.0.0.0`) is
displayed as the loopback address it is genuinely reachable on.

The line under `Local identity:` is the `.test` name derived from `APP_HOST`,
served by the [local router](#local-development-test-names-and-the-router).
The two are reported separately on purpose: the bind address says where the
socket is, the local URL says what name this machine routes to it, and
merging them into one "URL" would quietly become wrong the moment the port
changed. `APP_ENV=test` uses `APP_HOST` unchanged, so it gets no identity
line at all.

#### How the local name is derived

The registrable suffix is replaced, not appended to:

| `APP_HOST`               | local name                 |
| ------------------------ | -------------------------- |
| `ahdakademi.com`         | `ahdakademi.test`          |
| `ahdakademi.com.tr`      | `ahdakademi.test`          |
| `example.co.uk`          | `example.test`             |
| `www.example.com`        | `www.example.test`         |
| `admin.ahdakademi.com.tr`| `admin.ahdakademi.test`    |
| `localhost`              | `localhost.test`           |
| `ahdakademi.test`        | `ahdakademi.test`          |

A multi-label suffix is dropped whole: `ahdakademi.com.tr` is the same project
as `ahdakademi.com`, so both develop at `ahdakademi.test`. The second label is
only dropped beneath a two-letter country code and only when it is a registry
label (`com`, `co`, `org`, `edu`, `gov`, and similar), so `www.example.com`
keeps its `www` and `admin.checkmate.tr` keeps its `admin`.

If a **live** AhdCode session already owns that name, the next free suffix is
used — `ahdakademi1.test`, then `ahdakademi2.test`, and so on, always the
smallest free index. Ownership is decided by AhdCode's own route registry,
never by the hosts file: a leftover `127.0.0.1 ahdakademi.test` line from a
project that is no longer running is a harmless stale mapping and does not
push a new session onto a suffixed name.

A route is claimed only once the application is actually listening, and it is
released when the session stops — through Ctrl+C, `ahdcode stop`, or
`ahdcode kill` alike. A session that crashes leaves an entry behind; the next
`ahdcode dev` reclaims it after finding its owner unreachable, so no name is
held forever by a process that died.

If routing cannot be arranged at all, the session still runs and the banner
says so in one line. A convenience failing is never allowed to fail an
application that started correctly.

The launcher reads the process environment before the app-root `.env`.
It refuses `APP_ENV=production` for Web development and explicit HTTPS
sessions. Plain Web development refuses `APP_PROTOCOL=https`. Use
[`dev --https`](#local-https) for loopback TLS. That mode sets
`APP_PROTOCOL=https` for the child only. A refusal is a nonzero exit.

### `stop`: graceful shutdown

```bash
ahdcode stop app.dev
ahdcode stop app.run
```

`stop` is the graceful counterpart to `kill`: it asks the owning session (a
`dev` controller or a plain `run` supervisor) to shut down cleanly over the
same authenticated control channel `kill` uses, and — unlike `kill` — waits
to confirm the process has actually exited before reporting success. If
graceful shutdown does not finish within a few seconds, `stop` reports that
plainly rather than silently escalating to a forced kill; use
`ahdcode kill` for that. Given a bare source name (`app.ahd` rather than
`app.dev`/`app.run`) it resolves against whichever descriptor is live;
if both a `dev` and a `run` session are active for the same name, it
refuses to guess and asks for the explicit file.

`ahdcode kill app.dev` forcibly stops both the dev controller and whatever
child it currently owns, with no orphan left behind; `ahdcode kill app.run`
is unchanged from the description above.

Diagnostics include a stable code, source location, excerpt, and hint when
available. Compiler invocations use argument arrays rather than shell command
strings.

Running `ahdcode` without a command starts the REPL.

`lsp` starts the language server described in the [Language server
guide](LSP.md): stdio-only JSON-RPC and the v0.2.2 practical everyday feature
set (diagnostics, hover, completion with auto import, go to definition,
document symbols, signature help, find references, rename, semantic tokens,
inlay hints, code actions, formatting, workspace symbols, folding ranges, and
selection ranges), all compiler-backed. v0.4.0 modules such as `HTTP` and `HTML`
appear through that same compiler module interface; v0.5.0 `cookie`,
`sessions`, `Cookie`, `Session`, and `SessionStore`, and v0.6.0 `client`,
`clientRequest`, `Client`, `ClientRequest`, and `ClientResponse` use that path
too. There is
no HTTP-specific, Cookie-specific, or Session-specific LSP catalog. v0.3.0's `SQLite` uses the same path. It accepts no arguments other than an
optional `--stdio`
(accepted and ignored -- real LSP client libraries append it automatically
when they launch a server over stdio transport; `ahdcode lsp` never supports
any other transport, so the flag is a no-op) and never writes anything but
protocol frames to stdout.

## Local development: `.test` names and the router

`ahdcode dev` and `ahdcode databases` each host a small first-party reverse
router while they run. It is built from the Go standard library — there is no
Caddy, no nginx, no external service, and nothing installed on the machine —
and it disappears with the session that hosted it.

What it does is narrow by design:

- it listens on **loopback only**, never `0.0.0.0` and never an external
  interface, so nothing off this machine can reach it;
- it forwards only to destinations recorded in AhdCode's route registry, and
  the registry only ever accepts a loopback destination — checked when the
  entry is written and again when it is read;
- a `Host` header that is not in that allowlist is refused with `404`. A
  request has no way to name its own destination, so this is not an open
  proxy and cannot be turned into one;
- method, path, query, headers, and body are forwarded unchanged, and the
  application sees the name that was typed rather than the loopback port.

This router serves plaintext HTTP. Direct local HTTPS uses the application's
own loopback listener.

### Which port

Port 80 gives the clean URL and is attempted first. On most Unix machines an
unprivileged process cannot bind it, and that is platform policy rather than
something to work around: AhdCode does not elevate, does not retry
indefinitely, and does not wait for anyone. It takes the deterministic
fallback port `7357` immediately and prints the URL that actually works:

```text
http://ahdakademi.test:7357/
```

`AHDCODE_LOCAL_ROUTER_PORT` overrides that fallback when `7357` is already
spoken for on your machine.

Only one process can hold the port, but every AhdCode session serves every
registered route, so which one holds it does not matter. A session that
starts second keeps trying quietly in the background and takes over the
moment the first stops.

### `ahdcode local status`

Reports what is running and where it points, and changes nothing:

```text
AhdCode Local

Router: running
Bind: 127.0.0.1:7357
  Port 80 was not available, so local URLs carry :7357.

System hosts: /etc/hosts
  managed block: absent
  ahdakademi.test: mapped to 127.0.0.1
  ahddatabasestudio.test: not mapped
  (`ahdcode local hosts` shows how to add the missing ones)

Routes:
  ahdakademi.test
    url: http://ahdakademi.test:7357/
    -> 127.0.0.1:18437
    source: /home/ada/projects/ahd/app.ahd

Route registry: ~/.config/ahdcode/routes.json
Database registry: ~/.config/ahdcode/databases.json
```

A route whose owner is no longer running is listed separately as stale. Only
a session that answers its own authenticated control channel counts as live —
a recorded process id is never trusted, because operating systems reuse them.

### `ahdcode local hosts`

A `.test` name still has to resolve. AhdCode manages exactly one delimited
block in the system hosts file:

```text
# BEGIN AHDCODE LOCAL
127.0.0.1 ahdakademi.test
127.0.0.1 ahddatabasestudio.test
# END AHDCODE LOCAL
```

After you approve local host integration once, `ahdcode dev` maintains the
current hostname in that block automatically. The first TTY session asks:

```text
Enable local .test names? [Y/n]
```

Consent happens before any privilege prompt. A later project such as
`checkmate.test` is added through the same authorized mechanism; you do not
run a separate command for every hostname. `ahdcode local hosts apply` and
`remove` remain as manual administration and recovery.

`ahdcode local hosts` prints the block and whether it is in place.
`ahdcode local hosts apply` writes it; `ahdcode local hosts remove` takes it
back out. In all three cases:

- **everything outside the two markers is left byte for byte as it was.** The
  file is never regenerated and never reordered, and an entry you or another
  tool put there survives, including through removal of AhdCode's own block;
- only `127.0.0.1` mappings are ever written, and only for names AhdCode
  actually routes;
- when the file is already writable, the change is applied directly and no
  elevation is involved at all;
- otherwise the exact change is shown and **an explicit yes at an
  interactive prompt** is required before administrator access is requested.
  `sudo` is never invoked silently and never invoked without that answer;
- a **non-interactive** session never prompts and never elevates. It prints
  what would be needed and exits, rather than blocking on a password nobody
  is there to type.

`.test` is used rather than `.local` deliberately: `.test` is reserved by
RFC 6761 and will never be delegated, while `.local` is claimed by
mDNS/Bonjour on macOS and most Linux desktops.

Names accumulate in the block rather than being pruned each run: a loopback
mapping with nothing behind it is harmless, and removing it would mean asking
for administrator access again the next time the same project runs.

If you decline, or the platform does not support it, nothing breaks — the
application stays reachable at its own loopback address, which the banner
always prints.

## Add PWA files

`ahdcode pwa init [--name NAME] [--short-name NAME]` creates an optional
manifest, worker, public icons, and an AhdCode integration module in
`PWA/`. Use it at the application root. The command preserves existing
project files and refuses to replace an existing `PWA/` path.
[PWA setup and cache policy](PWA.md).

## Local HTTPS

Use `ahdcode dev --https app.ahd` for direct loopback TLS. Add
`--cert FILE --key FILE` before the entry to use an existing PEM pair.
HTTP remains the default. For certificate creation, browser trust,
renewal, port conflicts, and Secure cookies, see [Local HTTPS in Web](WEB.md#14-local-https).

## `ahdcode databases`

Launches the AhdDataStudio that belongs to this installed AhdCode version.
The ordinary path does not need the AhdCode repository, a parent-directory
walk, or `AHDCODE_ROOT`. The CLI materializes the exact-version Studio
bundled in the toolchain into the per-user cache, then starts it.

`AHDCODE_ROOT` remains a developer override: when it is set, that checkout's
`tools/AhdDataStudio` is used instead of the bundled copy. If the override
is set and Studio is missing there, the command fails rather than searching
elsewhere. Ordinary installed use leaves `AHDCODE_ROOT` unset.

`ahdcode databases list`, `add`, and `remove` never launch or materialize
Studio. They only read and write the per-user SQLite registry.

On launch, a missing `.env` is copied from `.env.example`. The new file has
mode `0600` on Unix and a private ACL on Windows for the current user,
SYSTEM, and Administrators. An existing `.env` keeps its contents and
permissions. The server binds only to `127.0.0.1:8081`.

Its canonical URL is:

```text
http://ahddatabasestudio.test/
```

served through the local router described above. The direct address remains
fully supported and is what the CLI opens whenever the clean name is not
resolvable on this machine:

```text
http://127.0.0.1:8081/AhdDataStudio
```

Studio also answers `GET /` with a redirect to `/AhdDataStudio`, so both the
clean name and the bare loopback root land somewhere useful.

The CLI reads the local hosts file without DNS lookups. Unless it finds an
unambiguous IPv4 mapping for `ahddatabasestudio.test`, it opens the direct
loopback URL. If local host integration is already authorized, the Studio
name is maintained automatically. First use on a TTY asks before any
privilege prompt. A refusal or a non-TTY session keeps Studio on the
direct bind address.

### `ahdcode databases list | add | remove`

Bare `ahdcode databases` still means "open AhdDataStudio". Beneath it are
three small commands over the per-user **database registry**, so a database
can become visible in Studio without editing any environment variable:

```bash
ahdcode databases add ./database/app.db
ahdcode databases list
ahdcode databases remove ./database/app.db
```

- **SQLite only** in this release. MySQL configuration stays exactly where it is —
  explicit AhdDataStudio settings — because a MySQL source is inseparable
  from credentials, and credentials have no place in a registry.
- `add` requires the file to exist; the registry describes databases, it does
  not create them. The path is canonicalized (absolute, cleaned, symlinks
  resolved), so the same file named two different ways stays one entry.
- `remove` forgets **exactly** that registry entry. **The SQLite file itself
  is never opened, moved, or deleted.**
- Nothing is ever scanned. An entry exists because a starter created that
  file or because you named it here.
- A registered file that is not present right now is listed as `unavailable`
  rather than dropped: a database on an unmounted volume has not been
  withdrawn, and only you should be able to remove it.

`list` prints one tab-separated row per entry (`driver`, path, state), which
is enough to grep or cut in a script.

### Automatic registration from `init web admin`

When [`ahdcode init web admin`](WEB.md) creates a SQLite database, it
registers that one file automatically. Nothing has to be added to
`AHD_DATA_SQLITE_PATHS` or `AHD_DATA_PROJECT_ROOT` first:

```bash
ahdcode init web admin
ahdcode databases      # the new database is already listed
```

Only the database the command itself created is registered — the project is
never scanned for others. Registration happens after the database is safely
in place, and if the registry cannot be written the failure is reported and
nothing is undone: the database and the generated application are both real
and correct, and the message says how to register it by hand.

The Studio `.env` list is still updated as well when one is discoverable, so
a v0.18 setup that already depends on `AHD_DATA_SQLITE_PATHS` keeps working
unchanged. `init web` does not create the Studio `.env`.

### Where the registries live

Both registries are per-user files under the OS configuration directory
(`~/.config/ahdcode/` on Linux, `~/Library/Application Support/ahdcode/` on
macOS). On Unix the directory is created with `0700` and files with `0600`.
On Windows files inherit their directory's ACL. Use a private directory
when setting `AHDCODE_LOCAL_HOME`. They hold local development
metadata only — hostnames, loopback ports, file paths — and never a password,
a token, or any database content. See [Env](ENV.md).

## `ahdcode migrate`

Applies versioned SQL files to one local SQLite database. This is a CLI
workflow. The `SQLite` and `MySQL` language modules have no migration methods,
and `ahdcode databases` still only lists, adds, and removes Studio registry
entries.

```bash
ahdcode migrate new create_notes
ahdcode migrate status --db notes.db
ahdcode migrate up --db notes.db
ahdcode migrate down --db notes.db --steps 1
```

The default directory is `migrations`, next to the working directory. `--dir`
accepts a relative or absolute path. `--driver` defaults to `sqlite`.
`mysql` and any other driver stop with a diagnostic and open no connection.
Usage mistakes exit 2. A migration that fails exits 1.

A name is one letter followed by letters, digits, or underscores, at most 64
characters. `new` writes the next contiguous version as two files:

```text
migrations/0001_create_notes.up.sql
migrations/0001_create_notes.down.sql
```

Only those names are accepted: `0001_create_notes.up.sql` and
`0001_create_notes.down.sql`. A version is the integer prefix, so `0002` and
`2` are the same version. Versions must be `1..N` with no gaps and no
duplicates. Other `*.sql` names, empty files, and files that are not valid
UTF-8 are errors. A directory entry that is a symbolic link is rejected, so a
link or junction cannot pull a script from outside the migration directory.
The same rejection applies to a migration root or parent that is a symbolic
link or an NTFS junction. `filepath` cleaning does not follow those links.
The final file is opened without following a reparse point, and the bytes
that are executed are the bytes that were checksummed. The directory lock
file and a recovery rename also refuse a symbolic link, junction, reparse
point, or other non-regular file. They do not follow a substituted lock file
and do not rename an incoming file onto another file. A parent directory
replaced by a link in the gap after that check is the remaining race; migrate
fails closed when it can see the link. The database path cannot be empty,
`:memory:`, a directory, or a symbolic link or junction. `up` creates a
missing database file and refuses a missing parent directory. `status` does
not create a missing database: it exits 0 and lists the pending files.
`down` does not create a missing database either.

`up` applies every pending file in version order. If a later file fails, the
versions already committed are printed before the error and are not undone.
`status` prints one `applied` or `pending` row per version. Running `up`
again when nothing is pending prints `No pending migrations.` and changes
nothing. A flag that does not belong to the subcommand, or the same flag
given twice, exits 2. `--help` and `-h` print this usage and exit 0.

Each file is executed as one SQLite script. AhdCode does not split the file
on semicolons, so triggers, quoted strings, and semicolons inside SQL stay
intact. A file of only whitespace or comments has no executable statement and
is not recorded. A comment followed by SQL is executable. The whole script
and the history row share one `BEGIN IMMEDIATE` transaction. While the script
runs, SQLite's authorizer rejects `BEGIN`, `COMMIT`, `ROLLBACK`, `END`,
`ATTACH`, `DETACH`, `VACUUM`, and writes to `ahd_schema_migrations`.
`SAVEPOINT`, `ROLLBACK TO`, and `RELEASE` stay allowed because they do not
end the outer transaction. A rejected or failed script rolls that transaction
back, so it is not recorded and it does not leave a committed partial schema.
Ordinary DDL such as `CREATE TABLE` writes SQLite's own schema table
internally; that internal write stays allowed. Direct edits of
`sqlite_master` still require `PRAGMA writable_schema`, which is rejected.

Creating a trigger does not authorize its body. Before the migration can
commit, migrate asks this build of SQLite to compile an `INSERT`, an
`UPDATE`, and a `DELETE` for every table or view that has a trigger in the
main or temporary schema. The `UPDATE` assigns each ordinary column to
itself. On a table that has a rowid it also assigns `rowid`, `oid`, and
`_rowid_`, because `UPDATE OF` those names is not compiled by assigning a
normal column. Generated columns are not assigned. Those statements are
prepared, not executed.

Coverage is one internal detail of this SQLite build's bytecode, not a stable
public SQLite API. A trigger counts as compiled only when `EXPLAIN` has the
usual eight columns and an `Init` instruction at address 0 whose `p4` is
exactly `-- TRIGGER`, a space, and that trigger's name. The opcode, address,
`p1`, `p3`, `p5`, and empty comment have to match that shape as well. Text in
another opcode, including a `DEFAULT`, a `CHECK`, or a string inside a
different trigger, is not coverage. If the bytecode layout is not the one
this command was tested against, the migration is rejected. A body that
writes `ahd_schema_migrations` is rejected. A trigger that does not appear as
that marker, including `UPDATE OF` a missing column or a generated column
that cannot be assigned, is rejected. A prepare error is not proof that the
body is safe.

The marker does not name the schema. Two triggers whose names match under
SQLite's ASCII case folding, including the same name in `main` and `temp` or
names that differ only by ASCII case, are rejected because one compiled body
cannot prove the other. `BEFORE`, `AFTER`, and `INSTEAD OF` triggers are
checked this way, including `UPDATE OF` a real column, views, `WITHOUT ROWID`
tables, multi-statement bodies, quoted identifiers, nested triggers, and a
trigger added by a later migration. An accepted trigger is an ordinary
application trigger. This check does not claim to recognize every future
SQLite bytecode shape.

The authorizer also rejects connection pragmas that would change locking,
authorization, or integrity for a later migration on the same connection:
`busy_timeout`, `locking_mode`, `trusted_schema`, `recursive_triggers`,
`foreign_keys`, `defer_foreign_keys`, `ignore_check_constraints`,
`query_only`, `read_uncommitted`, `cell_size_check`, `legacy_alter_table`,
`synchronous`, `writable_schema`, `journal_mode`, `wal_checkpoint`,
`wal_autocheckpoint`, `case_sensitive_like`, `secure_delete`,
`reverse_unordered_selects`, `temp_store`, `incremental_vacuum`, and
`schema_version`. Before each migration takes its lock, the command sets the
busy timeout back to five seconds and restores those connection settings to
the defaults it relies on. `auto_vacuum` stays allowed: it is database
storage configuration, not a connection flag this command resets. Schema
queries such as `table_info` stay allowed. Journal mode is not changed.

History lives in `ahd_schema_migrations`:

| column | meaning |
| --- | --- |
| `version` | integer primary key |
| `name` | the name after the version prefix |
| `up_checksum` | SHA-256 hex of the exact `.up.sql` bytes |
| `down_checksum` | SHA-256 hex of the `.down.sql` bytes, or empty when no down file existed at apply time |
| `applied_at` | Unix seconds |

`new` takes an operating-system lock on the migration directory and scans
again after it has the lock, so two creators cannot choose the same version.
The lock is released when the process exits, including after a crash. A
creator that is killed after publishing the down file is completed by the
next `new`; unpublished temporary files are discarded. An existing migration
that simply has no down file is not deleted.

The history name follows SQLite's case-insensitive ASCII identifier rules.
`AHD_SCHEMA_MIGRATIONS` is the same object as `ahd_schema_migrations`. A view,
or more than one object with that name, is an error; migrate does not create
a replacement table. The table is created inside the same transaction as the
first successful migration. Its columns, declared types, `NOT NULL`
constraints, primary key, and indexes must match exactly. Migrate reads
versions, checksums, and
timestamps with SQLite's value types and does not coerce text into integers
or NULL into an empty string. A wrong shape or a bad value stops the command.
The table is never altered, dropped, or rewritten to repair it. Applied
versions must be exactly `1..N`. A database version with no matching file, a
renamed file, or a checksum that no longer matches the bytes on disk stops
the command. Checksums are the raw file bytes, so changing CRLF to LF is a
modification. Metadata queries use bound parameters.

Two `ahdcode migrate up` processes take `BEGIN IMMEDIATE` on the same file.
The waiter waits up to five seconds. After the lock, it reads history again
and skips a version the other process already recorded. The same version is
not applied twice. A process killed inside the transaction leaves no history
row; the next `up` applies that version.

`down` is never run because `up` failed. It rolls back only the latest
applied version, or `--steps` of them, and only when the down script recorded
at apply time is still byte-for-byte the same file. Each step is its own
transaction: the down script runs, then that history row is deleted. A later
step that fails leaves the earlier rollbacks committed. Each committed
rollback is printed, then the error and the number of rollbacks that
committed, and the command exits 1. The generated down file contains a
statement that fails until you
replace it, so an unedited down cannot remove history. A migration applied
without a down file cannot be rolled back later, even if a down file appears
afterwards.

`ahdcode migrate` does not change journal mode. It rejects the connection
pragmas listed above, including `foreign_keys` and `synchronous`, and it
restores them before the next migration on that connection. It does not
delete application tables except by running the SQL you wrote. Persistent HTTP session tables (`ahd_session_meta`,
`ahd_sessions`, `ahd_session_revoked`) are not migration history. A migration
you write can change them only if its SQL says so.

Ordinary SQLite migrations need no administrator rights and no network.
The Windows `backupTo` URI path is not used.

See `examples/v2.10/migrations`.

## Check source without building

Use the checker's exit status and JSON report as the source validation step
when upgrading a project. See the [v2.12 upgrade checklist](UPGRADE_2.12.md).

```bash
ahdcode check app.ahd
ahdcode check "path with spaces/app.ahd" --format json
```

`check` reads the saved entry and its dependencies, resolves modules, parses
source, and runs the same semantic analysis used by `build` and the
[language server](LSP.md). The language server can also analyze unsaved
editor text. `format --check` checks canonical formatting. A successful
`check` establishes frontend validity. Go generation, linking, and runtime
behavior are checked by later stages. This command does not need Go on
`PATH`, run application code or helpers, or write build artifacts.

The default `--format text` prints diagnostics to stdout as
`path:line:column: severity CODE: message`, followed by any compiler hint.
Line and column start at 1. Columns count UTF-16 code units. Success without
diagnostics produces no text.

`--format json` writes one UTF-8 JSON document to stdout for both valid and
invalid sources. Version 1 has an integer `schemaVersion` and a
`diagnostics` array, which can be empty. Every diagnostic has `severity`
(`error` or `warning`), `code` (the compiler code), `message`, `path`,
and `range`. `hint` appears when the compiler supplies one. Ranges use
zero-based `line` and `character` values, count UTF-16 code units, and exclude
the end position. An entry-input error with no source span uses `"range": null`.

Paths use forward slashes and are relative to the resolved entry directory.
A source on a different volume is identified by its filename. Bundled
sources use `<ahdcode>/...`. The report omits source excerpts. Diagnostics
are ordered by path, range (absent ranges first), severity, code, message,
and hint.

For `x: Int := "bad"` in `app.ahd`:

```json
{
  "schemaVersion": 1,
  "diagnostics": [
    {
      "severity": "error",
      "code": "SEM004",
      "message": "initializer of x expects Int; received String",
      "path": "app.ahd",
      "range": {"start": {"line": 0, "character": 10}, "end": {"line": 0, "character": 15}},
      "hint": "use a value assignable to the declared type"
    }
  ]
}
```

Exit status is `0` when there are no errors (including warnings-only results),
`1` for source or input errors, and `2` for invalid CLI usage. Usage
messages and output-write failures go to stderr. Usage errors emit no JSON.
