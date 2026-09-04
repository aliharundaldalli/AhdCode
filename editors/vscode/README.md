# AhdCode editor extension

[English] · [Türkçe](README_TR.md)

This minimal extension adds AhdCode file recognition, lightweight syntax highlighting, a **Run AhdCode File** play button, and a connection to the AhdCode language server (compiler-backed diagnostics, hover, completion with auto import and access-aware Class members, go to definition, document symbols, signature help, find references, rename, semantic highlighting, inlay hints, quick fixes, formatting, workspace symbol search, folding ranges, and selection ranges) to VS Code-compatible editors.

The extension supplies AhdCode package branding and light/dark language icons.
An active third-party File Icon Theme may override the language icon; file
recognition and all run actions continue to work normally.

## Run the active file

Open a saved `.ahd` file and use one of:

- the play button in the editor title bar;
- `Run AhdCode File` in the Command Palette;
- `F6` while the `.ahd` editor has focus.

The extension saves the document, then starts a visible task named `AhdCode: Run filename.ahd`. It invokes the executable with an argument array equivalent to:

```text
ahdcode run /absolute/path/to/file.ahd
```

No shell command is constructed. The task runs with the file's containing directory as its working directory, so spaces and Unicode characters in paths are passed without shell quoting issues. A dedicated, cleared task terminal is reused for repeated runs.

## Executable discovery

By default, `ahdcode` must be on the environment `PATH` inherited by the editor. If the editor was launched before PATH changed, restart it.

Alternatively set `ahdcode.executablePath` to the absolute path of the executable. The setting is intentionally empty by default and does not contain a machine-specific path.

## Language server

Alongside Run File, the extension starts the same `ahdcode` executable as a background [language server](../../docs/LSP.md) (`ahdcode lsp`) once, when the extension activates -- resolved through the exact same `ahdcode.executablePath` setting / `PATH` lookup Run File uses. It communicates over stdio only; nothing about it opens a network port. It gives you:

- **Diagnostics**: lexer, parser, module/import, and semantic errors from the real compiler frontend, shown as normal editor problem markers -- kept live as you type, including in unsaved buffers, and cleared automatically once you fix them.
- **Hover**: hovering a resolved symbol shows its compiler-resolved type or signature.
- **Go to Definition**, **Document Symbols**, **Signature Help**, **Find References** (compile-graph scoped), **Completion** (modules, exports, members, locals, auto import), **Rename**, **Semantic Highlighting**, **Inlay Hints**, **Quick Fixes**, **Format Document**, **Workspace Symbols**, **Folding**, and **Selection Range** -- all through ordinary LSP capability negotiation.

The server never writes an open document back to its file on disk merely to analyze it.

If the executable cannot be found, or the server fails to start, one concise error message is shown -- not one per keystroke -- and Run File keeps working normally regardless. See [the language server's scope and limitations](../../docs/LSP.md) for honest boundaries (compile-graph-scoped references/rename, on-demand module discovery, no persistent workspace index).

## Development

1. Open `editors/vscode` in VS Code.
2. Press `F5` and choose **Run AhdCode Extension** if prompted.
3. In the Extension Development Host, open a `.ahd` file and use the play button.

Run the dependency-free tests with:

```bash
npm test
```

## Package and install locally

From `editors/vscode`:

```bash
npm run package
```

This runs `@vscode/vsce package` and creates a local `.vsix`; it does not publish anything.

Install in VS Code from the Command Palette with **Extensions: Install from VSIX...**, or with:

```bash
code --install-extension ahdcode-0.2.3.vsix
```

Google Antigravity IDE 1.107 exposes the same local VSIX CLI operation:

```bash
antigravity-ide --install-extension ahdcode-0.2.3.vsix
```

On macOS, if those launchers are not on PATH, use the application-bundled launchers:

```bash
/Applications/Visual\ Studio\ Code.app/Contents/Resources/app/bin/code --install-extension ahdcode-0.2.3.vsix
/Applications/Antigravity\ IDE.app/Contents/Resources/app/bin/antigravity-ide --install-extension ahdcode-0.2.3.vsix
```

The same package is used by both editors. The extension API baseline is VS Code 1.107, matching the tested Antigravity standalone extension host.

## Scope and limitations

This is intentionally a small extension: syntax highlighting, Run File, and a [language server](../../docs/LSP.md) with the v0.2.2 practical everyday feature set. It does not provide a debugger, breakpoints, or Marketplace publishing. Find references and rename are compile-graph scoped, not workspace-wide. Run File output still shows only as task-terminal output.
