# Fundamentals

[Back to README](../README.md) · [String API](STRING_API.md) · [List API](LIST_API.md)

These names are predeclared in every module and require no `bring`:

```text
write take str int real len clear between abs sum min max
```

| Function | Behavior |
|---|---|
| `write(value)` | writes one value followed by a newline |
| `take()` / `take(prompt)` | reads one line as String |
| `str(value)` | canonical deterministic text |
| `int(Real)` | truncates toward zero |
| `int(String)` | strict signed ASCII-decimal parse |
| `real(Int)` | explicit safe widening |
| `real(String)` | strict decimal/fraction/exponent parse |
| `len(value)` | String characters, List elements, or Pair entries |
| `clear(collection)` | empties List or Pair in place |
| `between(...)` | lazy stop-exclusive Int iteration |
| `abs(number)` | numeric magnitude with exact result type |
| `sum(list)` | numeric reduction; empty List gives `0` or `0.0` |
| `min(list)` / `max(list)` | numeric extrema; empty List raises `DomainError` |

Conversions trim surrounding Unicode whitespace in String input. `int(String)`
does not accept fractions, exponents, underscores, or base prefixes.
`real(String)` accepts decimal integer, fraction, and exponent forms but not
NaN or infinity. Invalid text raises `DomainError`; out-of-range text raises
`OverflowError`.

`str` preserves List order and Pair insertion order, quotes nested Strings,
prints a Class instance as `<ClassName>`, and never exposes attributes.

`clear` mutates the existing collection, so aliases observe it and Constant
collections reject it. The numeric reductions are pure reads and accept a
non-null Constant List.
