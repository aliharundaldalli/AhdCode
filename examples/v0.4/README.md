# AhdCode v0.4 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs introduce the v0.4.0 web foundation: an HTTP server, request
and response values, and safe structured HTML. Bind `127.0.0.1` so the page is
only on this machine, then open `http://127.0.0.1:8080/` in a browser.
`Server.start()` occupies the terminal until you stop the program.

Run the Web Notes app from a **temporary directory** so `notes.db` is not
created inside the repository. `ahdsqlite` must be installed beside `ahdcode`
(`go install ./cmd/ahdsqlite`). HTTP-only examples need no SQLite helper.

```bash
scratch="$(mktemp -d)"
cp examples/v0.4/03_web_notes.ahd "$scratch/"
cd "$scratch"
ahdcode run 03_web_notes.ahd
```

| Example | Topic |
|---|---|
| `01_http_hello.ahd` | Trusted static HTML over HTTP |
| `02_http_request.ahd` | Query parameters and `application/x-www-form-urlencoded` forms |
| `03_web_notes.ahd` | Web Notes App: SQLite persistence, escaped HTML, POST-redirect-GET |

A separate beginner demo that combines HTTP, HTML, SQLite, and Env as a small
library desk lives in
[ahdcode-library-demo](https://github.com/aliharundaldalli/ahdcode-library-demo)
(`v0.1.0`).

A multi-page Hatay seminar desk (home, register, login, one seminar note per
name) lives in
[ahdcode-seminer-demo](https://github.com/aliharundaldalli/ahdcode-seminer-demo)
(`v0.1.0`).
