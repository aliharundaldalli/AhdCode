# Math standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [Fundamentals](FUNDAMENTALS.md)

**Mathematics path:** **Math** → [Numeric](NUMERIC.md) → [Statistics](STATISTICS.md) → [Plot](PLOT.md)

`Math` holds the scalar functions of school and university mathematics:
roots, powers, logarithms, trigonometry, rounding, greatest common divisors,
and random numbers. Every function takes one or two numbers and returns one
number.

```ahd
bring Math

write(Math.sqrt(25))
write(Math.round(Math.PI, 2))
```

```text
5.0
3.14
```

Math is an explicit module: write `bring Math` once at the top of the file and
call its members as `Math.name(...)`. To use a few names without the prefix,
bring them selectively:

```ahd
from Math bring (PI, sqrt)

write(sqrt(PI))
```

```text
1.7724538509055159
```

## Which module do I need?

| You want to… | Use |
|---|---|
| compute with single numbers: `√x`, `sin x`, `ln x`, rounding | **Math** (this page) |
| take `abs`, `min`, `max`, or `sum` | [Fundamentals](FUNDAMENTALS.md) — always available, no `bring` |
| work with vectors, matrices, linear systems, or complex numbers | [Numeric](NUMERIC.md) |
| summarize data: mean, median, standard deviation, regression | [Statistics](STATISTICS.md) |
| draw a function or a data set | [Plot](PLOT.md) |

## Contents

