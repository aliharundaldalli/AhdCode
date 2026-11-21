# Packaging AhdCode applications

[English] · Türkçe

[Back to README](README.md) · [CLI](CLI.md) · [GUI](GUI.md)

> Added in v2.0.0.

`ahdcode package` turns an AhdCode program into a desktop application that
runs on a computer without AhdCode installed.

```text
ahdcode package <entry.ahd> [--name <name>] [--output <folder>] [--icon <icon.png>]
                [--target <os-arch>] [--helpers <folder>] [--console] [--web [--pwa]]
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
| `--web` | Include the project's reviewed public assets | off |
| `--pwa` | Include the PWA assets and check their layout. Requires `--web` | off |

There is no manifest file, package registry, or dependency manager: the
command line is the whole configuration.

## Deploy a Web application

```sh
ahdcode package app.ahd --web --name Site --output releases/v2.11
# For an app already wired with ahdcode pwa init:
ahdcode package app.ahd --web --pwa --name Site --output releases/v2.11-pwa
```

`--web` copies the reviewed `public/` tree beside the entry module and
retains its relative paths. `--pwa` additionally copies
`PWA/app.webmanifest`, `PWA/worker.js`, and `PWA/public/`. It requires
a valid manifest with local `/ahd-pwa/` icons and a registration script.
`PWA/Integration.ahd` is compiled from the app's source graph.
[PWA setup](PWA.md).

The Web package contains the executable and necessary helpers, with
the runtime assets beside the executable. Serve from that directory.
On Windows and Linux it is the top-level package directory. On macOS use
`Name.app/Contents/MacOS` as the working directory when running from a
terminal or service. For example, on Linux or macOS after changing to
the executable's directory:

```sh
APP_NAME=Site APP_ENV=production APP_PROTOCOL=https APP_HOST=example.com \
  SERVER_HOST=127.0.0.1 SERVER_PORT=8080 ./Site
```

On Windows the executable is `Site.exe`. Supply those variables through
the process environment, or create a private `.env` beside the executable
if the application loads one. For Admin, MVC, and CRUD, set the required
`DB_*` values and provision or securely move an initialized database
separately. Set persistent session file paths to writable private storage.
Supply other mail or application settings the same way. Behind Caddy, keep
the backend on loopback and pass the app and PWA routes through the same
HTTPS origin. [Local and production HTTPS](WEB.md#14-local-https).

The packager accepts CSS, JavaScript, static HTML, web manifests, common
images and fonts, and files named `LICENSE` in the approved public trees.
It rejects links and junctions, hidden paths, private directory names,
paths that collide on case-insensitive filesystems, unsupported file types,
and recognizable PEM private keys. Keep the
public trees reviewed. The packager cannot identify every secret in
otherwise valid JavaScript or other public content. It never copies
project configuration, source modules, or database files automatically.
Custom asset roots must be deployed separately at their configured paths.

Web builds refuse existing output directories and archive names. Choose
a fresh destination for each deployment and switch your service to it
after verification. Relative URLs, existing MIME types, and worker cache
headers stay under the same application routes. The zip or tar archive
uses sorted entries and fixed timestamps.

For a manual deployment, `ahdcode build app.ahd -o Site` creates the
executable alone. Supply the same asset layout and any applicable
runtime helpers explicitly.

## What a desktop application holds

| Platform | Application |
| --- | --- |
| macOS | `Name.app` — `Contents/MacOS` holds the program and its helpers, `Contents/Resources` the icon (`AppIcon.icns`), and `Contents/Resources/app` bundled images |
| Windows | a folder `Name/` with `Name.exe`, `runtime/`, and `resources/` for bundled images, and the same as `Name-windows-x64.zip` |
| Linux | a folder `Name/` with `Name`, `runtime/`, and `resources/` for bundled images, and the same as `Name-linux-x64.tar.gz` |

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
documentation, or AhdDataStudio. Nothing beside the entry module is copied
except `.png`, `.jpg`, and `.jpeg` files in a `resources` folder next to that
module. Those images are copied into the application's own resource folder:
`Contents/Resources/app` on macOS, and `resources` beside the program on
Windows and Linux. Any other file in that folder, including `.env`, `.git`,
databases, spreadsheets, CSV files, logs, and source files, stops packaging.
A symbolic link in that folder, including a `resources` folder that is itself
a symbolic link, also stops packaging. The link is not followed, so packaging
cannot copy a file that lives outside the folder.
A program that needs a data file creates it or asks for it with
[GUI dialogs](GUI.md#dialogs). A picture loaded with `GUI.resource` uses this
folder. `image("mark.png")` still uses the working directory.
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
uses the terminal's current folder, as any program does. Bundled images do
not use that folder. `GUI.resource` reads the application's resource folder.

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

| Target | Output | Notes |
| --- | --- | --- |
| `macos-arm64`, `windows-x64`, `linux-x64`, `linux-arm64` | native bundle / archive | standard targets (`--target` defaults to the host platform) |
| `macos-x64`, `windows-arm64` | `.app`, folder, or `.zip` | cross-compiled targets when matching `--helpers` are supplied |

By default, `ahdcode package` packages for the computer it runs on, with this
installation's helpers. A program is plain Go without C code, so AhdCode can
compile it for another platform too; the helpers, however, are built for one
platform each. To package for another platform, pass `--target` with
`--helpers` naming a folder with matching helpers built for that target:

```sh
ahdcode package app.ahd --target windows-x64 --helpers ./helpers/windows-x64
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
