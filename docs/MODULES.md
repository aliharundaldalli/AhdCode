# Modules

[English] · [Türkçe](MODULES_TR.md)

[Back to README](../README.md) · [Math](MATH.md) · [File and Path](FILESYSTEM.md)

A local module is a sibling `.ahd` file. The reference is one case-sensitive
identifier: `Utilities` resolves to `Utilities.ahd` beside the importing file.
v0.1 has no dotted paths, package-root search, or configurable module path.

Namespace import:

```ahd
bring Utilities
write(Utilities.greet("Ali"))
```

Direct import:

```ahd
from Utilities bring greet
write(greet("Ali"))
```

Selective multiline import:

```ahd
from Utilities bring (
    greet
    farewell
)
```

Public-all import:

```ahd
from Utilities bring all
```

`all` brings only public, non-`Confidential` symbols. Import collisions and
circular dependencies are compile-time errors.

`Math`, `Time`, `Latex`, `Path`, `File`, `Regex`, and `CSV` are compiler-registered and use
these same import forms. A local file cannot shadow a standard module of the
same name. They can also use the ordinary namespace alias form:

```ahd
bring File as F
F.writeText("note.txt", "hello")
```

See [Time](TIME.md), [CSV](CSV.md), and the other module-specific references
for their typed surfaces and catchable domain errors.
