# HTML standard module

[English] · [Türkçe](HTML_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [HTTP](HTTP.md) · [Student Guide](STUDENT_GUIDE_EN.md#36-a-small-web-page)

> **v0.15:** `HTML` is the low-level builder and stays exactly as documented
> here. [`Web.UI`](WEB.md#9-webui) is an ergonomic layer written in AhdCode
> *over* these primitives -- `Web.UI.p("x")` is `HTML.element("p", {},
> [HTML.text("x")])` -- and produces identical markup with identical escaping.
> `HTML.element` remains the escape hatch for any tag or attribute pattern
> `Web.UI` does not name. Neither module has a raw-markup helper.


If you are learning this module, start with the [HTML workshop](PRACTICAL_MODULES.md#8-html-build-safe-pages-and-parse-documents)
for safe page building, selectors, null checks, and parsing an HTTPS response;
use this page as the complete builder/parser reference.

`HTML` is the compiler-registered `builtin:HTML` module. The builder half
(`HTMLNode`) was introduced in AhdCode v0.4.0; the parsing half
(`HTML.parse`, `HTMLDocument`, `HTMLElement`) was added in v0.7.0. It is
explicit and a sibling `HTML.ahd` cannot shadow it:

```ahd
bring HTML
from HTML bring HTMLNode
from HTML bring HTMLDocument
from HTML bring HTMLElement
from HTML bring HTMLError
```

`HTML` is two independent things sharing one module:

- a small safe structured HTML **builder** (`HTML.text`, `HTML.element`,
  `HTML.render`, `HTML.document`) that produces trusted HTML text, and
- a **parser and query surface** (`HTML.parse`, `HTMLDocument`,
  `HTMLElement`) that reads HTML text and lets you find elements in it with
  a small CSS-like selector language.

It is not a template engine, not a browser, and not a full CSS engine.
There is no `HTML.raw` and no `HTML.div` shortcut. Dynamic text and
attribute values passed to the builder are escaped with Go's
`html.EscapeString` at render time.

## Public surface

```text
// Builder (v0.4.0)
HTML.text(value: String) -> HTMLNode
HTML.element(name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode
HTML.render(node: HTMLNode) -> String
HTML.document(title: String, body: List<HTMLNode>) -> String

// Parser (v0.7.0)
HTML.parse(source: String) -> HTMLDocument

HTMLDocument.select(selector: String) -> List<HTMLElement>
HTMLDocument.first(selector: String) -> HTMLElement?

HTMLElement.tag() -> String
HTMLElement.text() -> String
HTMLElement.attr(name: String) -> String?
HTMLElement.hasAttr(name: String) -> Bool
HTMLElement.select(selector: String) -> List<HTMLElement>
HTMLElement.first(selector: String) -> HTMLElement?

HTMLError  (derives from Error)
```

`HTMLNode`, `HTMLDocument`, and `HTMLElement` are opaque built-in Classes:
none can be constructed with `HTMLNode()` / `HTMLDocument()` /
`HTMLElement()`. `HTMLNode` comes only from `text`/`element`. `HTMLDocument`
comes only from `HTML.parse`. `HTMLElement` comes only from
`HTMLDocument.select`, `HTMLDocument.first`, `HTMLElement.select`, or
`HTMLElement.first`. `HTMLDocument` and `HTMLElement` are read-only: there is
no `setAttr`, `append`, `remove`, or any other mutation.

## Builder vs parser: two distinct concepts

A builder `HTMLNode` (something you constructed with `HTML.text`/
`HTML.element` to render) and a parsed `HTMLElement` (something
`HTML.parse` found for you) are unrelated types. There is no implicit or
explicit conversion between them: an `HTMLNode` is never accepted where an
`HTMLElement` is expected, and vice versa. If you parse a page and want to
re-render one of its elements with the builder, rebuild it explicitly with
`HTML.element(...)`.

## Escaping (builder)

```ahd
write(HTML.render(HTML.text("<script>alert(1)</script>")))
```

prints `&lt;script&gt;alert(1)&lt;/script&gt;`. `&`, `<`, `>`, and quotes in
text and attribute values are escaped. Turkish characters and emoji pass
through unchanged. The source String/List/Pair is not mutated.

Use `HTML.text` (or attribute values on `HTML.element`) for any user-provided
or database-provided text. Do not concatenate those values into a raw HTML
String.

## Elements (builder)

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

## Parsing: `HTML.parse` never fetches

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientResponse
bring HTML
from HTML bring HTMLDocument

client: Client := HTTP.client()
response: ClientResponse := client.get("https://example.com/notes")
document: HTMLDocument := HTML.parse(response.body())
```

`HTML.parse(source: String) -> HTMLDocument` takes exactly one String and
returns a parsed document built from it. There is no URL argument. Getting
the page and parsing it are two independent, explicit steps -- `HTML.parse`
itself is pure with respect to networking:

- it never makes an HTTP request,
- it never resolves a URL,
- it never loads an image, script, stylesheet, or iframe named in the
  markup, and
- it never executes anything.

Parsing `<img src="...">`, `<script src="...">`, `<link href="...">`, or
`<iframe src="...">` produces ordinary, unreachable `HTMLElement` values
naming those URLs as plain attribute text -- nothing is ever dialed.

## No JavaScript

```ahd
document: HTMLDocument := HTML.parse("<script>fetch('/x')</script>")
```

A `<script>` element is parsed as ordinary markup: its content becomes a
literal text child of the element, exactly like any other element's
content, reachable through `.text()` if you ask for it. It is never
interpreted as code. There is no JavaScript engine, no DOM, no event loop,
and `onclick`/`onload`/`onerror` attributes are ordinary attribute text --
calling `.attr("onclick")` returns the source string, and nothing ever runs
it.

## Parsing model

`HTML.parse` uses a real hand-written tokenizer and tree builder, not a
regular expression. It gives browser-like recovery for ordinary malformed
HTML -- unclosed tags, missing `<html>`/`<head>`/`<body>`, unquoted
attributes, mixed-case tags:

```ahd
document: HTMLDocument := HTML.parse("<div><p>Hello")
```

still produces a usable tree (a `div` containing a `p` containing the text
`Hello`) with no error. `HTML.parse` is **not** a validator: syntactically
scannable HTML is never rejected, and recovery from a mismatched or missing
end tag simply produces the most reasonable tree recoverable from the
input, not a diagnostic. Parsing only fails (`HTMLError`) for input larger
than an internal size limit or a nesting depth far beyond ordinary pages,
or when the source is not valid UTF-8.

Comments and the doctype are recognized and skipped; neither becomes part
of the parsed tree, and there is no public `Comment` or `Doctype` type.
`<script>` and `<style>` content is captured literally (no entity decoding,
no tag scanning inside it), matching how a browser treats those two
elements.

## Selectors: a frozen, small subset

`select`/`first` accept a small, explicit CSS-like selector language.
Anything outside this list is rejected with `HTMLError` rather than
approximated -- there is no partial match and no silent fallback.

Supported:

| Syntax | Meaning | Example |
| --- | --- | --- |
| `*` | any element | `*` |
| `tag` | tag name | `article`, `a`, `h2` |
| `#id` | id attribute (exact) | `#main` |
| `.class` | a class token (exact) | `.card` |
| `tag.class` etc. | compound (all must match) | `article.card.featured` |
| `[attr]` | attribute present | `[href]` |
| `[attr="value"]` | attribute exact value (quoted) | `[rel="next"]` |
| `A B` | B is a descendant of A | `article a` |
| `A > B` | B is a direct child of A | `article > h2` |
| `A, B` | selector list (either matches) | `h1, h2` |

Whitespace around `>` and `,` is tolerated. **Not** supported, and rejected
with `HTMLError`: pseudo-classes (`:first-child`, `:nth-child(...)`,
`:not(...)`, ...), pseudo-elements (`::before`), sibling combinators (`+`,
`~`), other attribute operators (`^=`, `$=`, `*=`, `~=`, `|=`), CSS escape
syntax, and XPath. An invalid selector always raises `HTMLError`; it never
silently matches nothing or falls back to a looser interpretation.

Matching rules:

- tag names and attribute names are matched **case-insensitively** (HTML's
  own rule -- `DIV` in source and `div` in a selector both match; `attr`
  and `hasAttr` also look up attribute names case-insensitively);
- `id` values, class tokens, and `[attr="value"]` values are matched
  **exactly and case-sensitively**;
- `.class` matches one whitespace-separated token of the `class` attribute;
- results are always in **document order**;
- a selector list's results are **de-duplicated**: an element matching more
  than one branch of `"a, b"` is reported once, at its first document-order
  position.

## Element scope

`HTMLDocument.select`/`.first` search the whole document.
`HTMLElement.select`/`.first` search only **that element's descendants** --
the element itself is never included, matching familiar
`querySelectorAll`-style scoping:

```ahd
articles: List<HTMLElement> := document.select("article.card")
firstArticle: HTMLElement := articles[0]
title: HTMLElement? := firstArticle.first("h2")
```

`title` is found only inside `firstArticle`, never in a different article
elsewhere in the document.

## Text and attributes

`tag()` returns the **normalized (lowercased)** tag name. `<DIV>` in source
and `<div>` both report `tag() == "div"`; there is no way to recover the
source's original capitalization, since it carries no meaning in HTML.

`text()` concatenates every descendant **text node**'s content, in document
order. It does not simulate CSS rendering or "visible text": whitespace is
not collapsed and no spacing is invented between elements. HTML character
references (`&amp;`, `&#65;`, ...) are already decoded by parsing. Comments
never contribute text. Call `.trim()` on the result yourself if you want
leading/trailing whitespace removed.

`attr(name)` returns the parsed attribute value exactly as written, or
`null` if the attribute is absent. `hasAttr(name)` tells present from
absent even when the value is empty (`<input disabled>` has `disabled` with
value `""`, and `hasAttr("disabled")` is `true`).

## No automatic URL resolution

```ahd
link: HTMLElement? := article.first("a")
href: String? := link.attr("href")  // exactly "/notes/1", never resolved
```

`HTML.parse` does not know a "page URL" -- it only ever sees the String you
gave it -- so a relative `href`/`src` value is returned exactly as parsed,
never turned into an absolute URL. There is no `baseURL` or `resolveURL` in
v0.7.0. Combine the site's known base with the returned String yourself
where an absolute URL is needed.

## What this module does not do

No template language, no `HTML.raw`, no browser, no headless rendering
engine, no JavaScript execution, no CSS layout or computed styles, no full
CSS selector engine (only the frozen subset above), no automatic resource
fetching, no URL resolution, and no DOM mutation API. The parsed
`HTMLDocument`/`HTMLElement` surface is read-only by design.
