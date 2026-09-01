# KeyValue standard module

[English] · [Türkçe](KEYVALUE_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Lists](LISTS.md) · [Collections](COLLECTIONS.md)

`KeyValue` is the compiler-registered `builtin:KeyValue` module. It is
explicit and a sibling `KeyValue.ahd` cannot shadow it:

```ahd
bring KeyValue
from KeyValue bring KeyValueError
```

## Pair stays the core type

`KeyValue` introduces no container of its own. There is no `Dictionary`, no
`Map`, no `Record`, no `Struct`, no `Tuple`, and no `Any`-keyed object bag.
It operates on the language's existing ordered, homogeneous
`Pair<K, V>`, and always hands back a `Pair` or a `List`.

`Pair` is not a Python `dict[Any, Any]`: it is ordered, its keys are `String`,
`Int`, or `Bool` and never null, and its value type is one type for the whole
`Pair`. `KeyValue` is a pure transformation layer over that, nothing more.

## Surface

```text
KeyValue.keys(pair: Pair<K, V>)                  -> List<K>
KeyValue.values(pair: Pair<K, V>)                -> List<V>
KeyValue.combine(keys: List<K>, values: List<V>) -> Pair<K, V>

KeyValue.with(pair: Pair<K, V>, key: K, value: V) -> Pair<K, V>
KeyValue.without(pair: Pair<K, V>, key: K)        -> Pair<K, V>

KeyValue.select(pair: Pair<K, V>, keys: List<K>)  -> Pair<K, V>
KeyValue.drop(pair: Pair<K, V>, keys: List<K>)    -> Pair<K, V>
KeyValue.rename(pair: Pair<K, V>, oldKey: K, newKey: K) -> Pair<K, V>

KeyValue.mapValues(pair: Pair<K, V>, transform: Function(V) -> U) -> Pair<K, U>

KeyValue.merge(left: Pair<K, V>, right: Pair<K, V>)     -> Pair<K, V>
KeyValue.overlay(base: Pair<K, V>, changes: Pair<K, V>) -> Pair<K, V>
```

## The operations are type-directed

`K`, `V`, and `U` are not AhdCode syntax; there is no user-facing generic
`Function` syntax and you never write `KeyValue.keys<String>(...)`. The
compiler computes each call's result type from the argument types actually
written:

```ahd
byName: Pair<String, Int> := {"a": 1}
byNumber: Pair<Int, String> := {1: "a"}

names: List<String> := KeyValue.keys(byName)
numbers: List<Int> := KeyValue.keys(byNumber)
```

Nothing is erased, and `Pair` invariance is unchanged: a `Pair<String, Int>`
never silently becomes a `Pair<String, Real>` by passing through one of these
operations.

Both spellings work:

```ahd
bring KeyValue
from KeyValue bring combine

record := combine(["name"], ["Ali"])
```

### The one boundary

The result type is computed at each call site, so these operations have no
single concrete `Function` type and cannot be stored as an unspecialized
`Function` value:

```ahd
stored := KeyValue.keys        // rejected at compile time
```

Call them directly, or wrap the exact shape you want in your own `Function`.

## keys and values

```ahd
record: Pair<String, String> := {"name": "Ali", "score": "91"}
write(KeyValue.keys(record))
write(KeyValue.values(record))
```

```text
["name", "score"]
["Ali", "91"]
```

Both return values in `Pair` insertion order, and both are fresh `List`
snapshots: mutating the result never reaches the `Pair`.

```ahd
snapshot := KeyValue.keys(record)
snapshot.add("injected")
write(record)                  // unchanged
```

`values` preserves the `Pair`'s value nullability:
`Pair<K, V?>` gives `List<V?>`.

## combine

```ahd
record := KeyValue.combine(
    ["name", "score", "department"]
    ["Ali", "91", "Mathematics"]
)
```

```text
{"name": "Ali", "score": "91", "department": "Mathematics"}
```

