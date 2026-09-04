# v0.14 multi-file web example

A small two-page site demonstrating AhdCode v0.14's application foundations:
`require(...)` local source composition, dependency-aware `ahdcode dev`, and
safe static asset serving.

```
app.ahd
Components/
    Layout.ahd       -- page shell; requires Shared/HTMLHelpers.ahd and
                         Components/Navigation.ahd (a nested require)
    Navigation.ahd    -- the nav bar
Pages/
    Home.ahd
    About.ahd
Shared/
    HTMLHelpers.ahd   -- small reusable HTML-node builders
public/
    app.css           -- served through Server.static, not a compiled-in route
```

## Run it

```bash
cd examples/v0.14/multi_file_web
ahdcode dev app.ahd
```

Open [http://127.0.0.1:8095/](http://127.0.0.1:8095/). `ahdcode stop app.dev`
stops it cleanly.

## What to try

- Edit `Components/Navigation.ahd` (add a link) while `ahdcode dev` is
  running: it rebuilds and restarts automatically, even though `app.ahd`
  itself never changed -- `dev` watches the whole resolved `require(...)`
  graph, not just the entry file.
- Edit `public/app.css` while running: the app does **not** rebuild, but a
  browser refresh (or `curl http://127.0.0.1:8095/assets/app.css`) sees the
  new content immediately. Static files are served straight from disk.
- Introduce a syntax error in `Pages/About.ahd`: `ahdcode dev` reports it and
  keeps the previously working app running; fixing the file recovers
  automatically.

## The require(...) rules this example exercises

- **App-root-relative, always.** `Components/Layout.ahd` requires
  `"Shared/HTMLHelpers.ahd"` and `"Components/Navigation.ahd"` using the
  exact same paths `app.ahd` would use, even though `Layout.ahd` itself
  lives one directory down -- every require path is resolved from the
  application root (`app.ahd`'s own directory), never from the requiring
  file.
- **One shared namespace.** `Pages/Home.ahd` calls `pageShell(...)` and
  `paragraph(...)` directly, unqualified, even though neither is declared in
  that file -- every required file's top-level declarations share one
  application-wide namespace.
- **File-local `bring`.** `Pages/Home.ahd` and `Pages/About.ahd` each
  declare their own `bring HTML` / `from HTML bring HTMLNode`, even though
  `Components/Layout.ahd` already brings the same module: a required file's
  `bring` never leaks to, or is inherited from, another file.
