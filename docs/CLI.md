# CLI

[English] · [Türkçe](CLI_TR.md)

[Back to README](../README.md) · [Formatter](FORMATTER.md) · [REPL](REPL.md) · [Language server](LSP.md)

The current command surface is:

```text
ahdcode
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode kill [--force] <app.run>
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
in the standard library reads or writes it, it is written `0600` because it
carries a control capability, and it is not a process manager, daemon
registry, or `dev`/watch mode.

`build` prints the produced executable path. Without `-o`, the compiler uses
the entry module's base name in the current working directory.

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
