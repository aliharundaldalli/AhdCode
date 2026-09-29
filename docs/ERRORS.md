# Errors

[English] · [Türkçe](ERRORS_TR.md)

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
| `IOError` | base class for input/output failures |
| `FileError` | `File` module operation failure; derives from `IOError` |
| `RegexError` | invalid pattern passed to `Regex.compile`; derives from `Error` |
| `CSVError` | malformed CSV, invalid delimiter, or invalid record/header shape; derives from `Error` |
| `UUIDError` | malformed UUID text, or no random source or valid clock for a new UUID; derives from `Error` |
| `PostgreSQLError` | PostgreSQL connection, query, execution, transaction, or value-kind failure; derives from `Error` |
| `ArchiveError` | unsafe, malformed, unsupported, or over-limit archive, or an archive creation/extraction failure; derives from `Error` |
| `ProcessError` | `Process.run` could not start the program, timed out, or exceeded its output budget (a non-zero exit is a result, not an error); derives from `Error` |
| `DNSError` | invalid host, host not found, resolver failure, or timeout in `DNS.lookup`; derives from `Error` |
| `TLSError` | `TLS.inspect` could not obtain a certificate (invalid input, connection refused, timeout, not TLS); an invalid certificate is a result, not an error; derives from `Error` |
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
