# AhdCode v0.7 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs introduce v0.7.0 HTML parsing: `HTML.parse` turns an HTML
String into a read-only `HTMLDocument`, and `select`/`first` find
`HTMLElement` values in it with a small CSS-like selector language (tag,
`#id`, `.class`, `[attr]`, `[attr="value"]`, descendant/child combinators,
comma lists). Parsing never makes a network request and never executes
script content -- it only reads the String you give it. There is no
scraper module, no browser, and no JavaScript engine.

```bash
ahdcode run examples/v0.7/01_parse_html.ahd
```

| Example | Topic |
|---|---|
| `01_parse_html.ahd` | Parse a literal HTML String; `select`, `first`, `text()`, `attr()` |
| `02_http_scrape.ahd` | HTTP Client `get` + `HTML.parse`, two independent steps |
| `03_scrape_to_sqlite.ahd` | Extract with `HTML`, persist with `SQLite`, bound parameters |

Example 02 needs internet access. It defaults to `https://example.com/`, a
small stable static page; set `SCRAPE_URL` to point at a page you control
instead. Example 03 opens `SQLite.open(":memory:")` so running it never
leaves a database file behind -- see [`examples/v0.3`](../v0.3/README.md)
for persisting to a real file.
