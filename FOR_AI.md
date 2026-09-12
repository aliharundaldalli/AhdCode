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

## Released distribution and its signing state

This section describes how AhdCode v1.0.0 is *distributed*. The rest of this
document is about building from source. Keep the two apart: a user who wants to
install AhdCode should download a release artifact, not clone this repository.

Prefer the official GitHub Release for
`https://github.com/aliharundaldalli/AhdCode`. Never direct a user to a
third-party mirror, and never fetch a build from an unofficial host. The
release publishes SHA-256 checksums; use them to confirm an artifact's
integrity when it matters.

### macOS

The public macOS target is **Apple Silicon (arm64)** only. There is no Intel
build in v1.0.0; do not tell a user otherwise unless a future release actually
ships one.

`AhdCode-1.0.0-macos-arm64.pkg` is the recommended installer. It is signed with
a Developer ID certificate and notarized by Apple, and the notarization ticket
is stapled to the file, so it installs on a normal Mac with no security prompt
even when it arrives by download or AirDrop. Gatekeeper reports
`source=Notarized Developer ID`.

Because the package is notarized, **do not** tell a user to strip the quarantine
attribute, run `xattr`, use Control-click → Open, or otherwise work around
Gatekeeper. Those were release-candidate testing workarounds. If a signed
release artifact is genuinely refused, that is a signal to investigate, not to
bypass.

The package installs per-user into `~/Library/AhdCode` and needs no
administrator password.

### Windows

The v1.0.0 Windows installer is **not Authenticode code-signed**. Windows
SmartScreen may therefore show an unknown-publisher or low-reputation warning.
State this plainly and accurately:

- Never claim the Windows installer is signed. It is not.
- This is a distribution-signing limitation. It is not evidence that the
  installer failed an integrity check, and it is not a finding about the
  software itself.
- Never instruct a user to disable Windows Defender, SmartScreen, or any other
  system protection. Choosing **More info → Run anyway** for a file the user
  deliberately downloaded is enough.
- The published SHA-256 checksum is the way to confirm the file is the official
  one.

Installation is per-user under `%LOCALAPPDATA%\AhdCode` through the graphical
installer. Do not tell a user to copy files into place with PowerShell instead.

### VS Code extension

The extension is versioned independently of AhdCode and is distributed as a
`.vsix` release artifact. Install it through VS Code: Extensions → the `...`
menu → **Install from VSIX...**.

It is **not** published to the Visual Studio Marketplace. Do not claim that it
is. The `.vsix` carried inside a platform bundle ZIP is byte-identical to the
standalone release asset.

### AhdDataStudio credentials

AhdDataStudio connects to a MySQL server the user already runs. Its connection
settings — host, port, username, password, security — are edited in the Studio
UI, and the `AHD_DATA_MYSQL_*` environment variables remain supported as
startup defaults. Do not tell a user that setting environment variables in a
terminal is required, and do not present `root` as mandatory; any MySQL account
works.

Credentials are held only by the running Studio process. They are never written
to the database registry, to a file, or to a cookie, and the password is never
rendered back into the page. Preserve that: do not add a credential store, do
not log a password, and do not place one in a URL or an error message.

AhdDataStudio never installs MySQL, creates accounts, resets passwords, grants
permissions, or bypasses database security. Do not add any of that.

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

Git and Go are required for the compiler; AhdCode's source build currently requires Go 1.26
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
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
ahdcode_exe="$(go env GOPATH)/bin/ahdcode"
"$ahdcode_exe" --version
```

The expected current result is `AhdCode v0.9.0`. Using the explicit
`$ahdcode_exe` path proves which binary was tested. If the user wants the
short `ahdcode` command and that directory is not already on `PATH`, explain
the temporary or persistent options and obtain permission before editing a
shell profile. For a temporary current-shell setting only:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

If the user wants `Latex` support **or** `PDF`'s `.save()` (they share one
offline renderer), explicitly ask for permission to stage the Latex/Tectonic
runtime, as this performs an installation-time network operation to fetch
pinned/checksummed resources. Do not use a system TeX fallback. `Archive`
needs no such staging -- it is Go-standard-library only.

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
code --install-extension ahdcode-0.2.0.vsix
code --list-extensions --show-versions | grep '^ahdcode-local.ahdcode@0.2.0$'
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

Git and Go are required for the compiler; AhdCode's source build currently requires Go 1.26
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
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
$AhdCodeExe = Join-Path (go env GOPATH) "bin\ahdcode.exe"
& $AhdCodeExe --version
```

