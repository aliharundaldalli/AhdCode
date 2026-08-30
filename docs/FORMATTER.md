# Formatter

[English] · [Türkçe](FORMATTER_TR.md)

[Back to README](../README.md) · [CLI](CLI.md)

Format a source file in place:

```bash
ahdcode format program.ahd
```

Check formatting without modifying the file:

```bash
ahdcode format --check program.ahd
```

The formatter is AST-aware. It preserves comments, string escapes,
interpolation, and exact multiline-string content, and it renders the single
canonical layout for everything else:

- A call, List literal, Pair literal, or Function/structure parameter list
  collapses onto one line, comma-separated, when that line fits in 80
  columns.
- One that does not fit breaks to one item per line, with **no comma at
  all** -- not even a trailing one.
- The `(parameters) -> ReturnType` shape of a Function signature is always
  kept together on the line that opens its body.
- Lambda uses the same parameter-list layout and canonical spacing:
  `lambda (x: Int) -> x > 0`; only its parameter list breaks when needed.
- Indentation is always 4 spaces per level, independent of how the source
  was indented.

Formatting is deterministic and idempotent: `format(format(source)) ==
format(source)`. In-place updates use an atomic temporary-file replacement
and preserve the file's permission bits. Invalid source is diagnosed and
left unchanged -- the formatter never partially rewrites a file that does
not already parse.

## Recommended style vs. valid syntax

These are two different things. **Valid syntax** is whatever the parser
accepts: AhdCode requires a comma between two items on the same line, but
newlines and trailing commas are otherwise all optional, and indentation
carries no meaning at all. See the [Language tour](LANGUAGE_TOUR.md) for the exact
grammar rule. **Recommended style** is the one layout `ahdcode format`
produces from any of them. Write whichever valid form is convenient --
generated code that mixes styles across a file is fine -- and let the
formatter normalize it.

The one placement rule that is not just style: the expression on the right
of `:=` or `=` must start on the same physical line as the operator. Every
other line break, including the one right after `:=`/`=` and before an
opening bracket, is free.

A short call, several equally valid spellings:

```ahd
add(1, 2)

add(
    1,
    2
)

add(
    1
    2
)
```

`ahdcode format` renders all three as:

```ahd
add(1, 2)
```

A Function signature, short vs. long -- the formatter decides the line
break based on width, not on how the source was written:

```ahd
check: Function := (
    x: Int,
) -> Bool {
    return x > 0
}

calculate: Function := (first: Int, second: Int, description: String, flag: Bool) -> Real {
    return first
}
```

becomes:

```ahd
check: Function := (x: Int) -> Bool {
    return x > 0
}

calculate: Function := (
    first: Int
    second: Int
    description: String
    flag: Bool
) -> Real {
    return first
}
```

A List and a Pair, written two different valid ways:

```ahd
numbers: List<Int> := [
    1,
    2,
    3,
]

scores: Pair<String, Int> := {"Ali": 85, "Ayşe": 92}
```

become:

```ahd
numbers: List<Int> := [1, 2, 3]

scores: Pair<String, Int> := {"Ali": 85, "Ayşe": 92}
```

`scores` was already short enough to stay on one line, so nothing about it
changes -- only spacing and the trailing comma on `numbers` do.
