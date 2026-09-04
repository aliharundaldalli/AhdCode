# CSV standard module

[English] · [Türkçe](CSV_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [File and Path](FILESYSTEM.md)

If you are learning this module, start with the [CSV workshop](PRACTICAL_MODULES.md#1-csv-transport-a-text-table-safely)
for rows versus records, quoting, conversion, and error handling; use this page
as the complete API reference.

`CSV` is the compiler-registered `builtin:CSV` module. It is explicit and a
sibling `CSV.ahd` cannot shadow it:

```ahd
bring CSV
from CSV bring CSVError
```

CSV is a text transport module. Every cell remains a `String`; it performs no
number/date inference and provides no table or DataFrame abstraction.

## Raw rows

```text
parse(text: String, delimiter: String = ",") -> List<List<String>>
stringify(rows: List<List<String>>, delimiter: String = ",") -> String
read(path: String, delimiter: String = ",") -> List<List<String>>
write(path: String, rows: List<List<String>>, delimiter: String = ",") -> Nothing
```

Raw parsing permits variable-width rows. Standard quoting is supported:
delimiters and newlines may appear inside quoted fields, and a quote is escaped
as `""`. LF and CRLF input, Unicode, and empty fields are accepted. The Go CSV
writer defines deterministic output; `stringify([])` is `""`.

```ahd
rows: List<List<String>> := CSV.parse("name,note\nAli,\"hello, world\"\n")
write(rows[1][1])
```

## Records

```text
parseRecords(text, delimiter = ",") -> List<Pair<String, String>>
readRecords(path, delimiter = ",") -> List<Pair<String, String>>
stringifyRecords(records, delimiter = ",") -> String
writeRecords(path, records, delimiter = ",") -> Nothing
```

The first row supplies headers. Empty input and a header-only document produce
an empty List. Headers must be non-empty and unique, and every data row must
have exactly the header width.

When records are written, the first Pair fixes column order. Later Pairs may
use a different insertion order, but must contain exactly the same keys.
Missing or extra keys raise `CSVError`. An empty records List stringifies to
`""`.

## Delimiters and errors

The delimiter must contain exactly one valid Unicode scalar and cannot be
quote, CR, or LF. Empty, multi-scalar, invalid UTF-8, and unsupported delimiter
values raise `CSVError`. Malformed quoting and record/header shape errors also
raise the catchable `CSVError`, which derives directly from `Error`.

File access failures from `read`/`write` preserve `FileError`/`IOError`
semantics. Relative paths use the process working directory; in the persistent
REPL that is the directory from which the REPL was launched.

```ahd
attempt {
    CSV.parse("a,\"unfinished")
} except CSVError as error {
    write(error.message)
}
```

## CSV is transport, Data is the table layer

CSV deliberately stops at text transport: it parses and serializes String rows
and header-keyed String records, and never infers a type. When you want to
filter, sort, group, or derive columns over that data, hand it to the
[Data module](DATA.md), whose `Table` is built on these same String cells and
reuses this module's reader and writer.
