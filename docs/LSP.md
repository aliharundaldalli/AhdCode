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
own. Every diagnostic and every hover comes directly from the real AhdCode
frontend -- the same lexer, parser, and semantic analyzer `ahdcode build` and
`ahdcode run` use -- through a thin document-aware compilation layer
(`internal/analysis`) and an LSP protocol translation layer (`internal/lsp`).
A standard module's exported members (`Math.PI`, `Excel.read`, and so on) are
never hand-listed for the editor: they come from the same
`StandardModuleInterfaces()` the compiler itself resolves `bring` against, so
a future standard module participates in diagnostics and hover automatically,
with no separate catalog to update.

## v0.2.0 capabilities

This is a foundation release. It implements exactly:

- **Document synchronization** -- `textDocument/didOpen`, `didChange`,
  `didClose`, using **full** document sync (`TextDocumentSyncKind.Full`).
  Every `didChange` carries the document's complete new text; the server
  re-analyzes the whole document from that snapshot. There is no incremental
  edit application and no incremental compiler in v0.2.0 -- correctness comes
  before that optimization.
- **Diagnostics** (`textDocument/publishDiagnostics`) -- lexer, parser,
  module/import, and semantic diagnostics, each carrying the compiler's own
  stable code, severity, message, and source range. A diagnostic's `source`
  is always `"ahdcode"`. Fixing a document publishes an empty diagnostic list
  so stale markers clear. An error in an imported module is published under
  that module's own document, not folded into the importing file.
- **Hover** (`textDocument/hover`) -- for an identifier the compiler
  confidently resolved to a real symbol: a variable, `Constant`, or `Local`
  declaration or use; a function declaration or call; a function or
  structure parameter; a `Class`; or an imported standard-module member.
  Hovering anywhere else (an operator, a literal, whitespace) returns no
  hover rather than a guess.

The server analyzes **unsaved editor text**. It never writes an open
document's buffer back to its file on disk merely to compile it -- the same
in-memory-entry approach the REPL already uses for its own session source,
generalized to any number of open documents. An imported module that is also
open in the editor is analyzed from its own unsaved buffer too; anything not
open is read from the real filesystem.

## Not in v0.2.0

Completion, go to definition, document symbols, find references, rename,
signature help, semantic tokens/highlighting, inlay hints, code actions,
quick fixes, auto import, refactoring, workspace-wide indexing, an
incremental compiler or parser, and a persistent compiler cache are all
unimplemented in this release. The `initialize` response advertises exactly
the capabilities above and nothing else, so a client never believes it can
request a feature this version does not have. Later releases are expected to
add individual features on top of this same foundation.

## Position encoding

AhdCode source positions are one-based lines/columns counted in Unicode code
points. LSP positions are zero-based lines/UTF-16 code units. The server
converts between the two using the real source text on every request --
never a bare "subtract one" -- so a position after a non-BMP character (most
emoji, for example) lands on the correct editor character rather than half a
UTF-16 surrogate pair off.
