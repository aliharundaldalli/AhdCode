# Modules

[Back to README](../README.md) · [Math](MATH.md)

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

`Math` is compiler-registered and uses these same import forms. A local
`Math.ahd` cannot shadow it.
