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
| Forms, validation, old input, error bags | Planned for v0.16 |
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
academy.get("/", homePage)
academy.get("/hakkinda", aboutPage)
academy.post("/selam", greetPage)
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
homePage: Function := (request: Request) -> Response {
    content: Local List<HTMLNode> := [
        Web.UI.section([Web.UI.h1(configuration().name), Web.UI.p(tagline())])
    ]
    return mainLayout(configuration(), "Ana Sayfa", "/", content)
}
```

`Web.UI` holds the primitive HTML components; your `Components/` directory
holds application-specific ones. They are intentionally separate.

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
| v0.17 | Richer routing, route groups, middleware composition, auth guards |

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
