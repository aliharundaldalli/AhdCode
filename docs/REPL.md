# REPL

[English] · [Türkçe](REPL_TR.md)

[Back to README](../README.md) · [CLI](CLI.md) · [File and Path](FILESYSTEM.md)

Start a persistent interactive session:

```bash
ahdcode
```

Startup prints a version banner matching `ahdcode --version` (currently
`AhdCode v0.5.0`), then the `ahd>` prompt:

```text
ahd> x := 5
ahd> x = x + 1
ahd> x
6
ahd> age := int(take("Age: "))
Age: 26
ahd> age + 1
27
ahd> square := lambda (x: Int) -> x^2
ahd> square(5)
25
```

The normal lexer, parser, semantic checker, and typed/lowered IR are shared
with file compilation. The REPL executes newly validated IR in one persistent
evaluator. It does not rerun successful source history, so output, input,
mutation, Class construction, module initialization, and file operations occur
exactly once.

Values and mutable bindings persist. List/Pair/Class objects retain identity,
so aliases observe later mutation. Named Functions, Classes, imports, and the
shared Math RNG state also remain in the session:

```text
ahd> a := [1, 2]
ahd> b := a
ahd> a.add(3)
ahd> b
[1, 2, 3]
ahd> bring Math
ahd> Math.seed(42)
ahd> Math.random()
0.7415648787718233
ahd> Math.random()
...
```

Lambda Function values also persist between commands and work directly as
callbacks, for example `values.map(lambda (x: Int) -> x^2)`. As elsewhere,
they are expression-only and require explicit `#` or `@` annotations to capture outer variables.

`take` prints and flushes its prompt, then consumes exactly one answer line
from the real terminal. That answer is not treated as another REPL command.

Local modules resolve from the directory where `ahdcode` was launched. The
same directory is the base for relative File paths:

```text
ahd> bring Engine
ahd> Engine.tick()
ahd> bring File
ahd> File.writeText("note.txt", "hello")
ahd> File.readText("note.txt")
hello
```

Multiline Functions, Classes, blocks, and expressions use the continuation
prompt `...>`. Ordinary declaration rules remain: redeclaring `x` in the same
scope is an error; mutation uses `=`. A failed semantic submission or an
uncaught AhdCode Error does not terminate the REPL or erase the preceding
successfully committed source context.

## SQLite in the REPL

`SQLite` works in the persistent session. An in-memory `Database` survives
successive successful entries; a failing SQL command reports `SQLiteError`
and leaves the session intact:

```text
ahd> bring SQLite
ahd> db := SQLite.open(":memory:")
ahd> db.execute("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
0
```

Relative file paths are resolved against the directory the REPL was started
in. See [SQLite](SQLITE.md).

## Latex and PDF in the REPL

`Latex`'s markup-building helpers (`document`, `table`, `section`, `escape`,
and the rest) and `PDF`'s document-building operations (`PDF.new()`,
`heading`, `paragraph`, `table`, `image`, `pageBreak`, `PDF.fromWord`,
`PDF.fromExcel`) all work normally in the REPL -- they only build values in
memory. Actually compiling a PDF invokes the offline Tectonic renderer as an
external process, which the persistent evaluator does not support:
`Latex.pdf(...)`, `Latex.pdfFile(...)`, and `PDFDocument.save(...)` all raise
an error (`LatexError`/`PDFError`) when called interactively. Run them from a
`.ahd` file with `ahdcode run` or `ahdcode build` instead. `Archive` has no
such limitation -- `Archive.zip`/`tar`/`tarGzip` work fully in the REPL,
since archiving uses only the Go standard library.