Insertion order follows the key `List`. The two Lists must be exactly the same
length — nothing is padded and nothing is truncated — and a duplicate key is
rejected rather than resolved by last-one-wins. Both are `KeyValueError`.

`K` must satisfy the existing `Pair` key rules, so a `List<Real>` or a
`List<String?>` key list is a compile-time error. Value nullability is
preserved: `combine(List<K>, List<V?>)` gives `Pair<K, V?>`. Two empty typed
Lists give a correctly typed empty `Pair`.

## with

`with` is the pure counterpart of `pair[key] = value`.

```ahd
base: Pair<String, String> := {"name": "Ali", "score": "90"}
updated := KeyValue.with(base, "score", "95")

write(base)      // {"name": "Ali", "score": "90"}
write(updated)   // {"name": "Ali", "score": "95"}
```

An existing key keeps its exact position and takes the new value — nothing is
reordered. An absent key is appended at the end. The original `Pair` is never
modified, so a `Constant Pair` may be passed safely.

## without

```ahd
write(KeyValue.without({"a": 1, "b": 2, "c": 3}, "b"))
```

```text
{"a": 1, "c": 3}
```

Every remaining entry keeps its source order. A missing key raises the
language's existing `KeyError`, matching `Pair`'s own `eject` semantics — it
is never silently ignored.

## select

```ahd
write(KeyValue.select({"a": 1, "b": 2, "c": 3}, ["c", "a"]))
```

```text
{"c": 3, "a": 1}
```

`select`'s output order follows the **requested key List**, not the source
`Pair` order. That is what makes it a reordering and projection tool.

An unknown key raises `KeyError`. A key requested twice raises
`KeyValueError`: a repeated request is a mistake in the caller's data, not
something to silently deduplicate. An empty key `List` gives a correctly typed
empty `Pair`.

## drop

```ahd
write(KeyValue.drop({"a": 1, "b": 2, "c": 3}, ["b"]))
```

```text
{"a": 1, "c": 3}
```

`drop` keeps the **source** `Pair`'s order for every retained entry. An
unknown key raises `KeyError` and a repeated request raises `KeyValueError`,
for the same reasons as `select`. An empty key `List` returns a structural
copy of the original.

## rename

```ahd
write(KeyValue.rename({"a": 1, "b": 2, "c": 3}, "b", "middle"))
```

```text
{"a": 1, "middle": 2, "c": 3}
```

The renamed entry keeps exactly its old position and its value. A missing old
key raises `KeyError`. If the new key already belongs to a *different* entry,
that raises `KeyValueError` rather than quietly discarding one of them.
Renaming a key to itself is a harmless no-op that returns a fresh,
structurally equivalent `Pair`.

## mapValues

```ahd
write(KeyValue.mapValues({"a": "10", "b": "20"}, lambda (value: String) -> int(value)))
```

```text
{"a": 10, "b": 20}
```

The key set and its order are unchanged; only the value type changes,
`V -> U`. The callback runs exactly once per value, in `Pair` insertion order,
and its errors propagate unchanged with their own type. A callback returning
`Nothing` is a compile-time error.

The callback's parameter must match the `Pair`'s value type including its
nullability, and the callback's own result nullability is preserved:

```text
KeyValue.mapValues(Pair<K, V>, Function(V) -> U?)  -> Pair<K, U?>
```

## merge and overlay

These are two functions on purpose, not one with a `Bool` flag, because the
name is what tells a reader which intent was meant.

**`merge` is a safe disjoint union.** A key present in both `Pair`s is a
`KeyValueError` — the module will not silently choose left-wins or
right-wins:

```ahd
write(KeyValue.merge({"a": 1, "b": 2}, {"c": 3}))
```

```text
{"a": 1, "b": 2, "c": 3}
```

```ahd
write(KeyValue.merge({"a": 1}, {"a": 9}))     // KeyValueError
```

Order is left order first, then right order.

**`overlay` is the explicitly named changes-win operation:**

```ahd
write(KeyValue.overlay({"a": 1, "b": 2}, {"b": 9, "c": 3}))
```

