# Time standard module

[Back to README](../README.md) · [Modules](MODULES.md) · [Math module](MATH.md)

Time is explicit, like Math. AhdCode has no namespace-qualified type syntax, so
a type is imported before it is named:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

The canonical identity is `builtin:Time`; a sibling `Time.ahd` cannot shadow
it. Every argument must be `NonNull`.

## Surface

```text
now() -> DateTime
monotonic() -> Real
sleep(milliseconds: Int) -> Nothing
duration(milliseconds: Int) -> Duration
between(first: DateTime, second: DateTime) -> Duration
dateTime(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime

Calendar.isLeapYear(year: Int) -> Bool
Calendar.daysInMonth(year: Int, month: Int) -> Int
Calendar.weekday(year: Int, month: Int, day: Int) -> Int
```

## Local time only

`now` reports host local time and `dateTime` builds a local civil moment. v0.1
has no UTC conversion, timezone objects or names, fixed offsets, DST
configuration, or timezone parsing and formatting.

## DateTime

Eight read-only `Int` attributes: `year`, `month`, `day`, `hour`, `minute`,
`second`, `millisecond`, `weekday`. Weekdays run Monday=1 to Sunday=7.

Every attribute is `Constant`, so `value.year = 2030` is the ordinary Constant
diagnostic rather than a Time-specific rule.

Members: `before`, `after`, `sameMoment`, and `toString`. `toString` is
deterministic and locale-independent, formatted `YYYY-MM-DD HH:MM:SS`;
milliseconds are read through the `millisecond` attribute. `str(value)` renders
`<DateTime>`, because Class attributes are never printed automatically.

There is no operator overloading, so ordering is `before`/`after` rather than
`<`/`>`. `==` and `same` keep the ordinary Class identity rule, so two
separately built equal moments are not `==`; `sameMoment` is the value
comparison.

## Validation

`dateTime` checks `year` 1..9999, `month` 1..12, `day` against that month of
that year, `hour` 0..23, `minute` 0..59, `second` 0..59, and `millisecond`
0..999. An impossible moment raises the catchable `ValueError` rather than
rolling over, so `2026-02-29` and `2026-02-30` are both rejected while
`2028-02-29` is valid.

`DateTime` and `Duration` are never constructed directly; they come only from
the Time functions, which validate first.

## Duration

Read-only `milliseconds: Int` and `seconds: Real`. A Duration may be negative,
and the sign is preserved rather than reduced to a magnitude.

`between(first, second)` is `second - first`, so reversing the arguments gives
a negative Duration and the same moment twice gives zero.

## Calendar

`Calendar` answers questions about the Gregorian calendar without a DateTime. A
leap year divides by 4, except a century year, which must divide by 400: 2028
and 2000 are leap years, 2100 and 1900 are not. An invalid year, month, or
date raises `ValueError`. v0.1 has no month or day names, localization, or
calendar rendering.

## Elapsed time and waiting

`monotonic` reports **seconds** on a clock that never moves backwards. Only
differences are meaningful; the absolute value has no calendar meaning.

`sleep` takes **milliseconds**. Zero returns immediately, and a negative
request raises `ValueError` rather than being clamped.

```ahd
start: Real := Time.monotonic()
Time.sleep(100)
elapsed: Real := Time.monotonic() - start
```

## Not in this version

No format strings, no `parse`, no ISO-8601 or RFC 3339 reader, no month or day
names, and no natural-language dates.
