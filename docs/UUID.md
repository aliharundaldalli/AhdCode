# UUID standard module

[English] · [Türkçe](UUID_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Identity](IDENTITY.md) · [Security](SECURITY.md) · [PostgreSQL](POSTGRESQL.md)

`UUID` is the compiler-registered `builtin:UUID` module, introduced in
AhdCode v1.4.0. It creates, parses, and compares RFC 9562 UUIDs: random
version 4 values and time-ordered version 7 values. It is written with the Go
standard library only and behaves the same in `ahdcode run`, native builds,
and the REPL.

## Public surface

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

`UUIDValue` is an opaque, immutable built-in Class. It has no constructor and
no implicit conversion from or to `String`.

## Which identifier to use

| | `Identity.id()` | `Security.token()` | `UUID.v4()` | `UUID.v7()` |
| --- | --- | --- | --- | --- |
| Purpose | opaque public id | secret credential | interoperable public id | time-ordered public id |
| Entropy | 128 random bits | 256 random bits | 122 random bits | at least 62 random bits plus milliseconds |
| Text | 22 characters, base64url | 43 characters, base64url | 36 characters, lowercase hex | 36 characters, lowercase hex |
| UUID compatible | no | no | yes | yes |
| Secret | no | yes | no | no; reveals its creation time |

- Use [`Identity.id()`](IDENTITY.md) for AhdCode's own public ids. It is
  unchanged in v1.4.0 and is still what the Web starters use.
- Use [`Security.token()`](SECURITY.md) for anything that must stay secret:
  sessions, password-reset links, API tokens. A UUID is never a secret.
- Use `UUID.v4()` or `UUID.v7()` when another system expects a UUID, such as a
  PostgreSQL `uuid` column or an external API.

There is no conversion between these kinds of values.

## Creating UUIDs

```ahd
bring UUID
from UUID bring UUIDValue

random: UUIDValue := UUID.v4()
ordered: UUIDValue := UUID.v7()
write(ordered.string())      // 01a0a0c9-8fa6-733f-9f09-81191abcce97
write(ordered.version())     // 7
```

`v4` takes 122 bits from the operating system's cryptographic random source.

### Version 7 ordering

A `v7` value starts with the Unix time in milliseconds and a sub-millisecond
fraction, followed by 62 random bits.

- **Promised:** within one process, every `UUID.v7()` is greater than the
  previous one, in `compare` and in text order. This holds even when the
  system clock moves backwards: the embedded time then continues from the
  last value issued until the clock catches up.
- **Not promised:** ordering between two processes or machines within the
  same millisecond, an exact creation time, or unguessability. Anyone who sees
  a `v7` value can read roughly when it was created.
- More than 4096 values per millisecond, sustained, move the embedded time
  ahead of the clock by the size of the burst.

Because newer values sort later, a `v7` primary key keeps index inserts near
the end of the index.

## Parsing and validation

```ahd
bring UUID
from UUID bring (UUIDValue, UUIDError)

id: UUIDValue := UUID.parse("017F22E2-79B0-7CC3-98C4-DC0C0C07398F")
write(id.string())                                   // 017f22e2-79b0-7cc3-98c4-dc0c0c07398f
write(UUID.isValid(r"{017f22e2-79b0-7cc3-98c4-dc0c0c07398f}"))  // false

attempt {
    UUID.parse("017f22e279b07cc398c4dc0c0c07398f")
}
except UUIDError as error {
    write(error.message)
}
```

UUID text is exactly 36 characters: hexadecimal digits in groups of 8-4-4-4-12
separated by `-`. Input digits may be upper or lower case; output is always
lowercase. Surrounding whitespace, braces, a `urn:uuid:` prefix, and the
32-digit form without hyphens are rejected. Every 128-bit value parses,
whatever its version or variant, including the zero UUID.

`parse` raises `UUIDError` and never repeats the rejected text in the message.
`isValid` never raises.

A `{` inside an ordinary AhdCode String starts interpolation. Write text that
contains braces as a raw String, `r"{…}"`, as above.

## Comparing

```ahd
a := UUID.parse("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
b := UUID.parse("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
write(a.equals(b))   // true
write(a == b)        // false: two different UUIDValue objects
write(a.compare(b))  // 0
write(str(a))        // <UUIDValue>
```

- `equals` compares all 128 bits.
- `compare` returns `-1`, `0`, or `1` in RFC 9562 byte order, which is the
  same as comparing the lowercase text.
- `==` keeps the identity meaning it has for every built-in Class, so use
  `equals` to compare values.
- `str(id)` is `<UUIDValue>`, like other built-in Classes. Call `id.string()`
  for the text.
- `version()` is the 4-bit version field as written (`0..15`). `isZero()` is
  true only for `00000000-0000-0000-0000-000000000000`, which `UUID.zero()`
  returns.

## Storing UUIDs

Store the 36-character text. With [PostgreSQL](POSTGRESQL.md), bind it into
a `uuid` column with a cast, and parse it when you read it back:

```ahd
db.execute(
    "INSERT INTO check_ins (id, student) VALUES ($1::uuid, $2)"
    [PostgreSQL.fromString(UUID.v7().string()), PostgreSQL.fromString(student)]
)
rows := db.query("SELECT id FROM check_ins ORDER BY id")
first: UUIDValue := UUID.parse(rows[0]["id"].string())
```

With MySQL or SQLite, a `CHAR(36)` or `TEXT` column works. Because the text is
lowercase and fixed-width, sorting it sorts `v7` values by creation time.

## Errors

Every failure is `UUIDError`, derived from `Error`:

```text
UUID text must be 36 characters in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
UUID random generation failed
UUID v7 requires a system clock between 1970 and the year 10889
```

## Non-goals

No version 1, 3, 5, 6, or 8 generation, no name-based UUIDs, no 16-byte binary
form, no conversion to or from `Identity.id()`, and no global uniqueness
registry. v1.4.0 does not add operators for `UUIDValue`: comparison is
`equals` and `compare`.

See also: [`examples/v0.1/69_uuid.ahd`](../examples/v0.1/69_uuid.ahd) ·
[`examples/v1.4/realtime_attendance`](../examples/v1.4/realtime_attendance/README.md).
