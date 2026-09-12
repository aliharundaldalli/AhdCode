# PDF standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [Latex](LATEX.md) · [Word](WORD.md) · [Excel](EXCEL.md) · [QR](QR.md) · [Barcode](BARCODE.md)

PDF creates immutable documents and renders real `.pdf` files offline. Import
it explicitly:

```ahd
bring PDF
from PDF bring PDFDocument
from PDF bring PDFError
```

The canonical module identity is `builtin:PDF`; a sibling `PDF.ahd` cannot
shadow it. PDF text is always ordinary text: every String a caller supplies is
escaped before it reaches the renderer, so `PDF` never exposes raw LaTeX/TeX
injection. Use [`Latex`](LATEX.md) directly when actual LaTeX source control
is the goal.

## Surface

```text
PDF.new()                              -> PDFDocument
PDF.fromWord(document: Word.Document)  -> PDFDocument
PDF.fromExcel(workbook: Excel.Workbook) -> PDFDocument

PDFDocument.heading(text: String, level: Int) -> PDFDocument
PDFDocument.paragraph(
    text: String,
    align: String = "left",
    bold: Bool = false,
    italic: Bool = false,
    underline: Bool = false
) -> PDFDocument
PDFDocument.table(
    headers: List<String>,
    rows: List<List<String>>,
    align: String = "left"
) -> PDFDocument
PDFDocument.image(
    path: String,
    size: Pair<String, Real> = {},
    transform: Pair<String, Real> = {}
) -> PDFDocument
PDFDocument.pageBreak()                -> PDFDocument
PDFDocument.qr(value: String, size: Real = 3.0, level: String = "M", align: String = "center") -> PDFDocument
PDFDocument.barcode(
    kind: String,
    value: String,
    width: Real = 8.0,
    height: Real = 2.0,
    align: String = "center"
) -> PDFDocument
PDFDocument.link(text: String, url: String, align: String = "left") -> PDFDocument
PDFDocument.bookmark(title: String, level: Int = 1) -> PDFDocument

PDFDocument.layout(
    paper: String,
    landscape: Bool = false,
    pageSize: Pair<String, Real> = {},
    margins: Pair<String, Real> = {}
) -> PDFDocument
PDFDocument.header(left: String, center: String = "", right: String = "") -> PDFDocument
PDFDocument.footer(left: String, center: String = "", right: String = "") -> PDFDocument
PDFDocument.pageNumbers(align: String = "center", total: Bool = false) -> PDFDocument
PDFDocument.metadata(
    title: String,
    author: String = "",
    subject: String = "",
    keywords: List<String> = [],
    creator: String = ""
) -> PDFDocument
PDFDocument.save(path: String)         -> Nothing

PDFDocument
PDFError
```

`PDFDocument` operations are positional-only, the same convention `Document`,
`String`, `List`, `Table`, and `Chart` already use:

```ahd
doc = doc.paragraph("Important", "center", true, false, false)
```

`qr`, `barcode`, `link`, `bookmark`, `layout`, `header`, `footer`,
`pageNumbers`, `metadata`, and the `image` transform are new in v1.3.0.

## Immutable construction

`PDF.new()` returns an empty PDFDocument. Every operation returns a new
PDFDocument and leaves its receiver unchanged; mutating an input List later,
or deleting an image source after `image()`, cannot change a PDFDocument
already built:

```ahd
base: PDFDocument := PDF.new()
first: PDFDocument := base.paragraph("One")
second: PDFDocument := base.paragraph("Two")
```

Content operations — `heading`, `paragraph`, `table`, `image`, `pageBreak`,
`qr`, `barcode`, `link`, and `bookmark` — appear in the order they are added.
`layout`, `header`, `footer`, `pageNumbers`, and `metadata` are document
settings: they apply to the whole document wherever they are called, and when
one is called more than once, the last call wins.

## Page layout

Without `layout`, every page is A4 portrait with 2.54 cm margins, exactly as
before v1.3.0. (`Latex.document` keeps its own default, US Letter; the two
modules do not share a default page.)

`layout(paper, landscape, pageSize, margins)` changes the page for the whole
document:

