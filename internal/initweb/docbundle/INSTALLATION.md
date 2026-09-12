# Installation, upgrade, and removal

Download the package for your operating system from the AhdCode release you
want to install. File names below use `<version>` for that release's number.
Every package is self-contained: it carries the compiler, a private Go
toolchain, AhdDataStudio, the SQLite, numeric and plot helpers, an offline
LaTeX engine, the project starters, and the English documentation.

## macOS (Apple Silicon)

The macOS package targets Apple Silicon Macs — M1, M2, M3, M4 and later arm64
models. There is no Intel build.

Double-click `AhdCode-<version>-macos-arm64.pkg` and follow the installer. The
package is signed with a Developer ID and notarized by Apple, so it opens
normally; no security workaround is needed. It installs for your account only,
asks for no administrator password, and writes nothing outside your home
folder.

Files go under `~/Library/AhdCode/versions/<version>`. `current` selects the
active version and `~/Library/AhdCode/bin/ahdcode` is the stable command. That
one folder is added to your PATH.

Then open a **new** Terminal and run `ahdcode --version`. A terminal that was
already open keeps the environment it started with; a new one picks up the
change immediately.

`AhdCode-<version>-macos-arm64.zip` is an alternate download that bundles the same
package together with the VS Code extension. A `.dmg` with the same payload is
also published for anyone who prefers a disk image; the `.pkg` is the
recommended installer.

## Windows x64

Double-click `AhdCode-<version>-windows-x64.exe` in File Explorer. Setup is a small
graphical per-user program: it shows what it will install, unpacks and checks
its embedded payload with a progress window, and finishes with a confirmation.
No console, no terminal, and no typed commands are involved.

The installer is not currently code-signed, so Windows SmartScreen may show an
unknown-publisher warning. Choose **More info** and then **Run anyway** to
continue. The published SHA-256 checksums let you confirm you have the official
file.

Files go under `%LOCALAPPDATA%\AhdCode\versions\<version>`. The stable command
is `%LOCALAPPDATA%\AhdCode\bin\ahdcode.exe`, and only that one folder is added
to your user PATH — once, on first installation. Administrator rights, Git, and
a system Go installation are not required, and an uninstall entry is registered
in Windows Installed Apps.

Then open a **new** PowerShell or Command Prompt and run `ahdcode --version`.

`AhdCode-<version>-windows-x64.exe --silent` installs with no windows at all, for
scripted deployment. Because setup is a graphical program, run it from a script
as `Start-Process -Wait` if you need to block until it finishes.
`AhdCode-<version>-windows-x64.zip` bundles the same installer with the VS Code
extension.

## Linux x64

Extract `AhdCode-<version>-linux-x64.tar.gz` and run:

```sh
tar -xzf AhdCode-<version>-linux-x64.tar.gz
cd AhdCode-<version>
sh install.sh --setup-path
```

Files live under `~/.local/share/ahdcode/versions/<version>`. The installer adds
one owned PATH block to `~/.profile`. Open a login shell or source that profile.

## What is included

The CLI embeds exact-version AhdDataStudio, first-party framework sources,
starters, Bootstrap, and English project documentation. The package also includes
private Go 1.27.0, `ahdsqlite`, `ahdnumeric`, `ahdplot`, and the pinned offline
Tectonic 0.17.0 engine plus its pinned resource bundle — 6,350,367 bytes since
v1.3.0 added fancyhdr and lastpage to the v1.2.0 TikZ, nine TikZ libraries, and
pgfornament. The uncompressed
engine size varies by platform. Native AhdCode builds use private Go before PATH;
the user's Go installation and settings are not changed. Core builds, SQLite,
Studio, and LaTeX require no network. MySQL and SMTP still require the external
servers you configure. See the payload's `THIRD_PARTY_NOTICES.md` and `licenses/`.

## Visual Studio Code extension

Every package carries the editor extension under `vscode/` in the installation
root, next to a short `README.txt`. AhdCode does not need it, and setup never
installs it for you.

In VS Code, open the Extensions view, open its `...` menu, choose **Install
from VSIX...**, and select `vscode/ahdcode-<version>.vsix` from the
installation root. Google Antigravity IDE offers the same operation. The file
carries everything it needs; no npm and no network are involved. The same
`.vsix` is published beside the platform artifacts as a standalone download.

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

On macOS, remove the installation root and its PATH block:

```sh
sh "$HOME/Library/AhdCode/current/install.sh" --uninstall
```

On Linux:

```sh
sh "$HOME/.local/share/ahdcode/current/install.sh" --uninstall
```

On Windows, use Installed Apps, which runs the installation root's
`uninstall.ps1`. Confirm `YES` when prompted. Removal deletes the owned product
root and only its PATH entry/block. It preserves projects, databases, `.env`
files outside that root, source repositories, registry data, and Studio/build
caches. Close AhdCode applications before removal. No system service is stopped.

## Isolated package verification

macOS/Linux setup accepts `--prefix /absolute/new/install-root` without changing
shell profiles. Invoke `<install-root>/bin/ahdcode`; test removal with the same
`--prefix` and `--uninstall`. Do not point a prefix at a project or existing
unowned directory. Release packaging is reproduced with
`python3 tooling/distribution/build.py --help` from the source checkout.
