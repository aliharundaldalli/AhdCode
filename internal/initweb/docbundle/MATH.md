# Math standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [List API](LIST_API.md)

Math is explicit:

```ahd
bring Math
write(Math.sqrt(25))
```

Direct and selective imports also work:

```ahd
from Math bring (
    PI
    sqrt
)
```

## Surface

```text
PI E
round floor ceil
sqrt sin cos tan log log10 exp
asin acos atan atan2 sinh cosh tanh
hypot log2 cbrt radians degrees
gcd lcm
seed random randomInt
```

`round` returns Real and rounds exact halves away from zero. Its optional
digits argument is restricted to `0..15`. `floor` and `ceil` return Int.
Trigonometric functions use radians. `log` is natural logarithm; `log10` is
base ten. `^` is exponentiation; there is no `Math.pow`. `abs`, `sum`, `min`,
and `max` are [Fundamentals](FUNDAMENTALS.md), not Math members.

## Trigonometry, logarithms, and angles

```text
Math.asin(value: Real) -> Real        Math.sinh(value: Real) -> Real
Math.acos(value: Real) -> Real        Math.cosh(value: Real) -> Real
Math.atan(value: Real) -> Real        Math.tanh(value: Real) -> Real
Math.atan2(y: Real, x: Real) -> Real  Math.hypot(x: Real, y: Real) -> Real
Math.log2(value: Real) -> Real        Math.cbrt(value: Real) -> Real
Math.radians(degrees: Real) -> Real   Math.degrees(radians: Real) -> Real
```

An Int argument widens to Real, as it does for every Real parameter.

- `asin` and `acos` require `-1 <= value <= 1`; `asin` returns `[-π/2, π/2]`
  and `acos` returns `[0, π]`. `atan` returns `(-π/2, π/2)`.
- `atan2(y, x)` is the angle of the point `(x, y)` in `[-π, π]`. Note the
  argument order: `y` first. `atan2(0, 0)` is `0.0`.
- `hypot(x, y)` is `√(x² + y²)` without overflowing for large inputs.
- `log2` requires a value greater than zero. `cbrt` accepts negative values:
  `Math.cbrt(-27.0)` is `-3.0`.
- `radians(d)` is `d * PI / 180` and `degrees(r)` is `r * 180 / PI`. Neither
  normalizes: `Math.degrees(Math.radians(720.0))` is `720.0`. Angles are plain
  Real values; there is no angle type.

Outside a domain the function raises `DomainError`. A result too large for a
finite Real (such as `Math.cosh(1000.0)`) raises `OverflowError`; NaN and
infinity never reach a program.

```ahd
bring Math

write(Math.degrees(Math.atan2(1.0, 1.0)))
write(Math.hypot(3, 4))
write(Math.log2(1024.0))
write(Math.cbrt(-8.0))
attempt {
    write(Math.asin(2.0))
} except DomainError as error {
    write(error.message)
}
```

```text
45.0
5.0
10.0
-2.0
Math.asin requires a value between -1 and 1
```

## Greatest common divisor and least common multiple

```text
Math.gcd(first: Int, second: Int) -> Int
Math.lcm(first: Int, second: Int) -> Int
```

Both results are never negative. `gcd(0, 0)` is `0`, and `lcm` is `0` when
either argument is `0`. `lcm` is computed as `|(first / gcd) * second|` with
checked arithmetic. A result that does not fit Int raises `OverflowError`
instead of wrapping around; this includes `Math.gcd(-9223372036854775808, 0)`,
whose magnitude is one more than the largest Int.

```ahd
bring Math

write(Math.gcd(-12, 18))
write(Math.lcm(first: 4, second: 6))
write(Math.lcm(7, 0))
```

```text
6
12
0
```

## Random state

A fresh native process initializes one shared SplitMix64 state from operating-
system entropy. The public generator is pseudo-random and is not suitable for
cryptographic use. Unseeded startup is not reproducible.

```ahd
bring Math
write(Math.random())
write(Math.randomInt(1, 10))
```

`random()` returns `0.0 <= value < 1.0`. `randomInt(min, max)` uses inclusive
bounds and raises `DomainError` for reversed bounds.

Use explicit seeding for tests and simulations:

```ahd
Math.seed(42)
write(Math.random())
```

Reseeding with the same Int reproduces the same SplitMix64 sequence.
`Math.random`, `Math.randomInt`, and `List.shuffle` consume this same
program-wide state. Equal `randomInt` bounds and empty/singleton shuffle consume
no state.