| `paper` | Size (cm) |
|---|---|
| `"A3"` | 29.7 x 42 |
| `"A4"` | 21 x 29.7 |
| `"A5"` | 14.8 x 21 |
| `"Letter"` | 21.59 x 27.94 |
| `"Legal"` | 21.59 x 35.56 |
| `"Custom"` | `pageSize` |

- `paper` is case-sensitive.
- `landscape: true` turns the page sideways.
- `pageSize` is used only with `"Custom"`, which requires both `"width"` and
  `"height"`, in centimeters.
- `margins` accepts any of `"top"`, `"right"`, `"bottom"`, and `"left"`, in
  centimeters; a side you leave out keeps 2.54 cm.
- Every length must be greater than 0 and at most 1000, and the margins must
  leave room for content.

```ahd
doc = doc.layout("A5", true)
doc = doc.layout("Letter", false, {}, {"top": 3.0, "bottom": 2.0})
doc = doc.layout("Custom", false, {"width": 10.0, "height": 6.0}, {"top": 0.5, "right": 0.5, "bottom": 0.5, "left": 0.5})
```

## Headers, footers, and page numbers

`header(left, center, right)` and `footer(left, center, right)` put plain text
in three regions of every page. The text is escaped like all PDF text and must
be a single line.

A PDF shows the page number centered in the footer by default. A header alone
keeps it. A footer replaces it, and `pageNumbers(align, total)` places it in one
footer region — `"3"`, or `"3 / 12"` with `total: true`. That region of the
footer must be empty; otherwise `save` raises `PDFError`. `footer("")` removes
the footer, page number included.

```ahd
doc = doc.header("AhdCode Analytics", "Quarterly Report", "Q3 2026")
doc = doc.footer("Confidential").pageNumbers("right", true)
```

