# AhdCode editor extension

[English] · [Türkçe](README_TR.md)

This minimal extension adds AhdCode file recognition, lightweight syntax highlighting, and a **Run AhdCode File** play button to VS Code-compatible editors.

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
code --install-extension ahdcode-0.1.4.vsix
```

Google Antigravity IDE 1.107 exposes the same local VSIX CLI operation:

```bash
antigravity-ide --install-extension ahdcode-0.1.4.vsix
```

On macOS, if those launchers are not on PATH, use the application-bundled launchers:

```bash
/Applications/Visual\ Studio\ Code.app/Contents/Resources/app/bin/code --install-extension ahdcode-0.1.4.vsix
/Applications/Antigravity\ IDE.app/Contents/Resources/app/bin/antigravity-ide --install-extension ahdcode-0.1.4.vsix
```

The same package is used by both editors. The extension API baseline is VS Code 1.107, matching the tested Antigravity standalone extension host.

## Scope and limitations

This is intentionally a small run-and-highlight integration. It does not provide an LSP, completion, semantic highlighting, a debugger, breakpoints, or Marketplace publishing. Compiler and runtime diagnostics are shown as normal task-terminal output; they are not converted into editor problem entries.
