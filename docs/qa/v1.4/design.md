# v1.4.0 Realtime Web & Data design

Status: **approved and frozen**. This file is the contract implementation
follows. A change to anything below needs a concrete contradiction with the
existing AhdCode design and an explicit approval; "it became easy" is not a
reason.

Baseline: `origin/main` `d3d4c84` (v1.3.0). Development branch: `v1.4.0-dev`.

## Scope

v1.4.0 contains exactly:

1. `bring UUID` — UUID v4 and v7, strict parsing, the zero UUID.
2. `bring PostgreSQL` — a network database module in the MySQL family.
3. WebSocket **server** support hosted in `HTTP`, with a `Web` passthrough.
4. `Env.secret(name)` — the `NAME_FILE` secret convention.
5. Acceptance programs, one realtime dogfood application, documentation, QA.

### Explicitly deferred — do not add to v1.4.0

| Deferred | Reason |
| --- | --- |
| AhdDataStudio PostgreSQL support | doubles database UI QA |
| `ahdcode init web` PostgreSQL bootstrap | multiplies the starter matrix |
| Switching generated starters to `Env.secret` | starter change outside the approved scope |
| WebSocket client | server side is the release |
| Binary WebSocket messages, subprotocols, compression | no `Bytes` type; compression attack surface |
| WebSocket registration on `RouteSet` / `RouteGroup` | guards need `RequestContext.respond`, which an upgrade cannot honour |
| Graceful 1001 "going away" on shutdown | HTTP has no shutdown hook |
| EventBus / PubSub | an untyped bus breaks static typing; a typed one needs generic class declarations |
| OpenAPI / Swagger UI | no static route table, no typed bodies, no path parameters |
| Scoped API auth framework | `Security` + guards are the primitives; auth stays application code |
| CORS, trusted proxies, `Request.remoteAddress`, security headers | one later Web hardening release |
| Local HTTPS / certificate authority | privileged machine-wide state, unchanged |
| PostgreSQL LISTEN/NOTIFY, COPY, arrays, composite types, savepoints, isolation levels, pool tuning, bound binary parameters, `schema` option, statement cache API | MySQL parity; smaller frozen surface |
| `UUID.max()`, `UUIDValue.timestamp()`, `UUID.tryParse`, v1/v3/v5/v6/v8 | no approved use case |
| `CEqual` / `CCompare` / `CStr` on any built-in class | no compiler precedent for UUID |

### Slip policy

- **PostgreSQL may slip.** If it grows substantially beyond this design or
  cannot reach the QA bar in reasonable scope, it moves cleanly to v1.5.0 and
  v1.4.0 ships as UUID + WebSocket server + `Env.secret` ("Realtime Web").
  QA is never reduced to keep it.
- **WebSocket may not slip silently.** If this design proves unsafe or
  incompatible with HTTP's serialization or lifecycle model, implementation
  stops and the finding is reported before any re-scoping or versioning.

## Compatibility

- No grammar change. No type-system change. No rename or removal.
- Existing HTTP, Web, MySQL, SQLite, Security, Env, and Identity programs
  compile and behave the same.
- HTTP handler serialization is preserved and extended to WebSocket callbacks.
- A program that uses neither WebSocket nor PostgreSQL gets an unchanged
  build workspace (no new vendor tree).
- `UUID` and `PostgreSQL` become reserved standard module names; a sibling
  `UUID.ahd` / `PostgreSQL.ahd` can no longer be brought (same effect as QR,
  Barcode, Cron in earlier minors). This goes in the release notes.
- Minimum Go for source builds stays 1.26 (CI uses `go.mod`).

## Identity.id, Security.token, UUID

