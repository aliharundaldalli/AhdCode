# Statistics standard module

[English] · [Türkçe](STATISTICS_TR.md)

[Back to README](../README.md) · [Data](DATA.md) · [Modules](MODULES.md)

Statistics is descriptive statistics over typed numeric Lists. It is explicit,
like Math, Time, Regex, CSV, and Data:

```ahd
bring Statistics
from Statistics bring StatisticsError
```

The canonical identity is `builtin:Statistics`; a sibling `Statistics.ahd`
cannot shadow it. Every argument is `NonNull`.

Statistics does **not** depend on Data. A `Table` cell is a `String`, so a
program converts explicitly before asking for a statistic — which is what keeps
both modules strict instead of introducing a dynamic numeric value.

## Typed input, typed results

Every function is published as an explicit `Int`/`Real` overload pair, resolved
by ordinary overload resolution. There is no weakly typed entry point, so the
static type of a result is always known.

```text
sum(values: List<Int>)   -> Int        sum(values: List<Real>)   -> Real
min(values: List<Int>)   -> Int        min(values: List<Real>)   -> Real
max(values: List<Int>)   -> Int        max(values: List<Real>)   -> Real
range(values: List<Int>) -> Int        range(values: List<Real>) -> Real
mode(values: List<Int>)  -> Int        mode(values: List<Real>)  -> Real

mean(values)           -> Real
median(values)         -> Real
variance(values)       -> Real
sampleVariance(values) -> Real
stdDev(values)         -> Real
sampleStdDev(values)   -> Real
quantile(values, probability: Real) -> Real
```

A statistic whose answer is one of the input's own values — `min`, `max`,
`mode`, and the difference `range` — keeps the element type. A statistic that
averages or measures spread is always `Real`, because the average of whole
numbers is generally not whole.

`Int` results use the language's ordinary checked arithmetic, so an `Int` sum or
range that leaves signed 64-bit range raises `OverflowError` rather than
wrapping.

## No String coercion

A numeric statistic never reads text. This does not compile:

```ahd
Statistics.mean(["10", "20", "30"])
```

Values that arrive as Data cells are converted explicitly:

```ahd
scores: List<Real> := students.column("score").map(
    lambda (value: String) -> real(value)
)

average: Real := Statistics.mean(scores)
```

## Mathematical definitions

`mean` is the arithmetic mean.

`median` is the middle value of the ordered data, averaging the two middle
values when the count is even. It is always `Real`, so the even case needs no
separate rule: `median([1, 2, 3, 4])` is `2.5` and `median([1, 2, 3])` is `2.0`.

`variance` and `stdDev` are the **population** forms, dividing by `n`.
`sampleVariance` and `sampleStdDev` are the **sample** forms, dividing by
`n - 1` (Bessel's correction). Both names are published so the definition is
never left implicit:

```ahd
values: List<Int> := [3, 1, 4, 1, 5]

write(Statistics.variance(values))        // 2.56   population, / n
write(Statistics.sampleVariance(values))  // 3.2    sample, / (n - 1)
```

`stdDev` is the square root of `variance`, and `sampleStdDev` the square root
of `sampleVariance`.

`mode` is the most frequent value. When several values tie for the highest
frequency, the one that occurs **first in the input** wins, so the result never
depends on map iteration order:

```ahd
write(Statistics.mode([2, 3, 3, 2]))  // 2
write(Statistics.mode([3, 2, 2, 3]))  // 3
```

`quantile(values, probability)` uses linear interpolation between the
neighbouring order statistics. With the data ordered ascending and `n` values,
the position is `probability * (n - 1)`; when that position falls between two
values, the result interpolates between them by the fractional part.

- `probability` must be in `0.0..1.0`; anything else raises `StatisticsError`
  rather than being clamped.
- `probability` `0.0` is the minimum and `1.0` is the maximum.
- A single-value List is its own quantile for every valid probability.

```ahd
values: List<Int> := [1, 2, 3, 4]

write(Statistics.quantile(values, 0.0))   // 1.0
write(Statistics.quantile(values, 0.25))  // 1.75
write(Statistics.quantile(values, 0.5))   // 2.5
write(Statistics.quantile(values, 1.0))   // 4.0
```

## Empty and undefined input

`sum` of an empty List is the additive identity — `0` for `Int` and `0.0` for
`Real` — because that is the one total which keeps `sum(a) + sum(b)` equal to
the sum of the combined values.

Every other statistic is mathematically undefined for an empty List and raises
`StatisticsError`:

```text
mean([])      median([])      min([])       max([])
range([])     variance([])    stdDev([])    mode([])
quantile([], p)
```

`sampleVariance` and `sampleStdDev` additionally require at least two values,
because dividing by `n - 1` is undefined for a single one.

```ahd
attempt {
    write(Statistics.mean(empty))
} except StatisticsError as error {
    write(error.message)
}
```

`StatisticsError` derives directly from `Error`. It is used only for statistics
that are undefined for their input; it is not reused for Data, CSV, or
filesystem failures.

## Finite results

AhdCode's `Real` is finite by the language's existing contract: ordinary
arithmetic reports a domain or range error rather than producing `NaN` or an
infinity. Statistics keeps that contract — a statistic never hands back `NaN` or
an infinity, and reports `StatisticsError` instead if one would arise.

## Input is never modified

Ordering the data for a median or a quantile works on a snapshot, so the
caller's List keeps its order:

```ahd
values: List<Int> := [3, 1, 2]

write(Statistics.median(values))  // 2.0
write(values)                     // [3, 1, 2]
```

## What Statistics is not

The `Statistics` module is descriptive statistics only. There is no inferential testing, no
regression, no distribution, no random sampling, and no plotting. There is no
`frequency` function either: a frequency table would be `Pair<K, Int>`, and a
Pair key must be `String`, `Int`, or `Bool`, so `List<Real>` input has no
expressible result. `mode` covers the common need, and
[`Table.valueCounts`](DATA.md) counts String cells.
