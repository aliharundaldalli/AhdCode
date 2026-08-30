# Getting started

[English] · [Türkçe](GETTING_STARTED_TR.md)

[Back to README](../README.md) · [Language tour](LANGUAGE_TOUR.md) · [CLI](CLI.md)

## Install the compiler

AhdCode currently builds with Go 1.25 or newer.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode
export PATH="$(go env GOPATH)/bin:$PATH"
```

Confirm the installation:

```bash
ahdcode --version
```

## Your first program

Create `hello.ahd`:

```ahd
name := "AhdCode"
write("Hello {name}")
```

The compiler infers `String`; the binding is still statically typed. Write an
explicit annotation (`name: String := ...`) whenever it communicates intent or
inference is insufficient.

For a short reusable operation, an expression-only lambda creates a value of
the existing `Function` type:

```ahd
square := lambda (value: Int) -> value^2
write(square(5))
```

Lambda parameters require explicit types; the return type is inferred from
the single expression. Use a normal Function for a block or multiple steps.

Run it:

```bash
ahdcode run hello.ahd
```

Build a native executable:

```bash
ahdcode build hello.ahd -o hello
./hello
```

## Input

`take` reads one line. It returns text, so numeric input uses an explicit
conversion:

```ahd
name := take("Name: ")
age := int(take("Age: "))

write("{name} is {age}")
```

## Format source

```bash
ahdcode format hello.ahd
ahdcode format --check hello.ahd
```

The first command updates the file atomically. The second only checks whether
the file is already canonical.

Next: read the [language tour](LANGUAGE_TOUR.md), learn how to act on
[diagnostics](DIAGNOSTICS.md), or run the
[curated examples](../examples/v0.1/README.md), including UTC Time, CSV, and
[Data tables](DATA.md).
