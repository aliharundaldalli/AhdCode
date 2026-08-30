# Math standard module

[English] · [Türkçe](MATH_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [List API](LIST_API.md)

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
seed random randomInt
```

`round` returns Real and rounds exact halves away from zero. Its optional
digits argument is restricted to `0..15`. `floor` and `ceil` return Int.
Trigonometric functions use radians. `log` is natural logarithm; `log10` is
base ten. `^` is exponentiation; there is no `Math.pow`. `abs`, `sum`, `min`,
and `max` are [Fundamentals](FUNDAMENTALS.md), not Math members.

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