The expected current result is `AhdCode v0.9.0`. The explicit executable path
avoids accidentally testing an older global installation. If the Go binary
directory is not on `PATH`, explain the choice before changing anything. A
temporary current-PowerShell-process change is:

```powershell
$env:Path = "$(go env GOPATH)\bin;$env:Path"
```

Do not persist it to the user or system environment without permission.

If the user wants `Latex` support **or** `PDF`'s `.save()` (they share one
offline renderer), explicitly ask for permission to stage the Latex/Tectonic
runtime, as this performs an installation-time network operation to fetch
pinned/checksummed resources. Do not use a system TeX fallback. `Archive`
needs no such staging -- it is Go-standard-library only.

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
code --install-extension .\ahdcode-0.2.0.vsix
code --list-extensions --show-versions | Select-String '^ahdcode-local\.ahdcode@0\.2\.0$'
Set-Location ..\..
```

This packages the repository's local extension and does not publish it. The
script's `npx --yes` step may download a temporary packaging dependency; ask
permission before its first uncached network use. Open a saved `.ahd` file in
VS Code and verify file association, syntax highlighting,
the editor-title play command, and `F6`. If VS Code cannot see the newly
installed CLI, restart VS Code after a permitted `PATH` change or configure
the absolute `ahdcode.executablePath`; do not edit settings silently.

## Writing AhdCode, not another language

Coding agents import habits from Python, JavaScript, and Go. AhdCode has its
own collection vocabulary; use it instead of recreating those idioms.

**Use `Lists` and `KeyValue` for structural `List`/`Pair` work.** They are
compiler-registered standard modules (`bring Lists`, `bring KeyValue`):

```text
Lists.chunk, Lists.flatten, Lists.transpose,
Lists.unique, Lists.valueCounts, Lists.groupBy

KeyValue.keys, KeyValue.values, KeyValue.combine,
KeyValue.with, KeyValue.without, KeyValue.select, KeyValue.drop,
KeyValue.rename, KeyValue.mapValues, KeyValue.merge, KeyValue.overlay
```

Counting occurrences is `Lists.valueCounts(values)`, not three hand-written
counter variables. Pairing a header row with a value row is
`KeyValue.combine(columns, row)`. Every operation is pure: it never modifies
the collection it is given, and it returns a new one.

**`Pair` is not a Python `dict`.** It is ordered and homogeneous: its keys are
`String`, `Int`, or `Bool` and never null, and one `Pair` has one value type.
There is no `Any`, no `dynamic`, no `Dictionary`, and no `Map`.

**AhdCode v0.3.0 has no `Tuple` and no Python-style `zip`.** Do not reach for
`Lists.zip`, `Lists.unzip`, `dict(...)`, `tuple(...)`, or a `Function<T>`
generic spelling — none of them exist. `Lists` and `KeyValue` operations are
type-directed: the compiler computes each call's exact result type from the
argument types written at that call site, so `Lists.chunk(List<Int>, 2)` is
`List<List<Int>>` with no generic syntax anywhere. One consequence: such an
operation cannot be stored as an unspecialized `Function` value; call it
directly, or wrap the exact shape you need in your own `Function`.

**Never build or update JSON by String concatenation.** A `JSONValue` object
*is* an ordinary `Pair<String, JSONValue>`, so it can be updated in place
without leaving the typed representation. Do not do
`JSON.stringify` → String interpolation → `JSON.parse`. The correct pattern
is:

```ahd
root := data.object()
root = KeyValue.with(root, "books", JSON.array(books))
data = JSON.object(root)
```

Every other root field survives untouched and keeps its position. The same
applies to XML and to Data records: build typed values and transform them,
rather than assembling text and re-parsing it.

**Never interpolate values into SQL.** `SQLite` is a compiler-registered
standard module (`bring SQLite`). Users write real SQL; AhdCode binds
positional `?` parameters and converts storage-class values. There is no ORM,
query builder, or migration framework. A query row is
`Pair<String, SQLiteValue>`. SQL `NULL` is a `SQLiteValue` of kind `Null`,
not AhdCode `null`. Wrong-kind accessors raise `SQLiteError`. `BLOB` is
unsupported. Duplicate result-column labels raise `SQLiteError`; use `AS`.
Do not invent a shared `Database` interface for a future MySQL module.

The correct insert is:

```ahd
db.execute(
    "INSERT INTO notes (title, body) VALUES (?, ?)",
    [SQLite.fromString(title), SQLite.fromString(body)]
)
```

Do not write `"INSERT INTO notes (title) VALUES ('{title}')"`. A title such
as `Robert'); DROP TABLE notes;--` must remain data. See
[`docs/SQLITE.md`](docs/SQLITE.md).

