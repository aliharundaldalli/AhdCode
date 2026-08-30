# CLI

[English] · [Türkçe](CLI_TR.md)

[Back to README](../README.md) · [Formatter](FORMATTER.md) · [REPL](REPL.md)

The current command surface is:

```text
ahdcode
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode format [--check] <file.ahd>
ahdcode --help
ahdcode --version
```

`run` compiles through the normal frontend and Go backend, then executes the
native result. Arguments after the entry (optionally after `--`) are forwarded
to the generated process, although v0.1 publishes no language-level argument
API yet.

`build` prints the produced executable path. Without `-o`, the compiler uses
the entry module's base name in the current working directory.

Diagnostics include a stable code, source location, excerpt, and hint when
available. Compiler invocations use argument arrays rather than shell command
strings.

Running `ahdcode` without a command starts the REPL.
