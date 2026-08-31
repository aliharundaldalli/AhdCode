# Word standard module

[English] · [Türkçe](WORD_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Plot](PLOT.md)

Word creates immutable documents, writes real `.docx` packages, and reads a
small semantic subset of existing DOCX files. Import it explicitly:

```ahd
bring Word
from Word bring Document
from Word bring WordError
```

The canonical module identity is `builtin:Word`; a sibling `Word.ahd` cannot
shadow it. Word is implemented with the Go standard library and requires no
Office installation, external process, network access, or additional runtime
bundle.

Word creates and reads a bounded semantic DOCX subset. It is not an Office
clone and does not promise pixel-perfect round-trip preservation.

## Surface

```text
Word.new()                 -> Document
Word.read(path: String)    -> Document

Document.heading(text: String, level: Int) -> Document
Document.paragraph(
    text: String,
    align: String = "left",
    bold: Bool = false,
    italic: Bool = false,
    underline: Bool = false
) -> Document
Document.table(
    headers: List<String>,
    rows: List<List<String>>,
    merges: List<List<Int>> = [],
    align: String = "left"
) -> Document
Document.image(path: String, size: Pair<String, Real> = {}) -> Document
Document.pageBreak()       -> Document
Document.save(path: String) -> Nothing

Document.text()            -> String
Document.paragraphs()      -> List<String>
Document.headings()        -> List<String>
Document.tables()          -> List<List<List<String>>>

Document
WordError
```

The parameter labels above document argument order. `Document` operations are
positional-only in source code:

```ahd
document = document.paragraph("Important", "center", true, false, false)
```

`document.paragraph(text: "Important", bold: true)` is deliberately rejected.
All compiler-supplied type operations—such as String, List, Table, Chart,
Vector, and Matrix members—use the same positional-only dispatch mechanism.
Adding named arguments only for Document would redesign that shared API, so
Word follows the existing language rule. Module functions such as
`Word.read(path: "report.docx")` still use ordinary Function call rules.

## Immutable construction

`Word.new()` returns an empty Document. Every content operation returns a new
Document and leaves its receiver unchanged:

```ahd
base: Document := Word.new()
first: Document := base.paragraph("One")
second: Document := base.paragraph("Two")

write(base.text())
write(first.text())
write(second.text())
```

Headers, rows, merge descriptors, and image bytes are copied when an operation
runs. Mutating an input List later cannot change the Document, and deleting an
image source after `image()` cannot remove it from a later DOCX.

## Headings and paragraphs

Heading levels are `1` through `6`; another value raises `WordError`.
Paragraph alignment is exactly `"left"`, `"center"`, `"right"`, or
`"justify"`. Bold, italic, and underline apply to the paragraph's single text
run.

```ahd
document: Document := Word.new()
document = document.heading("Quarterly report", 1)
document = document.paragraph("Prepared offline.")
document = document.paragraph("Approved", "right", true, true, true)
document = document.pageBreak()
```

Text is XML-escaped, prohibited XML 1.0 control characters are discarded, and
Unicode is preserved.

## Tables and merges

Every row must have exactly the same number of cells as `headers`, and at least
one column is required. Table alignment is `"left"`, `"center"`, or
`"right"`.

Each merge descriptor has four Int values:

```text
[row, column, rowSpan, columnSpan]
```

Coordinates are zero-based and row `0` is the header row. A merge must remain
inside the table, have positive spans, cover more than one cell, and not
overlap another merge. Horizontal spans use WordprocessingML `gridSpan`;
vertical spans use `vMerge` restart/continuation cells.

```ahd
document = document.table(
    ["Region", "Q1", "Q2"]
    [
        ["North", "10", "12"]
        ["South", "8", "11"]
    ]
    [
        [0, 0, 1, 3]
        [1, 0, 2, 1]
    ]
    "center"
)
```

Malformed descriptors, negative coordinates, zero spans, 1x1 merges,
out-of-bounds regions, overlaps, and row-width mismatches raise `WordError`
before a package is written. This row-width guarantee holds for every
Document, including one produced by `Word.read`: saving never silently
truncates a ragged table, so a defensive `WordError` is the only possible
outcome if an internal table is ever not rectangular.

## Images

Word embeds PNG and JPEG bytes immediately. `size` accepts only `"width"`
and `"height"`, measured in centimeters:

```ahd
document = document.image("chart.png")
document = document.image("chart.png", {"width": 12.0})
document = document.image("photo.jpg", {"height": 6.0})
document = document.image("logo.png", {"width": 4.0, "height": 3.0})
```

One dimension preserves the natural aspect ratio; both dimensions use the
explicit box; no dimensions use the natural pixel size at 96 DPI. Dimensions
must be positive. A missing file, undecodable data, unsupported format, key,
or dimension raises `WordError`.

Plot integrates explicitly through a file boundary:

```ahd
chart.save("scores.png")
document = document.image("scores.png", {"width": 14.0})
```

## Saving

`save(path)` accepts a `.docx` destination and returns `Nothing`; do not assign
its result:

```ahd
document.save("report.docx")
```

The package contains deterministic relationship IDs, media names, styles, and
ZIP member order. Saving identical Document content twice produces identical
bytes. Word assembles and validates the complete package before atomically
publishing it in the destination directory, so a failed save does not replace
an existing destination.

## Reading and accessors

`Word.read(path)` recovers paragraph text, Heading 1–6 text, and table cell
text from `word/document.xml`. Tabs and line breaks inside a paragraph become
`\t` and `\n`. Unsupported formatting, custom styles, headers, footers,
comments, page geometry, images, and unknown relationships are ignored rather
than reproduced.

`Word.read` is semantic, not formatting-preserving. A horizontally merged
cell (`gridSpan`) is expanded back to one empty logical column per spanned
position, at the position the span occurred, so a merged header still lines
up with the unmerged columns beneath it and every table row stays
rectangular. A vertically merged cell (`vMerge`) is *not* reconstructed as a
merge on read; its continuation cells simply read back as empty text in their
own row. Merged-cell visual topology may therefore be flattened after a
read/save cycle, but logical column position and cell text are always
preserved — silent data loss is never acceptable, and reading never drops a
cell's text to make a table's rows agree in width.

```ahd
loaded: Document := Word.read("report.docx")
write(loaded.text())
write(loaded.headings())
write(loaded.paragraphs())
write(loaded.tables())
```

`text()` joins headings and paragraphs with newline characters. `tables()`
returns each table as its logical rows, with the first row at index `0`; a
row recovered from a merged cell may contain empty-String columns where the
merge occurred, but every row in a table has the same width. Accessor Lists
are fresh snapshots.

Reading is bounded: the implementation limits archive bytes, entry count,
individual and total uncompressed sizes, and compression ratio. It rejects
absolute/traversal paths, duplicate members, invalid ZIP data, missing or
malformed `word/document.xml`, oversized content, and unreasonable
compression with `WordError`. It never follows relationships or accesses the
network.

## Errors

`WordError` covers Word-specific validation, image, DOCX packaging, save, and
read failures:

```ahd
attempt {
    Word.read("missing.docx")
}
except WordError as error {
    write(error.message)
}
```

Static argument count and type mistakes remain compiler diagnostics; they do
not become runtime `WordError` values.
