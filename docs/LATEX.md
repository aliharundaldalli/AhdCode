# Latex standard module

[English] · [Türkçe](LATEX_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Time module](TIME.md)

If you are learning this module, start with the [Latex workshop](PRACTICAL_MODULES.md#6-latex-create-an-academic-pdf-or-slide-deck)
for Article, Report, Beamer, equations, tables, Plot figures, and bibliography;
use this page as the full helper and compilation reference.

Latex turns AhdCode Strings into PDF documents. It is explicit, like Math and
Time, and it works with the ordinary module forms including aliases:

```ahd
bring Latex
bring Latex as L
from Latex bring LatexError
```

The canonical identity is `builtin:Latex`; a sibling `Latex.ahd` cannot shadow
it. Every argument must be `NonNull`.

## Surface

```text
pdf(source: String, output: String, sourceOutput: String = "") -> Nothing
pdfFile(input: String, output: String)   -> Nothing

escape(text: String)                     -> String

document(
    body: String, title: String = "", author: String = "", date: String = "",
    type: String = "Article", margin: Real = 2.54, color: String = "",
    cover: String = "", theorems: Pair<String, String> = {},
    theme: String = "Default", landscape: Bool = false,
    paper: String = "Letter", pageSize: Pair<String, Real> = {},
    margins: Pair<String, Real> = {}, subject: String = "",
    keywords: List<String> = [], creator: String = ""
)                                         -> String

chapter(title: String)                   -> String
section(title: String)                   -> String
subsection(title: String)                -> String
frame(title: String, body: String)       -> String

equation(source: String, label: String = "") -> String
theorem(type: String, body: String, label: String = "") -> String

table(headers: List<String>, rows: List<List<String>>, mathColumns: List<Int> = []) -> String
image(path: String, size: Pair<String, Real> = {}, transform: Pair<String, Real> = {}) -> String
figure(
    path: String, caption: String, label: String = "",
    size: Pair<String, Real> = {}, transform: Pair<String, Real> = {}
)                                         -> String

minipage(body: String, width: Real, alignment: String = "left") -> String
center(body: String)                     -> String
pageBreak()                              -> String
contents()                               -> String

ref(label: String)                       -> String
cite(key: String)                        -> String
bibliography(references: Pair<String, String>) -> String

tikz(source: String, libraries: List<String> = [])    -> String
overlay(source: String, libraries: List<String> = []) -> String
border(inset: Real = 1.0, thickness: Real = 1.0, color: String = "") -> String

LatexError
```

## Text helpers

`escape` is **text-context** escaping. It handles the TeX-special characters
`\ { } $ & # % _ ^ ~` and nothing more — it does not claim to sanitize raw
mathematics.

`chapter`, `section`, and `subsection` escape their titles. `equation`
deliberately does **not** escape: it takes raw LaTeX math source, which is the
point. Raw String literals (v0.1.14) make this pleasant to write, since a
backslash needs no escaping of its own:

```ahd
body += L.equation(
    r"\|x+y\| \leq \|x\|+\|y\|"
)
```

## One `document()` for Article, Report, and Beamer

There is a single `document(...)` function for every supported document
type, selected by the `type` parameter — never separate `Latex.report()` or
`Latex.beamer()` functions:

```ahd
source: String := L.document(
    body: body
    title: "Numerical Analysis"
    author: "Ali Harun"
    date: "31 August 2026"
    type: "Report"
    margin: 2.5
    color: "#1F4E79"
    cover: cover
    theorems: theoremTypes
    theme: "Default"
)
```

`type` accepts exactly `"Article"`, `"Report"`, and `"Beamer"`; the default is
`"Article"`. An existing three-argument call, `L.document(body, title,
author)`, continues to work unchanged and still produces an `Article` with
every new parameter at its default.

- **`date`** defaults to `""` and is never filled in automatically with the
  system date — output stays deterministic across runs and machines.
- **`margin`** is one document-wide value in **centimeters**, defaulting to
  `2.54` (the effective v0.1.14 layout). It must be positive. `margins`
  (v1.3.0) overrides individual sides, `paper` and `pageSize` choose the page,
  and orientation is the separate `landscape` parameter.
- **`color`** is an optional `#RRGGBB` accent color (empty by default,
  preserving v0.1.14 output exactly). When set, it defines an `ahdaccent`
  color used for AhdCode-generated accents — the title/cover area and, for
  Beamer, the presentation's structural color. An invalid value raises
  `ValueError`.
- **`cover`** is ordinary generated LaTeX content (empty by default),
  inserted before the title page and followed by a page break; when `cover`
  is `""`, title/author/date behavior is byte-identical to v0.1.14.
  Ordering is always cover, then title, then body:
  ```ahd
  cover: String := L.center(
      L.image("logo.png", {"width": 5.0})
  )
  source := L.document(body: body, title: "Numerical Analysis", cover: cover)
  ```
- **`type: "Report"`** uses the `report` document class and enables
  `chapter`; **`type: "Beamer"`** uses the `beamer` document class, renders
  the title as a title-page frame instead of `\maketitle`, and supports the
  narrow slide surface described below.
- **`theme`** accepts exactly `"Default"`, `"Madrid"`, and `"Warsaw"`
  (case-sensitive), defaults to `"Default"`, and is the tenth positional
  parameter, followed only by `landscape`. Madrid and Warsaw require
  `type: "Beamer"`; selecting either for Article or Report raises
  `ValueError`. Unknown theme names also raise `ValueError` and are never
  interpolated into LaTeX source. A custom `color` is applied after the theme,
  so it overrides the theme's structural accent while retaining the theme
  layout.
- **`landscape`** (v1.2.0) defaults to `false`. `true` turns every page of an
  Article or Report sideways on the same paper, with the same `margin`. It is
  a general layout switch, not a certificate mode. Beamer slides are already
  wide, so `landscape: true` with `type: "Beamer"` raises `ValueError`.
- **`paper`** (v1.3.0) defaults to `"Letter"`, the US Letter page every Latex
  document has always used, and accepts exactly `"A3"`, `"A4"`, `"A5"`,
  `"Letter"`, `"Legal"`, or `"Custom"`. `"Custom"` requires **`pageSize`**
  with both `"width"` and `"height"` in centimeters; `pageSize` with any other
  paper raises `ValueError`.
- **`margins`** (v1.3.0) accepts any of `"top"`, `"right"`, `"bottom"`, and
  `"left"` in centimeters; a side left out uses `margin`. Every length must be
  greater than 0 and at most 1000, and the margins must leave room for
  content. A non-default `paper`, a `pageSize`, or `margins` with
  `type: "Beamer"` raises `ValueError`: slides have their own size.
- **`subject`**, **`keywords`**, and **`creator`** (v1.3.0) set PDF document
  properties. Once any of them is set, `title` and `author` are recorded as
  properties too. Each keyword must be non-empty and free of commas. With all
  three left empty, the document is byte-for-byte what it was before v1.3.0.

```ahd
source := L.document(
    body: body
    title: "Quarterly Report"
    author: "AhdCode Analytics"
    paper: "A4"
    margins: {"top": 3.0, "bottom": 2.5}
    subject: "Quarterly results"
    keywords: ["report", "2026"]
    creator: "AhdCode"
)
```

## Article, Report, Beamer

**Article** is the existing v0.1.14 baseline, unchanged when `type` is
omitted or `"Article"`.

**Report** genuinely uses the `report` document class and adds `chapter` on
top of the existing `section`/`subsection`:

```ahd
body += L.chapter("Introduction")
body += L.section("Background")
```

**Beamer** genuinely compiles offline with the bundled resource bundle — no
system TeX, no network, no runtime download. Its scope is intentionally
narrow: `document`, `frame`, `section`, `equation`, `table`, `image`, and
`contents`. Theme support is deliberately bounded to Default, Madrid, and
Warsaw; there is no arbitrary theme passthrough. There are no overlays,
`\pause`, transitions, speaker notes, custom navigation symbols, or a columns
abstraction. `frame` builds one slide:

```ahd
slides: String := ""
slides += L.frame("Contents", L.contents())
slides += L.frame("First Slide", L.equation(r"E = mc^2"))

presentation := L.document(
    body: slides
    title: "Talk"
    type: "Beamer"
    theme: "Madrid"
    color: "#1F4E79"
)
```

## Equation labels and `ref`

`equation(source, label)` takes an optional label. One `ref(label)` resolves
a label produced by `equation`, `theorem`, or `figure` — there are no
separate `eqRef`/`theoremRef`/`figureRef` functions:

```ahd
body += L.equation(
    r"\|x+y\| \leq \|x\|+\|y\|"
    "eq:triangle"
)
body += "See " + L.ref("eq:triangle") + "."
```

## User-defined theorem types

There is one generic `theorem(type, body, label)` helper — never separate
`lemma`/`definition`/`corollary`/`proposition`/`remark` functions. The
available theorem types, and how each one's counter behaves, are configured
through `document(theorems: ...)`:

```ahd
theoremTypes: Pair<String, String> := {
    "Theorem": "section"
    "Lemma": "Theorem"
    "Definition": "section"
    "Corollary": "Theorem"
}

source := L.document(body: body, type: "Article", theorems: theoremTypes)

body += L.theorem(type: "Theorem", body: "Every finite-dimensional normed space is complete.", label: "thm:finite")
```

The Pair's **key** is the public theorem type name; the **value** is its
counter rule:

```text
""            -> an independent, document-wide counter
"section"     -> reset by section
"subsection"  -> reset by subsection
"chapter"     -> reset by chapter (Report documents only)
"<type name>" -> share that (already-declared) type's counter
```

`"Theorem": "section"` with `"Lemma": "Theorem"` and `"Corollary": "Theorem"`
numbers conceptually like `Theorem 1.1`, `Lemma 1.2`, `Corollary 1.3` — the
three types share one counter that resets per section.

A display name never becomes a raw TeX identifier: each theorem type gets a
generated, collision-safe internal name. `document()` rejects, as
`LatexError`, an empty type name, a `theorem()` call for a type that was
never registered, a shared-counter rule naming an unknown or not-yet-declared
type (which also catches a self- or circular reference), and a `"chapter"`
rule outside a Report document.

## Image and figure

`image(path, size, transform)` is an unnumbered figure fragment;
`figure(path, caption, label, size, transform)` is numbered, captioned, and
(with a label) referenceable via `ref`. `size` is `Pair<String, Real>` with
only `"width"`/`"height"` keys, in centimeters: width only or height only
preserves aspect ratio, both fit explicitly, and an empty Pair uses the
image's natural size.

```ahd
body += L.image("logo.png", {"width": 6.0})
body += L.figure("result.pdf", "Numerical solution", "fig:solution", {"width": 12.0})
```

Supported formats are PNG, PDF, JPEG, and (v1.3.0) SVG, which stays vector;
see [SVG assets](#svg-assets-v130).

`transform` (v1.3.0) is a `Pair<String, Real>` with these keys:

| Key | Meaning | Range |
|---|---|---|
| `"rotation"` | degrees, counterclockwise | -360 to 360 |
| `"opacity"` | 0 is invisible, 1 is opaque | 0 to 1 |
| `"trimLeft"`, `"trimTop"`, `"trimRight"`, `"trimBottom"` | centimeters removed from that edge of the sized image | 0 to 1000 |

The image is sized first, then trimmed, rotated, and faded; vector content
stays vector. An unknown key, a value out of range, or trims that would remove
the whole width or height raise `ValueError`. Without a transform, an image
fragment is exactly what it was before v1.3.0.

```ahd
body += L.image("photo.jpg", {"width": 8.0}, {"trimTop": 1.0, "trimBottom": 1.0})
body += L.figure("stamp.png", "Approved", "fig:stamp", {"width": 4.0}, {"rotation": 12.0, "opacity": 0.6})
```

There are no subfigures or exposed `graphicx`/float-placement options.

### Asset staging

`pdf`/`pdfFile` compile in an isolated temporary workspace, so an image path
cannot simply be assumed to exist there. `image`/`figure` resolve their path
against the compiling program's working directory (the same rule `chart.save`
and `File` use) and stage a copy of that file into the compilation workspace
automatically — no dev-repository path, accidental working-directory
behavior, system TeX, or network access is involved:

```ahd
chart.save("chart.png")

body += L.figure("chart.png", "Results", "fig:results", {"width": 12.0})

source := L.document(body: body, type: "Report")
L.pdf(source: source, output: "report.pdf")
```

A missing or unreadable asset is a `LatexError` raised at compile time, not a
silently broken PDF. `pdfFile`'s existing document-relative asset resolution
is unchanged.

## Layout helpers

```ahd
left := L.minipage(leftBody, 7.0, "left")
right := L.minipage(rightBody, 7.0, "right")
body += left + right

body += L.center(
    L.minipage(content, 10.0, "center")
)

body += L.pageBreak()
```

`minipage`'s `width` is centimeters; `alignment` is exactly `"left"`,
`"center"`, or `"right"`, applied to the content inside the minipage. `center`
is a separate, simpler wrapper. There is no CSS-like layout system and no
grid/flex abstraction.

`contents()` emits a table of contents fragment for Article/Report:

```ahd
body += L.contents()
```

For Beamer, `contents()` does not silently become a frame — write the frame
explicitly:

```ahd
slides += L.frame("Contents", L.contents())
```

## Citations and bibliography

`cite(key)` is a bibliography citation, kept distinct from `ref` (an internal
document reference to an equation/theorem/figure label):

```ahd
body += "As shown in " + L.cite("Hardy1934") + "."
```

`bibliography(references)` renders a reference list from a `Pair<String,
String>` of citation key to exact bibliography text, in insertion order:

```ahd
references: Pair<String, String> := {
    "Yildiz2016": "B. Yıldız, Article title, Journal Name, 2016."
    "Hardy1934": "G. H. Hardy, J. E. Littlewood and G. Pólya, Inequalities, 1934."
}

body += L.bibliography(references)
```

Latex never sorts references, infers author/year/journal, formats APA or
IEEE, uses BibTeX, requires a `.bib` file, or rewrites the provided text —
the value is used exactly as given.

## Table

`table` is unchanged from v0.1.14: deterministic `booktabs` source, every
cell escaped, and `mathColumns: List<Int>` opting specific zero-based columns
into raw inline math (`\( ... \)`) instead of escaping. See the v0.1.14
behavior above; nothing about it changed for v0.1.15.

## Vector graphics with TikZ (v1.2.0)

Borders, ornaments, seals, badges, watermarks, diagrams, arrows, and positioned
labels belong to the same Latex document as the text. TikZ/PGF is the vector
drawing foundation and is bundled offline with the Latex runtime, so decorating
a PDF never requires a second PDF library or a pre-rendered border image.

```text
document typography    -> Latex
vector graphics        -> TikZ/PGF, through Latex.tikz and Latex.overlay
ready-made ornaments   -> pgfornament
raster images          -> Latex.image and Latex.figure
PDF output             -> Latex.pdf
```

TikZ stays TikZ. AhdCode does not translate drawing commands and publishes no
`line`, `circle`, or `path` wrappers: the helpers below place your TikZ source
into the document unchanged and load exactly the bundled libraries it names.

### Write TikZ in raw triple Strings

A raw triple String, `r"""..."""`, keeps backslashes, braces, brackets, and `%`
exactly as typed, spans lines, and performs no `{...}` interpolation, so
`{AhdCode}` inside a node stays text. In a normal String the same braces would
be an interpolation.

```ahd
bring Latex as L

drawing: String := L.tikz(r"""
\draw (0,0) rectangle (4,2);
\node at (2,1) {AhdCode};
""")
write(drawing)
```

To place program data inside TikZ, join raw parts with escaped text; the raw
parts stay TikZ and the data stays text:

```ahd
bring Latex as L

name: String := "Ayşe & Ali"
label: String := L.tikz(r"\node[draw] {" + L.escape(name) + r"};")
write(label)
```

### tikz

`tikz(source, libraries)` returns a `tikzpicture` fragment that sits in the
text flow, like an image, so it works inside `center`, `minipage`, or a frame.
Options for the whole picture go inside the source, for example
`\begin{scope}[scale=2] ... \end{scope}`.

### overlay

`overlay(source, libraries)` draws against the page itself instead of the text.
Its source can use TikZ's page anchors: `current page.north`,
`current page.south west`, `current page.north east`, `current page.center`,
and the rest.

```ahd
bring Latex as L

watermark: String := L.overlay(r"""
\node[opacity=0.08, rotate=30, scale=8] at (current page.center) {DRAFT};
""")
body: String := r"\thispagestyle{empty}" + "\n" + watermark + L.section("Report")
write(L.document(body))
```

An overlay is drawn on the page being filled when the fragment is reached, and
it never moves that page's text. Put it at the start of the page it decorates;
for several pages, add one per page. Nothing is rasterized.

### border

`border(inset, thickness, color)` is an ordinary overlay that draws one
rectangular page border: `inset` centimeters from every page edge (default
`1.0`, not negative), `thickness` in points (default `1.0`, positive), and an
optional `#RRGGBB` `color`. Two calls give a double border. Anything more
elaborate — rounded corners, dashes, ornaments — is TikZ in `overlay`.

### Bundled libraries and pgfornament

`libraries` accepts exactly these names, case-sensitively:

```text
calc  positioning  arrows.meta  shapes.geometric
decorations.pathmorphing  decorations.pathreplacing
patterns  fit  backgrounds  pgfornament
```

`pgfornament` loads the pgfornament package and its 196 Vectorian ornaments,
drawn with `\pgfornament[width=3cm]{63}`; the package's `symmetry` option
mirrors one ornament into each corner. Any other name raises `ValueError`
before anything compiles.

`document()` loads TikZ only when its body or cover contains a `tikz`,
`overlay`, or `border` fragment, and then loads each requested library once, in
the order above. A document without such a fragment is byte-for-byte what it
was before v1.2.0. A hand-written `\begin{tikzpicture}` in a body does not load
TikZ for you — use `Latex.tikz`. A complete source you pass to `Latex.pdf` may
load `\usepackage{tikz}` and the libraries above itself.

### A certificate

```ahd
bring Latex as L
from Latex bring LatexError

frame: String := L.border(inset: 0.8, thickness: 2.4, color: "#1F4E79")
frame += L.border(inset: 1.25, thickness: 0.6, color: "#B08D57")
corner: String := L.overlay(
    source: r"""
\node[anchor=north west] at ([shift={(1.5cm,-1.5cm)}]current page.north west)
    {\pgfornament[width=3cm]{63}};
"""
    libraries: ["pgfornament"]
)
title: String := r"{\Huge\bfseries Certificate of Achievement}\par\vspace{1cm}" + "\n"
title += r"{\LARGE\itshape " + L.escape("Ayşe Yılmaz") + r"}\par" + "\n"
body: String := r"\thispagestyle{empty}" + "\n" + frame + corner
body += r"\vspace*{\fill}" + "\n" + L.center(title) + r"\vspace*{\fill}" + "\n"

attempt {
    L.pdf(L.document(body: body, landscape: true), "certificate.pdf")
} except LatexError as error {
    write(error.message)
}
```

The complete certificate — ornaments in every corner, a watermark, a TikZ
seal, and signature lines — is
[`examples/v0.1/61_tikz_certificate.ahd`](../examples/v0.1/61_tikz_certificate.ahd).

### TikZ errors and safety

The helpers validate their own input when a fragment is built: an unbundled
library name, a negative border inset, a thickness that is not positive, an
invalid border color, and `landscape` with Beamer raise `ValueError`.

TikZ source itself is Latex input and is not checked by the AhdCode compiler.
A TikZ syntax error, a library loaded by hand that is not bundled, or a missing
package fails compilation with `LatexError`, keeping the first TeX error:

```text
compilation failed: error: document.tex:13: Package tikz Error: Cannot parse this coordinate.
compilation failed: error: document.tex:3: Package tikz Error: I did not find the tikz library 'shadows'. ...
compilation failed: error: document.tex:3: ! LaTeX Error: File `tcolorbox.sty' not found.
```

TikZ is not a sandbox. It runs in the same untrusted-mode engine as every Latex
document: shell escape stays unavailable, and nothing is downloaded.

There is no TCPDF or FPDF, Canvas, SVG drawing module, browser renderer,
second PDF engine, drawing API that mirrors TikZ commands, certificate module,
arbitrary package loading, or rasterized decoration.

## Professional documents (v1.3.0)

Running headers and footers with page numbers, content at exact positions on
the page, links, PDF outline bookmarks, QR codes, barcodes, and SVG logos are
ordinary Latex fragments: build them with the helpers below and join them into
`document()`'s body like any other fragment. `document()` loads a supporting
package only when a fragment needs it, so a document without these fragments
is byte-for-byte what it was before v1.3.0.

```text
running header/footer  -> header, footer, pageNumber, pageCount
exact page position    -> place
navigation             -> link, bookmark
machine-readable codes -> qr, barcode
page and properties    -> document(paper:, pageSize:, margins:, subject:, keywords:, creator:)
vector logos           -> image and figure with an .svg path
```

A complete report with all of them is
[`examples/v0.1/66_latex_professional_report.ahd`](../examples/v0.1/66_latex_professional_report.ahd),
and a certificate with a verification QR code is
[`examples/v0.1/64_verifiable_certificate.ahd`](../examples/v0.1/64_verifiable_certificate.ahd).

### header, footer, pageNumber, pageCount

`header(left, center, right)` and `footer(left, center, right)` set the three
regions at the top and bottom of every page. The regions hold generated Latex
— escaped text, `pageNumber()`, `pageCount()`, a `link`, an `image` such as an
SVG logo, or a `qr` symbol — so pass ordinary text through `escape`.
`pageNumber()` is the current page number and `pageCount()` the document's
total page count; both also work in body text.

```ahd
logo: String := L.image("logo.svg", {"height": 0.8})
header: String := L.header(logo, L.escape("Quarterly Report"), L.escape("Q3 2026"))
footer: String := L.footer(L.link("ahdcode.org", "https://ahdcode.org"), "", "Page " + L.pageNumber() + " of " + L.pageCount())
source := L.document(body: header + footer + body, paper: "A4")
```

A Latex document shows the page number centered in the footer by default. A
`header` alone keeps it; a `footer` sets all three footer regions, so include
`pageNumber()` where the number should appear. The settings take effect from
the page on which the fragment is reached: put them at the start of the body to
cover every page, and add another `header` or `footer` later to change the
following pages. Headers and footers also apply to title and chapter pages and
draw no rules. They require an Article or Report document; with Beamer,
`document()` raises `ValueError`.

### place

`place(content, x, y, anchor)` puts content at an exact position on the page.
`x` and `y` are centimeters from the top-left corner of the page, from 0 to
1000, and `anchor` names the point of the content that lands there:
`"north west"` (the default), `"north"`, `"north east"`, `"west"`, `"center"`,
`"east"`, `"south west"`, `"south"`, or `"south east"`. Placed content never
moves the page's text. It is drawn on the page being filled when the fragment
is reached, so place content once for each page it belongs on. For drawing with
TikZ coordinates and page anchors, use `overlay`.

```ahd
body += L.place(L.image("logo.svg", {"height": 2.0}), 2.0, 2.0)
body += L.place(L.qr("https://ahdcode.org/verify?id=42", 2.5, "Q"), 19.0, 27.5, "south east")
```

### link and bookmark

`link(text, url)` makes `text` clickable; `text` is escaped. `url` must start
with `https://`, `http://`, or `mailto:` and must not contain control
characters; any other scheme, including `javascript:` and `file:`, raises
`ValueError`. Special characters in the URL are encoded, so the link opens
exactly the given URL and cannot break out into the document. Links are drawn
without a colored box.

`bookmark(title, level)` adds an entry to the PDF outline — the viewer's
sidebar — pointing at the position where the fragment is reached. Level 1 is a
top-level entry; levels 2 to 4 nest under the nearest preceding entry of a
smaller level. `chapter`, `section`, and `subsection` already appear in the
outline on their own; use `bookmark` for anything else, such as a cover, a
table, or a verification code.

```ahd
body += L.bookmark("Verification code") + L.center(L.qr("https://ahdcode.org/verify?id=42"))
body += "Verify at " + L.link("ahdcode.org/verify", "https://ahdcode.org/verify?id=42") + "."
```

The page size, margins, and PDF properties are `document()` parameters; see
[One `document()` for Article, Report, and Beamer](#one-document-for-article-report-and-beamer).

## QR codes and barcodes (v1.3.0)

`qr(value, size, level)` draws a QR symbol `size` centimeters square (default
`3.0`), quiet zone included, at level `"L"`, `"M"` (the default), `"Q"`, or
`"H"`. `barcode(kind, value, width, height)` draws a `"Code128"`, `"EAN13"`, or
`"UPCA"` barcode `width` by `height` centimeters (default `8.0` by `2.0`), quiet
zones included. Both are TikZ vector fragments built by the same encoders as
the [QR](QR.md) and [Barcode](BARCODE.md) modules — no image file and no
rasterizing — and work in the body, in `center` and `minipage`, in `header` and
`footer`, and in `place`. They need no `bring QR` or `bring Barcode`.

```ahd
body += L.center(L.qr("https://ahdcode.org", 3.0, "Q"))
body += L.barcode("EAN13", "590123412345", 6.0, 2.0)
```

An invalid value — an empty or oversized QR value, an unknown level or kind, a
wrong EAN-13 check digit, a non-ASCII Code 128 character — raises `ValueError`
when the fragment is built, with the same message the QR and Barcode modules
give.

## SVG assets (v1.3.0)

`image` and `figure` accept an `.svg` path. When the document compiles, the SVG
is converted inside the program into PGF drawing commands, so it stays vector
in the PDF. Nothing is rasterized, and no browser, Inkscape, shell command, or
network access is involved; the conversion reads only the SVG file itself.
`PDFDocument.image` uses the same converter.

Supported:

- the elements `svg`, `g`, `path` (every path command, including arcs),
  `rect` (including rounded corners), `circle`, `ellipse`, `line`,
  `polyline`, `polygon`, `use` referring to an element of the same file
  (`#id`), `defs`, `title`, `desc`, `metadata`, and `style`;
- presentation attributes, inline `style` attributes, and `<style>` rules
  whose selectors are an element name, `.class`, `#id`, or `*`;
- `fill` and `stroke` as a hex color, `rgb()`, a CSS color name, `none`, or
  `currentColor`; `fill-rule`; `opacity`, `fill-opacity`, and
  `stroke-opacity`;
- `stroke-width`, `stroke-linecap`, `stroke-linejoin`, `stroke-miterlimit`,
  `stroke-dasharray`, and `stroke-dashoffset`; `display: none` and
  `visibility`;
- `transform` with `matrix`, `translate`, `scale`, `rotate`, `skewX`, and
  `skewY`; `viewBox`, `preserveAspectRatio`, `width`, and `height`, with
  lengths in `px`, `pt`, `pc`, `mm`, `cm`, or `in`.

Rejected with a message naming what is unsupported, never dropped or
approximated:

- text (`text`, `tspan`, `textPath`) — convert text to outlines in the SVG
  editor;
- gradients, patterns, clipping paths, masks, filters, markers, `symbol`,
  `switch`, and colors with their own transparency such as `#00000080` or
  `rgba(0, 0, 0, 0.5)` — use `fill-opacity` or `stroke-opacity` instead;
- embedded or linked images, `script`, `foreignObject`, links (`a`),
  animation, nested `svg`, event attributes such as `onload`, `url(...)`
  references, and any reference outside the file;
- CSS at-rules such as `@import`, a DOCTYPE or entity declarations, and
  processing instructions.

An SVG is limited to 5 MiB, 100,000 elements, 64 levels of nesting, and
2,000,000 path segments. Latex converts an SVG when the document compiles, so an
unsupported SVG raises `LatexError`; `PDFDocument.image` converts it when called
and raises `PDFError`.

## Compiling

`pdf` compiles a source String; `pdfFile` compiles an existing `.tex` file and
resolves document-relative assets such as `\includegraphics` against the input
file's directory.

`pdf` takes an optional third `sourceOutput` argument, `"" `(default) or
`"tex"`:

```ahd
pdf(source: String, output: String, sourceOutput: String = "") -> Nothing
```

`sourceOutput: ""` — the existing, unchanged contract: only `output` (a
`.pdf`) is published. `sourceOutput: "tex"` additionally publishes a sibling
`.tex` file containing the caller's exact `source` bytes:

```ahd
attempt {
    L.pdf(document, "cikti.pdf", "tex")
    write("PDF ve TEX başarıyla oluşturuldu!")
}
except LatexError as error {
    write("Dosyalar oluşturulamadı: {error.message}")
}
```

produces `cikti.pdf` and `cikti.tex`. The sibling path is derived by
replacing the trailing `.pdf` with `.tex` (`report.pdf` → `report.tex`,
`folder/report.pdf` → `folder/report.tex`); `output` must end in `.pdf` when
`sourceOutput` is `"tex"`, or `LatexError` is raised before compiling. Any
`sourceOutput` value other than the exact, case-sensitive `""` or `"tex"`
also raises `LatexError` — there is no arbitrary passthrough string.

The `.tex` sidecar is the caller's own source, verbatim: not a transformed
temporary Tectonic input, not compiler-added debugging content, not a
temporary path, and not renderer metadata. Publication order is: compile and
verify the PDF first, then — only once the PDF has already been atomically
published — write the `.tex` sidecar. A compile failure publishes neither
file and never touches an existing destination. Because there is no
filesystem-wide two-file transaction, a `.tex`-sidecar write failure after a
successful PDF publish leaves the new PDF published and the sidecar
unwritten; this is two individually atomic renames, not one two-file atomic
transaction.

`pdfFile` is unchanged: it still takes exactly `(input, output)`, because its
caller already owns the `.tex` file on disk.

**Release packages include this runtime. The following staging instructions are for source-build contributors only.**

Compilation is done by the offline Tectonic engine and a local resource bundle. The standard source installation (`go install`) does not install the LaTeX runtime files. A user who wants to use LaTeX must explicitly stage them once using the `package-latex` tool:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

This command performs a one-time network operation to fetch and verify the pinned resources, placing them in your Go binary directory alongside `ahdcode`:

```text
libexec/ahdcode/latex/tectonic
libexec/ahdcode/latex/ahdcode-latex.ttb
libexec/ahdcode/latex/THIRD_PARTY_NOTICES.txt
```

Once staged, AhdCode never runs a `tectonic` found on `PATH`, never falls back to a system TeX installation, and never downloads anything at run time. If the offline engine or bundle is missing, that is a `LatexError`.

## Offline by construction

The engine is invoked with an isolated per-invocation cache and a local-bundle
only policy, so a supported document compiles on a fresh machine with an empty
cache and no network. There is no separately installed TeX distribution and no
runtime resource download. This includes Beamer: the staged resource bundle
carries `beamer.cls`, its `beamerbase*` components, the PGF/TikZ core it
builds on, and `translator`, so a Beamer presentation compiles exactly like
Article/Report — offline, with no system TeX. Since v1.2.0 it also carries
TikZ, the nine TikZ libraries `Latex.tikz` accepts, and pgfornament with its
Vectorian ornaments, and since v1.3.0 `fancyhdr` and `lastpage` for headers,
footers, and page counts; every file is pinned by checksum in the resource
manifest.

## Security

The engine runs in untrusted mode, so `\write18` shell escape is unavailable,
and no AhdCode source construct can enable it. The engine is launched with an
argument vector — never a shell command string — so paths containing spaces,
Unicode, quotes, `$`, `;`, `&`, or parentheses stay safe. Asset staging copies
files by path, never through a shell, and rejects a missing, unreadable, or
unsupported-format asset before compilation starts. An SVG asset is converted
by AhdCode itself from the file's bytes: nothing in it is executed, and it
cannot load another file or a URL.

Compilation is bounded by a 30-second timeout. On timeout the engine process is
terminated, temporary files are removed, and a `LatexError` is raised.

## Output safety

Source compiles in a unique secure temporary directory that is removed on both
success and failure. The PDF is produced to a temporary location, checked for
existence, regular-file status, non-zero size, and the `%PDF-` signature, and
only then moved into the requested destination. A failed compile therefore
never destroys an already valid destination PDF.

## ValueError and LatexError

Input-domain validation follows the existing Latex API contract and raises
`ValueError`: invalid `document()` type, margin, color, or theme; a non-Default
theme outside Beamer; invalid theorem registration/reference; invalid table,
minipage, or image-size options; an unsupported image extension; since
v1.2.0, an unbundled TikZ library name, invalid `border` values, and
`landscape` with Beamer; and, since v1.3.0, an invalid `qr` or `barcode` value,
`place` coordinate or anchor, `link` URL, `bookmark` level, image `transform`,
`paper`, `pageSize`, `margins`, or keyword, and page layout, headers, or
footers with Beamer. Theme validation deliberately does not introduce a
different error class.

`LatexError` covers execution failures: compilation failure, a missing staged
engine or bundle, timeout, engine process failure, a PDF that was not produced,
and an asset file that cannot be staged, including an unsupported SVG. Engine
diagnostics are bounded so a malformed document cannot flood the terminal,
while the first useful TeX error is preserved.

```ahd
bring Latex as L
from Latex bring LatexError

attempt {
    L.pdf(source: source, output: "report.pdf")
} except LatexError as error {
    write(error.message)
}
```

## Supported baseline

`article`, `report`, `beamer`, `amsmath`/`amssymb`/`mathtools`, `graphicx`,
`booktabs`, `array`, `geometry`, `xcolor`, `hyperref`, `fontspec`, PGF and
TikZ with the `calc`, `positioning`, `arrows.meta`, `shapes.geometric`,
`decorations.pathmorphing`, `decorations.pathreplacing`, `patterns`, `fit`, and
`backgrounds` libraries, `pgfornament`, `translator`, Latin Modern fonts,
Computer Modern maths, the Default/Madrid/Warsaw Beamer theme closure, and
hyphenation data. Unicode text — including Turkish — works out of the box.

Not in this version: BibTeX, a package manager, a drawing API that mirrors
TikZ commands, TikZ libraries beyond the list above, arbitrary Beamer themes,
Beamer overlays, speaker notes, a PDF editor or parser, and Markdown or HTML
conversion.
