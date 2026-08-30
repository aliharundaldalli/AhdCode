# Language tour

[Back to README](../README.md) · [Types and null](TYPES_AND_NULL.md)

## Declarations and mutation

`:=` creates a binding. `=` changes an existing binding.

```ahd
name := "Ali"       // inferred String, still statically typed
count: Int := 3     // explicit annotations remain available
count = 4
```

Inference preserves the complete static type. `count = "four"` is still an
error. Bare `value := null` is invalid because no underlying type can be
inferred; write `value: User? := null`.

Line breaks otherwise carry no meaning -- indentation is purely for
readability, and `ahdcode format` (see [Formatter](FORMATTER.md)) chooses it
for you -- but the value on the right of `:=` or `=` must start on the same
physical line as the operator:

```ahd
scores: List<Int> := [
    1
    2
]
```

is valid (`[` opens right after `:=`), while writing the `[` on its own line
after `:=` is rejected with a dedicated error rather than a generic parse
failure.

Inside an executable nested scope, a new declaration uses `Local`:

```ahd
if count > 0 {
    message: Local := "Ready"
    write(message)
}
```

A Function that reads or mutates a module-root binding declares that access
with `Global`:

```ahd
counter: Int := 0

increase: Function := (
) -> Nothing {
    counter: Global Int
    counter++
}
```

## Values and collections

```ahd
score: Int := 90
average: Real := 87.5
passed: Bool := score >= 50
student: String := "Ayşe"

scores: List<Int> := [90, 85, 92]
grades: Pair<String, Int> := {
    "Ali": 90
    "Ayşe": 92
}
```

Lists and Pairs are reference objects. Aliases observe mutation. A `Constant`
reference deep-freezes its reachable object graph.

## Conditions and loops

```ahd
if passed {
    write("Passed")
}
else {
    write("Try again")
}

for value in between(1, 4) {
    write(value)
}
```

Only `Bool` is a condition; AhdCode has no truthiness.

## Functions and Classes

```ahd
square: Function := (
    value: Int
) -> Int {
    return value^2
}
```

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Constant Int
    )

    describe: Function := (
    ) -> String {
        return "{attribute.number}: {attribute.name}"
    }
}
```

A Class may define `==`, ordering, arithmetic, unary `-`, and `str()`
behavior through ten exact
[Class Protocol Methods](PROTOCOLS.md) (`CEqual`, `CCompare`, `CAdd`,
`CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower`, `CNegate`,
`CStr`) -- an ordinary method using regular Function syntax:

```ahd
Vector2: Class<> := {
    structure: Attributes := (x: Real, y: Real)
    CAdd: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }
}
```

## Errors and modules

Use `attempt`, `except`, `ultimately`, and `toss` for catchable errors. Use
`bring ModuleName` for a namespace or `from ModuleName bring name` for a direct
symbol. Local modules are sibling files. `Math`, `Time`, `Latex`, `Path`, and
`File` are explicit standard modules; `File` failures are catchable.

Continue with [Functions](FUNCTIONS.md), [Classes](CLASSES.md),
[Class Protocol Methods](PROTOCOLS.md), [Collections](COLLECTIONS.md), and
[Modules](MODULES.md).
