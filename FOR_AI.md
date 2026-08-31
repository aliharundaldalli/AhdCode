# AhdCode local setup guide for AI coding agents

This is a practical setup and verification guide, not a language
specification. The official repository is:

```text
https://github.com/aliharundaldalli/AhdCode
```

Support in this guide is limited to native macOS and native Windows. Do not
substitute WSL instructions for Windows, and do not infer Linux steps.

## Safety contract

Before changing the machine or repository:

1. Detect the operating system.
2. Inspect the existing tools and checkout.
3. Explain every missing prerequisite and why it is needed.
4. Ask the user for explicit permission before installing system software,
   invoking an installer/package manager, elevating privileges, or changing a
   shell profile or system/user `PATH`.

Never silently use `sudo`, Homebrew, `apt`, `winget`, Chocolatey, an installer,
or administrator privileges. Never run `git reset --hard`, `git clean`, delete
uncommitted files, commit secrets, expose API keys, push, tag, create a release,
or modify AhdCode language semantics unless the user explicitly asks for that
specific action. Preserve unrelated user work.

Do not trust an older globally installed `ahdcode`. Build the current checkout
and verify the executable at its explicit freshly installed path.

If the repository directory already exists, do not clone another copy. Enter
it and inspect before doing anything else:

```bash
git status --short
git branch --show-current
git log --oneline -5
```

On PowerShell the same Git commands are valid. If the repository does not
exist in the user-selected workspace, clone only the official URL:

```bash
git clone https://github.com/aliharundaldalli/AhdCode.git
cd AhdCode
```

## macOS

### 1. Inspect prerequisites

```bash
sw_vers
git --version
go version
code --version
node --version
npm --version
```

Git and Go are required for the compiler; AhdCode currently requires Go 1.25
or newer. VS Code and its `code` launcher are required only for editor-extension
installation. Node.js/npm are required only to test and package the repository's
local VSIX. If any needed command is missing or Go is too old, stop, explain
the gap, and ask before installing or changing `PATH`.

### 2. Inspect or clone the repository

Use the existing-checkout safety commands above. Clone only when no checkout
exists. Do not discard a dirty worktree.

### 3. Build, test, install, and verify the CLI

From the repository root:

```bash
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot
ahdcode_exe="$(go env GOPATH)/bin/ahdcode"
"$ahdcode_exe" --version
```

The expected current result is `AhdCode v0.1.15`. Using the explicit
`$ahdcode_exe` path proves which binary was tested. If the user wants the
short `ahdcode` command and that directory is not already on `PATH`, explain
the temporary or persistent options and obtain permission before editing a
shell profile. For a temporary current-shell setting only:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

If the user wants Latex support, explicitly ask for permission to stage the Latex runtime, as this performs an installation-time network operation to fetch pinned/checksummed resources. Do not use a system TeX fallback.

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

### 4. Run a real smoke program

```bash
smoke_dir="$(mktemp -d)"
printf '%s\n' 'write("AhdCode is ready.")' > "$smoke_dir/ready.ahd"
"$ahdcode_exe" run "$smoke_dir/ready.ahd"
rm -f "$smoke_dir/ready.ahd"
rmdir "$smoke_dir"
```

Require the output `AhdCode is ready.` before declaring CLI setup complete.
The temporary directory is outside the repository and is removed by exact
path; do not delete or clean repository files.

### 5. Test, package, install, and verify the VS Code extension

From the repository root:

```bash
cd editors/vscode
npm test
npm run package
code --install-extension ahdcode-0.1.4.vsix
code --list-extensions --show-versions | grep '^ahdcode-local.ahdcode@0.1.4$'
cd ../..
```

`npm run package` uses the repository's actual `@vscode/vsce package` script
and creates a local VSIX; it does not publish to Marketplace. Its `npx --yes`
step may download a temporary packaging dependency, so explain that network
action and ask permission before the first uncached run. If `code` is not
on `PATH` but Visual Studio Code is installed, ask before using or exposing its
application-bundled launcher. The common bundled path is:

