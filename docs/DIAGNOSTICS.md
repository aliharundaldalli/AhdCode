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

6. v0.1.11 lambdas do not capture surrounding `Local` values or Function
parameters. Pass the value as another lambda parameter or use a normal
Function.

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
