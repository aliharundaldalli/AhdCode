# Latex standard module

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
section(title: String)                   -> String
subsection(title: String)                -> String
equation(source: String)                 -> String
document(body: String, title: String = "", author: String = "") -> String
table(headers: List<String>, rows: List<List<String>>) -> String

LatexError
```

## Text helpers

`escape` is **text-context** escaping. It handles the TeX-special characters
`\ { } $ & # % _ ^ ~` and nothing more — it does not claim to sanitize raw
mathematics.

`section` and `subsection` escape their titles. `equation` deliberately does
**not** escape: it takes raw LaTeX math source, which is the point.

`document` returns a complete document. Its preamble names the bundled Latin
Modern font files explicitly, so a document renders identically on a machine
with no fonts installed.

`table` produces deterministic `booktabs` source and escapes every cell. A row
whose column count differs from the headers is a `ValueError`.

## Compiling

`pdf` compiles a source String; `pdfFile` compiles an existing `.tex` file and
resolves document-relative assets such as `\includegraphics` against the input
file's directory.

Compilation is done by a **bundled** Tectonic engine and a **bundled** local
resource bundle that ship with an AhdCode installation:

```text
libexec/ahdcode/latex/tectonic
libexec/ahdcode/latex/ahdcode-latex.ttb
libexec/ahdcode/latex/THIRD_PARTY_NOTICES.txt
```

AhdCode never runs a `tectonic` found on `PATH`, never falls back to a system
TeX installation, and never downloads anything at run time. If the bundled
engine or bundle is missing, that is a `LatexError`.

## Offline by construction

The engine is invoked with an isolated per-invocation cache and a local-bundle
only policy, so a supported document compiles on a fresh machine with an empty
cache and no network. There is no separately installed TeX distribution and no
runtime resource download.

## Security

The engine runs in untrusted mode, so `\write18` shell escape is unavailable,
and no AhdCode source construct can enable it. The engine is launched with an
argument vector — never a shell command string — so paths containing spaces,
Unicode, quotes, `$`, `;`, `&`, or parentheses stay safe.

Compilation is bounded by a 30-second timeout. On timeout the engine process is
terminated, temporary files are removed, and a `LatexError` is raised.

## Output safety

Source compiles in a unique secure temporary directory that is removed on both
success and failure. The PDF is produced to a temporary location, checked for
existence, regular-file status, non-zero size, and the `%PDF-` signature, and
only then moved into the requested destination. A failed compile therefore
never destroys an already valid destination PDF.

## LatexError

One error covers the Latex-specific failures: compilation failure, a missing
bundled engine or bundle, timeout, engine process failure, and a PDF that was
not produced. Engine diagnostics are bounded so a malformed document cannot
flood the terminal, while the first useful TeX error is preserved.

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

`article`, `amsmath`/`amssymb`/`mathtools`, `graphicx`, `booktabs`, `array`,
`geometry`, `xcolor`, `hyperref`, `fontspec`, Latin Modern fonts, Computer
Modern maths, and hyphenation data. Unicode text — including Turkish — works
out of the box.

Not in this version: BibTeX, a package manager, TikZ or Beamer abstractions, a
PDF editor or parser, and Markdown or HTML conversion.