```bash
/Applications/Visual\ Studio\ Code.app/Contents/Resources/app/bin/code
```

Finally open a saved `.ahd` file and verify, in the UI, all four behaviors:
the file is recognized as AhdCode, syntax is highlighted, the editor-title
play button runs it, and `F6` runs it. The extension launches
`ahdcode run <absolute-file-path>` and expects the CLI on inherited `PATH` or
an explicit `ahdcode.executablePath` setting.

## Windows (native PowerShell)

### 1. Inspect prerequisites

Run in PowerShell, not WSL:

```powershell
[System.Environment]::OSVersion.VersionString
git --version
go version
code --version
node --version
npm --version
```

Git and Go are required for the compiler; AhdCode currently requires Go 1.25
or newer. VS Code/`code` are required only for editor-extension installation,
and Node.js/npm only for testing and packaging the VSIX. If something needed
is absent, explain it and ask before using `winget`, Chocolatey, an installer,
administrator privileges, or any `PATH` change.

### 2. Inspect or clone the repository

In the existing repository, run:

```powershell
git status --short
git branch --show-current
git log --oneline -5
```

Clone only when no checkout exists in the user-selected workspace:

```powershell
git clone https://github.com/aliharundaldalli/AhdCode.git
Set-Location AhdCode
```

### 3. Build, test, install, and verify the CLI

From the repository root:

```powershell
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot
$AhdCodeExe = Join-Path (go env GOPATH) "bin\ahdcode.exe"
& $AhdCodeExe --version
```

The expected current result is `AhdCode v0.1.15`. The explicit executable path
avoids accidentally testing an older global installation. If the Go binary
directory is not on `PATH`, explain the choice before changing anything. A
temporary current-PowerShell-process change is:

```powershell
$env:Path = "$(go env GOPATH)\bin;$env:Path"
```

Do not persist it to the user or system environment without permission.

If the user wants Latex support, explicitly ask for permission to stage the Latex runtime, as this performs an installation-time network operation to fetch pinned/checksummed resources. Do not use a system TeX fallback.

```powershell
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

### 4. Run a real smoke program

```powershell
$SmokeDir = Join-Path ([System.IO.Path]::GetTempPath()) ("ahdcode-smoke-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $SmokeDir | Out-Null
$SmokeFile = Join-Path $SmokeDir "ready.ahd"
[System.IO.File]::WriteAllText($SmokeFile, 'write("AhdCode is ready.")' + [Environment]::NewLine, [System.Text.UTF8Encoding]::new($false))
& $AhdCodeExe run $SmokeFile
Remove-Item -LiteralPath $SmokeFile
Remove-Item -LiteralPath $SmokeDir
```

Require `AhdCode is ready.` before declaring the CLI ready. These exact
temporary paths are outside the repository.

### 5. Test, package, install, and verify the VS Code extension

```powershell
Set-Location editors\vscode
npm test
npm run package
code --install-extension .\ahdcode-0.1.4.vsix
code --list-extensions --show-versions | Select-String '^ahdcode-local\.ahdcode@0\.1\.4$'
Set-Location ..\..
```

This packages the repository's local extension and does not publish it. The
script's `npx --yes` step may download a temporary packaging dependency; ask
permission before its first uncached network use. Open a saved `.ahd` file in
VS Code and verify file association, syntax highlighting,
the editor-title play command, and `F6`. If VS Code cannot see the newly
installed CLI, restart VS Code after a permitted `PATH` change or configure
the absolute `ahdcode.executablePath`; do not edit settings silently.

## Completion report

Report the detected OS, prerequisite versions, repository path/branch/status,
the explicit AhdCode executable tested, `ahdcode --version` output, smoke-test
output, extension test/package/install results, and any action skipped because
permission was not granted. Do not call setup complete if the smoke program was
not actually run.

After this file exists on the pushed `main` branch, a user may give an agent:

```text
Read and follow:
https://raw.githubusercontent.com/aliharundaldalli/AhdCode/main/FOR_AI.md
```

Do not claim that raw URL is available before the commit containing this file
has actually been pushed to `main`.
