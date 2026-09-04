# require(...)

[English] · [Türkçe](REQUIRE_TR.md)

[Back to README](../README.md) · [CLI](CLI.md) · [Modules](MODULES.md)

`require("Path/To/File.ahd")` (v0.14) composes another local `.ahd` source
file into this program at **compile time**. It is not a runtime include, not
dynamic loading, not a package import, and not a textual preprocessor: the
compiler resolves it, parses the target file, and folds its declarations
into the same program, before semantic analysis ever runs. A built program
never reads its own `.ahd` sources again — moving or deleting the original
source tree does not affect a binary `ahdcode build` already produced.

```ahd
require("Components/Navbar.ahd")

Navbar()
```

## `bring` versus `require`

- `bring HTTP` means *use a standard AhdCode module*.
- `require("Pages/Home.ahd")` means *compose this local source file into the
  application*.

`bring` is unchanged: it still resolves standard modules (and, as before,
sibling `.ahd` files brought by module name), each analyzed as its own
compilation unit with its own exported interface. `require(...)` is a
different, flatter mechanism, described below — the two are not
interchangeable and neither is being redesigned into the other.

## Literal paths only

The argument must be a single, plain, compile-time string literal — no
interpolation, no concatenation, no variable:

```ahd
require("Pages/Home.ahd")              // valid

path: String := "Pages/Home.ahd"
require(path)                          // rejected: PAR014, not a literal

require("Pages/" + name)               // rejected: PAR014, not a literal
```

## Module root only

`require(...)` is a module-root statement, exactly like `bring`. It is
rejected inside a `Function` body, a loop, a condition, an `attempt` block,
or any other nested scope (`PAR005`) — the statement is still parsed, so the
rest of the file recovers normally, but nothing is composed from a rejected
`require`.

## `.ahd` only

Only AhdCode source files can be required. `require("data.json")` or
`require("public/app.css")` is rejected (`SEM048`); static assets are a
separate concern — see [Server.static](HTTP.md#static-files).

## Application-root-relative resolution

Every `require(...)` path is resolved relative to the **application root**
— the directory containing the entry `.ahd` file — never relative to the
file that wrote the `require(...)`. Given:

```
/project/app.ahd
/project/Components/Nav.ahd
/project/Shared/Theme.ahd
```

with `app.ahd` containing `require("Components/Nav.ahd")`, and
`Components/Nav.ahd` itself containing `require("Shared/Theme.ahd")`, the
second path still resolves to `/project/Shared/Theme.ahd`, **not**
`/project/Components/Shared/Theme.ahd`. A require path means the same file
everywhere it is written, no matter how deep the require chain is.

## Path security

Absolute paths are rejected (`SEM048`). A path that would escape the
application root — through plain `../` traversal or through a symlink whose
canonical target lands outside the root — is rejected the same way,
`require("../secret.ahd")` included. A symlink whose canonical target stays
inside the root is followed normally; that is ordinary filesystem behavior,
not a special case.

## Deduplication and canonical identity

Two spellings of the same file — `require("Shared/A.ahd")` and
`require("Shared/./A.ahd")`, from the same file or different ones — resolve
to one canonical identity and are compiled exactly once. Composition order
is deterministic for a given source tree: it follows the explicit require
edges the compiler discovers, never filesystem directory enumeration order.

## Cycles

A require cycle (`A.ahd` → `B.ahd` → `C.ahd` → `A.ahd`) is a compile error
(`SEM047`) naming the chain; it never hangs and never recurses until stack
overflow.

## Missing files

`require("Pages/Missing.ahd")` fails cleanly (`SEM046`) naming the
requesting file, the literal path as written, and the resolved path the
compiler expected to find — never a bare Go error.

## Shared declarations, file-local `bring`

Every required file's top-level declarations — `Function`, `Class`, module
constants — join **one shared application namespace**. A function declared
in `Components/Card.ahd` is called unqualified from any other required
file, with no package or namespace syntax:

```ahd
// Components/Card.ahd
Card: Function := (title: String) -> HTMLNode {
    ...
}

// app.ahd
require("Components/Card.ahd")
Card("Hello")
```

A duplicate top-level declaration name across two required files is the
same `SEM002` duplicate-declaration error a single file would raise.

**`bring` does not follow this sharing.** A required file must declare the
standard modules it uses itself:

```ahd
// Components/Card.ahd
bring HTML
from HTML bring HTMLNode
```

`app.ahd` doing `bring HTTP` does not excuse `Components/Card.ahd` from
declaring its own `bring HTML` to use `HTML.*` or `HTMLNode` — and the
reverse is equally true. A file that references a name only available
through another file's `bring` fails with `SEM049`, naming the module it
needs to bring itself.

## Offline, deterministic, no package manager

`require(...)` never touches the network, never consults a registry, and
never scans the filesystem for candidates beyond the paths a program's own
`require(...)` statements name. There is no `ahd.toml`, no manifest, no
semantic version, no lockfile, no remote/Git/URL dependency, and no package
cache — `require(...)` is local application source composition, nothing
more.

## `ahdcode dev` and `require(...)`

`ahdcode dev` watches the entry file plus the resolved `require(...)` graph
plus any `require(...)` target the latest build attempt named but could not
find yet, so creating a missing required file rebuilds automatically
without touching the file that requires it. See the
[CLI guide](CLI.md#dev-watch-scope).