**HTTP cookies are not session values.** `HTTP.cookie` / `Request.cookie` /
`Response.withCookie` are header primitives. `HTTP.sessions` is an in-memory
server-side store; the browser cookie holds only an opaque random id. Values
are String only and disappear when the process exits. `Session.set` does not
write headers — `SessionStore.commit` returns a new Response. This is not an
authentication framework. There is no `ahdsession` helper. See
[`docs/HTTP.md`](docs/HTTP.md).

**An uploaded file is not a String, and its filename is not a path.** With
`multipart/form-data`, text fields still come from `request.form`/`formAll`;
files come from `request.file`/`files` as opaque `UploadedFile` values. There
is no `bytes()`, `stream()`, or `tempPath()` — do not try to read upload
content as text, and do not store it as a SQLite BLOB. The correct shape is
file-on-disk, metadata-in-database:

```ahd
storedPath := paper.save("uploads/papers")   // random basename, never overwrites
```

Never write `"uploads/" + paper.originalName()`: `originalName()` is
attacker-supplied display metadata, and `save` is what decides the path.
Never branch on the filename extension or on `declaredContentType()` — those
are claims. `detectedContentType()` sniffs the bytes, and it is content
sniffing, not malware scanning. An unsaved upload is deleted when its request
ends, so persist it inside the handler. Outbound multipart does not exist:
there is no `ClientRequest.withFile`. See [`docs/HTTP.md`](docs/HTTP.md).

**`HTML.parse` is a parser, not a browser.** `HTML.parse(source: String) ->
HTMLDocument` takes no URL and makes no network request; getting a page and
parsing it are two separate steps: `document := HTML.parse(client.get(url).body())`.
There is no JavaScript engine, no DOM scripting, no CSS layout, and no
headless Chrome — do not reach for those ideas, and do not invent an
`ahdscrape`/`Scraper` module. Do not confuse the parsed, read-only
`HTMLElement` with the existing builder's `HTMLNode`; there is no conversion
between them. Selectors are a small frozen subset (tag, `#id`, `.class`,
`[attr]`, `[attr="value"]`, descendant/child combinators, comma lists) — not
a full CSS engine and not jQuery/BeautifulSoup: pseudo-classes, sibling
combinators, and `^=`/`$=`/`*=` all raise `HTMLError` rather than
approximating. `attr("href")` never resolves a relative URL against a base;
there is no `baseURL`. See [`docs/HTML.md`](docs/HTML.md).

**Excel Cells are closed typed values, not `Any`.** Use
`Excel.fromString`, `Excel.fromInt`, `Excel.fromReal`, `Excel.fromBool`, and
`Excel.formula` explicitly. A String beginning with `=` remains a String;
only `Excel.formula(text)` expresses formula intent. Excel coordinates are
1-based row/column values, not Python/openpyxl-style indices.