| | `Identity.id()` | `Security.token()` | `UUID.v4()` | `UUID.v7()` |
| --- | --- | --- | --- | --- |
| Purpose | opaque public id | secret credential | interoperable public id | time-ordered public id |
| Entropy | 128 random bits | 256 random bits | 122 random bits | ≥ 62 random bits + ms time |
| Text | 22 chars base64url | 43 chars base64url | 36 chars lowercase hex | 36 chars lowercase hex |
| UUID compatible | no | no | yes | yes |
| Secret | no | yes | no | no; leaks creation time |

`Identity.id()` is unchanged and remains the starters' `public_id`. No
conversion exists between these values.

## UUID

```text
bring UUID
from UUID bring (UUIDValue, UUIDError)

UUID.v4()                  -> UUIDValue
UUID.v7()                  -> UUIDValue
UUID.parse(text: String)   -> UUIDValue
UUID.isValid(text: String) -> Bool
UUID.zero()                -> UUIDValue

UUIDValue.string()                  -> String
UUIDValue.version()                 -> Int
UUIDValue.isZero()                  -> Bool
UUIDValue.equals(other: UUIDValue)  -> Bool
UUIDValue.compare(other: UUIDValue) -> Int

UUIDError  (derives from Error)
```

- `UUIDValue` is an opaque, immutable built-in Class. No constructor, no
  implicit conversion from or to `String`.
- `==` keeps built-in Class identity semantics. `str(id)` is `<UUIDValue>`.
  Canonical text is `id.string()`.
- Text grammar: exactly 36 characters, `8-4-4-4-12` hexadecimal digits with
  `-` at offsets 8, 13, 18, 23. Hex digits are case-insensitive on input;
  output is always lowercase. Whitespace, braces, `urn:uuid:`, and the
  32-digit form are rejected. Every 128-bit value parses (any version,
  variant, the zero UUID, the max UUID).
- `parse` raises `UUIDError` without echoing the input. `isValid` never raises.
- `version()` is the 4-bit version field (`0..15`) as written.
- `isZero()` is true only for `00000000-0000-0000-0000-000000000000`.
- `equals` compares all 128 bits. `compare` returns `-1`, `0`, or `1` in
  RFC 9562 §6.11 big-endian byte order, which equals lowercase text order.
- `v4`: 122 bits from `crypto/rand`, version 4, variant `10`.
- `v7`: RFC 9562 §5.7. 48-bit Unix milliseconds, then a 12-bit
  sub-millisecond fraction in `rand_a` (§6.2 method 3), together a 60-bit
  clock value; 62 bits from `crypto/rand` in `rand_b`; version 7, variant
  `10`. Under one process-wide lock: `next = max(now60, last60 + 1)`.
  - **Promise:** within one process every `UUID.v7()` is strictly greater
    than the previous one in byte and text order, including when the wall
    clock moves backwards (the embedded time then continues from the last
    issued value until the clock catches up — RFC 9562 §6.2).
  - Under sustained generation above 4096 per millisecond the embedded time
    runs ahead of the clock by the size of the burst.
  - **Not promised:** ordering across processes or machines within one
    millisecond; exact creation time; unguessability.
  - A clock before 1970 or past the 48-bit millisecond range raises
    `UUIDError`.
- Entropy failure raises `UUIDError`.
- Implementation: Go standard library only, in `ahdruntime`, shared by the
  evaluator and native programs. No dependency on Go 1.27's `uuid` package.

Messages:

```text
UUID text must be 36 characters in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
UUID random generation failed
UUID v7 requires a system clock between 1970 and the year 10889
```

## Env.secret

```text
Env.secret(name: String) -> String?
```

| `NAME` | `NAME_FILE` | Result |
| --- | --- | --- |
| absent | absent | `null` |
| present (even `""`) | absent | value of `NAME` |
| absent | non-empty path | file contents |
| absent | `""` | `EnvError` |
| present | present | `EnvError` (mutually exclusive) |

- `name` follows `Env.set` validation.
- The path is used as given (relative to the working directory); symlinks
  are followed.
