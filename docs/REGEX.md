# Regex standard module

[English] · [Türkçe](REGEX_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Types and Null](TYPES_AND_NULL.md)

Regex is explicit, like Math and Time:

```ahd
bring Regex
from Regex bring Pattern
from Regex bring RegexError
```

The canonical identity is `builtin:Regex`; a sibling `Regex.ahd` cannot
shadow it. Every argument is `NonNull`.

## Compiling a pattern

```text
Regex.compile(pattern: String) -> Pattern
```

`Regex.compile` compiles a pattern written in Go `regexp` (RE2) syntax and
returns a `Pattern` instance. An invalid pattern raises the catchable
`RegexError`. `Pattern` is a compiler-supplied Class -- it is never
constructed directly, only produced by `Regex.compile`, the same way
`Time.dateTime` is the only source of a `DateTime`.

The Class is named `Pattern`, not `Regex`: `bring Regex` already binds the
name `Regex` to the module namespace, so the compiled-pattern type needs its
own name to be importable by itself with `from Regex bring Pattern`.

```ahd
bring Regex
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")
```

## Surface

```text
matches(text: String)                      -> Bool
find(text: String)                         -> String?
findAll(text: String)                      -> List<String>
groups(text: String)                       -> List<String>?
replace(text: String, replacement: String) -> String
split(text: String)                        -> List<String>
```

`matches` reports whether the pattern is found **anywhere** in `text` -- it
is not an implicit full-string match. Anchor the pattern yourself
(`"^...$"`) if you need that.

```ahd
digits.matches("order #482")   // true
digits.matches("no digits")    // false
```

`find` returns the first match, or `null` if the pattern does not occur.
Because the result is `String?`, the ordinary nullable-use rules apply before
you can use it:

```ahd
first: String? := digits.find("order #482, item #7")
if first != null {
    write(first)
}
```

`findAll` returns every non-overlapping match, in order; an empty
`List<String>` when there is no match. `replace` rewrites **every** match
with `replacement`, which may reference capture groups as `$1`, `$2`, and so
on. `split` divides `text` on every match of the pattern.

```ahd
write(digits.findAll("order #482, item #7"))       // ["482", "7"]
write(digits.replace("order #482, item #7", "N"))  // "order #N, item #N"

whitespace: Pattern := Regex.compile("\\s+")
write(whitespace.split("one   two\tthree"))          // ["one", "two", "three"]
```

`groups` returns the first match's full match text followed by its capture
groups (index `0` is the whole match), or `null` if the pattern does not
occur anywhere in `text`. An unmatched optional group reports as an empty
`String`.

```ahd
entry: Pattern := Regex.compile("([a-zA-Z]+)-([0-9]+)")
parts: List<String>? := entry.groups("item-42")
if parts != null {
    write("whole: {parts[0]}, name: {parts[1]}, number: {parts[2]}")
}
```

## RegexError

`RegexError` derives directly from `Error` (not `IOError`) and is raised only
by `Regex.compile` on invalid pattern syntax. No `Pattern` operation --
`matches`, `find`, `findAll`, `groups`, `replace`, `split` -- can fail once a
`Pattern` exists.

```ahd
bring Regex
from Regex bring Pattern
from Regex bring RegexError

attempt {
    Regex.compile("(unterminated")
} except RegexError as error {
    write(error.message)
}
```

## Caching and determinism

A `Pattern`'s only observable state is its source pattern text; the compiled
matcher itself is cached internally by pattern text, so using the same
`Pattern` value repeatedly -- or calling `Regex.compile` again with an
identical pattern string -- does not repeatedly pay compilation cost.
Matching, replacement, and splitting are deterministic for a given pattern
and input, and behave identically in the native backend and the persistent
REPL.
