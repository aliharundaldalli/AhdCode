# Installation, upgrade, and removal

Use the RC package for your operating system. RC packages are candidates for
independent QA; final v1.0.0 is not yet released. Windows live installation QA
is required before claiming Windows release readiness.

## macOS (Apple Silicon)

Open `AhdCode-1.0.0-rc.1-macos-arm64.dmg`, then run `Install.command`.
The installer explains its per-user location and PATH change before installation.
It installs into `~/Library/Application Support/AhdCode/versions/<version>`.
`current` selects the active version; `bin/ahdcode` is the stable launcher.
One clearly marked, removable PATH block is added to `~/.zprofile`.
Open a new login shell and run `ahdcode --version`.

This RC is not Developer ID signed or notarized. Its checksum identifies the
candidate bytes. Distribution signing/notarization remains required for a normal
trusted macOS download experience before final release.

## Windows x64

Run `AhdCode-1.0.0-rc.1-windows-x64.exe`. This is a native per-user setup program
with a console confirmation. It extracts and checks its embedded payload, keeps
versions under `%LOCALAPPDATA%\AhdCode\versions`, and adds only its stable `bin`
directory to the user PATH. It does not require administrator access or Git.
It registers an uninstall entry in Windows Installed Apps. Sign out and back in
if an existing terminal launcher has cached the old environment.
Windows live installation and removal QA is still required. The RC is unsigned.

## Linux x64

Extract `AhdCode-1.0.0-rc.1-linux-x64.tar.gz` and run:

```sh
sh install.sh --setup-path
```

Files live under `~/.local/share/ahdcode/versions/<version>`. The installer adds
one owned PATH block to `~/.profile`. Open a login shell or source that profile.

## What is included

The CLI embeds exact-version AhdDataStudio, first-party framework sources,
starters, Bootstrap, and English project documentation. The package also includes
private Go 1.27.0, `ahdsqlite`, `ahdnumeric`, `ahdplot`, and the pinned offline
Tectonic 0.17.0 engine plus a 5,546,077-byte resource bundle. The uncompressed
engine size varies by platform. Native AhdCode builds use private Go before PATH;
the user's Go installation and settings are not changed. Core builds, SQLite,
Studio, and LaTeX require no network. MySQL and SMTP still require the external
servers you configure. See the payload's `THIRD_PARTY_NOTICES.md` and `licenses/`.

## First application

```sh
ahdcode --version
mkdir demo
cd demo
ahdcode init web mvc
ahdcode dev app.ahd
```

The wizard creates SQLite and registers it without a manual path variable.
Use `ahdcode databases` in another terminal to open bundled AhdDataStudio.
Use `ahdcode init web crud` in a separate empty directory for the same application
with flatter source organization. `AHDCODE_ROOT` is a developer override only.

`.test` names follow the existing platform authorization flow. If hosts changes
are declined or unavailable, use the printed loopback URL. Local development is
HTTP. Do not run hosts-recovery commands for every project.

## Upgrade and uninstall

Install a different version over the same owned root. Previous versions remain
on disk; the stable launcher switches to the new version. Installing the same
version does not overwrite its files. A legacy Go-installed CLI is not removed;
PATH ordering determines which CLI runs (`command -v ahdcode` / `where ahdcode`).

On macOS:

```sh
sh "$HOME/Library/Application Support/AhdCode/current/install.sh" --uninstall
```

On Linux:

```sh
sh "$HOME/.local/share/ahdcode/current/install.sh" --uninstall
```

On Windows, use Installed Apps or run the installation root's `uninstall.ps1`
with PowerShell. Confirm `YES` when prompted. Removal deletes the owned product
root and only its PATH entry/block. It preserves projects, databases, `.env`
files outside that root, source repositories, registry data, and Studio/build
caches. Close AhdCode applications before removal. No system service is stopped.

## Isolated package verification

macOS/Linux setup accepts `--prefix /absolute/new/install-root` without changing
shell profiles. Invoke `<install-root>/bin/ahdcode`; test removal with the same
`--prefix` and `--uninstall`. Do not point a prefix at a project or existing
unowned directory. Release packaging is reproduced with
`python3 tooling/distribution/build.py --help` from the source checkout.
