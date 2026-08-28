# Getting started

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
name: String := "AhdCode"
write("Hello {name}")
```

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
name: String := take("Name: ")
age: Int := int(take("Age: "))

write("{name} is {age}")
```

## Format source

```bash
ahdcode format hello.ahd
ahdcode format --check hello.ahd
```

The first command updates the file atomically. The second only checks whether
the file is already canonical.

Next: read the [language tour](LANGUAGE_TOUR.md) or run the
[curated examples](../examples/v0.1/README.md).