```text
{"a": 1, "b": 9, "c": 3}
```

An existing base key keeps its position and takes the new value; a
changes-only key is appended in `changes` insertion order. Neither source is
modified.

Both require exactly the same `Pair` type, including value nullability.

## Ordering is a public contract

| Operation | Order |
| --- | --- |
| `keys`, `values` | `Pair` insertion order |
| `combine` | key-List order |
| `with`, existing key | its existing position |
| `with`, new key | appended last |
| `without` | surviving source order |
| `select` | requested key-List order |
| `drop` | surviving source order |
| `rename` | the original key's position |
| `mapValues` | original key order |
| `merge` | left order, then right order |
| `overlay`, existing key | its base position |
| `overlay`, new key | appended in `changes` order |

## Nullability

`Pair` keys are never null. Value nullability is structural and is preserved
exactly:

```text
KeyValue.values(Pair<K, V?>)                 -> List<V?>
KeyValue.combine(List<K>, List<V?>)          -> Pair<K, V?>
KeyValue.mapValues(Pair<K, V>, V -> U?)      -> Pair<K, U?>
```

`KeyValue.with(pair, key, null)` is accepted only when the `Pair`'s value type
is nullable.

## Shallow structural semantics

Every operation is pure with respect to collection *structure*: it never
mutates the `Pair` or `List` it is given, so a `Constant` collection may be
passed safely, and every returned collection is structurally independent.

The transformation is **shallow**. Keys and values are carried over by
reference, not deep-copied:

```ahd
byName: Pair<String, Student> := {"ali": student}
copied := KeyValue.with(byName, "ayse", other)
write(copied["ali"] same student)     // true
```

`with`, `select`, `drop`, and the rest copy the `Pair` structure, not the
`Student` object. This is structural immutability, not deep immutability.

## Errors

`KeyValueError` derives directly from `Error` and covers the module's own
structural failures:

- a `combine` length mismatch;
- a duplicate `combine` key;
- a duplicate `select` or `drop` request;
- a `rename` collision with an existing different key;
- a `merge` key collision.

A genuinely **missing** `Pair` key uses the language's existing `KeyError`, so
`without`, `select`, `drop`, and `rename` report a missing key exactly the way
`Pair` itself already does.

Type-invalid calls stay compile-time type errors. A callback that raises keeps
its own error type — it is never wrapped.

## Updating JSON without String surgery

A `JSONValue` object is an ordinary `Pair<String, JSONValue>`, so `KeyValue`
updates a JSON document without ever leaving the typed representation:

```ahd
object: Pair<String, JSONValue> := data.object()

updatedObject: Pair<String, JSONValue> := KeyValue.with(
    object
    "books"
    JSON.array(newBooks)
)

JSON.write(JSON.object(updatedObject), "library.json", true)
```

Every other root field survives untouched and keeps its position. This
replaces the `JSON.stringify` → String concatenation → `JSON.parse` round trip
entirely; see `examples/v0.1/49_json_record_update.ahd`.

## Non-goals

`KeyValue` deliberately does not add:

- `entries` — an "entry" has no honest representation until AhdCode has a real
  type for one; a one-element `Pair` is not it;
- `invert`, or other speculative conveniences;
- any heterogeneous container, `Tuple`, `Set`, or `Any`-typed bag;
- JSON-, Data-, or SQL-specific behavior. `KeyValue` stays a general `Pair`
  module, so JSON, Data, CSV, and future storage modules can all reuse it
  without each inventing a conversion layer.

## See also

- [Lists](LISTS.md) — the same idea for `List`
- [Collections](COLLECTIONS.md) — `List` and `Pair` fundamentals
- [JSON](JSON.md) — typed JSON documents
- [Data](DATA.md) — tabular data built on `Pair<String, String>` rows
- `examples/v0.1/48_key_value.ahd`, `examples/v0.1/49_json_record_update.ahd`,
  `examples/v0.1/50_data_records.ahd`
