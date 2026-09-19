# Packaging desktop applications

[English] · [Türkçe](PACKAGING_TR.md)

[Back to README](../README.md) · [CLI](CLI.md) · [GUI](GUI.md)

> Added in v2.0.0.

`ahdcode package` turns an AhdCode program into a desktop application that
runs on a computer without AhdCode installed.

```text
ahdcode package <entry.ahd> [--name <name>] [--output <folder>] [--icon <icon.png>]
                [--target <os-arch>] [--helpers <folder>] [--console]
```

```sh
ahdcode package examples/v2.0/ledger_app/main.ahd --name Ledger --icon ledger.png
```

| Option | Meaning | Default |
| --- | --- | --- |
| `--name` | The application's name: up to 64 letters, digits, spaces, dots, underscores, or hyphens | the entry file's name |
| `--output`, `-o` | The folder the application is written to | `dist` |
| `--icon` | A square PNG icon, 16 to 4096 pixels on a side | the AhdCode icon |
| `--target` | The platform to package for (see [Targets](#targets)) | this computer |
| `--helpers` | A folder holding that platform's AhdCode helpers | this installation's |
| `--console` | Windows only: keep a console window for a desktop program | no console |

There is no manifest file, package registry, or dependency manager: the
command line is the whole configuration.

## What an application holds

| Platform | Application |
| --- | --- |
| macOS | `Name.app` — `Contents/MacOS` holds the program and its helpers, `Contents/Resources` the icon (`AppIcon.icns`) |
| Windows | a folder `Name/` with `Name.exe` and `runtime/`, and the same as `Name-windows-x64.zip` |
| Linux | a folder `Name/` with `Name` and `runtime/`, and the same as `Name-linux-x64.tar.gz` |

An application holds exactly:

- the compiled program;
- the bundled helpers its program uses, decided from the compiled program
  itself: `ahdgui` for GUI; `ahdgraphics` for Graphics; `ahdplot`,
  `ahdplotview`, and `ahdgui` (for the viewer's Save dialog) for Plot;
  `ahdsqlite` for SQLite; `ahdnumeric` for Numeric; the LaTeX engine for
  Latex;
- `ahdcode-app.json` (the name and icon file name), the icon, and on macOS
  `Info.plist`, `PkgInfo`, and `AppIcon.icns`.

It never holds the AhdCode compiler, its Go toolchain, the language server,
documentation, AhdDataStudio, or any file from the project folder: nothing
beside the entry module is copied — not `.env`, `.git`, databases,
spreadsheets, CSV files, screenshots, logs, or source files. A program that
needs a data file creates it or asks for it with [GUI dialogs](GUI.md#dialogs).
Before anything is written, a leak check refuses the application if it would
hold any other file, a file with a forbidden name, or metadata that looks
like a secret. It cannot look inside the compiled program: a password written
in the program's own source is part of the program.

Packaging an application again replaces the application it made before. A
folder at the same place that `ahdcode package` did not make is never
removed; choose another `--output` or `--name`.

## Running without AhdCode

A packaged program finds its helpers only inside its own application — beside
its executable, or in `runtime/`. It never uses an AhdCode installation,
`PATH`, `AHDCODE_ROOT`, or the `AHDCODE_*_RUNTIME` variables, so the
application can be copied anywhere and to another computer of the same
platform. A missing helper is reported as missing from the application.

An application opened from the macOS Finder starts in the user's home
folder instead of the root folder, so a relative path such as
`"ledger.db"` means a file in the home folder. Started from a terminal, it
uses the terminal's current folder, as any program does.

## Name and icon

The application's windows — GUI windows, Graphics canvases, Plot viewers, and
dialogs — show the application's name and icon, not AhdCode's:

- macOS: the bundle's `Info.plist` names the application (identifier
  `org.ahdcode.app.<name>`), and the Dock shows `AppIcon.icns`. The bundle's
  main program opens no window of its own — its helpers do — so the bundle is
  marked as an agent (`LSUIElement`), and the helper windows carry the
  application's identity. The default AhdCode icon is given the rounded
  macOS shape; an icon passed with `--icon` is used as it is.
- Windows: the icon is linked into `Name.exe`, and the helper windows use it.
  A program that uses GUI, Graphics, or Plot opens no console window; use
  `--console` to keep one.
- Linux: the helper windows use the icon where the desktop shows window
  icons.

## Targets

| Target | Output | Status |
| --- | --- | --- |
| `macos-arm64` | `.app` | live-tested for this release |
| `windows-x64` | folder and `.zip` | package-tested (structure, executable format, icon resource); not run on Windows for this release |
| `linux-x64` | folder and `.tar.gz` | package-tested (structure, executable format); not run on Linux for this release |
| `macos-x64`, `windows-arm64`, `linux-arm64` | as above | compiled only |

By default, `ahdcode package` packages for the computer it runs on, with this
installation's helpers. A program is plain Go without C code, so AhdCode can
compile it for another platform too; the helpers, however, are built for one
platform each. To package for another platform, pass `--target` with
`--helpers` naming a folder with that platform's helpers — for example the
`libexec/ahdcode` folder of the AhdCode download for that platform:

```sh
ahdcode package app.ahd --target windows-x64 --helpers ~/Downloads/AhdCode-windows/libexec/ahdcode
```

There is no installer, `.msi`, `.deb`, `.rpm`, AppImage, or Flatpak output.

## macOS: signing and Gatekeeper

AhdCode does not ask for an Apple developer account. On Apple Silicon every
program must carry a signature, so the application's program is signed
ad hoc as it is built, and the bundled helpers keep AhdCode's own signatures;
the bundle as a whole is not signed with a developer certificate or
notarized.

An application you packaged opens on your own Mac. Copied to another Mac
through a browser download, AirDrop, or e-mail, macOS marks it as
downloaded, and Gatekeeper refuses to open an application that is not
notarized; the receiver can allow it once from **System Settings › Privacy &
Security**. Signing and notarizing with your own Developer ID is outside
`ahdcode package`.

## Errors

A packaging problem is reported with the code `PKG001` and nothing is
written: an invalid name or icon, an unknown target, a missing helper, a
cross-platform target without `--helpers`, a folder in the way, or a failed
leak check. Compile errors in the program are reported as for `ahdcode build`.