For a logo or other graphics in a header, custom wording such as
"Page 3 of 12", or content placed at an exact position on the page, use
[`Latex.header`, `Latex.footer`, and `Latex.place`](LATEX.md#professional-documents-v130).

## Headings, paragraphs, and text safety

Heading levels are `1` through `6`; another value raises `PDFError`. Paragraph
alignment is exactly `"left"`, `"center"`, `"right"`, or `"justify"`.

```ahd
doc: PDFDocument := PDF.new()
doc = doc.heading("Quarterly report", 1)
doc = doc.paragraph("Prepared offline.")
doc = doc.paragraph("Approved", "right", true, true, true)
doc = doc.pageBreak()
```

Every String reaching `PDFDocument` — heading and paragraph text, table
cells, header and footer text, link text, bookmark titles, and document
properties — is escaped before it becomes renderer source. A String containing
`\ { } $ & # % _ ^ ~` appears as ordinary text; none of it is ever interpreted
as a rendering command. PDF has no raw-content or raw-markup escape hatch.

## Table

Every row must have exactly the same number of cells as `headers`, and at
least one column is required. Alignment is `"left"`, `"center"`, or
`"right"`, applied to every column. There is no per-cell merge or span; a
ragged row raises `PDFError` before anything is rendered — never padded,
truncated, or repaired.

```ahd
doc = doc.table(
    ["Region", "Q1", "Q2"]
    [
        ["North", "10", "12"]
        ["South", "8", "11"]
    ]
    "center"
)
```

## Image

PDF accepts PNG, JPEG, and SVG images. PNG and JPEG bytes are embedded
immediately, the same way `Word.image` does, so a PDFDocument never depends on
the source file surviving or on the working directory staying the same. An SVG
is converted to vector drawing commands as soon as `image()` is called, so an
unsupported SVG raises `PDFError` right there; see
[SVG images](#svg-images).

`size` accepts only `"width"` and `"height"`, measured in centimeters:

```ahd
doc = doc.image("chart.png")
doc = doc.image("chart.png", {"width": 12.0})
doc = doc.image("logo.svg", {"width": 4.0, "height": 3.0})
```

One dimension preserves the natural aspect ratio; both dimensions use the
explicit box; no dimensions use the image's natural size. Dimensions must be
positive. A missing file, undecodable data, unsupported format, key, or
dimension raises `PDFError`.

`transform` (v1.3.0) accepts these keys:

| Key | Meaning | Range |
|---|---|---|
| `"rotation"` | degrees, counterclockwise | -360 to 360 |
| `"opacity"` | 0 is invisible, 1 is opaque | 0 to 1 |
| `"trimLeft"`, `"trimTop"`, `"trimRight"`, `"trimBottom"` | centimeters removed from that edge of the sized image | 0 to 1000 |

The image is sized first, then trimmed, rotated, and faded. A trim that would
remove the whole width or height raises `PDFError`.

```ahd
doc = doc.image("photo.jpg", {"width": 8.0}, {"trimTop": 1.0, "trimBottom": 1.0})
doc = doc.image("stamp.png", {"width": 4.0}, {"rotation": 12.0, "opacity": 0.6})
```

Images flow like words: two images added one after the other share a line.
Add a paragraph or a page break between them to place one below the other.

### SVG images

An SVG becomes vector drawing commands inside the PDF. It is never rasterized,
and no browser, Inkscape, or other external converter is involved. Shapes,
paths (including arcs), groups with transforms, fills, strokes, dashes, and
opacity are supported. Text, gradients, patterns, masks, clipping, filters,
markers, embedded images, scripts, animation, and external references are
rejected with a `PDFError` that names the unsupported feature, instead of
being dropped or approximated. The full list is in
[Latex's SVG assets](LATEX.md#svg-assets-v130); PDF and Latex use the same
converter.

## QR codes and barcodes

`qr(value, size, level, align)` and `barcode(kind, value, width, height,
align)` (v1.3.0) draw vector symbols from the same encoders as the
[QR](QR.md) and [Barcode](BARCODE.md) modules, so a value produces the same
modules everywhere. Sizes are centimeters with the quiet zones included, and
`align` is `"left"`, `"center"`, or `"right"`. `kind` is `"Code128"`,
`"EAN13"`, or `"UPCA"`. `bring QR` or `bring Barcode` is not needed.

```ahd
doc = doc.qr("https://ahdcode.org/verify?id=42")
doc = doc.qr("https://ahdcode.org/verify?id=42", 2.5, "Q", "right")
doc = doc.barcode("EAN13", "590123412345", 6.0, 2.0, "left")
doc = doc.barcode("Code128", "AHD-2026-0042")
```

An invalid value — an empty QR value, a wrong EAN-13 check digit, a non-ASCII
Code 128 character — raises `PDFError` when the operation is called, with the
same message the QR and Barcode modules give.

## Links, bookmarks, and document properties

`link(text, url, align)` (v1.3.0) adds one line of clickable text. `url` must
start with `https://`, `http://`, or `mailto:`; any other scheme — including
`javascript:` and `file:` — and control characters raise `PDFError`. Special
characters are encoded safely, so the link opens exactly the URL given. Links
are drawn as plain text, without a colored box.

`bookmark(title, level)` adds an entry to the PDF outline (the viewer's
sidebar) that jumps to the position where it is added. Level 1 is a top-level
entry; levels 2 to 4 nest under the nearest preceding entry of a smaller
level. The outline holds exactly the `bookmark` calls: headings do not add
entries of their own.

`metadata(title, author, subject, keywords, creator)` sets the document
properties PDF viewers show. Each keyword must be non-empty and free of
commas, and every value must be a single line.

```ahd
doc = doc.bookmark("Summary").heading("Summary", 1)
doc = doc.link("ahdcode.org", "https://ahdcode.org", "center")
doc = doc.metadata("Quarterly Report", "AhdCode Analytics", "Q3 results", ["report", "2026"], "AhdCode")
```

## Saving

`save(path)` accepts a `.pdf` destination and returns `Nothing`:

```ahd
doc.save("report.pdf")
```

`save` builds the PDFDocument's content into a LaTeX body internally (never
exposed), compiles it through AhdCode's existing offline Tectonic renderer —
the same low-level engine invocation, secure temporary workspace, and
atomic same-directory publish `Latex.pdf` uses — and verifies the `%PDF-`
signature before publishing. A failed compile never replaces an existing
destination. PDF never produces a `.tex` sidecar; use
[`Latex.pdf(source, output, "tex")`](LATEX.md#compiling) when the exact LaTeX
source is also wanted.

The renderer packages for links, headers, vector codes, SVG images, and image
transforms are included only when a document uses them. A document built only
from operations that existed in v1.2.0 renders exactly as it did in v1.2.0.

## Word and Excel conversion

`PDF.fromWord` and `PDF.fromExcel` are semantic conversions of another
module's own typed document — not Office/Excel print emulation, and not a
DOCX/XLSX-to-PDF pixel-perfect renderer. Neither reads or writes the source
document; both leave it completely unchanged.

### `PDF.fromWord`

Preserves headings, paragraph text/alignment/bold/italic/underline, table
content, images (converted from Word's embedded bytes and EMU dimensions),
and page breaks. A table's merge geometry has no PDF equivalent and is
dropped; the table's cell text is fully preserved either way. If the source
`Document` came from `Word.read`, it already carries whatever `Word.read`
itself could recover — see [Word's reading contract](WORD.md#reading-and-accessors)
— `PDF.fromWord` cannot recover formatting `Word.read` already discarded.

```ahd
wordDocument := Word.new()
wordDocument = wordDocument.heading("Report", 1)
wordDocument = wordDocument.paragraph("Hello")

pdfDocument := PDF.fromWord(wordDocument)
pdfDocument.save("report.pdf")
```

### `PDF.fromExcel`

Every Sheet becomes a heading (the Sheet name) followed by a table over its
used range, in Workbook order. The used range's first row becomes the table
header and the remaining rows become the body — a presentational choice only;
Excel workbooks have no formal header-row concept, and no cell is ever
dropped either way. String/Int/Real/Bool cells are displayed deterministically,
Blank stays empty, and a Formula cell shows its formula *source text* — never
a fabricated or cached result, because AhdCode does not evaluate Excel
formulas. A merge's non-anchor cells are already guaranteed Blank by Excel's
own model, so the plain grid never loses a value; PDF does not attempt
multi-column cell spanning in the output table. A zero-Sheet Workbook raises
`PDFError`. A Sheet whose used range is wider than 10 columns also raises
`PDFError` rather than silently dropping columns or attempting a best-effort
multi-page layout.

```ahd
book := Excel.new()
book = book.addSheet("Results")

sheet := book.sheet("Results")
sheet = sheet.setCell(1, 1, Excel.fromString("Name"))
sheet = sheet.setCell(1, 2, Excel.fromInt(91))
book = book.withSheet(sheet)

pdf := PDF.fromExcel(book)
pdf.save("results.pdf")
```

Both conversions return an ordinary PDFDocument, so `layout`, `header`,
`footer`, `pageNumbers`, `metadata`, and the other operations apply to them
too.

## Renderer

`PDF` shares its low-level renderer with `Latex.pdf`: the same staged offline
Tectonic engine, the same `--untrusted` invocation, the same secure temporary
workspace, and the same atomic publish. See [Latex's offline/security/output
safety sections](LATEX.md#offline-by-construction) for the full contract —
none of it differs for PDF. A native build that uses `PDF` requires the same
staged `libexec/ahdcode/latex/` resources `Latex` requires; see
[`package-latex`](LATEX.md#compiling).

## Errors

`PDFError` covers PDF-specific validation — heading levels, alignments, table
shape, page layout, header and footer text, URLs, bookmark levels, document
properties, QR and barcode values, images, and SVG content — as well as
rendering and save failures:

```ahd
attempt {
    doc.save("report.txt")
}
except PDFError as error {
    write(error.message)
}
```

Static argument count and type mistakes remain compiler diagnostics; they do
not become runtime `PDFError` values.

## Not in this version

PDF reading/parsing, editing, forms, signatures, encryption, merging,
splitting, per-cell table merges, OCR, HTML/URL/browser rendering,
JavaScript, raw LaTeX/TeX, graphics or custom page-number wording inside
headers and footers, and absolute positioning are not part of v1.3.0. Use
[`Latex`](LATEX.md) for the last three.
