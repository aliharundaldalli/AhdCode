# AhdCode Practical Module Workshops

[Back to README](../README.md) · [Student Guide](STUDENT_GUIDE_EN.md) · [Modules](MODULES.md)

This guide teaches the AhdCode modules that are most useful together through
small workflows, rather than presenting them only as API lists. If you know the
language basics from the student guide, you can work through these workshops in
order.

Each workshop answers four questions:

1. What problem does this module solve?
2. Which AhdCode types does it accept and return?
3. What is the most common mistake?
4. How does its result cross into another module?

Short code blocks within one workshop continue from variables created in the
preceding blocks unless stated otherwise. To run a complete workshop, combine
its blocks from top to bottom in one `.ahd` file.

## Contents

- [Learning map](#learning-map)
- [Ready-to-run examples](#ready-to-run-examples)
- [1. CSV: transport a text table safely](#1-csv-transport-a-text-table-safely)
- [2. Data: reshape a String table](#2-data-reshape-a-string-table)
- [3. Plot: turn data into a readable chart](#3-plot-turn-data-into-a-readable-chart)
- [4. Excel: create a real XLSX with typed cells](#4-excel-create-a-real-xlsx-with-typed-cells)
- [5. Word: build a DOCX report with tables and figures](#5-word-build-a-docx-report-with-tables-and-figures)
- [6. Latex: create an academic PDF or slide deck](#6-latex-create-an-academic-pdf-or-slide-deck)
- [7. HTTP and HTTPS: requests, responses, and failures](#7-http-and-https-requests-responses-and-failures)
- [8. HTML: build safe pages and parse documents](#8-html-build-safe-pages-and-parse-documents)
- [9. Final project: produce two reports from one dataset](#9-final-project-produce-two-reports-from-one-dataset)
- [Which document should I read?](#which-document-should-i-read)

## Learning map

```text
CSV text ──> CSV records ──> Data Table ──> numeric List
                                │                │
                                │                ├──> Statistics / Plot
                                │                │
                                ├──> CSV output  ├──> Word report
                                │                │
                                └───────────────> Excel workbook

HTTPS URL ──> HTTP ClientResponse.body() ──> HTML.parse ──> selected data

text + table + figure ──> Word (.docx) or Latex (.pdf)
```

The arrows do not mean implicit conversion. AhdCode keeps module boundaries
explicit. A CSV cell containing `"91"` is still a `String`; convert it with
`int(...)` or `real(...)` before passing it to Statistics or Plot.

## Ready-to-run examples

Run these complete programs while working through the guide:

- [CSV](../examples/v0.1/22_csv.ahd) and [Data basics](../examples/v0.1/23_data.ahd)
- [Data + Plot](../examples/v0.1/29_data_plot.ahd)
- [Data → Statistics → Plot → Latex](../examples/v0.1/36_full_workflow.ahd)
- [Word with a Plot figure](../examples/v0.1/39_word_plot.ahd)
- [JSON → Data → Statistics → Word](../examples/v0.1/45_structured_data_report.ahd)
- [Excel basics](../examples/v0.1/51_excel_basic.ahd),
  [styles](../examples/v0.1/52_excel_styles.ahd), and
  [read/write round trip](../examples/v0.1/54_excel_roundtrip.ahd)
- [HTTPS client examples](../examples/v0.6/README.md)
- [HTML parsing and scraping examples](../examples/v0.7/README.md)

## 1. CSV: transport a text table safely

CSV does not calculate. Its job is to read and write rows and columns of text
correctly. A name containing a comma, a note containing a quote, or a newline
inside a cell cannot be parsed safely with `split(",")`; `CSV` applies the
format's quoting rules for you.

### 1.1 Rows and records

In the raw row model, the header is an ordinary row:

```ahd
bring CSV

source: String := "name,city,score\nAli,Adana,91\nAyse,Ankara,78\n"
rows: List<List<String>> := CSV.parse(source)

write(rows[0])
write(rows[1][0])
```

When the first row contains headers, records are easier to read:

```ahd
records: List<Pair<String, String>> := CSV.parseRecords(source)

write(records[0]["name"])
write(records[0]["score"])
```

Use this rule of thumb:

- Use `parse` / `read` when the first row is not a special header.
- Use `parseRecords` / `readRecords` when the first row names the columns.
- Move to `Data` when you need filtering, sorting, grouping, or derivation.

### 1.2 Why quoting matters

The comma inside the second cell below is data, not a column separator:

```ahd
bring CSV

source: String := "name,note\nAli,\"fast, careful\"\n"
rows: List<List<String>> := CSV.parse(source)
write(rows[1][1])
```

The result is `fast, careful`. A quote inside a cell is escaped as `""`.
`CSV.stringify(...)` and `CSV.stringifyRecords(...)` add quoting when needed;
do not assemble CSV text by hand.

### 1.3 Convert numbers explicitly

CSV cannot know whether `"91"` is a score, an identifier, or literal text.
Code that understands the column decides:

```ahd
bring CSV

records: List<Pair<String, String>> := CSV.parseRecords(
    "name,score\nAli,91\nAyse,78\n"
)

total: Int := 0
for record in records {
    total = total + int(record["score"])
}
write(total)
```

Real files may contain invalid values. Treat conversion failure as a data
quality issue:

```ahd
attempt {
    score: Local Int := int(record["score"])
    write(score)
}
except DomainError as error {
    badValue: Local String := record["score"]
    write("invalid score: {badValue}")
}
```

### 1.4 Delimiters and files

The same module can read a semicolon-delimited file:

```ahd
bring CSV

rows: List<List<String>> := CSV.read("students.csv", ";")
CSV.write("copy.csv", rows, ";")
```

A delimiter must be one Unicode scalar. `","`, `";"`, and `"\t"` are valid;
an empty String or a two-character `"||"` delimiter is not.

### 1.5 Structural failures

```ahd
bring CSV
from CSV bring CSVError

attempt {
    CSV.parseRecords("name,score\nAli\n")
}
except CSVError as error {
    write("bad CSV structure: {error.message}")
}
```

`CSVError` covers malformed quoting, invalid delimiters, and records whose
width does not match the header. A missing file is not a CSV grammar problem;
it retains its `FileError` / `IOError` meaning.

### Workshop task

Create a semicolon-delimited document with the header `name;midterm;final` and
three students. Parse it as records, calculate `(midterm + final) / 2`, and
print each result. Replace one score with `missing` and verify that you catch
the conversion failure.

See the [CSV reference](CSV.md) for the complete API and boundary conditions.

## 2. Data: reshape a String table

`Data` is the table layer above CSV. It knows column names and can filter,
sort, derive, transform, and group rows. Every cell remains a `String`.

### 2.1 Create and inspect a Table

```ahd
bring Data
from Data bring Table

students: Table := Data.fromCSV(
    "name,department,score\nAli,Math,91\nAyse,Physics,78\nDeniz,Math,85\n"
)

write(students.columns())
write(students.rowCount())
write(students.columnCount())
write(students.row(0))
write(students.column("score"))
```

`row(-1)` returns the last row. An invalid row raises `IndexError`; an unknown
column raises `DataError`. Results from `columns`, `rows`, `row`, and `column`
are snapshots, so changing them never changes the Table.

### 2.2 Filter, sort, and select

```ahd
passed: Table := students.filter(
    lambda (row: Pair<String, String>) -> int(row["score"]) >= 80
)

ranked: Table := passed.sort(
    lambda (row: Pair<String, String>) -> -int(row["score"])
)

summary: Table := ranked.select(["name", "score"])
write(summary.toCSV())
write(students.rowCount())
```

The final line still prints `3`: each transformation returned a new Table.
`sort("score")` would use lexicographic String order, where `"100"` may come
before `"20"`. Use a key function with explicit conversion for numeric order.

### 2.3 Clean and derive a column

`transform` rewrites one existing String column. `derive` creates a new column
from the complete row:

```ahd
clean: Table := students.transform(
    "name"
    lambda (value: String) -> value.trim().capitalize()
)

labelled: Table := clean.derive(
    "status"
    lambda (row: Pair<String, String>) -> {
        if int(row["score"]) >= 80 {
            return "passed"
        }
        return "support needed"
    }
)
```

The callback must return `String`. A numeric result is not silently converted;
write `str(value)` when a derived number should become a cell.

### 2.4 Group and count

```ahd
groups: Pair<String, Table> := students.groupBy("department")

for department in groups {
    group: Local Table := groups[department]
    write("{department}: {str(group.rowCount())}")
}

write(students.valueCounts("department"))
```

Each `groupBy` value is an ordinary Table. There is intentionally no automatic
aggregation language; explicitly convert the numeric column of each group and
calculate what you need.

### 2.5 Cross into Statistics

```ahd
bring Statistics

scores: List<Real> := students.column("score").map(
    lambda (value: String) -> real(value)
)

write(Statistics.mean(scores))
write(Statistics.median(scores))
write(Statistics.stdDev(scores))
```

The boundary is deliberate: `Data` understands table shape, while
`Statistics` understands numeric lists.

### 2.6 Data checkpoints

Before publishing a transformed dataset, check at least these questions:

- Are all expected column names present?
- Is the row count in the expected range?
- Can every numeric cell be converted?
- Is an empty String allowed, or should it be an error?
- If order matters, did you use numeric or lexicographic ordering?

### Workshop task

Group the students by department. Convert the score column of each group to a
`List<Real>` and calculate its mean. Then keep scores of 80 or above, order
them from highest to lowest, and write `passed.csv`.

See the [Data reference](DATA.md) for every transformation and error contract.

## 3. Plot: turn data into a readable chart

`Plot` creates PNG, SVG, and PDF charts from `List<Int>`, `List<Real>`, or a
Numeric Vector. It does not clean data or coerce `"91"` into a number; perform
that conversion explicitly at the Data boundary.

### 3.1 Move from Data to Plot

```ahd
bring Data
from Data bring Table
bring Plot
from Plot bring Chart

students: Table := Data.fromCSV(
    "name,score\nAli,91\nAyse,78\nDeniz,85\n"
)

names: List<String> := students.column("name")
scores: List<Real> := students.column("score").map(
    lambda (value: String) -> real(value)
)

chart: Chart := Plot.bar(names, scores)
chart = chart.title("Exam Scores")
chart = chart.xLabel("Student")
chart = chart.yLabel("Score")
chart.save("scores.png")
```

`title`, `xLabel`, and `yLabel` return new Chart values, so assign the result.
`save` writes a file and returns `Nothing`.

### 3.2 Choose the right chart

| Question | Good starting point |
|---|---|
| How do values differ across categories? | `Plot.bar` |
| How does a value change over time or order? | `Plot.line` |
| How do two numeric variables move together? | `Plot.scatter` |
| What is the shape of one numeric distribution? | `Plot.histogram` |
| How do group distributions compare? | `Plot.box` |

Line and scatter charts require equally sized x/y inputs. Empty data, invalid
bin counts, and mismatched lengths raise `PlotError`.

### 3.3 Statistics summarizes; Plot shows shape

```ahd
bring Statistics

average: Real := Statistics.mean(scores)
spread: Real := Statistics.stdDev(scores)

histogram: Chart := Plot.histogram(scores, 5)
histogram = histogram.title(
    "Mean {str(average)}, standard deviation {str(spread)}"
)
histogram.save("score-distribution.svg")
```

A mean is one summary number; a histogram shows how the same observations are
distributed. Neither replaces the other.

### 3.4 Use one figure in Word and Latex

Save the chart once, then embed that PNG into both documents:

```ahd
bring Word
from Word bring Document
bring Latex as L

word: Document := Word.new()
word = word.heading("Score Distribution", 1)
word = word.image("scores.png", {"width": 14.0})
word.save("score-report.docx")

latexBody: String := L.section("Score Distribution")
latexBody += L.figure(
    "scores.png"
    "Student exam scores"
    "fig:scores"
    {"width": 12.0}
)
latexBody += "Figure " + L.ref("fig:scores") + " shows the results.\n"
```

Word and Latex do not redraw the chart; they embed the saved image. Therefore
`chart.save(...)` must succeed first. This Excel version does not embed images
or charts in XLSX: write typed data to Excel and publish visual explanation
through Plot plus Word/Latex.

### 3.5 Chart quality checklist

- Put units in axis labels.
- Check what the category order means.
- Avoid unreadable bar charts with too many categories.
- Change histogram bin count and check whether the story becomes misleading.
- Open the saved figure and visually inspect its title and labels.
- In a report, state the data source and calculation rule beside the chart.

### Workshop task

Create a bar chart and histogram from the same score list. Save both as PNG,
calculate the mean with Statistics, and put it in the title. Embed the bar
chart into both a Word and Latex report.

See the [Plot reference](PLOT.md) for chart types, styles, and subplots.

## 4. Excel: create a real XLSX with typed cells

Keep three objects separate in your mental model:

```text
Workbook  the complete file
Sheet     one named grid in the workbook
Cell      one Blank, String, Int, Real, Bool, or Formula value
```

Coordinates are 1-based: `(1, 1)` is A1 and `(2, 3)` is C2.

### 4.1 The immutable update cycle

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet

book: Workbook := Excel.new().addSheet("Scores")
sheet: Sheet := book.sheet("Scores")

sheet = sheet.setRow(1, 1, [
    Excel.fromString("Name")
    Excel.fromString("Midterm")
    Excel.fromString("Final")
    Excel.fromString("Average")
])

sheet = sheet.setRow(2, 1, [
    Excel.fromString("Ali")
    Excel.fromInt(80)
    Excel.fromInt(92)
    Excel.formula("=AVERAGE(B2:C2)")
])

book = book.withSheet(sheet)
book.save("scores.xlsx")
```

There are two assignments: `setRow` returns a new Sheet, while `withSheet`
returns a new Workbook. If you omit either assignment, the change never
reaches the Workbook you save.

### 4.2 Choose a Cell type explicitly

```ahd
textCell := Excel.fromString("91")
numberCell := Excel.fromInt(91)
decimalCell := Excel.fromReal(91.5)
flagCell := Excel.fromBool(true)
emptyCell := Excel.blank()
formulaCell := Excel.formula("=SUM(B2:B20)")
```

`Excel.fromString("=SUM(A1:A3)")` is safe plain text.
`Excel.formula("=SUM(A1:A3)")` is a formula. AhdCode stores the formula; Excel,
Numbers, or another spreadsheet application calculates it.

### 4.3 Headers and number formats

```ahd
from Excel bring CellStyle

header: CellStyle := Excel.style()
header = header.bold(true)
header = header.fillColor("#1F4E79").textColor("#FFFFFF")
header = header.horizontal("center").border("thin", "#000000")

sheet = sheet.style(sheet.range(1, 1, 1, 4), header)
sheet = sheet.style(
    sheet.range(2, 4, 20, 4)
    Excel.style().numberFormat("0.00")
)
sheet = sheet.columnWidth(1, 24.0)
sheet = sheet.columnWidth(2, 12.0)
```

A style never changes the AhdCode Cell kind. Applying `yyyy-mm-dd` does not
turn a String into a date type.

### 4.4 Merge and Range

```ahd
sheet = sheet.setCell(1, 1, Excel.fromString("Exam Results"))
sheet = sheet.merge(sheet.range(1, 1, 1, 4))
```

A merge keeps its top-left anchor; all covered cells must already be Blank.
Overlapping merges and writes to a non-anchor merged cell raise `ExcelError`.

### 4.5 Save and verify by reading

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell
from Excel bring Range

loaded: Workbook := Excel.read("scores.xlsx")
page: Sheet := loaded.sheet("Scores")
cell: Cell := page.cell(2, 2)

write(cell.kind())
write(cell.int())
used: Range? := page.usedRange()
if used != null {
    write(used.address())
}
```

Use the accessor matching `Cell.kind()`. Calling `int()` on a String Cell, or
`string()` on an Int Cell, raises `ExcelError`. `usedRange()` may be `null` on
an empty Sheet.

### Workshop task

Write midterm and final scores for three students. Make the average column a
formula, style the header, apply `0.00` to numeric output, then read the file
back and verify the first student's name and midterm score.

See the [Excel reference](EXCEL.md) for Range, styles, dimensions, and XLSX
reading limits.

## 5. Word: build a DOCX report with tables and figures

`Word` does not remote-control a Word application. It creates a real `.docx`
package from an AhdCode `Document`, with no Office installation required.

### 5.1 Build a document step by step

```ahd
bring Word
from Word bring Document

document: Document := Word.new()
document = document.heading("Laboratory Report", 1)
document = document.paragraph("Prepared by: Ayse Yilmaz")
document = document.heading("Results", 2)
document = document.table(
    ["Sample", "pH", "Status"]
    [["A", "7.1", "Normal"], ["B", "6.8", "Review"]]
)
document = document.paragraph("End of report", "right", true)
document.save("laboratory.docx")
```

Every content method returns a new Document. `save` returns `Nothing`, so do
not write `document = document.save(...)`.

### 5.2 Positional paragraph parameters

The argument order is:

```text
paragraph(text, align, bold, italic, underline)
```

For centered, bold, italic text:

```ahd
document = document.paragraph(
    "Approved"
    "center"
    true
    true
    false
)
```

Alignment is `left`, `center`, `right`, or `justify`; heading level is between
`1` and `6`.

### 5.3 Embed a Plot figure

```ahd
bring Plot
from Plot bring Chart

chart: Chart := Plot.bar(
    ["Math", "Physics", "History"]
    [88.0, 74.5, 91.0]
)
chart = chart.title("Course Averages")
chart.save("averages.png")

document = document.heading("Chart", 2)
document = document.image("averages.png", {"width": 14.0})
```

Word accepts PNG and JPEG. Supplying one dimension preserves aspect ratio;
`width` and `height` are measured in centimeters.

### 5.4 Move a Data table into Word

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("name,score\nAli,91\nAyse,78\n")
wordRows: List<List<String>> := table.rows().map(
    lambda (row: Pair<String, String>) -> [row["name"], row["score"]]
)

document = document.table(table.columns(), wordRows)
```

This boundary is natural because both Data cells and Word table cells are
Strings. Writing the output column order explicitly avoids accidentally
depending on unrelated Pair construction order.

### 5.5 Read an existing document

```ahd
loaded: Document := Word.read("laboratory.docx")
write(loaded.headings())
write(loaded.paragraphs())
write(loaded.tables())
write(loaded.text())
```

Reading is semantic: it recovers heading, paragraph, and table text. Custom
styles, comments, headers/footers, and every pixel-level Word layout detail are
not preserved. Do not treat a read/save cycle as “open and close without
changing anything.”

### Workshop task

Filter a Data table to students who passed. Add name and score columns to a
Word table, write the Statistics mean in a paragraph, and embed a Plot chart at
12 cm width.

See the [Word reference](WORD.md) for merged cells, image sizing, DOCX reading,
and `WordError`.

## 6. Latex: create an academic PDF or slide deck

The `Latex` module produces document fragments as Strings. Combine them into a
body, wrap it with `Latex.document`, and compile it offline with `Latex.pdf`.

### 6.1 Separate safe prose from raw mathematics

```ahd
bring Latex as L

body: String := ""
body += L.section("Experimental Results")
body += L.escape("Success was 92%, cost was $5, and A&B worked together.")
body += L.equation(r"\bar{x} = \frac{1}{n}\sum_{i=1}^{n} x_i", "eq:mean")
body += "Equation " + L.ref("eq:mean") + " defines the mean.\n"
```

Use `escape` for ordinary prose and `equation` for raw LaTeX mathematics.
Never concatenate untrusted user text directly into a raw document body or
equation.

### 6.2 Article, Report, and Beamer

```ahd
article: String := L.document(
    body: body
    title: "Short Article"
    author: "Ayse Yilmaz"
    type: "Article"
)
```

- `Article` is suitable for assignments, papers, and notes.
- `Report` supports long documents with `chapter`.
- `Beamer` turns `frame(...)` fragments into slides.

A Report:

```ahd
reportBody: String := L.chapter("Findings")
reportBody += L.section("Summary")
reportBody += L.escape("Three experiments were completed.")

report: String := L.document(
    body: reportBody
    title: "Term Report"
    type: "Report"
    margin: 2.5
    color: "#1F4E79"
)
```

A Beamer deck:

```ahd
slides: String := ""
slides += L.frame("Problem", L.escape("What are we measuring?"))
slides += L.frame("Result", L.equation(r"E = mc^2"))

deck: String := L.document(
    body: slides
    title: "Physics Presentation"
    type: "Beamer"
    theme: "Madrid"
)
```

### 6.3 Generate a Plot figure and embed it

```ahd
bring Plot
from Plot bring Chart

scores: List<Real> := [91.0, 78.0, 85.0]
chart: Chart := Plot.bar(["Ali", "Ayse", "Deniz"], scores)
chart = chart.title("Exam Scores").yLabel("Score")
chart.save("averages.png")

body += L.figure(
    "averages.png"
    "Student exam scores"
    "fig:averages"
    {"width": 12.0}
)
body += "Figure " + L.ref("fig:averages") + " compares the scores.\n"
```

Plot creates the image; Latex embeds, captions, numbers, and references it.
Saving the chart is therefore a required earlier step.

### 6.4 Add a table and bibliography

```ahd
body += L.table(
    ["Student", "Score"]
    [["Ali", "91"], ["Ayse", "78"]]
)

body += "See " + L.ref("fig:averages") + ".\n"
body += "The method follows " + L.cite("Source2026") + ".\n"
body += L.bibliography({
    "Source2026": "A. Author, Data Analysis, 2026."
})
```

Ordinary table headers and cells are escaped. Consult the reference for
`mathColumns` when a column intentionally contains raw mathematics.

### 6.5 Compile and keep the source

```ahd
from Latex bring LatexError

source: String := L.document(body: body, title: "Analysis", type: "Report")

attempt {
    L.pdf(source, "analysis.pdf", "analysis.tex")
    write("analysis.pdf is ready")
}
except LatexError as error {
    write("compilation failed: {error.message}")
}
```

The third argument preserves the `.tex` source, which is useful for learning
and diagnosing compilation failures. The offline renderer must have been
staged once as described in the student guide.

### Workshop task

Create a Report containing a title, two sections, a labelled equation, a
table, a Plot figure, and one bibliography entry. Keep both `.pdf` and `.tex`.

See the [Latex reference](LATEX.md) for every document parameter, theorem, and
layout helper.

## 7. HTTP and HTTPS: requests, responses, and failures

`HTTP` has two distinct roles:

```text
Server                 your program accepts incoming requests
Client / ClientRequest your program sends a request to another service
```

This workshop focuses on outgoing HTTP(S). Use “A small web page” in the
student guide for the local Server side.

### 7.1 Read a URL

```text
https://api.example.com:443/v1/students?active=true
\___/   \_____________/ \_/ \__________/ \_________/
scheme         host     port      path        query
```

HTTPS protects HTTP with TLS. AhdCode verifies the server with the system's
trusted roots and has no option to disable certificate verification.

### 7.2 GET and inspect the response

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientResponse

client: Client := HTTP.client(10)
response: ClientResponse := client.get("https://example.com/")

write(response.status())
write(response.url())
contentType: String? := response.header("Content-Type")
if contentType != null {
    write(contentType)
}
write(response.body())
```

`HTTP.client(10)` uses a ten-second timeout for the complete request. A header
may be absent, so `header(...)` returns `String?`.

### 7.3 Separate HTTP error statuses from transport failure

```ahd
from HTTP bring HTTPError

attempt {
    response: Local ClientResponse := client.get("https://example.com/missing")
    if response.status() >= 400 {
        write("server status: {str(response.status())}")
    } else {
        write(response.body())
    }
}
except HTTPError as error {
    write("request could not be sent: {error.message}")
}
```

`404`, `429`, and `500` are responses from a server you reached, so they return
`ClientResponse`. DNS, TLS, invalid URL, and timeout failures have no valid
HTTP response and raise `HTTPError`.

### 7.4 JSON POST and secrets

```ahd
bring JSON
from JSON bring JSONValue
bring Env

token: String := Env.getOr("API_TOKEN", "")
payload: JSONValue := JSON.object({
    "name": JSON.fromString("Ayse")
    "score": JSON.fromInt(91)
})

request := HTTP.clientRequest("POST", "https://api.example.com/v1/results")
request = request.withHeader("Authorization", "Bearer {token}")
request = request.withHeader("Content-Type", "application/json")
request = request.withBody(JSON.stringify(payload))

response := client.send(request)
if response.status() >= 200 and response.status() < 300 {
    parsed: JSONValue := JSON.parse(response.body())
    write(JSON.stringify(parsed))
}
```

Do not put the token in source code. `withHeader` replaces the value for that
name, while `addHeader` appends another value. Both return a new immutable
ClientRequest.

### 7.5 Safe client checklist

- Use a reasonable timeout for every external request.
- Check status before parsing a success body.
- If you expect JSON, consider both `Content-Type` and the actual body.
- Read passwords and API tokens with `Env`; never print them in logs.
- Calling arbitrary user-supplied URLs can create SSRF; your application
  should define the allowed hosts.
- Never try to bypass HTTPS certificate verification.

### Workshop task

Fetch an HTTPS page with a five-second Client. Print its status and final URL,
and process the body only for a 2xx response. Try an unreachable host and a
404 path to observe the two different failure channels.

See the [HTTP reference](HTTP.md) for Server, cookies, sessions, uploads, and
all Client limits.

## 8. HTML: build safe pages and parse documents

The HTML module has two independent halves:

```text
HTML.text / element / document  builds safe markup
HTML.parse / select / first     inspects existing markup
```

`HTML.parse` never downloads a URL or runs JavaScript. When a network request
is needed, fetch a String body with HTTP Client first.

### 8.1 Build a safe dynamic page

```ahd
bring HTML

userName: String := "<script>alert(1)</script>"

page: String := HTML.document(
    "Student Dashboard"
    [
        HTML.element("h1", {}, [HTML.text("Welcome")])
        HTML.element("p", {"class": "student"}, [HTML.text(userName)])
        HTML.element("a", {"href": "/results"}, [HTML.text("Results")])
    ]
)
```

`HTML.text` escapes `<`, `>`, `&`, and quotes, so the user name is displayed
as data rather than executed as markup. `HTTP.html(page)` sets the content
type; it does not perform another escape pass. Never concatenate dynamic data
directly between raw tags.

### 8.2 Parse with null checks

```ahd
bring HTML
from HTML bring HTMLDocument

document: HTMLDocument := HTML.parse(
    "<article class=\"card\"><h2>Exam</h2><a href=\"/1\">Open</a></article>"
)

heading := document.first("article.card > h2")
if heading != null {
    write(heading.text())
}
```

`first` returns `null` when nothing matches; `select` returns an empty List.
Do not write `first(...).text()` without the null guard.

### 8.3 Selectors and element scope

The supported CSS-like subset includes:

- Tag: `article`
- ID: `#main`
- Class: `.card`
- Attribute: `[href]` or `[data-id="42"]`
- Compound selector: `article.card[data-id]`
- Descendant: `article a`
- Direct child: `article > h2`
- Alternatives: `h1, h2`

Selecting on an `HTMLElement` limits the search to that element's subtree:

```ahd
cards := document.select("article.card")
for card in cards {
    title: Local := card.first("h2")
    link: Local := card.first("a[href]")
    if title != null {
        if link != null {
            href: Local String? := link.attr("href")
            if href != null {
                write("{title.text()} -> {href}")
            }
        }
    }
}
```

`:nth-child`, `+`, and `~` are unsupported. An invalid or unsupported selector
raises `HTMLError` instead of guessing.

### 8.4 Fetch with HTTPS, then parse HTML

```ahd
bring HTTP
bring HTML
from HTTP bring ClientResponse
from HTML bring HTMLDocument

response: ClientResponse := HTTP.client(10).get("https://example.com/")
if response.status() == 200 {
    document: HTMLDocument := HTML.parse(response.body())
    title := document.first("h1")
    if title != null {
        write(title.text())
    }
}
```

A relative `/about` link is not automatically resolved to
`https://example.com/about`. This module is not a browser: it has no URL
resolver, JavaScript engine, CSS layout, or screenshot facility.

### 8.5 Ethical and resilient scraping

- Follow the site's terms and access rules.
- Do not send unlimited requests at high speed.
- Expect selectors to break when page design changes.
- Null-check every `first` result.
- For important datasets, store the source URL and retrieval time.
- Content created later by JavaScript may not exist in the plain HTTP body.

### Workshop task

Create a small HTML String containing three `article.card` elements. Extract
each title and `href` into `List<Pair<String, String>>`, then serialize the
records with `CSV.stringifyRecords`.

See the [HTML reference](HTML.md) for the complete builder, parser, and
selector contract.

## 9. Final project: produce two reports from one dataset

This project brings the responsibilities together:

1. Read `results.csv` with `CSV.readRecords`.
2. Create a Table with `Data.fromRecords`.
3. Convert scores explicitly into `List<Real>`.
4. Calculate the mean with `Statistics.mean`.
5. Create a PNG chart with `Plot.bar`.
6. Write typed cells and formulas to `results.xlsx` with Excel.
7. Write a table and figure to `results.docx` with Word.
8. Optionally produce `results.pdf` from the same material with Latex.

Do not write the project in one leap. Verify each boundary:

```text
CSV record count
    ↓
Table columns and row count
    ↓
numeric list length and mean
    ↓
saved chart file
    ↓
Excel Cell kinds after reading the XLSX back
    ↓
Word headings and table text after reading the DOCX back
```

These checks are the difference between “the program ran” and “the output is
correct.”

## Which document should I read?

| Need | Start here | Then use the full reference |
|---|---|---|
| Read or write CSV | Workshop 1 | [CSV](CSV.md) |
| Filter and group a table | Workshop 2 | [Data](DATA.md) |
| Create and embed a chart | Workshop 3 | [Plot](PLOT.md) |
| Produce XLSX | Workshop 4 | [Excel](EXCEL.md) |
| Produce a DOCX report | Workshop 5 | [Word](WORD.md) |
| Produce an academic PDF or slides | Workshop 6 | [Latex](LATEX.md) |
| Call an API or website | Workshop 7 | [HTTP](HTTP.md) |
| Build or parse HTML | Workshop 8 | [HTML](HTML.md) |

Use the [English Student Guide](STUDENT_GUIDE_EN.md) for language fundamentals
and the [compiled example index](../examples/v0.1/README.md) for more complete
programs.
