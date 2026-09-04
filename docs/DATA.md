# Data standard module

[English] · [Türkçe](DATA_TR.md)

[Back to README](../README.md) · [KeyValue](KEYVALUE.md) · [CSV](CSV.md) · [Modules](MODULES.md)

If you are learning this module, start with the [Data workshop](PRACTICAL_MODULES.md#2-data-reshape-a-string-table)
for filtering, numeric ordering, derivation, grouping, and the Statistics
boundary; use this page as the complete behavior reference.

Data is a small, strict, immutable table layer built on the existing String,
List, Pair, Function, Lambda, and CSV machinery. It is explicit, like Math,
Time, Regex, and CSV:

```ahd
bring Data
from Data bring Table
from Data bring DataError
```

The canonical identity is `builtin:Data`; a sibling `Data.ahd` cannot shadow
it. Every argument is `NonNull`.

The division of labour is deliberate:

```text
CSV        text transport
Data       table structure and transformation
your code  numeric work, through explicit conversion
```

## Every cell is a String

A `Table` cell is always a `String`. Data never infers `Int`, `Real`, `Bool`,
`DateTime`, or `null`, and never converts a value implicitly:

```text
"95"     stays String
"3.14"   stays String
"true"   stays String
""       stays an empty String
```

An empty cell is an empty `String`. It is **not** `null`, and Data has no
missing-value model in v0.1.12 — no `NA`, no `fillNull`, no `dropNull`.

When you want a number, convert explicitly, exactly as anywhere else in
AhdCode:

```ahd
total: Int := int(row["score"])
weight: Real := real(row["weight"])
```

That is also how a numeric List is produced:

```ahd
values: List<Real> := table.column("score").map(
    lambda (value: String) -> real(value)
)
```

The canonical row is `Pair<String, String>`, and a collection of rows is
`List<Pair<String, String>>`.

## Creating a Table

```text
Data.fromRows(columns: List<String>, rows: List<List<String>>) -> Table
Data.fromRecords(records: List<Pair<String, String>>)          -> Table
Data.fromCSV(text: String, delimiter: String = ",")            -> Table
Data.readCSV(path: String, delimiter: String = ",")            -> Table
```

`Table` is compiler-supplied: it is never constructed directly, only produced
by these functions or by another Table operation.

`fromRows` keeps the column order you give. Column names must be non-empty and
unique, and every row must have exactly `len(columns)` cells; a mismatch raises
`DataError`. Rows are never padded or truncated, and no column name is invented.
Zero rows are valid and keep the schema.

`fromRecords` takes the **first** record as the canonical column order. Later
records may use any insertion order but must carry exactly the same key set;
a missing or extra key raises `DataError`. Values are copied into canonical
order, and the caller's Pairs are never mutated. Empty records produce an empty
zero-column Table, because no schema can be inferred.

```ahd
table: Table := Data.fromRecords([{"b": "2", "a": "1"}, {"a": "3", "b": "4"}])
write(table.columns())
```

=>

```text
["b", "a"]
```

## CSV integration

`fromCSV` and `readCSV` reuse the CSV module's reader, so quoting, escaped
quotes, embedded newlines, LF/CRLF, Unicode, and custom delimiters behave
exactly as in [CSV](CSV.md). Data defines no second CSV grammar.

The first CSV row is the header. Unlike `CSV.parseRecords`, Data keeps the
schema of a header-only document:

```ahd
table: Table := Data.fromCSV("name,score\n")
write(table.columns())
write(table.rowCount())
```

=>

```text
["name", "score"]
0
```

Empty CSV input produces zero columns and zero rows. Header names must be
non-empty and unique, and every data row must match the header width.

## Immutability and snapshots

Every Table operation is pure. `select`, `drop`, `rename`, `filter`, `sort`,
`reverse`, `derive`, `transform`, `head`, and `tail` all return a **new**
Table and leave the source untouched. There is no `setCell`, `appendRow`,
`deleteRow`, or in-place mode in v0.1.12.

Everything a Table hands back is a fresh snapshot, so mutating a result can
never reach the Table:

```ahd
columns: List<String> := table.columns()
columns.add("injected")
write(table.columns())
```

The Table's own columns are unchanged. The same holds for `row()`, `rows()`,
and `column()`. The Table's internal storage is not a published attribute: it
cannot be read, and `has` does not report it.

## Shape and access

```text
rowCount()            -> Int
columnCount()         -> Int
columns()             -> List<String>
rows()                -> List<Pair<String, String>>
row(index: Int)       -> Pair<String, String>
column(name: String)  -> List<String>
```

`row` follows the ordinary List index rules, so a negative index counts from
the end and an invalid index raises `IndexError`. An unknown column name raises
`DataError` rather than quietly returning an empty value.

## head and tail

```text
head(count: Int = 5) -> Table
tail(count: Int = 5) -> Table
```

A count larger than `rowCount()` returns every row; zero returns a Table with
no rows but the same columns. A negative count raises `DataError`. Row order is
preserved.

## select, drop, and rename

```text
select(columns: List<String>) -> Table
drop(columns: List<String>)   -> Table
rename(oldName: String, newName: String) -> Table
```

`select` uses the requested order as the output order. `drop` keeps the
original order of the columns that remain. Both require every named column to
exist and reject a repeated name in the request; neither silently ignores an
unknown name.

`rename` preserves the column's position. The new name must be non-empty, and
it may not collide with a different existing column.

## filter

```text
filter(function: Function) -> Table
```

The contract is exactly `(Pair<String, String>) -> Bool`, checked at compile
time. The callback receives a row snapshot, runs exactly once per row in source
order, and the rows it accepts are kept in source order.

```ahd
adults: Table := table.filter(
    lambda (row: Pair<String, String>) -> int(row["age"]) >= 18
)
```

## sort

```text
sort(column: String)     -> Table
sort(function: Function) -> Table
```

`sort(column)` is a stable, ascending, lexical String ordering of that column.
`sort(function)` uses a key Function returning `Int`, `Real`, or `String`,
reusing the List keyed-sort contract: stable, ascending, and the key runs
exactly once per row. There is no comparator callback and no descending flag —
negate a numeric key, or use `reverse()`:

```ahd
ranked: Table := table.sort(
    lambda (row: Pair<String, String>) -> -int(row["score"])
)
```

## reverse

```text
reverse() -> Table
```

Reverses row order and leaves the columns unchanged.

## transform and derive

```text
transform(column: String, function: Function) -> Table
derive(name: String, function: Function)      -> Table
```

`transform` rewrites one existing column through `(String) -> String`. Only
that column changes, and its position is preserved. The callback must return a
`String`; there is no implicit conversion from `Int` or `Real`.

```ahd
cleaned: Table := table.transform("name", lambda (value: String) -> value.trim())
```

`derive` appends a new column built by `(Pair<String, String>) -> String` from
each complete row. The name must be non-empty and must not already exist —
rewriting an existing column is `transform`'s job, not `derive`'s.

```ahd
labelled: Table := table.derive(
    "status",
    lambda (row: Pair<String, String>) -> str(int(row["score"]) >= 60)
)
```

## unique, valueCounts, and groupBy

```text
unique(column: String)      -> List<String>
valueCounts(column: String) -> Pair<String, Int>
groupBy(column: String)     -> Pair<String, Table>
pivotCount(rows: String, columns: String) -> Table
```

All three key on the column's String cell and use first-occurrence order. Rows
inside a group keep source order, and every grouped Table has the same schema
as the source. An unknown column raises `DataError`; an empty table produces an
empty result.

```ahd
groups: Pair<String, Table> := table.groupBy("department")

for department in groups {
    group: Local Table := groups[department]
    write("{department}: {group.rowCount()}")
}
```

Aggregation syntax is deliberately absent; a group is an ordinary Table.

## pivotCount

```text
pivotCount(rows: String, columns: String) -> Table
```

A strict count cross-tabulation: one row per distinct value of the `rows`
column, one generated column per distinct value of the `columns` column, and
each cell the number of source rows in that combination.

```ahd
students: Table := Data.fromCSV(
    "name,department,grade\nAli,Math,A\nAyse,Physics,B\nMehmet,Math,A\nZeynep,Physics,A\n"
)

write(students.pivotCount("department", "grade").toCSV())
```

=>

```text
department,A,B
Math,2,0
Physics,1,1
```

Both axes use first-occurrence order, matching `groupBy` and `valueCounts`, so
the result never depends on map iteration order. An absent combination counts
`"0"` — this is count semantics, not missing data. Counts are `String` cells
like every other cell, so a program converts explicitly to do arithmetic on
them. The arguments are positional because a built-in type operation takes no
named arguments; no new syntax was added for this method.

An unknown column raises `DataError` naming it, and naming the same column for
both axes is rejected rather than silently producing a diagonal.

`pivotCount` is deliberately the only cross-tabulation. It is not a general
pivot: there is no aggregation callback, no value column, no multi-index, and no
missing-value model.

## toCSV and writeCSV

```text
toCSV(delimiter: String = ",")                  -> String
writeCSV(path: String, delimiter: String = ",") -> Nothing
```

Both use the CSV module's writer, so quoting and delimiters match
`CSV.stringify` exactly. Output is the header row followed by the data rows, in
Table column order. A header-only Table writes its header; a zero-column,
zero-row Table produces `""`.

## Errors

`DataError` derives directly from `Error` and covers Data-specific structural
failures: a duplicate or empty column name, a row-width mismatch, a record
key-set mismatch, an unknown column, a repeated `select`/`drop` request, a
negative `head`/`tail` count, and a `derive` target that already exists.

Other domains keep their own error types, so the failure names the layer that
actually failed:

| failure | error |
|---|---|
| Data/Table structure | `DataError` |
| CSV syntax or an invalid delimiter | `CSVError` |
| filesystem access from `readCSV`/`writeCSV` | `FileError` / `IOError` |
| an invalid `row()` index | `IndexError` |

```ahd
attempt {
    write(table.column("age"))
} except DataError as error {
    write(error.message)
}
```

## Table as a value

A Table is an ordinary Class reference. `type(table)` reports `"Table"`, and
`id()` and `same` behave as they do for any other reference. Table does not
implement `CEqual`, `CCompare`, or `CStr`, so `==` and `same` keep ordinary
reference identity — Data does not invent value equality for tables.

## What Data is not

Data is **not** pandas, and is not DataFrame-compatible. v0.1.12 deliberately
has no join, merge, concat, pivot, melt, MultiIndex, index labels, query
strings, SQL, window functions, rolling, resample, categorical dtypes, lazy
execution, or expression trees. It has no schema inference, no automatic
numeric or datetime parsing, and no null inference.

It also has no statistics. `sum`, `mean`, `median`, `variance`, `stdev`,
`quantile`, `correlation`, and `describe` belong to a planned Statistics layer,
which will consume `List<Int>` and `List<Real>` — which is exactly what an
explicit conversion already produces:

```ahd
scores: List<Real> := table.column("score").map(
    lambda (value: String) -> real(value)
)
```

Keeping numeric work explicit is what lets Data stay statically typed instead
of becoming a dynamic value system.
