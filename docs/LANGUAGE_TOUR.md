# Language tour

[English] · [Türkçe](LANGUAGE_TOUR_TR.md)

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

For a single expression, `lambda` creates a value of that same `Function`
type. Parameters are explicitly typed and the return type is inferred:

```ahd
squareShort := lambda (value: Int) -> value^2
values := [1, 2, 3]
squares := values.map(lambda (value: Int) -> value^2)
```

Lambda has one expression only: there is no block/statement lambda, no
separate Lambda type, and no implicit coercion.

If a lambda needs a variable from outside its parameters, you must declare the dependency explicitly in square brackets:

```ahd
minimum: Int := 50
passed := values.filter(lambda [#minimum] (score: Int) -> score >= minimum)
```

Use `#` for `Local` by-value capture and `@` for live `Global` dependency. No dependency is inferred automatically.

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
symbol. Local modules are sibling files. `Math`, `Time`, `Latex`, `Word`, `Excel`,
`PDF`, `Archive`, `Path`, `Regex`, `CSV`, `Data`, `File`, `Statistics`, `Plot`,
`Numeric`, `JSON`, `SQLite`, `HTTP`, `HTML`, `XML`, `Env`, `Lists`, and `KeyValue` are explicit standard
modules; their domain and file failures are catchable AhdCode errors.

Continue with [Functions](FUNCTIONS.md), [Classes](CLASSES.md),
[Class Protocol Methods](PROTOCOLS.md), [Collections](COLLECTIONS.md), and
[Modules](MODULES.md). Time covers local, UTC, and fixed-offset instants; CSV
transports String rows and records, and Data turns them into an immutable
`Table` of String cells; JSON and XML are typed structured-data models; SQLite
is a typed local-database bridge; HTTP is a local web server with Request and
Response values, cookies, and in-memory server-side sessions, plus an outbound
Client for HTTP/HTTPS APIs; HTML is a small safe structured builder; Env reads process/`.env` configuration; Lists and KeyValue are the pure
structural transformation layer over `List` and `Pair`; Excel adds typed,
immutable XLSX workbooks; PDF builds immutable documents and renders them
offline to real `.pdf` files (sharing Latex's staged Tectonic renderer), with
semantic `PDF.fromWord`/`PDF.fromExcel` conversion; Archive packages files
into real ZIP/TAR/TAR.GZ archives, creation-only, using nothing beyond the Go
standard library. See [Time](TIME.md), [CSV](CSV.md), [Data](DATA.md), [Statistics](STATISTICS.md), [Plot](PLOT.md), [Numeric](NUMERIC.md), [Word](WORD.md), [Excel](EXCEL.md), [PDF](PDF.md), [Archive](ARCHIVE.md), [JSON](JSON.md), [SQLite](SQLITE.md), [HTTP](HTTP.md), [HTML](HTML.md), [XML](XML.md), [Env](ENV.md), [Lists](LISTS.md), [KeyValue](KEYVALUE.md), and the
[diagnostics guide](DIAGNOSTICS.md).