- The file must be at most 1 MiB, valid UTF-8, and contain no NUL byte.
- Exactly one trailing `\n` or `\r\n` is removed; nothing else is trimmed.
- An unreadable or missing file raises `EnvError`; it never falls back to
  `NAME`.
- Messages name the variable; they never include the value, the file
  contents, or the path.

## WebSocket server

Home: `HTTP` owns the route table, the handler mutex, and the server
lifecycle. `Web` passes through.

```text
HTTP.websocket(onMessage: Function(WebSocket, String) -> Nothing) -> WebSocketEndpoint

WebSocketEndpoint.withOpen(handler: Function(WebSocket, Request) -> Nothing)      -> WebSocketEndpoint
WebSocketEndpoint.withClose(handler: Function(WebSocket, Int, String) -> Nothing) -> WebSocketEndpoint
WebSocketEndpoint.withAccept(check: Function(Request) -> Response?)               -> WebSocketEndpoint
WebSocketEndpoint.withAllowedOrigins(origins: List<String>)                       -> WebSocketEndpoint
WebSocketEndpoint.withMaxMessageBytes(bytes: Int)                                 -> WebSocketEndpoint
WebSocketEndpoint.withMaxQueuedMessages(count: Int)                               -> WebSocketEndpoint
WebSocketEndpoint.withMaxConnections(count: Int)                                  -> WebSocketEndpoint

Server.websocket(path: String, endpoint: WebSocketEndpoint) -> Nothing

WebSocket.id()                                           -> String
WebSocket.send(text: String)                             -> Bool
WebSocket.close(code: Int := 1000, reason: String := "") -> Nothing
WebSocket.isOpen()                                       -> Bool

Web.websocket(onMessage: Function(WebSocket, String) -> Nothing) -> WebSocketEndpoint
App.websocket(path: String, endpoint: WebSocketEndpoint)         -> Nothing
Web re-exports WebSocket and WebSocketEndpoint.
```

Defaults and ranges: `maxMessageBytes` 65536 (`1..16777216`),
`maxQueuedMessages` 64 (`1..4096`), `maxConnections` 1024 (`1..1000000`).

`WebSocketEndpoint` is immutable configuration (like `Cookie`) and holds no
live connections. Broadcasting is application code over an explicit
registry. `WebSocket.id()` is a process-unique opaque identifier that is
never reused.

### Handshake pipeline

The WebSocket branch runs before the dispatcher reads a request body.

1. Route match: exact path or trailing `/*`, same rules as HTTP routes.
   Registering a path that also has a `GET` route raises `HTTPError`;
   registration after `start()` raises `HTTPError`. A non-`GET` method is
   405 with `Allow`.
2. Missing or invalid upgrade headers: `426 Upgrade Required` with
   `Sec-WebSocket-Version: 13` when the request is not an upgrade attempt;
   `400` when it is a malformed one.
3. Connection limit reached: `503`.
4. Origin policy: `403`.
   - Default (no `withAllowedOrigins`): same-origin. No `Origin` header is
     allowed (non-browser client). Otherwise the Origin host, including a
     non-default port, must equal the request `Host`, case-insensitively.
   - `withAllowedOrigins`: exact `scheme://host[:port]` match after
     lowercasing. No wildcards, no paths; `"*"`, an empty list, or a
     malformed origin raises `HTTPError` at configuration time.
5. `withAccept` check, under the server mutex: a returned `Response` is
   written as-is and no upgrade happens; `null` continues; a raised error is
   a 500 and is logged like a handler failure.
6. `onOpen(socket, request)` under the server mutex, before the upgrade
   response. Messages sent, and a close requested, before the upgrade
   completes wait in the socket's queue.
7. `101 Switching Protocols`: no subprotocol, compression disabled; read and
   write deadlines inherited from the HTTP server are cleared on the
   hijacked connection. If the upgrade fails after `onOpen`,
   `onClose(1006, "")` still runs.

