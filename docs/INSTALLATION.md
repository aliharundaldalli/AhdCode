# Installation, upgrade, and removal

Download the published AhdCode **v2.11.0** stable package for your operating
system (macOS `arm64`/`x64`, Windows `x64`, or Linux `amd64`/`arm64`). `v2.11.0`
toolchain packages include the compiler, all helpers (`ahdsqlite`,
`ahdnumeric`, `ahdplot`, `ahdgraphics`, `ahdgui`, `ahdplotview`), embedded
AhdDataStudio, starters, and English documentation, and rely on the Go version
specified in `go.mod` on `PATH` for native `ahdcode build` and `ahdcode package`
commands. `v2.8.0` through `v2.10.0` are historical source-only releases. Self-contained
bundles that also embed private Go 1.27.0, the offline Tectonic LaTeX engine,
and the VS Code `.vsix` remain available on `v2.7.0` and return at `v3.0`.

## macOS (Apple Silicon and x64 packaging)

AhdCode `v2.11.0` distributes native `.pkg`, `.dmg`, and `.tar.gz` toolchain
packages for Apple Silicon (`arm64` — M1, M2, M3, M4 and later) and cross-compiled
Intel x86-64 (`x64`):

- `.pkg`: Recommended native installer (`AhdCode-<version>-macos-<arch>.pkg`, or
  `-unsigned` when built without Apple Developer ID signing and notarization).
  Double-click it in Finder or run
  `installer -pkg <file>.pkg -target CurrentUserHomeDirectory`.
- `.dmg`: Disk image wrapping the `.pkg` installer alongside `INSTALL.txt` and
  `VERSION`.
- `.tar.gz`: Portable toolchain archive containing `install.sh`, `INSTALL.txt`,
  `VERSION`, and `payload/`. Extract it and run `sh install.sh --setup-path`.

When signed with a Developer ID and notarized by Apple with the ticket stapled,
Gatekeeper accepts `.pkg` and `.dmg` artifacts without a warning: `.tar.gz`
carries Developer ID signed Mach-O binaries with the hardened runtime and a
secure timestamp, while `.pkg` and `.dmg` are both Developer ID signed and
Apple-notarized/stapled. Verify signed and notarized artifacts with Gatekeeper
enabled before or after downloading:

```sh
spctl --status
shasum -a 256 -c SHA256SUMS.txt
pkgutil --check-signature AhdCode-<version>-macos-arm64.pkg
xcrun stapler validate AhdCode-<version>-macos-arm64.pkg
xcrun stapler validate AhdCode-<version>-macos-arm64.dmg
spctl -a -vv -t install AhdCode-<version>-macos-arm64.pkg
spctl -a -vv -t open --context context:primary-signature AhdCode-<version>-macos-arm64.dmg
codesign --verify --deep --strict --verbose=2 path/to/AhdCode-<version>-macos-arm64/payload/bin/ahdcode
```

Local development builds created without notarization credentials
carry `-unsigned` in their filenames and `INSTALL.txt`; verify downloads
against `SHA256SUMS.txt` (`shasum -a 256 -c SHA256SUMS.txt`), and never disable
or bypass Gatekeeper protections.

The installer puts files under `~/Library/AhdCode/versions/<version>`, leaving
previously installed versions (such as `2.9.0`) intact on disk. `current`
selects the active version, and `~/Library/AhdCode/bin/ahdcode` is the stable
command added once (idempotently) to `~/.zprofile`, `~/.zshrc`, and
`~/.bash_profile`. No administrator password is required, nothing is written to
system directories, and existing projects, databases, and
`~/Library/Application Support/ahdcode` data are preserved.

Then open a **new** Terminal and run `ahdcode version`. A terminal that was
already open keeps the environment it started with; a new one picks up the
change immediately. To roll back to an earlier installed version (for example
`2.9.0`):

