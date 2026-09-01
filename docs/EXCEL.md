# Excel

[English] · [Türkçe](EXCEL_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Lists](LISTS.md) · [KeyValue](KEYVALUE.md) · [Data](DATA.md) · [JSON](JSON.md)

`bring Excel` provides a strongly typed, immutable, offline XLSX layer. It
creates and semantically reads real Excel-compatible `.xlsx` ZIP/XML packages
using the native runtime; Microsoft Excel, LibreOffice, Python, helper
executables, and network access are not required.

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet

book: Workbook := Excel.new().addSheet("Students")
sheet: Sheet := book.sheet("Students")
sheet = sheet.setCell(1, 1, Excel.fromString("Name"))
sheet = sheet.setCell(1, 2, Excel.fromString("Score"))
sheet = sheet.setCell(2, 1, Excel.fromString("Ali"))
sheet = sheet.setCell(2, 2, Excel.fromInt(91))
book = book.withSheet(sheet)
book.save("students.xlsx")
```

## Types and immutability

The public types are `Workbook`, `Sheet`, `Cell`, `Range`, `CellStyle`, and
`ExcelError`. `Workbook` and `Sheet` transformations return independent new
values. A Sheet obtained with `book.sheet(name)` has no hidden back-reference;
after editing it, explicitly reinsert it with `book.withSheet(sheet)`.

`Excel.new()` is empty and does not invent `Sheet1`. `addSheet` appends a new
blank Sheet; `withSheet` replaces an existing Sheet with the same exact name
without changing its position. Unknown or duplicate names raise `ExcelError`.
Names are case-insensitively unique, contain at most 31 Unicode characters,
and cannot contain `: \ / ? * [ ]` or unsafe XML controls.

```text
Excel.new()                    -> Workbook
Excel.read(path)              -> Workbook
Workbook.addSheet(name)       -> Workbook
Workbook.sheet(name)          -> Sheet
Workbook.withSheet(sheet)     -> Workbook
Workbook.sheets()             -> List<String>
Workbook.sheetCount()         -> Int
Workbook.save(path)           -> Nothing
```

Saving a zero-Sheet Workbook or a non-`.xlsx` destination raises
`ExcelError`.

## Typed Cells

A Cell has exactly one of `Blank`, `String`, `Int`, `Real`, `Bool`, or
`Formula`. There is no `Any`, dynamic cell value, or scalar-to-Cell coercion.

```text
Excel.blank()
Excel.fromString(value)
Excel.fromInt(value)
Excel.fromReal(value)
Excel.fromBool(value)
Excel.formula(expression)

