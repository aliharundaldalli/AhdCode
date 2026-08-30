# Functions

[Back to README](../README.md) · [Classes](CLASSES.md)

Functions are named declarations at module root; methods are named declarations
inside a Class. v0.1 has no nested Function declarations or lambdas.

```ahd
greet: Function := (
    name: String
    title: String := "Student"
) -> String {
    return "Hello {title} {name}"
}
```

Commas between parameters are optional wherever a newline already separates
them, and a trailing comma is always allowed -- `(name: String, title: String
:= "Student")` on one line is the same declaration. `ahdcode format` (see
[Formatter](FORMATTER.md)) renders whichever spelling you use in the single
recommended style: one line if it fits, otherwise one parameter per line with
no comma at all, as above.

Calls are entirely positional or entirely named:

```ahd
write(greet("Ali"))
write(greet(name: "Ali", title: "Dr"))
```

`greet("Ali", title: "Dr")` is invalid because it mixes the two forms.
Required parameters precede default parameters.

## Return behavior

A value-returning Function must return a compatible value on every reachable
path. It may return `null` only when its declared return type is nullable, such
as `User?`. A `Nothing` Function may use bare `return` or reach its end.

## Overloads

```ahd
double: Function := (
    value: Int
) -> Int {
    return value * 2
}

double: Overload Function := (
    value: Real
) -> Real {
    return value * 2.0
}
```

Exact argument matches beat safe widening. Equal best matches are compile-time
ambiguities, and return type alone cannot choose an overload.

## Function values and callbacks

```ahd
apply: Function := (
    operation: Function
    value: Int
) -> Int {
    return operation(value)
}

result: Int := apply(double, 5)
```

Users write only `Function`, but it is not dynamic. Every Function binding or
parameter must resolve to one concrete callable signature at compile time.
