# CLI

[English] · [Türkçe](CLI_TR.md)

[Back to README](../README.md) · [Formatter](FORMATTER.md) · [REPL](REPL.md) · [Language server](LSP.md)

The current command surface is:

```text
ahdcode
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode dev <entry.ahd>
ahdcode stop <app.dev|app.run>
ahdcode kill [--force] <app.dev|app.run>
ahdcode format [--check] <file.ahd>
ahdcode lsp
ahdcode --help
ahdcode --version
```

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
framework, `dev` adds a banner naming the application and its canonical
development URL:

```
AhdCode Web
  Ahd Akademi (development)

  http://ahdakademi.com.test
```

The URL is `APP_PROTOCOL` and `APP_HOST` with `.test` appended, which is the
local identity of the application's public host. `dev` reads `APP_*` with the
application's own precedence — process environment first, then the app-root
`.env` — and only ever to decide what to print. It never exports a variable
and never passes one to the child.

It refuses one configuration: `APP_ENV=production`. Running a production
contract through the development command would mean either treating it as
development or rewriting `APP_ENV`, so `dev` reports the mismatch, starts
nothing, and exits non-zero.

An `https` development URL is explained rather than downgraded. v0.15 ships no
local certificate authority, `.test` resolver, or development gateway, so
`dev` says what is missing and leaves `APP_PROTOCOL` alone — see
[Web](WEB.md#14-local-https--current-limitation).

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
