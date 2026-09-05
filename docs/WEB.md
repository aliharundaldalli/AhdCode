# Web

[English] · [Türkçe](WEB_TR.md)

[Back to README](../README.md) · [HTTP](HTTP.md) · [HTML](HTML.md) · [Modules](MODULES.md) · [Env](ENV.md) · [require(...)](REQUIRE.md)

`Web` is AhdCode's first-party web framework, introduced in v0.15.0. One
import covers the ordinary application:

```ahd
bring Web
```

## 1. What Web is

Web is a **facade**. It composes the existing `HTTP`, `HTML`, `Session`,
`Security`, and `Env` primitives rather than replacing them, and it re-exports
the types those primitives already define. A `Request` reached through `Web`
is the same `Request` `HTTP` hands a handler — the same type, not a copy of
it, so a handler written against `Web` registers on a bare `HTTP.Server` with
no conversion.

Almost all of Web is written in AhdCode itself. The framework ships as
bundled AhdCode source embedded in the compiler, compiled in the same pass as
your own files. Go handles what only Go can: sockets, TLS, the filesystem
boundary, cryptography, and the low-level HTTP implementation.

## 2. What Web is not

v0.15 deliberately does **not** include:

| Not included | Where it stands |
| --- | --- |
| ORM, query builder, migrations | Use `SQLite` / `MySQL` explicitly |
| Forms, validation, old input, error bags | Released in v0.16 (see section 16) |
| Middleware chains, route groups, auth guards | Planned for v0.17 |
| Template language | `Web.document` is a shell, not a template |
| Virtual DOM, hydration, reactive state, hooks | Not planned; pages are composed on the server |
| Frontend bundler, npm, Node, CSS-in-JS | Not planned; assets are files on disk |
| Browser live reload / HMR | Not planned for v0.15 |
| ACME, Let's Encrypt, certificate renewal | Terminate TLS at a reverse proxy |

There is no package manager either. `bring Web` resolves offline, from bytes
embedded in the compiler: no registry, no manifest, no lockfile, no download.

## 3. `bring Web`

`bring Web` gives you a namespace plus the types an application declares:

```ahd
bring Web
from Web bring (Request, Response, HTMLNode, App, AppConfig)
```

Web re-exports `Request`, `Response`, `Server`, `Session`, `SessionStore`,
`Cookie`, `UploadedFile`, `HTTPError`, `HTMLNode`, and `HTMLError`, and adds
its own `App`, `AppConfig`, `UIKit`, and `WebConfigError`.

`Web` is a reserved first-party name: a sibling `Web.ahd` in your project
cannot shadow it, exactly as a sibling `HTTP.ahd` cannot shadow `HTTP`.

## 4. Application structure

The convention, composed with the released
[`require(...)`](REQUIRE.md) rule:

```
app.ahd
.env
.env.example

Config/
    App.ahd

Components/
    Navbar.ahd
    Notice.ahd

Layouts/
    Main.ahd

Pages/
    Home.ahd
    About.ahd

public/
    app.css
    logo.svg
```

This is a **convention**, not package semantics. `require("Pages/Home.ahd")`
resolves from the application root — the entry file's own directory — however
deep the require chain gets.

## 5. Config

The rule is one-directional:

```
.env / process environment
        ↓
     Config
        ↓
app / pages / components
```

`Config/App.ahd` is the only file that reads the environment. Pages, layouts,
and components receive an already validated `AppConfig`.

```ahd
bring Web
from Web bring AppConfig
bring Env

application: AppConfig := Web.configure()

configuration: Function := () -> AppConfig {
    application: Global AppConfig
    return application
}

tagline: Function := () -> String {
    return Env.getOr("APP_TAGLINE", "AhdCode ile yazılmış küçük bir akademi")
}
```

`Web.configure()` loads `.env` if it exists, then validates the whole
contract, raising `WebConfigError` naming the key at fault.

The compiler does not enforce the Config convention in v0.15, and there is no
linter for it. It is a discipline the canonical structure makes easy.

## 6. Environments

Six keys are frozen:

| Key | Values | Notes |
| --- | --- | --- |
| `APP_NAME` | any non-empty String | |
| `APP_ENV` | `development`, `test`, `production` | lowercase, no default |
| `APP_HOST` | a host, e.g. `ahdakademi.com` | no scheme, path, or port |
| `APP_PROTOCOL` | `http`, `https` | lowercase, no default |
| `SERVER_HOST` | e.g. `127.0.0.1` | the socket this process binds |
| `SERVER_PORT` | 1–65535 | converted and range-checked |

Nothing is silently defaulted. An application that does not say which
environment it is running in is misconfigured, not "probably development"; one
that does not name a protocol is not quietly served over `http`. Either
default would decide cookie security and redirect targets by accident.

`APP_HOST` is a **host**, not a URL:

