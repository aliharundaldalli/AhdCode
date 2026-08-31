# Latex standard module

[English] · [Türkçe](LATEX_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Time module](TIME.md)

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
pdf(source: String, output: String)      -> Nothing
pdfFile(input: String, output: String)   -> Nothing

escape(text: String)                     -> String

document(
    body: String, title: String = "", author: String = "", date: String = "",
    type: String = "Article", margin: Real = 2.54, color: String = "",
    cover: String = "", theorems: Pair<String, String> = {},
    theme: String = "Default"
)                                         -> String

chapter(title: String)                   -> String
section(title: String)                   -> String
subsection(title: String)                -> String
frame(title: String, body: String)       -> String

equation(source: String, label: String = "") -> String
theorem(type: String, body: String, label: String = "") -> String

table(headers: List<String>, rows: List<List<String>>, mathColumns: List<Int> = []) -> String
image(path: String, size: Pair<String, Real> = {})    -> String
figure(path: String, caption: String, label: String = "", size: Pair<String, Real> = {}) -> String

minipage(body: String, width: Real, alignment: String = "left") -> String
center(body: String)                     -> String
pageBreak()                              -> String
contents()                               -> String

ref(label: String)                       -> String
cite(key: String)                        -> String
bibliography(references: Pair<String, String>) -> String

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
  `2.54` (the effective v0.1.14 layout); there is no per-side margin, paper
  size, or orientation control. It must be positive.
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
  (case-sensitive), defaults to `"Default"`, and is the final positional
  parameter. Madrid and Warsaw require `type: "Beamer"`; selecting either
  for Article or Report raises `ValueError`. Unknown theme names also raise
  `ValueError` and are never interpolated into LaTeX source. A custom
  `color` is applied after the theme, so it overrides the theme's structural
  accent while retaining the theme layout.

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

`image(path, size)` is an unnumbered figure fragment; `figure(path, caption,
label, size)` is numbered, captioned, and (with a label) referenceable via
`ref`. `size` is `Pair<String, Real>` with only `"width"`/`"height"` keys, in
centimeters: width only or height only preserves aspect ratio, both fit
explicitly, and an empty Pair uses the image's natural size.

```ahd
body += L.image("logo.png", {"width": 6.0})
body += L.figure("result.pdf", "Numerical solution", "fig:solution", {"width": 12.0})
```

Supported formats are PNG, PDF, and JPEG. There is no crop, trim, rotation,
subfigures, or exposed `graphicx`/float-placement options.

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

## Compiling

`pdf` compiles a source String; `pdfFile` compiles an existing `.tex` file and
resolves document-relative assets such as `\includegraphics` against the input
file's directory.

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
Article/Report — offline, with no system TeX.

## Security

The engine runs in untrusted mode, so `\write18` shell escape is unavailable,
and no AhdCode source construct can enable it. The engine is launched with an
argument vector — never a shell command string — so paths containing spaces,
Unicode, quotes, `$`, `;`, `&`, or parentheses stay safe. Asset staging copies
files by path, never through a shell, and rejects a missing, unreadable, or
unsupported-format asset before compilation starts.

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
minipage, or image-size options; and an unsupported image extension. Theme
validation deliberately does not introduce a different error class.

`LatexError` covers execution failures: compilation failure, a missing staged
engine or bundle, timeout, engine process failure, a PDF that was not produced,
and an asset file that cannot be staged. Engine diagnostics are bounded so a
malformed document cannot flood the terminal, while the first useful TeX
error is preserved.

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
`booktabs`, `array`, `geometry`, `xcolor`, `hyperref`, `fontspec`, the PGF/
TikZ core and `translator` packages Beamer builds on, Latin Modern fonts,
Computer Modern maths, the Default/Madrid/Warsaw Beamer theme closure, and
hyphenation data. Unicode text — including Turkish — works out of the box.

Not in this version: BibTeX, a package manager, a general TikZ drawing API,
arbitrary Beamer themes, overlays, speaker notes, a PDF editor or parser, and
Markdown or HTML conversion.
