# Errors

[Back to README](../README.md)

AhdCode errors are catchable Class values derived from `Error`.

```ahd
attempt {
    toss(DomainError("invalid value"))
}
except DomainError as error {
    write(error.message)
}
ultimately {
    write("finished")
}
```

- `attempt` runs protected code.
- `except ErrorType as error` handles a matching error.
- `ultimately` always runs, including before a pending return completes.
- `toss` raises an Error instance.

Common built-in errors include:

| Error | Typical cause |
|---|---|
| `DivisionByZeroError` | division or modulo by zero |
| `OverflowError` | checked Int or finite Real overflow |
| `DomainError` | valid type but invalid mathematical/search domain |
| `IndexError` | invalid List/String index |
| `KeyError` | missing Pair key |
| `NullError` | runtime null safety boundary |
| `ConstantError` | mutation through a deep-frozen reference |
| `ValueError` | invalid runtime value such as negative String repetition |

Custom errors use ordinary inheritance:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}
```
