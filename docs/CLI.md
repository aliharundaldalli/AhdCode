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
stops the application immediately. The run file is the application's identity:
a bare pid is deliberately not accepted, and a file that is not a well-formed
AhdCode run descriptor is refused without signalling anything. A descriptor
whose process is already gone is reported as stale and removed rather than
acted on, and starting a second `run` while a live descriptor exists fails
with the pid and the `kill` command to use instead of silently colliding on
the port.

The descriptor is internal CLI metadata, not a language-level format: nothing
in the standard library reads or writes it, and it is not a process manager,
daemon registry, or `dev`/watch mode.

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
