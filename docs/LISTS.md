# Lists standard module

[English] · [Türkçe](LISTS_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [KeyValue](KEYVALUE.md) · [List API](LIST_API.md)

`Lists` is the compiler-registered `builtin:Lists` module. It is explicit and
a sibling `Lists.ahd` cannot shadow it:

```ahd
bring Lists
from Lists bring ListsError
```

`Lists` exists for the pure structural transformations the core `List` type
does not naturally provide as members. It deliberately duplicates nothing:
`add`, `eject`, `sort`, `reverse`, `shuffle`, `count`, `index`, `map`,
`filter`, slicing, and `List + List` stay exactly where they are, on `List`
itself. See [List API](LIST_API.md) for those.

## Surface

```text
Lists.chunk(values: List<T>, size: Int)          -> List<List<T>>
Lists.flatten(values: List<List<T>>)             -> List<T>
Lists.transpose(rows: List<List<T>>)             -> List<List<T>>
Lists.unique(values: List<T>)                    -> List<T>
Lists.valueCounts(values: List<K>)               -> Pair<K, Int>
Lists.groupBy(values: List<T>, key: Function(T) -> K) -> Pair<K, List<T>>
```

## The operations are type-directed

`T` and `K` above are not AhdCode syntax. There is no user-facing generic
`Function` syntax, and you never write `Lists.chunk<Int>(...)`. These
operations are supplied by the compiler, which computes each call's result
type from the argument types you actually wrote:

```ahd
numbers: List<Int> := [1, 2, 3]
words: List<String> := ["a", "b"]

intChunks: List<List<Int>> := Lists.chunk(numbers, 2)
stringChunks: List<List<String>> := Lists.chunk(words, 1)
```

Both call sites have one exact static type. Nothing is erased to an `Object`,
`Any`, or `dynamic` representation, no conversion happens on the way in, and
`List<List<Int>>` never widens to `List<List<Real>>`.

Both spellings work:

```ahd
bring Lists
from Lists bring chunk

parts := chunk([1, 2, 3], 2)
```

### The one boundary

Because the result type is computed from the arguments at each call site,
these operations have no single concrete `Function` type, and so no
unspecialized `Function` value:

```ahd
stored := Lists.chunk        // rejected at compile time
```

This is a compile-time diagnostic, not a runtime surprise. Call the operation
directly, or wrap the exact shape you want in your own `Function`:

```ahd
pageInts: Function := (values: List<Int>) -> List<List<Int>> {
    return Lists.chunk(values, 10)
}
```

## chunk

```ahd
write(Lists.chunk([1, 2, 3, 4, 5], 2))
```

```text
[[1, 2], [3, 4], [5]]
```

Source order is preserved. The final chunk is short rather than padded — no
filler value is invented. An empty source produces a correctly typed empty
`List`, and a `size` larger than the source produces one chunk holding
everything.

`size` must be greater than zero; `0` or a negative size raises `ListsError`
rather than guessing what was meant.

## flatten

```ahd
empty: List<Int> := []
write(Lists.flatten([[1, 2], [3], empty, [4, 5]]))
```

```text
[1, 2, 3, 4, 5]
```

This is exactly one level of flattening. `List<List<List<Int>>>` flattens to
`List<List<Int>>`, not to `List<Int>`: there is no recursive flatten, because
the depth of a nesting is a decision the caller should make explicitly.

The inner Lists must be non-null (`List<List<T>>`, not `List<List<T>?>`).
A nullable inner List has no defined contribution — skipping it would silently
drop data — so it is a compile-time type error.

## transpose

```ahd
write(Lists.transpose([[1, 2, 3], [4, 5, 6]]))
```

```text
[[1, 4], [2, 5], [3, 6]]
```

`transpose` requires rectangular input: **every row must have exactly the same
length**. Ragged input raises `ListsError`:

```text
transpose requires rectangular rows: row 1 has 2 element(s); expected 3
```

The row number is the 0-based index in the source `List`. Nothing is padded,
nothing is truncated, and nothing is guessed. Silent rectangularity repair is
how table data quietly disappears, so it is explicitly not offered.

Edge cases:

```text
[]                -> []
[[], []]          -> []
[[1, 2, 3]]       -> [[1], [2], [3]]
```

Transposing twice restores the original shape.

## unique

```ahd
write(Lists.unique([3, 1, 3, 2, 1]))
```

```text
[3, 1, 2]
```

The **first** occurrence of each distinct value is kept, in first-occurrence
order. Distinctness uses ordinary AhdCode `==` — the same rule `in`,
`List.count`, and `List.index` use. Values are never stringified, hashed by
address, or compared by any invented rule, so `List` and `Pair` elements
compare deeply and `Class` instances compare the way `==` already defines for
them.

With a nullable element type, `null` is one ordinary distinct value:

```ahd
values: List<String?> := ["a", null, "a", null, "b"]
write(Lists.unique(values))
```

```text
["a", null, "b"]
```

A `List<Function>` is rejected at compile time: `==` defines no comparison for
`Function` values, so there is nothing for `unique` to use.

## valueCounts

```ahd
write(Lists.valueCounts([1, 1, 3, 2, 1, 3]))
```

```text
{1: 3, 3: 2, 2: 1}
```

Keys appear in first-occurrence order. The element type must satisfy the
existing `Pair` key rules — `String`, `Int`, or `Bool`, and never nullable:

```ahd
write(Lists.valueCounts(["Math", "Physics", "Math"]))
```

```text
{"Math": 2, "Physics": 1}
```

`List<Real>`, `List<Class>`, `List<List<Int>>`, and `List<String?>` are all
rejected at compile time. Nothing is converted to `String` to make it fit;
this release does not widen what a `Pair` key may be.

## groupBy

```ahd
write(Lists.groupBy(["Ali", "Ayse", "Bora", "Ahmet"], lambda (name: String) -> name[0]))
```

```text
{"A": ["Ali", "Ayse", "Ahmet"], "B": ["Bora"]}
```

Keys appear in first-key-occurrence order, and the elements inside each group
keep their source order.

The key `Function` must take exactly the `List`'s element type, **including
its nullability**, and must return a non-null `Pair` key type. It runs exactly
once per element, left to right, over a shallow snapshot of the source — so a
callback that structurally mutates the source `List` cannot change what is
iterated, matching `List.map` and `List.filter`.

If the callback raises, the error propagates unchanged, with its own type; no
partial result is returned.

```ahd
values: List<String?> := ["a", null]
write(Lists.groupBy(values, lambda (value: String?) -> str(value)))
```

## Ordering is a public contract

These orders are guaranteed, not implementation accidents:

| Operation | Order |
| --- | --- |
| `chunk` | source order, in consecutive runs |
| `flatten` | outer order, then inner order |
| `transpose` | column index, then row index |
| `unique` | first occurrence |
| `valueCounts` | first occurrence of each key |
| `groupBy` keys | first occurrence of each key |
| `groupBy` members | source order |

## Nullability

Structural nullability is preserved exactly, never erased and never dropped:

```text
Lists.chunk(List<String?>, n)   -> List<List<String?>>
Lists.flatten(List<List<T?>>)   -> List<T?>
Lists.transpose(List<List<T?>>) -> List<List<T?>>
Lists.unique(List<String?>)     -> List<String?>
```

`Pair` keys are never null, so `Lists.valueCounts(List<String?>)` is invalid,
and `Lists.groupBy`'s key `Function` must return a non-null key type.

## Shallow structural semantics

Every operation is pure with respect to collection *structure*: it never
mutates the `List` it is given, so a `Constant List` may be passed safely, and
every returned `List` — outer and inner — is a new, structurally independent
collection.

The transformation is **shallow**. Referenced elements are carried over by
reference, not deep-copied:

```ahd
boxes: List<Box> := [Box(label: "one"), Box(label: "two")]
parts: List<List<Box>> := Lists.chunk(boxes, 1)
parts[0][0].label = "changed"
write(boxes[0].label)          // changed
```

Chunking `List<Student>` produces new Lists holding the *same* `Student`
objects. This is structural immutability, not deep immutability.

## Errors

`ListsError` derives directly from `Error` and covers the module's own
structural failures:

- a `chunk` size of zero or less;
- ragged `transpose` input.

Type-invalid calls stay compile-time type errors and never reach `ListsError`.
A callback that raises keeps its own error type — it is never wrapped.

```ahd
attempt {
    write(Lists.transpose([[1, 2, 3], [4, 5]]))
}
except ListsError as error {
    write(error.message)
}
```

## Non-goals

`Lists` is not a general functional-collections library. It deliberately does
not add:

- `zip` / `unzip` — AhdCode has no `Tuple`, and `Pair` keys are restricted, so
  Python's `zip` semantics have no honest representation yet;
- `sum` / `mean` / `min` / `max` — numeric and statistical work belongs to
  `Math`, `Statistics`, and `Numeric`;
- a recursive flatten, `invert`, or other speculative conveniences;
- iterators, lazy collections, streams, or parallel collections.

## See also

- [KeyValue](KEYVALUE.md) — the same idea for `Pair`
- [List API](LIST_API.md) — the core `List` members
- [Collections](COLLECTIONS.md) — `List` and `Pair` fundamentals
- [Data](DATA.md) — tabular data built on these shapes
- `examples/v0.1/47_lists.ahd`
