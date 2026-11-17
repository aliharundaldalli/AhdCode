# Understanding diagnostics

[English] · [Türkçe](DIAGNOSTICS_TR.md)

[Back to README](../README.md) · [Getting started](GETTING_STARTED.md) · [Errors](ERRORS.md)

A diagnostic reports a code, severity, source span, explanation, and usually an
actionable hint. Codes help identify a rule but are not promised as a permanent
external ABI. Fix the first diagnostic in a malformed construct before acting
on later messages; v0.1.11 deliberately suppresses common parser cascades while
continuing to report independent errors later in the file.

## Common corrections

1. An assigned expression must start beside its operator.

```ahd
# invalid: PAR010
value :=
    5

# corrected
value := 5
```

2. A method chain cannot start with `.` on a new line.

```ahd
# invalid: PAR013
important := entries
    .filter(lambda (x: String) -> x != "")

# corrected
filtered := entries.filter(lambda (x: String) -> x != "")
important := filtered
```

3. A declaration inside an executable block needs `Local`.

```ahd
# invalid
if ready {
    count: Int := 1
}

# corrected
if ready {
    count: Local Int := 1
}
```

4. Guard a nullable value before a non-null operation.

```ahd
# invalid: user may be null
write(user.name)

# corrected
if user is not null {
    write(user.name)
}
```

5. A lambda body is one expression, never a block.

```ahd
# invalid
double := lambda (x: Int) -> { return x * 2 }

# corrected
double := lambda (x: Int) -> x * 2
```

6. A lambda reads a binding outside its own parameters only through its
explicit dependency list: `#name`/`Local name` for an enclosing `Local` or
Function parameter, `@name`/`Global name` for a module binding. A bare name
(`lambda [minimum] (...)`) is rejected — state the kind, as in
`lambda [#minimum] (...)`. See [Functions](FUNCTIONS.md#explicit-lambda-dependencies).

7. Function diagnostics name wrong arity, argument type, or named argument.
Correct the call to match the declared parameter names and types; AhdCode does
not silently coerce unrelated values.

8. Protocol diagnostics name the contract. For example, fix a `CCompare`
return from `String` to `Int`, or a `CStr` return to `String`; do not add new
protocol names.

9. Invalid regular expressions raise catchable `RegexError` at runtime:

```ahd
attempt { Regex.compile("(unfinished") }
except RegexError as error { write(error.message) }
```

10. Invalid dates, fixed offsets outside -840..840, and timestamps outside
DateTime year 1..9999 raise `ValueError`. Correct the component or timestamp;
no Go panic or stack trace is part of the language behavior.

11. Malformed quoting, invalid delimiters, duplicate/empty headers, and record
shape mismatches raise `CSVError`:

```ahd
attempt { CSV.parse("a,\"unfinished") }
except CSVError as error { write(error.message) }
```

Incomplete initializers, assignments, binary operands, indexes, calls, lists,
pairs, and lambda bodies use construct-aware messages where the parser can
identify the missing piece. Closing-delimiter diagnostics name the expected
`)`, `]`, or `}` and point at the recovery location.

## One mistake, one diagnostic

Since v2.8.0 the compiler reports one diagnostic for one mistake, so the first
message on the screen is the one to fix:

- A line reports only its first syntax error.
- After an error the lexer reports, such as an unclosed string, nothing later
  in the file is reported until that one is fixed.
- Type checking runs only on a file that parses, so a syntax error is never
  followed by errors about names it left undeclared.
- A statement that already reported an error does not also report the missing
  end of its line, and a block whose `{` is missing does not also report its
  `}`.

12. A reserved keyword cannot be a name (`PAR015`). It is reported once, where
it is first used as a name, with a suggestion:

```text
error [PAR015] main.ahd:1:1
'not' is a reserved keyword in AhdCode
  1 | not: Int := 70
    | ^^^
  hint: choose a different identifier, such as 'score'
```

13. A declaration introduces its name with `:=`. A lone `=` only changes a
name that already exists.

```ahd
# invalid: PAR002
count: Int = 5

# corrected
count: Int := 5
```

14. A block opens on the line that introduces it.

```ahd
# invalid: PAR002
if ready
{
    write("go")
}

# corrected
if ready {
    write("go")
}
```

A Function is the one exception: its `uses [...]` list and its `{` may each
start a new line, and `ahdcode format` joins them into one header.

15. A statement written the way another language writes it (`function`, `def`,
`var`, `let`, `print`, `elif`, and similar) is answered with the AhdCode
spelling in the hint.

16. A declaration with a modifier and no type takes the type of its value:
`limit: Constant := 20.0` is a `Constant Real`. Before v2.8.0 this spelling
failed with an internal `IR002` diagnostic.

`FMT001` is not a mistake in a program. `ahdcode format` checks that the
layout it produced has exactly the tokens and comments of the source; if it
does not, the formatter reports `FMT001` and leaves the file unchanged
instead of writing a damaged one. Please report the file.
