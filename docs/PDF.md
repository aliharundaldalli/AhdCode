# PDF standard module

[English] · [Türkçe](PDF_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Latex](LATEX.md) · [Word](WORD.md) · [Excel](EXCEL.md)

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
PDFDocument.image(path: String, size: Pair<String, Real> = {}) -> PDFDocument
PDFDocument.pageBreak()                -> PDFDocument
PDFDocument.save(path: String)         -> Nothing

PDFDocument
PDFError
```

`PDFDocument` operations are positional-only, the same convention `Document`,
`String`, `List`, `Table`, and `Chart` already use:

```ahd
doc = doc.paragraph("Important", "center", true, false, false)
```

## Immutable construction

`PDF.new()` returns an empty PDFDocument. Every content operation returns a
new PDFDocument and leaves its receiver unchanged; mutating an input List
later, or deleting an image source after `image()`, cannot change a
PDFDocument already built:

```ahd
base: PDFDocument := PDF.new()
first: PDFDocument := base.paragraph("One")
second: PDFDocument := base.paragraph("Two")
```

## Page layout

v0.1.20 uses a deliberately fixed layout: A4, portrait, a 2.54cm margin (the
same default `Latex.document()` already uses). There is no page-size,
orientation, or margin configuration in this release.

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

Every String reaching `PDFDocument` — heading text, paragraph text, table
cell text — is escaped before it becomes renderer source. A String containing
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

PDF embeds PNG and JPEG bytes immediately, the same way `Word.image` does, so
a PDFDocument never depends on the source file surviving or on the working
directory staying the same. `size` accepts only `"width"` and `"height"`,
measured in centimeters:

```ahd
doc = doc.image("chart.png")
doc = doc.image("chart.png", {"width": 12.0})
doc = doc.image("logo.png", {"width": 4.0, "height": 3.0})
```

One dimension preserves the natural aspect ratio; both dimensions use the
explicit box; no dimensions use the renderer's own natural sizing. Dimensions
must be positive. A missing file, undecodable data, unsupported format, key,
or dimension raises `PDFError`.

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

## Renderer

`PDF` shares its low-level renderer with `Latex.pdf`: the same staged offline
Tectonic engine, the same `--untrusted` invocation, the same secure temporary
workspace, and the same atomic publish. See [Latex's offline/security/output
safety sections](LATEX.md#offline-by-construction) for the full contract —
none of it differs for PDF. A native build that uses `PDF` requires the same
staged `libexec/ahdcode/latex/` resources `Latex` requires; see
[`package-latex`](LATEX.md#compiling).

## Errors

`PDFError` covers PDF-specific validation, image, rendering, and save
failures:

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

PDF reading/parsing, editing, annotations, forms, signatures, encryption,
merging, splitting, page-layout configuration, per-cell table merges, OCR,
HTML/URL/browser rendering, and JavaScript are not part of v0.1.20.
