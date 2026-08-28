# Types and null safety

[Back to README](../README.md) · [Collections](COLLECTIONS.md)

## Core types

| Type | Meaning |
|---|---|
| `Int` | checked signed 64-bit integer |
| `Real` | finite 64-bit floating point |
| `String` | immutable Unicode text |
| `Bool` | `true` or `false` |
| `List<T>` | ordered homogeneous mutable collection |
| `Pair<K, V>` | insertion-ordered homogeneous key/value collection |
| `Class` | class declaration and instance identity |
| `Function` | statically resolved named callable value |
| `Nothing` | Function return type with no runtime value |

`Int` safely widens to `Real` where allowed. Mutable generic collections are
invariant: `List<Int>` is not a `List<Real>`.

## `null` is state, not a separate type

```ahd
student: Student := null
```

The binding is still a `Student`; its value is absent. The compiler tracks
whether an expression is definitely null, maybe null, or non-null.

```ahd
if student != null {
    write(student.name)
}
```

The branch refines `student` to non-null. Short-circuiting also refines:

```ahd
if student != null and student.age >= 18 {
    write(student.name)
}
```

Unsafe member access, arithmetic, calls, indexing, and other operations on a
possibly-null value are rejected. Runtime checks remain a final safety layer.

List elements and Pair values may be null when their surrounding type is
known. Pair keys may not be null. A `Constant` cannot be initialized to null.

## `Nothing` is different

`Nothing` is the return type of an action:

```ahd
report: Function := (
    message: String
) -> Nothing {
    write(message)
}
```

It cannot be stored, printed, interpolated, or returned as a value.
