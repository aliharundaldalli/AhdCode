# Types, inference, and null safety

[Back to README](../README.md) · [Collections](COLLECTIONS.md)

## Static types and inference

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

An annotation is optional when the initializer has one unambiguous complete
static type:

```ahd
age := 25
name := "Ali"
```

These are not dynamic variables. `age = "Ali"` is a compile-time error.
Explicit annotations remain useful and sometimes required:

```ahd
age: Int := 25
```

Inside a nested executable scope, scope intent is still explicit:

```ahd
name: Local := "Ali"       // inferred Local String
user: Local User? := null  // explicit type because null cannot infer User
```

A bare nested `name := "Ali"` is invalid. A Function that accesses module
storage still follows the existing explicit `Global` rules. Function
parameters and returns stay explicitly typed, and Class syntax remains
`Person: Class<> := { ... }`.

## Nullable types

Plain `T` is non-nullable. `T?` says that a value may be absent:

```ahd
user: User? := null
```

Nullability composes at each structural level:

```text
User           non-null User
User?          nullable User
List<User?>    non-null List with nullable elements
List<User>?    nullable List with non-null elements
List<User?>?   nullable List with nullable elements
```

Inference preserves this complete type. If `fetchUser()` returns `User?`, then
`user := fetchUser()` infers `User?`. Bare `value := null` is invalid because
`null` cannot determine an underlying type; `value: User? := null` is valid.

`T` safely fits `T?`. The reverse direction is rejected unless control flow
has proven the expression non-null:

```ahd
if user != null {
    write(user.name)
}
```

Short-circuiting refines the right side:

```ahd
if user != null and user.age >= 18 {
    write(user.name)
}
```

An early null return also refines later code. Reassigning a nullable binding
invalidates earlier proof. Nullable parameter/return types and overloads use
the same rules: `(Int)` and `(Int?)` are distinct signatures, and an unresolved
`Int?` cannot silently call the non-null overload.

A nullable collection receiver must be narrowed before member/index access.
For `List<T?>`, each element is independently nullable and must likewise be
checked. `Constant` deep-freeze behavior is unchanged by nullability.
AhdCode has no truthiness, optional chaining, null-coalescing, or force-unwrap
syntax in v0.1.

## `Nothing` is different

`Nothing` is the return type of an action, not a nullable value:

```ahd
report: Function := (
    message: String
) -> Nothing {
    write(message)
}
```

It cannot be stored, printed, interpolated, or returned as a value.