Cell.kind()       Cell.isBlank()
Cell.string()     Cell.int()       Cell.real()
Cell.bool()       Cell.formula()
```

Wrong-kind access raises `ExcelError`. `real()` safely widens an `Int`; `int()`
never narrows a `Real`. Non-finite Reals, unsafe XML text, and oversized
Strings are rejected rather than converted or truncated.

A String beginning with `=` remains literal text:

```ahd
safeText := Excel.fromString("=SUM(A1:A100)") // String
formula := Excel.formula("=SUM(A1:A100)")    // Formula
```

Formula text must begin with `=`, contain text after it, and fit Excel's
formula length. AhdCode stores and XML-escapes the expression but does not
parse, type-check, calculate, execute links, or fetch network content. Reading
returns the leading `=` and ignores cached results.

Saved workbooks mark every Formula Cell for recalculation: the generated XLSX
carries a placeholder cached value and workbook calculation metadata
(`fullCalcOnLoad`, `forceFullCalc`, a real `calcId`) so Excel, Numbers, and
other spreadsheet applications recompute and display the real result as soon
as the file is opened, without the user pressing F9 or re-entering the
formula. The placeholder is XLSX interoperability metadata only; AhdCode never
computes it and `Cell.formula()` never returns it.

## Coordinates, ranges, and bulk writes

Excel coordinates are deliberately 1-based: `(1, 1)` is `A1`. Rows are
`1..1048576`; columns are `1..16384` (`XFD`). Values outside those limits
raise `ExcelError`.

```text
Sheet.name()                                      -> String
Sheet.cell(row, column)                           -> Cell
Sheet.setCell(row, column, value)                 -> Sheet
Sheet.range(startRow, startColumn, endRow, endColumn) -> Range
Sheet.setRow(row, startColumn, values)            -> Sheet
Sheet.setRange(range, values)                     -> Sheet
Sheet.cells(range)                                -> List<List<Cell>>
Sheet.usedRange()                                 -> Range?
```

Unset coordinates read as explicit Blank Cells. `setRow` accepts a
`List<Cell>`. `setRange` requires exactly the Range's row count and exactly its
column count in every row; it never pads, truncates, repairs ragged data, or
partially changes the source Sheet. `cells` returns a fresh exact rectangle,
including Blank coordinates.

`Range` exposes `startRow`, `startColumn`, `endRow`, `endColumn`, `rowCount`,
`columnCount`, and `address`. `usedRange()` is `null` for an empty Sheet and
otherwise covers non-Blank, formula, styled, and merged cells. Row/column
dimensions alone do not expand it.

## Merges and styles

`sheet.merge(range)` preserves the top-left anchor. Every other covered Cell
must already be Blank, and merges cannot overlap. A non-Blank covered Cell
raises `ExcelError`; no value is discarded or moved. Writes into a merged
non-anchor coordinate are also rejected. `merges()` returns a fresh ordered
`List<Range>`.

`Excel.style()` creates an immutable patch. Each operation specifies only one
property, so applying a bold-only patch preserves an existing fill. Explicit
`bold(false)` disables bold; an unspecified bold property leaves it alone.

```text
bold(Bool)              italic(Bool)          underline(Bool)
fontSize(Real)          textColor("#RRGGBB") fillColor("#RRGGBB")
horizontal(String)      vertical(String)      wrap(Bool)
numberFormat(String)    border(style, color)
```

Horizontal values are `left`, `center`, `right`; vertical values are `top`,
`center`, `bottom`. Border styles are `none`, `thin`, `medium`, `thick`,
`dashed`, `dotted`, and `double`. Colors use uppercase `#RRGGBB`. Number
formats such as `General`, `0`, `0.00`, `0%`, and `yyyy-mm-dd` are explicit
Excel format strings; they do not change an Int/Real Cell into a date,
currency, or percentage type.

Use `sheet.style(range, patch)`, `sheet.columnWidth(column, width)`, and
`sheet.rowHeight(row, height)`. Values must be positive finite Reals within
the supported Excel limits.

## XLSX reading, writing, and security

Generated packages use deterministic sheet, relationship, and style IDs,
deterministic ZIP member order, and inline Strings. Complete generated-package
validation happens before an atomic destination replacement, so a failed save
does not destroy an existing file.

`Excel.read` supports ordered Sheets, shared and inline Strings, Blank/Int/
Real/Bool/Formula Cells, merges, the documented style subset, and dimensions.
Numeric lexical spelling decides `Int` versus `Real`: `91` is Int, while
`1.0`, `3.14`, and `1e3` are Real. Style formats do not infer dates or other
types. An external application may rewrite `1.0` as `1`, in which case that
lexical intent is no longer recoverable.

Reading is semantic, not pixel-perfect. Unsupported presentation features
are not preserved. Unsupported Cell types or advanced shared/array formulas
whose content cannot be reconstructed safely raise `ExcelError` rather than
silently losing a value or formula. Input is bounded by archive, entry,
uncompressed-size, and compression-ratio limits; duplicate or traversing ZIP
members, malformed XML, DTDs, broken internal relationships, and external
worksheet/shared-string/style targets are rejected. External links are never
opened and no network request is made.

## Composition and scope

Use `Lists.transpose` for `List<List<Cell>>` structure, `KeyValue.keys/values`
for records, `Data` for String-table semantics, and explicit JSONValue kind
handling before choosing a Cell constructor. Excel does not duplicate those
modules and does not accept `Table`, `Pair`, or `JSONValue` through an `Any`
bridge.

v0.1.20 is XLSX-only. It does not include `.xls`, `.xlsm`, macros, charts,
images, pivot tables, rich-text runs, formula calculation, date inference,
encryption, print layout, or a user-facing ZIP API. `Excel` itself has no PDF
export, but `PDF.fromExcel(workbook)` converts a Workbook into a PDF document
without going through Excel at all -- see [PDF](PDF.md).