- [Int and Real in calculations](#int-and-real-in-calculations)
- [Constants](#constants)
- [Powers and roots](#powers-and-roots)
- [Exponentials and logarithms](#exponentials-and-logarithms)
- [Trigonometry](#trigonometry)
- [Rounding](#rounding)
- [Absolute value, minimum, maximum](#absolute-value-minimum-maximum)
- [Greatest common divisor and least common multiple](#greatest-common-divisor-and-least-common-multiple)
- [Random numbers](#random-numbers)
- [Domain and overflow errors](#domain-and-overflow-errors)
- [Worked examples](#worked-examples)
- [Quick reference](#quick-reference)

## Int and Real in calculations

AhdCode has two number types. `Int` is a whole number (signed 64-bit) and
`Real` is a decimal number (64-bit floating point). Knowing which one an
expression produces explains most of what Math does.

- An `Int` is accepted wherever a `Real` is expected, so `Math.sqrt(16)` works
  and returns `4.0`. The opposite is never automatic: use `int(x)` (which
  truncates toward zero) or one of the rounding functions below.
- `/` always produces a `Real`, even for two Ints: `7 / 2` is `3.5` and
  `6 / 3` is `2.0`.
- `%` is the Int remainder. Its sign follows the left operand: `7 % 3` is `1`
  and `-7 % 3` is `-1`.
- `^` is exponentiation (there is no `Math.pow`). `Int ^ Int` stays an `Int`
  and needs a non-negative exponent; with a `Real` on either side the result
  is `Real`.
- Dividing by zero raises `DivisionByZeroError`, for Int and Real alike.
- A result that leaves the `Int` range raises `OverflowError` instead of
  wrapping around. A constant expression that overflows is already a compile
  error.
- A `Real` result is always finite: AhdCode programs never see NaN or
  infinity. Where another language would return one, AhdCode raises an error
  (see [Domain and overflow errors](#domain-and-overflow-errors)).

```ahd
write(7 / 2)
write(type(6 / 3))
write(-7 % 3)
write(2 ^ 10)
write(2.0 ^ 0.5)
```

```text
3.5
Real
-1
1024
1.4142135623730951
```

`Real` is binary floating point, so a decimal fraction such as `0.1` is stored
approximately and the last digit of a result can differ from the exact value:
`Math.sin(Math.PI / 6)` prints `0.49999999999999994`, not `0.5`. Round only
when you present a value; keep full precision while you calculate.

## Constants

```text
Math.PI  -> Real    π = 3.141592653589793
Math.E   -> Real    e = 2.718281828459045, the base of the natural logarithm
```

Both are constants: they cannot be assigned.

```ahd
bring Math

radius := 3.0
write(Math.round(Math.PI * radius ^ 2, 2))
write(Math.round(2 * Math.PI * radius, 2))
```

```text
28.27
18.85
```

## Powers and roots

```text
x ^ n                      power (operator)
Math.sqrt(value) -> Real   square root, value >= 0
Math.cbrt(value) -> Real   cube root, any sign
Math.hypot(x, y) -> Real   √(x² + y²)
```

- `sqrt` rejects negative values with `DomainError`; Math does not return
  complex roots.
- `cbrt` accepts negative values: `Math.cbrt(-27.0)` is `-3.0`. The operator
  form `(-27.0) ^ (1.0 / 3.0)` raises `DomainError` instead, because a
  negative base with a fractional exponent has no Real value in general.
- `hypot(x, y)` is the length of the vector `(x, y)`, computed without the
  overflow that `Math.sqrt(x ^ 2 + y ^ 2)` risks for very large inputs.
- Any other root is a power: the fourth root of 81 is `81.0 ^ 0.25`.

```ahd
bring Math

write(Math.sqrt(2))
write(Math.cbrt(-8.0))
write(Math.hypot(3, 4))
write(81.0 ^ 0.25)
```

```text
1.4142135623730951
-2.0
5.0
3.0000000000000004
```

`81.0 ^ 0.25` is mathematically `3`; the final `4` in the last place is the
floating-point rounding described in
[Int and Real in calculations](#int-and-real-in-calculations). Use
`Math.round(81.0 ^ 0.25, 10)` when you show such a value.

## Exponentials and logarithms

```text
Math.exp(value) -> Real     eˣ
Math.log(value) -> Real     natural logarithm ln x, value > 0
Math.log10(value) -> Real   base-10 logarithm, value > 0
Math.log2(value) -> Real    base-2 logarithm, value > 0
```

`log` is the **natural** logarithm (base e); it is not base 10. Every
logarithm needs a value greater than zero and raises `DomainError` for zero or
a negative value. `exp` raises `OverflowError` once `eˣ` is too large for a
finite `Real` (from about `x = 710`).

For any other base, divide two logarithms: log_b(x) = ln x / ln b. The
quotient is a floating-point value, so `log₃ 81` prints as
`4.000000000000001`; round it before comparing it with a whole number.

```ahd
bring Math

write(Math.exp(1))
write(Math.log(Math.E))
write(Math.log10(1000))
write(Math.log2(1024.0))
write(Math.log(81.0) / Math.log(3.0))
```

```text
2.718281828459045
1.0
3.0
10.0
4.000000000000001
```

## Trigonometry

### Angles are in radians

Every trigonometric function works in **radians**. An angle is a plain `Real`;
there is no separate angle type. Convert degrees with `radians` and back with
`degrees`:

```text
Math.radians(degrees) -> Real   degrees * π / 180
Math.degrees(radians) -> Real   radians * 180 / π
```

Neither function normalizes: `Math.degrees(Math.radians(720.0))` is `720.0`,
not `0.0`.

### Sine, cosine, tangent

```text
Math.sin(value) -> Real
Math.cos(value) -> Real
Math.tan(value) -> Real
```

```ahd
bring Math

for degrees in between(0, 91, 30) {
    angle: Local := Math.radians(degrees)
    write("{degrees}°  sin={Math.round(Math.sin(angle), 4)}  cos={Math.round(Math.cos(angle), 4)}")
}
```

```text
0°  sin=0.0  cos=1.0
30°  sin=0.5  cos=0.866
60°  sin=0.866  cos=0.5
90°  sin=1.0  cos=0.0
```

`Math.PI / 2` is only the closest `Real` to π/2, so `Math.tan(Math.PI / 2)` is
a very large finite number rather than an error.

### Inverse functions

```text
Math.asin(value) -> Real      value in -1..1, result in [-π/2, π/2]
Math.acos(value) -> Real      value in -1..1, result in [0, π]
Math.atan(value) -> Real      result in (-π/2, π/2)
Math.atan2(y, x) -> Real      angle of the point (x, y), result in [-π, π]
```

`atan2` answers "which direction does the point `(x, y)` lie in?" and, unlike
`atan(y / x)`, knows the quadrant and handles `x = 0`. Note the argument order:
**`y` first**. `Math.atan2(0, 0)` is `0.0`.

```ahd
bring Math

write(Math.degrees(Math.atan2(1.0, 1.0)))
write(Math.degrees(Math.atan2(1.0, -1.0)))
write(Math.degrees(Math.asin(0.5)))
```

```text
45.0
135.0
30.000000000000004
```

### Hyperbolic functions

```text
Math.sinh(value) -> Real
Math.cosh(value) -> Real
Math.tanh(value) -> Real
```

A result too large for a finite `Real`, such as `Math.cosh(1000.0)`, raises
`OverflowError`.

## Rounding

```text
Math.round(value) -> Real               nearest whole value
Math.round(value, digits: Int) -> Real  nearest value with `digits` decimals, digits in 0..15
Math.floor(value) -> Int                largest Int <= value
Math.ceil(value) -> Int                 smallest Int >= value
int(value) -> Int                       drops the fraction (toward zero), no bring needed
```

The four differ on negative numbers and on exact halves:

| value | `Math.round` | `Math.floor` | `Math.ceil` | `int` |
|---|---|---|---|---|
| `2.5` | `3.0` | `2` | `3` | `2` |
| `-2.5` | `-3.0` | `-3` | `-2` | `-2` |
| `-2.7` | `-3.0` | `-3` | `-2` | `-2` |

- `round` returns a **Real** and rounds exact halves away from zero. Use it to
  present a value: `Math.round(3.14159, 2)` is `3.14`. A `digits` value
  outside `0..15` raises `DomainError`.
- `floor` and `ceil` return an **Int**, so they turn a Real into a whole
  number you can count or index with. A value beyond the `Int` range raises
  `OverflowError`.
- Since `/` always produces a `Real`, whole-number division is
  `Math.floor(a / b)` for non-negative values.

```ahd
bring Math

write(Math.round(3.14159, 2))
write(Math.floor(-2.5))
write(Math.ceil(-2.5))
write(int(-2.7))
write(Math.floor(17 / 5))
```

```text
3.14
-3
-2
-2
3
```

## Absolute value, minimum, maximum

`abs`, `min`, `max`, and `sum` are not Math members. They are
[Fundamentals](FUNDAMENTALS.md), available in every file without `bring`, and
they keep the element type: `abs(-3)` is the Int `3`, `abs(-3.5)` is `3.5`.
`min` and `max` take a List; an empty List raises `DomainError`, while `sum` of
an empty List is `0` (or `0.0`).

```ahd
write(abs(-3))
write(min([3, 1, 2]))
write(max([1.5, 2.5]))
write(sum([1, 2, 3, 4]))
```

```text
3
1
2.5
10
```

For the mean, median, spread, and more of a data set, continue with
[Statistics](STATISTICS.md).

## Greatest common divisor and least common multiple

```text
Math.gcd(first: Int, second: Int) -> Int
Math.lcm(first: Int, second: Int) -> Int
```

Both work on Ints and never return a negative number. `gcd(0, 0)` is `0`, and
`lcm` is `0` when either argument is `0`. A result that does not fit an `Int`
raises `OverflowError` instead of wrapping around; this includes
`Math.gcd(-9223372036854775808, 0)`, whose magnitude is one more than the
largest Int.

A common use is reducing a fraction:

```ahd
bring Math

numerator := 42
denominator := 56
divisor := Math.gcd(numerator, denominator)
write("{Math.floor(numerator / divisor)}/{Math.floor(denominator / divisor)}")
write(Math.lcm(first: 4, second: 6))
```

```text
3/4
12
```

## Random numbers

```text
Math.random() -> Real                  0.0 <= value < 1.0
Math.randomInt(min: Int, max: Int) -> Int  min <= value <= max (both bounds included)
Math.seed(value: Int) -> Nothing       restart the sequence
```

A new program starts from operating-system entropy, so two runs give
different numbers. `Math.seed(n)` makes the sequence reproducible, which is
what tests and simulations need: the same seed always produces the same
sequence.

```ahd
bring Math

Math.seed(42)
first := [Math.randomInt(1, 6), Math.randomInt(1, 6), Math.randomInt(1, 6)]
Math.seed(42)
second := [Math.randomInt(1, 6), Math.randomInt(1, 6), Math.randomInt(1, 6)]
write(first == second)
```

```text
true
```

- `randomInt` raises `DomainError` when `min > max`; equal bounds return that
  value.
- The generator (SplitMix64) is pseudo-random and **not suitable for
  cryptography**. Use [Security](SECURITY.md) for tokens and secrets.
- `Math.random`, `Math.randomInt`, and `List.shuffle` share one program-wide
  state. Equal `randomInt` bounds and shuffling an empty or one-element List
  consume no state.

## Domain and overflow errors

Math never returns NaN or infinity. An input outside a function's domain
raises `DomainError`; a result too large for its type raises `OverflowError`.
Both can be caught:

```ahd
bring Math

attempt {
    write(Math.sqrt(-1.0))
} except DomainError as error {
    write(error.message)
}
attempt {
    write(Math.exp(1000.0))
} except OverflowError as error {
    write(error.message)
}
```

```text
Math.sqrt requires a non-negative value
Math.exp result exceeds finite Real range
```

| Call | Requirement | Otherwise |
|---|---|---|
| `sqrt(x)` | `x >= 0` | `DomainError` |
| `log(x)`, `log10(x)`, `log2(x)` | `x > 0` | `DomainError` |
| `asin(x)`, `acos(x)` | `-1 <= x <= 1` | `DomainError` |
| `round(x, digits)` | `digits` in `0..15` | `DomainError` |
| `randomInt(min, max)` | `min <= max` | `DomainError` |
| `exp`, `sinh`, `cosh`, `Real ^ Real` | finite result | `OverflowError` |
| `floor`, `ceil` | result fits `Int` | `OverflowError` |
| `gcd`, `lcm`, `Int ^ Int` | result fits `Int` | `OverflowError` |
| `Int ^ Int` | exponent `>= 0` | `DomainError` |
| `a / b`, `a % b` | `b != 0` | `DivisionByZeroError` |

## Worked examples

### Roots of a quadratic equation

```ahd
bring Math

solveQuadratic: Function := (a: Real, b: Real, c: Real) -> List<Real> {
    discriminant: Local := b ^ 2 - 4 * a * c
    if discriminant < 0 {
        return []
    }
    root: Local := Math.sqrt(discriminant)
    return [(-b + root) / (2 * a), (-b - root) / (2 * a)]
}

write(solveQuadratic(1, -3, 2))
write(solveQuadratic(1, 2, 5))
```

```text
[2.0, 1.0]
[]
```

The empty List for `x² + 2x + 5` says there is no Real root. Checking the
discriminant first is what keeps `Math.sqrt` inside its domain.

### Distance and direction between two points

```ahd
bring Math

dx := 4.0 - 1.0
dy := 6.0 - 2.0
write(Math.hypot(dx, dy))
write(Math.round(Math.degrees(Math.atan2(dy, dx)), 1))
```

```text
5.0
53.1
```

### Compound interest

A balance of 1000 at 5% yearly interest for 10 years is 1000 · 1.05¹⁰; the
number of years needed to double it is ln 2 / ln 1.05.

```ahd
bring Math

balance := 1000.0 * 1.05 ^ 10
write(Math.round(balance, 2))
years := Math.log(2.0) / Math.log(1.05)
write(Math.ceil(years))
```

```text
1628.89
15
```

To tabulate or draw a function over many points, build the x values with
[`Numeric.linspace`](NUMERIC.md#creating-vectors) and pass them to
[Plot](PLOT.md#quick-start-plotting-a-function).

## Quick reference

| Member | Signature | Meaning |
|---|---|---|
| `PI`, `E` | `Real` | π and e |
| `sqrt` | `(value: Real) -> Real` | √x, `x >= 0` |
| `cbrt` | `(value: Real) -> Real` | ∛x, any sign |
| `hypot` | `(x: Real, y: Real) -> Real` | √(x² + y²) |
| `exp` | `(value: Real) -> Real` | eˣ |
| `log` | `(value: Real) -> Real` | ln x, `x > 0` |
| `log10` | `(value: Real) -> Real` | log₁₀ x, `x > 0` |
| `log2` | `(value: Real) -> Real` | log₂ x, `x > 0` |
| `sin`, `cos`, `tan` | `(value: Real) -> Real` | radians |
| `asin`, `acos` | `(value: Real) -> Real` | `-1 <= x <= 1` |
| `atan` | `(value: Real) -> Real` | inverse tangent |
| `atan2` | `(y: Real, x: Real) -> Real` | direction of `(x, y)` |
| `sinh`, `cosh`, `tanh` | `(value: Real) -> Real` | hyperbolic functions |
| `radians` | `(degrees: Real) -> Real` | degrees → radians |
| `degrees` | `(radians: Real) -> Real` | radians → degrees |
| `round` | `(value: Real) -> Real`, `(value: Real, digits: Int) -> Real` | halves away from zero |
| `floor`, `ceil` | `(value: Real) -> Int` | round down / up to an Int |
| `gcd`, `lcm` | `(first: Int, second: Int) -> Int` | never negative |
| `random` | `() -> Real` | `0.0 <= r < 1.0` |
| `randomInt` | `(min: Int, max: Int) -> Int` | inclusive bounds |
| `seed` | `(value: Int) -> Nothing` | reproducible sequence |

An `Int` argument is accepted wherever the table says `Real`.

**Next:** [Numeric](NUMERIC.md) — vectors, matrices, linear systems, and
complex numbers.