Approved change during implementation: the original order sent `101` before
`onOpen`. Dogfood showed a client could then see the connection open before
the application had registered the socket, so a broadcast in that window
silently missed it.

### Concurrency and lifecycle

- Every WebSocket callback runs holding the **same** per-server mutex as HTTP
  handlers. No two handlers or callbacks on one Server ever run at the same
  time. A slow callback delays every request, exactly like a slow handler.
- Per socket: `onOpen` exactly once and first; `onMessage` in arrival order;
  `onClose(code, reason)` exactly once and last. A rejected handshake runs no
  callback. When the client sees the connection open, `onOpen` has already
  returned.
- No re-entrancy: `close()` inside a callback schedules `onClose` after the
  current callback returns.
- Inbound backpressure: the next frame is read only after the previous
  callback returned. There is no inbound queue.
- Outbound: `send` never blocks. It enqueues on a bounded per-socket queue
  drained by the runtime. `send` returns `false` when the socket is closed or
  closing, or when the queue is full; a full queue closes the socket with
  `1008` and reason `outbound queue full`. Each frame write has an internal
  10-second deadline; a stalled write is an abnormal close.
- Text frames only, validated as UTF-8 (invalid: `1007`). A binary frame
  closes with `1003`. A message larger than `maxMessageBytes` closes with
  `1009`. Fragmented messages are reassembled up to the limit.
- Ping/pong is automatic and internal: a ping every 30 seconds; no pong
  within 30 seconds is an abnormal close. Client pings are answered.
- `close(code, reason)` accepts `1000`, `1001`, `1008`, `1011`, and
  `3000..4999`, with a reason of at most 123 UTF-8 bytes; anything else raises
  `HTTPError`. Closing an already closed or closing socket does nothing. The
  close handshake waits at most 5 seconds.
- A lost connection, ping timeout, or write stall reports `onClose(1006, "")`.
  `1006` is never sent on the wire.
- An error raised by `onOpen` or `onMessage` is written to stderr (not to the
  peer), the socket closes with `1011`, and `onClose` still runs. An error
  raised by `onClose` is only logged. The server keeps serving.
- A disconnect is not an `HTTPError`.
- Process exit (Ctrl+C, `ahdcode kill`, a `dev` rebuild) drops connections
  without close frames. There is no automatic reconnection.
- No goroutine, channel, thread, or lock concept is exposed to AhdCode.

### Security

Same-origin default against cross-site WebSocket hijacking; authentication
before upgrade through `withAccept`; bounded message size, queue, and
connection count; no compression; UTF-8 validation; message contents never
logged; errors never include payloads. Sessions are readable in `withAccept`
and `onOpen`; there is no response to commit, so session mutation there is
unsupported.

Library: `github.com/coder/websocket` v1.8.15 (ISC, no dependencies),
vendored. AhdCode performs its own origin check and passes
`InsecureSkipVerify` to the library only after that check passed.

## PostgreSQL

