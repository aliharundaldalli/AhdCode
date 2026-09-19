# Time standard module

[English] · [Türkçe](TIME_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Diagnostics](DIAGNOSTICS.md)

`Time` is the explicit, compiler-registered `builtin:Time` module. A sibling
`Time.ahd` cannot shadow it. Import its Classes before naming them:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

## Surface

```text
now() -> DateTime
utc() -> DateTime
timestamp() -> Int
fromTimestamp(milliseconds: Int) -> DateTime
dateTime(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
dateTimeUTC(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
dateTimeOffset(year, month, day, offsetMinutes, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
monotonic() -> Real
sleep(milliseconds: Int) -> Nothing
duration(milliseconds: Int) -> Duration
between(first: DateTime, second: DateTime) -> Duration
parseISO(text: String) -> DateTime
```

`now` uses the host's local civil time. `utc` uses UTC. `dateTime` constructs a
local civil value, `dateTimeUTC` constructs UTC, and `dateTimeOffset` uses a
fixed offset in whole minutes. The supported offset range is -840..840
inclusive. Invalid civil components, offsets, and unrepresentable timestamps
raise `ValueError`.

The public offset model has minute precision: `offsetMinutes` is always whole
minutes, and every offset AhdCode source can name is a whole minute. A few
historical host-local zones sit at an offset that includes seconds — for
example `Europe/Istanbul` is `+01:55:52` before 1880. Such a moment is still
represented exactly: `offsetMinutes` reports the whole-minute part, and the
leftover seconds are kept as runtime representation rather than being
truncated, so the instant never shifts. The seconds remainder is not a
published attribute, so it is not readable and `has` does not report it.

## Unix milliseconds and conversions

A timestamp is signed milliseconds since `1970-01-01 00:00:00 UTC`.
`Time.timestamp()` reads the current timestamp. `Time.fromTimestamp(value)`
returns its UTC representation; negative timestamps are supported when the
resulting year is in 1..9999.

```ahd
epoch: DateTime := Time.fromTimestamp(0)
turkey: DateTime := epoch.toOffset(180)

write(epoch.timestamp())
write(turkey.hour)
write(epoch.sameMoment(turkey))
```

Conversions preserve the instant:

```text
value.timestamp() -> Int
value.toUTC() -> DateTime
value.toLocal() -> DateTime
value.toOffset(offsetMinutes: Int) -> DateTime
```

`before`, `after`, `sameMoment`, and `Time.between` compare instants, not the
displayed clock fields, so differently offset values compare correctly.

## DateTime

Nine read-only `Int` attributes are available: `year`, `month`, `day`, `hour`,
`minute`, `second`, `millisecond`, `weekday`, and `offsetMinutes`. Weekdays run
Monday=1 through Sunday=7. `offsetMinutes` is the value's offset east of UTC.

Members are `before`, `after`, `sameMoment`, `timestamp`, `toUTC`, `toLocal`,
`toOffset`, `toString`, `toISO`, `add`, and `subtract`. The existing `toString()` output remains
`YYYY-MM-DD HH:MM:SS`; it deliberately does not append milliseconds or an
offset. `str(value)` and `write(value)` show `DateTime(` followed by the
`toISO()` text and `)` (v1.8.0), and a Duration shows as
`Duration(1500 ms)`.

`DateTime` does not implement `CCompare` or `CEqual`. Use the named instant
operations; ordinary `==` and `same` retain Class identity semantics.

## Validation, Duration, and Calendar

Civil constructors validate year 1..9999, Gregorian dates, hour 0..23, minute
and second 0..59, and millisecond 0..999. `DateTime` and `Duration` cannot be
constructed directly.

`Duration` exposes read-only `milliseconds: Int` and `seconds: Real`.
`between(first, second)` means `second - first` and may be negative.

```text
duration.add(other: Duration) -> Duration
duration.subtract(other: Duration) -> Duration
duration.negate() -> Duration
duration.abs() -> Duration
```

