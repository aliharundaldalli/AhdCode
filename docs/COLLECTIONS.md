# Collections

[Back to README](../README.md) · [List API](LIST_API.md)

## Lists

```ahd
numbers: List<Int> := [1, 2, 3]
write(numbers[0])
write(numbers[-1])
write(numbers[1:])
```

Lists are homogeneous, ordered reference objects. Negative indices are
supported; invalid indices raise `IndexError`. A bare empty List needs an
explicit element type.

```ahd
alias: List<Int> := numbers
alias[0] = 9
write(numbers[0]) // 9
```

`==` compares collection contents deeply. `same` compares object identity.

## Pairs

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}

scores["Ali"] = 90
write(scores["Ali"])
```

Pairs preserve insertion order. Updating a key keeps its position; ejecting
and re-adding it appends it. Missing keys raise `KeyError`. Valid key types are
`String`, `Int`, and `Bool`; Real, Class, and null keys are not supported.

`value in scores` checks Pair keys. `has` is only for Class member existence.

## Constant and deep freeze

```ahd
values: Constant List<Int> := [1, 2, 3]
```

The object and its reachable reference graph are frozen. Mutation through any
alias raises `ConstantError` (and directly known mutation is rejected during
checking). An independent mutable copy requires future library functionality;
v0.1 does not yet publish `copy` or `deepCopy`.

Iteration over List elements and Pair keys uses a shallow snapshot. Structural
mutation does not change what the active loop visits.
