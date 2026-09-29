# Statistics standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [Data](DATA.md)

**Mathematics path:** [Math](MATH.md) → [Numeric](NUMERIC.md) → **Statistics** → [Plot](PLOT.md)

`Statistics` describes a data set: where its centre is (mean, median, mode),
how spread out it is (range, variance, standard deviation, quantiles), and how
two data sets move together (covariance, correlation, a fitted line). Its
input is a typed List of numbers.

```ahd
bring Statistics

scores: List<Int> := [72, 85, 90, 64, 85, 78, 95, 58]
write(Statistics.mean(scores))
write(Statistics.median(scores))
write(Statistics.mode(scores))
```

```text
78.375
81.5
85
```

Statistics is an explicit module. Bring its error Class too when you want to
catch it by name:

```ahd
bring Statistics
from Statistics bring StatisticsError
```

The canonical identity is `builtin:Statistics`; a sibling `Statistics.ahd`
cannot shadow it. Every argument is `NonNull`, and no function modifies the
List it receives.

## Contents

- [At a glance](#at-a-glance)
- [Int and Real input, typed results](#int-and-real-input-typed-results)
- [Centre: mean, median, mode](#centre-mean-median-mode)
- [Spread: range, variance, standard deviation](#spread-range-variance-standard-deviation)
- [Quantiles](#quantiles)
- [Two Lists: covariance, correlation, and a fitted line](#two-lists-covariance-correlation-and-a-fitted-line)
- [Empty and undefined input](#empty-and-undefined-input)
- [Data from text, CSV, and tables](#data-from-text-csv-and-tables)
- [Worked example: an exam report](#worked-example-an-exam-report)
- [What Statistics is not](#what-statistics-is-not)

## At a glance

| Function | Result | Meaning |
|---|---|---|
| `sum(values)` | element type | total; `0` / `0.0` for an empty List |
| `min(values)`, `max(values)` | element type | smallest / largest value |
| `range(values)` | element type | `max - min` |
| `mode(values)` | element type | most frequent value; ties go to the first one seen |
| `mean(values)` | `Real` | arithmetic mean |
| `median(values)` | `Real` | middle of the sorted data |
| `variance(values)` | `Real` | population variance, divides by `n` |
| `sampleVariance(values)` | `Real` | sample variance, divides by `n - 1` |
| `stdDev(values)` | `Real` | √`variance` |
| `sampleStdDev(values)` | `Real` | √`sampleVariance` |
| `quantile(values, probability)` | `Real` | value below which that share of the data lies |
| `covariance(x, y)` | `Real` | population covariance |
| `sampleCovariance(x, y)` | `Real` | sample covariance |
| `correlation(x, y)` | `Real` | Pearson's r, in `-1.0..1.0` |
| `linearRegression(x, y)` | `Pair<String, Real>` | least-squares line: `slope`, `intercept` |

`values`, `x`, and `y` are each a `List<Int>` or a `List<Real>`.

## Int and Real input, typed results

Every function is published as an explicit `Int`/`Real` overload pair, so the
static type of a result is always known:

- A statistic whose answer is one of the input's own values — `min`, `max`,
  `mode`, and the difference `range` — **keeps the element type**: an Int List
  gives an Int.
- A statistic that averages or measures spread — `mean`, `median`, the
  variances, the standard deviations, `quantile`, and the two-List functions —
  is always **`Real`**, because the average of whole numbers is generally not
  whole.
- An `Int` `sum` or `range` that leaves the signed 64-bit range raises
  `OverflowError` instead of wrapping around.

```ahd
bring Statistics

write(type(Statistics.max([3, 8, 5])))
write(type(Statistics.mean([3, 8, 5])))
write(Statistics.median([1, 2, 3]))
```

```text
Int
Real
2.0
```

A numeric statistic never reads text. Passing `List<String>` is a compile
error, even when the Strings hold digits:

```ahd
bring Statistics

write(Statistics.mean(["10", "20", "30"]))
```

See [Data from text, CSV, and tables](#data-from-text-csv-and-tables) for the
explicit conversion.

## Centre: mean, median, mode

- `mean` is the arithmetic mean: the sum divided by the count.
- `median` is the middle value of the sorted data, averaging the two middle
  values when the count is even: `median([1, 2, 3, 4])` is `2.5`. Sorting
  works on a copy; your List keeps its order.
- `mode` is the most frequent value. When several values tie for the highest
  frequency, the one that occurs **first in the input** wins, so the result
  never depends on hidden ordering.

```ahd
bring Statistics

values: List<Int> := [3, 1, 2]
write(Statistics.median(values))
write(values)
write(Statistics.mode([2, 3, 3, 2]))
write(Statistics.mode([3, 2, 2, 3]))
```

```text
2.0
[3, 1, 2]
2
3
```

The mean reacts to extreme values and the median does not; comparing the two
is a quick check for outliers.

## Spread: range, variance, standard deviation

`range` is `max - min`. `variance` and `stdDev` are the **population** forms,
dividing by `n`; `sampleVariance` and `sampleStdDev` are the **sample** forms,
dividing by `n - 1` (Bessel's correction). Use the population form when the
List is the entire group you care about, and the sample form when it is a
sample used to estimate a larger population. Both names are published so the
definition is never implicit.

```ahd
bring Statistics

values: List<Int> := [3, 1, 4, 1, 5]
write(Statistics.range(values))
write(Statistics.variance(values))
write(Statistics.sampleVariance(values))
write(Statistics.stdDev([2, 4, 4, 4, 5, 5, 7, 9]))
```

```text
4
2.56
3.2
2.0
```

## Quantiles

`quantile(values, probability)` returns the value below which the given share
of the data lies: `0.5` is the median, `0.25` and `0.75` are the first and
third quartiles. It uses linear interpolation: with the data sorted ascending
and `n` values, the position is `probability * (n - 1)`, and a position between
two values interpolates between them by its fractional part.

- `probability` must be in `0.0..1.0`; anything else raises `StatisticsError`
  rather than being clamped.
- `0.0` gives the minimum and `1.0` the maximum.
- A single-value List is its own quantile for every valid probability.

```ahd
bring Statistics

values: List<Int> := [1, 2, 3, 4]
write(Statistics.quantile(values, 0.0))
write(Statistics.quantile(values, 0.25))
write(Statistics.quantile(values, 0.5))
write(Statistics.quantile(values, 1.0))
```

```text
1.0
1.75
2.5
4.0
```

The interquartile range, a spread measure that ignores outliers, is
`quantile(values, 0.75) - quantile(values, 0.25)`.

## Two Lists: covariance, correlation, and a fitted line

```text
covariance(first, second)       -> Real
sampleCovariance(first, second) -> Real
correlation(first, second)      -> Real
linearRegression(x, y)          -> Pair<String, Real>
```

Each argument is a `List<Int>` or a `List<Real>`, in any combination. The two
Lists must have the same length; the values at the same position form one
pair `(xᵢ, yᵢ)`.

- `covariance` is the **population** covariance, dividing by `n`; it needs at
  least one pair. `sampleCovariance` divides by `n - 1` and needs at least two.
- `correlation` is Pearson's correlation coefficient `r`: `1.0` is a perfect
  rising line, `-1.0` a perfect falling one, and `0.0` no linear relation. It
  needs at least two pairs and raises `StatisticsError` when either List has
  zero variance (all values equal), because `r` is then undefined. A rounding
  overshoot of the last bit is clamped to the boundary.
- `linearRegression(x, y)` fits `y = slope · x + intercept` by ordinary least
  squares and returns the Pair `{"slope": …, "intercept": …}` in that key
  order. It needs at least two pairs and x values that are not all equal. When
  every y is the same value, the slope is `0.0` and the intercept is exactly
  that value.

Every function centres the data on its mean before multiplying (a two-pass
computation), so large offsets such as years or timestamps do not wash out
the result.

```ahd
bring Math
bring Statistics

hours: List<Int> := [1, 2, 3, 4, 5]
scores: List<Real> := [52.0, 57.5, 61.0, 68.5, 71.0]
write(Statistics.covariance(hours, scores))
write(Math.round(Statistics.correlation(hours, scores), 3))
fit := Statistics.linearRegression(x: hours, y: scores)
write(fit)
predicted := fit["slope"] * 6 + fit["intercept"]
write("after 6 hours: {Math.round(predicted, 1)}")
```

```text
9.8
0.991
{"slope": 4.9, "intercept": 47.3}
after 6 hours: 76.7
```

To draw the data and the fitted line, see
[Plot](PLOT.md#multiple-series).

## Empty and undefined input

`sum` of an empty List is the additive identity — `0` for `Int` and `0.0` for
`Real` — because that keeps `sum(a) + sum(b)` equal to the sum of the
combined values.

Every other statistic is mathematically undefined for an empty List and raises
`StatisticsError`: `mean`, `median`, `min`, `max`, `range`, `mode`,
`variance`, `stdDev`, and `quantile`. `sampleVariance` and `sampleStdDev`
additionally need at least two values, because dividing by `n - 1` is
undefined for one. Lists of different lengths, and the two-List cases above,
raise `StatisticsError` too.

```ahd
bring Statistics
from Statistics bring StatisticsError

empty: List<Real> := []
write(Statistics.sum(empty))
attempt {
    write(Statistics.mean(empty))
} except StatisticsError as error {
    write(error.message)
}
attempt {
    write(Statistics.sampleVariance([5]))
} except StatisticsError as error {
    write(error.message)
}
```

```text
0.0
mean is undefined for an empty List
sampleVariance requires at least two values
```

`StatisticsError` derives directly from `Error`. It is used only for
statistics that are undefined for their input; it is not reused for Data, CSV,
or filesystem failures.

AhdCode's `Real` is always finite: ordinary arithmetic reports a domain or
range error rather than producing `NaN` or an infinity, and Statistics keeps
that contract — a statistic never returns `NaN` or an infinity, and reports
`StatisticsError` instead if one would arise.

## Data from text, CSV, and tables

Statistics does **not** depend on [Data](DATA.md) or [CSV](CSV.md). A `Table`
cell and a CSV field are `String`s, so convert them explicitly with `int` or
`real` before asking for a statistic. This is what keeps both modules strict
instead of introducing a dynamic numeric value.

```ahd
bring Statistics

raw: List<String> := ["72", "85.5", "90"]
values: List<Real> := raw.map(lambda (value: String) -> real(value))
write(Statistics.mean(values))
```

```text
82.5
```

With a Data `Table`, the same conversion applies to a column:

```ahd
scores: List<Real> := students.column("score").map(
    lambda (value: String) -> real(value)
)
average: Real := Statistics.mean(scores)
```

`real` raises `DomainError` for text that is not a number, so bad input is
reported where it is read rather than inside a statistic. A Numeric `Vector`
is passed as `vector.values()`.

## Worked example: an exam report

```ahd
bring Math
bring Statistics

scores: List<Int> := [72, 85, 90, 64, 85, 78, 95, 58]

write("students: {len(scores)}")
write("mean:     {Statistics.mean(scores)}")
write("median:   {Statistics.median(scores)}")
write("range:    {Statistics.min(scores)}..{Statistics.max(scores)}")
write("std dev:  {Math.round(Statistics.sampleStdDev(scores), 2)}")
write("Q1..Q3:   {Statistics.quantile(scores, 0.25)}..{Statistics.quantile(scores, 0.75)}")
passed := scores.filter(lambda (score: Int) -> score >= 70)
write("passed:   {len(passed)}")
```

```text
students: 8
mean:     78.375
median:   81.5
range:    58..95
std dev:  12.88
Q1..Q3:   70.0..86.25
passed:   6
```

The sample standard deviation is used here because one class is usually read
as a sample of all the students who could take the exam. To show the
distribution, continue with
[`Plot.histogram` and `Plot.box`](PLOT.md#chart-types).

## What Statistics is not

The `Statistics` module is descriptive statistics only, plus one simple
least-squares line. There is no inferential testing (no p-values or hypothesis
tests), no multiple, polynomial, or logistic regression, no `rSquared` or
regression object, no probability distributions, no random sampling, no
machine learning, and no plotting. There is no `frequency` function either: a
frequency table would be `Pair<K, Int>`, and a Pair key must be `String`,
`Int`, or `Bool`, so `List<Real>` input has no expressible result. `mode`
covers the common need, and [`Table.valueCounts`](DATA.md) counts String
cells. For random numbers, see [Math](MATH.md#random-numbers).

**Next:** [Plot](PLOT.md) — drawing functions and data.
