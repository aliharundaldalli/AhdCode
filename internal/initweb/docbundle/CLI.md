# CLI

[English] · Türkçe

[Back to README](README.md) · [Formatter](FORMATTER.md) · [REPL](REPL.md) · [Language server](LSP.md)

The current command surface is:

```text
ahdcode
ahdcode init web [empty|basic|admin|mvc|crud]
ahdcode databases
ahdcode databases list
ahdcode databases add <file.db>
ahdcode databases remove <file.db>
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode dev <entry.ahd>
ahdcode stop <app.dev|app.run>
ahdcode kill [--force] <app.dev|app.run>
ahdcode local status
ahdcode local hosts [apply|remove]
ahdcode format [--check] <file.ahd>
ahdcode lsp
ahdcode --help
ahdcode --version
```

`ahdcode init web` writes a Web starter into the **current directory**. On a
TTY it asks Empty, Basic, Admin, MVC, or CRUD. Non-interactive use must pass
`empty`, `basic`, `admin`, `mvc`, or `crud`. Templates, Bootstrap 5.3.3, and
the AhdCode logo are embedded: offline, no package manager, no overwrite.
Every starter also receives `AHDCODE.md` and an English `Documents/AhdCode/`
snapshot for this version. Those files are not runtime. Next:
`ahdcode dev app.ahd`.

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

`dev` reads `APP_*` with the application's own precedence — process
environment first, then the app-root `.env` — and only ever to decide what to
print. It never exports a variable and never passes one to the child.

It refuses two configurations, before starting anything:

- `APP_ENV=production`. Running a production contract through the development
  command would mean either treating it as development or rewriting
  `APP_ENV`.
- `APP_PROTOCOL=https`. `dev` serves plaintext HTTP, so starting the child
  would mean serving `http` while the configuration says `https`. v0.19
  routes `.test` names over plaintext HTTP and still ships no local
  certificate authority and no certificate management, and `dev` neither
  downgrades the protocol nor generates an untrusted certificate — see
  [Web](WEB.md#14-local-https--current-limitation).

In both cases `dev` reports the mismatch, starts no child, opens no listener,
leaves no `.dev` descriptor, changes neither variable, and exits non-zero.

A program that never wrote `bring Web` is unaffected by all of this, even if
`APP_ENV` happens to be set in its environment.

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

Local development is **plaintext HTTP** in this release. There is no local TLS, no
certificate authority, and no ACME.

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

On launch, a missing `.env` is copied from `.env.example` with mode `0600`;
an existing `.env` is preserved. The server binds only to `127.0.0.1:8081`.

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
macOS), created `0700`, with each file written atomically at `0600`.
`AHDCODE_LOCAL_HOME` overrides the location. They hold local development
metadata only — hostnames, loopback ports, file paths — and never a password,
a token, or any database content. See [Env](ENV.md).