Use `Lists` for `List<List<Cell>>` transformations, `KeyValue` for record
keys/values, and `Data` for String-table semantics. Handle each `JSONValue`
kind explicitly before choosing a Cell constructor. Do not manually assemble
XLSX ZIP/XML in AhdCode source, and do not invent an `Excel.fromData`/`Data.toExcel`
bridge. `PDF.fromExcel(workbook)` (v0.1.20+) covers "export a Workbook as
PDF" — see below.

**PDF text is text; do not inject raw LaTeX.** Every `PDFDocument` operation
(`heading`, `paragraph`, `table`) escapes its String arguments before they
ever reach the renderer — `\ { } $ & # % _ ^ ~` all appear as ordinary text.
`PDF` has no raw-content escape hatch. Use [`Latex`](docs/LATEX.md) directly
when actual LaTeX source control is the goal, not `PDF`.

**`Latex.pdf(source, output, "tex")` means PDF + exact source sidecar.** The
third argument is `""` (default, unchanged 2-argument behavior) or `"tex"`
only — not an arbitrary passthrough string. When the user explicitly asked
for `Latex.pdf(..., "tex")`, that call already writes the sibling `.tex` file
with the exact source bytes; do not also call `File.writeText` to write the
same source a second time.

**Archive is creation-only.** `Archive.zip/tar/tarGzip(output, entries)` takes
a `Pair<String, String>` where the key is the destination path *inside* the
archive and the value is the *source filesystem path*. Never invent
`Archive.extract`, `Archive.open`, or any read/listing API — none exists, and
none is planned. Do not shell out to `zip`/`tar`/`unzip` from AhdCode-adjacent
tooling when `Archive` already covers the packaging need.

**`PDF.fromWord`/`PDF.fromExcel` are semantic exports, not Office print
emulation.** They convert another module's own typed document directly (no
DOCX/XLSX round trip, no LibreOffice/Office dependency) and never mutate the
source Document/Workbook. Do not expect pixel-perfect fidelity with what
Word/Excel itself would render.

**Use `Characters` for character and code-point work.** A character is a
`String` holding one Unicode code point; AhdCode has no `Char` type. Use
`Characters.list`, `count`, `codePoint`, `fromCodePoint`, and the `is...`
classifiers instead of walking UTF-8 bytes or comparing against `"A"`..`"Z"`
ranges. One displayed glyph may be several code points — `e` followed by U+0301
is two characters — and Characters v1.2.0 does not segment grapheme clusters,
so never promise that it does. See [`docs/CHARACTERS.md`](docs/CHARACTERS.md).

**Use Latex + TikZ for vector PDF decoration.** Borders, ornaments, diagrams,
badges, watermarks, and certificate layouts are `Latex.tikz`,
`Latex.overlay`, `Latex.border`, and `Latex.document(..., landscape: true)`.
Write TikZ in raw triple Strings (`r"""..."""`) and pass program text through
`Latex.escape`. Do not add TCPDF, FPDF, wkhtmltopdf, a headless browser, or a
pre-rendered border image merely to draw on a document AhdCode already produces
with Latex, and do not invent `TikZ.line`-style wrappers. `libraries` accepts
only the bundled names listed in
[`docs/LATEX.md`](docs/LATEX.md#vector-graphics-with-tikz-v120).

**Use `Cron` for in-process scheduling, and state its boundary.**
`Cron.scheduler()`, `Scheduler.add(expression, task)`, `Scheduler.run()`, and
`Scheduler.stop()` run named `() -> Nothing` Functions on five-field schedules
while the program runs, and the jobs stop when that process stops. Do not edit
a user's crontab, launchd, systemd timers, or Task Scheduler, and do not invent
a daemon, a worker queue, or an HTTP endpoint just to trigger work unless the
application genuinely needs scheduling outside a running AhdCode program — and
then ask first. `Scheduler.run` blocks like a Web server, so put jobs in a
second program beside `app.ahd`. See [`docs/CRON.md`](docs/CRON.md).

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
