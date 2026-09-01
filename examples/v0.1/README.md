# AhdCode v0.1 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs are small, working introductions to the v0.1 language.

```bash
ahdcode run examples/v0.1/01_hello.ahd
```

The input examples can be run interactively:

```bash
ahdcode run examples/v0.1/02_input.ahd
ahdcode run examples/v0.1/14_grade_app.ahd
```

Before running `16_latex.ahd`, remember that the `Latex` module requires you to stage the offline compiler once using:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

| Example | Topic |
|---|---|
| `01_hello.ahd` | declarations, interpolation, output |
| `02_input.ahd` | `take`, `int`, terminal input |
| `03_grade_average.ahd` | Lists and Fundamentals reductions |
| `04_loops.ahd` | `while`, post-check `until`, `for`, `between` |
| `05_functions.ahd` | Functions, defaults, named calls, callbacks |
| `06_list_api.ahd` | List mutation, map/filter, deterministic shuffle |
| `07_string_api.ahd` | immutable String operations |
| `08_pair.ahd` | insertion-ordered Pair workflow |
| `09_class.ahd` | structure attributes and methods |
| `10_errors.ahd` | `attempt`, `except`, `ultimately`, `toss` |
| `11_modules.ahd` | direct import from `Greeting.ahd` |
| `12_math.ahd` | explicit Math module and seeding |
| `13_null_safety.ahd` | flow-sensitive null refinement |
| `14_grade_app.ahd` | compact interactive CLI application |
| `15_time.ahd` | Time module: DateTime, Duration, Calendar, monotonic |
| `16_latex.ahd` | Latex module: module alias, helpers, PDF, LatexError |
| `17_filesystem.ahd` | inferred declarations, Path, UTF-8 File I/O, FileError |
| `18_protocols.ahd` | Class Protocol Methods, `type()`, `id()` |
| `19_regex.ahd` | Regex module: `Pattern`, match/find/replace/split/groups, `RegexError` |
| `20_lambda.ahd` | expression-only Function values, inference, callbacks, and normal Function contrast |
| `21_time_utc.ahd` | UTC, Unix milliseconds, fixed offsets, and instant-preserving conversion |
| `22_csv.ahd` | raw CSV transport, header records, quoting, Unicode, and multiline fields |
| `23_data.ahd` | Data tables: CSV to `Table`, filter, keyed sort, derive, groupBy, and explicit conversion |
| `24_capture.ahd` | explicit lambda capture lists, capture by value, and the lambda/Function split |
| `25_statistics.ahd` | Statistics: sum, mean, median, mode, dispersion, quantile, and undefined-input errors |
| `26_data_statistics.ahd` | Data and Statistics together: pivotCount, explicit conversion, and a captured threshold |
| `27_raw_strings.ahd` | Raw String literals: no escapes, no interpolation, Regex quantifiers, and LaTeX source |
| `28_plot.ahd` | Plot module: line, scatter, bar, histogram, box, errorBar, multiple series, save, and subplots |
| `29_data_plot.ahd` | Data, Regex (with a raw String pattern), explicit conversion, Statistics, and Plot together |
| `30_plot_show.ahd` | Manual smoke test for `chart.show()`/`figure.show()` -- not part of automated CI |
| `31_complex.ahd` | Complex literals, widening, operations, and canonical output |
| `32_numeric.ahd` | Numeric vectors, matrices, decompositions, solve, and eigenvalues |
| `33_numeric_plot.ahd` | Numeric `Vector` inputs passed directly to Plot |
| `34_latex_report.ahd` | Article/Report document options, cover, figures, theorems, references, and bibliography |
| `35_latex_beamer.ahd` | offline Beamer frames, contents, equations, tables, and accent color |
| `36_full_workflow.ahd` | Data → Numeric/Statistics → Plot → Latex Report workflow |
| `37_word_document.ahd` | Word headings, formatted paragraphs, alignment, page breaks, and DOCX save |
| `38_word_read.ahd` | Word DOCX semantic read-back: text, headings, paragraphs, and tables |
| `39_word_plot.ahd` | Plot PNG embedded into an immutable Word Document |
| `40_word_table_merge.ahd` | horizontal/vertical Word table merges and alignment |
| `41_latex_beamer_themes.ahd` | bounded Default/Madrid/Warsaw Beamer themes and custom color |
| `42_json.ahd` | JSON module: object/array construction, parse, stringify, typed accessors, get/at, JSONError |
| `43_xml.ahd` | XML module: element/text construction, attributes, parse, mixed content, stringify, XMLError |
| `44_env.ahd` | Env module: a self-created `.env` fixture, load/get/getOr/exists/set/unset, override precedence, EnvError |
| `45_structured_data_report.ahd` | JSON to Data to Statistics/Plot to Word: a small structured-data reporting workflow |
| `46_word_roundtrip.ahd` | the fixed Word merged-table semantic read/save behavior (v0.1.17) |
| `47_lists.ahd` | Lists module: chunk, flatten, transpose, unique, valueCounts, groupBy, and ListsError |
| `48_key_value.ahd` | KeyValue module: keys, values, combine, with, without, select, drop, rename, mapValues, merge, overlay, and KeyValueError |
| `49_json_record_update.ahd` | updating one JSON object field with `KeyValue.with`, with no stringify/parse round trip |
| `50_data_records.ahd` | headers plus value rows through `KeyValue.combine` into a `Data.Table` |
| `51_excel_basic.ahd` | typed Workbook/Sheet/Cell construction, Formula, and XLSX save |
| `52_excel_styles.ahd` | Range styling, safe merge, number formats, widths, and heights |
| `53_excel_data.ahd` | Data and KeyValue records converted explicitly to Cells; Lists transpose |
| `54_excel_roundtrip.ahd` | XLSX save/read/save/read and Formula-versus-String safety |
| `55_pdf_basic.ahd` | typed PDFDocument construction: heading, paragraph, table, page break, save |
| `56_pdf_word_excel.ahd` | `PDF.fromWord` and `PDF.fromExcel` semantic conversion into PDF |
| `57_archive.ahd` | Archive module: creation-only ZIP, TAR, and TAR.GZ packaging |
| `58_latex_pdf_tex_archive.ahd` | `Latex.pdf(..., "tex")` source sidecar packaged into a ZIP with Archive |

`Greeting.ahd` is the sibling module used by `11_modules.ahd`.