```text
bring PostgreSQL
from PostgreSQL bring (PostgreSQLDatabase, PostgreSQLTransaction, PostgreSQLResult, PostgreSQLValue, PostgreSQLError)

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

### Connection

- `connect` dials and runs a bounded ping before returning.
- `database` `null` or `""`: PostgreSQL's protocol default (the database
  named after the role).
- `security`: `"tls"` requires TLS with system roots and hostname
  verification; `"none"` is explicit plaintext. No other value, no downgrade.
- `port` `1..65535`; `timeoutSeconds` `1..9223372036` bounds dial, TLS
  handshake, and each statement. A statement that times out is cancelled on
  the server best-effort.
- `PG*` environment variables, `~/.pgpass`, and service files have no effect.
  One approved exception (recorded during implementation): pgx reads the
  service file whenever `PGSERVICE` is set, so a `PGSERVICE` naming a service
  that cannot be read makes `connect` fail closed with
  `PostgreSQL connection failed: PGSERVICE names a service that cannot be read; AhdCode never uses service files, so unset PGSERVICE`.
  A readable service still has no effect, and nothing ever connects elsewhere.

### Statements

- Placeholders are PostgreSQL's `$1..$n`. SQL text is never rewritten; `?`
  is never translated.
- Extended protocol with server-side binding and no statement cache.
- More than one statement in one call raises `PostgreSQLError`.
- `fromString` binds text that the server converts in context; use SQL casts
  (`$1::uuid`, `$1::numeric`, `$1::timestamptz`) where context is ambiguous.
- No `lastInsertId`; generated values come from `RETURNING` through `query`.

### Result type mapping

| PostgreSQL | `kind()` | Value |
| --- | --- | --- |
| NULL | `Null` | |
| `boolean` | `Bool` | |
| `smallint`, `integer`, `bigint` | `Int` | |
| `real`, `double precision` | `Real` | NaN or ±Infinity raise, naming the column |
| `numeric` | `String` | canonical text; never coerced to `Real` |
| `uuid` | `String` | lowercase canonical |
| `json`, `jsonb` | `String` | server text |
| `date` | `String` | `YYYY-MM-DD` |
| `timestamp` | `String` | `YYYY-MM-DD HH:MM:SS[.ffffff]` |
| `timestamptz` | `String` | UTC, `YYYY-MM-DD HH:MM:SS[.ffffff]+00`; session TimeZone untouched |
| `bytea` | `Binary` | read-only |
| text family, enums, `time`, `timetz`, `interval`, `inet`, `money`, ranges, other scalars | `String` | server text output |
| arrays, composite / record | — | `PostgreSQLError` naming the column |

Wrong-kind accessors raise `PostgreSQLError`. Duplicate result column labels
raise `PostgreSQLError`.

### Transactions and pool

- `begin()` pins one pooled connection (READ COMMITTED).
- After a failed statement, later statements in that transaction raise.
- `commit()` on an aborted transaction raises
  `transaction was rolled back because an earlier statement failed` and ends
  the transaction. `rollback()` after a failed `commit()` does nothing and
  does not raise. Otherwise `commit` / `rollback` are one-shot as in MySQL.
- The pool is safe for concurrent use; there is no tuning API. `close()` is
  idempotent; use after close raises.

### Errors

Category messages as MySQL: connection failed, connection timed out, TLS
verification failed, query failed, execution failed, transaction failed,
value-kind mismatch. Query and execution failures include the SQLSTATE and
the server's primary message; server `DETAIL`, `HINT`, and `WHERE` fields are
never included. Connection-stage failures never include raw driver text.

Library: `github.com/jackc/pgx/v5` v5.11.0 (MIT) with its pinned
dependencies, vendored.

## Offline builds

- New embedded vendor trees for WebSocket and PostgreSQL, composed with the
  existing MySQL and codes trees; builds stay `-mod=vendor`,
  `GOPROXY=off`, `GOSUMDB=off`.
- The WebSocket tree is required only when a program uses a WebSocket
  export, not whenever it brings `HTTP` or `Web`.
- UUID and `Env.secret` are standard library only.

## CI

- The existing `ci` job stays unchanged and hermetic.
- A separate `postgresql` job runs the integration suite against a
  `postgres:18` service container.
- Locally the suite is opt-in through `AHDCODE_TEST_POSTGRESQL_HOST`,
  `_PORT`, `_USERNAME`, `_PASSWORD`, `_DATABASE`, `_SECURITY`, and skips
  cleanly when they are absent.

## Implementation order

1. This design.
2. UUID.
3. `Env.secret`.
4. Vendor composition for N trees; WebSocket and PostgreSQL trees.
5. WebSocket server (stop and report on a design contradiction).
6. PostgreSQL (may slip).
7. Acceptance programs and the realtime dogfood application.
8. Documentation EN, TR, `FOR_AI.md`, notices, docbundle.
9. Full QA, then report for release approval. No tag, package, or release
   without explicit approval.