Duration arithmetic is exact Int millisecond arithmetic. A result outside Int
raises `OverflowError`; so do `negate()` and `abs()` of the most negative
Duration. A Duration is only a length of time: there are no months, years, or
calendar periods.

```text
Calendar.isLeapYear(year: Int) -> Bool
Calendar.daysInMonth(year: Int, month: Int) -> Int
Calendar.weekday(year: Int, month: Int, day: Int) -> Int
```

`monotonic()` returns elapsed seconds on a non-decreasing clock. `sleep` takes
milliseconds; zero returns immediately and a negative value raises
`ValueError`.

## ISO 8601 text

`Time.parseISO(text)` reads one strict subset of RFC 3339, and
`value.toISO()` writes it:

```text
YYYY-MM-DDTHH:MM:SSZ
YYYY-MM-DDTHH:MM:SS±HH:MM
YYYY-MM-DDTHH:MM:SS.fZ          (.f, .ff, or .fff before Z or ±HH:MM)
```

- The `T` separator and the time-zone designator are required: `Z` for UTC or
  a `±HH:MM` offset in -14:00..+14:00. The letters are uppercase, the digits
  ASCII, and nothing may precede or follow the value.
- A fraction has one to three digits and means milliseconds: `.3` is 300 ms
  and `.35` is 350 ms. Four or more digits are rejected rather than rounded.
- Everything else raises `ValueError`: a date without a time, a time without
  a designator, a space instead of `T`, `Sep 18 2026`, `18/09/2026`,
  `UTC+3`, `Europe/Istanbul`, and impossible values such as
  `2026-02-30T00:00:00Z`, hour 24, or second 60.

The result keeps the written offset: `Time.parseISO("2026-09-18T13:30:00+03:00")`
has `offsetMinutes` 180. `toISO()` always writes three millisecond digits, `Z`
for a zero offset, and `±HH:MM` otherwise, so every value it writes reads back
as the same moment: `Time.parseISO(value.toISO()).sameMoment(value)` is true.

```ahd
bring Time

meeting := Time.parseISO("2026-09-18T13:30:00.5+03:00")
write(meeting.toISO())
write(meeting.toUTC().toISO())
write(meeting.offsetMinutes)
attempt {
    Time.parseISO("2026-09-18 13:30")
} except ValueError as error {
    write("rejected")
}
```

```text
2026-09-18T13:30:00.500+03:00
2026-09-18T10:30:00.500Z
180
rejected
```

A historical local offset that includes seconds (see above) has no `±HH:MM`
form, so `toISO()` raises `ValueError` for it. Write such a moment with
`value.toUTC().toISO()`.

## Instant arithmetic

```text
value.add(duration: Duration) -> DateTime
value.subtract(duration: Duration) -> DateTime
```

`add` and `subtract` move the instant by an exact number of milliseconds and
keep the value's offset: a fixed offset stays the same offset and UTC stays
UTC. A local value keeps the offset it had; no daylight-saving rule is applied
to the result. A result outside years 1..9999 raises `ValueError`.
`Time.between(first, second)` is still `second - first`; AhdCode has no `+` or
`-` operator for DateTime or Duration.

```ahd
bring Time

start := Time.parseISO("2026-09-18T23:30:00+03:00")
hour := Time.duration(3600000)
later := start.add(hour)
write(later.toISO())
write(later.subtract(duration: hour.add(hour)).toISO())
write(Time.between(start, later).milliseconds)
write(hour.negate().abs().milliseconds)
```

```text
2026-09-19T00:30:00.000+03:00
2026-09-18T22:30:00.000+03:00
3600000
3600000
```

## Deliberate boundary

Time has UTC and fixed minute offsets, not a timezone database. There are no
named/IANA zones, DST rules or configuration objects, lenient or
natural-language date parsers, format strings, localized names, calendar
periods (months or years), or DateTime operators. `parseISO` reads only the
strict RFC 3339 subset above.
