# List API

[English] · [Türkçe](LIST_API_TR.md)

[Back to README](../README.md) · [Collections](COLLECTIONS.md) · [Math](MATH.md)

For `List<T>`:

| Operation | Behavior |
|---|---|
| `add(value)` | append in place |
| `eject(index)` | remove index in place; negative index allowed |
| `reverse()` | reverse in place |
| `sort()` | stable ascending natural sort for Int, Real, or String |
| `sort(keyFunction)` | stable ascending sort by Int/Real/String key |
| `shuffle()` | in-place unbiased Fisher–Yates permutation |
| `count(value)` | deep-equality match count |
| `index(value)` | first deep-equality match |
| `map(function)` | new List of transformed snapshot elements |
| `filter(function)` | new List of kept snapshot elements |

Mutating operations preserve object identity, so aliases observe them.
`map` and `filter` never mutate the receiver. v0.1 has no lambda syntax; pass a
named Function.

```ahd
double: Function := (
    value: Int
) -> Int {
    return value * 2
}

values: List<Int> := [3, 1, 2]
mapped: List<Int> := values.map(double)
values.sort()

write(values)
write(mapped)
```

`index` raises `DomainError` when no value matches. Natural sort rejects
unsupported element types. Constant or deep-frozen Lists reject all mutation.

`shuffle` uses the same program-wide pseudo-random state as `Math.random` and
`Math.randomInt`. Explicit seeding makes it reproducible:

```ahd
bring Math
Math.seed(42)
values.shuffle()
```

An empty or singleton shuffle consumes no random state.
