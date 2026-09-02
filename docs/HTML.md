# HTML standard module

[English] · [Türkçe](HTML_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [HTTP](HTTP.md) · [Student Guide](STUDENT_GUIDE_EN.md#36-a-small-web-page)

`HTML` is the compiler-registered `builtin:HTML` module, introduced in
AhdCode v0.4.0. It is explicit and a sibling `HTML.ahd` cannot shadow it:

```ahd
bring HTML
from HTML bring HTMLNode
from HTML bring HTMLError
```

`HTML` is a small safe structured HTML builder. It is not a template engine,
DOM, parser, or CSS engine. There is no `HTML.raw` and no `HTML.div` shortcut.
Dynamic text and attribute values are escaped with Go's `html.EscapeString` at
render time. Nodes store the original unescaped String.

## Public surface

```text
HTML.text(value: String) -> HTMLNode
HTML.element(name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode
HTML.render(node: HTMLNode) -> String
HTML.document(title: String, body: List<HTMLNode>) -> String

HTMLError  (derives from Error)
```

`HTMLNode` is an opaque built-in Class: it cannot be constructed with
`HTMLNode()`, has no instance members, and is obtained only from `text` and
`element`. All arguments are positional. Empty attributes are `{}`. Empty
children are `[]`. Attribute and child order are preserved.

## Escaping

```ahd
write(HTML.render(HTML.text("<script>alert(1)</script>")))
```

prints `&lt;script&gt;alert(1)&lt;/script&gt;`. `&`, `<`, `>`, and quotes in
text and attribute values are escaped. Turkish characters and emoji pass
through unchanged. The source String/List/Pair is not mutated.

Use `HTML.text` (or attribute values on `HTML.element`) for any user-provided
or database-provided text. Do not concatenate those values into a raw HTML
String.

## Elements

Tag names must start with an ASCII letter, then letters, digits, or hyphens.
Attribute names also allow `_`. Empty names, spaces, quotes, `<`, `>`, `=`,
and `/` are rejected with `HTMLError`. Duplicate attribute names raise
`HTMLError`.

Void elements cannot have children: `area`, `base`, `br`, `col`, `embed`,
`hr`, `img`, `input`, `link`, `meta`, `param`, `source`, `track`, `wbr`.
Passing children to a void element raises `HTMLError`. They render without a
closing tag.

```ahd
node: HTMLNode := HTML.element("h1", {}, [HTML.text("Hello")])
input: HTMLNode := HTML.element("input", {"name": "title", "value": userTitle}, [])
```

`HTML.document(title, body)` returns a complete page String:

```text
<!doctype html><html><head><meta charset="utf-8"><title>…</title></head><body>…</body></html>
```

The title is escaped. Send that String with `HTTP.html(...)` so the response
has `text/html; charset=utf-8`.

## Trusted static HTML vs the builder

`HTTP.html(r"""...""")` sends a String you wrote in source. It does **not**
sanitize. That is appropriate for a fixed hello page.

`HTML.text(userProvidedValue)` is for dynamic content. Web Notes stores titles
and bodies as SQLite Strings, then renders them with `HTML.text`, so
`<script>` stays data in the database and appears escaped in the page.

## What this module does not do

No template language, no `HTML.raw`, no HTML parser, no CSS, no JavaScript
module, no DOM, and no component framework. The builder is small on purpose.
