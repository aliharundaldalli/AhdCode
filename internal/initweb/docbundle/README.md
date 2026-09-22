<p align="center">
  
</p>

# AhdCode

[![CI](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml/badge.svg)](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml)

[English](README.md) · Türkçe

AhdCode is an independently developed, statically checked, general-purpose
programming language focused on readable syntax, explicit intent,
predictable semantics, and native compilation. It comes with its own
toolchain, standard library, language server, Web framework, database
modules, GUI, Graphics, interactive Plot viewer, and desktop application
packaging. It is used in practice by a small community; it is not a
mainstream language.

This is **v2.4.0**, **Mathematical Rendering & Plot Styling**. It adds
bundled offline Tectonic labels with exact embedded TeX glyph rendering,
typed line and marker styling, configurable legend positions and improved
math layout, current-view Surface Save, and automatic Windows user-PATH
registration. v2.3.0, **Language Ergonomics & Plot Polish**, remains available
as the previous release.

v2.0.0 is a major release, **Desktop Application Completion**: it completes
the first-party desktop application foundation — ListBox, Select, TextArea,
PasswordInput, TableView, dialogs, change callbacks, and resizable windows in
GUI; a toolbar and 3D Surface plotting in the Plot viewer; and
`ahdcode package` for self-contained desktop applications — all without new
syntax or a type-system change. See [What is new in v2.0.0](#what-is-new-in-v200).

v2.1.0 adds binary-safe outbound downloads, outbound multipart uploads, SMTP
attachments, a WebSocket client, and fixes the v2.0 Plot Save runtime
discovery regression. See [What is new in v2.1.0](#what-is-new-in-v210).

It ships as a self-contained platform package: the `ahdcode` CLI, a private
Go 1.27.0 toolchain, AhdDataStudio, the `ahdsqlite`, `ahdnumeric`, `ahdplot`,
`ahdgraphics`, and `ahdgui` helpers, the interactive Plot viewer `ahdplotview`, an offline Tectonic LaTeX engine with its pinned resource
bundle, the Web starters, and an exact-version English documentation bundle.
See [Installation](INSTALLATION.md).

The 1.0 Web surface, introduced by v0.20.0 as **Web Assets, Resource
Boundaries & Application Patterns**, keeps components as ordinary functions that
return HTML. Layouts declare CSS and JavaScript with `Web.Assets`;
`managedAssets` serves only those declared files. `Identity.id()` mints
public identifiers. Web applications apply explicit body, upload, and
timeout limits. `ahdcode init web` now offers Empty, Basic, Admin, MVC, and
CRUD starters, and every project receives an exact-version English
documentation bundle.

A first TTY `ahdcode dev` asks once before enabling local `.test` names.
After that approval, new project hostnames are maintained automatically.
`ahdcode local hosts apply` remains a recovery command, not a required step
on the happy path. `ahdcode databases` starts the exact-version AhdDataStudio
bundled in the installed CLI; it does not need the AhdCode repository or
`AHDCODE_ROOT`. `AHDCODE_ROOT` is a developer override only.

Local development is still HTTP: there is no local TLS, certificate
authority, or ACME. No language syntax changed in this release. See
[CLI](CLI.md#local-development-test-names-and-the-router) and
[Web](WEB.md).

v0.18.5, **Web Starter & Application Bootstrap**, turns `ahdcode init web`
into a starter wizard: Empty, Basic, or Admin. Empty is a polished welcome
application. Basic adds common application and mail configuration. Admin
adds login, a dashboard, and SQLite or MySQL bootstrap with one
administrator. v0.17 route, guard, form, CSRF, and flash APIs are unchanged.
See [Web](WEB.md#18-v018-web-starters-and-application-bootstrap).

v0.17.0, **Web Init, Context Routes, Groups & Guards**, keeps `ahdcode init web`
and adds explicit context-aware route registration, route groups, and
ordered policy-agnostic guards. `context.respond` stays the only
finalizer. There is no general middleware chain, auth framework, or ORM.
See [Web](WEB.md#17-v017-context-routes-groups-and-guards) and
`examples/v0.17/routes_guards`.

v0.16.0, **Request Context, Forms, Validation, CSRF & Flash**, adds an explicit
request/session context (`RequestContext`), one-time response and session
finalization (`context.respond`), typed form access (`Forms`), ordered
validation errors (`ValidationErrors`), selected safe old input (`OldInput`),
session-bound CSRF (`Web.UI.csrfField`), and consumed flash messages
(`context.flashSet`, `context.flashTake`). The implementation is bundled AhdCode;
existing HTTP/Session/Web.UI APIs and v0.15.1 clean routes remain compatible.
See the [workflow and exact API](WEB.md#16-v016-request-context-forms-validation-csrf-and-flash)
and the runnable example.

v0.15.1 adds one-segment HTTP trailing path wildcards (`/*`), enabling clean
path parameter routing without URL query strings.

v0.15.0, **Web Foundations**, adds [`Web`](WEB.md): a first-party web
framework written mostly in AhdCode itself and bundled with the compiler, so
`bring Web` resolves offline with no package manager, registry, manifest, or
lockfile, and a built executable keeps no runtime dependency on framework
source. It composes the existing HTTP and HTML primitives rather than
replacing them and re-exports their types unchanged, so a `Request` reached
through `Web` *is* HTTP's `Request`.
[`Web.UI`](WEB.md#9-webui) is a semantic HTML component layer — `section`,
`h1`, `p`, `a`, `img`, `table`, `form` and the rest — where every text entry
point escapes and there is no raw-markup helper anywhere.
Pages, Layouts, and Components are ordinary Functions returning `HTMLNode`:
no virtual DOM, no hydration, no template language, and no JavaScript runtime.
A frozen environment contract (`APP_NAME`, `APP_ENV`, `APP_HOST`,
`APP_PROTOCOL`, `SERVER_HOST`, `SERVER_PORT`) keeps the public URL and the
bind address separate for reverse-proxy deployments, with no silent defaults
and no `.env` values ever embedded in a binary. Still deliberately out of
scope: ORM, middleware, auth, bundler, and browser live reload. Local trusted
HTTPS for `<APP_HOST>.test` is
[deferred](WEB.md#14-local-https--current-limitation) rather than
approximated: it would need permanently privileged system state.

v0.14.1 is a tooling hotfix for `require(...)` (language server, formatter,
and editor highlighting). Language semantics are unchanged from v0.14.0.

v0.14.0, **Application Foundations**, adds the remaining framework-independent
groundwork for larger server-side AhdCode applications: compile-time local
source composition with [`require(...)`](REQUIRE.md), so a program can
be split across files without a package manager; `ahdcode dev` (v0.13) now
watches the entry file plus the whole resolved `require(...)` graph, not just
the entry, so editing any required file rebuilds and restarts automatically;
and [`server.static`](HTTP.md#static-files) serves local static
assets (CSS, JS, SVG, images, fonts) from one explicit filesystem root with
path-traversal, symlink-escape, and dotfile protection built in, so every
application does not reimplement that itself. This is deliberately not a Web
framework release: no templating language, no forms/middleware/router
framework, no package manager, and no browser live reload.
AhdDataStudio is restructured onto these
foundations as its own dogfood, split from one file into files grouped by
responsibility, with no behavior change. v0.13.0 adds `ahdcode dev`: a
MAMP/Vite-style foreground watch-rebuild-restart loop built as orchestration
around the same build pipeline `ahdcode build`/`run` already use, plus
`ahdcode stop`, the graceful counterpart to the existing (forced-by-default)
`kill` — `stop` waits to confirm the process actually exited instead of
just signaling it. A failed rebuild always leaves the previously working
process running untouched, including the very first build; a runtime crash
after a successful build is reported without retrying the same binary.
v0.12.0 adds AhdDataStudio: a first-party
localhost MySQL + SQLite development application written in AhdCode, not a
compiler builtin. Start it with `ahdcode databases` and open
[http://ahddatabasestudio.test/](http://ahddatabasestudio.test/) (or
[http://127.0.0.1:8081/AhdDataStudio](http://127.0.0.1:8081/AhdDataStudio)).
It binds `127.0.0.1` only, discovers MySQL schemas with `database: null`,
scopes SQLite files to configured project paths, and uses CSRF-protected
POST forms for generated CRUD. This release also fixes a parser hang on
malformed nested String literals. MySQL from v0.11 remains offline-buildable
through the bundled vendored driver. v0.11.0 adds [MySQL](MYSQL.md): `MySQL.connect` dials a real server and
verifies it is reachable before returning, `database` may be `null` so a
connection can list every database the credentials can see with `SHOW
DATABASES` before selecting one, every query is server-side parameter-bound,
an independent `MySQLTransaction` never shares mutable state with concurrent
requests, and `DECIMAL`/binary values stay exact rather than being coerced.
The vendored `github.com/go-sql-driver/mysql` driver is embedded in AhdCode
itself and copied into a generated program's build as `vendor/`, so a
MySQL-using program still builds fully offline, the same guarantee every
other generated program already has. v0.10.0 adds [Security](SECURITY.md): `Security.passwordHash` /
`passwordVerify` wrap Argon2id password hashing behind one self-describing
stored string, `Security.token` returns a 256-bit URL-safe random token, and
`Security.secureEqual` compares two Strings in constant time for CSRF tokens
and the like — three focused primitives, not an authentication framework.
v0.9.1 adds binary-safe [HTTP](HTTP.md) file responses: `HTTP.file` and
`HTTP.download` stream a stored file's exact bytes back to the client
without ever passing them through an AhdCode `String`, with an explicit
`contentType` and, for `download`, a presentation filename that is
independent of the stored path and safely encoded even for non-ASCII names.
v0.9.0 adds send-only [SMTP](SMTP.md) mail: an immutable `SMTPClient`
configures host, port, and security (`starttls`, `tls`, or explicit `none`),
an immutable `SMTPMessage` carries To/Cc/Bcc, Reply-To, UTF-8 Subject, and
text and/or HTML bodies, and `send` opens one SMTP connection per message
with AUTH PLAIN only after TLS. There is no IMAP, no attachments, no mail
queue, and no provider shortcut. v0.8.0 adds multipart form handling and safe file uploads: a handler reads
uploaded files with `Request.file` / `Request.files`, inspects
`originalName`, `size`, and both the declared and the content-sniffed MIME
type, and persists one with `UploadedFile.save(directory)` under a
crypto-random name the uploader cannot influence — so a hostile filename can
neither escape the directory nor overwrite an existing file. Uploaded bytes
are never an AhdCode `String` and never a database BLOB: applications store
the file on disk and keep only its path and metadata in
[SQLite](SQLITE.md). It also adds `ahdcode kill app.run`, which stops an
application started with `ahdcode run` without looking up ports and pids by
hand. v0.7.0 adds HTML parsing and a small CSS-like selector language on top of the
existing [HTML](HTML.md) builder: `HTML.parse(source)` turns an HTML
String -- typically an `HTTP` Client response body -- into a read-only
`HTMLDocument`, and `select`/`first` find `HTMLElement` values in it by tag,
id, class, attribute, and descendant/child combinators. Parsing never fetches
a network resource and never executes script content; it only tokenizes and
builds a tree. v0.6.0 adds an outbound [HTTP](HTTP.md) Client with
HTTPS, timeouts, and explicit JSON/Env API interoperability. v0.5.0 added
HTTP cookies and
in-memory server-side sessions on top of the v0.4.0 web foundation: a typed
HTTP server, Request/Response values, and a small safe [HTML](HTML.md)
builder, so an AhdCode program can be opened in a browser on this machine. A
session cookie holds only an opaque random identifier; session values stay on
the server and disappear when the process exits. This is not an authentication
framework. v0.3.0 began practical application development with a typed
[SQLite](SQLITE.md) bridge. HTTP uses Go's `net/http` inside the runtime;
there is no companion HTTP, cookie, session, or client helper. Inbound
multipart uploads arrived in v0.8.0 and WebSocket server endpoints in v1.4.0;
outbound file attachments, a WebSocket client, and an AI vendor module are
still not part of the release.

v0.2.2 completed the practical everyday AhdCode language server on top of
v0.2.1's diagnostics, hover, completion, go to definition, document symbols,
signature help, and find references. v0.2.2 adds rename, semantic highlighting,
inlay hints, code actions/quick fixes, auto import, document formatting,
workspace symbol search, folding ranges, and selection ranges — all backed
directly by the real compiler frontend on unsaved editor buffers with full
document synchronization. See [`docs/LSP.md`](LSP.md) for scope and
honest limitations (compile-graph-scoped references/rename, on-demand module
discovery, no persistent workspace index). The bundled
VS Code extension launches the same server. Language
semantics are unchanged from v0.1.20, which added the [PDF](PDF.md) and
[Archive](ARCHIVE.md) modules and a `Latex.pdf` source sidecar.

```ahd
greet: Function := (
    name: String
) -> String {
    return "Hello {name}"
}

names: List<String> := ["Ali", "Ayşe"]

for name in names {
    write(greet(name))
}
```

## Why AhdCode?

- Declaration and mutation look different: `:=` declares, `=` mutates.
- Static checking rejects unrelated implicit conversions and truthiness.
- Explicit nullable types (`T?`) compose with collections, while flow-sensitive
  checks narrow proven non-null values.
- Lists, Pairs, Classes, Functions, modules, errors, and native executables are
  part of the v0.1 core.
- Expression-only `lambda (<typed parameters>) -> <expression>` creates a
  value of the existing `Function` type; it is not a separate callable type.
- A small, closed set of [Class Protocol Methods](PROTOCOLS.md) lets a
  Class define `==`, ordering, arithmetic, unary `-`, and `str()` behavior.
- A [Regex module](REGEX.md) compiles patterns to a `Pattern` value with
  `matches`, `find`, `findAll`, `groups`, `replace`, and `split`.
- [Time](TIME.md) supports local, UTC, fixed-minute-offset, and Unix
  millisecond representations without introducing a timezone database.
- The strict [CSV module](CSV.md) transports raw String rows or
  header-keyed String records with native and persistent-REPL parity.
- The [Data module](DATA.md) adds an immutable `Table` of String cells for
  filtering, sorting, grouping, and deriving columns; it infers no types, so
  numeric work stays an explicit `int(...)` / `real(...)` conversion.
- An expression lambda may read outside values through an explicit dependency
  list: `#name`/`Local name` for a lexical capture, `@name`/`Global name` for a
  module binding, as in
  `lambda [#minimum, @Maximum] (score: Int) -> score >= minimum and score <= Maximum`;
  neither kind is ever inferred or implicit.
- The [Statistics module](STATISTICS.md) provides typed descriptive
  statistics over `List<Int>` and `List<Real>`, with no String coercion.
- The [Numeric module](NUMERIC.md) adds immutable Real-oriented vectors,
  matrices, linear algebra, and additive `Vector` overloads in Plot.
- The [Word module](WORD.md) builds immutable formatted documents, merged
  tables, embedded Plot images, and bounded semantic DOCX read-back without
  requiring Office or an external runtime.
- The [Excel module](EXCEL.md) reads and writes real `.xlsx` packages
  through typed immutable Workbook/Sheet/Cell/Range values. Formula intent is
  explicit, merges reject value loss, and native executables remain offline
  and relocation-safe.
- The [PDF module](PDF.md) builds immutable `PDFDocument` values and
  renders them offline to real `.pdf` files through the same staged Tectonic
  renderer `Latex` uses, plus semantic `PDF.fromWord`/`PDF.fromExcel`
  conversion of another module's own typed document.
- The [Archive module](ARCHIVE.md) packages files into real ZIP, TAR,
  and TAR.GZ archives offline, creation-only, using nothing beyond the Go
  standard library.
- [Lists](LISTS.md) and [KeyValue](KEYVALUE.md) add pure structural
  transformations of `List` and `Pair` — `chunk`, `flatten`, `transpose`,
  `unique`, `valueCounts`, `groupBy`, and `keys`, `values`, `combine`, `with`,
  `select`, `drop`, `rename`, `mapValues`, `merge`, `overlay`. They are
  type-directed: each call's exact result type is computed from its argument
  types, with no generic syntax and nothing erased.
- The formatter defines one canonical presentation while preserving comments.
- The [language server](LSP.md) (`ahdcode lsp`) exposes the compiler's
  own diagnostics, hover, go to definition, document symbols, signature
  help, find references, and completion over standard stdio LSP -- no
  second parser, no hand-maintained symbol catalog, and no writes to a
  document's file while it's open and unsaved in an editor.

## Design Principles: Stable Principles, Evolving Pre-1.0 Surface

AhdCode is grounded in enduring design principles:

- **Readability over minimum line count:** syntax favors clarity and structure over clever compactness.
- **Explicit intent:** plain English keywords; declaration (`:=`) and mutation (`=`) are visibly distinct.
- **Strict static typing:** no `Any`/dynamic fallback; no unrelated silent coercion; no truthiness.
- **Safe and unique inference only:** omitted type annotations are inferred only when unambiguous; the compiler never guesses.
- **Deterministic behavior:** no hidden mutable runtime state, no magical global side effects.
- **Ordinary Functions and libraries before new syntax where practical.**
- **Canonical formatting:** one single authoritative presentation style enforced by `ahdcode format`.
- **Diagnostics as product behavior:** precise, construct-aware errors with actionable hints.

### Language Evolution Before 1.0

AhdCode never treated the pre-1.0 period as permanently feature-frozen, nor did it casually churn syntax. The core principles above remain constant. Where real implementation, dogfooding, and practical application needs demonstrated concrete gaps, pre-1.0 language decisions were revised deliberately. Capabilities such as declaration type inference, explicit nullable types (`T?`), expression-only lambdas with explicit lexical/global dependency lists (`#name`, `@name`), and the closed set of Class Protocol Methods reflect deliberate evolutions that strictly preserve static typing, determinism, explicitness, and the rejection of hidden magic.

## Architecture Taxonomy

To maintain conceptual clarity, AhdCode's capabilities are organized into four distinct architectural layers:

1. **Core Language:**
   - Explicit declarations (`:=`) and mutation (`=`), explicit nested scope (`Local`, `Global`)
   - Static type system and nullability: non-nullable `T`, explicit nullable `T?`, `Nothing`, and flow-sensitive null narrowing
   - Named Functions and expression-only lambdas (`lambda (...) -> expr`) with explicit captures (`#name`, `@name`)
   - Classes, single inheritance, and the fixed set of ten [Class Protocol Methods](PROTOCOLS.md) (`CEqual`, `CCompare`, `CAdd`, `CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower`, `CNegate`, `CStr`)
   - Deterministic control flow (`if`/`else`, `for`/`between`, `attempt`/`except`/`ultimately`/`toss`)
   - Predeclared fundamentals (`write`, `take`, `str`, `int`, `real`, `len`, `clear`, `abs`, `sum`, `min`, `max`, `type`, `id`) and structured error taxonomy
   - Module resolution (`bring`, `from ... bring`) and compile-time local source composition ([`require(...)`](REQUIRE.md))

2. **Standard Library (First-party Bundled Modules):**
   - **Mathematics & Computation:** [`Math`](MATH.md), [`Bits`](BITS.md) (bitwise operations on `Int`), [`Regex`](REGEX.md), [`Statistics`](STATISTICS.md), [`Numeric`](NUMERIC.md), [`Plot`](PLOT.md), [`Graphics`](GRAPHICS.md) (Canvas windows and Turtle drawing), [`GUI`](GUI.md) (small desktop windows with click and key callbacks)
   - **Data & Collections:** [`Lists`](LISTS.md), [`KeyValue`](KEYVALUE.md), [`Characters`](CHARACTERS.md) (Unicode code points and classification), [`CSV`](CSV.md), [`Data`](DATA.md), [`JSON`](JSON.md), [`XML`](XML.md), [`UUID`](UUID.md) (RFC 9562 version 4 and time-ordered version 7 identifiers)
   - **Document Generation:** [`Word`](WORD.md), [`Excel`](EXCEL.md), [`PDF`](PDF.md), [`Latex`](LATEX.md), [`QR`](QR.md) (QR codes), [`Barcode`](BARCODE.md) (Code 128, EAN-13, UPC-A), [`Archive`](ARCHIVE.md)
   - **System & Environment:** [`Time`](TIME.md), [`Cron`](CRON.md) (bounded in-process scheduling), [`Path`](FILESYSTEM.md), [`File`](FILESYSTEM.md), [`Env`](ENV.md), [`Terminal`](TERMINAL.md) (v1.5.0: standard error, flushing, terminal detection and size, styled text, pretty layout)

3. **First-Party Runtime / Framework Modules:**
   - **Network, Server & Storage Primitives:** [`HTTP`](HTTP.md) (in-memory server, request/response, cookies, sessions, static file server, [WebSocket endpoints](WEBSOCKET.md), client), [`HTML`](HTML.md) (semantic builder, parser, selector engine), [`Security`](SECURITY.md) (Argon2id hashing, bcrypt compatibility, secure tokens, constant-time comparison, SHA-2 digests, HMAC, encodings, RS256 signatures, AES-256-GCM), [`SQLite`](SQLITE.md) (local typed database bridge), [`MySQL`](MYSQL.md) (network database with connection pool and transactions), [`PostgreSQL`](POSTGRESQL.md) (network database with connection pool and transactions), [`SMTP`](SMTP.md) (send-only mail client)
   - **Web Application Framework:** [`Web`](WEB.md) (first-party bundled web framework, [`Web.UI`](WEB.md#9-webui) semantic components, `RequestContext`, typed `Forms`, ordered `ValidationErrors`, selected `OldInput`, session-bound CSRF, and flash lifecycle)

4. **Developer Tools:**
   - **Compiler & Toolchain:** `ahdcode build`, `ahdcode run`, `ahdcode dev` (watch-rebuild loop with automatic `.test` identities), `ahdcode stop`, `ahdcode local` (local routes and managed hostnames)
   - **Canonical Formatter:** `ahdcode format` (syntax tree-driven, comment-preserving)
   - **Interactive REPL:** `ahdcode repl` (persistent multi-line environment)
   - **Editor & Language Server:** `ahdcode lsp`, official VS Code / Antigravity extension (`editors/vscode`)
   - **Diagnostics Engine:** construct-aware compiler diagnostics with clear error codes and hints
   - **Local Developer UI:** AhdDataStudio (localhost MySQL and SQLite management tool), `ahdcode databases list|add|remove` (per-user SQLite registry)

## Installation

Use the platform package described in [Installation](INSTALLATION.md). It includes the private Go toolchain, offline LaTeX, SQLite/numeric/plot helpers, Studio, starters, and docs.

## Build from source

AhdCode currently requires Go 1.26 or newer.

```bash
cd AhdCode
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
go -C cmd/ahdgraphics install .
go -C cmd/ahdgui install .
go -C cmd/ahdplotview install .
```

The commands above install the compiler and the local numeric, plot, SQLite,
Graphics window, and GUI window helpers, and the interactive Plot viewer. These window helpers are their own Go modules, so
they are installed with `go -C cmd/ahdgraphics install .`,
`go -C cmd/ahdgui install .`, and `go -C cmd/ahdplotview install .`.
If you plan to use the `Latex` module **or** the `PDF` module's `.save()` (they
share one offline renderer), you must also stage the offline Latex/Tectonic
runtime bundle. `Archive` needs no such staging -- it is Go-standard-library
only. Staging requires a one-time network fetch to download pinned,
checksummed resources:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

After staging, ordinary AhdCode Latex execution remains strictly offline.

Ensure Go's binary directory is on `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

## CLI quick start

```bash
mkdir my-app
cd my-app
ahdcode init web
ahdcode dev app.ahd
```

`dev` prints both the socket it bound and the local name it now routes:

```text
  Open:
  http://127.0.0.1:8080

  Local identity:
  http://ahdakademi.test/
  Bind: 127.0.0.1:8080
```

On a terminal, `ahdcode init web` asks which starter to write:

- **Empty** — welcome application, no database, no login
- **Basic** — the same shell plus common `.env` / mail configuration
- **Admin** — Home, Login, Dashboard, and one administrator on SQLite or MySQL

You can also pass the starter: `ahdcode init web empty`, `basic`, or `admin`.
Templates and Bootstrap 5.3.3 ship inside the CLI (MIT, local files, no CDN).
`.env` is gitignored; Admin SQLite `database/*.db` files are ignored too.
Existing files and existing databases are never overwritten.

```bash
ahdcode run examples/v0.1/01_hello.ahd
ahdcode dev examples/v0.1/01_hello.ahd
ahdcode build examples/v0.1/01_hello.ahd -o hello
ahdcode format examples/v0.1/01_hello.ahd
ahdcode format --check examples/v0.1/01_hello.ahd
ahdcode
```

See the [CLI guide](CLI.md), [formatter guide](FORMATTER.md),
[REPL guide](REPL.md), and [language server guide](LSP.md).

## Documentation

- Türkçe Öğrenci Rehberi
- [English Student Guide](STUDENT_GUIDE_EN.md)
- [Web — the first-party web framework](WEB.md)
- [require(...) — local source composition](REQUIRE.md)
- [Practical Module Workshops](PRACTICAL_MODULES.md) — learn CSV, Data,
  Plot, Excel, Word, Latex, HTTP(S), and HTML through end-to-end projects
- [Getting started](GETTING_STARTED.md)
- [Language tour](LANGUAGE_TOUR.md)
- [Types and null safety](TYPES_AND_NULL.md)
- [Control flow](CONTROL_FLOW.md)
- [Functions](FUNCTIONS.md)
- [Classes](CLASSES.md)
- [Class Protocol Methods](PROTOCOLS.md)
- [Collections](COLLECTIONS.md)
- [Modules](MODULES.md)
- [Errors](ERRORS.md)
- [Fundamentals](FUNDAMENTALS.md)
- [String API](STRING_API.md)
- [List API](LIST_API.md)
- [Math module](MATH.md)
- [Time module](TIME.md)
- [Latex module](LATEX.md)
- [Word module](WORD.md)
- [Excel module](EXCEL.md)
- [PDF module](PDF.md)
- [QR module](QR.md)
- [Barcode module](BARCODE.md)
- [Archive module](ARCHIVE.md)
- [File and Path modules](FILESYSTEM.md)
- [Regex module](REGEX.md)
- [CSV module](CSV.md)
- [Data module](DATA.md)
- [Statistics module](STATISTICS.md)
- [Plot module](PLOT.md)
- [Numeric module and Complex scalars](NUMERIC.md)
- [JSON module](JSON.md)
- [SQLite module](SQLITE.md)
- [PostgreSQL module](POSTGRESQL.md)
- [HTTP module](HTTP.md)
- [WebSocket endpoints](WEBSOCKET.md)
- [HTML module](HTML.md)
- [SMTP module](SMTP.md)
- [XML module](XML.md)
- [Env module](ENV.md)
- [Lists module](LISTS.md)
- [KeyValue module](KEYVALUE.md)
- [UUID module](UUID.md)
- [Terminal module](TERMINAL.md)
- [Graphics module](GRAPHICS.md)
- [GUI module](GUI.md)
- [Packaging desktop applications](PACKAGING.md)
- [Understanding diagnostics](DIAGNOSTICS.md)
- [Language server](LSP.md)
- AI-assisted local setup
- Curated v0.1 examples
- v0.3 SQLite Notes App
- v0.4 Web Notes App
- v0.5 cookies and sessions
- v0.6 HTTP Client
- v0.7 HTML parsing and web scraping
- v0.8 multipart forms and file uploads
- v0.9 SMTP mail sending
- v0.12 MySQL raffle — join codes, hashed admin login, announced winner
- v0.14 multi-file web example — require(...), dependency-aware dev, static assets
- v1.4 realtime attendance — Web, PostgreSQL, WebSocket, UUID v7, `Env.secret`, and Cron
- v1.5 Terminal demo — `Terminal.emit`, standard error, terminal detection, styled text, and pretty layout
- v1.7 standard-library completion — trigonometry and gcd/lcm, strict ISO time text and instant arithmetic, and a Table join with statistics
- v2.0 desktop applications — a SQLite ledger with a TableView, dialogs, and CSV export; a 3D Surface; and a small application to package
- v2.1 application I/O and network — a binary file round trip over HTTP, a WebSocket client and server pair, and mail with attachments
- v2.2 student performance — one desktop application that shows the same grades as a line, bar, histogram, box, error bar, pie, heatmap, and 3D surface
- v2.3 examples — explicit named-Function `uses` captures and Plot/Surface mathematical labels
- v2.4 examples — embedded Tectonic math glyphs, plot styling, legend positions, and Surface labels
- v1.9 GUI colors and interactive Plot — an order form with colors and a disabled Save button, and a chart in AhdCode's own viewer
- v1.8 GUI and events — a small GUI ledger backed by SQLite and a Turtle driven by arrow keys and clicks
- v1.6 Graphics and Turtle — a Cartesian Canvas, shapes, a Turtle star and spiral, PNG/SVG export, and a regular-polygon lesson
- AhdDataStudio — local MySQL + SQLite development UI
- [v0.4 Library Demo](https://github.com/aliharundaldalli/ahdcode-library-demo) (separate beginner web app)
- [v0.4 Seminar Demo](https://github.com/aliharundaldalli/ahdcode-seminer-demo) (Hatay, multi-page)
- [v0.16 Math Portal](https://github.com/aliharundaldalli/ahdcode-math-portal) (RequestContext, forms, validation, CSRF, flash)
- [Full v0.1 language specification](AHDCODE_LANGUAGE_SPEC_v0.1.md)

## Editor extension

The local VS Code-compatible extension in `editors/vscode`
recognizes `.ahd`, provides syntax highlighting, runs the active file from
the editor title play button, Command Palette, or `F6`, and connects to the
[language server](LSP.md) (`ahdcode lsp`) for compiler-backed
diagnostics and hover. The same VSIX targets VS Code and Antigravity. See its
installation guide.

## Current limitations

## What is new in v2.4.0 <a id="what-is-new-in-v240"></a>

v2.4.0, **Mathematical Rendering & Plot Styling**, completes the Plot and
Surface presentation work introduced in v2.3.0:

- Plot and Surface math labels use the bundled offline Tectonic engine and
  exact embedded Type-1C glyph programs, with no host-font substitution;
- typed `LineStyle`, `Marker`, `lineWidth`, `markerSize`, and `LegendPosition`
  values make plot styling explicit and keep legends aligned with series;
- legend insets and math-text baseline/layout handling keep formulas away from
  chart edges and avoid clipping across PNG, SVG, and PDF output;
- the interactive Surface viewer's Save exports the current visible view while
  programmatic `Surface.save(path)` remains canonical;
- the Windows installer adds the per-user AhdCode bin directory to PATH once
  and broadcasts the environment update.

Math mode remains whole-string `$...$`; mixed rich text is not supported.

See the v2.4 examples,
[Plot](PLOT.md), and [Installation](INSTALLATION.md) references.

## What is new in v2.3.0 <a id="what-is-new-in-v23"></a>

v2.3.0, **Language Ergonomics & Plot Polish**, is a focused language and
visualization release:

- named Functions inside executable blocks can declare explicit `uses`
  dependencies with `#` lexical captures and `@` module captures, using the
  same model as lambda captures;
- the compiler, formatter, diagnostics, and language server understand those
  captures, including completion, definition, hover, references, rename,
  semantic tokens, and missing-capture quick fixes;
- references and rename can inspect bounded compiler-authoritative workspace
  snapshots, including imported files, without a background index;
- the Plot Surface viewer's Save records the current visible camera view,
  while programmatic `Surface.save(path)` remains canonical and deterministic;
- Plot and Surface labels accept whole-string `$...$` mathematical text through
  the existing offline math-text path. Mixed rich text, animation, and a
  general-purpose 3D camera/scene API remain out of scope.

See the v2.3 examples,
[Functions](FUNCTIONS.md), [LSP](LSP.md), and
[Plot](PLOT.md) references.

## What is new in v2.2 <a id="what-is-new-in-v22"></a>

v2.2.0, **Plot Completeness**, is a deliberately small visualization release.
It closes the gaps that showed up while using AhdCode's own GUI and Plot
together, and adds no syntax and no type-system change.

- [`Plot.pie(labels, values)`](PLOT.md#pie): one slice per category, in
  the order given, with its category legend on by default and its share
  written on each slice. A pie has no axes, so `xLabel` and `yLabel` on one
  raise `PlotError` instead of being quietly dropped.
- [`Plot.heatmap(xLabels, yLabels, values)`](PLOT.md#heatmap): a
  labelled grid whose colour carries the numbers, with a colour-scale
  legend. The Matrix is one row per y label and one column per x label.
- [`Surface.xCategories` and `Surface.yCategories`](PLOT.md#naming-the-coordinates):
  names for a Surface's x and y coordinates, so a grid of courses and years
  stops being labelled `1` to `5`. They are presentation only — the geometry
  is untouched, and `xLabel`/`yLabel` remain the axis titles.

Both new charts are ordinary `Chart` values: they take `title`, `legend`,
`size`, `save` to PNG, SVG, and PDF, `show` in the interactive viewer, and a
place in a `Plot.subplots` Figure.

See the v2.2 example.

## What is new in v2.1.0 <a id="what-is-new-in-v210"></a>

v2.0 completed the desktop application foundation. v2.1, **Application I/O
and Network Completion**, closes the remaining practical file and network
gaps that CLI, Web, and desktop programs share. It adds no syntax, no
type-system change, and no byte type: binary data stays opaque and travels
file to file.

- [`HTTP`](HTTP.md): `Client.download(url, path)` and
  `Client.sendToFile(request, path)` stream a response body straight to a
  file and return a `ClientFileResponse` — status, headers, final URL, and
  the number of bytes written, with no `body()` because the payload is in
  the file. A failed transfer never damages the destination.
- [`HTTP`](HTTP.md): `ClientRequest.withMultipartField` and
  `withMultipartFile` send a `multipart/form-data` body, streaming each file
  from disk. Outbound multipart completes the pair with the inbound uploads
  v0.8 added.
- [`WebSocket`](WEBSOCKET.md): a synchronous client —
  `HTTP.webSocketClient(url)` configures it, `connect()` opens one
  `WebSocketConnection`, and `receive()` waits for the next message. No
  callbacks, no background event loop, no reconnect.
- [`SMTP`](SMTP.md): `SMTPMessage.withAttachment(path, fileName, contentType)`
  attaches real files, base64-encoded straight from disk. A message without
  attachments produces exactly the MIME it always did.
- The v2.0 Plot viewer's Save is fixed for a program that uses Plot but not
  GUI; it needs no environment variable.

See the v2.1 examples.

## What is new in v2.0.0 <a id="what-is-new-in-v200"></a>

v2.0.0 is a **major** release, **Desktop Application Completion**. It closes
the foundational desktop roadmap without new syntax or a type-system change;
every v1.9 program keeps working:

- [`GUI`](GUI.md): ListBox, Select, TextArea, PasswordInput, and a
  read-oriented TableView with scrolling and selection; `onChange` for text
  fields and Checkboxes, and selection callbacks; resizable windows whose
  tables and lists use the extra space; Shift+Tab; and native file, folder,
  save, message, and confirmation dialogs. The foundational GUI roadmap is
  complete: future additions are focused, use-case-driven widgets rather than
  missing platform foundations.
- [`Plot`](PLOT.md): the viewer gains a toolbar — Save (PNG, SVG, PDF,
  written by the renderer exactly as `save()` writes), Zoom Out, Zoom In,
  Rotate Left, Rotate Right, and Fit — and `Plot.surface` draws a 3D surface
  from a Numeric Matrix, with an orbit/pan/zoom viewer, wireframe mode, and
  PNG export.
- [`ahdcode package`](PACKAGING.md) turns a program into a desktop
  application — a macOS `.app`, or a Windows or Linux folder and archive —
  that holds only the helpers it needs and runs without AhdCode installed.
- Editors complete, hover, and show signatures for every Chart and Figure
  member, fixing a gap from v1.8 and v1.9.

See the v2.0 examples.

## What is new in v1.9.0 <a id="what-is-new-in-v190"></a>

v1.9.0 is a **minor** release, **Desktop Polish + Interactive Plot**. It
changes how `show()` presents a chart, adds colors and an enabled state to
the GUI, and names AhdCode's own windows; the core grammar and the type
system are unchanged.

- [`Plot`](PLOT.md): `chart.show()` and `figure.show()` open AhdCode's
  own interactive viewer instead of the operating system's image viewer:
  scroll to zoom around the pointer, drag to pan, Q/E to turn the view a
  quarter turn, R to reset, and Escape to close. These interactions change
  only the viewer; they do not change the Chart or Figure, their data, or
  exported PNG, SVG, or PDF files. `save()` is unchanged.
- [`GUI`](GUI.md): basic colors — `setBackground` on a Window and a
  Container, and `setForeground`/`setBackground` on a Label, Button,
  TextInput, and Checkbox, spelled like Graphics colors — and an enabled
  state (`setEnabled`/`isEnabled`) for a Button, TextInput, and Checkbox.
  AhdCode GUI remains a deliberately small toolkit for forms, utilities, and
  educational desktop applications.
- GUI, Graphics, and Plot viewer windows show the AhdCode name and icon: on
  macOS **AhdCode** in the menu bar and the AhdCode icon in the Dock, on
  Windows and Linux the window icon where the system shows one. This names
  AhdCode's own windows; it does not package user programs.

See the v1.9 examples.

## What is new in v1.8.0 <a id="what-is-new-in-v180"></a>

v1.8.0 is a **minor** release, **GUI Foundations + Minimal Events**. It adds
one standard module, two Graphics Canvas members, and readable output for five
value Classes; the core grammar and the type system are unchanged.

- [`GUI`](GUI.md): a new standard module for small desktop windows with a
  Label, a Button, a single-line TextInput, and a Checkbox, arranged in Columns
  and Rows, plus `Button.onClick` and `Window.onKey` callbacks. It is meant
  for learning and small desktop applications such as forms, data-entry tools,
  simple database front ends, and utilities; it is not a Tkinter-compatible
  toolkit, a game engine, or a professional creative-application framework.
  Each Window is drawn by the bundled `ahdgui` helper.
- [`Graphics`](GRAPHICS.md): `Canvas.onClick` and `Canvas.onKey` run a
  Function on a click (Cartesian coordinates) or a key press, so a Turtle can
  be driven by the arrow keys without reading the terminal.
- Callbacks run one at a time on the program's own path; their shapes are
  checked at compile time, and a callback's error propagates out of `wait()`
  unchanged.
- `str`, `write`, and `Terminal.pretty` now show the contents of a Vector,
  Matrix, DateTime, Duration, and Table, for example `Vector([3.0, 4.0])` and
  `Duration(1500 ms)`, instead of only the Class name.

See the v1.8 examples.

## What is new in v1.7.0 <a id="what-is-new-in-v170"></a>

v1.7.0 is a **minor** release, **Standard Library Completion**. It strengthens
six existing modules with additive functions and members; the core grammar,
the type system, and every previously released function stay unchanged, and
no dependency is added.

- [`Math`](MATH.md): `asin`, `acos`, `atan`, `atan2`, `sinh`, `cosh`,
  `tanh`, `hypot`, `log2`, `cbrt`, `radians`, `degrees`, `gcd`, and `lcm`.
  Angles stay plain Real values; there is no angle type and no `Math.pow`.
- [`Time`](TIME.md): `Time.parseISO` and `DateTime.toISO` for a strict
  RFC 3339 subset (a `Z` or `±HH:MM` designator is required), instant
  arithmetic with `DateTime.add` and `subtract`, and exact Duration `add`,
  `subtract`, `negate`, and `abs`. There is still no named time-zone database.
- [`Numeric`](NUMERIC.md): `Vector.at`, `norm`, `outer`, and `cross`;
  `Matrix.at`, `row`, `column`, `diagonal`, `norm` (Frobenius), `hadamard`,
  and `matvec`. No broadcasting.
- [`Statistics`](STATISTICS.md): `covariance`, `sampleCovariance`,
  `correlation` (Pearson), and `linearRegression` returning
  `{"slope", "intercept"}`. No statistical test suite.
- [`Data`](DATA.md): `Table.concat` and `Table.innerJoin` with one shared
  key or a left and right key. Inner join only, no missing values, and no
  automatic column suffixes.
- [`Security`](SECURITY.md): `bcryptHash` and `bcryptVerify` from the
  already pinned `golang.org/x/crypto`, for compatibility and migration. New
  applications should prefer Argon2id.

Every addition behaves identically compiled and in `ahdcode run` and the REPL,
and editors complete, hover, and show signatures for each one. See the
v1.7 examples.

## What is new in v1.6.0 <a id="what-is-new-in-v160"></a>

v1.6.0 is a **minor** release, **Graphics + Turtle**. It adds one standard
module and keeps the core grammar, the type system, and every previously
released function unchanged.

**New standard module: [`Graphics`](GRAPHICS.md)** — visual programming
and 2D drawing:

- `Graphics.open(width, height, title, background)` opens a window with a
  Canvas whose coordinates are Cartesian: `(0, 0)` is the center and `+y` points
  up.
- `canvas.line`, `canvas.circle`, and `canvas.rectangle` draw with named colors
  or `#RRGGBB`/`#RRGGBBAA`; `fill` is optional (`String?`), and a rectangle is
  given by its lower-left corner.
- `canvas.save("x.png")` and `canvas.save("x.svg")` export the drawing;
  `canvas.wait()` keeps the window open until it is closed, and
  `canvas.close()` closes it.
- `canvas.turtle()` returns a Turtle pen (`forward`, `left`, `right`,
  `moveTo`, `penUp`, `penDown`, `setColor`, `setWidth`, `home`, `x`, `y`,
  `heading`) whose geometry is deterministic and does not depend on time or
  frame rate.

Canvas and Turtle calls follow the usual rule: all positional or all named.
Compiled programs, `ahdcode run`, and the REPL share one implementation. Each
Canvas is drawn by the bundled `ahdgraphics` helper, a separate program that
keeps the window library out of the compiler and out of compiled programs.
Graphics has no sprites, animation loop, keyboard or mouse input, audio,
widgets, or 3D. See the
Graphics and Turtle examples.

## What is new in v1.5.0 <a id="what-is-new-in-v150"></a>

v1.5.0 is a **minor** release, **Terminal**. It adds one standard module and
keeps the core grammar, the type system, and every previously released function
unchanged.

**New standard module: [`Terminal`](TERMINAL.md)** — the terminal-specific
behaviour `write`, `take`, and `str` deliberately leave out:

- `Terminal.emit(parts, separator, ending)` joins a `List<String>` onto standard
  output and converts nothing.
- `Terminal.error(text, ending)` writes to standard error, and
  `Terminal.flush()` writes buffered standard output immediately.
- `Terminal.isInteractive()`, `Terminal.width()`, and `Terminal.height()`
  describe standard output; the sizes are `Int?` and never guessed.
- `Terminal.supportsColor()` and `Terminal.style(...)` add color only on an
  interactive terminal, honour `NO_COLOR` and `TERM=dumb`, and return plain text
  when output is redirected.
- `Terminal.pretty(value)` lays out a List or Pair over lines for reading.

Terminal is not a TUI framework: there is no cursor control, raw keyboard input,
or progress output. It adds no dependency, compiled programs and the REPL
produce identical output, and editors now show the `?` of nullable parameters
and returns in hover and signature help. `Terminal` is now a standard module
name: a local `Terminal.ahd` beside a program is no longer what `bring Terminal`
loads. See the Terminal demo.

## What is new in v1.4.0 <a id="what-is-new-in-v140"></a>

v1.4.0 is a **minor** release, **Realtime Web & Data**. It adds two standard
modules, WebSocket server endpoints, and `Env.secret`. The core grammar, the
type system, and every previously released function keep their behaviour.

**New standard module: [`UUID`](UUID.md)** — `UUID.v4()` and the
time-ordered `UUID.v7()` make immutable `UUIDValue`s. `UUID.parse` accepts only
the canonical 36-character form, and values compare with `equals` and
`compare`. `Identity.id()` is unchanged.

**New standard module: [`PostgreSQL`](POSTGRESQL.md)** — `connect`,
`execute`, `query`, and `begin` in the MySQL family, with `$1` placeholders,
`boolean`, exact `numeric`, `timestamptz` in UTC, `RETURNING`, and PostgreSQL's
aborted-transaction rule made explicit. Only `connect`'s arguments choose the
server: `PG*` variables and `~/.pgpass` have no effect. pgx is vendored, so
builds stay offline.

**WebSocket endpoints in [`HTTP`](WEBSOCKET.md)** — `HTTP.websocket`,
`Server.websocket`, and `App.websocket` host text-message endpoints on the same
server as your routes. Callbacks run one at a time with HTTP handlers, `onOpen`
has returned before the client sees the connection open, the default origin
policy is same-origin, `withAccept` authenticates before the upgrade, and
message size, queue, and connection count are bounded.

**Extended module: [`Env`](ENV.md#secret)** — `Env.secret(name)` reads
`NAME`, or the file named by `NAME_FILE`, the convention container platforms
use for mounted secrets.

`UUID` and `PostgreSQL` are now standard module names: a local `UUID.ahd` or
`PostgreSQL.ahd` beside a program is no longer what `bring UUID` or
`bring PostgreSQL` loads, as with `QR` and `Barcode` in v1.3.0. See
`examples/v0.1/69_uuid.ahd` through
`examples/v0.1/72_websocket_echo.ahd`
and the realtime attendance application.

## What is new in v1.3.0 <a id="what-is-new-in-v130"></a>

v1.3.0 is a **minor** release, **Professional Documents & Machine Codes**. It
adds two standard modules and professional document features for `Latex` and
`PDF`. The core grammar, the type system, and every previously released
function keep their behaviour; new parameters are optional and come last, and
a document that uses none of the new features renders exactly as before.

**New standard module: [`QR`](QR.md)** — `QR.create(value, level)` makes
an immutable `QRCode` at error-correction level L, M, Q, or H, exposes its
module matrix, and saves sharp PNG or vector SVG files, with `QRError`.

**New standard module: [`Barcode`](BARCODE.md)** — `Barcode.code128`,
`Barcode.ean13`, and `Barcode.upca` make immutable `BarcodeCode` values with
computed or verified check digits, expose the bar pattern, and save PNG or SVG
files, with `BarcodeError`.

**Extended module: [`Latex`](LATEX.md#professional-documents-v130)** —
`qr`, `barcode`, `place`, `header`, `footer`, `pageNumber`, `pageCount`,
`link`, and `bookmark`; `document(paper:, pageSize:, margins:, subject:,
keywords:, creator:)`; and `image`/`figure` with SVG files and a `transform`
of rotation, opacity, and trims. The offline bundle adds `fancyhdr` and
`lastpage`.

**Extended module: [`PDF`](PDF.md)** — `layout`, `header`, `footer`,
`pageNumbers`, `qr`, `barcode`, `link`, `bookmark`, `metadata`, and `image`
with SVG files and a `transform`, with every String still escaped and no raw
TeX.

SVG files become vector drawing inside the program: never rasterized, and with
no browser, Inkscape, or other external converter. QR codes and barcodes are
generated offline by a vendored, MIT-licensed encoder. See
`examples/v0.1/62_qr.ahd` through
`examples/v0.1/68_svg_assets.ahd`.

## What is new in v1.2.0 <a id="what-is-new-in-v120"></a>

v1.2.0 is a **minor** release, **Scheduling, Vector Documents & Character
Utilities**. It adds two standard modules and vector graphics for `Latex`. The
core grammar, the type system, and every previously released function keep
their behaviour; `Latex.document` gains one optional final parameter.

**New standard module: [`Cron`](CRON.md)** — bounded, in-process
scheduling on classic five-field schedules: `Cron.scheduler()`,
`Scheduler.add(expression, task)`, `Scheduler.run()`, `Scheduler.stop()`,
`Cron.next(expression, after)`, and `CronError`. Jobs run while the program
that scheduled them runs. Cron is not the operating system's crontab, a daemon,
or a job queue, and it never changes system scheduling configuration.

**New standard module: [`Characters`](CHARACTERS.md)** — Unicode
code-point operations with no `Char` type: `list`, `count`, `codePoint`,
`fromCodePoint`, `isLetter`, `isDigit`, `isWhitespace`, `isUpper`, `isLower`,
`isAlphaNumeric`, `isPunctuation`, `isSymbol`, and `CharactersError`. A
character is a `String` holding one code point; grapheme clusters are not
segmented.

**Extended module: [`Latex`](LATEX.md#vector-graphics-with-tikz-v120)** —
TikZ/PGF vector graphics through the same offline pipeline: `Latex.tikz`,
`Latex.overlay`, `Latex.border`, and `Latex.document(..., landscape: true)`.
The offline resource bundle now also carries TikZ, nine TikZ libraries, and
pgfornament with its ornaments, so certificates, page borders, watermarks, and
diagrams need neither a second PDF library nor a pre-rendered image. See
`examples/v0.1/61_tikz_certificate.ahd`.

## What is new in v1.1.0 <a id="what-is-new-in-v110"></a>

v1.1.0 is a **minor** release. It adds one new standard module and extends an
existing one. The core grammar, the type system, and every previously released
function keep their v1.0.0 behaviour.

**New standard module: [`Bits`](BITS.md)** — bitwise operations on the
language's signed 64-bit `Int`. AhdCode's grammar has no bitwise operators
(`and`, `or` and `not` are the logical operators and `^` is exponentiation), so
these are named calls:

`bitAnd`, `bitOr`, `bitXor`, `bitNot`, `shiftLeft`, `shiftRight`,
`shiftRightUnsigned`, `rotateLeft`, `rotateRight`, `count`, `leadingZeros`,
`trailingZeros`, plus the `BitsError` error type raised when a shift or rotate
distance falls outside `0..63`.

**Extended module: [`Security`](SECURITY.md)** — the password primitives
are unchanged; the module now also covers digests, message authentication, the
common encodings, RS256 signatures and authenticated symmetric encryption:

`sha256`, `sha512`, `hmacSHA256`, `hmacVerify`, `base64Encode`, `base64Decode`,
`base64UrlEncode`, `base64UrlDecode`, `hexEncode`, `hexDecode`, `randomHex`,
`rsaSignSHA256`, `rsaVerifySHA256`, `aesEncrypt`, `aesDecrypt`.

Where a cryptographic choice exists it is made once and is not exposed as a
knob: SHA-2 for digests, HMAC-SHA256 for MACs, RSASSA-PKCS1-v1_5 over SHA-256
(the algorithm JWT calls RS256) for signatures, and AES-256-GCM for encryption.

---

AhdCode v1.0.0 is the first stable release. The exclusions below are deliberate design decisions, not gaps awaiting a later version.

Within the language, AhdCode intentionally excludes block/statement lambdas, arbitrary/implicit mutable closures, general user-defined operator overloading (outside the ten fixed Class Protocol Methods), multiple return values/tuples, reflection, traits/interfaces, and multiple inheritance. In tooling, AhdCode uses compile-time local source composition ([`require(...)`](REQUIRE.md)) and bundled offline modules rather than an external package manager or remote registry. Language server references and rename use compiler-authoritative, bounded on-demand workspace snapshots; they do not start an asynchronous background index. See the [specification's unsupported-feature list](AHDCODE_LANGUAGE_SPEC_v0.1.md#40-unsupported-v01-features).

## Repository map

```text
cmd/ahdcode/         CLI entry point and command router
cmd/ahdnumeric/      bundled advanced linear-algebra helper
cmd/ahdplot/         bundled chart-rendering helper
cmd/ahdgraphics/     bundled Graphics window helper (its own Go module)
cmd/ahdgui/          bundled GUI window helper (its own Go module)
cmd/ahdplotview/     bundled interactive Plot viewer (its own Go module)
cmd/ahdidentity/     shared AhdCode window name and icon for the window helpers
cmd/ahdsqlite/       bundled CGO-free SQLite helper
internal/            compiler frontend, backend, runtime, formatter, LSP, and REPL
internal/framework/  bundled first-party Web framework source
editors/vscode/      VS Code / Antigravity editor extension
docs/                authoritative reference guides and tutorials
examples/v0.1/       curated core language programs
examples/v0.3/       SQLite Notes App
examples/v0.4/       Web Notes App
examples/v0.5/       cookies and in-memory sessions
examples/v0.6/       outbound HTTP Client and JSON APIs
examples/v0.7/       HTML parsing, selectors, and web scraping
examples/v0.8/       multipart forms, file uploads, and upload metadata
examples/v0.9/       SMTP text/HTML mail through Env-configured servers
examples/v0.12/      MySQL raffle demo (join codes and announced winner)
examples/v0.14/      multi-file web app with require(...) and static assets
examples/v0.15/      Math Portal dogfood application
examples/v0.16/      forms, validation, CSRF, and flash workflow
examples/v2.3/       v2.3 release examples
tools/AhdDataStudio/ first-party local MySQL + SQLite development UI
AHDCODE_LANGUAGE_SPEC_v0.1.md
                     authoritative core language specification
```

## Development and credits

AhdCode is designed and specified by Ali Harun Daldallı. Implementation,
documentation, and testing have been developed with extensive AI assistance,
including OpenAI Codex, Anthropic Claude, and Google Gemini. Their roles vary
by task; language design and final technical decisions remain with the project
author.

## License

AhdCode is available under the MIT License.
