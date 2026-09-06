# MySQL standard module

[English] · [Türkçe](MYSQL_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [SQLite](SQLITE.md) · [Env](ENV.md) · [Student Guide](STUDENT_GUIDE_EN.md#51-mysql-a-network-database-server)

`MySQL` is the compiler-registered `builtin:MySQL` module, introduced in
AhdCode v0.11.0. It connects to a real MySQL server over the network using
`github.com/go-sql-driver/mysql`, a pure-Go implementation of the MySQL wire
protocol, vendored into AhdCode itself so a generated MySQL program builds
without ever touching the network (see "Offline builds" below). There is no
`mysql` client library dependency, no CGO, and no external helper process.

`MySQL` is deliberately a separate module from [`SQLite`](SQLITE.md), with
its own type names (`MySQLDatabase`, `MySQLTransaction`, `MySQLResult`,
`MySQLValue`, `MySQLError`) rather than a shared `Database`/`Value`
abstraction. SQLite is a local file; MySQL is a network server with its own
connection lifecycle, authentication, and transaction model — collapsing them
into one generic interface would blur real differences instead of
simplifying anything. A program can `bring` both at once with no collision.

## Public surface

```text
MySQL.connect(
    host: String
    username: String
    password: String
    port: Int := 3306
    database: String? := null
    security: String := "tls"
    timeoutSeconds: Int := 10
) -> MySQLDatabase

MySQL.nullValue()                    -> MySQLValue
MySQL.fromInt(value: Int)            -> MySQLValue
MySQL.fromReal(value: Real)          -> MySQLValue
MySQL.fromString(value: String)      -> MySQLValue

MySQLDatabase.ping()                                              -> Nothing
MySQLDatabase.execute(sql: String, params: List<MySQLValue> := []) -> MySQLResult
MySQLDatabase.query(sql: String, params: List<MySQLValue> := [])   -> List<Pair<String, MySQLValue>>
MySQLDatabase.begin()                                              -> MySQLTransaction
MySQLDatabase.close()                                              -> Nothing

MySQLTransaction.execute(sql: String, params: List<MySQLValue> := []) -> MySQLResult
MySQLTransaction.query(sql: String, params: List<MySQLValue> := [])   -> List<Pair<String, MySQLValue>>
MySQLTransaction.commit()                                              -> Nothing
MySQLTransaction.rollback()                                            -> Nothing

MySQLResult.affectedRows() -> Int
MySQLResult.lastInsertId() -> Int?

MySQLValue.kind()   -> String
MySQLValue.isNull() -> Bool
MySQLValue.int()    -> Int
MySQLValue.real()   -> Real
MySQLValue.string() -> String
MySQLValue.isBinary()     -> Bool
MySQLValue.binarySize()   -> Int
MySQLValue.binaryBase64() -> String

MySQLError  (derives from Error)
```

`MySQLDatabase`, `MySQLTransaction`, `MySQLResult`, and `MySQLValue` are
opaque built-in Classes: they cannot be constructed directly, have no public
attributes, and are obtained only from the functions and methods above.

## Connecting

```ahd
bring Env
bring MySQL
from MySQL bring MySQLDatabase

host := Env.getOr("MYSQL_HOST", "127.0.0.1")
username := Env.getOr("MYSQL_USERNAME", "app")
password := Env.getOr("MYSQL_PASSWORD", "")

db: MySQLDatabase := MySQL.connect(host, username, password)
```

Never hardcode a real password in source; read it with [Env](ENV.md), the
same convention [SMTP](SMTP.md) uses.

`connect` does not merely construct a lazy handle: it dials the server and
runs a bounded ping before returning, so a `MySQLDatabase` you hold is
already known to be reachable. Connecting to an unreachable host, a wrong
password, or a wrong username all raise `MySQLError`, mapped to a small set
of category messages — never the raw driver error, which could otherwise
echo back connection details.

### `database` is optional

```ahd
admin: MySQLDatabase := MySQL.connect(host, username, password, 3306, null, "tls")
rows := admin.query("SHOW DATABASES")
```

Passing `null` (or simply omitting the argument) connects without selecting
a default database — the credentials alone decide what that connection can
see. This is what lets a database-administration tool discover every
database/schema a set of credentials is authorized to see with `SHOW
DATABASES` or `INFORMATION_SCHEMA`, before any one of them is chosen. An
empty String (`""`) is treated exactly like omitting it. Passing a real
schema name selects that database for every later statement on that
connection, exactly like any ordinary MySQL client.

### Security modes

Exact lowercase values only — no aliases, no downgrade:

| Value | Meaning |
|---|---|
| `"tls"` (default) | TLS required; system trust roots; hostname verified; no insecure-skip |
| `"none"` | Explicit unencrypted connection |

If `security` is `"tls"` and the server's certificate is untrusted, expired,
or its identity does not match the host you connected to, `connect` raises
`MySQLError`. There is no public trust-all or hostname-skip switch — the same
posture [SMTP](SMTP.md)'s `"tls"` mode takes. `"none"` is explicit and
intended for trusted local development; it does not pretend to be secure.

`port` must be in `1..65535`. `timeoutSeconds` must be in
`1..9223372036` and bounds the dial, the TLS handshake, and every later
statement's network I/O on that connection — the same duration contract
[HTTP](HTTP.md) and [SMTP](SMTP.md) use, chosen so a `time.Duration`
conversion can never overflow.

## Parameterized queries

```ahd
db.execute(
    "INSERT INTO users (name, email) VALUES (?, ?)"
    [MySQL.fromString(name), MySQL.fromString(email)]
)
```

Every `?` is a real server-side bound parameter — the SQL text is never
rewritten, and a value is never spliced into it. A String stored this way
that happens to look like SQL (`Robert'); DROP TABLE users;--`) is stored as
ordinary text; the table is never touched. This is the correct defense
against SQL injection. AhdCode does not additionally sanitize or escape SQL
text, and applications must not build queries by concatenating untrusted
input into the SQL String itself.

`MySQL.nullValue()`, `.fromInt`, `.fromReal`, and `.fromString` build a
`MySQLValue`. There is no implicit conversion from a bare `String` or `Int`
argument, and no `MySQL.fromBinary`: parameterized binary input is out of
scope for v0.11.0, since `MySQLValue.isBinary()` only ever appears on a value
read back from a query.

## Reading rows

```ahd
rows: List<Pair<String, MySQLValue>> := db.query(
    "SELECT id, name FROM users ORDER BY id"
)
for row in rows {
    write("{row["id"].int()}: {row["name"].string()}")
}
```

Each row is a `Pair` whose keys are the result column labels in result-column
order, the same shape [SQLite](SQLITE.md) uses. A query whose result has two
columns with the same label (`SELECT a.id, b.id`) raises `MySQLError` — alias
one of them with `AS` rather than have AhdCode silently pick one.

### Type mapping

| MySQL type | `kind()` | Read with |
|---|---|---|
| `NULL` | `"Null"` | `isNull()` |
| `TINYINT` … `BIGINT`, `YEAR` (signed or unsigned, fits `Int`) | `"Int"` | `int()` |
| `FLOAT`, `DOUBLE` | `"Real"` | `real()` |
| `CHAR`, `VARCHAR`, `TEXT` family, `ENUM`, `SET` | `"String"` | `string()` |
| `DECIMAL` / `NUMERIC` | `"String"` | `string()` |
| `DATE`, `TIME`, `DATETIME`, `TIMESTAMP` | `"String"` | `string()` |
| `JSON` | `"String"` | `string()` |
| unsigned integer too large for `Int` (near `BIGINT UNSIGNED`'s ceiling) | `"String"` | `string()` |
| `BLOB` family, `BINARY`, `VARBINARY`, `BIT` | `"Binary"` | `isBinary()`, `binarySize()`, `binaryBase64()` |

The mapping is decided by the column's own declared type, never guessed from
the text — the same discipline [SQLite](SQLITE.md) applies. Calling the wrong
accessor (`int()` on a `"String"` value, `string()` on `"Int"`) raises
`MySQLError` rather than silently converting.

**`DECIMAL` stays a `String` on purpose.** A `DECIMAL(10,2)` value like
`"19.99"` is never coerced into `Real`, because binary floating point cannot
represent every decimal fraction exactly — coercing it would silently corrupt
money and other exact quantities. Convert explicitly if you actually need
arithmetic (and mind the precision loss when you do).

**Dates and times are plain `String`s** (e.g. `"2026-01-15 10:30:00"`),
exactly as MySQL sends them. v0.11.0 has no temporal storage type; treat the
text as opaque or parse it with [Time](TIME.md) yourself.

### Binary values

```ahd
value: MySQLValue := rows[0]["payload"]
if value.isBinary() {
    write(str(value.binarySize()))
    encoded := value.binaryBase64()
}
```

A `BLOB` may contain any byte, including `NUL` and invalid UTF-8; it never
becomes an AhdCode `String` and is never forced through one. `binaryBase64()`
is the one way to get it out as text. There is no public `Bytes` type in
v0.11.0, and binary parameters bound *into* a query are out of scope — this
value model is read-only.

## Results

```ahd
result: MySQLResult := db.execute("UPDATE users SET active = ? WHERE id = ?", [...])
changed: Int := result.affectedRows()
newID: Int? := result.lastInsertId()
```

`MySQLResult` is immutable and belongs to the one `execute` call that
produced it — it is never a mutable "last result" hanging off the
`MySQLDatabase`, which matters the moment two requests share one connection
concurrently. `lastInsertId()` is `null` when the statement generated no new
id (an `UPDATE`, or an `INSERT` into a table with no `AUTO_INCREMENT`
column) — never a spurious value carried over from an earlier statement.

## Transactions

```ahd
tx: MySQLTransaction := db.begin()
attempt {
    tx.execute("UPDATE accounts SET balance = balance - ? WHERE id = ?", [...])
    tx.execute("UPDATE accounts SET balance = balance + ? WHERE id = ?", [...])
    tx.commit()
} except MySQLError as error {
    tx.rollback()
    write(error.message)
}
```

`begin()` returns an independent `MySQLTransaction` that pins its own
underlying connection: two transactions opened from the same `MySQLDatabase`
— including from two concurrent requests — never interfere with each other,
and neither does an ordinary `db.execute` running alongside them. `commit()`
and `rollback()` are each one-shot: calling either a second time, or using
the transaction afterward, raises `MySQLError` rather than silently doing
nothing.

## Concurrency

A `MySQLDatabase` is safe for concurrent use from multiple requests — it is
backed by Go's ordinary `database/sql` connection pool, the same pooling
every other Go MySQL client relies on. AhdCode does not wrap it in one global
lock: independent `execute`/`query` calls run concurrently, and only an
explicit `MySQLTransaction` groups statements together. There is no public
pool-tuning API in v0.11.0; the runtime's own conservative defaults apply.

## Closing

```ahd
db.close()
```

Releases pooled connections. Closing twice safely does nothing further; any
operation on a `MySQLDatabase` after `close()` — or on a `MySQLTransaction`
after `commit()`/`rollback()` — raises `MySQLError` rather than reusing torn-
down state.

## Errors

Every failure is `MySQLError`, derived from `Error`, with a small set of
category messages: connection failed, connection timed out, TLS verification
failed, query failed, execution failed, transaction failed (including
already-closed reuse), or a value-kind mismatch. Query/execution failures
include the server's own error code and message, since that text originates
on the server and never carries your password; connection-stage failures
never include the driver's raw error text, so a password or address embedded
in it can never leak into a diagnostic, log, or stack trace.

## Offline builds

Unlike a hand-written third-party dependency, the vendored driver ships
inside AhdCode itself: `ahdcode build` on a program that `bring`s `MySQL`
copies the exact pinned `github.com/go-sql-driver/mysql` and
`filippo.io/edwards25519` source (embedded in the `ahdcode` binary at
`internal/backend/golang/ahdruntime/mysqlvendor`) into that program's private
build workspace as `vendor/`, then builds with `go build -mod=vendor`. The
build never contacts a module proxy and never depends on the local Go module
cache being warm — a MySQL-using AhdCode program is exactly as reproducibly
buildable offline as every other generated program. A program that does not
use MySQL is completely unaffected: it keeps today's bare, dependency-free
`go.mod`.

This is the same design principle AhdCode already applies to
[`Latex.pdf`](LATEX.md)'s Tectonic engine: reimplementing a mature protocol
client or rendering engine from scratch would add risk without adding
language value, so AhdCode embeds the real implementation — pinned, audited
once at development time, and shipped offline — behind its own small, typed,
error-checked surface. The MySQL wire protocol (including
`caching_sha2_password`'s RSA key exchange) is exactly that kind of code:
AhdCode's job is the safe, typed `connect`/`execute`/`query`/`begin` contract
above, not re-deriving a production-grade protocol implementation. See
[`THIRD_PARTY_NOTICES_MYSQL.md`](../THIRD_PARTY_NOTICES_MYSQL.md) for the
vendored code's licenses.

## Non-goals

No ORM or active-record layer, no model classes, no query or schema builder,
no migrations, no connection-string/DSN mini-language, no stored-procedure or
trigger framework, no replication or binlog API, no administration server,
no connection pool tuning API, and no generic `Database` interface shared
with [SQLite](SQLITE.md). Bound binary parameters and a public `Bytes` type
are also out of scope for v0.11.0. MariaDB may work incidentally over the
same wire protocol, but only MySQL 8.x is a tested target.

See also: [Student Guide — MySQL](STUDENT_GUIDE_EN.md#51-mysql-a-network-database-server) ·
[`examples/v0.11`](../examples/v0.11/README.md).