```
ahdakademi.com          accepted
admin.checkmate.tr      accepted

https://ahdakademi.com  rejected — scheme is APP_PROTOCOL
ahdakademi.com/path     rejected — a host has no path
ahdakademi.com:8080     rejected — port is SERVER_PORT
```

A malformed value is rejected with a message that says which rule it broke,
rather than trimmed into something that merely looks right.

`APP_ENV=test` is a recognized, isolated environment for automated tests. It
does not enable production behaviour and does not append `.test`. v0.15 adds
no testing DSL.

Configuration errors name the offending key and never echo its value, so a
bad `DB_PASSWORD` cannot reach a log through the error path.

## 7. App

```ahd
require("Config/App.ahd")
require("Pages/Home.ahd")
bring Web
from Web bring App

academy: App := Web.app(application)

academy.assets("/assets", "public")
academy.get("/", home)
academy.get("/hakkinda", about)
academy.post("/selam", greet)
academy.start()
```

`App` owns two things: the validated configuration and the bound server.

| Method | Meaning |
| --- | --- |
| `get(path, handler)` | register GET on one exact path |
| `post(path, handler)` | register POST on one exact path |
| `route(method, path, handler)` | register any supported method |
| `assets(prefix, root)` | serve a directory of static files |
| `start()` | bind and serve; does not return |
| `configuration()` | the validated `AppConfig` |

A handler is an ordinary `Function(Request) -> Response`, type-checked exactly
as `HTTP` checks it — passing the wrong shape is a compile error naming the
expected signature.

