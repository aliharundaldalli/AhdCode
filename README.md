<p align="center">
  <img src="editors/vscode/images/ahdcode-logo.png" alt="AhdCode logo" width="360">
</p>

# AhdCode

[![CI](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml/badge.svg)](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml)

AhdCode is an experimental statically checked general-purpose programming
language focused on readable syntax, explicit intent, predictable semantics,
and native compilation.

The current release target is **v0.1**. The core language works end to end, but
the project is not production-ready and breaking changes may occur before 1.0.

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
- Flow-sensitive null checks make absence explicit without new nullable type
  syntax.
- Lists, Pairs, Classes, Functions, modules, errors, and native executables are
  part of the v0.1 core.
- The formatter defines one canonical presentation while preserving comments.

## Build from source

AhdCode currently requires Go 1.25 or newer.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode
```

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
- [Collections](docs/COLLECTIONS.md)
- [Modules](docs/MODULES.md)
- [Errors](docs/ERRORS.md)
- [Fundamentals](docs/FUNDAMENTALS.md)
- [String API](docs/STRING_API.md)
- [List API](docs/LIST_API.md)
- [Math module](docs/MATH.md)
- [Time module](docs/TIME.md)
- [Curated v0.1 examples](examples/v0.1/README.md)
- [Full v0.1 language specification](AHDCODE_LANGUAGE_SPEC_v0.1.md)

## Editor extension

The local VS Code-compatible extension in [`editors/vscode`](editors/vscode)
recognizes `.ahd`, provides syntax highlighting, and runs the active file from
the editor title play button, Command Palette, or `F6`. The same VSIX targets
VS Code and Antigravity. See its [installation guide](editors/vscode/README.md).

## Current limitations

v0.1 intentionally has no lambdas, tuple returns, user-defined operator
overloading, interfaces, multiple inheritance, debugger, LSP, package search
paths, or web runtime. Modules are sibling `.ahd` files, and the editor
extension is a lightweight run-and-highlight integration. See the
[specification's unsupported-feature list](AHDCODE_LANGUAGE_SPEC_v0.1.md#38-unsupported-v01-features).

## Repository map

```text
cmd/ahdcode/       CLI entry point
internal/          compiler, runtime, formatter, and REPL
editors/vscode/    VS Code / Antigravity extension
docs/              end-user guides
examples/v0.1/     curated working programs
AHDCODE_LANGUAGE_SPEC_v0.1.md
                   authoritative language contract
```

## License

AhdCode is available under the [MIT License](LICENSE).
