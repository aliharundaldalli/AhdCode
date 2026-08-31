# XML standard module

[English] · [Türkçe](XML_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [JSON](JSON.md)

`XML` is the compiler-registered `builtin:XML` module. It is explicit and a
sibling `XML.ahd` cannot shadow it:

```ahd
bring XML
from XML bring XMLNode
from XML bring XMLDocument
from XML bring XMLError
```

XML is structured data support, not an XML standards suite: it is a small,
bounded node model, not a full DOM, and there is no `Any` or dynamic escape
hatch anywhere in it.

## The node model

`XMLNode` represents exactly two kinds:

```text
Element
Text
```

This is deliberate: a small closed set covers ordered, mixed text/element
content without a large DOM class hierarchy. There is no separate public
type for Comment, CDATA, ProcessingInstruction, Doctype, or Entity — parsing
ignores comments and processing instructions, and a `CDATA` section is
recovered as an ordinary `Text` node, the same as any other text.

`XMLDocument` wraps exactly one root `XMLNode`, which must be an `Element`.
`XMLNode` and `XMLDocument` are both immutable: no accessor exposes a
mutable alias into a receiver, and every accessor that itself returns node
data returns a fresh, independent value.

## Construction

```text
XML.text(value: String)                                        -> XMLNode
XML.element(name: String, attributes: Pair<String,String>, children: List<XMLNode>) -> XMLNode
XML.document(root: XMLNode)                                     -> XMLDocument
```

`XML.document(root)` requires an `Element` root; passing a `Text` node
raises `XMLError`. No String concatenation is needed for ordinary
construction:

```ahd
student: XMLNode := XML.element(
    "student"
    {"id": "42"}
    [
        XML.element("name", {}, [XML.text("Ali")])
        XML.element("score", {}, [XML.text("91")])
    ]
)
document: XMLDocument := XML.document(student)
```

`XML.element` constructs unqualified (no-namespace) elements only —
producing a namespace-qualified element is only possible by parsing
existing namespace-qualified XML, not by construction, in this release (see
[Namespaces](#namespaces)).

## Parsing and reading

```text
XML.parse(source: String) -> XMLDocument
XML.read(path: String)    -> XMLDocument
```

Parsing uses Go's standard library `encoding/xml` decoder, the same
token-walking approach `Word.read` uses for DOCX — not a hand-written
grammar. A document must have exactly one root element: no root, more than
one top-level element, or non-whitespace content before/after the root all
raise `XMLError`. A duplicate attribute on one element is invalid XML and
raises `XMLError`. Child order and mixed text/element order are always
preserved exactly as parsed.

Because AhdCode's ordinary quoted strings interpret `{` and `}` as
interpolation delimiters and XML text is full of `<`, `>`, and `"`, write
literal XML source as a raw String:

```ahd
document: XMLDocument := XML.parse(r'<a id="1"><b>text</b></a>')
```

Parsing is bounded: input larger than 8&nbsp;MiB, and element nesting past
256 levels, both raise `XMLError` before completing.

## XMLNode accessors

```text
kind()      -> String
name()      -> String
namespace() -> String
text()      -> String

attribute(name: String) -> String?
attributes()             -> Pair<String, String>

children() -> List<XMLNode>
elements() -> List<XMLNode>

XMLDocument.root() -> XMLNode
```

`kind()` returns exactly `"Element"` or `"Text"`, and never raises.

`name()`, `namespace()`, `attribute()`, `attributes()`, `children()`, and
`elements()` are `Element`-only: calling any of them on a `Text` node raises
`XMLError`, the same uniform wrong-kind rule the JSON module's `JSONValue`
uses.

`text()` is the one member valid on both kinds, with kind-dependent
semantics: for a `Text` node it is that node's own content; for an
`Element` it is the concatenation of that element's *direct* `Text`
children, in document order — nested descendant text is not flattened in.

```ahd
p: XMLNode := XML.parse(r'<p>one<b>two</b>three</p>').root()
write(p.text())     -- "onethree" (direct Text children only)
```

`attribute(name)` returns `String?`: `null` means the attribute is absent.
`attributes()` returns every attribute as an insertion-ordered
`Pair<String, String>`.

`children()` returns every direct `Element`/`Text` child in document order;
`elements()` returns only the `Element` children, still in order.

`XMLDocument.root()` returns the document's root `XMLNode` — the way back
from a parsed/read `XMLDocument` into its node tree.

## Serialization

```text
XML.stringify(document: XMLDocument, pretty: Bool = false) -> String
XML.write(document: XMLDocument, path: String, pretty: Bool = false) -> Nothing
```

Both modes escape `&`, `<`, `>` in text and additionally `"`, and the three
whitespace control characters, in attribute values; both produce valid,
well-formed XML.

Compact output (`pretty = false`, the default) always round-trips exactly:
`XML.parse(XML.stringify(document))` describes the identical tree.

Pretty output uses a fixed two-space indent, but only inserts that
indentation between a run of purely-`Element` children. An element whose
content is text-only or mixed text/element is always rendered with its
children inline, in both modes, because inserting whitespace next to a
`Text` child would add content that was not in the original tree. This is
the same well-known trade-off every XML pretty-printer makes — pretty
output is a human-readability convenience, and compact output is the one
guaranteed to be lossless.

`XML.write` stages its output and publishes it atomically, the same
temp-file-then-rename convention Word and JSON use: a failed write never
disturbs a file that was already at the destination.

## Attributes

For the v0.1.17 surface, ordinary unqualified attributes are a
`Pair<String, String>`, insertion order preserved. A `Pair` cannot itself
carry a duplicate key, so `XML.element`'s `attributes` argument can never
produce a duplicate — but raw parsed XML can, and a duplicate attribute in
source text raises `XMLError`. Numeric or other non-String attribute values
are never coerced automatically; convert explicitly with `str(...)` first.

## Namespaces

XML namespace support in this release is intentionally bounded:

- **Parsing is namespace-aware.** `XML.parse`/`XML.read` resolve each
  element's namespace prefix to its full URI (via Go's `encoding/xml`
  decoder) and expose it through `namespace()`; an element with no
  namespace reports `""`.
- **The exact original prefix spelling is not preserved.** Only the
  resolved URI is kept — this release does not promise byte-for-byte
  prefix round-trip.
- **Construction is unqualified only.** `XML.element` has no namespace
  parameter; there is no way to *construct* a namespace-qualified element
  from AhdCode source in this release, only to parse one from existing XML.
- **Attributes stay unqualified.** The small `Pair<String, String>`
  attribute surface does not carry per-attribute namespace qualification.

This is a deliberate boundary, not an oversight: a full namespace-authoring
API (QName types, prefix bindings, per-attribute namespaces) would add
significant surface for a release scoped to structured data support.

## Security

`XML.parse`/`XML.read` never access the network and never execute anything.
Go's `encoding/xml` decoder does not fetch or process an external DTD
subset and does not expand custom general entities by default — an
undefined entity reference is a parse error, not a substitution — so
classic XXE and billion-laughs attacks are not reachable without additional
code that this module does not add. Comments, processing instructions, and
DOCTYPE declarations are recognized as bytes but never interpreted as an
execution mechanism.

## Errors

`XMLError` derives directly from `Error` and covers every XML-specific
failure: malformed input, no root element, multiple root elements, an
invalid (`Text`) `XMLDocument` root, a duplicate attribute, a wrong-kind
node method, depth/size limits, and a missing/unreadable/unwritable file.

```ahd
attempt {
    XML.parse("<a><b></a></b>")
} except XMLError as error {
    write(error.message)
}
```

## Non-goals

XML is structured data support, not an XML standards suite. This release
has no XPath, no XSLT, no XML Schema/XSD validation, no DTD processing, no
external entity resolution, no full DOM mutation API, no dedicated
Comment/ProcessingInstruction API, no namespace-prefix management
framework, no canonical XML (C14N), no digital signatures, no SOAP
framework, and no HTML parsing.