Its *name* is irrelevant to routing; see
[10.1 Naming](#101-naming) for the recommended convention.

## 8. Responses

| Call | Returns |
| --- | --- |
| `Web.html(node, status := 200)` | HTML response from a node tree |
| `Web.page(title, body, head := [], status := 200)` | full document as a response |
| `Web.document(title, body, head := [])` | full document as markup |
| `Web.text(body, status := 200)` | plain-text response |
| `Web.redirect(location, status := 303)` | redirect (303 suits POST-then-redirect) |
| `Web.response(status, body, contentType)` | anything else |
| `Web.render(node)` | node tree to markup |
| `Web.sessions(...)` | a `SessionStore` |

## 9. Web.UI

`Web.UI` is the semantic HTML component layer. Two shapes cover nearly
everything:

```
text elements       tag(text, attributes := {})
container elements  tag(children, attributes := {})
```

```ahd
Web.UI.section(
    [
        Web.UI.h1("Ahd Akademi")
        Web.UI.p("Hoş geldiniz")
        Web.UI.img("/assets/logo.svg", "Ahd Akademi")
    ]
    {"class": "hero"}
)
```

against the same tree built with the low-level module:

```ahd
HTML.element(
    "section"
    {"class": "hero"}
    [
        HTML.element("h1", {}, [HTML.text("Ahd Akademi")])
        HTML.element("p", {}, [HTML.text("Hoş geldiniz")])
        HTML.element("img", {"src": "/assets/logo.svg", "alt": "Ahd Akademi"}, [])
    ]
)
```

Both produce identical markup. `HTML.element` remains available and remains
the escape hatch.

### Attributes

Attributes are always the last parameter and always optional, one typed
`Pair<String, String>`. That one door covers `id`, `class`, `title`, `data-*`,
`aria-*`, and everything else, with no `Any`, no dynamic values, and no
per-tag parameter explosion:

```ahd
Web.UI.p("Kayıt açıldı", {"class": "lead", "data-role": "notice", "aria-live": "polite"})
```

A helper's own attributes are appended after yours, so ordering is
deterministic. A helper never writes into the Pair you passed: a Pair is a
reference, and a shared attribute map would otherwise accumulate every
element's attributes.

### Children

A text element that also needs markup children has a `Nodes` companion:

```ahd
Web.UI.p("Plain text")
Web.UI.pNodes([Web.UI.text("Hello "), Web.UI.strong("Ali")])
Web.UI.aNodes("/", [Web.UI.img("/logo.svg", "Home")])
```

`h1Nodes`…`h6Nodes`, `pNodes`, `spanNodes`, `liNodes`, `ddNodes`, `aNodes`,
`tdNodes`, `thNodes`, `labelNodes`, `buttonNodes`, `blockquoteNodes`,
`summaryNodes`, and `figcaptionNodes` exist for this.

### The element groups

**Core** — `text`, `element`, `render`, `stylesheet`

**Document** — `html`, `head`, `body`, `title`

**Page structure** — `header`, `footer`, `main`, `section`, `article`,
`aside`, `nav`, `div`, `address`

**Headings** — `h1`–`h6` (+ `Nodes`)

**Text** — `p`, `span`, `blockquote` (+ `Nodes`), `strong`, `em`, `b`, `i`,
`u`, `small`, `mark`, `code`, `pre`, `time`, `br`, `hr`

**Lists** — `ul`, `ol`, `dl`, `li` (+ `liNodes`), `dt`, `dd` (+ `ddNodes`)

**Links and media** — `a` (+ `aNodes`), `img`, `picture`, `source`, `figure`,
`figcaption` (+ `Nodes`)

**Tables** — `table`, `caption`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td`
(+ `thNodes`, `tdNodes`)

**Forms** — `form`, `formTo`, `label`, `labelFor`, `input`, `textarea`,
`select`, `option`, `button`, `fieldset`, `legend` (+ `labelNodes`,
`buttonNodes`)

**Disclosure** — `details`, `summary` (+ `summaryNodes`)

### Escaping

Every String that becomes page content goes through `HTML.text`, which
escapes it:

```ahd
Web.UI.p("<script>alert(1)</script>")
```

renders as

```html
<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>
```

There is no `raw()`, `unsafeHTML()`, `innerHTML()`, or `trustedHTML()`
anywhere in `Web` or `HTML`. An ordinary UI call cannot turn a database or
form value into executable HTML, and v0.15 adds no way to opt out.

### Accessibility

`img` requires `alt`. A decorative image passes `""` deliberately; no
signature lets `alt` be forgotten.

`labelFor(target, text)` writes the `for` attribute; the matching `id` is
yours to set on the input:

```ahd
Web.UI.labelFor("isim", "Adınız")
Web.UI.input("text", "isim", "", {"id": "isim"})
```

Heading hierarchy is your responsibility, and button-versus-link semantics
stay explicit. Web never generates an accessibility attribute you did not ask
for.

## 10. Pages, Layouts, Components

```
App        manages the application
Page       creates content
Layout     wraps the page
Component  creates a reusable fragment
```

**These are conventions, not language constructs.** Each is an ordinary
AhdCode `Function` returning an `HTMLNode` (or a `Response`). There is no base
class, no lifecycle, no state engine, no render scheduler, no hooks, and no
registry.

A **Component**:

```ahd
bring Web
from Web bring HTMLNode

notice: Function := (title: String, message: String) -> HTMLNode {
    return Web.UI.section([Web.UI.h2(title), Web.UI.p(message)], {"class": "notice"})
}
```

A **Layout**:

```ahd
require("Components/Navbar.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig)

mainLayout: Function := (config: AppConfig, title: String, current: String, content: List<HTMLNode>) -> Response {
    body: Local List<HTMLNode> := [
        navbar(config.name, current)
        Web.UI.main(content)
        Web.UI.footer([Web.UI.p("{config.name} — {config.effectiveURL()}")])
    ]
    return Web.page("{title} — {config.name}", body, [Web.UI.stylesheet("/assets/app.css")])
}
```

A **Page**:

```ahd
home: Function := (request: Request) -> Response {
    content: Local List<HTMLNode> := [
        Web.UI.section([Web.UI.h1(configuration().name), Web.UI.p(tagline())])
    ]
    return mainLayout(configuration(), "Ana Sayfa", "/", content)
}
```

`Web.UI` holds the primitive HTML components; your `Components/` directory
holds application-specific ones. They are intentionally separate.

### 10.1 Naming

`Page`, `Layout` and `Component` are **application organization conventions,
not special language constructs.** Nothing in the compiler, the router or the
runtime looks at a name.

**Handler names are ordinary AhdCode identifiers. `Page` is not a required
suffix.** A route is given a `Function` **value**:

```ahd
portal.get("/admin/users/edit/*", adminUserEdit)
```

Whatever that function is called — `adminUserEdit`, `adminUserEditPage`,
`admin_user_edit` — the router receives the same value and behaves
identically. Applications written against earlier releases that use
`registerPage` or `profilePage` stay source-compatible; nothing is deprecated.

New examples in this documentation drop the redundant suffix, and name the
action explicitly when one screen needs more than one handler:

```
register            // GET  /register
registerSubmit      // POST /register

adminQuestionEdit   // GET  the edit form
adminQuestionSave   // POST the edit form
```

**Recommended file names use PascalCase**, so the word boundaries in the
handler name are the word boundaries in the file name:

```
Pages/Admin/UserEdit.ahd        ->  adminUserEdit    (camelCase, preferred)
                                ->  admin_user_edit  (snake_case, also valid)
Pages/Admin/QuestionEdit.ahd    ->  adminQuestionEdit
Pages/Admin/Users.ahd           ->  adminUsers
```

The style terms, used precisely:

```
adminUserEdit      camelCase
AdminUserEdit      PascalCase
admin_user_edit    snake_case
```

Mixed spellings such as `admin_UserEdit` combine two conventions without
adding meaning; the examples avoid them. `adminUserEdit` is preferred simply
because the existing AhdCode examples are predominantly camelCase — the
identifier grammar accepts underscores, so `admin_user_edit` is ordinary valid
code, and the choice belongs to the application.

Identifiers are **case-sensitive**, and this is a naming convention rather than
a lookup rule, so collapsed or case-folded spellings are not alternate
spellings of the same name:

```
adminuseredit   adminUSEREDIT   adminUseredit   admin_UserEdit
```

None of those is `adminUserEdit`. There is no automatic route discovery, no
filename reflection, no filename-to-function lookup, no case-insensitive or
fuzzy handler matching, no `Page` annotation and no `page` keyword. The
require chain is explicit and the route argument is a plain value.

## 11. Static assets

```ahd
academy.assets("/assets", "public")
```

`public/app.css` is then served at `/assets/app.css`. This delegates to the
released `server.static`.

Editing a static file changes what the browser gets on the next request. It
does **not** rebuild AhdCode source, because none of it is AhdCode source.
There is no hot module replacement and no injected browser refresh: reload the
page.

## 12. Dev mode

```bash
ahdcode dev app.ahd
```

```
AhdCode Dev

Watching: /path/to/app.ahd

→ Building...
✓ Build succeeded
→ Starting...
✓ Running

AhdCode Web
  Ahd Akademi (development)

  Open:
  http://127.0.0.1:8080

  Development identity:
  http://ahdakademi.com.test
  (.test is not locally routed in v0.15)

Waiting for changes...
```

**Open the address under `Open:`.** That is `SERVER_HOST` and `SERVER_PORT`,
the socket the application actually binds, and it is the only address that
works on this machine.

The line under `Development identity:` is the name the application is
*configured* with. v0.15 derives it but does not resolve it — there is no
bundled `.test` resolver — so it is shown for reference and marked as such,
never as somewhere to click.

The v0.13/v0.14 dependency-aware dev controller is unchanged and still owns
the source graph, the watcher, last-good, rebuild, restart, and stop. Web adds
a banner and two safety checks.

Editing any source file in the `require(...)` graph rebuilds and restarts.
Editing `public/app.css` does not.

### Refusals

`ahdcode dev` **refuses** `APP_ENV=production`:

```
✗ APP_ENV is production, but this is the development command.
  ahdcode dev runs an application in development.
  Set APP_ENV=development for local work, or run the built
  executable directly for a production configuration.
  Nothing was started and APP_ENV was not changed.
```

It also **refuses** `APP_PROTOCOL=https`:

```
✗ Local HTTPS is not available in AhdCode v0.15.
  ahdcode dev serves plaintext HTTP, so it cannot honour
  APP_PROTOCOL=https.

  Configured identity:
  https://ahdakademi.com.test

  Set APP_PROTOCOL=http for local development, or terminate
  HTTPS with an external local proxy in front of 127.0.0.1:8080.
  Nothing was started and APP_PROTOCOL was not changed.
```

`ahdcode dev` starts the application, and the application binds a plaintext
HTTP socket. There is no path in v0.15 by which `APP_PROTOCOL=https` results
in TLS here, so starting the child would mean serving `http` while the
configuration says `https`. It refuses instead of downgrading: a silent
downgrade would hide a secure-cookie or mixed-content problem until
production.

In both refusals nothing is started, no listener is opened, no `.dev`
descriptor is left behind, `APP_ENV` and `APP_PROTOCOL` are unchanged, and the
command exits non-zero.

`APP_ENV=test` runs normally. It uses `APP_HOST` unchanged, so the banner
reports the bind address alone and shows no `.test` identity.

## 13. `.test`

For `APP_ENV=development`, the local identity is `APP_HOST` with `.test`
appended:

```
APP_HOST=ahdakademi.com   →   ahdakademi.com.test
```

`.test` is a reserved special-use TLD (RFC 6761); using it means development
traffic can never resolve to the real host by accident.

**`.test` is a logical identity in v0.15, not a routable address.** AhdCode
installs no DNS, no resolver entry, and no `/etc/hosts` record for it, so the
name does not resolve through ordinary macOS resolution and cannot be opened
in a browser. The directly usable local address is `SERVER_HOST:SERVER_PORT`,
which is what `ahdcode dev` prints first.

Production uses `APP_HOST` exactly — the real canonical domain — and never
gains the suffix. `APP_ENV=test` uses `APP_HOST` unchanged too.

`AppConfig` exposes the derivations:

| Call | `development` | `production` |
| --- | --- | --- |
| `url()` | `https://ahdakademi.com` | `https://ahdakademi.com` |
| `developmentURL()` | `https://ahdakademi.com.test` | `https://ahdakademi.com.test` |
| `developmentHost()` | `ahdakademi.com.test` | `ahdakademi.com.test` |
| `effectiveURL()` | `https://ahdakademi.com.test` | `https://ahdakademi.com` |
| `address()` | `127.0.0.1:8080` | `127.0.0.1:8080` |

## 14. Local HTTPS — current limitation

`.test` does not resolve on its own, and v0.15 **does not ship** a local
certificate authority, a `.test` resolver, or a development gateway. There is
no `ahdcode trust` command in this release.

Reaching `https://<APP_HOST>.test` with no visible port needs three
permanently privileged pieces of system state at once:

1. a root-installed resolver for the `.test` domain (`/etc/resolver/test` plus
   a resolver process, or root-managed `/etc/hosts` entries),
2. a listener on privileged port 443, or a root-installed packet redirect,
3. a certificate authority in the system trust store.

That is a long-lived, privileged local network daemon. It is deferred rather
than approximated: v0.15 installs no system state, requests no privilege, and
adds no local trust artifacts.

Because of that, `APP_PROTOCOL=https` makes `ahdcode dev`
[refuse to start](#refusals) rather than serve the application over plaintext
http while calling it https. It never silently downgrades `https` to `http`,
and it never generates an untrusted certificate. A silent downgrade would hide
a secure-cookie or mixed-content problem until production.

For local work today, set `APP_PROTOCOL=http` and use
`http://127.0.0.1:SERVER_PORT`, or terminate TLS with a proxy you already run.

## 15. Production

```
    Internet
        ↓
Cloudflare / Caddy / nginx
        ↓
    public HTTPS
        ↓
 AhdCode application
        ↓
127.0.0.1:SERVER_PORT
```

The distinction that matters:

| | Keys | Meaning |
| --- | --- | --- |
| Public identity | `APP_PROTOCOL`, `APP_HOST` | what a person types; what belongs in a link |
| Process socket | `SERVER_HOST`, `SERVER_PORT` | what this process binds |

`APP_PROTOCOL=https` describes the **public URL**. It does not mean the
AhdCode process itself terminates TLS. Behind Cloudflare, Caddy, or nginx it
usually does not:

```
APP_PROTOCOL=https
APP_HOST=ahdakademi.com     →   https://ahdakademi.com

SERVER_HOST=127.0.0.1
SERVER_PORT=8080            →   127.0.0.1:8080
```

Never derive a public URL from `SERVER_PORT`.

v0.15 is not a production certificate manager: no ACME, no Let's Encrypt
automation, no DNS challenges, no renewal service. If your `HTTP` primitives
already support direct TLS, that remains available and unchanged.

## 16. `.env`

`.env` is a **local development convenience**.

- Do not commit a real `.env`. Commit `.env.example` with names and
  placeholders, no secrets.
- A variable already present in the process environment **wins** over `.env`,
  so containers and CI override predictably without editing files.
- The grammar is plain `KEY=value` with optional quoting. There is no
  interpolation, no command substitution, and nothing is executed.
- `ahdcode build` does **not** embed `.env`. The built executable is an
  environment-independent artifact; run it with a different environment and it
  is a different deployment.

Secrets — `DB_PASSWORD`, API keys, SMTP passwords, tokens — stay runtime
configuration and never enter a binary.

```bash
cp .env.example .env
```

## 17. Database

Conventional keys, documented but **not** consumed by Web:

```
DB_HOST
DB_PORT
DB_NAME
DB_USERNAME
DB_PASSWORD
```

Database access stays explicit — `bring MySQL` or `bring SQLite` — and
`Config/Database.ahd` reads and validates these. There is no ORM and no
generic `Web.Database`.

## 18. Errors

Web adds exactly one error type: `WebConfigError`, for a missing or malformed
environment contract. Everything else keeps the identity that already
describes it — `HTTPError`, `HTMLError`, session errors, `SecurityError` —
because a catch-all `WebError` would only rebrand failures the existing
modules already report well.

## 19. Security boundaries

- **Escaping** is not optional. Every text entry point escapes; there is no
  raw-markup helper.
- **Secrets** never enter a binary and never appear in configuration errors.
- **Sessions, CSRF, and password hashing** remain the explicit `Session`,
  `Security`, and `HTTP` primitives. Web adds no magic around them.
- **Static assets** go through the released `server.static` boundary.
- **`bring Web`** resolves offline from embedded bytes. No download, ever.

## 20. Relation to HTTP and HTML

Web never replaces the low-level modules. This stays valid and unchanged:

```ahd
bring HTTP
bring HTML

server: Server := HTTP.server("127.0.0.1", 8080)
node: HTMLNode := HTML.element("p", {}, [HTML.text("hello")])
```

Every v0.4–v0.14 application keeps compiling and running. Web is additive;
there is no migration.

Reach for the low-level modules when you need a tag or attribute pattern
`Web.UI` does not name, or when you want the server without the framework's
configuration contract.

## 21. What is next

| Release | Area |
| --- | --- |
| v0.16 | Forms, validation, CSRF conveniences, flash, old input, form errors |
| v0.17 | `ahdcode init web`, context-aware routes, route groups, ordered guards |
| v0.18 | Web starters: Empty, Basic, Admin; local Bootstrap; Admin DB bootstrap |

## Starting a project

```bash
mkdir my-app
cd my-app
ahdcode init web
ahdcode dev app.ahd
```

On a terminal, `init web` asks Empty, Basic, or Admin. You can also run
`ahdcode init web empty|basic|admin`. This is a pre-1.0 change from the
v0.17 immediate scaffold.

Templates and [Bootstrap 5.3.3](https://getbootstrap.com/) (MIT) ship inside
the CLI. Generated pages load only local files:

- `/assets/vendor/bootstrap/bootstrap.min.css`
- `/assets/vendor/bootstrap/bootstrap.bundle.min.js`
- `/assets/style.css`
- `/assets/main.js`
- `/assets/ahdcode-logo.png`

There is no CDN, npm, or network fetch during `init web`.

## 18. v0.18: Web starters and application bootstrap

v0.18 does not change RequestContext, RouteSet, guards, Forms, CSRF, Flash,
Security, SQLite, MySQL, HTTP, or SMTP. It changes the **startup experience**.

### Empty

A polished welcome application. No database, login, dashboard, repositories,
or mail keys. `.env` stays the six application keys.

### Basic

The same shell plus `Config/Mail.ahd` and `MAIL_*` keys that those functions
actually read. `MAIL_SECURITY` defaults to `starttls` (SMTP on port 587).
Empty `MAIL_HOST` still lets the app start. No database and no authentication.

### Admin

Public Home, Login, Dashboard, and POST `/logout` (CSRF, then redirect to
`/`). `signedIn` is ordinary generated application code:

```ahd
routes.get("/dashboard", dashboard, signedIn)
```

Login uses Form, ValidationErrors, CSRF, session rotation, Flash, and email
old input only. Failures do not say whether the email exists.

#### SQLite

Creates `database/<name>.db` and `database/schema.sql`, applies the schema,
and inserts the administrator with `Security.passwordHash`. An existing
`.db` file stops init. Generated `database/*.db` files are gitignored;
`schema.sql` is not.

#### MySQL

Asks host, port, database name, username, and password (hidden). Uses the
released MySQL contract (`tls` or `none`; default `tls`). After local
conflict checks:

1. connect without selecting a database
2. refuse if the database already exists (no `--force`, no DROP, no ALTER)
3. `CREATE DATABASE` with a validated identifier `[A-Za-z][A-Za-z0-9_]*`
4. apply schema and insert the administrator
5. write application files

Init does not create MySQL users or change GRANT permissions. If the
supplied account cannot `CREATE DATABASE`, init reports that and stops.
If this invocation created a new database and a later step fails, the
database is **not** dropped.

Environment key names:

Empty: `APP_NAME`, `APP_ENV`, `APP_HOST`, `APP_PROTOCOL`, `SERVER_HOST`,
`SERVER_PORT`

Basic also: `MAIL_HOST`, `MAIL_PORT`, `MAIL_USERNAME`, `MAIL_PASSWORD`,
`MAIL_FROM_ADDRESS`, `MAIL_FROM_NAME`, `MAIL_SECURITY`

Admin SQLite also: `DB_DRIVER`, `DB_DATABASE`

Admin MySQL also: `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`,
`DB_SECURITY`

## 17. v0.17: context routes, groups, and guards

v0.17 is additive. `App.get(path, handler)` with
`Function(Request) -> Response` still compiles. A second registration
layer opens one `RequestContext` per request and still requires
`context.respond` on the path that runs.

```ahd
routes := Web.routes(site, sessions)
routes.get("/profile", profile)

admin := routes.group("/admin")
admin.get("/users", adminUsers, authenticated, adminOnly)
```

| Piece | Contract |
| --- | --- |
| handler | `Function(RequestContext) -> Response` |
| guard | `Function(RequestContext) -> Response?` |
| `null` | continue |
| `Response` | stop; must already be `context.respond(...)` |

Guard order is the extra arguments of `get`/`post`/`route`, then the
handler. Defaults allow every request. There is no `next()`, no `use()`,
no hidden finalizer, no auth policy inside Web, and no named route
parameters. Function attributes cannot store a guard list, so the checks
stay on the registration line.

Group join is explicit. `/admin` + `/users` is `/admin/users`. `/admin` +
`/*` is `/admin/*`. Fragments with `?`, `#`, `//`, or a guessed repair
raise `WebRouteError`. HTTP still owns exact-vs-`/*` matching.

A focused example is
[`examples/v0.17/routes_guards`](../examples/v0.17/routes_guards).

v0.17 does **not** add a general middleware chain, an auth framework, an
ORM, automatic route discovery, or a frontend runtime.

## Example

A complete application lives in
[`examples/v0.15/ahd_academi`](../examples/v0.15/ahd_academi): config layer,
one layout, two pages, two components, GET and POST routes, static CSS, and
`require(...)` composition.

```bash
cd examples/v0.15/ahd_academi
cp .env.example .env
ahdcode dev app.ahd
```

## 16. v0.16: request context, forms, validation, CSRF and flash

v0.16 reduces the repeated session, validation, and form plumbing
found in the Math Portal. It preserves ordinary `Function -> HTMLNode`
composition and explicit data flow. All new conveniences are bundled AhdCode
source; the native HTTP parser, SessionStore, and Security implementation stay
unchanged. There is no new dependency or language syntax.

### Run the complete example

```bash
cd examples/v0.16/forms_validation
ahdcode run app.ahd
```

Open `http://127.0.0.1:8160/register`. Submit invalid input, then valid input;
follow the redirect to `/profile` and refresh to see the message disappear.
No database, account provisioning, `.env` file, or downloaded assets are needed.
The example binds loopback and explicitly uses a non-Secure cookie for local
HTTP. For HTTPS applications create the store with `secure: true` (or use the
existing `AppConfig.isSecure()` policy). An optional shell `SERVER_PORT` changes
the example port; `.env.example` is documentation, not automatically loaded.

### One context, one explicit outgoing response

Create one context per incoming request with `Web.context(request, store)`.
The store is chosen by the application, normally once at startup. The context
holds the original `request`, the opened `session`, the commit `store`, and a
`finalized` flag. It is an ordinary reference value passed to Functions; there
is no request singleton, service container, implicit authorization, or hook.

Build a response and call `context.respond(response)` exactly once on the
executed return path, including 200, 403, 404, 422, and redirect responses. It
commits through the selected SessionStore, then marks the context finalized.
A second call, including through an alias, raises `WebContextError` before
committing again. If commit raises, finalization has not succeeded. There are
no automatic retries. A handler must return the finalized response.

Finish session mutations **before** responding. Use `context.session` with
existing login, rotate, or destroy functions. The context's mutating CSRF/flash
methods reject use after finalization. Its fields are ordinary AhdCode class
attributes, not an access-control boundary: do not replace its session/store,
reset its flag, or mix low-level commit with context finalization for the same
response. Low-level `HTTP`, `Request.form`, `SessionStore.open/commit`, and every
v0.15 `Web.UI` helper remain available without migration. Both exact routes and
the v0.15.1 trailing one-segment `/*` routes accept the same handlers; exact
routes still take precedence.

### GET, POST, validate, redirect

On GET, create a context, render `Web.UI.csrfField(context)` inside the form,
and finalize the page. On POST, get `context.form()`, explicitly verify CSRF,
validate, then either render errors and selected old input or perform the
application mutation and redirect. This handler uses the view and store accessor
from the [complete runnable example](../examples/v0.16/forms_validation/app.ahd):

```ahd
registerSubmit: Function := (request: Request) -> Response {
    context: Local RequestContext := Web.context(request, exampleSessions())
    form: Local Form := context.form()
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    errors: Local ValidationErrors := Web.errors()
    errors.required("name", form.value("name"), "Name is required.")
    errors.email("email", form.value("email"), "Enter an email address.")
    errors.minLength("password", form.value("password"), 10, "Use at least 10 characters.")
    errors.matches("password_confirmation", form.value("password_confirmation"), form.value("password"), "Passwords must match.")
    if errors.any() {
        return context.respond(registerView(context, form.old(["name", "email"]), errors, 422))
    }
    context.flashSet("notice", "Form accepted. No account was created.")
    return context.respond(Web.redirect("/profile"))
}
```

`Web.form(request)` also works without a session or context, for forms that do
not need CSRF. It wraps the parser's existing snapshot; it never parses a second
body or invents dynamic values. `value` returns the first submitted string,
unchanged, with a fallback only when missing. `optional` distinguishes null
from an empty string; `hasField` tests presence, including empty values.
Normalize deliberately with `.trim()` or `.lower()` where appropriate. Do not
trim passwords unless that is an explicit application policy. Query strings
remain separate: use `Request.query` for a search form using the URL query.

`integer` returns null for absence and raises `FormValueError` for empty,
malformed, or overflowing input, following the existing `int(String)` grammar.
`checkbox` returns false for absence, true for `on`/`true`/`1`, and false for
`off`/`false`/`0`; other strings raise `FormValueError`. Values are case-sensitive.
Use `hasField` first if missing and explicit false must differ. Errors never
include submitted values. Multipart files still use `Request.file/files`.

### Validation is explicit and ordered

`Web.errors()` creates a fresh collection. Rule methods append errors when the
supplied value fails. They neither read requests nor transform submitted values.
`required` rejects empty or whitespace-only values. Length rules use AhdCode
`len(String)` semantics and inclusive bounds; choose non-negative bounds.
`matches` is ordinary value equality for confirmation fields, not secret-token
verification. `email` checks a single `@`, nonempty local/domain parts, no space,
tab, CR or LF, and at least two nonempty domain labels. It is a shape check, not
RFC validation or proof of deliverability. `hexColor` accepts exactly `#RRGGBB`.
`oneOf` uses exact string membership, including case.

Errors remain in rule/add call order. `first` returns null if none;
`forField` and `messages` return independent lists of strings. Multiple errors
for one field are retained. `errors.add("email", "Already registered.")`
keeps database uniqueness and other business policy in application code. Render
messages through `Web.UI` text nodes, as the example's FormErrors component does.

### Old input is a deliberate allowlist

`form.old(["name", "email"])` creates a new `OldInput` value containing only
those present fields. The field list is required; there is no automatic copy-all
operation or session persistence. Passwords, confirmations, reset verifiers,
CSRF tokens, and other secrets are excluded by leaving them off the allowlist.
**The framework does not guess sensitive field names or override an explicit
selection. The application must select only safe fields.** Use an empty list
for an empty form. Pass `OldInput` to a view explicitly; it is not a request global.

`old.value` returns an ordinary string. `Web.UI.input` escapes it in the value
attribute: `<script>alert(1)</script>` remains data. Password controls should
receive `""`, as in the example. Old input is not automatically available after
a redirect; this first version demonstrates immediate validation re-rendering.

### CSRF and flash state

`csrfToken()` uses `Security.token()` once per session and keeps it under the
reserved `__web_csrf` session key. It remains stable through expected requests
and rotation that preserves session values. Clearing/destroying the session
requires new tokens. `Web.UI.csrfField(context)` renders an escaped hidden
`_csrf` input; it does not submit or validate anything. `csrfValid()` rejects
missing/empty expected or submitted tokens and uses `Security.secureEqual`.
Verification does not mint tokens. Never log token or password values.

`flashSet(key, value)` stores a string under `__web_flash:{key}`. `flashTake(key)`
returns a nullable string and removes it; a second take returns null. Choose your
own keys and visual categories. The next rendering handler should take the
message and finalize its response so the removal persists. Refresh then sees no
message. There is no automatic rendering or request-age expiration: a message
stays pending until explicitly consumed. Reserve `__web_csrf` and the
`__web_flash:` prefix for Web, including when mixing older application helpers.

The store retains its released in-memory, process-local lifetime and cookie
policy. Context adds neither durable sessions nor cross-process synchronization.
Application user/site context, guards, and DB/JSON conversion stay explicit.
No middleware, auth framework, ORM, JSON redesign, routing redesign, or asset
system is included in this release.

### Exact v0.16 API

```text
Web.context(request: Request, store: SessionStore) -> RequestContext
Web.form(request: Request) -> Form
Web.errors() -> ValidationErrors
Web.UI.csrfField(context: RequestContext) -> HTMLNode

RequestContext.respond(response: Response) -> Response
RequestContext.form() -> Form
RequestContext.csrfToken() -> String
RequestContext.csrfValid() -> Bool
RequestContext.flashSet(key: String, value: String) -> Nothing
RequestContext.flashTake(key: String) -> String?

Form.optional(name: String) -> String?
Form.value(name: String, fallback: String := "") -> String
Form.hasField(name: String) -> Bool
Form.integer(name: String) -> Int?
Form.checkbox(name: String) -> Bool
Form.old(safeFields: List<String>) -> OldInput
OldInput.value(name: String, fallback: String := "") -> String

ValidationErrors.add(field: String, message: String) -> Nothing
ValidationErrors.any() -> Bool
ValidationErrors.hasField(field: String) -> Bool
ValidationErrors.first(field: String) -> String?
ValidationErrors.forField(field: String) -> List<String>
ValidationErrors.messages() -> List<String>
ValidationErrors.required(field: String, value: String, message: String) -> Nothing
ValidationErrors.minLength(field: String, value: String, minimum: Int, message: String) -> Nothing
ValidationErrors.maxLength(field: String, value: String, maximum: Int, message: String) -> Nothing
ValidationErrors.matches(field: String, value: String, expected: String, message: String) -> Nothing
ValidationErrors.email(field: String, value: String, message: String) -> Nothing
ValidationErrors.oneOf(field: String, value: String, allowed: List<String>, message: String) -> Nothing
ValidationErrors.hexColor(field: String, value: String, message: String) -> Nothing
```

New concrete types and constructor attributes:

| Type | Attributes |
| --- | --- |
| `RequestContext` | `request: Request`, `session: Session`, `store: SessionStore`, `finalized: Bool := false` |
| `Form` | `request: Request` |
| `FormField` | `name: String`, `value: String` |
| `OldInput` | `fields: List<FormField>` |
| `FieldError` | `field: String`, `message: String` |
| `ValidationErrors` | `entries: List<FieldError>` |

Use `Web.context`, `Web.form`, `Web.errors`, and `Form.old` as the normal
constructors. `WebContextError` and `FormValueError` inherit `Error` and its
attributes; existing error identities are unchanged. New APIs appear through
the compiled ModuleInterface in completion, hover, and signature help.
