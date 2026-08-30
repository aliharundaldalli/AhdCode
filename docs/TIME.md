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
```

`now` uses the host's local civil time. `utc` uses UTC. `dateTime` constructs a
local civil value, `dateTimeUTC` constructs UTC, and `dateTimeOffset` uses a
fixed offset in whole minutes. The supported offset range is -840..840
inclusive. Invalid civil components, offsets, and unrepresentable timestamps
raise `ValueError`.

The public offset model has minute precision. A historical host-local offset
that contains a seconds component cannot be represented without changing the
instant, so local construction/conversion raises `ValueError` for that rare
case rather than silently truncating it.

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
`toOffset`, and `toString`. The existing `toString()` output remains
`YYYY-MM-DD HH:MM:SS`; it deliberately does not append milliseconds or an
offset. `str(value)` remains the ordinary Class rendering `<DateTime>`.

`DateTime` does not implement `CCompare` or `CEqual`. Use the named instant
operations; ordinary `==` and `same` retain Class identity semantics.

## Validation, Duration, and Calendar

Civil constructors validate year 1..9999, Gregorian dates, hour 0..23, minute
and second 0..59, and millisecond 0..999. `DateTime` and `Duration` cannot be
constructed directly.

`Duration` exposes read-only `milliseconds: Int` and `seconds: Real`.
`between(first, second)` means `second - first` and may be negative.

```text
Calendar.isLeapYear(year: Int) -> Bool
Calendar.daysInMonth(year: Int, month: Int) -> Int
Calendar.weekday(year: Int, month: Int, day: Int) -> Int
```

`monotonic()` returns elapsed seconds on a non-decreasing clock. `sleep` takes
milliseconds; zero returns immediately and a negative value raises
`ValueError`.

## Deliberate boundary

v0.1.11 adds UTC and fixed minute offsets, not a timezone database. There are
no named/IANA zones, DST configuration objects, date parsers, ISO-8601/RFC
3339 readers, format strings, localized names, or natural-language dates.
