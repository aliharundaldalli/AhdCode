# Templates standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [HTML](HTML.md) · [HTTP](HTTP.md) · [Web](WEB.md)

`Templates` is the compiler-registered `builtin:Templates` module. It is
explicit and a sibling `Templates.ahd` cannot shadow it:

```ahd
bring Templates
from Templates bring TemplateSet
from Templates bring TemplateData
from Templates bring TemplateError
```

`Templates` renders HTML files with Go's `html/template`. It is not a second
template language, not a browser, and not a replacement for the
[`HTML`](HTML.md) builder. Use `HTML` when you build a small tree in AhdCode.
Use `Templates` when the page lives in `.html` files beside the program.

Ordinary values are escaped. The only raw markup path is
`Templates.trusted`. Template actions cannot call AhdCode functions, open
files, or reach the network.

## Public surface

```text
Templates.nullValue()                         -> TemplateData
Templates.fromBool(value: Bool)               -> TemplateData
Templates.fromInt(value: Int)                 -> TemplateData
Templates.fromReal(value: Real)               -> TemplateData
Templates.fromString(value: String)           -> TemplateData
Templates.list(values: List<TemplateData>)    -> TemplateData
Templates.object(values: Pair<String, TemplateData>) -> TemplateData
Templates.trusted(markup: String)             -> TemplateData

Templates.parse(name: String, source: String) -> TemplateSet
Templates.sources(names: List<String>, bodies: List<String>) -> TemplateSet
Templates.files(root: String)                 -> TemplateSet

TemplateSet.render(name: String, data: TemplateData) -> String
TemplateSet.layout(layout: String, page: String, data: TemplateData) -> String

TemplateError  (derives from Error)
```

`TemplateData` and `TemplateSet` are opaque. They cannot be constructed with
`TemplateData()` or `TemplateSet()`. `TemplateData` has no members.

## Data

Every value that reaches a template is built with an explicit constructor.
There is no `Any` and no automatic conversion from an ordinary AhdCode
value:

```ahd
data: TemplateData := Templates.object({
    "title": Templates.fromString("Ana sayfa")
    "count": Templates.fromInt(3)
    "topics": Templates.list([
        Templates.fromString("Şablonlar")
        Templates.fromString("Yerleşim")
    ])
})
```

`Templates.object` evaluates its Pair argument exactly once (native and
evaluator). `fromReal` rejects a non-finite number. Nested lists and objects
are a JSON tree: child values are embedded directly, so encoding size grows
with the data rather than exponentially. List and object constructors enforce
a maximum depth of 32 (root at depth 0) and a maximum of 10000 nodes,
including the new container. A duplicate object key raises `TemplateError`.

## Escaping and trusted markup

`fromString` stores an ordinary string. Go's `html/template` escapes it when
the action writes into an HTML context, so `<script>` becomes visible text.
An apostrophe becomes `&#39;`. That is the default for untrusted content.

`Templates.trusted` stores markup that is already trusted. It renders without
that escaping. Use it only for content the application itself produced, for
example a fragment already built with `HTML.render`. Do not pass request
bodies, query values, cookie values, or database fields through `trusted`
unless they were sanitized by a separate, explicit step.

There is no other raw-insertion helper.

## Shared templates and page isolation

Names under `pages/` are **page-private**. Everything else (`layouts/`,
`components/`, and other paths) is **shared**.

Each render adds only the selected page's definitions to a clone of the
shared prototype. Pages can supply definitions for shared layout blocks,
such as `content` and `sidebar`. Every file owns its template name, and
only pages can define names under `pages/`. A conflicting file name or a
duplicate definition across shared files raises `TemplateError` at load time.

Concurrent `render` and `layout` calls each use their own clone.
`layout(layout, page, data)` requires the name of a loaded shared
template for `layout` and the name of a loaded template under `pages/`
for `page`. Either kind of template can be rendered directly.

## Loading templates

Names are slash-separated relative paths. They must not begin with `/`, must
not contain `\`, `:`, NUL, or a newline, must not be a Windows volume path
such as `C:/a.html`, and must not use `.`, `..`, or a dot-prefixed segment.
At most 16 path segments and 128 characters.

```ahd
pages: TemplateSet := Templates.files("templates")
```

`files(root)` walks that directory once, at load time. Only regular `.html`
files are loaded. A symbolic-link or junction root is refused. Any reparse
point under the tree is refused. Ancestors of the root are checked for
symlinks and Windows junctions. Leaf reads are size-bounded and reject a
final reparse point. A parent directory replaced by a link after that check
is the remaining race; it is not closed on every platform.

`parse` and `sources` enforce the same per-template size (1&nbsp;MiB), UTF-8,
name, and set-size (256) limits as `files`. Because an ordinary AhdCode
String treats `{...}` as interpolation, write template source that contains
`{{...}}` as a raw String:

```ahd
page: String := r"""
{{define "content"}}<h1>{{.title}}</h1>{{end}}
"""
set: TemplateSet := Templates.parse("pages/home.html", page)
```

## Render, layout, and output limits

```ahd
home: Function := (request: Request) -> Response {
    pages: Global TemplateSet
    return HTTP.html(pages.layout(
        "layouts/site.html"
        "pages/home.html"
        data
    ))
}
```

A Function that reads the module-root `pages` binding must declare
`pages: Global TemplateSet`.

Execution output is capped at 8&nbsp;MiB. Exceeding that raises
`TemplateError`; successful HTML is never truncated silently.

Serve the result with [`HTTP.html`](HTTP.md) or [`Web.markup`](WEB.md).
Those functions set `text/html; charset=utf-8` and do **not** escape again.

## Error pages

[`server.notFound`](HTTP.md) and [`server.internalError`](HTTP.md) take a
`Function(Request) -> Response`. The status written is always 404 or 500.
A `Location` header from the handler is dropped so a forced error page cannot
become a redirect. If the handler fails, the client receives the plain text
`Not Found` or `Internal Server Error`. The failure text, stack, and paths
are not sent. The other error page is not called.

## Live reload

`ahdcode dev` watches regular `.html` files under `templates/` beside the
entry file. Create, modify, delete, restore, and rename are noticed. Saving
one rebuilds and restarts the process; the debounce window coalesces a burst
into one rebuild. The watch does not follow symbolic links. `ahdcode run`
does not watch them. A program started without `ahdcode dev` does not inject
the reload script.

Templates are loaded once when `Templates.files` runs. Editing a file on
disk does not change an already running process until that process is
restarted.

## Ownership and limits

- At most 256 templates per set, 1&nbsp;MiB per template source.
- At most 1024 loaded `TemplateSet` handles per process. Sets are retained for
  the process lifetime; there is no silent eviction of a live set. Loading
  beyond that limit raises `TemplateError`.
- Data nesting depth at most 32; at most 10000 data nodes.
- Rendered output at most 8&nbsp;MiB.
- [`ahdcode package`](PACKAGING.md) builds desktop bundles. It does not copy
  a `templates/` or `public/` directory into the package. That packaging
  limitation is deferred; a template-based web app needs those directories
  beside the running process's working directory, or it should load templates
  with `parse` / `sources` instead.

## Turkish starter

`ahdcode init web basic` writes a small Turkish site that uses this module:
`templates/layouts/site.html`, `templates/components/nav.html`, pages for
home, about, 404, and 500, routes `/`, `/hakkinda`, and `/ornek-hata`, and
handlers that pass `Templates.object({...})` into `pages.layout`. See
[Web starters](WEB.md#16-v018-web-starters).
