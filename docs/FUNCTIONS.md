# Functions

[English] · [Türkçe](FUNCTIONS_TR.md)

[Back to README](../README.md) · [Classes](CLASSES.md)

Functions are named declarations at module root; methods are named declarations
inside a Class. Nested Function declarations remain unsupported. A lambda is
an expression-only shorthand that creates an anonymous value of the existing
`Function` type; it is not a new type or a second callable system.

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

## Expression lambdas

```ahd
square := lambda (x: Int) -> x^2
positive: Function := lambda (x: Int) -> x > 0

values := [1, 2, 3]
squares := values.map(lambda (x: Int) -> x^2)
```

The exact syntax is `lambda (<typed parameters>) -> <expression>`. Every
parameter has an explicit static type; the return type and return nullability
come from the one body expression. Zero parameters and every parameter type
already valid for a Function are supported. A compatible lambda works anywhere
a `Function` value is accepted, including `map`, `filter`, and the existing
`sort` key callback. No implicit coercion is added.

Lambda parameters are required; default-valued parameters remain available on
named Function declarations, not expression lambdas, in v0.1.11.

A lambda cannot contain a block or statements. Use the unchanged named
Function syntax when control flow, declarations, loops, error handling, or
multiple steps are needed. Ordinary `Local`/`Global` visibility rules stay
unchanged.

## Explicit lambda dependencies

A lambda reads a binding from outside its own parameters only by listing that
binding in an explicit dependency list, written between `lambda` and the
parameters. Each entry states its own kind:

- `#name` or `Local name` — a lexical capture of an enclosing binding, by
  value.
- `@name` or `Global name` — an explicit dependency on a module/global
  binding, mirroring the `Global` declaration an ordinary Function already
  needs to touch module state.

```ahd
keepAbove: Function := (
    minimum: Int
    scores: List<Int>
) -> List<Int> {
    return scores.filter(lambda [#minimum] (score: Int) -> score >= minimum)
}

Maximum: Int := 100
inRange: Function := lambda [#minimum, @Maximum] (score: Int) -> score >= minimum and score <= Maximum
```

Several entries are separated by commas: `lambda [#low, #high] (v: Int) -> ...`.
`#name`/`Local name` and `@name`/`Global name` are alternate spellings of the
same dependency, and a list may mix compact and full spellings freely; the
formatter preserves whichever spelling the source used.

A dependency is never inferred, and a bare name (`lambda [minimum] (...)`, the
unpublished pre-v0.1.13 spelling) is rejected: every entry must state whether
it is Local or Global. Reading an enclosing `Local` or Function parameter that
the list omits is a compile-time error naming the binding, so what a lambda
depends on — and how — is visible where the lambda is written:

```text
SEM043  local "minimum" is not a lambda dependency
SEM007  module binding "Maximum" requires an explicit Global dependency
```

A lambda written without a dependency list reads nothing outside its own
parameters, so every lambda that compiled before v0.1.13 still compiles
unchanged. `lambda [] (...)` is accepted and means the same thing.

`#`/`Local` names only a binding of an enclosing callable: a Function
parameter, a `Local`, or a `for`/`except` binding. `@`/`Global` names only a
module-root value binding. A module-root Class, Function, or namespace is
reached by ordinary lookup and must not be listed under either kind.

**A `#`/`Local` capture is by value.** It reads what the binding held when the
lambda value was created, so a later change to that binding is not visible
inside the lambda:

```ahd
step: Local Int := 1
first: Local Function := lambda [#step] (x: Int) -> x + step
step = step + 100
second: Local Function := lambda [#step] (x: Int) -> x + step
// first(0) is 1 and second(0) is 101
```

Reference values follow the language's ordinary rules: capturing a `List`,
`Pair`, or Class instance copies the reference, exactly as passing it as a
parameter does, so the referenced object stays shared.

A captured name is read-only inside the lambda. `#`/`Local` gives the lambda
the enclosing value, not ownership of the enclosing variable, and v0.1.13 adds
no mutable closure cell or reference box.

**An `@`/`Global` dependency is not a capture.** It does not snapshot the
module binding or copy it into closure storage; it reads the real binding
under AhdCode's ordinary global-mutation rules, so a legal mutation after the
lambda was created is visible the next time the lambda runs:

```ahd
Maximum: Int := 100
check: Function := lambda [@Maximum] (score: Int) -> score <= Maximum

check(50)  // true, Maximum is 100
Maximum = 40
check(50)  // false, @Maximum observes the live binding
```

Dependencies work with every existing callback, including
[Data](DATA.md)'s `filter`, `sort`, `transform`, and `derive`:

```ahd
strong: Local Table := table.filter(
    lambda [#minimum] (row: Pair<String, String>) -> int(row["score"]) >= minimum
)
```

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
