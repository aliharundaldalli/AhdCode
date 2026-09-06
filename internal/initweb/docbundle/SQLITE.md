# SQLite standard module

[English] · [Türkçe](SQLITE_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [JSON](JSON.md) · [Student Guide](STUDENT_GUIDE_EN.md#35-sqlite-a-database-that-remembers)

`SQLite` is the compiler-registered `builtin:SQLite` module, introduced in
AhdCode v0.3.0. It is explicit and a sibling `SQLite.ahd` cannot shadow it:

```ahd
bring SQLite
from SQLite bring Database
from SQLite bring SQLiteValue
from SQLite bring SQLiteError
```

`SQLite` is a safe, typed bridge to a real local SQLite database. You write
ordinary SQL; AhdCode performs parameter binding and typed value conversion,
and nothing else. There is no ORM, no query builder, no schema inference, no
migration framework, and no `Any`: every value that crosses from SQL into
AhdCode is a `SQLiteValue` whose kind you read explicitly.

## Public surface

```text
SQLite.open(path: String)         -> Database
SQLite.nullValue()                -> SQLiteValue
SQLite.fromInt(value: Int)        -> SQLiteValue
SQLite.fromReal(value: Real)      -> SQLiteValue
SQLite.fromString(value: String)  -> SQLiteValue

Database.execute(sql: String, parameters: List<SQLiteValue> = []) -> Int
Database.query(sql: String, parameters: List<SQLiteValue> = [])   -> List<Pair<String, SQLiteValue>>
Database.lastInsertId()                                           -> Int
Database.begin()                                                  -> Nothing
Database.commit()                                                 -> Nothing
Database.rollback()                                               -> Nothing
Database.close()                                                  -> Nothing

SQLiteValue.kind()    -> String     // "Null", "Int", "Real", or "String"
SQLiteValue.isNull()  -> Bool
SQLiteValue.int()     -> Int
SQLiteValue.real()    -> Real
SQLiteValue.string()  -> String

SQLiteError  (derives from Error)
```

`Database` and `SQLiteValue` are opaque built-in Classes: they cannot be
constructed with `Database()` or `SQLiteValue()`, have no public attributes,
and are obtained only from the functions above. All arguments are positional.
`Int` widens to `Real` for `SQLite.fromReal`, exactly as in an ordinary
`x: Real := 3` assignment.

## Opening a database

```ahd
db: Database := SQLite.open("notes.db")
memory: Database := SQLite.open(":memory:")
```

`path` is a filesystem path, or the exact marker `":memory:"` for a private
in-memory database that disappears when the `Database` is closed or the
program ends. Ordinary file behavior follows SQLite:

- a missing database file is created;
- parent directories are **not** created; opening `data/app.db` when `data/`
  does not exist raises `SQLiteError`;
- a relative path is resolved against the program's working directory (in the
  REPL, the directory the REPL was started in);
- paths may contain spaces and non-ASCII characters.

The path is never interpreted as a URI or DSN; there is no query-string
syntax. AhdCode does not invent a database directory and never writes a file
you did not name.

## Storage classes, not declared types

SQLite stores each individual value with one of five runtime storage classes.
AhdCode maps the **storage class of each value**, never the declared column
type:

| SQLite storage class | `SQLiteValue.kind()` | Read with          |
| -------------------- | -------------------- | ------------------ |
| `NULL`               | `"Null"`             | `isNull()`         |
| `INTEGER`            | `"Int"`              | `int()` or `real()` |
| `REAL`               | `"Real"`             | `real()`           |
| `TEXT`               | `"String"`           | `string()`         |
| `BLOB`               | — unsupported —      | raises `SQLiteError` |

Consequences you should expect:

- A column declared `BOOLEAN` holds `INTEGER` values `0`/`1`; you read them
  with `int()`. There is no Bool inference.
- A column declared `DATE` or `DATETIME` holding `'2026-09-02'` is `TEXT`;
  you read it with `string()` and parse it yourself when the application
  needs to. There is no date inference.
- A column declared `REAL` stores the bound `Int` `7` as `REAL` `7.0`, so
  `kind()` reports `"Real"`. That is SQLite's own type affinity at work.
- A column declared without a type (or `TEXT` holding `'12'`) keeps whatever
  storage class the inserted value had. `'12'` stays `"String"`; `int()` on
  it raises `SQLiteError` rather than parsing it.
- `real()` accepts kind `Real` and kind `Int` (widened exactly as
  `Real := Int`). `int()` accepts only `Int`; `string()` accepts only
  `String`. A `Null` is never readable as a number or text.
- A `REAL` that SQLite computes as infinity or NaN (for example
  `1e308 * 10`) cannot be an AhdCode `Real` and raises `SQLiteError`.
- Querying a `BLOB` value raises `SQLiteError` naming the column. Bytes are
  never stringified or base64-encoded silently; other columns of the same
  table remain readable.

SQL `NULL` is a `SQLiteValue` of kind `Null`, **not** an AhdCode `null`. A
query row is always structurally `Pair<String, SQLiteValue>`, and the
language's nullable system is not involved.

## Parameters: real binding, never interpolation

```ahd
db.execute(
    "INSERT INTO students (name, score) VALUES (?, ?)",
    [
        SQLite.fromString("Ayşe")
        SQLite.fromReal(91.5)
    ]
)

rows := db.query(
    "SELECT id, name, score FROM students WHERE score >= ? ORDER BY id",
    [SQLite.fromReal(80.0)]
)
```

Positional `?` placeholders are the supported public style. Each `?` is bound
to the `SQLiteValue` at the same position through SQLite's own parameter
binding API. The SQL text is passed to SQLite unchanged; parameter values
travel separately and are never spliced, escaped, or quoted into the text.
Therefore a value such as

```text
Robert'); DROP TABLE notes;--
```

is stored as exactly that text: it is data, never SQL. The same holds for
quotes, semicolons, newlines, backslashes, Turkish characters, and emoji.

The number of placeholders must equal the number of parameters, otherwise
`SQLiteError` is raised before anything runs. Named parameters (`:name`,
`@name`, `$name`) are not part of the v0.3.0 public API.

## One call, one statement

`execute` and `query` run exactly one SQL statement. Text containing a second
statement (`"DELETE FROM a; DELETE FROM b"`) raises `SQLiteError` without
running anything, so a parameter list can never apply to a different
statement than the one you meant. A trailing `;`, whitespace, or comment is
fine. Run multiple statements as multiple calls; use a transaction if they
must succeed or fail together.

## execute

```text
Database.execute(sql: String, parameters: List<SQLiteValue> = []) -> Int
```

Runs one statement and returns the number of rows it inserted, updated, or
deleted (including rows changed by triggers). `CREATE TABLE`, `CREATE INDEX`,
`DROP TABLE`, `PRAGMA`, and other statements that change no rows return `0`.

## query

```text
Database.query(sql: String, parameters: List<SQLiteValue> = []) -> List<Pair<String, SQLiteValue>>
```

Runs one statement and fully materializes every result row before returning.
Each row is a `Pair<String, SQLiteValue>` whose keys are the result column
labels **in result-column order**; the `List` holds the rows **in the order
SQLite returned them**. A query with no matching rows returns an empty
`List`.

```ahd
for row in db.query("SELECT id, title FROM notes ORDER BY id") {
    write("{row["id"].int()}: {row["title"].string()}")
}
```

### Row order is a SQL contract

SQLite does not promise any row order unless the statement has `ORDER BY`.
Rows often come back in insertion order for a simple table, but that is a
storage detail, not a guarantee, and it changes with indexes, deletes, and
query plans. The AhdCode `List` preserves whatever order SQLite produced; it
never invents one. Write `ORDER BY` whenever order matters.

### Duplicate column labels

A `Pair` cannot hold two entries with the same key, so a result row with two
columns of the same label raises `SQLiteError`:

```sql
SELECT a.id, b.id FROM a JOIN b ON b.a_id = a.id      -- SQLiteError: duplicate "id"
```

Give each column a distinct name with `AS`:

```sql
SELECT a.id AS a_id, b.id AS b_id FROM a JOIN b ON b.a_id = a.id
```

AhdCode does not overwrite, rename, or number columns for you.

## lastInsertId

```text
Database.lastInsertId() -> Int
```

Returns SQLite's connection-local `last_insert_rowid()`: the row id of the
most recent successful `INSERT` on **this** `Database`. It is not a general
primary-key discovery API. Call it immediately after the relevant `INSERT`;
another `INSERT` on the same `Database` replaces the value, and a different
`Database` never sees it. With `INTEGER PRIMARY KEY AUTOINCREMENT` this is the
new row's `id`. Before any insert it returns `0`.

## Transactions

```ahd
db.begin()

attempt {
    db.execute(
        "UPDATE accounts SET balance = balance - ? WHERE id = ?",
        [SQLite.fromReal(10.0), SQLite.fromInt(1)]
    )
    db.execute(
        "UPDATE accounts SET balance = balance + ? WHERE id = ?",
        [SQLite.fromReal(10.0), SQLite.fromInt(2)]
    )
    db.commit()
}
except SQLiteError as error {
    db.rollback()
    write(error.message)
}
```

Semantics:

- A `Database` has at most one active transaction. `begin()` while one is
  active raises `SQLiteError`; there are no nested transactions or savepoints.
- `commit()` or `rollback()` with no active transaction raises `SQLiteError`.
- Every `execute`/`query` between `begin()` and `commit()`/`rollback()` runs
  inside that transaction and sees its own uncommitted changes.
- `commit()` publishes every change made since `begin()`; `rollback()`
  discards them all, including statements that succeeded before a later one
  failed.
- A failing statement does **not** end the transaction by itself. Catch the
  `SQLiteError` and call `rollback()` (or `commit()` if you want the earlier
  work kept).
- Outside `begin()`/`commit()`, each statement is its own auto-committed
  transaction.

## Connection model and close

Each `Database` is exactly one logical SQLite connection; there is no hidden
pool. That is what makes `:memory:` databases and transaction state
predictable: `begin`, `execute`, `query`, and `commit` on one `Database`
always observe the same connection.

Assignment aliases the same connection; it never opens a second one:

```ahd
db: Database := SQLite.open(":memory:")
same: Database := db
db.close()
same.execute("SELECT 1")     // SQLiteError: the Database is closed
```

`close()` behavior:

- `close()` releases the connection. Afterwards `execute`, `query`,
  `lastInsertId`, `begin`, `commit`, and `rollback` on that `Database` (and
  on every alias) raise `SQLiteError` with the message
  `the Database is closed`.
- `close()` is idempotent: closing an already closed `Database` succeeds.
- `close()` while a transaction is active raises `SQLiteError` and leaves the
  transaction untouched. Nothing is ever committed or discarded implicitly;
  call `commit()` or `rollback()` first.
- Two `SQLite.open` calls, even on the same path, are two independent
  connections with independent transactions and `lastInsertId()` values. Two
  `SQLite.open(":memory:")` calls are two unrelated empty databases.

When the program (or REPL session) ends, every connection it still holds is
released by the operating system; SQLite's journal keeps the file consistent,
but you should still `close()` explicitly so intent is visible.

## Errors

`SQLiteError` derives from `Error` and is the only Error class the module
raises. Its `message` keeps SQLite's own text where SQLite produced one
(`no such table: notes`, `UNIQUE constraint failed: notes.title`,
`near "SELEC": syntax error`) and an explicit AhdCode explanation otherwise.
Situations that raise it:

- malformed SQL;
- a missing table, column, or function;
- `UNIQUE`, `NOT NULL`, `CHECK`, and `FOREIGN KEY` constraint violations;
- a placeholder/parameter count mismatch, or a non-finite `Real` parameter;
- text containing more than one statement, or no statement;
- a `BLOB` result value, or a non-finite `REAL` result;
- duplicate result column labels;
- `int()`, `real()`, or `string()` on a `SQLiteValue` of the wrong kind;
- `begin`, `commit`, `rollback`, or `close` in an invalid transaction state;
- any operation on a closed `Database`;
- an empty path, an unwritable path, or a missing parent directory in
  `SQLite.open`;
- the bundled SQLite helper being unavailable (see below).

Go driver errors are never exposed directly; classification is always
`SQLiteError`. In the REPL an uncaught `SQLiteError` is reported like any
other error and the session, its variables, and the open `Database`
survive.

## REPL

The module works identically in the persistent REPL:

```text
ahd> bring SQLite
ahd> db := SQLite.open(":memory:")
ahd> db.execute("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
0
ahd> db.execute("INSERT INTO items (name) VALUES (?)", [SQLite.fromString("Çay")])
1
ahd> db.query("SELECT id, name FROM items ORDER BY id")[0]["name"].string()
Çay
```

The `Database` value and the in-memory database persist across successful
entries; a failing SQL entry reports `SQLiteError` and leaves the session
intact. `SQLiteError` must be imported (`from SQLite bring SQLiteError`)
before it is named in `attempt`/`except`, exactly like `JSONError`.

## Editor support

Because `SQLite` is an ordinary compiler-registered module, the AhdCode
language server (v0.2.2 and later) discovers it with no SQLite-specific code:
`bring SQL` completes to `SQLite`, `SQLite.` lists the real members with
their signatures, `from SQLite bring SQL` offers `SQLiteError` and
`SQLiteValue`, and hover/signature help show the compiler's own types. No
editor-extension change was needed for v0.3.0.

## Dependency and portability model

Generated AhdCode programs remain Go-standard-library only. The SQLite engine
lives in the bundled `ahdsqlite` helper (`cmd/ahdsqlite`), which links
[`github.com/ncruces/go-sqlite3`](https://github.com/ncruces/go-sqlite3)
v0.35.4 (MIT), a pure-Go, CGO-free SQLite: the real SQLite C library
(SQLite 3.53.x, public domain) compiled to WebAssembly and then translated to
ordinary Go source by `wasm2go`, so it is compiled by the Go toolchain like
any other package. There is no system `libsqlite3`, no `sqlite3`
command-line tool, no CGO, and no network access.
`CGO_ENABLED=0 go build ./...` succeeds. The helper is installed next to the
compiler:

```bash
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
```

At the first `SQLite.open` the program starts one helper process and talks to
it over a narrow JSON protocol (`internal/sqliteproto`), exactly the way the
`Numeric` and `Plot` modules isolate their own third-party dependencies. Helper
discovery checks `AHDCODE_SQLITE_RUNTIME` (a file or its directory), the
directory recorded at build time, the running executable's directory, and the
installed `libexec/ahdcode` directory. If no helper is found, the first
database operation raises `SQLiteError` explaining how to set
`AHDCODE_SQLITE_RUNTIME`.

Database files written by AhdCode are ordinary SQLite 3 files, readable by
every other SQLite implementation (for example Python's `sqlite3` module or
the `sqlite3` CLI), and AhdCode reads files those tools produce. See
[`THIRD_PARTY_NOTICES_SQLITE.md`](../THIRD_PARTY_NOTICES_SQLITE.md).

## Non-goals

This release is intentionally a SQL bridge and nothing more: no ORM or
active-record layer, no model classes or row-to-Class mapping, no query or
schema builder, no migrations, no relationships, no connection pool API, no
async SQL or background threads, no encryption layer, no SQL rewriting
(AhdCode never injects `LIMIT`, `ORDER BY`, or quoting), no `BLOB` support
until AhdCode has a binary-data type, no named-parameter API, no savepoints,
and no generic database interface shared with a future MySQL module. `SQLite`
is its own module.

See also: [Student Guide — SQLite](STUDENT_GUIDE_EN.md#35-sqlite-a-database-that-remembers) ·
[`examples/v0.3/01_sqlite_notes.ahd`](../examples/v0.3/01_sqlite_notes.ahd).
