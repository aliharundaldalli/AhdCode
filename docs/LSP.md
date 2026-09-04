# Language server

[English] · [Türkçe](LSP_TR.md)

[Back to README](../README.md) · [CLI](CLI.md) · [Language Tour](LANGUAGE_TOUR.md)

```bash
ahdcode lsp
```

starts the AhdCode language server. It speaks the standard [Language Server
Protocol](https://microsoft.github.io/language-server-protocol/) over
**stdin/stdout only** -- there is no TCP port, no HTTP endpoint, no browser,
no daemon, and no socket mode. `stdout` carries protocol frames exclusively;
`ahdcode lsp` never writes a version banner, a log line, or any other
human-readable text to it. Any editor that speaks LSP can launch it as a
child process and talk to it -- the VS Code extension in
[`editors/vscode`](../editors/vscode) is one client among possible others,
not a special case.

## What the compiler, not the LSP, decides

The language server has no parser, type checker, or symbol catalog of its
own. Every diagnostic, hover, completion item, rename target, semantic token,
inlay hint, and quick fix comes directly from the real AhdCode frontend --
the same lexer, parser, and semantic analyzer `ahdcode build` and
`ahdcode run` use -- through a thin document-aware compilation layer
(`internal/analysis`) and an LSP protocol translation layer (`internal/lsp`).
A standard module's exported members (`Math.PI`, `Excel.read`, and so on) are
never hand-listed for the editor: they come from the same
`StandardModuleInterfaces()` the compiler itself resolves `bring` against.

## Capabilities (v0.2.2)

The practical everyday AhdCode LSP feature set is **complete** as of v0.2.2.
The `initialize` response advertises exactly what exists below and nothing
else (no incremental sync, no range formatting, no semantic-token deltas, no
call/type hierarchy, no debugger).

- **Document synchronization** -- `textDocument/didOpen`, `didChange`,
  `didClose`, using **full** document sync. Every `didChange` carries the
  document's complete new text; the server re-analyzes the whole document
  from that snapshot. There is no incremental compiler.
- **Diagnostics** (`textDocument/publishDiagnostics`) -- lexer, parser,
  module/import, and semantic diagnostics with stable compiler codes.
  Fixing a document publishes an empty list so stale markers clear.
- **Hover** (`textDocument/hover`) -- compiler-resolved symbols only; no
  guesses on literals, operators, or whitespace.
- **Go to Definition** (`textDocument/definition`) -- across the compile
  graph, including imported modules.
- **Document Symbols** (`textDocument/documentSymbol`) -- top-level
  declarations and Class members as children.
- **Signature Help** (`textDocument/signatureHelp`) -- active call
  signature with parameter tracking.
- **Find References** (`textDocument/references`) -- every use of the
  symbol at the cursor, scoped to the **current compile graph** (the open
  entry document plus everything it transitively imports). This is not a
  workspace-wide index.
- **Completion** (`textDocument/completion`) -- module names after
  `bring`/`from`; exported names after `from <module> bring`; namespace or
  Class members after `.` (including **access-aware Confidential members**:
  suggested only when the compiler says they are accessible from the cursor
  context); in-scope locals, parameters, and module-root declarations; auto
  import for uniquely discoverable exported symbols from sibling user modules;
  and a restrained keyword set.
- **Rename** (`textDocument/prepareRename`, `textDocument/rename`) --
  semantic-symbol renames using the same identity as Definition and
  References, scoped to the compile graph. Invalid identifiers, keywords,
  literals, operators, builtins, and unresolved symbols are rejected.
  `prepareRename` validates new names with the real lexer rules.
- **Semantic Tokens** (`textDocument/semanticTokens/full`) -- highlighting
  from compiler/AST facts (namespace, type, function, method, parameter,
  variable, property, keyword, string, number, comment; declaration and
  readonly modifiers where truthful). Positions are UTF-16 correct.
- **Inlay Hints** (`textDocument/inlayHint`) -- inferred types on
  declarations that omit an explicit type, and restrained parameter-name
  hints on positional call arguments when the selected callable is known.
- **Code Actions** (`textDocument/codeAction`) -- conservative quick fixes
  only, each tied to a structured compiler diagnostic:
  - `SEM006` -- add missing `Local` in a nested executable scope
  - `PAR009` (for-loop binding message) -- remove invalid `Local` from a
    `for` iteration binding
  - export-not-found import diagnostics -- import the missing symbol when
    uniquely identified
- **Document Formatting** (`textDocument/formatting`) -- calls the existing
  in-process formatter on the open buffer; never shells out and never writes
  the file to disk.
- **Workspace Symbols** (`workspace/symbol`) -- on-demand search across
  discoverable `.ahd` modules in workspace roots and the entry document's
  directory. No persistent index; Confidential exports are omitted
  cross-module.
- **Folding Range** (`textDocument/foldingRange`) -- function/Class bodies,
  control-flow blocks, and similar AST-backed spans.
- **Selection Range** (`textDocument/selectionRange`) -- progressive
  expansion through AST ancestors.

The server analyzes **unsaved editor text**. It never writes an open
document's buffer back to its file on disk merely to compile it. An imported
module that is also open in the editor is analyzed from its own unsaved
buffer; anything not open is read from the filesystem.

A file that uses [`require(...)`](REQUIRE.md) is analyzed as part of the
nearest ancestor `app.ahd` that itself contains `require(...)`, not as a
private mini-program whose directory is the application root. That is the
same composition `ahdcode build` uses, so a `Pages/Home.ahd` buffer sees
helpers from `Shared/` and does not report false "file not found" errors on
its own `require("Shared/...")` paths. Diagnostics stay on the file that
owns the span -- a type error in a required file is not painted onto the
entry's `require("...")` string.

## Auto import and module discovery

Auto import and workspace symbols discover user modules by scanning workspace
roots (from LSP `initialize`) plus the entry document's directory for sibling
`.ahd` files. Standard modules come from `StandardModuleInterfaces()`. There
is no hard-coded symbol registry, no background watcher, and no persistent
database. When two modules export the same name, completion shows distinct
entries with module details rather than silently picking one.

## Not implemented (by design)

Incremental parser/compiler, persistent workspace indexing, semantic-token
deltas, range formatting, call/type hierarchy, code lens, debugger/DAP, AI
completion, and generative code actions remain out of scope. Later releases
may improve performance and add richer refactorings without changing the v0.2.2
feature boundary above.

## Position encoding

AhdCode source positions are one-based lines/columns counted in Unicode code
points. LSP positions are zero-based lines/UTF-16 code units. The server
converts between the two using the real source text on every request --
never a bare "subtract one" -- so a position after a non-BMP character (most
emoji, for example) lands on the correct editor character rather than half a
UTF-16 surrogate pair off.
