# PostgreSQL standard module

[English] · [Türkçe](POSTGRESQL_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [MySQL](MYSQL.md) · [SQLite](SQLITE.md) · [Env](ENV.md) · [UUID](UUID.md)

`PostgreSQL` is the compiler-registered `builtin:PostgreSQL` module,
introduced in AhdCode v1.4.0. It connects to a PostgreSQL server over the
network using `github.com/jackc/pgx/v5`, a pure-Go implementation of the
PostgreSQL wire protocol, vendored into AhdCode itself so a generated
PostgreSQL program builds without touching the network (see "Offline builds"
below). There is no `libpq`, no CGO, and no external helper process.

`PostgreSQL` is a separate module in the [MySQL](MYSQL.md) family, with its
own type names rather than a shared `Database` abstraction. The two modules
look alike on purpose, and differ where the servers differ; see
"Differences from MySQL" below. A program can `bring` MySQL, PostgreSQL, and
SQLite at once with no collision.

## Public surface

```text
PostgreSQL.connect(
    host: String
    username: String
    password: String
    port: Int := 5432
    database: String? := null
    security: String := "tls"
    timeoutSeconds: Int := 10
) -> PostgreSQLDatabase

PostgreSQL.nullValue()               -> PostgreSQLValue
PostgreSQL.fromInt(value: Int)       -> PostgreSQLValue
PostgreSQL.fromReal(value: Real)     -> PostgreSQLValue
PostgreSQL.fromString(value: String) -> PostgreSQLValue
PostgreSQL.fromBool(value: Bool)     -> PostgreSQLValue

PostgreSQLDatabase.ping()                                                    -> Nothing
PostgreSQLDatabase.execute(sql: String, params: List<PostgreSQLValue> := []) -> PostgreSQLResult
PostgreSQLDatabase.query(sql: String, params: List<PostgreSQLValue> := [])   -> List<Pair<String, PostgreSQLValue>>
PostgreSQLDatabase.begin()                                                   -> PostgreSQLTransaction
PostgreSQLDatabase.close()                                                   -> Nothing

PostgreSQLTransaction.execute(sql: String, params: List<PostgreSQLValue> := []) -> PostgreSQLResult
PostgreSQLTransaction.query(sql: String, params: List<PostgreSQLValue> := [])   -> List<Pair<String, PostgreSQLValue>>
PostgreSQLTransaction.commit()                                                  -> Nothing
PostgreSQLTransaction.rollback()                                                -> Nothing

PostgreSQLResult.affectedRows() -> Int

PostgreSQLValue.kind()         -> String
PostgreSQLValue.isNull()       -> Bool
PostgreSQLValue.bool()         -> Bool
PostgreSQLValue.int()          -> Int
PostgreSQLValue.real()         -> Real
PostgreSQLValue.string()       -> String
PostgreSQLValue.isBinary()     -> Bool
PostgreSQLValue.binarySize()   -> Int
PostgreSQLValue.binaryBase64() -> String

PostgreSQLError  (derives from Error)
```

`PostgreSQLDatabase`, `PostgreSQLTransaction`, `PostgreSQLResult`, and
`PostgreSQLValue` are opaque built-in Classes: they cannot be constructed
directly and are obtained only from the functions and methods above.

## Connecting

```ahd
bring Env
bring PostgreSQL
from PostgreSQL bring PostgreSQLDatabase

host := Env.getOr("DB_HOST", "127.0.0.1")
username := Env.getOr("DB_USERNAME", "app")
password: String? := Env.secret("DB_PASSWORD")
if password != null {
    db: PostgreSQLDatabase := PostgreSQL.connect(host, username, password, 5432, "app")
}
```

Never hardcode a real password in source. [`Env.secret`](ENV.md) reads
`DB_PASSWORD`, or the file named by `DB_PASSWORD_FILE`, which is how
container platforms mount secrets.

`connect` dials the server, authenticates, and runs a bounded ping before it
returns, so a `PostgreSQLDatabase` you hold is known to be reachable. An
unreachable host, a wrong password, or a missing database raise
`PostgreSQLError`.

### `database` is optional

`null`, or `""`, lets the server choose its default: the database named after
the role. Any other value selects that database for every connection in the
pool.

### Security modes

Exact lowercase values only. There are no aliases and no downgrade:

| Value | Meaning |
|---|---|
| `"tls"` (default) | TLS required; system trust roots; hostname verified; TLS 1.2 or newer; no insecure-skip |
| `"none"` | Explicit plaintext connection |

With `"tls"`, a server that refuses TLS, presents an untrusted or expired
certificate, or does not match the host raises
`PostgreSQL TLS verification failed`. The trust roots can be extended only the
standard way, with `SSL_CERT_FILE`. `"none"` is for trusted local
development; it does not pretend to be secure.

`port` must be in `1..65535`. `timeoutSeconds` must be in `1..9223372036`
and bounds the dial, the TLS handshake, and each statement.

### The environment does not choose the server

Only `connect`'s arguments decide where and how AhdCode connects.
`PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE`,
`PGOPTIONS`, `PGTZ`, `PGAPPNAME`, and every other `PG*` variable, `~/.pgpass`,
`~/.postgresql/` certificates, and service files have no effect.

One exception is visible: when `PGSERVICE` names a service that cannot be
read, `connect` stops with
`PostgreSQL connection failed: PGSERVICE names a service that cannot be read; AhdCode never uses service files, so unset PGSERVICE`.
A readable service file is still ignored, and nothing ever connects
somewhere else.

## Parameterized queries

```ahd
db.execute(
    "INSERT INTO users (name, email, active) VALUES ($1, $2, $3)"
    [PostgreSQL.fromString(name), PostgreSQL.fromString(email), PostgreSQL.fromBool(true)]
)
```

Placeholders are PostgreSQL's own `$1`, `$2`, …. Every statement is sent with
the extended protocol and its values are bound on the server; the SQL text is
never rewritten and a value is never spliced into it. A `?` is ordinary SQL
text and is never translated. Applications must not build SQL by
concatenating untrusted input into the String.

- One call runs exactly one statement. `"UPDATE …; DROP TABLE …"` raises
  `PostgreSQL execution failed: (42601) cannot insert multiple commands into a prepared statement`
  and runs neither.
- The number of values must match the placeholders:
  `PostgreSQL query failed: the statement has 2 placeholder(s) but 1 parameter(s) were passed`.
- `fromString` sends text that the server converts for the place it is used.
  Where the type is ambiguous, cast in SQL: `$1::uuid`, `$1::numeric`,
  `$1::timestamptz`.
- There is no `fromBinary`; binary values are read-only in v1.4.0.

There is no `lastInsertId`. Ask for generated values with `RETURNING` and
read them through `query`:

```ahd
created := db.query(
    "INSERT INTO books (id, title) VALUES ($1::uuid, $2) RETURNING added_at"
    [PostgreSQL.fromString(UUID.v7().string()), PostgreSQL.fromString(title)]
)
write(created[0]["added_at"].string())
```

## Reading rows

Each row is a `Pair` whose keys are the result column labels, the same shape
MySQL and SQLite use. Two columns with the same label
(`SELECT a.id, b.id …`) raise
`query result has duplicate column "id"; alias it with AS`.

### Type mapping

The mapping is decided by each column's declared type, never by its content.

| PostgreSQL | `kind()` | Value |
|---|---|---|
| NULL | `"Null"` | |
| `boolean` | `"Bool"` | `bool()` |
| `smallint`, `integer`, `bigint` | `"Int"` | `int()` |
| `real`, `double precision` | `"Real"` | `real()`; NaN and ±Infinity raise, naming the column |
| `numeric` | `"String"` | exact text such as `"19.990"`; never coerced to Real |
| `uuid` | `"String"` | lowercase canonical text |
| `json`, `jsonb` | `"String"` | the server's text |
| `date` | `"String"` | `YYYY-MM-DD` |
| `timestamp` | `"String"` | `YYYY-MM-DD HH:MM:SS[.ffffff]` |
| `timestamptz` | `"String"` | UTC, `YYYY-MM-DD HH:MM:SS[.ffffff]+00` |
| `bytea` | `"Binary"` | `isBinary()`, `binarySize()`, `binaryBase64()` |
| text family, enums, `time`, `interval`, `inet`, `money`, ranges, other scalars | `"String"` | the server's text |
| arrays, composite and record values | — | raise `PostgreSQLError`, naming the column |

Calling the wrong accessor raises, for example
`int() requires kind Int; this PostgreSQLValue has kind String (check kind() first)`.
`real()` also reads an `"Int"` value.

**Dates and times do not depend on session settings.** `date`, `timestamp`,
and `timestamptz` are read in PostgreSQL's binary form and written by
AhdCode, so `DateStyle` and `TimeZone` never change the text you get. A
`timestamptz` is an instant and is always shown in UTC with `+00`; AhdCode
never changes the session `TimeZone`. `infinity`, `-infinity`, and BC dates
(`"0044-03-15 BC"`) keep PostgreSQL's spelling. Trailing zeros of a fraction
are removed: `07:30:00.120` reads as `"07:30:00.12"`.

**`numeric` stays a String on purpose**, for the same reason as MySQL's
`DECIMAL`: binary floating point cannot hold every decimal fraction.

**Arrays and composite values** are not read in v1.4.0. Convert them in SQL:
`array_to_json(tags)::text`, `row_to_json(address)::text`.

### Binary values

A `bytea` value may hold any byte; it never becomes a String.
`binaryBase64()` is the one way to get it out as text.

## Results

`PostgreSQLResult.affectedRows()` is the row count the server reported for
that one `execute` call.

## Transactions

```ahd
tx: PostgreSQLTransaction := db.begin()
attempt {
    tx.execute("UPDATE accounts SET balance = balance - $1::numeric WHERE id = $2", [...])
    tx.execute("UPDATE accounts SET balance = balance + $1::numeric WHERE id = $2", [...])
    tx.commit()
} except PostgreSQLError as error {
    tx.rollback()
    write(error.message)
}
```

`begin()` pins one pooled connection and starts a `READ COMMITTED`
transaction. PostgreSQL treats a failed statement differently from MySQL:

- After a statement fails, the transaction is aborted. Every later statement
  in it raises `(25P02) current transaction is aborted, commands ignored until end of transaction block`.
- `commit()` on an aborted transaction raises
  `PostgreSQL transaction was rolled back because an earlier statement failed`
  and ends the transaction; nothing was committed.
- `rollback()` after a failed `commit()` does nothing and does not raise, so
  the pattern above never raises twice.
- Otherwise `commit()` and `rollback()` are one-shot: a second call, or any
  statement afterwards, raises
  `this PostgreSQLTransaction is already committed or rolled back`.

There are no savepoints and no isolation-level option in v1.4.0.

## Concurrency and timeouts

A `PostgreSQLDatabase` is a connection pool that is safe for concurrent use;
there is no global lock and no pool-tuning API. Statements run from HTTP
handlers or WebSocket callbacks still run one at a time, because those
callbacks do.

A statement that runs longer than `timeoutSeconds` is cancelled on the server
and raises `PostgreSQL connection timed out`. The pool stays usable.

## Closing

`db.close()` rolls back every transaction still open on the database —
nothing is committed implicitly — and releases the pool. Closing twice does
nothing further. Any later use raises `this PostgreSQLDatabase is closed`.

## Errors

Every failure is `PostgreSQLError`, derived from `Error`:

| Message starts with | When |
|---|---|
| `PostgreSQL connection failed` | dial, authentication, missing database |
| `PostgreSQL connection timed out` | a dial or statement exceeded `timeoutSeconds` |
| `PostgreSQL TLS verification failed` | `"tls"` could not be established or verified |
| `PostgreSQL query failed` | `query` |
| `PostgreSQL execution failed` | `execute` |
| `PostgreSQL transaction failed` | `begin`, `commit`, `rollback` |

Server errors add their SQLSTATE and primary message, for example
`PostgreSQL execution failed: (23505) duplicate key value violates unique constraint "users_email_key"`.
The server's `DETAIL`, `HINT`, and `WHERE` fields are never included, because
they can repeat row values. Connection-stage failures never include the
driver's raw text, and no message ever contains the password.

## Differences from MySQL

| | MySQL | PostgreSQL |
|---|---|---|
| Placeholders | `?` | `$1`, `$2`, … |
| Default port | `3306` | `5432` |
| Generated ids | `MySQLResult.lastInsertId()` | `RETURNING` through `query` |
| Booleans | integers | `boolean` reads as `"Bool"`; `fromBool` binds one |
| Dates and times | the server's text | fixed formats; `timestamptz` in UTC |
| Arrays, composite values | — | raise, naming the column |
| A failed statement in a transaction | the transaction stays usable | the transaction is aborted; `commit()` raises |
| More than one statement per call | not supported | raises, and none run |

## Offline builds

`ahdcode build` on a program that brings `PostgreSQL` copies the pinned
`github.com/jackc/pgx/v5` v5.11.0 source and its dependencies
(`jackc/pgpassfile`, `jackc/pgservicefile`, `jackc/puddle/v2`,
`golang.org/x/sync`, `golang.org/x/text`), embedded in the `ahdcode` binary,
into the program's build workspace as `vendor/`, and builds with
`-mod=vendor`. The build never contacts a module proxy. A program that does
not use PostgreSQL does not receive this tree. See
[`THIRD_PARTY_NOTICES_POSTGRESQL.md`](../THIRD_PARTY_NOTICES_POSTGRESQL.md)
for the licenses.

## Non-goals

No ORM, no query or schema builder, no migrations, no connection strings, and
no shared `Database` interface. Not in v1.4.0: `LISTEN`/`NOTIFY`, `COPY`,
arrays, composite values, savepoints, isolation levels, pool tuning, bound
binary parameters, a `schema` option, and a statement-cache API.
AhdDataStudio and `ahdcode init web` do not offer PostgreSQL yet. PostgreSQL 18
is the tested server, with PostgreSQL 17 also run in CI.

See also: [`examples/v0.1/71_postgresql.ahd`](../examples/v0.1/71_postgresql.ahd) ·
[`examples/v1.4/realtime_attendance`](../examples/v1.4/realtime_attendance/README.md).
