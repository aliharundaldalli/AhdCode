<p align="center">
  <img src="editors/vscode/images/ahdcode-logo.png" alt="AhdCode logo" width="360">
</p>

# AhdCode

[![CI](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml/badge.svg)](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml)

[English](README.md) · [Türkçe](README_TR.md)

AhdCode is an experimental statically checked general-purpose programming
language focused on readable syntax, explicit intent, predictable semantics,
and native compilation.

The current release is **v0.1.17**. The core language works end to end, but
the project is not production-ready and breaking changes may occur before 1.0.

v0.1.17 — Structured Data & Configuration — adds typed [JSON](docs/JSON.md)
and [XML](docs/XML.md) modules and a compact [Env](docs/ENV.md) module for
process/`.env` configuration, and fixes a confirmed Word merged-table
silent data-loss bug on read/save round-trips.

```ahd
greet: Function := (
    name: String
) -> String {
    return "Hello {name}"
}

names: List<String> := ["Ali", "Ayşe"]

for name in names {
    write(greet(name))
}
```

## Why AhdCode?

- Declaration and mutation look different: `:=` declares, `=` mutates.
- Static checking rejects unrelated implicit conversions and truthiness.
- Explicit nullable types (`T?`) compose with collections, while flow-sensitive
  checks narrow proven non-null values.
- Lists, Pairs, Classes, Functions, modules, errors, and native executables are
  part of the v0.1 core.
- Expression-only `lambda (<typed parameters>) -> <expression>` creates a
  value of the existing `Function` type; it is not a separate callable type.
- A small, closed set of [Class Protocol Methods](docs/PROTOCOLS.md) lets a
  Class define `==`, ordering, arithmetic, unary `-`, and `str()` behavior.
- A [Regex module](docs/REGEX.md) compiles patterns to a `Pattern` value with
  `matches`, `find`, `findAll`, `groups`, `replace`, and `split`.
- [Time](docs/TIME.md) supports local, UTC, fixed-minute-offset, and Unix
  millisecond representations without introducing a timezone database.
- The strict [CSV module](docs/CSV.md) transports raw String rows or
  header-keyed String records with native and persistent-REPL parity.
- The [Data module](docs/DATA.md) adds an immutable `Table` of String cells for
  filtering, sorting, grouping, and deriving columns; it infers no types, so
  numeric work stays an explicit `int(...)` / `real(...)` conversion.
- An expression lambda may read outside values through an explicit dependency
  list: `#name`/`Local name` for a lexical capture, `@name`/`Global name` for a
  module binding, as in
  `lambda [#minimum, @Maximum] (score: Int) -> score >= minimum and score <= Maximum`;
  neither kind is ever inferred or implicit.
- The [Statistics module](docs/STATISTICS.md) provides typed descriptive
  statistics over `List<Int>` and `List<Real>`, with no String coercion.
- The [Numeric module](docs/NUMERIC.md) adds immutable Real-oriented vectors,
  matrices, linear algebra, and additive `Vector` overloads in Plot.
- The [Word module](docs/WORD.md) builds immutable formatted documents, merged
  tables, embedded Plot images, and bounded semantic DOCX read-back without
  requiring Office or an external runtime.
- The formatter defines one canonical presentation while preserving comments.

## Build from source

AhdCode currently requires Go 1.25 or newer.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot
```

The command above installs the compiler and the local numeric/plot rendering helpers.
If you plan to use the `Latex` module, you must also stage the offline Latex runtime bundle. This requires a one-time network fetch to download pinned, checksummed resources:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

After staging, ordinary AhdCode Latex execution remains strictly offline.

Ensure Go's binary directory is on `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

## CLI quick start

```bash
ahdcode run examples/v0.1/01_hello.ahd
ahdcode build examples/v0.1/01_hello.ahd -o hello
ahdcode format examples/v0.1/01_hello.ahd
ahdcode format --check examples/v0.1/01_hello.ahd
ahdcode
```

See the [CLI guide](docs/CLI.md), [formatter guide](docs/FORMATTER.md), and
[REPL guide](docs/REPL.md).

## Documentation

- [Türkçe Öğrenci Rehberi](docs/STUDENT_GUIDE_TR.md)
- [English Student Guide](docs/STUDENT_GUIDE_EN.md)
- [Getting started](docs/GETTING_STARTED.md)
- [Language tour](docs/LANGUAGE_TOUR.md)
- [Types and null safety](docs/TYPES_AND_NULL.md)
- [Control flow](docs/CONTROL_FLOW.md)
- [Functions](docs/FUNCTIONS.md)
- [Classes](docs/CLASSES.md)
- [Class Protocol Methods](docs/PROTOCOLS.md)
- [Collections](docs/COLLECTIONS.md)
- [Modules](docs/MODULES.md)
- [Errors](docs/ERRORS.md)
- [Fundamentals](docs/FUNDAMENTALS.md)
- [String API](docs/STRING_API.md)
- [List API](docs/LIST_API.md)
- [Math module](docs/MATH.md)
- [Time module](docs/TIME.md)
- [Latex module](docs/LATEX.md)
- [Word module](docs/WORD.md)
- [File and Path modules](docs/FILESYSTEM.md)
- [Regex module](docs/REGEX.md)
- [CSV module](docs/CSV.md)
- [Data module](docs/DATA.md)
- [Statistics module](docs/STATISTICS.md)
- [Plot module](docs/PLOT.md)
- [Numeric module and Complex scalars](docs/NUMERIC.md)
- [JSON module](docs/JSON.md)
- [XML module](docs/XML.md)
- [Env module](docs/ENV.md)
- [Understanding diagnostics](docs/DIAGNOSTICS.md)
- [AI-assisted local setup](FOR_AI.md)
- [Curated v0.1 examples](examples/v0.1/README.md)
- [Full v0.1 language specification](AHDCODE_LANGUAGE_SPEC_v0.1.md)

## Editor extension

The local VS Code-compatible extension in [`editors/vscode`](editors/vscode)
recognizes `.ahd`, provides syntax highlighting, and runs the active file from
the editor title play button, Command Palette, or `F6`. The same VSIX targets
VS Code and Antigravity. See its [installation guide](editors/vscode/README.md).

## Current limitations

v0.1 intentionally has no block/statement lambdas, implicit/general mutable closure cells, tuple
returns, reflection, interfaces, multiple inheritance, debugger, LSP, package
search paths, or web runtime.
Operator behavior is user-definable only through the ten fixed
[Class Protocol Methods](docs/PROTOCOLS.md), not a general overloading
mechanism. Modules are sibling `.ahd` files, and the editor extension is a
lightweight run-and-highlight integration. See the
[specification's unsupported-feature list](AHDCODE_LANGUAGE_SPEC_v0.1.md#40-unsupported-v01-features).

## Repository map

```text
cmd/ahdcode/       CLI entry point
cmd/ahdnumeric/    bundled advanced linear-algebra helper
cmd/ahdplot/       bundled chart-rendering helper
internal/          compiler, runtime, formatter, and REPL
editors/vscode/    VS Code / Antigravity extension
docs/              end-user guides
examples/v0.1/     curated working programs
AHDCODE_LANGUAGE_SPEC_v0.1.md
                   authoritative language contract
```

## Development and credits

AhdCode is designed and specified by Ali Harun Daldallı. Implementation,
documentation, and testing have been developed with extensive AI assistance,
including OpenAI Codex, Anthropic Claude, and Google Gemini. Their roles vary
by task; language design and final technical decisions remain with the project
author.

## License

AhdCode is available under the [MIT License](LICENSE).
