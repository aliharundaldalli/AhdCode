# Fundamentals

[Back to README](../README.md) · [String API](STRING_API.md) · [List API](LIST_API.md)

These names are predeclared in every module and require no `bring`:

```text
write take str int real len clear between abs sum min max type id
```

| Function | Behavior |
|---|---|
| `write(value)` | writes one value followed by a newline |
| `take()` / `take(prompt)` | reads one line as String |
| `str(value)` | canonical deterministic text |
| `int(Real)` | truncates toward zero |
| `int(String)` | strict signed ASCII-decimal parse |
| `real(Int)` | explicit safe widening |
| `real(String)` | strict decimal/fraction/exponent parse |
| `len(value)` | String characters, List elements, or Pair entries |
| `clear(collection)` | empties List or Pair in place |
| `between(...)` | lazy stop-exclusive Int iteration |
| `abs(number)` | numeric magnitude with exact result type |
| `sum(list)` | numeric reduction; empty List gives `0` or `0.0` |
| `min(list)` / `max(list)` | numeric extrema; empty List raises `DomainError` |
| `type(value)` | canonical AhdCode type name as String |
| `id(reference)` | opaque runtime identity of a Class instance, List, or Pair |

Conversions trim surrounding Unicode whitespace in String input. `int(String)`
does not accept fractions, exponents, underscores, or base prefixes.
`real(String)` accepts decimal integer, fraction, and exponent forms but not
NaN or infinity. Invalid text raises `DomainError`; out-of-range text raises
`OverflowError`.

`str` preserves List order and Pair insertion order, quotes nested Strings,
never exposes attributes, and prints a Class instance as `<ClassName>` unless
that Class implements the `CStr` [Class Protocol Method](PROTOCOLS.md), in
which case `str` (and therefore `write` and String interpolation, which share
the same conversion) dispatch to it instead.

`clear` mutates the existing collection, so aliases observe it and Constant
collections reject it. The numeric reductions are pure reads and accept a
non-null Constant List.

## `type(value)`

`type(value) -> String` is a runtime/introspection aid, most useful in the
REPL. It is not a reflection framework: it returns the canonical AhdCode type
name as a String, never a first-class Type object and never a Go
implementation name.

```ahd
write(type(5))          // "Int"
write(type(5.0))        // "Real"
write(type("Ali"))      // "String"
write(type(true))       // "Bool"
write(type(null))       // "Null"

numbers: List<Int> := [1, 2]
write(type(numbers))    // "List<Int>"
```

For a Class instance, `type` reports the **most-derived runtime Class name**,
not the statically declared type:

```ahd
Animal: Class<> := { structure: Attributes := (name: String) }
Dog: Class<Animal> := { structure: Attributes := (SuperClass.attributes) }

pet: Animal := Dog(name: "Rex")
write(type(pet))        // "Dog", not "Animal"
```

For a nullable value that currently holds a non-null value, `type` reports
that value's own type, not the declared type's `?`:

```ahd
x: Int? := 5
write(type(x))          // "Int"
```

`type(null)` always reports `"Null"`. This is an intrinsic case in the
Fundamental itself, not a new source-level `Null` declaration type -- `x :=
null` is still rejected exactly as in v0.1.7.

## `id(reference)`

`id(reference) -> Int` returns a runtime-managed identity number for
debugging, logging, and introspection. It is **not** a memory address and
carries no guarantee beyond the current process or REPL session.

Only reference values with meaningful AhdCode identity are accepted in
v0.1.8: a Class instance, a List, or a Pair. A primitive (`Int`, `Real`,
`Bool`, `String`) has no identity to report and is a compile-time error:

```ahd
id(5)       // compile-time error
id("Ali")   // compile-time error
```

```ahd
a := [1, 2]
b := a
c := [1, 2]

write(id(a) == id(b))   // true -- same object
write(id(a) == id(c))   // false -- distinct, separately allocated objects

a.add(3)
write(id(a) == id(b))   // still true -- mutation never changes identity
```

The number is opaque and process/session-local: it is not a memory address,
not guaranteed stable between separate program runs, not serialization data,
and not a persistent database identifier. Do not depend on its ordering or on
any particular starting value.

`id` requires a proven-`NonNull` argument, exactly like any other operation on
a nullable reference:

```ahd
user: User? := fetchUser()
id(user)                 // compile-time error: user may be null
if user != null {
    write(id(user))      // fine once narrowed
}
```

`id` does not replace `same`. `a same b` is the ordinary programmatic
identity test used in everyday AhdCode code; `id(a)` exists specifically to
print or log that identity. For any two live List, Pair, or Class instance
values, `(a same b) == (id(a) == id(b))`.