```sh
ln -sfn versions/2.9.0 ~/Library/AhdCode/.current-new && mv -fh ~/Library/AhdCode/.current-new ~/Library/AhdCode/current
```

To build macOS distribution packages from source:

```sh
python3 tooling/distribution/build.py --platform darwin --macos-arch both --without-bundled-runtimes --without-vsix --output dist/macos
```

Pass `--sign-app`, `--sign-installer`, and `--notary-profile` when Apple
Developer ID certificates and `notarytool` credentials are available.

Legacy v2.7.0 `AhdCode-2.7.0-macos-arm64.dmg` also carried `Install.command`
for anyone who prefers running `install.sh --setup-path`. The `.pkg` is the
recommended installer.

## Windows x64

Double-click `AhdCode-2.11.0-windows-x64.exe` in File Explorer (or run
`AhdCode-2.11.0-windows-x64.exe --silent` for automated setup). Setup is a small
graphical per-user program: it shows what it will install, unpacks and checks
its embedded payload with a progress window, and finishes with a confirmation.
No console, no terminal, and no typed commands are involved.

This release's installer is not Authenticode-signed, so Windows SmartScreen
may show an unknown-publisher warning. Verify the file against the published SHA-256 checksum before using
the system-provided **More info → Run anyway** path, and do not disable Windows
security protections.

Files go under `%LOCALAPPDATA%\AhdCode\versions\2.11.0` (with previous
versions such as `2.7.0` preserved alongside). The stable command
is `%LOCALAPPDATA%\AhdCode\bin\ahdcode.exe`, and only that one folder is added
to your user PATH — once, on first installation. Administrator rights and Git
are not required, and an uninstall entry is registered in Windows Installed Apps.
Native `ahdcode build` and `ahdcode package` commands use the Go version required
by `go.mod` on your `PATH`.

Then open a **new** PowerShell or Command Prompt and run `ahdcode --version`.
The installer broadcasts the per-user environment change, but terminals and IDEs
that were already open keep their original environment. Restart those programs
before checking from an integrated terminal. A fresh shell should resolve the
stable launcher with `where.exe ahdcode`.

Repairing or upgrading is safe: rerunning setup keeps the existing user PATH
entries exactly as they are and leaves one equivalent AhdCode entry. Uninstall
removes only equivalent `%LOCALAPPDATA%\AhdCode\bin` entries and preserves
unrelated PATH entries; it does not edit the machine PATH.

`AhdCode-2.11.0-windows-x64.exe --silent` installs with no windows at all, for
scripted deployment. Because setup is a graphical program, run it from a script
as `Start-Process -Wait` if you need to block until it finishes.

## Linux (x86-64 and arm64)

Choose the package for your processor — `uname -m` prints `x86_64` for
`amd64` and `aarch64` for `arm64`:

- `AhdCode-2.11.0-linux-amd64.tar.gz` — 64-bit Intel/AMD
- `AhdCode-2.11.0-linux-arm64.tar.gz` — 64-bit ARM (for example Ampere,
  AWS Graviton, Raspberry Pi 4/5 with a 64-bit OS)

```sh
tar -xzf AhdCode-2.11.0-linux-amd64.tar.gz
cd AhdCode-2.11.0-linux-amd64
sh install.sh --setup-path
```

The installer checks that the package runs on this machine before it
installs anything. Files live under `~/.local/share/ahdcode/versions/2.11.0`. The installer adds
one owned PATH block to `~/.profile`. Open a login shell or source that profile.

## What is included

The CLI embeds exact-version AhdDataStudio, first-party framework sources,
starters, Bootstrap, and English project documentation. The package also includes
private Go 1.27.0, `ahdsqlite`, `ahdnumeric`, `ahdplot`, `ahdgraphics`, `ahdgui`, `ahdplotview`, and the pinned offline
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
from VSIX...**, and select `vscode/ahdcode-0.2.3.vsix` from the
installation root. Google Antigravity IDE offers the same operation. The file
carries everything it needs; no npm and no network are involved.

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
