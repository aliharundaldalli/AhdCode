# Ahd Akademi — AhdCode v0.15 Web example

[English] · [Türkçe](README_TR.md)

A small but real Web application: one import, a config layer, a layout, two
pages, two components, GET and POST routes, and static CSS.

## Run it

```bash
cp .env.example .env
ahdcode dev app.ahd
```

Then open `http://127.0.0.1:8080/`.

`ahdcode dev` prints the canonical development URL
(`http://ahdakademi.com.test`). `.test` names the local identity of
`APP_HOST`; v0.15 does not ship the resolver that makes it open by itself, so
use the loopback address and port for now. See
[docs/WEB.md](../../../docs/WEB.md#14-local-https--current-limitation).

Stop it with `ahdcode stop app.dev`.

## What it shows

```
app.ahd            bring Web, routes, static assets, start
.env.example       the environment contract, no secrets
Config/App.ahd     the only file that reads the environment
Layouts/Main.ahd   the shared document shell
Pages/Home.ahd     GET / and POST /selam
Pages/About.ahd    GET /hakkinda
Components/        navbar and notice, ordinary Functions
public/            app.css and logo.svg, served from disk
```

- **`bring Web`** is the whole import. No `bring HTTP`, no `bring HTML`.
- **Config layer.** `Config/App.ahd` calls `Web.configure()` once and exposes
  it through `configuration()`. No page reaches for `Env`.
- **Pages, Layouts, Components** are ordinary Functions returning `HTMLNode`
  or `Response`. No base class, no lifecycle, no registry.
- **`Web.UI`** builds the tree: `nav`, `main`, `section`, `h1`, `h2`, `p`,
  `a`, `img`, `ul`, `li`, `table`, `formTo`, `labelFor`, `input`, `button`,
  `footer`.
- **A POST route** reads the field with the released low-level
  `request.form("isim")`, which returns a nullable String handled explicitly.
  There is no Forms framework in v0.15.
- **Escaping.** Type `<script>alert(1)</script>` into the greeting form and
  the page shows those characters as text.
- **`require(...)`** composes every file from the application root.
- **Static assets.** Editing `public/app.css` serves new bytes on the next
  request without rebuilding AhdCode source. Editing any `.ahd` file rebuilds
  and restarts.

## Environments

```bash
# development: the local identity gains .test
APP_ENV=development APP_HOST=ahdakademi.com   →  ahdakademi.com.test

# production: APP_HOST exactly, never .test
APP_ENV=production  APP_HOST=ahdakademi.com   →  ahdakademi.com
```

`ahdcode dev` refuses `APP_ENV=production` rather than run a production
configuration under development semantics.

The public URL and the bind address are separate: `APP_PROTOCOL`/`APP_HOST`
say what a person types, `SERVER_HOST`/`SERVER_PORT` say what this process
binds. Behind a reverse proxy they differ by design.

## Notes

Do not commit a real `.env`. `.env.example` carries names and placeholders
only, and `ahdcode build` never embeds either one — the built executable reads
its configuration from the environment at start-up.
